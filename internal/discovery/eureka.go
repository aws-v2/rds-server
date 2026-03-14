package discovery

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"rds/internal/logger"

	"go.uber.org/zap"
)

// EurekaConfig holds Eureka registration configuration
type EurekaConfig struct {
	ServerURL         string
	AppName           string
	HostName          string
	IPAddr            string
	Port              int
	VipAddress        string
	InstanceID        string
	HeartbeatInterval time.Duration
}

// GetEurekaConfig reads Eureka configuration from environment variables
func GetEurekaConfig() *EurekaConfig {
	return &EurekaConfig{
		ServerURL:         getEnv("EUREKA_SERVER_URL", "http://localhost:8761/eureka"),
		AppName:           getEnv("EUREKA_APP_NAME", "RDS-SERVICE"), // Updated from LAMBDA-SERVICE
		HostName:          getEnv("EUREKA_HOSTNAME", "localhost"),
		IPAddr:            getEnv("EUREKA_IP_ADDR", "127.0.0.1"),
		Port:              getEnvInt("RDS_PORT", 8087), // Updated to match RDS port config
		VipAddress:        getEnv("EUREKA_VIP_ADDRESS", "rds-service"),
		InstanceID:        getEnv("EUREKA_INSTANCE_ID", "rds-service:8087"),
		HeartbeatInterval: getEnvDuration("EUREKA_HEARTBEAT_INTERVAL", 30*time.Second),
	}
}

// RegisterWithEureka registers the service instance with Eureka server
func RegisterWithEureka(config *EurekaConfig) error {
	instance := map[string]interface{}{
		"instance": map[string]interface{}{
			"instanceId": config.InstanceID,
			"hostName":   config.HostName,
			"app":        config.AppName,
			"ipAddr":     config.IPAddr,
			"vipAddress": config.VipAddress,
			"status":     "UP",
			"port": map[string]interface{}{
				"$":        config.Port,
				"@enabled": "true",
			},
			"dataCenterInfo": map[string]interface{}{
				"@class": "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo",
				"name":   "MyOwn",
			},
			"healthCheckUrl": fmt.Sprintf("http://%s:%d/api/v1/rds/health/status", config.HostName, config.Port),
			"statusPageUrl":  fmt.Sprintf("http://%s:%d/api/v1/rds/health/status", config.HostName, config.Port),
			"homePageUrl":    fmt.Sprintf("http://%s:%d/", config.HostName, config.Port),
		},
	}

	jsonData, err := json.Marshal(instance)
	if err != nil {
		return fmt.Errorf("failed to marshal Eureka registration data: %w", err)
	}

	url := fmt.Sprintf("%s/apps/%s", config.ServerURL, config.AppName)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create registration request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register with Eureka: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("eureka registration failed with status %d: %s", resp.StatusCode, string(body))
	}

	logger.Log.Info("✅ Successfully registered with Eureka server", zap.String("url", config.ServerURL))
	return nil
}

// SendHeartbeat sends periodic heartbeats to Eureka server
func SendHeartbeat(config *EurekaConfig) {
	ticker := time.NewTicker(config.HeartbeatInterval)
	defer ticker.Stop()

	url := fmt.Sprintf("%s/apps/%s/%s", config.ServerURL, config.AppName, config.InstanceID)
	client := &http.Client{Timeout: 5 * time.Second}

	for range ticker.C {
		req, _ := http.NewRequest("PUT", url, nil)

		resp, err := client.Do(req)
		if err != nil {
			logger.Log.Error("⚠️  Heartbeat request failed", zap.Error(err))
			continue
		}

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
			logger.Log.Warn("⚠️  Heartbeat failed", zap.Int("status", resp.StatusCode))
		} else {
			logger.Log.Debug("💓 Heartbeat sent successfully to Eureka")
		}

		resp.Body.Close()
	}
}

// DeregisterFromEureka removes the service instance from Eureka
func DeregisterFromEureka(config *EurekaConfig) error {
	url := fmt.Sprintf("%s/apps/%s/%s", config.ServerURL, config.AppName, config.InstanceID)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create deregistration request: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to deregister from Eureka: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("deregistration failed with status %d: %s", resp.StatusCode, string(body))
	}

	logger.Log.Info("✅ Successfully deregistered from Eureka server")
	return nil
}

// Helpers (internal to discovery pakcage)

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return fallback
}
