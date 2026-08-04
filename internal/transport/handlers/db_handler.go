package handler


import (
	"fmt"
	"log"
	"net/http"

	"rds/internal/application"
	"rds/internal/domain"
	"rds/internal/transport/middleware"
	"rds/internal/utils"

	"github.com/gin-gonic/gin"
)

// --- ClaudeDBHandler ---

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

func (h *ClaudeDBHandler) CreateDatabase(c *gin.Context) {
	requestID := c.GetString("requestID")
	userID := c.GetString("userId")


	var input domain.CreateDatabasePayload

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("[Handler:CreateDatabase] Payload unmarshal, bad request for requestID %s, with error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("Bad request"))
		return
	}

	if userID == "" {
		log.Printf("[Handler:CreateDatabase] UserID not present for requestID %s", requestID)
		utils.RespondError(c, http.StatusUnauthorized, fmt.Errorf("Unauthorized user"))

		return
	}

	idempotencyKey := requestID

	req := domain.CreateDatabaseRequest{
		SessionID :requestID,
		InstanceName:           input.InstanceName,
		User:           input.User,
		Password:       input.Password,
		OwnerID:        userID,
		IdempotencyKey: idempotencyKey,
	}

	resp, err := h.dbService.CreateDatabase(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "conflict: database is currently provisioning" {
			log.Printf("[Handler:CreateDatabase] Service call, conflict while database provisioning requestID %s, error %s", requestID, err.Error())
			utils.RespondError(c, http.StatusConflict, fmt.Errorf("conflict: database is currently provisioning"))
			return
		}
		log.Printf("[Handler:CreateDatabase] Service call, requestID %s  error %s", requestID, err.Error())

		utils.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	utils.RespondSucces(c, http.StatusCreated, "Database provisioned successfully", resp)
}

func (h *ClaudeDBHandler) AssignVPC(c *gin.Context) {
	requestID := c.GetString("requestID")
	userID := c.GetString("userId")
	databaseID := c.Param("id")

	var payload domain.AssignVPCPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Printf("[Handler:AssignVPC] Payload unmarshal, bad request for requestID %s, with error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("Bad request"))
		return
	}

	if err := h.dbService.AssignVPC(c.Request.Context(), userID, databaseID, payload.VPCID); err != nil {
		log.Printf("[Handler:AssignVPC] Service call,  for requestID %s, with error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("Error assigning vpc"))
		return
	}

	utils.RespondSucces(c, http.StatusOK, "Database VPC assignment updated successfully", gin.H{"id": databaseID, "status": "SUCCES"})
}

func (h *ClaudeDBHandler) CreateVPC(c *gin.Context) {
	requestID := c.GetString("requestID")
	userID := c.GetString("userId")
	var payload domain.CreateVPCPayload

	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Printf("[Handler:CreateVPC] Payload unmarshal, bad request for requestID %s, with error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("Bad request"))
		return
	}

	vpcID, err := h.dbService.CreateVPC(c.Request.Context(), userID, payload.Name)
	if err != nil {
		log.Printf("[Handler:CreateVPC] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("Error creating vpc"))
		return
	}

	utils.RespondSucces(c, http.StatusAccepted, "VPC creation request submitted successfully", gin.H{"id": vpcID, "status": "SUCCES"})
}

func (h *ClaudeDBHandler) ListVPCs(c *gin.Context) {
	requestID := c.GetString("requestID")
	userID := c.GetString("userId")
	vpcs, err := h.dbService.ListVPCs(c.Request.Context(), userID)
	if err != nil {
		log.Printf("[Handler:ListVPCs] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("Error listing vpc"))
		return
	}

	utils.RespondSucces(c, http.StatusOK, "VPCs fetched successfully", vpcs)
}

func (h *ClaudeDBHandler) ListDatabases(c *gin.Context) {
	requestID := c.GetString("requestID")
	userID := c.GetString("userId")
	dbs, err := h.dbService.ListDatabases(c.Request.Context(), userID)
	if err != nil {
		log.Printf("[Handler:ListDatabases] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("Error listing databases"))
		return
	}

	outputs := make([]domain.DBSummary, 0, len(dbs))
	for _, db := range dbs {
		port := db.GatewayPort
		 
		outputs = append(outputs, domain.DBSummary{
			ID:         db.ID,
			Name:       db.DBName,
			DBPort:       port,
			PublicPort: db.GatewayPort,
			GatewayIP:  db.GatewayIP,
			Status:    domain.DBStatus(db.Status),
		})
	}

	utils.RespondSucces(c, http.StatusOK, "Databases fetched successfully", outputs)
}

func (h *ClaudeDBHandler) GetDatabase(c *gin.Context) {
	id := c.Param("id")
	requestID := c.GetString("requestID")
	userID := c.GetString("userId")

	data, err := h.dbService.GetDatabaseWithConnectionString(c.Request.Context(), id, userID)
	if err != nil {
		log.Printf("[Handler:GetDatabase] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("Error listing 1 database"))
		return
	}

	utils.RespondSucces(c, http.StatusOK, "Database fetched successfully", data)
}

func (h *ClaudeDBHandler) DeleteDatabase(c *gin.Context) {
	id := c.Param("id")
	requestID := c.GetString("requestID")
	userID := c.GetString("userId")

	if err := h.dbService.DeleteDatabase(c.Request.Context(), id, userID); err != nil {
		log.Printf("[Handler:DeleteDatabase] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("Error deleting database"))
		return
	}

	utils.RespondSucces(c, http.StatusOK, "Database deleted successfully", gin.H{"id": id, "status": "DELETED"})
}

func (h *ClaudeDBHandler) RotateCredentials(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("userId")
	requestID := c.GetString("requestID")

	resp, err := h.dbService.RotateCredentials(c.Request.Context(), id, userID)
	if err != nil {
		log.Printf("[Handler:RotateCredentials] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("Error rotating creds"))
		return
	}

	utils.RespondSucces(c, http.StatusOK, "Credentials rotated successfully", resp)
}

// --- 2. Compute Lifecycle ---

func (h *ClaudeDBHandler) StartDatabase(c *gin.Context) {
	id := c.Param("id")
	requestID := c.GetString("requestID")
	userID := c.GetString("userId")

	if err := h.dbService.StartDatabase(c.Request.Context(), id, userID); err != nil {
		log.Printf("[Handler:StartDatabase] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("Error rotating starting database"))
		return
	}

	utils.RespondSucces(c, http.StatusOK, "Database started successfully", gin.H{"id": id, "status": "AVAILABLE"})
}

func (h *ClaudeDBHandler) StopDatabase(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("userId")
	requestID := c.GetString("requestID")

	if err := h.dbService.StopDatabase(c.Request.Context(), id, userID); err != nil {
		log.Printf("[Handler:StopDatabase] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("Error stoping database"))
		return
	}

	utils.RespondSucces(c, http.StatusOK, "Database stopped successfully", gin.H{"id": id, "status": "STOPPED"})
}

func (h *ClaudeDBHandler) RebootDatabase(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("userId")
	requestID := c.GetString("requestID")

	if err := h.dbService.RebootDatabase(c.Request.Context(), id, userID); err != nil {
		log.Printf("[Handler:RebootDatabase] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("Error rebooting database"))

		return
	}

	utils.RespondSucces(c, http.StatusOK, "Database rebooted successfully", gin.H{"id": id, "status": "AVAILABLE"})
}

func (h *ClaudeDBHandler) ModifyDatabase(c *gin.Context) {
	id := c.Param("id")
	userID := c.GetString("userId")
	requestID := c.GetString("requestID")

	db, err := h.dbService.GetDatabase(c.Request.Context(), id, userID)
	if err != nil {
		log.Printf("[Handler:ModifyDatabase] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("Error Modifying database"))
		return
	}

	var payload domain.ModifyDatabasePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Printf("[Handler:ModifyDatabase] Payload unmarshal, bad request for requestID %s, with error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("Bad request"))
		return
	}

	if err := h.dbService.ModifyDatabase(c.Request.Context(), id, userID, payload.VpcID); err != nil {
		log.Printf("[Handler:ModifyDatabase] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to create snapshot"))

		return
	}

	utils.RespondSucces(c, http.StatusOK, "Database modification initiated successfully", gin.H{
		"id":     db.ID,
		"status": "UPDATING",
	})
}

