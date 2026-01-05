package domain

import "time"

// AuditAction represents the type of action performed on an instance
type AuditAction string

const (
	ActionCreate AuditAction = "create"
	ActionStart  AuditAction = "start"
	ActionStop   AuditAction = "stop"
	ActionDelete AuditAction = "delete"
	ActionUpdate AuditAction = "update"
)

// AuditLog represents an audit trail entry for instance operations
type AuditLog struct {
	ID         string                 `json:"id"`
	InstanceID string                 `json:"instanceId"`
	Action     AuditAction            `json:"action"`
	ActorID    string                 `json:"actorId"` // User who performed the action
	Details    map[string]interface{} `json:"details"` // Additional context about the action
	Timestamp  time.Time              `json:"timestamp"`
}

// NewAuditLog creates a new audit log entry
func NewAuditLog(instanceID, actorID string, action AuditAction, details map[string]interface{}) *AuditLog {
	return &AuditLog{
		InstanceID: instanceID,
		Action:     action,
		ActorID:    actorID,
		Details:    details,
		Timestamp:  time.Now(),
	}
}
