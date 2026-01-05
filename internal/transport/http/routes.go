package http

import (
	"rds/internal/middleware"

	"github.com/gin-gonic/gin"
)

// Handlers struct holds all handler dependencies
type Handlers struct {
	Instance  *InstanceHandler
	Health    *HealthHandler
	Config    *ConfigHandler
	Validator middleware.APIKeyValidator
}

// RegisterRoutes registers all application routes
func RegisterRoutes(router *gin.Engine, handlers *Handlers) {
	// API v1 group
	v1 := router.Group("/api/v1/rds")

	// Apply authentication middleware to all routes
	v1.Use(middleware.APIKeyAuthMiddleware(handlers.Validator))

	// Register domain-specific routes
	registerInstanceRoutes(v1, handlers.Instance)
	registerHealthRoutes(v1, handlers.Health)
	registerConfigRoutes(v1, handlers.Config)
}

// registerInstanceRoutes registers all instance management routes
func registerInstanceRoutes(v1 *gin.RouterGroup, handler *InstanceHandler) {
	instances := v1.Group("/instances")
	{
		// Create new instance
		instances.POST("", handler.CreateInstance)

		// List all instances for the authenticated user
		instances.GET("", handler.ListInstances)

		// Get specific instance details
		instances.GET("/:id", handler.GetInstance)

		// Delete instance
		instances.DELETE("/:id", handler.DeleteInstance)

		// Start instance
		instances.POST("/:id/start", handler.StartInstance)

		// Stop instance
		instances.POST("/:id/stop", handler.StopInstance)

		// Get instance audit logs
		instances.GET("/:id/logs", handler.GetInstanceLogs)
	}
}

// registerHealthRoutes registers all health check routes
func registerHealthRoutes(v1 *gin.RouterGroup, handler *HealthHandler) {
	health := v1.Group("/health")
	{
		health.GET("/ping", handler.Ping)
		health.GET("/status", handler.GetStatus)
	}
}

// registerConfigRoutes registers all configuration management routes
func registerConfigRoutes(v1 *gin.RouterGroup, handler *ConfigHandler) {
	// Configuration routes are nested under instances
	v1.GET("/instances/:id/config", handler.GetConfiguration)
	v1.PUT("/instances/:id/config", handler.SetConfiguration)
	v1.GET("/instances/:id/config/history", handler.GetConfigurationHistory)
}
