package database

// import (
// 	"database/sql"
// 	"embed"
// 	"fmt"

// 	"github.com/golang-migrate/migrate/v4"
// 	"github.com/golang-migrate/migrate/v4/database/postgres"
// 	"github.com/golang-migrate/migrate/v4/source/iofs"
// )

 

// // RunMigrations executes all pending database migrations
// func RunMigrations(db *sql.DB, dbName string) error {
// 	// Check if the driver is postgres
// 	// We can try to get the driver name from the db connection if available, 
// 	// or assume that since we are using postgres.WithInstance, it will fail for others.
// 	// To be safe and informative, we'll try to create the driver and handle the error gracefully if it's not postgres.
	
// 	driver, err := postgres.WithInstance(db, &postgres.Config{})
// 	if err != nil {
// 		// If it's not postgres, we might be using SQLite fallback.
// 		// Since migrations are Postgres-specific, we skip them for other drivers.
// 		fmt.Printf("Warning: Migration driver initialization failed (likely not using PostgreSQL): %v. Skipping migrations.\n", err)
// 		return nil
// 	}

// 	sourceDriver, err := iofs.New(migrationsFS, "migrations")
// 	if err != nil {
// 		return fmt.Errorf("failed to create migration source: %w", err)
// 	}

// 	m, err := migrate.NewWithInstance("iofs", sourceDriver, dbName, driver)
// 	if err != nil {
// 		return fmt.Errorf("failed to create migrator: %w", err)
// 	}

// 	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
// 		return fmt.Errorf("failed to run migrations: %w", err)
// 	}

// 	return nil
// }

// // GetMigrationVersion returns the current migration version
// func GetMigrationVersion(db *sql.DB, dbName string) (version uint, dirty bool, err error) {
// 	driver, err := postgres.WithInstance(db, &postgres.Config{})
// 	if err != nil {
// 		return 0, false, fmt.Errorf("failed to create migration driver: %w", err)
// 	}

// 	sourceDriver, err := iofs.New(migrationsFS, "migrations")
// 	if err != nil {
// 		return 0, false, fmt.Errorf("failed to create migration source: %w", err)
// 	}

// 	m, err := migrate.NewWithInstance("iofs", sourceDriver, dbName, driver)
// 	if err != nil {
// 		return 0, false, fmt.Errorf("failed to create migrator: %w", err)
// 	}

// 	version, dirty, err = m.Version()
// 	if err != nil {
// 		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
// 	}

// 	return version, dirty, nil
// }
