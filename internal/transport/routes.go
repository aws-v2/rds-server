package transport

import (
	handler "rds/internal/transport/handlers"
	http "rds/internal/transport/handlers"
	"rds/internal/transport/middleware"

	"github.com/gin-gonic/gin"
)

// Handlers struct holds all handler dependencies
type Handlers struct {
	ClaudeDB *handler.ClaudeDBHandler
	Health   *handler.HealthHandler
	Config   *handler.ConfigHandler
	Docs     *handler.DocsHandler
}

// RegisterRoutes registers all application routes
func RegisterRoutes(router *gin.Engine, handlers *Handlers) {
	// API v1 group
	v1 := router.Group("/api/v1/rds")

	// Use user details auth middleware
	v1.Use(middleware.AuthContextMiddleware())

	// Register domain-specific routes
	if handlers.ClaudeDB != nil {
		registerClaudeDBRoutes(v1, handlers.ClaudeDB)
	}
	registerHealthRoutes(v1, handlers.Health)
	registerConfigRoutes(v1, handlers.Config)
	registerDocsRoutes(v1, handlers.Docs)
}
func registerDocsRoutes(v1 *gin.RouterGroup, handler *http.DocsHandler) {
	docs := v1.Group("/docs")
	{
		docs.GET("", handler.GetManifest)
		docs.GET("/:slug", handler.GetDoc)
	}
}
func registerClaudeDBRoutes(api *gin.RouterGroup, handler *http.ClaudeDBHandler) {
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
		databases.PUT("/:id/vpc", handler.AssignVPC)

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

	vpcs := api.Group("/vpcs")
	{
		vpcs.GET("", handler.ListVPCs)
		vpcs.POST("", handler.CreateVPC)
	}

	// --- 6. Volume Management Routes (`/api/v1/rds/volumes`) ---
	volumes := api.Group("/volumes")
	{
		volumes.POST("", handler.CreateVolume)
		volumes.GET("", handler.ListVolumes)
		volumes.GET("/:id", handler.GetVolume)
		volumes.DELETE("/:id", handler.DeleteVolume)
	}

	// --- 7. Scaling Policy Routes (`/api/v1/rds/scaling-policies`) ---
	scaling := api.Group("/scaling-policies")
	{
		scaling.POST("", handler.CreateScalingPolicy)
		scaling.GET("", handler.GetScalingPolicies)
		scaling.PUT("/:id", handler.UpdateScalingPolicy)
		scaling.DELETE("/:id", handler.DeleteScalingPolicy)
	}
}

// registerHealthRoutes registers all health check routes
func registerHealthRoutes(v1 *gin.RouterGroup, handler *handler.HealthHandler) {
	health := v1.Group("/health")
	{
		health.GET("/ping", handler.Ping)
		health.GET("/", handler.Health)
	}
}
func registerConfigRoutes(v1 *gin.RouterGroup, handler *handler.ConfigHandler) {
	v1.GET("/instances/:id/config", handler.GetConfiguration)
	v1.PUT("/instances/:id/config", handler.SetConfiguration)
	v1.GET("/instances/:id/config/history", handler.GetConfigurationHistory)
}
 