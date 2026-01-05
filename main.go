package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"rds/internal/application"
	"rds/internal/infrastructure/database"
	"rds/internal/infrastructure/docker"
	"rds/internal/infrastructure/event"
	"rds/internal/infrastructure/repository"
	"rds/internal/middleware"
	"rds/internal/transport/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Connect to NATS (for IAM auth)
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	log.Printf("Connecting to NATS at %s...", natsURL)
	natsAdapter, err := event.NewNATSAdapter(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsAdapter.Close()

	// 2. Connect to PostgreSQL
	dbConfig := database.Config{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnvInt("DB_PORT", 5432),
		User:            getEnv("DB_USER", "postgres"),
		Password:        getEnv("DB_PASSWORD", "postgres"),
		Database:        getEnv("DB_NAME", "rds_db"),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
		ConnMaxLifetime: time.Duration(getEnvInt("DB_CONN_MAX_LIFETIME", 5)) * time.Minute,
		ConnMaxIdleTime: time.Duration(getEnvInt("DB_CONN_MAX_IDLE_TIME", 10)) * time.Minute,
	}

	log.Println("Connecting to PostgreSQL...")
	db, err := database.NewPostgresDB(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 3. Run migrations
	log.Println("Running database migrations...")
	if err := database.RunMigrations(db, dbConfig.Database); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Migrations completed successfully")

	// 4. Initialize repository
	repo := repository.NewPostgresRepository(db)

	// 5. Initialize Docker adapter
	log.Println("Initializing Docker adapter...")
	dockerAdapter, err := docker.NewDockerAdapter()
	if err != nil {
		log.Fatalf("Failed to create Docker adapter: %v", err)
	}

	// 6. Create IAM validator
	iamValidator := middleware.NewIAMValidator(natsAdapter.GetConnection())

	// 7. Initialize services
	log.Println("Initializing services...")
	auditService := application.NewAuditService(repo)
	instanceService := application.NewInstanceService(repo, dockerAdapter, auditService)
	healthService := application.NewHealthService(repo, dockerAdapter)
	configService := application.NewConfigService(repo, auditService)

	// 8. Initialize handlers
	log.Println("Initializing HTTP handlers...")
	handlers := &http.Handlers{
		Instance:  http.NewInstanceHandler(instanceService, auditService),
		Health:    http.NewHealthHandler(healthService),
		Config:    http.NewConfigHandler(configService),
		Validator: iamValidator,
	}

	// 9. Setup router
	router := gin.Default()
	http.RegisterRoutes(router, handlers)

	// 10. Start server
	port := getEnv("RDS_PORT", "8082")
	log.Printf("🚀 RDS Service starting on port %s...", port)
	log.Printf("📝 API endpoints:")
	log.Printf("  - POST   /api/v1/rds/instances")
	log.Printf("  - GET    /api/v1/rds/instances")
	log.Printf("  - GET    /api/v1/rds/instances/:id")
	log.Printf("  - DELETE /api/v1/rds/instances/:id")
	log.Printf("  - POST   /api/v1/rds/instances/:id/start")
	log.Printf("  - POST   /api/v1/rds/instances/:id/stop")
	log.Printf("  - GET    /api/v1/rds/health/ping")
	log.Printf("  - GET    /api/v1/rds/health/status")

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Warning: Invalid integer for %s, using default %d", key, defaultValue)
		return defaultValue
	}
	return value
}
