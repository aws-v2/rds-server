package repository

import (
    "context"
    "fmt"
    "time"

    "github.com/google/uuid"
		
	"rds/internal/domain")

func (r *PostgresRepository) CreateScalingPolicy(ctx context.Context, tenantID string, req domain.ScalingPolicyRequest) error {
    query := `
        INSERT INTO scaling_policies (
            id, tenant_id, name, policy_type,
            min_capacity, max_capacity, target_value,
            scale_in_cooldown, scale_out_cooldown,
            created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    `
    now := time.Now()
    id := uuid.New().String()

    _, err := r.db.ExecContext(ctx, query,
        id, tenantID, req.Name, req.PolicyType,
        req.MinCapacity, req.MaxCapacity, req.TargetValue,
        req.ScaleInCooldown, req.ScaleOutCooldown,
        now, now,
    )
    if err != nil {
        return fmt.Errorf("CreateScalingPolicy: %w", err)
    }
    return nil
}

func (r *PostgresRepository) GetScalingPolicies(ctx context.Context, tenantID string) ([]domain.ScalingPolicy, error) {
    query := `
        SELECT
            id, tenant_id, name, policy_type,
            min_capacity, max_capacity, target_value,
            scale_in_cooldown, scale_out_cooldown,
            created_at, updated_at
        FROM scaling_policies
        WHERE tenant_id = $1
        ORDER BY created_at DESC
    `
    rows, err := r.db.QueryContext(ctx, query, tenantID)
    if err != nil {
        return nil, fmt.Errorf("GetScalingPolicies: %w", err)
    }
    defer rows.Close()

    var policies []domain.ScalingPolicy
    for rows.Next() {
        var p domain.ScalingPolicy
        if err := rows.Scan(
            &p.ID, &p.TenantID, &p.Name, &p.PolicyType,
            &p.MinCapacity, &p.MaxCapacity, &p.TargetValue,
            &p.ScaleInCooldown, &p.ScaleOutCooldown,
            &p.CreatedAt, &p.UpdatedAt,
        ); err != nil {
            return nil, fmt.Errorf("GetScalingPolicies scan: %w", err)
        }
        policies = append(policies, p)
    }
    return policies, rows.Err()
}

func (r *PostgresRepository) UpdateScalingPolicy(ctx context.Context, tenantID, policyID string, req domain.UpdateScalingPolicyRequest) error {
    query := `
        UPDATE scaling_policies
        SET
            name              = COALESCE($1, name),
            min_capacity      = COALESCE($2, min_capacity),
            max_capacity      = COALESCE($3, max_capacity),
            target_value      = COALESCE($4, target_value),
            scale_in_cooldown = COALESCE($5, scale_in_cooldown),
            scale_out_cooldown = COALESCE($6, scale_out_cooldown),
            updated_at        = $7
        WHERE id = $8 AND tenant_id = $9
    `
    result, err := r.db.ExecContext(ctx, query,
        req.Name, req.MinCapacity, req.MaxCapacity,
        req.TargetValue, req.ScaleInCooldown, req.ScaleOutCooldown,
        time.Now(),
        policyID, tenantID,
    )
    if err != nil {
        return fmt.Errorf("UpdateScalingPolicy: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("UpdateScalingPolicy rows affected: %w", err)
    }
    if rows == 0 {
        return fmt.Errorf("scaling policy %s not found for tenant %s", policyID, tenantID)
    }
    return nil
}

func (r *PostgresRepository) DeleteScalingPolicy(ctx context.Context, tenantID, policyID string) error {
    query := `
        DELETE FROM scaling_policies
        WHERE id = $1 AND tenant_id = $2
    `
    result, err := r.db.ExecContext(ctx, query, policyID, tenantID)
    if err != nil {
        return fmt.Errorf("DeleteScalingPolicy: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("DeleteScalingPolicy rows affected: %w", err)
    }
    if rows == 0 {
        return fmt.Errorf("scaling policy %s not found for tenant %s", policyID, tenantID)
    }
    return nil
}