package middleware

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// APIKeyValidator defines the interface for API key validation
type APIKeyValidator interface {
	ValidateKey(accessKeyID, secretAccessKey string) (string, error)
}

// IAMValidator validates API keys via NATS communication with IAM service
type IAMValidator struct {
	natsConn *nats.Conn
}

// ValidationRequest represents the NATS validation request
type ValidationRequest struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
}

// ValidationResponse represents the NATS validation response
type ValidationResponse struct {
	Valid    bool     `json:"valid"`
	UserID   string   `json:"userId"`
	Policies []string `json:"policies"`
}

// NewIAMValidator creates a new IAM validator
func NewIAMValidator(natsConn *nats.Conn) *IAMValidator {
	return &IAMValidator{
		natsConn: natsConn,
	}
}

// ValidateKey validates an access key via NATS request to IAM service
func (v *IAMValidator) ValidateKey(accessKeyID, secretAccessKey string) (string, error) {
	// Create validation request
	request := ValidationRequest{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal validation request: %w", err)
	}

	// Send NATS request with timeout
	msg, err := v.natsConn.Request("iam.auth.validate", requestBytes, 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("NATS validation request failed: %w", err)
	}

	// Parse response
	var response ValidationResponse
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal validation response: %w", err)
	}

	// Check if valid
	if !response.Valid {
		return "", fmt.Errorf("invalid credentials")
	}

	return response.UserID, nil
}
