package main

import (
	"database/sql"
	"log"
	"time"

	"rds/internal/application"
	"rds/internal/config"
	"rds/internal/discovery"
	"rds/internal/infrastructure/database"
	"rds/internal/infrastructure/docker"
	"rds/internal/infrastructure/event"
	"rds/internal/infrastructure/repository"
	"rds/internal/logger"
	"rds/internal/transport/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// 0. Initialize Logger
	logger.Init()
	defer logger.Log.Sync()
	logger.Log.Info("Starting RDS Service...")

	// 0.1 Load Configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Log.Fatal("Failed to load configuration", zap.Error(err))
	}

	// 0.2 Eureka Configuration
	eurekaConfig := discovery.GetEurekaConfig()
	eurekaConfig.ServerURL = cfg.Eureka.ServerURL

	if err := discovery.RegisterWithEureka(eurekaConfig); err != nil {
		logger.Log.Error("Failed to register with Eureka", zap.Error(err))
	} else {
		go discovery.SendHeartbeat(eurekaConfig)
	}

	// 1. Connect to NATS (for IAM auth)
	logger.Log.Info("Connecting to NATS", zap.String("url", cfg.NATS.URL))
	var natsAdapter *event.NATSAdapter
	if cfg.NATS.User != "" && cfg.NATS.Password != "" {
		natsAdapter, err = event.NewNATSAdapterWithAuth(cfg.NATS.URL, cfg.NATS.User, cfg.NATS.Password)
	} else {
		natsAdapter, err = event.NewNATSAdapter(cfg.NATS.URL)
	}

	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer natsAdapter.Close()

	// 2. Connect to PostgreSQL with Retries
	dbConfig := database.Config{
		Host:            cfg.DB.Host,
		Port:            cfg.DB.Port,
		User:            cfg.DB.User,
		Password:        cfg.DB.Password,
		Database:        cfg.DB.Database,
		SSLMode:         cfg.DB.SSLMode,
		MaxOpenConns:    cfg.DB.MaxOpenConns,
		MaxIdleConns:    cfg.DB.MaxIdleConns,
		ConnMaxLifetime: cfg.DB.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.DB.ConnMaxIdleTime,
	}

	var db *sql.DB
	maxRetries := 4

	for i := 0; i < maxRetries; i++ {
		log.Printf("Attempting to connect to PostgreSQL (attempt %d)...", i+1)
		db, err = database.NewPostgresDB(dbConfig)
		if err == nil {
			break
		}
		log.Printf("Failed to connect to PostgreSQL on attempt %d: %v", i+1, err)
		if i < maxRetries-1 {
			time.Sleep(2 * time.Second)
		}
	}

	if err != nil {
		log.Printf("Could not connect to PostgreSQL after retries, falling back to SQLite")
		sqlitePath := "lambda.db" // Using a safe local default for now
		db, err = database.NewSQLiteDB(sqlitePath)
		if err != nil {
			log.Fatalf("Failed to connect to SQLite fallback: %v", err)
		}
		log.Printf("Connected to SQLite fallback at %s", sqlitePath)
	} else {
		log.Println("Successfully connected to PostgreSQL")
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

	// 7. Initialize services
	log.Println("Initializing services...")
	auditService := application.NewAuditService(repo)
	healthService := application.NewHealthService(repo, dockerAdapter)
	configService := application.NewConfigService(repo, auditService)
	claudeDBService := application.NewClaudeDBService(repo, dockerAdapter, cfg.Server.Region)
	volumeService := application.NewVolumeService(repo, dockerAdapter, cfg.Server.Region)
	snapshotService := application.NewSnapshotService(repo, dockerAdapter, cfg.Server.Region, claudeDBService)
	docsService := application.NewDocsService("docs")

	workerService := application.NewWorkerService(repo, dockerAdapter)
	workerService.Start()
	defer workerService.Stop()

	// 8. Initialize handlers
	log.Println("Initializing HTTP handlers...")

	// Temporarily passing nil for IAMValidator since it was stripped out in a previous PR (Conversation 5e4b9bb0-c307-4ef0-a8f4-52054b727f6e)
	claudeDBHandler := http.NewClaudeDBHandler(claudeDBService, volumeService, snapshotService, nil)
	healthHandler := http.NewHealthHandler(healthService)
	configHandler := http.NewConfigHandler(configService)
	docsHandler := http.NewDocsHandler(docsService)

	handlers := &http.Handlers{
		ClaudeDB: claudeDBHandler,
		Health:   healthHandler,
		Config:   configHandler,
		Docs:     docsHandler,
	}

	// 9. Setup router
	router := gin.Default()
	http.RegisterRoutes(router, handlers)

	// 10. Start server
	log.Printf("🚀 RDS Service starting on port %s...", cfg.Server.Port)
	log.Printf("📝 API endpoints:")
	log.Printf("  - POST   /api/v1/rds/databases")
	log.Printf("  - GET    /api/v1/rds/health/ping")
	log.Printf("  - GET    /api/v1/rds/health/status")

	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
