package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"strconv"
)

type Config struct {
	// Database
	DB DBConfig

	// NATS
	NATS NATSConfig

	// Server
	Server ServerConfig

	// Eureka
	Eureka EurekaConfig

	// Profiles
	Profile string
}

type DBConfig struct {
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

type NATSConfig struct {
	URL      string
	User     string
	Password string
}

type ServerConfig struct {
	Port        string
	ServiceName string
	StoragePath string
	Region      string
}

type EurekaConfig struct {
	ServerURL string
}

func Load() (*Config, error) {
	cfg := &Config{
		DB: DBConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "root"),
			Password:        getEnv("DB_PASSWORD", "root"),
			Database:        getEnv("DB_NAME", "rds_db"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getEnvDuration("DB_CONN_MAX_IDLE_TIME", 10*time.Minute),
		},
		NATS: NATSConfig{
			URL:      getEnv("NATS_URL", "nats://localhost:4222"),
			User:     getEnv("NATS_USER", "auth-server"),
			Password: getEnv("NATS_PASSWORD", "auth-secret"),
		},
		Server: ServerConfig{
			Port:        getEnv("RDS_PORT", "8082"), // Keeping existing RDS conventions here
			ServiceName: getEnv("SERVICE_NAME", "rds-server"),
			StoragePath: getEnv("CODE_STORAGE_PATH", "./storage"),
			Region:      getEnv("AWS_REGION", "eu-north-1"),
		},
		Eureka: EurekaConfig{
			ServerURL: getEnv("EUREKA_SERVER_URL", "http://localhost:8761/eureka"),
		},
		Profile: getEnv("APP_PROFILE", "DEV"),
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return defaultVal
}

type ConfigResponse struct {
	Name            string           `json:"name"`
	Profiles        []string         `json:"profiles"`
	PropertySources []PropertySource `json:"propertySources"`
}

type PropertySource struct {
	Name   string                 `json:"name"`
	Source map[string]interface{} `json:"source"`
}

// LoadConfig fetches configuration from Spring Cloud Config Server and sets them as environment variables
func LoadConfig(url string) error {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("config server returned status: %d", resp.StatusCode)
	}

	var configResp ConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&configResp); err != nil {
		return fmt.Errorf("failed to decode config response: %w", err)
	}

	// Iterate through property sources and set environment variables
	// Higher priority sources come first in the slice
	for i := len(configResp.PropertySources) - 1; i >= 0; i-- {
		ps := configResp.PropertySources[i]
		for k, v := range ps.Source {
			val := fmt.Sprintf("%v", v)
			os.Setenv(k, val)
		}
	}

	return nil
}
