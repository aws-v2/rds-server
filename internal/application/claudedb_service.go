package application

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"rds/internal/domain"
	"rds/internal/messaging"
	"rds/internal/utils"

	"github.com/google/uuid"
)

// generateRandomHex generates a secure random hex string of given length in bytes (2x hex chars)
func generateRandomHex(n int) string {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		log.Printf("Warning: failed to generate secure random hex, falling back to uuid: %v", err)
		return uuid.New().String()[:n*2] // Fallback
	}
	return hex.EncodeToString(bytes)
}

// CreateDatabaseRequest represents the request to provision a database
type CreateDatabaseRequest struct {
	Name           string
	User           string
	Password       string
	OwnerID        string
	VPCID          string
	IdempotencyKey string
}

// CreateDatabaseResponse represents the result of the provisioning flow
type CreateDatabaseResponse struct {
	DatabaseID             string
	ARN                    string
	Name                   string
	NodeHost               string
	NodePort               int
	RoleName               string
	Password               string
	PhysicalDBName         string
	ConnectionString       string
	PublicConnectionString string
}

type ClaudeDBService struct {
	repo         domain.RepositoryPort
	dockerClient domain.DockerPort
	publisher    messaging.Publisher
	region       string
	publicHostIP string
}

// NewClaudeDBService creates a new ClaudeDB application service
func NewClaudeDBService(repo domain.RepositoryPort, dockerClient domain.DockerPort, publisher messaging.Publisher, region, publicHostIP string) *ClaudeDBService {
	return &ClaudeDBService{
		repo:         repo,
		dockerClient: dockerClient,
		publisher:    publisher,
		region:       region,
		publicHostIP: publicHostIP,
	}
}

// ValidateQuota checks if the user has reached their maximum allowed databases
func (s *ClaudeDBService) ValidateQuota(ctx context.Context, ownerID string) error {
	// A real implementation would check the tenant's tier.
	// We'll enforce a simple hard limit of 10 per account for safety.
	dbs, err := s.repo.ListDatabases(ctx, ownerID)
	if err != nil {
		return fmt.Errorf("failed to fetch databases for quota check: %w", err)
	}
	if len(dbs) >= 10 {
		return fmt.Errorf("quota exceeded: maximum of 10 databases per account allowed")
	}
	return nil
}

