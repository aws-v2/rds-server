package domain

import "time"

// InstanceStatus represents the current state of a database instance
type InstanceStatus string

const (
	StatusRunning InstanceStatus = "running"
	StatusStopped InstanceStatus = "stopped"
	StatusFailed  InstanceStatus = "failed"
)

// DBInstance represents a managed PostgreSQL database instance
type DBInstance struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Engine      string         `json:"engine"` // e.g., "postgres:17"
	Port        int            `json:"port"`
	Host        string         `json:"host"` // Container host (default: "localhost")
	User        string         `json:"user"`
	Password    string         `json:"password"` // Stored encrypted
	OwnerID     string         `json:"ownerId"`
	ContainerID string         `json:"containerId"`
	Status      InstanceStatus `json:"status"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

// IsRunning returns true if the instance is in running state
func (i *DBInstance) IsRunning() bool {
	return i.Status == StatusRunning
}

// IsStopped returns true if the instance is in stopped state
func (i *DBInstance) IsStopped() bool {
	return i.Status == StatusStopped
}

// IsFailed returns true if the instance is in failed state
func (i *DBInstance) IsFailed() bool {
	return i.Status == StatusFailed
}

// MarkAsRunning sets the instance status to running
func (i *DBInstance) MarkAsRunning() {
	i.Status = StatusRunning
	i.UpdatedAt = time.Now()
}

// MarkAsStopped sets the instance status to stopped
func (i *DBInstance) MarkAsStopped() {
	i.Status = StatusStopped
	i.UpdatedAt = time.Now()
}

// MarkAsFailed sets the instance status to failed
func (i *DBInstance) MarkAsFailed() {
	i.Status = StatusFailed
	i.UpdatedAt = time.Now()
}
