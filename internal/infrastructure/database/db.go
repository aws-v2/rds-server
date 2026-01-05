package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Config holds database connection configuration
type Config struct {
	Host            string
	Port            int
	User            string
	Password        string
	Database        string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database, cfg.SSLMode,
	)

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

	return db, nil
}

// DefaultConfig returns recommended production values for database configuration
func DefaultConfig() Config {
	return Config{
		MaxOpenConns:    25,              // Limit total connections
		MaxIdleConns:    5,               // Keep some connections ready
		ConnMaxLifetime: 5 * time.Minute, // Recycle connections
		ConnMaxIdleTime: 10 * time.Minute, // Close idle connections
		SSLMode:         "require",
	}
}
