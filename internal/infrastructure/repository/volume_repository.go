package repository

import (
	"context"
	"database/sql"
	"fmt"
	"rds/internal/domain"

	"github.com/google/uuid"
)

// CreateVolume inserts a new volume record
func (r *PostgresRepository) CreateVolume(ctx context.Context, vol *domain.Volume) error {
	if vol.ID == "" {
		vol.ID = uuid.New().String()
	}

	query := `
		INSERT INTO volumes (id, account_id, arn, name, size_gb, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		vol.ID, vol.AccountID, vol.ARN, vol.Name, vol.SizeGB, vol.Status,
	).Scan(&vol.CreatedAt, &vol.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}

	return nil
}

// GetVolume retrieves a volume by ID
func (r *PostgresRepository) GetVolume(ctx context.Context, id string) (*domain.Volume, error) {
	query := `
		SELECT id, account_id, arn, name, size_gb, status, created_at, updated_at, deleted_at
		FROM volumes WHERE id = $1
	`
	vol := &domain.Volume{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&vol.ID, &vol.AccountID, &vol.ARN, &vol.Name, &vol.SizeGB, &vol.Status,
		&vol.CreatedAt, &vol.UpdatedAt, &vol.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get volume: %w", err)
	}
	return vol, nil
}

// ListVolumes lists all volumes for an account
func (r *PostgresRepository) ListVolumes(ctx context.Context, accountID string) ([]*domain.Volume, error) {
	query := `
		SELECT id, account_id, arn, name, size_gb, status, created_at, updated_at, deleted_at
		FROM volumes 
		WHERE account_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes: %w", err)
	}
	defer rows.Close()

	var volumes []*domain.Volume
	for rows.Next() {
		vol := &domain.Volume{}
		if err := rows.Scan(
			&vol.ID, &vol.AccountID, &vol.ARN, &vol.Name, &vol.SizeGB, &vol.Status,
			&vol.CreatedAt, &vol.UpdatedAt, &vol.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan volume: %w", err)
		}
		volumes = append(volumes, vol)
	}
	return volumes, nil
}

// UpdateVolumeStatus updates the status of a volume
func (r *PostgresRepository) UpdateVolumeStatus(ctx context.Context, id string, status domain.VolumeStatus) error {
	query := `UPDATE volumes SET status = $1, updated_at = NOW() WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update volume status: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteVolume marks a volume as deleted
func (r *PostgresRepository) DeleteVolume(ctx context.Context, id string) error {
	query := `UPDATE volumes SET status = 'DELETED', deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete volume: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateSnapshot inserts a new snapshot record
func (r *PostgresRepository) CreateSnapshot(ctx context.Context, snap *domain.Snapshot) error {
	if snap.ID == "" {
		snap.ID = uuid.New().String()
	}

	query := `
		INSERT INTO snapshots (id, account_id, arn, name, database_id, volume_id, size_gb, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		snap.ID, snap.AccountID, snap.ARN, snap.Name, snap.DatabaseID, snap.VolumeID, snap.SizeGB, snap.Status,
	).Scan(&snap.CreatedAt, &snap.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}

	return nil
}

// GetSnapshot retrieves a snapshot by ID
func (r *PostgresRepository) GetSnapshot(ctx context.Context, id string) (*domain.Snapshot, error) {
	query := `
		SELECT id, account_id, arn, name, database_id, volume_id, size_gb, status, created_at, updated_at, deleted_at
		FROM snapshots WHERE id = $1
	`
	snap := &domain.Snapshot{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&snap.ID, &snap.AccountID, &snap.ARN, &snap.Name, &snap.DatabaseID, &snap.VolumeID, &snap.SizeGB, &snap.Status,
		&snap.CreatedAt, &snap.UpdatedAt, &snap.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}
	return snap, nil
}

// ListSnapshots lists all snapshots for an account, optionally filtering by database
func (r *PostgresRepository) ListSnapshots(ctx context.Context, accountID string, databaseID *string) ([]*domain.Snapshot, error) {
	var query string
	var args []interface{}

	if databaseID != nil {
		query = `
			SELECT id, account_id, arn, name, database_id, volume_id, size_gb, status, created_at, updated_at, deleted_at
			FROM snapshots 
			WHERE account_id = $1 AND database_id = $2 AND deleted_at IS NULL
			ORDER BY created_at DESC
		`
		args = []interface{}{accountID, *databaseID}
	} else {
		query = `
			SELECT id, account_id, arn, name, database_id, volume_id, size_gb, status, created_at, updated_at, deleted_at
			FROM snapshots 
			WHERE account_id = $1 AND deleted_at IS NULL
			ORDER BY created_at DESC
		`
		args = []interface{}{accountID}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []*domain.Snapshot
	for rows.Next() {
		snap := &domain.Snapshot{}
		if err := rows.Scan(
			&snap.ID, &snap.AccountID, &snap.ARN, &snap.Name, &snap.DatabaseID, &snap.VolumeID, &snap.SizeGB, &snap.Status,
			&snap.CreatedAt, &snap.UpdatedAt, &snap.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan snapshot: %w", err)
		}
		snapshots = append(snapshots, snap)
	}
	return snapshots, nil
}

// UpdateSnapshotStatus updates the status of a snapshot
func (r *PostgresRepository) UpdateSnapshotStatus(ctx context.Context, id string, status domain.SnapshotStatus) error {
	query := `UPDATE snapshots SET status = $1, updated_at = NOW() WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update snapshot status: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteSnapshot marks a snapshot as deleted
func (r *PostgresRepository) DeleteSnapshot(ctx context.Context, id string) error {
	query := `UPDATE snapshots SET status = 'DELETED', deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete snapshot: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
