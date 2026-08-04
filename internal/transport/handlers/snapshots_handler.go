package handler


import (
	"fmt"
	"log"
	"net/http"

	"rds/internal/application"
	"rds/internal/domain"
	"rds/internal/utils"

	"github.com/gin-gonic/gin"
)

// --- 3. Backup & Restore ---

func (h *ClaudeDBHandler) CreateSnapshot(c *gin.Context) {
	userID := c.GetString("userId")
	requestID := c.GetString("requestID")

	var payload domain.CreateSnapshotPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Printf("[Handler:CreateSnapshot] Payload unmarshal, bad request for requestID %s, with error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("Bad request"))
		return
	}

	snap, err := h.snapshotService.CreateSnapshot(c.Request.Context(), payload, userID)
	if err != nil {
		log.Printf("[Handler:CreateSnapshot] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to create snapshot"))

		return
	}

	utils.RespondSucces(c, http.StatusCreated, "Snapshot created successfully", snap)
}

func (h *ClaudeDBHandler) ListSnapshots(c *gin.Context) {
	accountID := c.GetString("userId")
	requestID := c.GetString("requestID")

	dbID := c.Query("databaseId")

	var dbIDPtr *string
	if dbID != "" {
		dbIDPtr = &dbID
	}

	snaps, err := h.snapshotService.ListSnapshots(c.Request.Context(), accountID, dbIDPtr)
	if err != nil {
		log.Printf("[Handler:ListSnapshots] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to list snapshots"))

		return
	}

	utils.RespondSucces(c, http.StatusOK, "Snapshots fetched successfully", snaps)
}

func (h *ClaudeDBHandler) DeleteSnapshot(c *gin.Context) {
	id := c.Param("snapshot_id")
	accountID := c.GetString("userId")
	requestID := c.GetString("requestID")

	if err := h.snapshotService.DeleteSnapshot(c.Request.Context(), id, accountID); err != nil {
		log.Printf("[Handler:DeleteSnapshot] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to delete snapshots"))

		return
	}

	utils.RespondSucces(c, http.StatusOK, "Snapshot deleted successfully", gin.H{"id": id, "status": "DELETED"})
}

func (h *ClaudeDBHandler) RestoreDatabase(c *gin.Context) {
	accountID := c.GetString("userId")
	requestID := c.GetString("requestID")

	var payload domain.RestoreDatabasePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Printf("[Handler:RestoreDatabase] Payload unmarshal, bad request for requestID %s, with error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("Bad request"))
		return
	}

	result, err := h.snapshotService.RestoreDatabase(c.Request.Context(), application.RestoreDatabaseRequest{
		SnapshotID: payload.SnapshotID,
		NewName:    payload.NewName,
		AccountID:  accountID,
	})
	if err != nil {
		log.Printf("[Handler:RestoreDatabase] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to Restore Database"))

		return
	}

	utils.RespondSucces(c, http.StatusCreated, "Database restored successfully", result)
}
