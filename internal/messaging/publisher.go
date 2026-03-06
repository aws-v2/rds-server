package messaging

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// Publisher defines the interface for instance network operations.
type Publisher interface {
	PrepareInstanceNetwork(tenantID, resourceID, vpcID string) (privateIP, gateway, bridgeName string, err error)
	ReleaseInstanceNetwork(tenantID, resourceID, vpcID string) error
	GetDefaultVPC(tenantID string) (vpcID, bridgeName string, err error)
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
