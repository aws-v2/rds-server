package http

import (
	"github.com/gin-gonic/gin"
)

// Handlers struct holds all handler dependencies
type Handlers struct {
	ClaudeDB *ClaudeDBHandler
	Health   *HealthHandler
	Config   *ConfigHandler
}

// RegisterRoutes registers all application routes
func RegisterRoutes(router *gin.Engine, handlers *Handlers) {
	// API v1 group
	v1 := router.Group("/api/v1/rds")

	// Register domain-specific routes
	if handlers.ClaudeDB != nil {
		registerClaudeDBRoutes(v1, handlers.ClaudeDB)
	}
	registerHealthRoutes(v1, handlers.Health)
	registerConfigRoutes(v1, handlers.Config)
}

func registerClaudeDBRoutes(api *gin.RouterGroup, handler *ClaudeDBHandler) {
	databases := api.Group("/databases")
	{
		databases.POST("", handler.CreateDatabase)
		databases.GET("", handler.ListDatabases)
		databases.GET("/:id", handler.GetDatabase)
		databases.DELETE("/:id", handler.DeleteDatabase)
		databases.POST("/:id/rotate-credentials", handler.RotateCredentials)

		// 1. Power & Compute Lifecycle
		databases.POST("/:id/start", handler.StartDatabase)
		databases.POST("/:id/stop", handler.StopDatabase)
		databases.POST("/:id/reboot", handler.RebootDatabase)

		// 2. Compute Modification
		databases.PATCH("/:id", handler.ModifyDatabase)

		// 3. Backup & Restore
		databases.POST("/:id/snapshots", handler.CreateSnapshot)
		databases.GET("/:id/snapshots", handler.ListSnapshots)

		// 4. Telemetry
		databases.GET("/:id/metrics", handler.GetMetrics)
		databases.GET("/:id/logs", handler.GetLogs)

		// 4b. Aggregated Telemetry
		databases.GET("/metrics/aggregate", handler.GetAggregateMetrics)

		// 5. Configuration
		databases.GET("/:id/parameters", handler.GetParameters)
		databases.PATCH("/:id/parameters", handler.ModifyParameters)

		// Restore is a root-level operation on databases but conceptually uses a snapshot
		databases.POST("/restore", handler.RestoreDatabase)
	}

	snapshots := api.Group("/snapshots")
	{
		snapshots.DELETE("/:snapshot_id", handler.DeleteSnapshot)
	}

	// --- 6. Volume Management Routes (`/api/v1/rds/volumes`) ---
	volumes := api.Group("/volumes")
	{
		// Use auth middleware for real implementation, skipped for this refactor MVP
		volumes.POST("", handler.CreateVolume)
		volumes.GET("", handler.ListVolumes)
		volumes.GET("/:id", handler.GetVolume)
		volumes.DELETE("/:id", handler.DeleteVolume)
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
