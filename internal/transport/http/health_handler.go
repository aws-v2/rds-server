package http

import (
	"net/http"
	"rds/internal/application"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check requests
type HealthHandler struct {
	healthService *application.HealthService
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(healthService *application.HealthService) *HealthHandler {
	return &HealthHandler{
		healthService: healthService,
	}
}

// Ping handles simple ping requests
func (h *HealthHandler) Ping(c *gin.Context) {
	if h.healthService.Ping(c.Request.Context()) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"message": "pong",
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "error",
			"message": "service unavailable",
		})
	}
}

// GetStatus handles detailed health status requests
func (h *HealthHandler) GetStatus(c *gin.Context) {
	status, err := h.healthService.GetHealth(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	if status.Healthy {
		c.JSON(http.StatusOK, status)
	} else {
		c.JSON(http.StatusServiceUnavailable, status)
	}
}
