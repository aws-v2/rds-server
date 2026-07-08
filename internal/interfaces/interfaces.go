package interfaces

import (
	"context"
	"rds/internal/domain"
)

// RepositoryPort defines the interface for database operations
type RepositoryPort interface {
	// Instance operations
	CreateInstance(ctx context.Context, instance *domain.DBInstance) error
	GetInstance(ctx context.Context, id string) (*domain.DBInstance, error)
	ListInstances(ctx context.Context, ownerID string) ([]*domain.DBInstance, error)
	UpdateInstance(ctx context.Context, instance *domain.DBInstance) error
	DeleteInstance(ctx context.Context, id string) error
	GetNextAvailablePort(ctx context.Context) (int, error)

	// Audit operations
	CreateAuditLog(ctx context.Context, log *domain.AuditLog) error
	ListAuditLogs(ctx context.Context, instanceID string, limit int) ([]*domain.AuditLog, error)

	// Configuration operations
	SaveConfiguration(ctx context.Context, config *domain.Configuration) error
	GetConfiguration(ctx context.Context, instanceID string) ([]*domain.Configuration, error)
	SaveConfigHistory(ctx context.Context, history *domain.ConfigHistory) error
	GetConfigHistory(ctx context.Context, instanceID string) ([]*domain.ConfigHistory, error)

	// ClaudeDB Control Plane operations
	CreateDatabaseTx(ctx context.Context, db *domain.Database, cred *domain.Credential, op *domain.Operation) error
	GetDatabase(ctx context.Context, id string) (*domain.Database, error)
	GetDatabaseByIdempotencyKey(ctx context.Context, accountID, idempotencyKey string) (*domain.Database, error)
	ListDatabases(ctx context.Context, accountID string) ([]*domain.Database, error)
	GetNextNodePort(ctx context.Context) (int, error)
	ListAllActiveDatabases(ctx context.Context) ([]*domain.Database, error)
	UpdateDatabaseStatus(ctx context.Context, id string, status domain.DBStatus) error
	UpdateDatabasePublicPort(ctx context.Context, id string, publicPort int) error
	UpdateDatabaseNetwork(ctx context.Context, id, vpcID, privateIP, nodeHost string) error
	DeleteDatabase(ctx context.Context, id string) error
	HardDeleteDatabase(ctx context.Context, id string) error
	GetActiveCredential(ctx context.Context, databaseID string) (*domain.Credential, error)
	CreateScalingPolicy(ctx context.Context, tenantID string, policy domain.ScalingPolicyRequest) error
	GetScalingPolicies(ctx context.Context, tenantID string) ([]domain.ScalingPolicy, error)
	UpdateScalingPolicy(ctx context.Context, tenantID, policyID string, req domain.UpdateScalingPolicyRequest) error
	DeleteScalingPolicy(ctx context.Context, tenantID, policyID string) error

	// VPC operations
	CreateVPC(ctx context.Context, vpc *domain.VPC) error
	GetVPC(ctx context.Context, id string) (*domain.VPC, error)
	GetDefaultVPC(ctx context.Context, accountID string) (*domain.VPC, error)
	ListVPCs(ctx context.Context, accountID string) ([]*domain.VPC, error)
	DeleteVPC(ctx context.Context, id string) error

	// IP operations
	ListAllocatedIPs(ctx context.Context, vpcID string) ([]string, error)

	// Volume operations
	CreateVolume(ctx context.Context, vol *domain.Volume) error
	GetVolume(ctx context.Context, id string) (*domain.Volume, error)
	ListVolumes(ctx context.Context, accountID string) ([]*domain.Volume, error)
	UpdateVolumeStatus(ctx context.Context, id string, status domain.VolumeStatus) error
	DeleteVolume(ctx context.Context, id string) error

	// Snapshot operations
	CreateSnapshot(ctx context.Context, snap *domain.Snapshot) error
	GetSnapshot(ctx context.Context, id string) (*domain.Snapshot, error)
	ListSnapshots(ctx context.Context, accountID string, databaseID *string) ([]*domain.Snapshot, error)
	UpdateSnapshotStatus(ctx context.Context, id string, status domain.SnapshotStatus) error
	DeleteSnapshot(ctx context.Context, id string) error
}

// DockerPort defines the interface for Docker operations
type DockerPort interface {
	PullImage(ctx context.Context, image string) error
	CreateContainer(ctx context.Context, config domain.ContainerConfig) (string, error)
	StartContainer(ctx context.Context, containerID string) error
	StopContainer(ctx context.Context, containerID string) error
	RemoveContainer(ctx context.Context, containerID string) error
	GetContainerStatus(ctx context.Context, containerID string) (string, error)
	GetContainerInfo(ctx context.Context, containerID string) (*domain.ContainerInfo, error)
	GetContainerStats(ctx context.Context, containerID string) (*domain.ContainerStats, error)
	UpdateContainerResources(ctx context.Context, containerID string, cpuShares int64, memoryBytes int64) error
	EnsureNetwork(ctx context.Context, name, gateway string) error
	InspectNetwork(ctx context.Context, name string) (map[string]interface{}, error)
	RemoveNetwork(ctx context.Context, name string) error

	// Docker Volume operations
	CreateVolume(ctx context.Context, name string) error
	RemoveVolume(ctx context.Context, name string) error
	InspectVolume(ctx context.Context, name string) (map[string]interface{}, error)
}
