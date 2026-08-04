package domain

import (
	"encoding/json"
	"time"
)

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

type CreateDatabasePayload struct {
	InstanceName string `json:"name" binding:"required"`
	User         string `json:"user,omitempty"`
	Password     string `json:"password,omitempty"`
}

type RestoreDatabasePayload struct {
	SnapshotID string `json:"snapshotId" binding:"required"`
	NewName    string `json:"newName,omitempty"`
}

type AssignVPCPayload struct {
	VPCID string `json:"vpc_id" binding:"required"`
}
type CreateVPCPayload struct {
	Name string `json:"name" binding:"required"`
}
type CreateSnapshotPayload struct {
	Name       string `json:"name" binding:"required"`
	DatabaseID string `json:"databaseId" binding:"required"`
}

type ModifyDatabasePayload struct {
	InstanceClass    string `json:"instanceClass"`
	AllocatedStorage int    `json:"allocatedStorage,omitempty"`
	VpcID            string `json:"vpcId,omitempty"`
}
type ModifyParametersPayload struct {
	Parameters map[string]interface{} `json:"parameters" binding:"required"`
}
type DBSummary struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	DBPort     int      `json:"port"`
	PublicPort int      `json:"public_port"`
	VpcID      string   `json:"vpc_id"`
	PrivateIP  string   `json:"private_ip"`
	Status     DBStatus `json:"status"`
	GatewayIP  string   `json:"createdAt"`
}

type CreateVolumePayload struct {
	Name   string `json:"name" binding:"required"`
	SizeGB int    `json:"sizeGb" binding:"required,min=1"`
}

// Database represents a logical PostgreSQL database in ClaudeDB
type Database struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	UserID      string `json:"user_id"`
	DBName      string `json:"name"`         // name of the database
	VMIP        string `json:"vm_ip"`        // the ip of the vm
	GatewayIP   string `json:"gateway_ip"`   // the ip of the gateway
	GatewayPort int    `json:"gateway_port"` // the random port of the gateway
	VMDBPort    int    `json:"vm_db_port"`   // the port in the vm where the db runs, default is 5432
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updaetd_at"`
	DeletedAt   string `json:"deleted_at"`
	Engine string `json:"engine"`
	Region string `json:"region"`
}

type EC2Response struct {
	GatewayIP   string `json:"gateway_ip"`
	GatewayPort int    `json:"gateway_port"`
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

type ProvisionInstanceEvent struct {
	UserID     string          `json:"userID"`
	Profile    string          `json:"profile" binding:"required"`
	Name       string          `json:"name" binding:"required"`
	ResourceID string          `json:"resource_id"`
	Specs      VMSpecs         `json:"specs" binding:"required"`
	SessionID  string          `json:"session_id"`
	Assets     []Asset         `json:"assets,omitempty"`
	Config     json.RawMessage `json:"config,omitempty"` // profile-specific config, opaque to the core service
}

type VMSpecs struct {
	CPU     int `json:"cpu" binding:"required"`
	RAM     int `json:"ram" binding:"required"` // MB
	Storage int `json:"storage,omitempty"`      // GB, optional override of image default
}
type AssetSource string

const (
	AssetSourceObject AssetSource = "object" // single presigned file (e.g. lambda binary)
	AssetSourceZip    AssetSource = "zip"    // presigned zip (folder/bucket export) — unpack after download
	AssetSourceInline AssetSource = "inline" // small payload embedded directly, base64
)

type Asset struct {
	Name       string      `json:"name"`
	Source     AssetSource `json:"source"`
	URL        string      `json:"url,omitempty"`
	InlineData string      `json:"inline_data,omitempty"` // only for AssetSourceInline
	DestPath   string      `json:"dest_path"`             // where the agent places/unpacks it
	SHA256     string      `json:"sha256,omitempty"`
	Unpack     bool        `json:"unpack,omitempty"`     // true = unzip after download
	Executable bool        `json:"executable,omitempty"` // chmod +x after placing

	Path string `json:"path"`
}

// CreateDatabaseRequest represents the request to provision a database
type CreateDatabaseRequest struct {
	InstanceName   string
	User           string
	Password       string
	OwnerID        string
	VPCID          string
	IdempotencyKey string
	SessionID      string
}

// CreateDatabaseResponse represents the result of the provisioning flow
type CreateDatabaseResponse struct {
	DatabaseID             string
	ARN                    string
	Name                   string
	NodeHost               string
	NodePort               int
	RoleName               string
	Password               string
	PhysicalDBName         string
	ConnectionString       string
	PublicConnectionString string
}
