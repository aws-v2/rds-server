package repository

import (
	"context"
	"database/sql"
	"fmt"
	"rds/internal/domain"

	"github.com/google/uuid"
)

// CreateDatabaseTx executes the provisioning flow using the gap-finder
func (r *PostgresRepository) CreateDatabaseTx(ctx context.Context, db *domain.Database, cred *domain.Credential, op *domain.Operation) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if db.ID == "" {
		db.ID = uuid.New().String()
	}

	// 1. Insert Database (port is always 5432 — containers use VPC bridge networking)
	var idempotencyVal interface{}
	if db.IdempotencyKey != nil {
		idempotencyVal = *db.IdempotencyKey
	}

	queryDb := `
		INSERT INTO databases (id, account_id, arn, name, physical_db_name, node_host, node_port, public_port, private_ip, vpc_id, status, idempotency_key, created_at, updated_at)
		VALUES ($2, $3, $8, $4, $5, $1, 5432, $11, $9, $10, $6, $7, NOW(), NOW())
		RETURNING node_port, created_at, updated_at
	`

	err = tx.QueryRowContext(ctx, queryDb,
		db.NodeHost,       // $1
		db.ID,             // $2
		db.AccountID,      // $3
		db.Name,           // $4
		db.PhysicalDBName, // $5
		db.Status,         // $6
		idempotencyVal,    // $7
		db.ARN,            // $8
		db.PrivateIP,      // $9
		db.VPCID,          // $10
		db.PublicPort,     // $11
	).Scan(&db.NodePort, &db.CreatedAt, &db.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert database: %w", err)
	}

	// 2. Insert Credential
	if cred.ID == "" {
		cred.ID = uuid.New().String()
	}
	cred.DatabaseID = db.ID

	queryCred := `
		INSERT INTO credentials (id, database_id, role_name, encrypted_password, is_master, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	err = tx.QueryRowContext(ctx, queryCred,
		cred.ID,
		cred.DatabaseID,
		cred.RoleName,
		cred.EncryptedPassword,
		cred.IsMaster,
		cred.Status,
	).Scan(&cred.CreatedAt, &cred.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert credential: %w", err)
	}

	// 3. Insert Operation
	if op.ID == "" {
		op.ID = uuid.New().String()
	}
	op.DatabaseID = db.ID

	queryOp := `
		INSERT INTO operations (id, database_id, type, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING created_at, updated_at
	`
	err = tx.QueryRowContext(ctx, queryOp,
		op.ID,
		op.DatabaseID,
		op.Type,
		op.Status,
	).Scan(&op.CreatedAt, &op.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert operation: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetDatabase(ctx context.Context, id string) (*domain.Database, error) {
	query := `
		SELECT id, account_id, arn, name, physical_db_name, node_host, node_port, public_port, status, idempotency_key, private_ip, vpc_id, created_at, updated_at, deleted_at
		FROM databases WHERE id = $1 AND deleted_at IS NULL
	`
	db := &domain.Database{}
	var idempotencyKey *string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&db.ID, &db.AccountID, &db.ARN, &db.Name, &db.PhysicalDBName, &db.NodeHost, &db.NodePort, &db.PublicPort,
		&db.Status, &idempotencyKey, &db.PrivateIP, &db.VPCID, &db.CreatedAt, &db.UpdatedAt, &db.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}
	db.IdempotencyKey = idempotencyKey
	return db, nil
}

func (r *PostgresRepository) GetDatabaseByIdempotencyKey(ctx context.Context, accountID, idempotencyKey string) (*domain.Database, error) {
	query := `
		SELECT id, account_id, arn, name, physical_db_name, node_host, node_port, public_port, status, idempotency_key, private_ip, vpc_id, created_at, updated_at, deleted_at
		FROM databases WHERE account_id = $1 AND idempotency_key = $2 AND deleted_at IS NULL
	`
	db := &domain.Database{}
	var idKey *string
	err := r.db.QueryRowContext(ctx, query, accountID, idempotencyKey).Scan(
		&db.ID, &db.AccountID, &db.ARN, &db.Name, &db.PhysicalDBName, &db.NodeHost, &db.NodePort, &db.PublicPort,
		&db.Status, &idKey, &db.PrivateIP, &db.VPCID, &db.CreatedAt, &db.UpdatedAt, &db.DeletedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get database by idempotency key: %w", err)
	}
	db.IdempotencyKey = idKey
	return db, nil
}

func (r *PostgresRepository) ListDatabases(ctx context.Context, accountID string) ([]*domain.Database, error) {
	query := `
		SELECT id, account_id, arn, name, physical_db_name, node_host, node_port, public_port, status, idempotency_key, private_ip, vpc_id, created_at, updated_at, deleted_at
		FROM databases 
		WHERE account_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}
	defer rows.Close()

	var databases []*domain.Database
	for rows.Next() {
		db := &domain.Database{}
		var idKey *string
		if err := rows.Scan(
			&db.ID, &db.AccountID, &db.ARN, &db.Name, &db.PhysicalDBName, &db.NodeHost, &db.NodePort, &db.PublicPort,
			&db.Status, &idKey, &db.PrivateIP, &db.VPCID, &db.CreatedAt, &db.UpdatedAt, &db.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan database: %w", err)
		}
		db.IdempotencyKey = idKey
		databases = append(databases, db)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return databases, nil
}

func (r *PostgresRepository) UpdateDatabaseStatus(ctx context.Context, id string, status domain.DBStatus) error {
	query := `UPDATE databases SET status = $1, updated_at = NOW() WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update database status: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) UpdateDatabasePublicPort(ctx context.Context, id string, publicPort int) error {
	query := `UPDATE databases SET public_port = $1, updated_at = NOW() WHERE id = $2`
	res, err := r.db.ExecContext(ctx, query, publicPort, id)
	if err != nil {
		return fmt.Errorf("failed to update database public port: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) UpdateDatabaseNetwork(ctx context.Context, id, vpcID, privateIP, nodeHost string) error {
	query := `UPDATE databases SET vpc_id = $1, private_ip = $2, node_host = $3, updated_at = NOW() WHERE id = $4`
	res, err := r.db.ExecContext(ctx, query, vpcID, privateIP, nodeHost, id)
	if err != nil {
		return fmt.Errorf("failed to update database network: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) DeleteDatabase(ctx context.Context, id string) error {
	query := `UPDATE databases SET status = 'DELETED', deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete database: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) GetActiveCredential(ctx context.Context, databaseID string) (*domain.Credential, error) {
	query := `
		SELECT id, database_id, role_name, encrypted_password, is_master, status, created_at, updated_at
		FROM credentials
		WHERE database_id = $1 AND status = 'ACTIVE' AND is_master = true
		LIMIT 1
	`
	cred := &domain.Credential{}
	err := r.db.QueryRowContext(ctx, query, databaseID).Scan(
		&cred.ID, &cred.DatabaseID, &cred.RoleName, &cred.EncryptedPassword,
		&cred.IsMaster, &cred.Status, &cred.CreatedAt, &cred.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active credential: %w", err)
	}
	return cred, nil
}
