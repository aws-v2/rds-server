package main

import (
	"database/sql"
	"net/url"
	"rds/internal/application"
	"rds/internal/config"
	"rds/internal/discovery"
	"rds/internal/infrastructure/database"
	"rds/internal/infrastructure/docker"
	"rds/internal/infrastructure/event"
	"rds/internal/infrastructure/repository"
	"rds/internal/logger"
	"rds/internal/messaging"
	"rds/internal/transport/http"
	"rds/internal/utils"
	"time"

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

	logger.Log.Info("Application Profile", zap.String("profile", cfg.Profile))

	// 0.2 Eureka Configuration
	eurekaConfig := discovery.GetEurekaConfig()
	eurekaConfig.ServerURL = cfg.Eureka.ServerURL

	if err := discovery.RegisterWithEureka(eurekaConfig); err != nil {
		logger.Log.Error("Failed to register with Eureka", zap.Error(err))
	} else {
		go discovery.SendHeartbeat(eurekaConfig)
	}

	// 1. Connect to NATS (for IAM auth)
	if cfg.NATS.URL != "" {
		natsURL, err := url.Parse(cfg.NATS.URL)
		if err == nil {
			host := natsURL.Hostname()
			port := natsURL.Port()
			if port == "" {
				port = "4222"
			}
			logger.Log.Info("Checking NATS reachability", zap.String("host", host), zap.String("port", port))

			maxRetries := 5
			for i := 1; i <= maxRetries; i++ {
				if err := utils.CheckReachability(host, port, 2*time.Second); err == nil {
					logger.Log.Info("NATS is reachable")
					break
				} else {
					if i == maxRetries {
						logger.Log.Fatal("FATAL: NATS is not reachable after retries",
							zap.String("host", host),
							zap.String("port", port),
							zap.Error(err))
					}
					logger.Log.Warn("NATS not reachable, retrying...",
						zap.Int("attempt", i),
						zap.Int("max_retries", maxRetries))
					time.Sleep(2 * time.Second)
				}
			}
		}
	}

	logger.Log.Info("Connecting to NATS", zap.String("url", cfg.NATS.URL))
	var natsAdapter *event.NATSAdapter
	if cfg.NATS.User != "" && cfg.NATS.Password != "" {
		natsAdapter, err = event.NewNATSAdapterWithAuth(cfg.NATS.URL, cfg.NATS.User, cfg.NATS.Password)
	} else {
		natsAdapter, err = event.NewNATSAdapter(cfg.NATS.URL)
	}

	if err != nil {
		logger.Log.Fatal("Failed to connect to NATS", zap.Error(err))
	}
	defer natsAdapter.Close()

	// 1.1 Initialize NATS Publisher for Network Service
	var natsPublisher *messaging.NATSPublisher
	if cfg.NATS.URL != "" {
		logger.Log.Info("Initializing NATS Publisher", zap.String("url", cfg.NATS.URL))
		natsPublisher, err = messaging.NewNATSPublisher(cfg.NATS.URL, cfg.NATS.User, cfg.NATS.Password, cfg.NATS.Prefix)
		if err != nil {
			logger.Log.Error("Failed to initialize NATS Publisher", zap.Error(err))
		} else {
			defer natsPublisher.Close()
		}
	}

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
		NatsPrefix:      cfg.NATS.Prefix,
	}

	// TCP Reachability check for Postgres
	logger.Log.Info("Checking Postgres reachability", zap.String("host", dbConfig.Host), zap.Int("port", dbConfig.Port))
	maxRetries := 5
	for i := 1; i <= maxRetries; i++ {
		if err := utils.CheckReachability(dbConfig.Host, dbConfig.Port, 2*time.Second); err == nil {
			logger.Log.Info("Postgres is reachable")
			break
		} else {
			if i == maxRetries {
				// We don't fatal here yet because we have SQLite fallback, but we log the failure
				logger.Log.Error("Postgres is not reachable after retries",
					zap.String("host", dbConfig.Host),
					zap.Int("port", dbConfig.Port),
					zap.Error(err))
			} else {
				logger.Log.Warn("Postgres not reachable, retrying...",
					zap.Int("attempt", i),
					zap.Int("max_retries", maxRetries))
				time.Sleep(2 * time.Second)
			}
		}
	}

	var db *sql.DB
	for i := 1; i <= maxRetries; i++ {
		logger.Log.Info("Attempting to connect to PostgreSQL", zap.Int("attempt", i))
		db, err = database.NewPostgresDB(dbConfig)
		if err == nil {
			break
		}
		logger.Log.Warn("Failed to connect to PostgreSQL", zap.Int("attempt", i), zap.Error(err))
		if i < maxRetries {
			time.Sleep(2 * time.Second)
		}
	}

	if err != nil {
		logger.Log.Warn("Could not connect to PostgreSQL after retries, falling back to SQLite")
		sqlitePath := "lambda.db"
		db, err = database.NewSQLiteDB(sqlitePath)
		if err != nil {
			logger.Log.Fatal("Failed to connect to SQLite fallback", zap.Error(err))
		}
		logger.Log.Info("Connected to SQLite fallback", zap.String("path", sqlitePath))
	} else {
		logger.Log.Info("Successfully connected to PostgreSQL")
	}
	defer db.Close()

	// 3. Run migrations
	logger.Log.Info("Running database migrations...")
	if err := database.RunMigrations(db, dbConfig.Database); err != nil {
		logger.Log.Fatal("Failed to run migrations", zap.Error(err))
	}
	logger.Log.Info("Migrations completed successfully")

	// 4. Initialize repository
	repo := repository.NewPostgresRepository(db)

	// 5. Initialize Docker adapter
	logger.Log.Info("Initializing Docker adapter...")
	portAllocator := docker.NewPortAllocator()

dockerAdapter, err := docker.NewDockerAdapter(portAllocator)
if err != nil {
		logger.Log.Fatal("Failed to create Docker adapter", zap.Error(err))
	}

	// 7. Initialize services
	logger.Log.Info("Initializing services...")
	auditService := application.NewAuditService(repo)
	healthService := application.NewHealthService(repo, dockerAdapter)
	configService := application.NewConfigService(repo, auditService)
	claudeDBService := application.NewClaudeDBService(repo, dockerAdapter, natsPublisher, cfg.Server.Region, cfg.Server.PublicHostIP)
	volumeService := application.NewVolumeService(repo, dockerAdapter, cfg.Server.Region)
	snapshotService := application.NewSnapshotService(repo, dockerAdapter, cfg.Server.Region, claudeDBService)
	docsService := application.NewDocsService("docs")

	workerService := application.NewWorkerService(repo, dockerAdapter)
	workerService.Start()
	defer workerService.Stop()

	metricsCollector := application.NewMetricsCollector(repo, dockerAdapter, natsPublisher, cfg.Server.MetricsURL, cfg.Server.MetricsToken)
	metricsCollector.Start()
	defer metricsCollector.Stop()

	scalingService := application.NewScalingService(repo, dockerAdapter, natsAdapter, cfg.NATS.Prefix)
	scalingService.Start()
	defer scalingService.Stop()

	// 8. Initialize handlers
	logger.Log.Info("Initializing HTTP handlers...")

	// Temporarily passing nil for IAMValidator since it was stripped out in a previous PR
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
	logger.Log.Info("🚀 RDS Service starting", zap.String("port", cfg.Server.Port))
	logger.Log.Info("📝 API endpoints ready")

	if err := router.Run(":" + cfg.Server.Port); err != nil {
		logger.Log.Fatal("Failed to start server", zap.Error(err))
	}
}
