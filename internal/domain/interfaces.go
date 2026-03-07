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

	// ClaudeDB Control Plane operations
	CreateDatabaseTx(ctx context.Context, db *Database, cred *Credential, op *Operation) error
	GetDatabase(ctx context.Context, id string) (*Database, error)
	GetDatabaseByIdempotencyKey(ctx context.Context, accountID, idempotencyKey string) (*Database, error)
	ListDatabases(ctx context.Context, accountID string) ([]*Database, error)
	UpdateDatabaseStatus(ctx context.Context, id string, status DBStatus) error
	UpdateDatabasePublicPort(ctx context.Context, id string, publicPort int) error
	DeleteDatabase(ctx context.Context, id string) error
	GetActiveCredential(ctx context.Context, databaseID string) (*Credential, error)
	// Volume operations
	CreateVolume(ctx context.Context, vol *Volume) error
	GetVolume(ctx context.Context, id string) (*Volume, error)
	ListVolumes(ctx context.Context, accountID string) ([]*Volume, error)
	UpdateVolumeStatus(ctx context.Context, id string, status VolumeStatus) error
	DeleteVolume(ctx context.Context, id string) error

	// Snapshot operations
	CreateSnapshot(ctx context.Context, snap *Snapshot) error
	GetSnapshot(ctx context.Context, id string) (*Snapshot, error)
	ListSnapshots(ctx context.Context, accountID string, databaseID *string) ([]*Snapshot, error)
	UpdateSnapshotStatus(ctx context.Context, id string, status SnapshotStatus) error
	DeleteSnapshot(ctx context.Context, id string) error
}

// ContainerConfig represents the configuration for creating a Docker container
type ContainerConfig struct {
	Name         string
	Image        string
	Port         int
	User         string
	Password     string
	OwnerID      string
	Environment  map[string]string
	Labels       map[string]string
	VolumeSource string
	VolumeDest   string
	PrivateIP    string // e.g. "10.1.2.5" — allocated by network service
	BridgeName   string // e.g. "br-vpc-19b72a7e" — tenant's VPC bridge
	Gateway      string // e.g. "10.1.2.1"
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

	// Docker Volume operations
	CreateVolume(ctx context.Context, name string) error
	RemoveVolume(ctx context.Context, name string) error
	InspectVolume(ctx context.Context, name string) (map[string]interface{}, error)
}
