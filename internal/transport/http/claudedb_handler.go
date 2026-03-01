package http

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"rds/internal/application"
	"rds/internal/domain"
	"rds/internal/middleware" // Added new import

	"github.com/gin-gonic/gin"
)

// ClaudeDBHandler handles database instance requests for the new control plane
type ClaudeDBHandler struct {
	dbService       *application.ClaudeDBService
	volumeService   *application.VolumeService
	snapshotService *application.SnapshotService
	validator       *middleware.IAMValidator
}

// NewClaudeDBHandler creates a new ClaudeDB handler
func NewClaudeDBHandler(dbService *application.ClaudeDBService, volumeService *application.VolumeService, snapshotService *application.SnapshotService, validator *middleware.IAMValidator) *ClaudeDBHandler {
	return &ClaudeDBHandler{
		dbService:       dbService,
		volumeService:   volumeService,
		snapshotService: snapshotService,
		validator:       validator,
	}
}

func extractAccountID(c *gin.Context) (string, error) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header missing")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	token := parts[1]
	tokenParts := strings.Split(token, ".")
	if len(tokenParts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}

	payloadData, err := base64.RawURLEncoding.DecodeString(tokenParts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode token payload: %w", err)
	}

	var payload struct {
		UserID string `json:"userId"`
	}
	if err := json.Unmarshal(payloadData, &payload); err != nil {
		return "", fmt.Errorf("failed to parse token payload: %w", err)
	}

	if payload.UserID == "" {
		return "", fmt.Errorf("userId missing from token")
	}

	return payload.UserID, nil
}

// 3. Backup & Restore Lifecycles APIs

