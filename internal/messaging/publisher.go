package messaging

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"rds/internal/domain"
)

// Publisher defines the interface for instance network operations.
type Publisher interface {
	PrepareInstanceNetwork(tenantID, resourceID, vpcID string) (privateIP, gateway, bridgeName string, err error)
	ReleaseInstanceNetwork(tenantID, resourceID, vpcID string) error
	GetDefaultVPC(tenantID string) (vpcID, bridgeName string, err error)
	ExposeDatabase(tenantID, resourceID, privateIP, publicIP string, privatePort int) (publicPort int, err error)
	UnexposeDatabase(resourceID string) error
	ListVPCs(tenantID string) ([]domain.VPC, error)
	CreateVPC(tenantID, vpcName, requestedBy string) error
	ReconcileVPCs() error
}

// NATSPublisher implements the Publisher interface using NATS Request-Response pattern.
type NATSPublisher struct {
	nc *nats.Conn
}

type prepareRequest struct {
	CorrelationID string `json:"correlation_id"`
	TenantID      string `json:"tenant_id"`
	InstanceID    string `json:"instance_id"`
	VpcID         string `json:"vpc_id"`
}

type prepareResponse struct {
	Success    bool   `json:"success"`
	PrivateIP  string `json:"private_ip"`
	Gateway    string `json:"gateway"`
	BridgeName string `json:"bridge_name"`
}

type releaseRequest struct {
	CorrelationID string `json:"correlation_id"`
	TenantID      string `json:"tenant_id"`
	InstanceID    string `json:"instance_id"`
	VpcID         string `json:"vpc_id"`
}

type releaseResponse struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

type vpcGetDefaultRequest struct {
	CorrelationID string `json:"correlation_id"`
	TenantID      string `json:"tenant_id"`
}

type vpcGetDefaultResponse struct {
	VpcID         string `json:"vpc_id"`
	BridgeName    string `json:"bridge_name"`
	CorrelationID string `json:"correlation_id"`
}

type exposeRDSRequest struct {
	CorrelationID string `json:"correlation_id"`
	TenantID      string `json:"tenant_id"`
	ResourceID    string `json:"resource_id"`
	PrivateIP     string `json:"private_ip"`
	PrivatePort   int    `json:"private_port"`
	PublicIP      string `json:"public_ip"`
}

type exposeRDSResponse struct {
	Success    bool   `json:"success"`
	PublicPort int    `json:"public_port"`
	Error      string `json:"error"`
}

type unexposeRDSRequest struct {
	CorrelationID string `json:"correlation_id"`
	ResourceID    string `json:"resource_id"`
}

type unexposeRDSResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type createVPCEvent struct {
	CorrelationID string `json:"correlation_id"`
	TenantID      string `json:"tenant_id"`
	VPCName       string `json:"vpc_name"`
	RequestedBy   string `json:"requested_by"`
}

type listVPCsRequest struct {
	CorrelationID string `json:"correlation_id"`
	TenantID      string `json:"tenant_id"`
}

type listVPCsResponse struct {
	CorrelationID string       `json:"correlation_id"`
	TenantID      string       `json:"tenant_id"`
	VPCs          []domain.VPC `json:"vpcs"`
	Error         string       `json:"error,omitempty"`
}

type reconcileVPCsRequest struct {
	CorrelationID string `json:"correlation_id"`
}

type reconcileVPCsResponse struct {
	CorrelationID string `json:"correlation_id"`
	Success       bool   `json:"success"`
	Message       string `json:"message,omitempty"`
	Error         string `json:"error,omitempty"`
}

// NewNATSPublisher connects to NATS and returns a NATSPublisher instance.
func NewNATSPublisher(url, user, password string) (*NATSPublisher, error) {
	opts := []nats.Option{
		nats.UserInfo(user, password),
	}

	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &NATSPublisher{nc: nc}, nil
}

// PrepareInstanceNetwork sends a network preparation request to the Network Service via NATS.
func (p *NATSPublisher) PrepareInstanceNetwork(tenantID, resourceID, vpcID string) (string, string, string, error) {
	correlationID := uuid.New().String()
	req := prepareRequest{
		CorrelationID: correlationID,
		TenantID:      tenantID,
		InstanceID:    resourceID,
		VpcID:         vpcID,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to marshal request: %w", err)
	}
	log.Printf("[RDS-NATS] Sending PrepareInstanceNetwork request: %s", string(reqData))

	msg, err := p.nc.Request("dev.network.v1.instance.prepare", reqData, 5*time.Second)
	if err != nil {
		return "", "", "", fmt.Errorf("NATS request failed: %w", err)
	}

	log.Printf("[RDS-NATS] Received PrepareInstanceNetwork response: %s", string(msg.Data))

	var resp prepareResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return "", "", "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !resp.Success {
		return "", "", "", fmt.Errorf("network preparation failed")
	}

	return resp.PrivateIP, resp.Gateway, resp.BridgeName, nil
}

