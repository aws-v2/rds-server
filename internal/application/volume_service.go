package application

import (
	"context"
	"fmt"
	"rds/internal/domain"
	"rds/internal/interfaces"
	"rds/internal/utils"

	"github.com/google/uuid"
)

// VolumeService manages RDS storage volumes
type VolumeService struct {
	repo         interfaces.RepositoryPort
	dockerClient interfaces.DockerPort
	region       string
}

func NewVolumeService(repo interfaces.RepositoryPort, dockerClient interfaces.DockerPort, region string) *VolumeService {
	return &VolumeService{
		repo:         repo,
		dockerClient: dockerClient,
		region:       region,
	}
}

// CreateVolumeRequest represents the request to create a volume
type CreateVolumeRequest struct {
	Name      string
	SizeGB    int
	AccountID string
}

// CreateVolume allocates a new storage volume
func (s *VolumeService) CreateVolume(ctx context.Context, req CreateVolumeRequest) (*domain.Volume, error) {
	volID := uuid.New().String()
	arn := utils.GenerateVolumeARN(s.region, req.AccountID, volID)

	vol := &domain.Volume{
		ID:        volID,
		AccountID: req.AccountID,
		ARN:       arn,
		Name:      req.Name,
		SizeGB:    req.SizeGB,
		Status:    domain.VolumeStatusCreating,
	}

	// 1. Persist initial state
	if err := s.repo.CreateVolume(ctx, vol); err != nil {
		return nil, fmt.Errorf("failed to save volume state: %w", err)
	}

	// 2. Instruct Docker to create the volume
	// We prefix the name so it doesn't conflict easily
	dockerVolName := fmt.Sprintf("claudedb-vol-%s", vol.ID)
	if err := s.dockerClient.CreateVolume(ctx, dockerVolName); err != nil {
		_ = s.repo.UpdateVolumeStatus(ctx, vol.ID, domain.VolumeStatusError)
		return nil, fmt.Errorf("failed to provision docker volume: %w", err)
	}

	// 3. Mark as available
	if err := s.repo.UpdateVolumeStatus(ctx, vol.ID, domain.VolumeStatusAvailable); err != nil {
		return nil, fmt.Errorf("failed to update volume status to available: %w", err)
	}
	vol.Status = domain.VolumeStatusAvailable

	return vol, nil
}

// GetVolume retrieves volume details
func (s *VolumeService) GetVolume(ctx context.Context, id, accountID string) (*domain.Volume, error) {
	vol, err := s.repo.GetVolume(ctx, id)
	if err != nil {
		return nil, err
	}
	if vol.AccountID != accountID {
		return nil, fmt.Errorf("unauthorized")
	}
	return vol, nil
}

// ListVolumes retrieves all volumes for an account
func (s *VolumeService) ListVolumes(ctx context.Context, accountID string) ([]*domain.Volume, error) {
	return s.repo.ListVolumes(ctx, accountID)
}

// DeleteVolume removes a volume and its backing storage
func (s *VolumeService) DeleteVolume(ctx context.Context, id, accountID string) error {
	vol, err := s.GetVolume(ctx, id, accountID)
	if err != nil {
		return err
	}

	if vol.Status == domain.VolumeStatusInUse {
		return fmt.Errorf("cannot delete an in-use volume")
	}

	if err := s.repo.UpdateVolumeStatus(ctx, id, domain.VolumeStatusDeleting); err != nil {
		return err
	}

	dockerVolName := fmt.Sprintf("claudedb-vol-%s", vol.ID)
	if err := s.dockerClient.RemoveVolume(ctx, dockerVolName); err != nil {
		// Log error but might still soft delete if docker volume was already gone
		fmt.Printf("Warning: failed to remove docker volume: %v\n", err)
	}

	if err := s.repo.DeleteVolume(ctx, id); err != nil {
		return fmt.Errorf("failed to mark volume as deleted: %w", err)
	}

	return nil
}
