package domain

import "time"

// VPC represents a Virtual Private Cloud from the network service
type VPC struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	CIDRBlock  string    `json:"cidr_block"`
	BridgeName string    `json:"bridge_name"`
	TenantID   string    `json:"tenant_id"`
	Status     string    `json:"status"`
	IsDefault  bool      `json:"is_default"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
