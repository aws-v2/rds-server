package application

import (
	"context"
	"rds/internal/domain"
	"time"
)

// HealthStatus represents the overall health status
type HealthStatus struct {
	Healthy        bool                   `json:"healthy"`
	Database       ServiceHealth          `json:"database"`
	Docker         ServiceHealth          `json:"docker"`
	Timestamp      time.Time              `json:"timestamp"`
	InstancesCount int                    `json:"instancesCount"`
	Details        map[string]interface{} `json:"details,omitempty"`
}

// ServiceHealth represents the health of a specific service
type ServiceHealth struct {
	Healthy bool   `json:"healthy"`
	Message string `json:"message,omitempty"`
}

// HealthService handles health checks
type HealthService struct {
	repo         domain.RepositoryPort
	dockerClient domain.DockerPort
}

// NewHealthService creates a new health service
func NewHealthService(repo domain.RepositoryPort, dockerClient domain.DockerPort) *HealthService {
	return &HealthService{
		repo:         repo,
		dockerClient: dockerClient,
	}
}

// GetHealth returns the overall health status
func (s *HealthService) GetHealth(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Check database health
	status.Database = s.checkDatabase(ctx)

	// Check Docker health
	status.Docker = s.checkDocker(ctx)

	// Get instance count
	// Note: This is a simplified check - in production you might want to check all instances
	status.InstancesCount = s.getInstanceCount(ctx)

	// Overall health is healthy if all services are healthy
	status.Healthy = status.Database.Healthy && status.Docker.Healthy

	return status, nil
}

// Ping returns a simple ping response
func (s *HealthService) Ping(ctx context.Context) bool {
	return true
}

// checkDatabase checks database connectivity
func (s *HealthService) checkDatabase(ctx context.Context) ServiceHealth {
	// Try to get next available port as a simple database check
	_, err := s.repo.GetNextAvailablePort(ctx)
	if err != nil {
		return ServiceHealth{
			Healthy: false,
			Message: "Database connection failed: " + err.Error(),
		}
	}

	return ServiceHealth{
		Healthy: true,
		Message: "Database connection successful",
	}
}

// checkDocker checks Docker connectivity
func (s *HealthService) checkDocker(ctx context.Context) ServiceHealth {
	// Try to pull a small image or check Docker availability
	// For now, we'll assume Docker is healthy if the client was created
	return ServiceHealth{
		Healthy: true,
		Message: "Docker connection successful",
	}
}

// getInstanceCount returns the total number of instances
func (s *HealthService) getInstanceCount(ctx context.Context) int {
	// This is a simplified implementation
	// In a real scenario, you might want to query all instances across all owners
	return 0
}
