package application

import (
	"context"
	"fmt"
	"rds/internal/domain"
	"time"
)

// CreateInstanceRequest represents the request to create a new instance
type CreateInstanceRequest struct {
	Name     string
	User     string
	Password string
	OwnerID  string
}

// InstanceService handles database instance lifecycle management
type InstanceService struct {
	repo         domain.RepositoryPort
	dockerClient domain.DockerPort
	auditService *AuditService
}

// NewInstanceService creates a new instance service
func NewInstanceService(repo domain.RepositoryPort, dockerClient domain.DockerPort, auditService *AuditService) *InstanceService {
	return &InstanceService{
		repo:         repo,
		dockerClient: dockerClient,
		auditService: auditService,
	}
}

// CreateInstance creates a new database instance
func (s *InstanceService) CreateInstance(ctx context.Context, req CreateInstanceRequest) (*domain.DBInstance, error) {
	// Validate request
	if req.Name == "" || req.User == "" || req.Password == "" || req.OwnerID == "" {
		return nil, fmt.Errorf("missing required fields")
	}

	// Get next available port
	port, err := s.repo.GetNextAvailablePort(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get available port: %w", err)
	}

	// Pull PostgreSQL image
	image := "docker.io/library/postgres:17"
	if err := s.dockerClient.PullImage(ctx, image); err != nil {
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}

	// Create container configuration
	containerConfig := domain.ContainerConfig{
		Name:     req.Name,
		Image:    image,
	    HostPort:     port,  // ← 1000, 1001... allocated per instance
    ContainerPort: 5432,              // ← always 5432 inside the container
		User:     req.User,
		Password: req.Password,
		OwnerID:  req.OwnerID,
		Environment: map[string]string{
			"POSTGRES_DB": req.Name,
		},
		Labels: map[string]string{
			"service": "mini-rds",
		},
	}

	// Create container
	containerID, err := s.dockerClient.CreateContainer(ctx, containerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := s.dockerClient.StartContainer(ctx, containerID); err != nil {
		// Try to cleanup the container if start fails
		_ = s.dockerClient.RemoveContainer(ctx, containerID)
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Create instance record
	now := time.Now()
	instance := &domain.DBInstance{
		Name:        req.Name,
		Engine:      image,
		Port:        port,
		Host:        "localhost",
		User:        req.User,
		Password:    req.Password,
		OwnerID:     req.OwnerID,
		ContainerID: containerID,
		Status:      domain.StatusRunning,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Save to database
	if err := s.repo.CreateInstance(ctx, instance); err != nil {
		// Cleanup container if database save fails
		_ = s.dockerClient.StopContainer(ctx, containerID)
		_ = s.dockerClient.RemoveContainer(ctx, containerID)
		return nil, fmt.Errorf("failed to save instance: %w", err)
	}

	// Log the creation
	details := map[string]interface{}{
		"name":   req.Name,
		"engine": image,
		"port":   port,
	}
	_ = s.auditService.LogAction(ctx, instance.ID, req.OwnerID, domain.ActionCreate, details)

	return instance, nil
}

// GetInstance retrieves a database instance by ID
func (s *InstanceService) GetInstance(ctx context.Context, id, ownerID string) (*domain.DBInstance, error) {
	instance, err := s.repo.GetInstance(ctx, id)
	if err != nil {
		return nil, err
	}

	// Verify ownership
	if instance.OwnerID != ownerID {
		return nil, fmt.Errorf("unauthorized: instance does not belong to user")
	}

	return instance, nil
}

// ListInstances retrieves all instances for a user
func (s *InstanceService) ListInstances(ctx context.Context, ownerID string) ([]*domain.DBInstance, error) {
	return s.repo.ListInstances(ctx, ownerID)
}

// DeleteInstance deletes a database instance
func (s *InstanceService) DeleteInstance(ctx context.Context, id, ownerID string) error {
	// Get instance to verify ownership and get container ID
	instance, err := s.repo.GetInstance(ctx, id)
	if err != nil {
		return err
	}

	// Verify ownership
	if instance.OwnerID != ownerID {
		return fmt.Errorf("unauthorized: instance does not belong to user")
	}

	// Stop and remove container
	if err := s.dockerClient.StopContainer(ctx, instance.ContainerID); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	if err := s.dockerClient.RemoveContainer(ctx, instance.ContainerID); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Delete from database
	if err := s.repo.DeleteInstance(ctx, id); err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	// Log the deletion
	details := map[string]interface{}{
		"name":        instance.Name,
		"containerId": instance.ContainerID,
	}
	_ = s.auditService.LogAction(ctx, id, ownerID, domain.ActionDelete, details)

	return nil
}

// StartInstance starts a stopped database instance
func (s *InstanceService) StartInstance(ctx context.Context, id, ownerID string) error {
	// Get instance
	instance, err := s.repo.GetInstance(ctx, id)
	if err != nil {
		return err
	}

	// Verify ownership
	if instance.OwnerID != ownerID {
		return fmt.Errorf("unauthorized: instance does not belong to user")
	}

	// Check if already running
	if instance.IsRunning() {
		return fmt.Errorf("instance is already running")
	}

	// Start container
	if err := s.dockerClient.StartContainer(ctx, instance.ContainerID); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// Update instance status
	instance.MarkAsRunning()
	if err := s.repo.UpdateInstance(ctx, instance); err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	// Log the action
	_ = s.auditService.LogAction(ctx, id, ownerID, domain.ActionStart, nil)

	return nil
}

// StopInstance stops a running database instance
func (s *InstanceService) StopInstance(ctx context.Context, id, ownerID string) error {
	// Get instance
	instance, err := s.repo.GetInstance(ctx, id)
	if err != nil {
		return err
	}

	// Verify ownership
	if instance.OwnerID != ownerID {
		return fmt.Errorf("unauthorized: instance does not belong to user")
	}

	// Check if already stopped
	if instance.IsStopped() {
		return fmt.Errorf("instance is already stopped")
	}

	// Stop container
	if err := s.dockerClient.StopContainer(ctx, instance.ContainerID); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Update instance status
	instance.MarkAsStopped()
	if err := s.repo.UpdateInstance(ctx, instance); err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}

	// Log the action
	_ = s.auditService.LogAction(ctx, id, ownerID, domain.ActionStop, nil)

	return nil
}
