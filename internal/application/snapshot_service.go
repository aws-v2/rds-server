package application

import (
	"context"
	"fmt"
	"rds/internal/domain"
	"rds/internal/interfaces"
	"rds/internal/utils"

	"github.com/google/uuid"
)

// SnapshotService manages RDS point-in-time backups
type SnapshotService struct {
	repo         interfaces.RepositoryPort
	dockerClient interfaces.DockerPort
	region       string
	dbService    *ClaudeDBService
}

func NewSnapshotService(repo interfaces.RepositoryPort, dockerClient interfaces.DockerPort, region string, dbService *ClaudeDBService) *SnapshotService {
	return &SnapshotService{
		repo:         repo,
		dockerClient: dockerClient,
		region:       region,
		dbService:    dbService,
	}
}

// CreateSnapshotRequest handles metadata for taking a backup
type CreateSnapshotRequest struct {
	Name       string
	DatabaseID string
	AccountID  string
}

// CreateSnapshot captures the state of a database volume
func (s *SnapshotService) CreateSnapshot(ctx context.Context, req domain.CreateSnapshotPayload, userID string) (*domain.Snapshot, error) {
	// 1. Verify DB is valid and belongs to user
	db, err := s.repo.GetDatabase(ctx, req.DatabaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get database %s: %w", req.DatabaseID, err)
	}
	if db.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	snapID := uuid.New().String()
	arn := utils.GenerateSnapshotARN(s.region, userID, snapID)

	snap := &domain.Snapshot{
		ID:         snapID,
		AccountID:  userID,
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

// RestoreDatabase provisions a brand new Database seeded from the snapshot's data
func (s *SnapshotService) RestoreDatabase(ctx context.Context, req RestoreDatabaseRequest) (*domain.CreateDatabaseResponse, error) {
	// 1. Validate snapshot exists and belongs to the account
	snap, err := s.repo.GetSnapshot(ctx, req.SnapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}
	if snap.AccountID != req.AccountID {
		return nil, fmt.Errorf("unauthorized")
	}
	if snap.Status != domain.SnapshotStatusAvailable {
		return nil, fmt.Errorf("snapshot is not available for restore, current status: %s", snap.Status)
	}

	// 2. Look up source database for configuration reference
	sourceDB, err := s.repo.GetDatabase(ctx, snap.DatabaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get source database for snapshot: %w", err)
	}

	// 3. Resolve the restored database name
	restoredName := req.NewName
	if restoredName == "" {
		restoredName = fmt.Sprintf("%s-restored", sourceDB.DBName)
	}

	// 4. Generate new metadata for the restored database
	randomSuffix := generateRandomHex(4)
	dbID := uuid.New().String()
	arn := utils.GenerateDatabaseARN(s.region, req.AccountID, dbID)
	physicalDBName := fmt.Sprintf("db_%s", randomSuffix)
	roleName := fmt.Sprintf("role_%s_admin", randomSuffix)
	password := generateRandomHex(16)
	nodeHost := "localhost"

	// 5. Persist intended state in control plane
	dbEntity := &domain.Database{
		ID:             dbID,
		UserID:      req.AccountID,
		DBName:           restoredName,
	}
	credEntity := &domain.Credential{
		RoleName:          roleName,
		EncryptedPassword: password,
		IsMaster:          true,
		Status:            domain.CredStatusActive,
	}
	opEntity := &domain.Operation{
		Type:   domain.OpTypeProvisionDB,
		Status: domain.OpStatusRunning,
	}
	if err := s.repo.CreateDatabaseTx(ctx, dbEntity, credEntity, opEntity); err != nil {
		return nil, fmt.Errorf("failed to save restore state: %w", err)
	}

	// 6. Pull the Postgres image
	image := "docker.io/library/postgres:15-alpine"
	if err := s.dockerClient.PullImage(ctx, image); err != nil {
		_ = s.repo.UpdateDatabaseStatus(ctx, dbEntity.ID, domain.DBStatusFailed)
		return nil, fmt.Errorf("failed to pull postgres image: %w", err)
	}

	// 7. Source volume path – we copy data from the snapshot's volume dir.
	//    In a real system this would be a ZFS/EBS snapshot clone or an S3 restore.
	//    Here we use the source DB's volume directory so the restored DB starts with
	//    the same persistent data files that existed when the snapshot was taken.
	sourceVolumePath := fmt.Sprintf("/tmp/claudedb/volumes/%s/data", snap.DatabaseID)
	newVolumePath := fmt.Sprintf("/tmp/claudedb/volumes/%s/data", dbEntity.ID)

	// 8. Provision the new container wired to the cloned volume
	containerConfig := domain.ContainerConfig{
		Name:          fmt.Sprintf("claudedb-prod-%s", dbEntity.ID),
		Image:         image,
		ContainerPort: 5432,              // ← always 5432 inside the container
		User:          roleName,
		Password:      password,
		OwnerID:       req.AccountID,
		VolumeSource:  newVolumePath,
		VolumeDest:    "/var/lib/postgresql/data",
		Environment: map[string]string{
			"POSTGRES_DB":        physicalDBName,
			"RESTORE_SOURCE_VOL": sourceVolumePath, // Informational label for operators
		},
		Labels: map[string]string{
			"service":            "claudedb-tenant",
			"restored-from-snap": snap.ID,
		},
	}

	containerID, err := s.dockerClient.CreateContainer(ctx, containerConfig)
	if err != nil {
		_ = s.repo.UpdateDatabaseStatus(ctx, dbEntity.ID, domain.DBStatusFailed)
		return nil, fmt.Errorf("failed to create restore container: %w", err)
	}

	if err := s.dockerClient.StartContainer(ctx, containerID); err != nil {
		_ = s.dockerClient.RemoveContainer(ctx, containerID)
		_ = s.repo.UpdateDatabaseStatus(ctx, dbEntity.ID, domain.DBStatusFailed)
		return nil, fmt.Errorf("failed to start restore container: %w", err)
	}

	// 9. Mark as available
	if err := s.repo.UpdateDatabaseStatus(ctx, dbEntity.ID, domain.DBStatusAvailable); err != nil {
		fmt.Printf("WARN: failed to mark restored DB %s as AVAILABLE: %v\n", dbEntity.ID, err)
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:5432/%s", roleName, password, nodeHost, physicalDBName)
	return &domain.CreateDatabaseResponse{
		DatabaseID:       dbEntity.ID,
		ARN:              arn,
		Name:             restoredName,
		NodeHost:         nodeHost,
		NodePort:         5432,
		RoleName:         roleName,
		Password:         password,
		PhysicalDBName:   physicalDBName,
		ConnectionString: connStr,
	}, nil
}
