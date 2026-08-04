package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"rds/internal/domain"
	"rds/internal/interfaces"
	"rds/internal/messaging"
	"strings"
	"time"

	"sync"

	"github.com/google/uuid"
)

// ClaudeDBService is the RDS control plane. It no longer manages container runtimes directly.
// Instead, it publishes provisioning events to the EC2 service via NATS, which provisions
// a dedicated VM (HostType: "rds") with PostgreSQL injected via cloud-init.
type ClaudeDBService struct {
	repo         interfaces.RepositoryPort
	publisher    messaging.Publisher
	region       string
	publicHostIP string
	mu           sync.Mutex
}

const pendingNodeHostPrefix = "pending-rds://"

// generateRandomHex generates a secure random hex string of given length in bytes (2x hex chars)
func generateRandomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		log.Printf("Warning: failed to generate secure random hex, falling back to uuid: %v", err)
		return uuid.New().String()[:n*2] // Fallback
	}
	return hex.EncodeToString(bytes)
}

// generateRandomVPCSubnet creates a random 10.x.y.0/24 subnet to minimize collision chances
func generateRandomVPCSubnet() (subnet, gateway string) {
	b := make([]byte, 2)
	_, _ = rand.Read(b)
	o2 := (int(b[0]) % 240) + 10 // 10..249
	o3 := (int(b[1]) % 250) + 1  // 1..250
	return fmt.Sprintf("10.%d.%d.0/24", o2, o3), fmt.Sprintf("10.%d.%d.1", o2, o3)
}

func databaseEndpoint(db *domain.Database) (string, int) {

	return "NodeHost", 10
}

// NewClaudeDBService creates a new ClaudeDB control plane service
func NewClaudeDBService(repo interfaces.RepositoryPort, publisher messaging.Publisher, region, publicHostIP, natsPrefix string) *ClaudeDBService {
	return &ClaudeDBService{
		repo:         repo,
		publisher:    publisher,
		region:       region,
		publicHostIP: publicHostIP,
	}
}

// ValidateQuota checks if the user has reached their maximum allowed databases
func (s *ClaudeDBService) ValidateQuota(ctx context.Context, ownerID string) error {
	dbs, err := s.repo.ListDatabases(ctx, ownerID)
	if err != nil {
		return fmt.Errorf("failed to fetch databases for quota check: %w", err)
	}
	if len(dbs) >= 20 {
		return fmt.Errorf("quota exceeded: maximum of 10 databases per account allowed")
	}
	return nil
}

// CreateDatabase implements the RDS control plane provisioning flow.
//
// The flow is:
//  1. Idempotency check
//  2. Quota validation
//  3. Generate metadata (ID, ARN, credentials)
//  4. Persist the database record with status PROVISIONING
//  5. Publish a ProvisionInstanceEvent to the EC2 service (dev.v1.ec2.task.provision)
//     with profile="rds" so a dedicated Postgres VM is booted via cloud-init.
//  6. Return immediately — the DB stays in PROVISIONING until the EC2 service
//     reports back (via a future callback event) with the VM's IP and port.

