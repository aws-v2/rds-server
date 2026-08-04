package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"rds/internal/interfaces"
	"rds/internal/messaging"
	"time"
)

type RDSIngestRequest struct {
	InstanceID     string  `json:"instance_id"`
	CPUPercent     float64 `json:"cpu_percent"`
	MemUsedMB      int64   `json:"mem_used_mb"`
	MemPercent     float64 `json:"mem_percent"`
	StorageUsedGB  int64   `json:"storage_used_gb"`
	StorageTotalGB int64   `json:"storage_total_gb"`
	Connections    int64   `json:"connections"`
	IOPS           int64   `json:"iops"`
}

type MetricsCollector struct {
	repo         interfaces.RepositoryPort
	dockerClient interfaces.DockerPort
	publisher    messaging.Publisher
	metricsURL   string
	metricsToken string
	interval     time.Duration
	stopChan     chan struct{}
}

func NewMetricsCollector(repo interfaces.RepositoryPort, dockerClient interfaces.DockerPort, publisher messaging.Publisher, metricsURL, metricsToken string) *MetricsCollector {
	return &MetricsCollector{
		repo:         repo,
		dockerClient: dockerClient,
		publisher:    publisher,
		metricsURL:   metricsURL,
		metricsToken: metricsToken,
		interval:     15 * time.Second,
		stopChan:     make(chan struct{}),
	}
}

func (c *MetricsCollector) Start() {
	log.Printf("[METRICS] Starting RDS metrics collector targeting %s", c.metricsURL)
	go c.run()
}

func (c *MetricsCollector) Stop() {
	close(c.stopChan)
}

func (c *MetricsCollector) run() {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.collectAndSend()
		}
	}
}

func (c *MetricsCollector) collectAndSend() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbs, err := c.repo.ListAllActiveDatabases(ctx)
	if err != nil {
		log.Printf("[METRICS] Failed to list active databases: %v", err)
		return
	}

	for _, db := range dbs {
		containerName := fmt.Sprintf("claudedb-prod-%s", db.ID)
		stats, err := c.dockerClient.GetContainerStats(ctx, containerName)
		if err != nil {
			log.Printf("[METRICS] Failed to get stats for %s: %v", containerName, err)
			continue
		}

		memPercent := 0.0
		if stats.MemoryLimitBytes > 0 {
			memPercent = (float64(stats.MemoryUsageBytes) / float64(stats.MemoryLimitBytes)) * 100.0
		}

		// Fetch dynamic IAM token for this instance
		token, err := c.publisher.RequestInstanceToken(db.UserID, db.ID)
		if err != nil {
			log.Printf("[METRICS] Failed to fetch IAM token for %s: %v", db.ID, err)
			// We can decide to skip or use the default token.
			// User said "otherwise the metrics wont go", so let's skip on error.
			continue
		}

		payload := RDSIngestRequest{
			InstanceID:     db.ID,
			CPUPercent:     stats.CPUPercentage,
			MemUsedMB:      stats.MemoryUsageBytes / (1024 * 1024),
			MemPercent:     memPercent,
			StorageUsedGB:  0, // Future: could get from du inside container
			StorageTotalGB: 0,
			Connections:    0, // Future: could get from pg_stat_activity
			IOPS:           0,
		}

		if err := c.sendToServer(payload, token); err != nil {
			log.Printf("[METRICS] Failed to send metrics for %s: %v", db.ID, err)
		} else {
			log.Printf("[METRICS] Successfully sent metrics for %s (CPU: %.2f%%, MEM: %.2f%%)", db.ID, payload.CPUPercent, payload.MemPercent)
		}
	}
}

func (c *MetricsCollector) sendToServer(payload RDSIngestRequest, token string) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", c.metricsURL+"/rds/ingest", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// Use the dynamic token if provided, otherwise fallback to static token
	authToken := token
	if authToken == "" {
		authToken = c.metricsToken
	}

	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("server returned status: %d", resp.StatusCode)
	}

	return nil
}
