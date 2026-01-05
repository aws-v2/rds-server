package http

import (
	"net/http"
	"rds/internal/application"
	"rds/internal/infrastructure/dto"

	"strconv"

	"github.com/gin-gonic/gin"
)

// InstanceHandler handles database instance requests
type InstanceHandler struct {
	instanceService *application.InstanceService
	auditService    *application.AuditService
}

// NewInstanceHandler creates a new instance handler
func NewInstanceHandler(instanceService *application.InstanceService, auditService *application.AuditService) *InstanceHandler {
	return &InstanceHandler{
		instanceService: instanceService,
		auditService:    auditService,
	}
}

// CreateInstance handles instance creation requests
func (h *InstanceHandler) CreateInstance(c *gin.Context) {
	var input dto.CreateInstanceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Get user ID from context
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in context",
		})
		return
	}

	// Create instance
	req := application.CreateInstanceRequest{
		Name:     input.Name,
		User:     input.User,
		Password: input.Password,
		OwnerID:  userID.(string),
	}

	instance, err := h.instanceService.CreateInstance(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Convert to output DTO
	output := dto.InstanceOutput{
		ID:        instance.ID,
		Name:      instance.Name,
		Engine:    instance.Engine,
		Port:      instance.Port,
		User:      instance.User,
		Status:    string(instance.Status),
		CreatedAt: instance.CreatedAt,
	}

	c.JSON(http.StatusCreated, output)
}

// GetInstance handles requests to get a specific instance
func (h *InstanceHandler) GetInstance(c *gin.Context) {
	instanceID := c.Param("id")

	// Get user ID from context
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in context",
		})
		return
	}

	instance, err := h.instanceService.GetInstance(c.Request.Context(), instanceID, userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	output := dto.InstanceOutput{
		ID:        instance.ID,
		Name:      instance.Name,
		Engine:    instance.Engine,
		Port:      instance.Port,
		User:      instance.User,
		Status:    string(instance.Status),
		CreatedAt: instance.CreatedAt,
	}

	c.JSON(http.StatusOK, output)
}

// ListInstances handles requests to list all instances for a user
func (h *InstanceHandler) ListInstances(c *gin.Context) {
	// Get user ID from context
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in context",
		})
		return
	}

	instances, err := h.instanceService.ListInstances(c.Request.Context(), userID.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Convert to output DTOs
	outputs := make([]dto.InstanceOutput, 0, len(instances))
	for _, inst := range instances {
		outputs = append(outputs, dto.InstanceOutput{
			ID:        inst.ID,
			Name:      inst.Name,
			Engine:    inst.Engine,
			Port:      inst.Port,
			User:      inst.User,
			Status:    string(inst.Status),
			CreatedAt: inst.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, dto.ListInstancesOutput{
		Instances: outputs,
	})
}

// DeleteInstance handles instance deletion requests
func (h *InstanceHandler) DeleteInstance(c *gin.Context) {
	instanceID := c.Param("id")

	// Get user ID from context
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in context",
		})
		return
	}

	if err := h.instanceService.DeleteInstance(c.Request.Context(), instanceID, userID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Instance deleted successfully",
	})
}

// StartInstance handles requests to start a stopped instance
func (h *InstanceHandler) StartInstance(c *gin.Context) {
	instanceID := c.Param("id")

	// Get user ID from context
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in context",
		})
		return
	}

	if err := h.instanceService.StartInstance(c.Request.Context(), instanceID, userID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Instance started successfully",
	})
}

// StopInstance handles requests to stop a running instance
func (h *InstanceHandler) StopInstance(c *gin.Context) {
	instanceID := c.Param("id")

	// Get user ID from context
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User ID not found in context",
		})
		return
	}

	if err := h.instanceService.StopInstance(c.Request.Context(), instanceID, userID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Instance stopped successfully",
	})
}

// GetInstanceLogs handles requests to get audit logs for an instance
func (h *InstanceHandler) GetInstanceLogs(c *gin.Context) {
	instanceID := c.Param("id")

	// Get limit from query parameter (default: 50)
	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	logs, err := h.auditService.GetInstanceLogs(c.Request.Context(), instanceID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs": logs,
	})
}
