package http

import (
	"net/http"
	"rds/internal/application"

	"github.com/gin-gonic/gin"
)

// ConfigHandler handles configuration management requests
type ConfigHandler struct {
	configService *application.ConfigService
}

// NewConfigHandler creates a new configuration handler
func NewConfigHandler(configService *application.ConfigService) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
	}
}

// SetConfigurationRequest represents the request to set a configuration
type SetConfigurationRequest struct {
	Parameter string `json:"parameter" binding:"required"`
	Value     string `json:"value" binding:"required"`
}

// GetConfiguration retrieves all configurations for an instance
func (h *ConfigHandler) GetConfiguration(c *gin.Context) {
	instanceID := c.Param("id")

	configs, err := h.configService.GetConfiguration(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"configurations": configs,
	})
}

// SetConfiguration sets a configuration parameter for an instance
func (h *ConfigHandler) SetConfiguration(c *gin.Context) {
	instanceID := c.Param("id")

	var req SetConfigurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in context",
		})
		return
	}

	err := h.configService.SetConfiguration(c.Request.Context(), instanceID, req.Parameter, req.Value, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration updated successfully",
	})
}

// GetConfigurationHistory retrieves configuration history for an instance
func (h *ConfigHandler) GetConfigurationHistory(c *gin.Context) {
	instanceID := c.Param("id")

	history, err := h.configService.GetConfigurationHistory(c.Request.Context(), instanceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
	})
}
