package application

import (
	"context"
	"fmt"
	"rds/internal/domain"
	"rds/internal/utils"

	"github.com/google/uuid"
)

// SnapshotService manages RDS point-in-time backups
type SnapshotService struct {
	repo         domain.RepositoryPort
	dockerClient domain.DockerPort
	region       string
}

func NewSnapshotService(repo domain.RepositoryPort, dockerClient domain.DockerPort, region string) *SnapshotService {
	return &SnapshotService{
		repo:         repo,
		dockerClient: dockerClient,
		region:       region,
	}
}

// CreateSnapshotRequest handles metadata for taking a backup
type CreateSnapshotRequest struct {
	Name       string
	DatabaseID string
	AccountID  string
}

// CreateSnapshot captures the state of a database volume
func (s *SnapshotService) CreateSnapshot(ctx context.Context, req CreateSnapshotRequest) (*domain.Snapshot, error) {
	// 1. Verify DB is valid and belongs to user
	db, err := s.repo.GetDatabase(ctx, req.DatabaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database %s: %w", req.DatabaseID, err)
	}
	if db.AccountID != req.AccountID {
		return nil, fmt.Errorf("unauthorized")
	}

	snapID := uuid.New().String()
	arn := utils.GenerateSnapshotARN(s.region, req.AccountID, snapID)

	snap := &domain.Snapshot{
		ID:         snapID,
		AccountID:  req.AccountID,
		ARN:        arn,
		Name:       req.Name,
		DatabaseID: db.ID,
		VolumeID:   fmt.Sprintf("claudedb-vol-%s", db.ID), // In a real system, DB has a formal VolumeID
		SizeGB:     10,                                    // Mock size or fetch from docker volume inspect
		Status:     domain.SnapshotStatusCreating,
	}

	if err := s.repo.CreateSnapshot(ctx, snap); err != nil {
		return nil, err
	}

	// 2. Perform actual backup (Mocked for now)
	// In reality: Connect to Docker container -> Exec pg_dump -> Pipe to Object Storage OR clone ZFS/Docker volume
	fmt.Printf("MOCK: Taking snapshot of DB %s volume...\n", db.ID)

	// 3. Mark Available
	if err := s.repo.UpdateSnapshotStatus(ctx, snap.ID, domain.SnapshotStatusAvailable); err != nil {
		return nil, err
	}
	snap.Status = domain.SnapshotStatusAvailable

	return snap, nil
}

// ListSnapshots retrieves all snapshots for an account
func (s *SnapshotService) ListSnapshots(ctx context.Context, accountID string, databaseID *string) ([]*domain.Snapshot, error) {
	return s.repo.ListSnapshots(ctx, accountID, databaseID)
}

// DeleteSnapshot marks a backup as deleted
func (s *SnapshotService) DeleteSnapshot(ctx context.Context, id, accountID string) error {
	snap, err := s.repo.GetSnapshot(ctx, id)
	if err != nil {
		return err
	}
	if snap.AccountID != accountID {
		return fmt.Errorf("unauthorized")
	}

	if err := s.repo.UpdateSnapshotStatus(ctx, id, domain.SnapshotStatusDeleting); err != nil {
		return err
	}

	// Mock cleanup of S3 / actual storage files
	fmt.Printf("MOCK: Deleting snapshot %s from physical storage\n", snap.ID)

	return s.repo.DeleteSnapshot(ctx, id)
}

// RestoreDatabaseRequest defines the inputs for creating a new DB from a snapshot
type RestoreDatabaseRequest struct {
	SnapshotID string
	NewName    string
	AccountID  string
}

// RestoreDatabase provisions a brand new Database using a Snapshot's data as the seed
func (s *SnapshotService) RestoreDatabase(ctx context.Context, req RestoreDatabaseRequest) (interface{}, error) {
	// 1. Validate snapshot
	snap, err := s.repo.GetSnapshot(ctx, req.SnapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}
	if snap.AccountID != req.AccountID {
		return nil, fmt.Errorf("unauthorized")
	}
	if snap.Status != domain.SnapshotStatusAvailable {
		return nil, fmt.Errorf("snapshot is not available for restore: %s", snap.Status)
	}

	// 2. We'd call ClaudeDBService.CreateDatabase logic here, injecting the snapshot's data volume
	// rather than an empty volume. For this stub, we return an error indicating it requires wiring
	// to ClaudeDBService or an orchestration layer.
	return nil, fmt.Errorf("RESTORE logic requires orchestration layer wiring. Not implemented in isolated stub.")
}