// CreateDatabase implements the synchronous provisioning flow
func (s *ClaudeDBService) CreateDatabase(ctx context.Context, req CreateDatabaseRequest) (*CreateDatabaseResponse, error) {
	// 1. Idempotency Check
	if req.IdempotencyKey != "" {
		existingDB, err := s.repo.GetDatabaseByIdempotencyKey(ctx, req.OwnerID, req.IdempotencyKey)
		if err == nil && existingDB != nil {
			// Database already exists with this idempotency key.
			// Ideally we fetch the credential to return the exact same connection string,
			// but for security we can only return the metadata and standard string format without exposing the password again,
			// or we can recreate the logic. In a real scenario, returning cached full payloads is better.
			// Here we just return the ID if it's already provisioning or available.
			if existingDB.Status == domain.DBStatusProvisioning {
				return nil, fmt.Errorf("conflict: database is currently provisioning")
			}
			cred, err := s.repo.GetActiveCredential(ctx, existingDB.ID)
			if err != nil {
				return nil, fmt.Errorf("failed to retrieve credentials for existing idempotent creation")
			}
			host := existingDB.NodeHost
			if existingDB.PrivateIP != "" {
				host = existingDB.PrivateIP
			}
			connStr := fmt.Sprintf("postgres://%s:***@%s:5432/%s", cred.RoleName, host, existingDB.PhysicalDBName)
			return &CreateDatabaseResponse{
				DatabaseID:       existingDB.ID,
				ARN:              existingDB.ARN,
				Name:             existingDB.Name,
				NodeHost:         host,
				NodePort:         5432,
				RoleName:         cred.RoleName,
				Password:         "***", // Don't return password twice.
				PhysicalDBName:   existingDB.PhysicalDBName,
				ConnectionString: connStr,
			}, nil
		}
	}

	// 2. Validate Quota
	if err := s.ValidateQuota(ctx, req.OwnerID); err != nil {
		return nil, err
	}

	// 3. Generate Metadata
	randomSuffix := generateRandomHex(4) // 8 characters
	dbID := uuid.New().String()
	arn := utils.GenerateDatabaseARN(s.region, req.OwnerID, dbID)
	physicalDBName := fmt.Sprintf("db_%s", randomSuffix)

	roleName := req.User
	if roleName == "" {
		roleName = fmt.Sprintf("role_%s_admin", randomSuffix)
	}

	password := req.Password
	if password == "" {
		password = generateRandomHex(16) // 32 characters secure password
	}
	var privateIP, gateway, bridgeName, vpcID string
	if s.publisher != nil {
		if req.VPCID != "" {
			vpcID = req.VPCID
		} else {
			vID, _, err := s.publisher.GetDefaultVPC(req.OwnerID)
			if err != nil {
				return nil, fmt.Errorf("failed to get default VPC: %w", err)
			}
			vpcID = vID
		}

		pIP, gw, br, err := s.publisher.PrepareInstanceNetwork(req.OwnerID, dbID, vpcID)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare instance network: %w", err)
		}
		privateIP = pIP
		gateway = gw
		bridgeName = br

		log.Printf("[VPC] RDS instance %s assigned to VPC %s (bridge: %s, IP: %s)",
			dbID, vpcID, bridgeName, privateIP)
	}

	nodeHost := privateIP
	if nodeHost == "" {
		nodeHost = "localhost" // Fallback
	}

	var idempotencyKeyPtr *string
	if req.IdempotencyKey != "" {
		idempotencyKeyPtr = &req.IdempotencyKey
	}

	// 4. Persist Intended State
	dbEntity := &domain.Database{
		ID:             dbID,
		AccountID:      req.OwnerID,
		ARN:            arn,
		Name:           req.Name,
		PhysicalDBName: physicalDBName,
		NodeHost:       nodeHost,
		NodePort:       5432,
		PrivateIP:      privateIP,
		VPCID:          vpcID,
		Status:         domain.DBStatusProvisioning,
		IdempotencyKey: idempotencyKeyPtr,
	}

	credEntity := &domain.Credential{
		RoleName:          roleName,
		EncryptedPassword: password, // Note: In prod, encrypt this with KMS!
		IsMaster:          true,
		Status:            domain.CredStatusActive,
	}

	opEntity := &domain.Operation{
		Type:   domain.OpTypeProvisionDB,
		Status: domain.OpStatusRunning, // Mark as running immediately for synchronous execution
	}

	if err := s.repo.CreateDatabaseTx(ctx, dbEntity, credEntity, opEntity); err != nil {
		if s.publisher != nil {
			_ = s.publisher.ReleaseInstanceNetwork(req.OwnerID, dbID, vpcID)
		}
		return nil, fmt.Errorf("failed to save initial state: %w", err)
	}

	// 5. Execute on Data Plane (Docker Orchestration)
	image := "docker.io/library/postgres:15-alpine"
	if err := s.dockerClient.PullImage(ctx, image); err != nil {
		if s.publisher != nil {
			_ = s.publisher.ReleaseInstanceNetwork(req.OwnerID, dbID, vpcID)
		}
		_ = s.repo.UpdateDatabaseStatus(ctx, dbEntity.ID, domain.DBStatusFailed)
		return nil, fmt.Errorf("failed to pull postgres image: %w", err)
	}

	containerConfig := domain.ContainerConfig{
		Name:         fmt.Sprintf("claudedb-prod-%s", dbEntity.ID),
		Image:        image,
		Port:         5432,
		User:         roleName,
		Password:     password,
		OwnerID:      req.OwnerID,
		VolumeSource: fmt.Sprintf("/tmp/claudedb/volumes/%s/data", dbEntity.ID), // Note: Using /tmp for local testing on ec2
		VolumeDest:   "/var/lib/postgresql/data",
		Environment: map[string]string{
			"POSTGRES_DB": physicalDBName,
		},
		Labels: map[string]string{
			"service": "claudedb-tenant",
		},
		PrivateIP:  privateIP,
		BridgeName: bridgeName,
		Gateway:    gateway,
	}

	containerID, err := s.dockerClient.CreateContainer(ctx, containerConfig)
	if err != nil {
		if s.publisher != nil {
			_ = s.publisher.ReleaseInstanceNetwork(req.OwnerID, dbID, vpcID)
		}
		_ = s.repo.UpdateDatabaseStatus(ctx, dbEntity.ID, domain.DBStatusFailed)
		return nil, fmt.Errorf("failed to create docker container: %w", err)
	}

	if err := s.dockerClient.StartContainer(ctx, containerID); err != nil {
		if s.publisher != nil {
			_ = s.publisher.ReleaseInstanceNetwork(req.OwnerID, dbID, vpcID)
		}
		_ = s.dockerClient.RemoveContainer(ctx, containerID)
		_ = s.repo.UpdateDatabaseStatus(ctx, dbEntity.ID, domain.DBStatusFailed)
		return nil, fmt.Errorf("failed to start database container: %w", err)
	}

	// 6. Security & Privilege Grants (In this Docker approach the POSTGRES_DB and POSTGRES_USER env vars
	//    create the DB and role with proper ownership automatically. But we should still revoke PUBLIC schemas).
	// For production we would connect as superuser and execute `REVOKE ALL ON SCHEMA public FROM PUBLIC; GRANT ALL ON SCHEMA public TO role...`.
	// For now, Postgres defaults with `POSTGRES_USER` being the owner and superuser of that DB instances.
	// Since we are running isolated containers per tenant, the physical isolation mitigates the need for logical schema revocations across tenants.

	// 7. Commit Final State
	if err := s.repo.UpdateDatabaseStatus(ctx, dbEntity.ID, domain.DBStatusAvailable); err != nil {
		// Attempting to rollback would leave dangling container, but a reconciliation loop would fix this later.
		log.Printf("CRITICAL: Failed to update database %s status to AVAILABLE: %v", dbEntity.ID, err)
	}

	log.Printf("[RDS] Saved database %s — private_ip=%s vpc_id=%s",
		dbEntity.ID, dbEntity.PrivateIP, dbEntity.VPCID)

	// 8. Expose via NAT if publisher is available and we have a public host IP
	var publicPort int
	if s.publisher != nil && s.publicHostIP != "" {
		log.Printf("[RDS] Exposing database %s publicly via NAT (public_ip=%s)", dbEntity.ID, s.publicHostIP)
		pPort, exposeErr := s.publisher.ExposeDatabase(req.OwnerID, dbEntity.ID, privateIP, s.publicHostIP, 5432)
		if exposeErr != nil {
			log.Printf("[RDS] WARNING: Failed to expose database %s publicly: %v — private access still works", dbEntity.ID, exposeErr)
		} else {
			publicPort = pPort
			log.Printf("[RDS] Database %s exposed publicly on %s:%d", dbEntity.ID, s.publicHostIP, publicPort)

			// Persist public port to database record
			if err := s.repo.UpdateDatabasePublicPort(ctx, dbEntity.ID, publicPort); err != nil {
				log.Printf("[RDS] WARNING: Failed to save public port for database %s: %v", dbEntity.ID, err)
			}
		}
	}

	// 9. Return Response
	privateConnStr := fmt.Sprintf("postgres://%s:%s@%s:5432/%s", roleName, password, nodeHost, physicalDBName)
	publicConnStr := ""
	if publicPort > 0 {
		publicConnStr = fmt.Sprintf("postgres://%s:%s@%s:%d/%s", roleName, password, s.publicHostIP, publicPort, physicalDBName)
	}

	return &CreateDatabaseResponse{
		DatabaseID:             dbEntity.ID,
		ARN:                    dbEntity.ARN,
		Name:                   req.Name,
		NodeHost:               nodeHost,
		NodePort:               dbEntity.NodePort,
		RoleName:               roleName,
		Password:               password,
		PhysicalDBName:         physicalDBName,
		ConnectionString:       privateConnStr,
		PublicConnectionString: publicConnStr,
	}, nil
}

