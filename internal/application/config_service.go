package application

import (
	"context"
	"rds/internal/domain"
	"rds/internal/interfaces"
)

// ConfigService handles configuration management
type ConfigService struct {
	repo         interfaces.RepositoryPort
	auditService *AuditService
}

// NewConfigService creates a new configuration service
func NewConfigService(repo interfaces.RepositoryPort, auditService *AuditService) *ConfigService {
	return &ConfigService{
		repo:         repo,
		auditService: auditService,
	}
}

// SetConfiguration sets a configuration parameter for an instance
func (s *ConfigService) SetConfiguration(ctx context.Context, instanceID, parameter, value, appliedBy string) error {
	// Get existing config to save history
	existingConfigs, err := s.repo.GetConfiguration(ctx, instanceID)
	if err != nil {
		return err
	}

	var oldValue string
	for _, cfg := range existingConfigs {
		if cfg.Parameter == parameter {
			oldValue = cfg.Value
			break
		}
	}

	// Save new configuration
	config := domain.NewConfiguration(instanceID, parameter, value, appliedBy)
	if err := s.repo.SaveConfiguration(ctx, config); err != nil {
		return err
	}

	// Save configuration history
	if oldValue != "" {
		history := domain.NewConfigHistory(instanceID, parameter, oldValue, value, appliedBy)
		if err := s.repo.SaveConfigHistory(ctx, history); err != nil {
			return err
		}
	}

	// Log the configuration change
	details := map[string]interface{}{
		"parameter": parameter,
		"oldValue":  oldValue,
		"newValue":  value,
	}
	_ = s.auditService.LogAction(ctx, instanceID, appliedBy, domain.ActionUpdate, details)

	return nil
}

// GetConfiguration retrieves all configurations for an instance
func (s *ConfigService) GetConfiguration(ctx context.Context, instanceID string) ([]*domain.Configuration, error) {
	return s.repo.GetConfiguration(ctx, instanceID)
}

// GetConfigurationHistory retrieves configuration history for an instance
func (s *ConfigService) GetConfigurationHistory(ctx context.Context, instanceID string) ([]*domain.ConfigHistory, error) {
	return s.repo.GetConfigHistory(ctx, instanceID)
}