func (s *ClaudeDBService) CreateDatabase(ctx context.Context, req domain.CreateDatabaseRequest) (*domain.CreateDatabaseResponse, error) {
	// 1 TODO: step 1 here was an idemptotency check
	// putit back ,for now ithas to go
	// 2. Validate Quota
	if err := s.ValidateQuota(ctx, req.OwnerID); err != nil {
		return nil, err
	}
	fmt.Printf("\n\n quota passed\n\n")

	// 3. Generate Metadata
	randomSuffix := generateRandomHex(4) // 8 characters
	dbID := uuid.New().String()
	physicalDBName := fmt.Sprintf("db_%s", randomSuffix)

	roleName := req.User
	if roleName == "" {
		roleName = fmt.Sprintf("role_%s_admin", randomSuffix)
	}

	password := req.Password
	if password == "" {
		password = generateRandomHex(16) // 32 characters secure password
	}

	// 4. Persist Intended State
	dbEntity := &domain.Database{
		ID:          dbID,
		UserID:      req.OwnerID,
		DBName:      req.InstanceName,
		VMIP:        "2",
		GatewayIP:   "192.168.1.1",
		GatewayPort: 2323,
		Status:      "PROVISIONING",
		VMDBPort:    5432,
	}

	credEntity := &domain.Credential{
		RoleName:          roleName,
		EncryptedPassword: password,
		IsMaster:          true,
		Status:            domain.CredStatusActive,
	}

	opEntity := &domain.Operation{
		Type:   domain.OpTypeProvisionDB,
		Status: domain.OpStatusRunning,
	}

	if err := s.repo.CreateDatabaseTx(ctx, dbEntity, credEntity, opEntity); err != nil {
		return nil, fmt.Errorf("failed to save initial provisioning state: %w", err)
	}
	fmt.Printf("\n\n metadata passed\n\n")

	log.Printf("[RDS] Database record %s persisted (status=PROVISIONING). Dispatching to EC2 service.", dbID)

	// 5. Dispatch to EC2 Control Plane
	// The EC2 service will select an "rds" host, boot a VM, inject credentials
	// via cloud-init, and (eventually) publish a ready event back to us.

	provisionEvent := domain.ProvisionInstanceEvent{
		UserID:     req.OwnerID,
		Profile:    "rds",
		SessionID:  req.SessionID,
		Name:       req.InstanceName,
		ResourceID: dbEntity.ID,
		Specs: domain.VMSpecs{
			CPU:     2,
			RAM:     1024,
			Storage: 10,
		},
	}
	var eC2Response domain.EC2Response

	if s.publisher == nil {
		return nil, fmt.Errorf("The publisher instance is null, cancellingreques")

	}

	eC2Response, err := s.publisher.ProvisionRDSInstance(provisionEvent)
	if err != nil {
		log.Printf("[RDS] WARNING: Failed to publish ProvisionInstanceEvent for database %s: %v", dbID, err)
	}

	fmt.Print(eC2Response.GatewayIP)

	// 6. Return immediately with PROVISIONING status.
	// The connection string will be available once the EC2 VM is up.
	return &domain.CreateDatabaseResponse{
		DatabaseID:             dbEntity.ID,
		ARN:                    "dbEntity.ARN",
		Name:                   req.InstanceName,
		NodeHost:               eC2Response.GatewayIP, // populated after EC2 callback
		NodePort:               eC2Response.GatewayPort,
		RoleName:               roleName,
		Password:               password,
		PhysicalDBName:         physicalDBName,
		ConnectionString:       "", // populated after EC2 callback
		PublicConnectionString: "",
	}, nil
}

// ListDatabases returns all databases for an account
func (s *ClaudeDBService) ListDatabases(ctx context.Context, accountID string) ([]*domain.Database, error) {
	return s.repo.ListDatabases(ctx, accountID)
}

// ListVPCs returns all VPCs for an account
func (s *ClaudeDBService) ListVPCs(ctx context.Context, accountID string) ([]*domain.VPC, error) {
	return s.repo.ListVPCs(ctx, accountID)
}