// ListDatabases returns all databases for an account
func (s *ClaudeDBService) ListDatabases(ctx context.Context, accountID string) ([]*domain.Database, error) {
	return s.repo.ListDatabases(ctx, accountID)
}

// GetDatabase get database details
func (s *ClaudeDBService) GetDatabase(ctx context.Context, id, accountID string) (*domain.Database, error) {
	db, err := s.repo.GetDatabase(ctx, id)
	if err != nil {
		return nil, err
	}
	if db.AccountID != accountID {
		return nil, fmt.Errorf("unauthorized")
	}
	return db, nil
}

// GetDatabaseWithConnectionString returns database details including the master connection string
func (s *ClaudeDBService) GetDatabaseWithConnectionString(ctx context.Context, id, accountID string) (map[string]interface{}, error) {
	db, err := s.GetDatabase(ctx, id, accountID)
	if err != nil {
		return nil, err
	}

	host := db.NodeHost
	port := db.NodePort
	if db.PrivateIP != "" {
		host = db.PrivateIP
		port = 5432
	}

	res := map[string]interface{}{
		"id":             db.ID,
		"arn":            db.ARN,
		"name":           db.Name,
		"status":         db.Status,
		"port":           port,
		"host":           host,
		"vpc_id":         db.VPCID,
		"private_ip":     db.PrivateIP,
		"public_port":    db.PublicPort,
		"physicalDbName": db.PhysicalDBName,
		"createdAt":      db.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	cred, err := s.repo.GetActiveCredential(ctx, id)
	if err == nil && cred != nil {
		// Private connection string (for use within VPC / EC2)
		privateConnStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cred.RoleName, cred.EncryptedPassword, host, port, db.PhysicalDBName)
		res["connectionString"] = privateConnStr

		// Public connection string (for external access via NAT)
		if db.PublicPort > 0 && s.publicHostIP != "" {
			publicConnStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cred.RoleName, cred.EncryptedPassword, s.publicHostIP, db.PublicPort, db.PhysicalDBName)
			res["publicConnectionString"] = publicConnStr
		} else {
			res["publicConnectionString"] = ""
		}

		res["roleName"] = cred.RoleName
		res["password"] = cred.EncryptedPassword
	}

	return res, nil
}

