package handler


import (
	"fmt"
	"net/http"
	"rds/internal/application"
	"rds/internal/utils"

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

		utils.RespondSucces(c, http.StatusCreated, "Pinged health successfully", gin.H{
			"status":  "ok",
			"message": "pong",
		})

		// c.JSON(http.StatusOK, )
	} else {
		utils.RespondError(c, http.StatusServiceUnavailable, fmt.Errorf("service ping unavailable"))

	}
}


func (h *HealthHandler) Health(c *gin.Context) {
	if h.healthService.Ping(c.Request.Context()) {

		utils.RespondSucces(c, http.StatusCreated, "Pinged health successfully", gin.H{
			"ping": "pong",
		})

		// c.JSON(http.StatusOK, )
	} else {
		utils.RespondError(c, http.StatusServiceUnavailable, fmt.Errorf("service ping unavailable"))

	}
}
