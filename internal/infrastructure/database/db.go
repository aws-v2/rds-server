package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var MigrationFiles embed.FS

// Config holds database connection configuration
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	ChannelBinding  string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	NatsPrefix      string
}

// NewPostgresDB creates a new PostgreSQL database connection and runs
// migrations against it before returning.
func NewPostgresDB(cfg Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

	if cfg.ChannelBinding != "" {
		dsn += fmt.Sprintf(" channel_binding=%s", cfg.ChannelBinding)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(cfg.MaxOpenConns)       // Max connections in pool
	db.SetMaxIdleConns(cfg.MaxIdleConns)       // Max idle connections
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime) // Max lifetime of connection
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime) // Max idle time before closing

	// Verify connection with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Run migrations now that the connection is confirmed good
	version, dirty, err := GetMigrationVersion(db, cfg.Database)
	if err == nil && dirty {
		slog.Warn("Database is dirty, forcing version", slog.Uint64("version", uint64(version)))
		if err := ForceVersion(db, cfg.Database, int(version)); err != nil {
			return nil, fmt.Errorf("failed to force migration version: %w", err)
		}
	}

	if err := RunMigrations(db, cfg.Database); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// RunMigrations executes all pending migrations
func RunMigrations(db *sql.DB, dbName string) error {
	sourceDriver, err := iofs.New(MigrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	databaseDriver, err := postgres.WithInstance(db, &postgres.Config{
		DatabaseName: dbName,
	})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, dbName, databaseDriver)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	slog.Info("[MIGRATE] Running up migrations")

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}

// ForceVersion forces the migration version and clears dirty state
func ForceVersion(db *sql.DB, dbName string, version int) error {
	sourceDriver, err := iofs.New(MigrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	databaseDriver, err := postgres.WithInstance(db, &postgres.Config{
		DatabaseName: dbName,
	})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, dbName, databaseDriver)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	if err := m.Force(version); err != nil {
		return fmt.Errorf("failed to force version: %w", err)
	}

	return nil
}

// GetMigrationVersion returns current migration version and dirty state
func GetMigrationVersion(db *sql.DB, dbName string) (uint, bool, error) {
	sourceDriver, err := iofs.New(MigrationFiles, "migrations")
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migration source: %w", err)
	}

	databaseDriver, err := postgres.WithInstance(db, &postgres.Config{
		DatabaseName: dbName,
	})
	if err != nil {
		return 0, false, fmt.Errorf("failed to create database driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, dbName, databaseDriver)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migrator: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil {
		if err == migrate.ErrNilVersion {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}

	return version, dirty, nil
}

// DefaultConfig returns recommended production values for database configuration
func DefaultConfig() Config {
	return Config{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
		SSLMode:         "require",
	}
}

// NewSQLiteDB creates a new SQLite database connection
func NewSQLiteDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping sqlite database: %w", err)
	}

	return db, nil
}