func (h *ClaudeDBHandler) StartDatabase(c *gin.Context) {
	id := c.Param("id")
	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if err := h.dbService.StartDatabase(c.Request.Context(), id, accountID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Database starting successfully"})
}

func (h *ClaudeDBHandler) StopDatabase(c *gin.Context) {
	id := c.Param("id")
	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if err := h.dbService.StopDatabase(c.Request.Context(), id, accountID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Database stopping successfully"})
}

func (h *ClaudeDBHandler) RebootDatabase(c *gin.Context) {
	id := c.Param("id")
	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if err := h.dbService.RebootDatabase(c.Request.Context(), id, accountID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Database rebooting successfully"})
}

type CreateSnapshotPayload struct {
	Name       string `json:"name" binding:"required"`
	DatabaseID string `json:"databaseId" binding:"required"`
}

func (h *ClaudeDBHandler) CreateSnapshot(c *gin.Context) {
	var payload CreateSnapshotPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	snap, err := h.snapshotService.CreateSnapshot(c.Request.Context(), application.CreateSnapshotRequest{
		Name:       payload.Name,
		DatabaseID: payload.DatabaseID,
		AccountID:  accountID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, snap)
}

func (h *ClaudeDBHandler) ListSnapshots(c *gin.Context) {
	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	dbID := c.Query("databaseId")

	var dbIDPtr *string
	if dbID != "" {
		dbIDPtr = &dbID
	}

	snaps, err := h.snapshotService.ListSnapshots(c.Request.Context(), accountID, dbIDPtr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, snaps)
}

// 4. Telemetry & Observability APIs

func (h *ClaudeDBHandler) GetMetrics(c *gin.Context) {
	id := c.Param("id")
	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	db, err := h.dbService.GetDatabase(c.Request.Context(), id, accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Database not found"})
		return
	}

	// In a real implementation this would fetch from Prometheus or similar using the container ID / name
	mockMetrics := gin.H{
		"database_id":        db.ID,
		"cpu_usage_percent":  2.5,
		"memory_usage_mb":    145.2,
		"active_connections": 12,
		"disk_iops":          45,
	}

	c.JSON(http.StatusOK, mockMetrics)
}

func (h *ClaudeDBHandler) GetLogs(c *gin.Context) {
	id := c.Param("id")
	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	db, err := h.dbService.GetDatabase(c.Request.Context(), id, accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Database not found"})
		return
	}

	// In a real implementation this would multiplex docker logs stream
	mockLogs := []string{
		fmt.Sprintf("2023-10-27 10:00:00 UTC [1] LOG:  starting PostgreSQL 15.4 for ClaudeDB %s", db.Name),
		"2023-10-27 10:00:00 UTC [1] LOG:  listening on IPv4 address \"0.0.0.0\", port 5432",
		"2023-10-27 10:00:00 UTC [1] LOG:  listening on IPv6 address \"::\", port 5432",
		"2023-10-27 10:00:00 UTC [1] LOG:  listening on Unix socket \"/var/run/postgresql/.s.PGSQL.5432\"",
		"2023-10-27 10:00:00 UTC [26] LOG:  database system was shut down at 2023-10-27 09:59:59 UTC",
		"2023-10-27 10:00:00 UTC [1] LOG:  database system is ready to accept connections",
	}

	c.JSON(http.StatusOK, gin.H{
		"database_id": db.ID,
		"logs":        mockLogs,
	})
}

// 5. Configuration parameters APIs

func (h *ClaudeDBHandler) GetParameters(c *gin.Context) {
	id := c.Param("id")
	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	db, err := h.dbService.GetDatabase(c.Request.Context(), id, accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Database not found"})
		return
	}

	// Mock default postgresql.conf parameters
	params := map[string]interface{}{
		"max_connections":      100,
		"shared_buffers":       "128MB",
		"work_mem":             "4MB",
		"maintenance_work_mem": "64MB",
		"wal_level":            "replica",
	}

	c.JSON(http.StatusOK, gin.H{
		"database_id": db.ID,
		"parameters":  params,
	})
}

type ModifyParametersPayload struct {
	Parameters map[string]interface{} `json:"parameters" binding:"required"`
}

func (h *ClaudeDBHandler) ModifyParameters(c *gin.Context) {
	id := c.Param("id")
	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	db, err := h.dbService.GetDatabase(c.Request.Context(), id, accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Database not found"})
		return
	}

	var payload ModifyParametersPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Here a real implementation creates/modifies a postgresql.auto.conf inside the volume or container
	// and checks if a reboot is pending based on `pg_settings.context != 'user'`
	pendingReboot := false
	for k := range payload.Parameters {
		if k == "shared_buffers" || k == "max_connections" || k == "wal_level" {
			pendingReboot = true
		}
	}

	if pendingReboot {
		_ = h.dbService.RebootDatabase(c.Request.Context(), id, accountID) // Mocking setting state to pending for now
	}

	c.JSON(http.StatusOK, gin.H{
		"database_id":        db.ID,
		"message":            "Parameters applied successfully",
		"pending_reboot":     pendingReboot,
		"updated_parameters": payload.Parameters,
	})
}

type ModifyDatabasePayload struct {
	InstanceClass    string `json:"instanceClass" binding:"required"`
	AllocatedStorage int    `json:"allocatedStorage,omitempty"`
}

func (h *ClaudeDBHandler) ModifyDatabase(c *gin.Context) {
	id := c.Param("id")
	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	db, err := h.dbService.GetDatabase(c.Request.Context(), id, accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Database not found"})
		return
	}

	var payload ModifyDatabasePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mocking actual instance modification (which would involve stopping, changing resources, and starting)
	// For now just reboot to simulate downtime
	if err := h.dbService.RebootDatabase(c.Request.Context(), id, accountID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to modify database: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"database_id": db.ID,
		"message":     "Modification initiated successfully",
		"status":      "MODIFYING",
	})
}

type RestoreDatabasePayload struct {
	SnapshotID string `json:"snapshotId" binding:"required"`
	NewName    string `json:"newName,omitempty"`
}

func (h *ClaudeDBHandler) RestoreDatabase(c *gin.Context) {
	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var payload RestoreDatabasePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.snapshotService.RestoreDatabase(c.Request.Context(), application.RestoreDatabaseRequest{
		SnapshotID: payload.SnapshotID,
		NewName:    payload.NewName,
		AccountID:  accountID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to restore database: %v", err)})
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *ClaudeDBHandler) DeleteSnapshot(c *gin.Context) {
	id := c.Param("snapshot_id")
	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if err := h.snapshotService.DeleteSnapshot(c.Request.Context(), id, accountID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "snapshot marked for deletion"})
}

// --- 6. Volume Management APIs ---

type CreateVolumePayload struct {
	Name   string `json:"name" binding:"required"`
	SizeGB int    `json:"sizeGb" binding:"required,min=1"`
}

func (h *ClaudeDBHandler) CreateVolume(c *gin.Context) {
	var payload CreateVolumePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	vol, err := h.volumeService.CreateVolume(c.Request.Context(), application.CreateVolumeRequest{
		Name:      payload.Name,
		SizeGB:    payload.SizeGB,
		AccountID: accountID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, vol)
}

func (h *ClaudeDBHandler) ListVolumes(c *gin.Context) {
	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	vols, err := h.volumeService.ListVolumes(c.Request.Context(), accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, vols)
}

func (h *ClaudeDBHandler) GetVolume(c *gin.Context) {
	id := c.Param("id")
	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	vol, err := h.volumeService.GetVolume(c.Request.Context(), id, accountID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "volume not found"})
		return
	}

	c.JSON(http.StatusOK, vol)
}

func (h *ClaudeDBHandler) DeleteVolume(c *gin.Context) {
	id := c.Param("id")
	accountID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if err := h.volumeService.DeleteVolume(c.Request.Context(), id, accountID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "volume marked for deletion"})
}

type CreateDatabasePayload struct {
	Name     string `json:"name" binding:"required"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

func (h *ClaudeDBHandler) CreateDatabase(c *gin.Context) {
	var input CreateDatabasePayload
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")

	req := application.CreateDatabaseRequest{
		Name:           input.Name,
		User:           input.User,
		Password:       input.Password,
		OwnerID:        userID,
		IdempotencyKey: idempotencyKey,
	}

	resp, err := h.dbService.CreateDatabase(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "conflict: database is currently provisioning" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

func (h *ClaudeDBHandler) ListDatabases(c *gin.Context) {
	userID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	dbs, err := h.dbService.ListDatabases(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type Output struct {
		ID        string          `json:"id"`
		ARN       string          `json:"arn"`
		Name      string          `json:"name"`
		Port      int             `json:"port"`
		Status    domain.DBStatus `json:"status"`
		CreatedAt string          `json:"createdAt"`
	}

	outputs := make([]Output, 0, len(dbs))
	for _, db := range dbs {
		outputs = append(outputs, Output{
			ID:        db.ID,
			ARN:       db.ARN,
			Name:      db.Name,
			Port:      db.NodePort,
			Status:    db.Status,
			CreatedAt: db.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, gin.H{"databases": outputs})
}

func (h *ClaudeDBHandler) GetDatabase(c *gin.Context) {
	id := c.Param("id")
	userID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	db, err := h.dbService.GetDatabase(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        db.ID,
		"arn":       db.ARN,
		"name":      db.Name,
		"status":    db.Status,
		"port":      db.NodePort,
		"createdAt": db.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *ClaudeDBHandler) DeleteDatabase(c *gin.Context) {
	id := c.Param("id")
	userID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if err := h.dbService.DeleteDatabase(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Database deleted successfully"})
}

func (h *ClaudeDBHandler) RotateCredentials(c *gin.Context) {
	id := c.Param("id")
	userID, err := extractAccountID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.dbService.RotateCredentials(c.Request.Context(), id, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}
