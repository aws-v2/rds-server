package handler


import (
	"fmt"
	"log"
	"net/http"

	"rds/internal/domain"
	"rds/internal/utils"

	"github.com/gin-gonic/gin"
)

// --- 7. Scaling Policy Management ---

func (h *ClaudeDBHandler) CreateScalingPolicy(c *gin.Context) {
	userID := c.GetString("userId")
	requestID := c.GetString("requestID")

	var input domain.ScalingPolicyRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("[Handler:CreateScalingPolicy] Payload unmarshal, bad request for requestID %s, with error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("Bad request"))
		return
	}

	if err := h.dbService.CreateScalingPolicy(c.Request.Context(), userID, input); err != nil {
		log.Printf("[Handler:CreateScalingPolicy] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to Create Scaling policy"))

		return
	}

	utils.RespondSucces(c, http.StatusCreated, "Scaling policy creation request submitted", nil)
}

func (h *ClaudeDBHandler) GetScalingPolicies(c *gin.Context) {
	userID := c.GetString("userId")
	requestID := c.GetString("requestID")

	policies, err := h.dbService.GetScalingPolicies(c.Request.Context(), userID)
	if err != nil {
		log.Printf("[Handler:GetScalingPolicies] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("failed to get Scaling policy"))

		return
	}

	utils.RespondSucces(c, http.StatusOK, "Scaling policies fetched successfully", policies)
}

func (h *ClaudeDBHandler) UpdateScalingPolicy(c *gin.Context) {
	policyID := c.Param("id")
	userID := c.GetString("userId")
	requestID := c.GetString("requestID")

	var input domain.UpdateScalingPolicyRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("[Handler:UpdateScalingPolicy] Payload unmarshal, bad request for requestID %s, with error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusBadRequest, fmt.Errorf("Bad request"))
		return
	}

	if err := h.dbService.UpdateScalingPolicy(c.Request.Context(), userID, policyID, input); err != nil {
		log.Printf("[Handler:DeleteScalingPolicy] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("Error Deleting scaling"))
		return
	}

	utils.RespondSucces(c, http.StatusOK, "Scaling policy update request submitted", nil)
}

func (h *ClaudeDBHandler) DeleteScalingPolicy(c *gin.Context) {
	policyID := c.Param("id")
	userID := c.GetString("userId")
	requestID := c.GetString("requestID")

	if err := h.dbService.DeleteScalingPolicy(c.Request.Context(), userID, policyID); err != nil {
		log.Printf("[Handler:DeleteScalingPolicy] Service call, requestID %s  error %s", requestID, err.Error())
		utils.RespondError(c, http.StatusInternalServerError, fmt.Errorf("Error Deleting scaling"))
		return
	}

	utils.RespondSucces(c, http.StatusOK, "Scaling policy deletion request submitted", policyID)
}
