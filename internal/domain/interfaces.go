package domain

import "context"

// RepositoryPort defines the interface for database operations
type RepositoryPort interface {
	// Instance operations
	CreateInstance(ctx context.Context, instance *DBInstance) error
	GetInstance(ctx context.Context, id string) (*DBInstance, error)
	ListInstances(ctx context.Context, ownerID string) ([]*DBInstance, error)
	UpdateInstance(ctx context.Context, instance *DBInstance) error
	DeleteInstance(ctx context.Context, id string) error
	GetNextAvailablePort(ctx context.Context) (int, error)

	// Audit operations
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	ListAuditLogs(ctx context.Context, instanceID string, limit int) ([]*AuditLog, error)

	// Configuration operations
	SaveConfiguration(ctx context.Context, config *Configuration) error
	GetConfiguration(ctx context.Context, instanceID string) ([]*Configuration, error)
	SaveConfigHistory(ctx context.Context, history *ConfigHistory) error
	GetConfigHistory(ctx context.Context, instanceID string) ([]*ConfigHistory, error)
}

// ContainerConfig represents the configuration for creating a Docker container
type ContainerConfig struct {
	Name        string
	Image       string
	Port        int
	User        string
	Password    string
	OwnerID     string
	Environment map[string]string
	Labels      map[string]string
}

// ContainerInfo represents information about a running container
type ContainerInfo struct {
	ID     string
	Status string
	Port   int
}

// DockerPort defines the interface for Docker operations
type DockerPort interface {
	PullImage(ctx context.Context, image string) error
	CreateContainer(ctx context.Context, config ContainerConfig) (string, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
	RemoveContainer(ctx context.Context, containerID string) error
	GetContainerStatus(ctx context.Context, containerID string) (string, error)
	GetContainerInfo(ctx context.Context, containerID string) (*ContainerInfo, error)
}
