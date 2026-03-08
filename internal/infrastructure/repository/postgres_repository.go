package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"rds/internal/domain"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var (
	// ErrNotFound is returned when a record is not found
	ErrNotFound = errors.New("record not found")
	// ErrDuplicate is returned when a unique constraint is violated
	ErrDuplicate = errors.New("duplicate record")
)

// PostgresRepository implements domain.RepositoryPort
type PostgresRepository struct {
	db *sql.DB
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// CreateInstance inserts a new database instance into the database
func (r *PostgresRepository) CreateInstance(ctx context.Context, instance *domain.DBInstance) error {
	// Generate UUID if not provided
	if instance.ID == "" {
		instance.ID = uuid.New().String()
	}

	query := `
		INSERT INTO db_instances (id, name, engine, port, host, db_user, db_password, owner_id, container_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.ExecContext(ctx, query,
		instance.ID,
		instance.Name,
		instance.Engine,
		instance.Port,
		instance.Host,
		instance.User,
		instance.Password,
		instance.OwnerID,
		instance.ContainerID,
		instance.Status,
		instance.CreatedAt,
		instance.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create instance: %w", err)
	}

	return nil
}

// GetInstance retrieves a database instance by ID
func (r *PostgresRepository) GetInstance(ctx context.Context, id string) (*domain.DBInstance, error) {
	query := `
		SELECT id, name, engine, port, host, db_user, db_password, owner_id, container_id, status, created_at, updated_at
		FROM db_instances
		WHERE id = $1
	`

	instance := &domain.DBInstance{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&instance.ID,
		&instance.Name,
		&instance.Engine,
		&instance.Port,
		&instance.Host,
		&instance.User,
		&instance.Password,
		&instance.OwnerID,
		&instance.ContainerID,
		&instance.Status,
		&instance.CreatedAt,
		&instance.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	return instance, nil
}

// ListInstances retrieves all database instances for a specific owner
func (r *PostgresRepository) ListInstances(ctx context.Context, ownerID string) ([]*domain.DBInstance, error) {
	query := `
		SELECT id, name, engine, port, host, db_user, db_password, owner_id, container_id, status, created_at, updated_at
		FROM db_instances
		WHERE owner_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}
	defer rows.Close()

	var instances []*domain.DBInstance
	for rows.Next() {
		instance := &domain.DBInstance{}
		err := rows.Scan(
			&instance.ID,
			&instance.Name,
			&instance.Engine,
			&instance.Port,
			&instance.Host,
			&instance.User,
			&instance.Password,
			&instance.OwnerID,
			&instance.ContainerID,
			&instance.Status,
			&instance.CreatedAt,
			&instance.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan instance: %w", err)
		}
		instances = append(instances, instance)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return instances, nil
}

// UpdateInstance updates an existing database instance
func (r *PostgresRepository) UpdateInstance(ctx context.Context, instance *domain.DBInstance) error {
	query := `
		UPDATE db_instances
		SET name = $1, engine = $2, port = $3, host = $4, db_user = $5, db_password = $6, status = $7, updated_at = $8
		WHERE id = $9
	`

	result, err := r.db.ExecContext(ctx, query,
		instance.Name,
		instance.Engine,
		instance.Port,
		instance.Host,
		instance.User,
		instance.Password,
		instance.Status,
		instance.UpdatedAt,
		instance.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check update result: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteInstance deletes a database instance by ID
func (r *PostgresRepository) DeleteInstance(ctx context.Context, id string) error {
	query := `DELETE FROM db_instances WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check delete result: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// GetNextAvailablePort returns the next available port for a new instance
func (r *PostgresRepository) GetNextAvailablePort(ctx context.Context) (int, error) {
	query := `SELECT COALESCE(MAX(port), 5431) + 1 FROM db_instances`

	var port int
	err := r.db.QueryRowContext(ctx, query).Scan(&port)
	if err != nil {
		return 0, fmt.Errorf("failed to get next available port: %w", err)
	}

	return port, nil
}

// CreateAuditLog creates a new audit log entry
func (r *PostgresRepository) CreateAuditLog(ctx context.Context, log *domain.AuditLog) error {
	// Generate UUID if not provided
	if log.ID == "" {
		log.ID = uuid.New().String()
	}

	// Marshal details to JSON
	detailsJSON, err := json.Marshal(log.Details)
	if err != nil {
		return fmt.Errorf("failed to marshal audit log details: %w", err)
	}

	query := `
		INSERT INTO audit_logs (id, instance_id, action, actor_id, details, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err = r.db.ExecContext(ctx, query,
		log.ID,
		log.InstanceID,
		log.Action,
		log.ActorID,
		detailsJSON,
		log.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// ListAuditLogs retrieves audit logs for a specific instance
func (r *PostgresRepository) ListAuditLogs(ctx context.Context, instanceID string, limit int) ([]*domain.AuditLog, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, instance_id, action, actor_id, details, timestamp
		FROM audit_logs
		WHERE instance_id = $1
		ORDER BY timestamp DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, instanceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*domain.AuditLog
	for rows.Next() {
		log := &domain.AuditLog{}
		var detailsJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.InstanceID,
			&log.Action,
			&log.ActorID,
			&detailsJSON,
			&log.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		// Unmarshal details from JSON
		if len(detailsJSON) > 0 {
			err = json.Unmarshal(detailsJSON, &log.Details)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal audit log details: %w", err)
			}
		}

		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return logs, nil
}

// SaveConfiguration saves or updates a configuration parameter
func (r *PostgresRepository) SaveConfiguration(ctx context.Context, config *domain.Configuration) error {
	// Generate UUID if not provided
	if config.ID == "" {
		config.ID = uuid.New().String()
	}

	query := `
		INSERT INTO configurations (id, instance_id, parameter, value, applied_at, applied_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (instance_id, parameter)
		DO UPDATE SET value = EXCLUDED.value, applied_at = EXCLUDED.applied_at, applied_by = EXCLUDED.applied_by
	`

	_, err := r.db.ExecContext(ctx, query,
		config.ID,
		config.InstanceID,
		config.Parameter,
		config.Value,
		config.AppliedAt,
		config.AppliedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

// GetConfiguration retrieves all configurations for an instance
func (r *PostgresRepository) GetConfiguration(ctx context.Context, instanceID string) ([]*domain.Configuration, error) {
	query := `
		SELECT id, instance_id, parameter, value, applied_at, applied_by
		FROM configurations
		WHERE instance_id = $1
		ORDER BY parameter
	`

	rows, err := r.db.QueryContext(ctx, query, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get configurations: %w", err)
	}
	defer rows.Close()

	var configs []*domain.Configuration
	for rows.Next() {
		config := &domain.Configuration{}
		err := rows.Scan(
			&config.ID,
			&config.InstanceID,
			&config.Parameter,
			&config.Value,
			&config.AppliedAt,
			&config.AppliedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan configuration: %w", err)
		}
		configs = append(configs, config)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return configs, nil
}

// SaveConfigHistory saves a configuration change to history
func (r *PostgresRepository) SaveConfigHistory(ctx context.Context, history *domain.ConfigHistory) error {
	// Generate UUID if not provided
	if history.ID == "" {
		history.ID = uuid.New().String()
	}

	query := `
		INSERT INTO configuration_history (id, instance_id, parameter, old_value, new_value, changed_at, changed_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		history.ID,
		history.InstanceID,
		history.Parameter,
		history.OldValue,
		history.NewValue,
		history.ChangedAt,
		history.ChangedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to save config history: %w", err)
	}

	return nil
}

// GetConfigHistory retrieves configuration history for an instance
func (r *PostgresRepository) GetConfigHistory(ctx context.Context, instanceID string) ([]*domain.ConfigHistory, error) {
	query := `
		SELECT id, instance_id, parameter, old_value, new_value, changed_at, changed_by
		FROM configuration_history
		WHERE instance_id = $1
		ORDER BY changed_at DESC
		LIMIT 100
	`

	rows, err := r.db.QueryContext(ctx, query, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get config history: %w", err)
	}
	defer rows.Close()

	var history []*domain.ConfigHistory
	for rows.Next() {
		h := &domain.ConfigHistory{}
		err := rows.Scan(
			&h.ID,
			&h.InstanceID,
			&h.Parameter,
			&h.OldValue,
			&h.NewValue,
			&h.ChangedAt,
			&h.ChangedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan config history: %w", err)
		}
		history = append(history, h)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return history, nil
}
