package application

import (
	"context"
	"rds/internal/domain"
	"rds/internal/interfaces"
)

// AuditService handles audit logging operations
type AuditService struct {
	repo interfaces.RepositoryPort
}

// NewAuditService creates a new audit service
func NewAuditService(repo interfaces.RepositoryPort) *AuditService {
	return &AuditService{
		repo: repo,
	}
}

// LogAction creates an audit log entry for an action
func (s *AuditService) LogAction(ctx context.Context, instanceID, actorID string, action domain.AuditAction, details map[string]interface{}) error {
	log := domain.NewAuditLog(instanceID, actorID, action, details)
	return s.repo.CreateAuditLog(ctx, log)
}

// GetInstanceLogs retrieves audit logs for a specific instance
func (s *AuditService) GetInstanceLogs(ctx context.Context, instanceID string, limit int) ([]*domain.AuditLog, error) {
	return s.repo.ListAuditLogs(ctx, instanceID, limit)
}
