package domain

import "time"

type ScalingPolicy struct {
	ID                string    `json:"id"`
	TenantID          string    `json:"tenant_id"`
	Name              string    `json:"name"`
	PolicyType        string    `json:"policy_type"`
	MinCapacity       int       `json:"min_capacity"`
	MaxCapacity       int       `json:"max_capacity"`
	TargetValue       float64   `json:"target_value"`
	ScaleInCooldown   int       `json:"scale_in_cooldown"`
	ScaleOutCooldown  int       `json:"scale_out_cooldown"`
	TargetType        string    `json:"target_type"`
	TargetID          string    `json:"target_id"`
	MetricName        string    `json:"metric_name"`
	ScaleDownValue    float64   `json:"scale_down_value"`
	MaxInstances      int       `json:"max_instances"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ScalingPolicyRequest struct {
	Name              string  `json:"name"`
	PolicyType        string  `json:"policy_type"`
	MinCapacity       int     `json:"min_capacity"`
	MaxCapacity       int     `json:"max_capacity"`
	TargetValue       float64 `json:"target_value"`
	ScaleInCooldown   int     `json:"scale_in_cooldown"`
	ScaleOutCooldown  int     `json:"scale_out_cooldown"`
	TargetType        string  `json:"target_type"` // e.g., "instance", "asg"
	TargetID          string  `json:"target_id" binding:"required"`
	MetricName        string  `json:"metric_name" binding:"required"`
	ScaleDownValue    float64 `json:"scale_down_value"`
	MaxInstances      int     `json:"max_instances"`
}

type UpdateScalingPolicyRequest struct {
	Name              *string  `json:"name,omitempty"`
	MinCapacity       *int     `json:"min_capacity,omitempty"`
	MaxCapacity       *int     `json:"max_capacity,omitempty"`
	TargetValue       *float64 `json:"target_value,omitempty"`
	ScaleInCooldown   *int     `json:"scale_in_cooldown,omitempty"`
	ScaleOutCooldown  *int     `json:"scale_out_cooldown,omitempty"`
	ScaleDownValue    *float64 `json:"scale_down_value,omitempty"`
	MaxInstances      *int     `json:"max_instances,omitempty"`
}