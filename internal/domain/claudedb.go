package domain

import "time"

// DBStatus represents the current state of a database
type DBStatus string

const (
	DBStatusPending       DBStatus = "PENDING"
	DBStatusProvisioning  DBStatus = "PROVISIONING"
	DBStatusAvailable     DBStatus = "AVAILABLE"
	DBStatusMaintenance   DBStatus = "MAINTENANCE"
	DBStatusDeleted       DBStatus = "DELETED"
	DBStatusFailed        DBStatus = "FAILED"
	DBStatusDeleting      DBStatus = "DELETING"
	DBStatusStopped       DBStatus = "STOPPED"
	DBStatusPendingReboot DBStatus = "PENDING_REBOOT"
)

// Database represents a logical PostgreSQL database in ClaudeDB
type Database struct {
	ID             string
	AccountID      string
	ARN            string
	Name           string
	PhysicalDBName string
	NodeHost       string
	NodePort       int
	Status         DBStatus
	PrivateIP      string // the container's private IP on the VPC
	VPCID          string // which VPC this database belongs to
	IdempotencyKey *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      *time.Time
}

type CredentialStatus string

const (
	CredStatusActive   CredentialStatus = "ACTIVE"
	CredStatusRotating CredentialStatus = "ROTATING"
	CredStatusRevoked  CredentialStatus = "REVOKED"
)

type Credential struct {
	ID                string
	DatabaseID        string
	RoleName          string
	EncryptedPassword string
	IsMaster          bool
	Status            CredentialStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type OperationType string

const (
	OpTypeProvisionDB   OperationType = "PROVISION_DB"
	OpTypeDeleteDB      OperationType = "DELETE_DB"
	OpTypeCreateBackup  OperationType = "CREATE_BACKUP"
	OpTypeRestoreBackup OperationType = "RESTORE_BACKUP"
	OpTypeRotateCreds   OperationType = "ROTATE_CREDS"
	OpTypeStopDB        OperationType = "STOP_DB"
	OpTypeStartDB       OperationType = "START_DB"
	OpTypeRebootDB      OperationType = "REBOOT_DB"
)

type OperationStatus string

const (
	OpStatusQueued  OperationStatus = "QUEUED"
	OpStatusRunning OperationStatus = "RUNNING"
	OpStatusSuccess OperationStatus = "SUCCESS"
	OpStatusFailed  OperationStatus = "FAILED"
)

type Operation struct {
	ID           string
	DatabaseID   string
	Type         OperationType
	Status       OperationStatus
	ErrorMessage *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// VolumeStatus represents the state of a storage volume
type VolumeStatus string

const (
	VolumeStatusCreating  VolumeStatus = "CREATING"
	VolumeStatusAvailable VolumeStatus = "AVAILABLE"
	VolumeStatusInUse     VolumeStatus = "IN_USE"
	VolumeStatusDeleting  VolumeStatus = "DELETING"
	VolumeStatusDeleted   VolumeStatus = "DELETED"
	VolumeStatusError     VolumeStatus = "ERROR"
)

// Volume represents a storage volume attached to an instance
type Volume struct {
	ID        string
	AccountID string
	ARN       string
	Name      string
	SizeGB    int
	Status    VolumeStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}

// SnapshotStatus represents the state of a snapshot
type SnapshotStatus string

const (
	SnapshotStatusCreating  SnapshotStatus = "CREATING"
	SnapshotStatusAvailable SnapshotStatus = "AVAILABLE"
	SnapshotStatusDeleting  SnapshotStatus = "DELETING"
	SnapshotStatusDeleted   SnapshotStatus = "DELETED"
	SnapshotStatusError     SnapshotStatus = "ERROR"
)

// Snapshot represents a point-in-time backup of a Volume
type Snapshot struct {
	ID         string
	AccountID  string
	ARN        string
	Name       string
	DatabaseID string
	VolumeID   string
	SizeGB     int
	Status     SnapshotStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}
