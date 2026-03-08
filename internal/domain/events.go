package domain

import "time"

// CreateVPCEvent represents the payload for creating a VPC via NATS
type CreateVPCEvent struct {
	CorrelationID string `json:"correlation_id"`
	TenantID      string `json:"tenant_id"`
	VPCName       string `json:"vpc_name"`
	RequestedBy   string `json:"requested_by"`
}

// VPCCreatedEvent is received when a VPC is successfully provisioned
type VPCCreatedEvent struct {
	CorrelationID string    `json:"correlation_id"`
	TenantID      string    `json:"tenant_id"`
	VPCID         string    `json:"vpc_id"`
	CIDRBlock     string    `json:"cidr_block"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}
