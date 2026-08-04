package handler


import (
	"fmt"
	"log"
	"net/http"

	"rds/internal/domain"
	"rds/internal/utils"

	"github.com/gin-gonic/gin"
)


func (h *ClaudeDBHandler) GetParameters(c *gin.Context) {
	id := c.Param("id")
	accountID := c.GetString("userId")
	requestID := c.GetString("requestID")

	db, err := h.dbService.GetDatabase(c.Request.Context(), id, accountID)
	if err != nil {

		log.Printf("[Handler:GetParameters] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to get parameters"))

		return
	}

	params := map[string]interface{}{
		"max_connections":      100,
		"shared_buffers":       "128MB",
		"work_mem":             "4MB",
		"maintenance_work_mem": "64MB",
		"wal_level":            "replica",
	}

	utils.RespondSucces(c, http.StatusOK, "Parameters fetched successfully", gin.H{
		"databaseId": db.ID,
		"parameters": params,
	})
}

func (h *ClaudeDBHandler) ModifyParameters(c *gin.Context) {
	id := c.Param("id")
	accountID := c.GetString("userId")
	requestID := c.GetString("requestID")

	db, err := h.dbService.GetDatabase(c.Request.Context(), id, accountID)
	if err != nil {
		log.Printf("[Handler:ModifyParameters] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusNotFound, fmt.Errorf("failed to modify parameters"))
		return
	}

	var payload domain.ModifyParametersPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Printf("[Handler:ModifyParameters] Payload unmarshal, bad request for requestID %s, with error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("Bad request"))

		return
	}

	pendingReboot := false
	for k := range payload.Parameters {
		if k == "shared_buffers" || k == "max_connections" || k == "wal_level" {
			pendingReboot = true
		}
	}

	if pendingReboot {
		err := h.dbService.RebootDatabase(c.Request.Context(), id, accountID)
		if err != nil {
			log.Printf("[Handler:ModifyParameters] Service call, requestID %s  error %s", requestID, err.Error())
			utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to modify parameters"))
			return
		}
	}

	utils.RespondSucces(c, http.StatusOK, "Parameters applied successfully", gin.H{
		"databaseId":        db.ID,
		"pendingReboot":     pendingReboot,
		"updatedParameters": payload.Parameters,
	})
}