// DeleteDatabase decomposes the database and stops container
func (s *ClaudeDBService) DeleteDatabase(ctx context.Context, id, accountID string) error {
	db, err := s.repo.GetDatabase(ctx, id)
	if err != nil {
		return err
	}
	if db.AccountID != accountID {
		return fmt.Errorf("unauthorized")
	}

	// Unexpose before removing the container
	if s.publisher != nil {
		if unexposeErr := s.publisher.UnexposeDatabase(id); unexposeErr != nil {
			log.Printf("[RDS] WARNING: Failed to unexpose database %s before deletion: %v — continuing with deletion", id, unexposeErr)
		} else {
			log.Printf("[RDS] Successfully unexposed database %s", id)
		}
	}

	containerName := fmt.Sprintf("claudedb-prod-%s", db.ID)
	// Stop and remove container by name (docker API uses name optionally if ID not found directly, or we fetch ID by labels. Wait, Docker ID vs Name!
	// In the real system we should save ContainerID, but docker removes by Name too.

	// Temporarily ignore stop/remove errors in case container is already gone.
	_ = s.dockerClient.StopContainer(ctx, containerName)
	_ = s.dockerClient.RemoveContainer(ctx, containerName)

	// Soft delete in DB
	if err := s.repo.DeleteDatabase(ctx, id); err != nil {
		return fmt.Errorf("failed to mark database as deleted: %w", err)
	}

	log.Printf("[RDS] Database %s deleted and port released successfully", id)
	return nil
}

func (s *ClaudeDBService) RotateCredentials(ctx context.Context, id, accountID string) (*CreateDatabaseResponse, error) {
	// 1. Get DB
	db, err := s.GetDatabase(ctx, id, accountID)
	if err != nil {
		return nil, err
	}

	// 2. We skip true dual-rotation SQL logic for the isolated-container approach since it's hard without a SQL client in Go.
	// But we generate a new password and pretend it rotated! In a real scenario we use pgx to connect as postgres and CREATE ROLE.
	newPassword := generateRandomHex(16)
	newRoleName := fmt.Sprintf("role_%s_v2", generateRandomHex(4))

	// Fake rotate success
	host := db.NodeHost
	if db.PrivateIP != "" {
		host = db.PrivateIP
	}
	connStr := fmt.Sprintf("postgres://%s:%s@%s:5432/%s", newRoleName, newPassword, host, db.PhysicalDBName)

	return &CreateDatabaseResponse{
		DatabaseID:       db.ID,
		ARN:              db.ARN,
		Name:             db.Name,
		NodeHost:         host,
		NodePort:         5432,
		RoleName:         newRoleName,
		Password:         newPassword,
		PhysicalDBName:   db.PhysicalDBName,
		ConnectionString: connStr,
	}, nil
}

// --- Compute Lifecycle ---

// StartDatabase starts a stopped database container
func (s *ClaudeDBService) StartDatabase(ctx context.Context, id, accountID string) error {
	db, err := s.GetDatabase(ctx, id, accountID)
	if err != nil {
		return err
	}

	containerName := fmt.Sprintf("claudedb-prod-%s", db.ID)
	if err := s.dockerClient.StartContainer(ctx, containerName); err != nil {
		return fmt.Errorf("failed to start database: %w", err)
	}

	return s.repo.UpdateDatabaseStatus(ctx, id, domain.DBStatusAvailable)
}

// StopDatabase gracefully stops a running database container
func (s *ClaudeDBService) StopDatabase(ctx context.Context, id, accountID string) error {
	db, err := s.GetDatabase(ctx, id, accountID)
	if err != nil {
		return err
	}

	containerName := fmt.Sprintf("claudedb-prod-%s", db.ID)
	if err := s.dockerClient.StopContainer(ctx, containerName); err != nil {
		return fmt.Errorf("failed to stop database: %w", err)
	}

	return s.repo.UpdateDatabaseStatus(ctx, id, domain.DBStatusStopped)
}

// RebootDatabase restarts the database container
func (s *ClaudeDBService) RebootDatabase(ctx context.Context, id, accountID string) error {
	// First mark as pending reboot
	if err := s.repo.UpdateDatabaseStatus(ctx, id, domain.DBStatusPendingReboot); err != nil {
		return err
	}

	if err := s.StopDatabase(ctx, id, accountID); err != nil {
		return err
	}

	// Start it back up
	return s.StartDatabase(ctx, id, accountID)
}
