package domain

import "time"

// Configuration represents a configuration parameter for a database instance
type Configuration struct {
	ID         string    `json:"id"`
	InstanceID string    `json:"instanceId"`
	Parameter  string    `json:"parameter"` // Configuration parameter name
	Value      string    `json:"value"`     // Configuration parameter value
	AppliedAt  time.Time `json:"appliedAt"`
	AppliedBy  string    `json:"appliedBy"` // User who applied the configuration
}

// ConfigHistory represents a historical record of configuration changes
type ConfigHistory struct {
	ID         string    `json:"id"`
	InstanceID string    `json:"instanceId"`
	Parameter  string    `json:"parameter"`
	OldValue   string    `json:"oldValue"`
	NewValue   string    `json:"newValue"`
	ChangedAt  time.Time `json:"changedAt"`
	ChangedBy  string    `json:"changedBy"` // User who made the change
}

// NewConfiguration creates a new configuration entry
func NewConfiguration(instanceID, parameter, value, appliedBy string) *Configuration {
	return &Configuration{
		InstanceID: instanceID,
		Parameter:  parameter,
		Value:      value,
		AppliedAt:  time.Now(),
		AppliedBy:  appliedBy,
	}
}

// NewConfigHistory creates a new configuration history entry
func NewConfigHistory(instanceID, parameter, oldValue, newValue, changedBy string) *ConfigHistory {
	return &ConfigHistory{
		InstanceID: instanceID,
		Parameter:  parameter,
		OldValue:   oldValue,
		NewValue:   newValue,
		ChangedAt:  time.Now(),
		ChangedBy:  changedBy,
	}
}




