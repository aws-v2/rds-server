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


func (h *ClaudeDBHandler) CreateVolume(c *gin.Context) {
	accountID := c.GetString("userId")
	requestID := c.GetString("requestID")

	var payload domain.CreateVolumePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		log.Printf("[Handler:CreateVolume] Payload unmarshal, bad request for requestID %s, with error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("Bad request"))

		return
	}

	vol, err := h.volumeService.CreateVolume(c.Request.Context(), application.CreateVolumeRequest{
		Name:      payload.Name,
		SizeGB:    payload.SizeGB,
		AccountID: accountID,
	})
	if err != nil {
		log.Printf("[Handler:CreateVolume] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to get parameters"))
		return
	}
	utils.RespondSucces(c, http.StatusCreated, "Volume created successfully", vol)
}

func (h *ClaudeDBHandler) ListVolumes(c *gin.Context) {
	accountID := c.GetString("userId")
	requestID := c.GetString("requestID")

	vols, err := h.volumeService.ListVolumes(c.Request.Context(), accountID)
	if err != nil {
		log.Printf("[Handler:ListVolumes] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to list volumes"))

		return
	}

	utils.RespondSucces(c, http.StatusOK, "Volumes fetched successfully", vols)
}

func (h *ClaudeDBHandler) GetVolume(c *gin.Context) {
	id := c.Param("id")
	accountID := c.GetString("userId")
	requestID := c.GetString("requestID")

	vol, err := h.volumeService.GetVolume(c.Request.Context(), id, accountID)
	if err != nil {
		log.Printf("[Handler:GetVolume] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to get volume"))

		return
	}

	utils.RespondSucces(c, http.StatusOK, "Volume fetched successfully", vol)
}

func (h *ClaudeDBHandler) DeleteVolume(c *gin.Context) {
	id := c.Param("id")
	accountID := c.GetString("userId")
	requestID := c.GetString("requestID")

	if err := h.volumeService.DeleteVolume(c.Request.Context(), id, accountID); err != nil {
		log.Printf("[Handler:DeleteVolume] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to delete volume"))

		return
	}

	utils.RespondSucces(c, http.StatusOK, "Volume deleted successfully", gin.H{"id": id})
}