// CreateVPC creates a VPC metadata record (no Docker network)
func (s *ClaudeDBService) CreateVPC(ctx context.Context, accountID, vpcName string) (string, error) {
	subnet, gw := generateRandomVPCSubnet()
	vpcID := uuid.New().String()
	vpcRec := &domain.VPC{
		ID:         vpcID,
		Name:       vpcName,
		CIDRBlock:  subnet,
		BridgeName: "br-vpc-" + generateRandomHex(4),
		Subnet:     subnet,
		Gateway:    gw,
		TenantID:   accountID,
		Status:     "available",
		IsDefault:  false,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := s.repo.CreateVPC(ctx, vpcRec); err != nil {
		return "nil", fmt.Errorf("failed to create VPC: %w", err)
	}
	return vpcID, nil
}

// AssignVPC changes the VPC assignment of an existing database (metadata update only)
func (s *ClaudeDBService) AssignVPC(ctx context.Context, accountID, databaseID, newVPCID string) error {

	return nil
}

// GetDatabase get database details
func (s *ClaudeDBService) GetDatabase(ctx context.Context, id, accountID string) (*domain.Database, error) {
	db, err := s.repo.GetDatabase(ctx, id)
	if err != nil {
		return nil, err
	}
	if db.UserID != accountID {
		return nil, fmt.Errorf("unauthorized")
	}
	return db, nil
}

type CurrentDB struct {
	ID               string `json:"id"`
	DBUser           string `json:"db_user"`
	DBPassword       string `json:"db_password"`
	PublicConnectionString string `json:"public_connection_string"`
	VPCConnectionString string `json:"vpc_connection_string"`
	DBPort           int    `json:"db_port"`
	PublicPort       int    `json:"public_port"`
	Engine           string `json:"engine"`
	CreatedAt        string `json:"created_at"`
	VPCID            string `json:"vpc_id"` // the port in the vm where the db runs, default is 5432
	Status           string `json:"status"`
	VpcIP           string `json:"vpc_ip"`

	DBName string `json:"name"` // name of the database
	Region string `json:"region"`
}

// GetDatabaseWithConnectionString returns database details including the master connection string
func (s *ClaudeDBService) GetDatabaseWithConnectionString(ctx context.Context, id, accountID string) (CurrentDB, error) {
	db, err := s.GetDatabase(ctx, id, accountID)
	if err != nil {
		return CurrentDB{}, err
	}

	// host, port := databaseEndpoint(db)

	// reds := map[string]interface{}{
	// 	"id":             db.ID,
	// 	"name":           db.DBName,
	// 	"port":           port,
	// 	"host":           host,
	// }

	res := CurrentDB{
		ID:               db.ID,
		PublicConnectionString: fmt.Sprintf("postgres://%s:%s@%s:%d/%s", "username","password",db.GatewayIP,db.GatewayPort,db.DBName),
		VPCConnectionString: "connectionstring",
		DBPort:           db.VMDBPort, // this is the 5432
		PublicPort:       db.GatewayPort,
		DBUser:           "root-user",
		DBPassword:       "password",
		Engine:           "postgres-1",
		DBName:           db.DBName, // thisis the physicaldb name inthe fronted
		Region:           "region-1",
		CreatedAt:        db.CreatedAt,
		Status:           db.Status,

	}

	return res, nil
}

// DeleteDatabase soft-deletes the database record.
// The EC2 service is responsible for terminating the associated VM via its own lifecycle.
func (s *ClaudeDBService) DeleteDatabase(ctx context.Context, id, accountID string) error {
	db, err := s.repo.GetDatabase(ctx, id)
	if err != nil {
		return err
	}
	if db.UserID != accountID {
		return fmt.Errorf("unauthorized")
	}

	if err := s.repo.DeleteDatabase(ctx, id); err != nil {
		return fmt.Errorf("failed to mark database as deleted: %w", err)
	}

	log.Printf("[RDS] Database %s marked as deleted", id)
	return nil
}

func (s *ClaudeDBService) RotateCredentials(ctx context.Context, id, accountID string) (*domain.CreateDatabaseResponse, error) {
	db, err := s.GetDatabase(ctx, id, accountID)
	if err != nil {
		return nil, err
	}

	newPassword := generateRandomHex(16)
	newRoleName := fmt.Sprintf("role_%s_v2", generateRandomHex(4))

	host, port := databaseEndpoint(db)
	connStr := ""
	if host != "" {
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%d/%s", newRoleName, newPassword, host, port, "postgres")
	}

	return &domain.CreateDatabaseResponse{
		DatabaseID:       db.ID,
		ARN:              "db.ARN",
		Name:             db.DBName,
		NodeHost:         host,
		NodePort:         port,
		RoleName:         newRoleName,
		Password:         newPassword,
		PhysicalDBName:   "db.PhysicalDBName",
		ConnectionString: connStr,
	}, nil
}

// --- Compute Lifecycle ---
// These are metadata-only operations now. Full VM lifecycle orchestration
// (start/stop/reboot) will be dispatched to the EC2 service in a future pass.

// StartDatabase marks the database as available (VM lifecycle managed by EC2)
func (s *ClaudeDBService) StartDatabase(ctx context.Context, id, accountID string) error {
	if _, err := s.GetDatabase(ctx, id, accountID); err != nil {
		return err
	}
	return s.repo.UpdateDatabaseStatus(ctx, id, domain.DBStatusAvailable)
}

// StopDatabase marks the database as stopped (VM lifecycle managed by EC2)
func (s *ClaudeDBService) StopDatabase(ctx context.Context, id, accountID string) error {
	if _, err := s.GetDatabase(ctx, id, accountID); err != nil {
		return err
	}
	return s.repo.UpdateDatabaseStatus(ctx, id, domain.DBStatusStopped)
}

// ModifyDatabase handles changes to database configuration
func (s *ClaudeDBService) ModifyDatabase(ctx context.Context, id, accountID, newVpcID string) error {
	if newVpcID != "" {
		return s.AssignVPC(ctx, accountID, id, newVpcID)
	}
	return s.RebootDatabase(ctx, id, accountID)
}

// RebootDatabase marks the database as rebooting and then available
func (s *ClaudeDBService) RebootDatabase(ctx context.Context, id, accountID string) error {
	if err := s.repo.UpdateDatabaseStatus(ctx, id, domain.DBStatusPendingReboot); err != nil {
		return err
	}
	if err := s.StopDatabase(ctx, id, accountID); err != nil {
		return err
	}
	return s.StartDatabase(ctx, id, accountID)
}

// --- Scaling Policy Management ---

func (s *ClaudeDBService) CreateScalingPolicy(ctx context.Context, tenantID string, policy domain.ScalingPolicyRequest) error {
	return s.repo.CreateScalingPolicy(ctx, tenantID, policy)
}

func (s *ClaudeDBService) GetScalingPolicies(ctx context.Context, tenantID string) ([]domain.ScalingPolicy, error) {
	return s.repo.GetScalingPolicies(ctx, tenantID)
}

func (s *ClaudeDBService) UpdateScalingPolicy(ctx context.Context, tenantID, policyID string, req domain.UpdateScalingPolicyRequest) error {
	return s.repo.UpdateScalingPolicy(ctx, tenantID, policyID, req)
}

func (s *ClaudeDBService) DeleteScalingPolicy(ctx context.Context, tenantID, policyID string) error {
	return s.repo.DeleteScalingPolicy(ctx, tenantID, policyID)
}

func (s *ClaudeDBService) isDuplicateNodeHostError(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return (strings.Contains(errMsg, "unique_node_host_port") ||
		(strings.Contains(errMsg, "duplicate key") && strings.Contains(errMsg, "node_host")))
}

const (
	EventInstanceStarted      = "INSTANCE_STARTED"
	EventInstanceStopped      = "INSTANCE_STOPPED"
	EventHealthUpdate         = "HEALTH_UPDATE"
	EventProvisioningProgress = "PROVISIONING_PROGRESS"
	EventInstanceError        = "INSTANCE_ERROR"
	EventInstanceProvisioned  = "INSTANCE_PROVISIONED"
)
const (
	RepoReConcile     = "RECONCILE_REPO"
	RepoCreateRecord  = "CREATE_REPO_RECORD"
	RepoUpadateRecord = "UPDATE_REPO_RECORD"
	NetworkReconcile  = "NETWORK_RECONCILE"
	CreateOverlay     = "CREATE_OVERLAY"
	BuildCloudInit    = "BUILD_CLOUD_INIT"
	TransferOverlay   = "TRANSFER_OVERLAY"
	InjectOverlay     = "INJECT_ASSETS"
	StartVM           = "START_VM"
	VMProvisioned     = "VM_PROVISIONED"
	VMStarted         = "INSTANCE_STARTED"
	VMStoped          = "INSTANCE_STOPPED"
)

func (s *ClaudeDBService) HandleProvisionLifeCycle(ctx context.Context, event *messaging.InstanceLifecycleEventSub) error {
	if event == nil {
		return fmt.Errorf("nil lifecycle event")
	}

	// prefer an explicit ResourceID on the event, fall back to payload metadata
	resourceID := event.ResourceID
	if resourceID == "" && event.Payload.InstanceStartedMetadata.ResourceID != "" {
		resourceID = event.Payload.InstanceStartedMetadata.ResourceID
	}
	if resourceID == "" {
		return fmt.Errorf("lifecycle event missing resource id")
	}

	// Handle error case from EC2 side
	if event.EventType == EventInstanceError || strings.EqualFold(event.Stage, "FAILED") {
		_ = s.repo.UpdateDatabaseStatus(ctx, resourceID, domain.DBStatusFailed)
		return fmt.Errorf("instance lifecycle reported failure for resource %s", resourceID)
	}

	// Consider the instance ready only when EC2 signals started/completed AND stage indicates started
	isStartedEvent := event.EventType == EventInstanceStarted || strings.EqualFold(event.EventType, "INSTANCE_STARTED")
	isStartedStage := event.Stage == VMStarted || strings.EqualFold(event.Stage, "INSTANCE_STARTED") || strings.EqualFold(event.Stage, "COMPLETED")

	if isStartedEvent && isStartedStage {
		// gather networking details
		privateIP := event.Payload.InstanceStartedMetadata.InstancePrivateIp
		publicIP := event.Payload.InstanceStartedMetadata.InstancePublicIp
		servicePort := event.Payload.ServicePort

		// fetch DB record to validate and obtain VPC if missing
		// db, err := s.repo.GetDatabase(ctx, resourceID)
		// if err != nil {
		// 	return fmt.Errorf("failed to load database %s: %w", resourceID, err)
		// }

		vpcID := "db.VPCID"
		// if vpcID == "" {
		// 	// fall back to payload VPC if present
		// 	if event.Payload.VPCID != "" {
		// 		vpcID = event.Payload.VPCID
		// 	}
		// }

		// nodeHost: prefer public IP (for external access), but DB private IP is stored separately
		nodeHost := publicIP
		if nodeHost == "" {
			nodeHost = privateIP
		}

		if err := s.repo.UpdateDatabaseNetwork(ctx, resourceID, vpcID, privateIP, nodeHost); err != nil {
			if s.isDuplicateNodeHostError(err) {
				log.Printf("[RDS] duplicate node host when updating network for %s: %v", resourceID, err)
				// fall through: still attempt to update status/public port
			} else {
				return fmt.Errorf("failed to update database network for %s: %w", resourceID, err)
			}
		}

		if servicePort > 0 {
			if err := s.repo.UpdateDatabasePublicPort(ctx, resourceID, servicePort); err != nil {
				log.Printf("[RDS] failed to update public port for %s: %v", resourceID, err)
			}
		}

		if err := s.repo.UpdateDatabaseStatus(ctx, resourceID, domain.DBStatusAvailable); err != nil {
			return fmt.Errorf("failed to mark database %s available: %w", resourceID, err)
		}

		log.Printf("[RDS] Database %s marked AVAILABLE (host=%s private_ip=%s public_port=%d)", resourceID, nodeHost, privateIP, servicePort)
	}

	return nil
}
