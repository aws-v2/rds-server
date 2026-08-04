package handler


import (
	"fmt"
	"log"
	"math"
	"net/http"
	"time"

	"rds/internal/utils"

	"github.com/gin-gonic/gin"
)

// --- 4. Telemetry & Observability ---

func (h *ClaudeDBHandler) GetMetrics(c *gin.Context) {
	id := c.Param("id")
	accountID := c.GetString("userId")
	requestID := c.GetString("requestID")

	db, err := h.dbService.GetDatabase(c.Request.Context(), id, accountID)
	if err != nil {
		log.Printf("[Handler:GetMetrics] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to get metrics"))

		return
	}

	metrics := gin.H{
		"databaseId":        db.ID,
		"cpuUsagePercent":   2.5,
		"memoryUsageMb":     145.2,
		"activeConnections": 12,
		"diskIops":          45,
	}

	utils.RespondSucces(c, http.StatusOK, "Metrics fetched successfully", metrics)
}

func (h *ClaudeDBHandler) GetLogs(c *gin.Context) {
	id := c.Param("id")
	accountID := c.GetString("userId")
	requestID := c.GetString("requestID")

	db, err := h.dbService.GetDatabase(c.Request.Context(), id, accountID)
	if err != nil {
		log.Printf("[Handler:GetLogs] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to get logs"))

		return
	}

	logs := []string{
		fmt.Sprintf("2026-03-01 10:00:00 UTC [1] LOG:  starting PostgreSQL 15.4 for ClaudeDB %s", db.DBName),
		"2026-03-01 10:00:00 UTC [1] LOG:  listening on IPv4 address \"0.0.0.0\", port 5432",
		"2026-03-01 10:00:00 UTC [1] LOG:  listening on IPv6 address \"::\", port 5432",
		"2026-03-01 10:00:00 UTC [26] LOG:  database system was shut down at 2026-03-01 09:59:59 UTC",
		"2026-03-01 10:00:00 UTC [1] LOG:  database system is ready to accept connections",
	}

	utils.RespondSucces(c, http.StatusOK, "Logs fetched successfully", gin.H{
		"databaseId": db.ID,
		"logs":       logs,
	})
}

func (h *ClaudeDBHandler) GetAggregateMetrics(c *gin.Context) {
	userID := c.GetString("userId")
	requestID := c.GetString("requestID")

	dbs, err := h.dbService.ListDatabases(c.Request.Context(), userID)
	if err != nil {
		log.Printf("[Handler:GetAggregateMetrics] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to get aggregate metrics"))

		return
	}

	// 1. Summary Metrics
	var totalCPU float64
	var totalMemory float64
	var totalConnections int
	var totalDisk float64

	// 2. Breakdown for Bar Graphs (Per Database)
	type DBMetric struct {
		Name   string  `json:"name"`
		CPU    float64 `json:"cpu"`
		Memory float64 `json:"memory"`
		Status string  `json:"status"`
	}
	breakdown := make([]DBMetric, 0, len(dbs))

	for _, db := range dbs {
		// Mocked per-DB metrics
		cpu := 1.5 + math.Mod(float64(len(db.ID)), 5.0)
		mem := 120.0 + (float64(len(db.DBName)) * 10.0)
	 
		breakdown = append(breakdown, DBMetric{
			Name:   db.DBName,
			CPU:    cpu,
			Memory: mem,
		})
	}

	// 3. History for Line Graphs (Simulated hourly data in 5-min intervals)
	type DataPoint struct {
		Timestamp string  `json:"timestamp"`
		CPU       float64 `json:"cpu"`
		Memory    float64 `json:"memory"`
	}
	history := make([]DataPoint, 0, 12)
	now := time.Now().UTC()
	for i := 11; i >= 0; i-- {
		ts := now.Add(time.Duration(-i*5) * time.Minute)
		// Add some jitter to make it look realistic
		jitter := float64(i%3) * 0.5
		history = append(history, DataPoint{
			Timestamp: ts.Format(time.RFC3339),
			CPU:       totalCPU - jitter,
			Memory:    totalMemory - (jitter * 10),
		})
	}

	metrics := gin.H{
		"summary": gin.H{
			"totalDatabases":   len(dbs),
			"activeDatabases":  len(dbs), // Simplified
			"totalCpuUsage":    totalCPU,
			"totalMemoryUsage": totalMemory,
			"totalConnections": totalConnections,
			"totalDiskUsage":   totalDisk,
			"unitCpu":          "%",
			"unitMemory":       "MB",
			"unitDisk":         "GB",
		},
		"breakdown": breakdown,
		"history":   history,
	}

	utils.RespondSucces(c, http.StatusOK, "Aggregate metrics fetched successfully", metrics)
}