// ReleaseInstanceNetwork sends a network release request to the Network Service via NATS.
func (p *NATSPublisher) ReleaseInstanceNetwork(tenantID, resourceID, vpcID string) error {
	correlationID := uuid.New().String()
	req := releaseRequest{
		CorrelationID: correlationID,
		TenantID:      tenantID,
		InstanceID:    resourceID,
		VpcID:         vpcID,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	log.Printf("[RDS-NATS] Sending ReleaseInstanceNetwork request: %s", string(reqData))

	msg, err := p.nc.Request("dev.network.v1.instance.release", reqData, 5*time.Second)
	if err != nil {
		return fmt.Errorf("NATS request failed: %w", err)
	}

	log.Printf("[RDS-NATS] Received ReleaseInstanceNetwork response: %s", string(msg.Data))

	var resp releaseResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.Status != "success" {
		return fmt.Errorf("network release failed: %s", resp.Error)
	}

	return nil
}

// GetDefaultVPC fetches the default VPC for a tenant.
func (p *NATSPublisher) GetDefaultVPC(tenantID string) (string, string, error) {
	correlationID := uuid.New().String()
	req := vpcGetDefaultRequest{
		CorrelationID: correlationID,
		TenantID:      tenantID,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal request: %w", err)
	}
	log.Printf("[RDS-NATS] Sending GetDefaultVPC request: %s", string(reqData))

	msg, err := p.nc.Request("dev.network.v1.vpc.default.get", reqData, 5*time.Second)
	if err != nil {
		return "", "", fmt.Errorf("NATS request failed: %w", err)
	}

	log.Printf("[RDS-NATS] Received GetDefaultVPC response: %s", string(msg.Data))

	var resp vpcGetDefaultResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.VpcID == "" {
		return "", "", fmt.Errorf("no default VPC found for tenant")
	}

	return resp.VpcID, resp.BridgeName, nil
}

// Close closes the NATS connection.
func (p *NATSPublisher) Close() {
	if p.nc != nil {
		p.nc.Close()
	}
}

// ExposeDatabase sends a request to the Network Service to expose an RDS container publicly via DNAT/SNAT.
func (p *NATSPublisher) ExposeDatabase(tenantID, resourceID, privateIP, publicIP string, privatePort int) (int, error) {
	correlationID := uuid.New().String()
	req := exposeRDSRequest{
		CorrelationID: correlationID,
		TenantID:      tenantID,
		ResourceID:    resourceID,
		PrivateIP:     privateIP,
		PrivatePort:   privatePort,
		PublicIP:      publicIP,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}
	log.Printf("[RDS-NATS] Sending ExposeDatabase request: %s", string(reqData))

	msg, err := p.nc.Request("dev.network.v1.rds.expose", reqData, 5*time.Second)
	if err != nil {
		return 0, fmt.Errorf("NATS request failed: %w", err)
	}

	log.Printf("[RDS-NATS] Received ExposeDatabase response: %s", string(msg.Data))

	var resp exposeRDSResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return 0, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !resp.Success {
		return 0, fmt.Errorf("expose database failed: %s", resp.Error)
	}

	return resp.PublicPort, nil
}

// UnexposeDatabase sends a request to the Network Service to remove NAT rules and release the public port.
func (p *NATSPublisher) UnexposeDatabase(resourceID string) error {
	correlationID := uuid.New().String()
	req := unexposeRDSRequest{
		CorrelationID: correlationID,
		ResourceID:    resourceID,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	log.Printf("[RDS-NATS] Sending UnexposeDatabase request: %s", string(reqData))

	msg, err := p.nc.Request("dev.network.v1.rds.unexpose", reqData, 5*time.Second)
	if err != nil {
		return fmt.Errorf("NATS request failed: %w", err)
	}

	log.Printf("[RDS-NATS] Received UnexposeDatabase response: %s", string(msg.Data))

	var resp unexposeRDSResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("unexpose database failed: %s", resp.Error)
	}

	return nil
}

// ListVPCs requests the list of VPCs for a tenant from the Network Service via NATS.
func (p *NATSPublisher) ListVPCs(tenantID string) ([]domain.VPC, error) {
	correlationID := uuid.New().String()
	req := listVPCsRequest{
		CorrelationID: correlationID,
		TenantID:      tenantID,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	log.Printf("[RDS-NATS] Sending ListVPCs request for tenant %s", tenantID)

	msg, err := p.nc.Request("dev.network.v1.vpc.list", reqData, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("NATS request failed: %w", err)
	}

	var resp listVPCsResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("failed to list VPCs: %s", resp.Error)
	}

	return resp.VPCs, nil
}

// CreateVPC dispatches an asynchronous NATS message to create a new VPC.
func (p *NATSPublisher) CreateVPC(tenantID, vpcName, requestedBy string) error {
	correlationID := uuid.New().String()
	event := createVPCEvent{
		CorrelationID: correlationID,
		TenantID:      tenantID,
		VPCName:       vpcName,
		RequestedBy:   requestedBy,
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal create VPC event: %w", err)
	}

	log.Printf("[RDS-NATS] Publishing CreateVPC event for tenant %s, vpc %s", tenantID, vpcName)
	if err := p.nc.Publish("dev.network.v1.vpc.create", eventData); err != nil {
		return fmt.Errorf("failed to publish create VPC event: %w", err)
	}

	return nil
}

// ReconcileVPCs triggers a global VPC reconciliation in the Network Service.
func (p *NATSPublisher) ReconcileVPCs() error {
	correlationID := uuid.New().String()
	req := reconcileVPCsRequest{
		CorrelationID: correlationID,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal reconcile request: %w", err)
	}

	log.Printf("[RDS-NATS] Requesting global VPC reconciliation")
	msg, err := p.nc.Request("dev.network.v1.vpc.reconcile", reqData, 30*time.Second) // Long timeout for reconciliation
	if err != nil {
		return fmt.Errorf("NATS request failed: %w", err)
	}

	var resp reconcileVPCsResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal reconcile response: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("VPC reconciliation failed: %s", resp.Error)
	}

	log.Printf("[RDS-NATS] VPC reconciliation completed: %s", resp.Message)
	return nil
}

