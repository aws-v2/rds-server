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

type ConfigHandler struct {
	configService *application.ConfigService
}

func NewConfigHandler(configService *application.ConfigService) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
	}
}

func (h *ConfigHandler) GetConfiguration(c *gin.Context) {
	instanceID := c.Param("id")
	requestID := c.GetString("requestID")

	configs, err := h.configService.GetConfiguration(c.Request.Context(), instanceID)
	if err != nil {
		log.Printf("[Handler:GetConfiguration] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to get configurations"))
		return
	}

	utils.RespondSucces(c, http.StatusOK, "Configurations retrieved successfully", configs)

}

// SetConfiguration sets a configuration parameter for an instance
func (h *ConfigHandler) SetConfiguration(c *gin.Context) {
	instanceID := c.Param("id")
	requestID := c.GetString("requestID")
	userID := c.GetString("userId")

	var req domain.SetConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[Handler:SetConfiguration] Payload unmarshal, bad request for requestID %s, with error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("Bad request"))

		return
	}

	err := h.configService.SetConfiguration(c.Request.Context(), instanceID, req.Parameter, req.Value, userID)
	if err != nil {
		log.Printf("[Handler:SetConfiguration] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to set configurations"))
		return
	}
	utils.RespondSucces(c, http.StatusOK, "Configuration updated successfully", gin.H{
		"id":     instanceID,
		"status": "UPDATED",
	})

}

// GetConfigurationHistory retrieves configuration history for an instance
func (h *ConfigHandler) GetConfigurationHistory(c *gin.Context) {
	instanceID := c.Param("id")
	requestID := c.GetString("requestID")

	history, err := h.configService.GetConfigurationHistory(c.Request.Context(), instanceID)
	if err != nil {
		log.Printf("[Handler:GetConfigurationHistory] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to get configurations history"))
		return
	}
	utils.RespondSucces(c, http.StatusOK, "Configurations retrieved successfully", history)

}
