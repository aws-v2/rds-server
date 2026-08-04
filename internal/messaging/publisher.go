package messaging

import (
	"encoding/json"
	"fmt"
	"log"
	"rds/internal/domain"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// Publisher defines the interface for instance network operations.
type Publisher interface {
	ExposeDatabase(tenantID, resourceID, privateIP, publicIP string, privatePort int) (publicPort int, err error)
	UnexposeDatabase(resourceID string) error
	ListVPCs(tenantID string) ([]domain.VPC, error)
	CreateVPC(tenantID, vpcName, requestedBy string) error
	ReconcileVPCs() error
	RequestInstanceToken(userID, instanceID string) (string, error)
	PublishScalingPolicy(tenantID string, policy domain.ScalingPolicyRequest) error
	GetScalingPolicies(tenantID string) ([]domain.ScalingPolicy, error)
	UpdateScalingPolicy(tenantID, policyID string, req domain.UpdateScalingPolicyRequest) error
	DeleteScalingPolicy(tenantID, policyID string) error

	// ProvisionRDSInstance dispatches a VM provisioning request to the EC2 service.
	ProvisionRDSInstance(event domain.ProvisionInstanceEvent) (domain.EC2Response,error)
}

// NATSPublisher implements the Publisher interface using NATS Request-Response pattern.
type NATSPublisher struct {
	nc     *nats.Conn
	prefix string
}

type exposeRDSRequest struct {
	CorrelationID string `json:"correlation_id"`
	TenantID      string `json:"tenant_id"`
	ResourceID    string `json:"resource_id"`
	PrivateIP     string `json:"private_ip"`
	PrivatePort   int    `json:"private_port"`
	PublicIP      string `json:"public_ip"`
}

type exposeRDSResponse struct {
	Success    bool   `json:"success"`
	PublicPort int    `json:"public_port"`
	Error      string `json:"error"`
}

type unexposeRDSRequest struct {
	CorrelationID string `json:"correlation_id"`
	ResourceID    string `json:"resource_id"`
}

type unexposeRDSResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type createVPCEvent struct {
	CorrelationID string `json:"correlation_id"`
	TenantID      string `json:"tenant_id"`
	VPCName       string `json:"vpc_name"`
	RequestedBy   string `json:"requested_by"`
}

type listVPCsRequest struct {
	CorrelationID string `json:"correlation_id"`
	TenantID      string `json:"tenant_id"`
}

type listVPCsResponse struct {
	CorrelationID string       `json:"correlation_id"`
	TenantID      string       `json:"tenant_id"`
	VPCs          []domain.VPC `json:"vpcs"`
	Error         string       `json:"error,omitempty"`
}

type reconcileVPCsRequest struct {
	CorrelationID string `json:"correlation_id"`
}

type reconcileVPCsResponse struct {
	CorrelationID string `json:"correlation_id"`
	Success       bool   `json:"success"`
	Message       string `json:"message,omitempty"`
	Error         string `json:"error,omitempty"`
}

type instanceTokenRequest struct {
	InstanceID string `json:"instance_id"`
	UserID     string `json:"user_id"`
}

type instanceTokenResponse struct {
	Token string `json:"token"`
	Error string `json:"error,omitempty"`
}

// NewNATSPublisher connects to NATS and returns a NATSPublisher instance.
func NewNATSPublisher(url, user, password, prefix string) (*NATSPublisher, error) {
	opts := []nats.Option{
		nats.UserInfo(user, password),
	}

	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &NATSPublisher{nc: nc, prefix: prefix}, nil
}

// Close closes the NATS connection.
func (p *NATSPublisher) Close() {
	if p.nc != nil {
		p.nc.Close()
	}
}

// ExposeDatabase sends a request to the Network Service to expose an RDS container publicly via DNAT/SNAT.
func (p *NATSPublisher) ExposeDatabase(tenantID, resourceID, privateIP, publicIP string, privatePort int) (int, error) {
	correlationID := uuid.New().String()
	req := exposeRDSRequest{
		CorrelationID: correlationID,
		TenantID:      tenantID,
		ResourceID:    resourceID,
		PrivateIP:     privateIP,
		PrivatePort:   privatePort,
		PublicIP:      publicIP,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %w", err)
	}
	log.Printf("[RDS-NATS] Sending ExposeDatabase request: %s", string(reqData))

	msg, err := p.nc.Request(fmt.Sprintf("%s.network.rds.expose", p.prefix), reqData, 5*time.Second)
	if err != nil {
		return 0, fmt.Errorf("NATS request failed: %w", err)
	}

	log.Printf("[RDS-NATS] Received ExposeDatabase response: %s", string(msg.Data))

	var resp exposeRDSResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return 0, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !resp.Success {
		return 0, fmt.Errorf("expose database failed: %s", resp.Error)
	}

	return resp.PublicPort, nil
}

// UnexposeDatabase sends a request to the Network Service to remove NAT rules and release the public port.
func (p *NATSPublisher) UnexposeDatabase(resourceID string) error {
	correlationID := uuid.New().String()
	req := unexposeRDSRequest{
		CorrelationID: correlationID,
		ResourceID:    resourceID,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	log.Printf("[RDS-NATS] Sending UnexposeDatabase request: %s", string(reqData))

	msg, err := p.nc.Request(fmt.Sprintf("%s.network.rds.unexpose", p.prefix), reqData, 5*time.Second)
	if err != nil {
		return fmt.Errorf("NATS request failed: %w", err)
	}

	log.Printf("[RDS-NATS] Received UnexposeDatabase response: %s", string(msg.Data))

	var resp unexposeRDSResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("unexpose database failed: %s", resp.Error)
	}

	return nil
}

// ListVPCs requests the list of VPCs for a tenant from the Network Service via NATS.
func (p *NATSPublisher) ListVPCs(tenantID string) ([]domain.VPC, error) {
	correlationID := uuid.New().String()
	req := listVPCsRequest{
		CorrelationID: correlationID,
		TenantID:      tenantID,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	log.Printf("[RDS-NATS] Sending ListVPCs request for tenant %s", tenantID)

	msg, err := p.nc.Request(fmt.Sprintf("%s.network.vpc.list", p.prefix), reqData, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("NATS request failed: %w", err)
	}

	var resp listVPCsResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("failed to list VPCs: %s", resp.Error)
	}

	return resp.VPCs, nil
}

// CreateVPC dispatches an asynchronous NATS message to create a new VPC.
func (p *NATSPublisher) CreateVPC(tenantID, vpcName, requestedBy string) error {
	correlationID := uuid.New().String()
	event := createVPCEvent{
		CorrelationID: correlationID,
		TenantID:      tenantID,
		VPCName:       vpcName,
		RequestedBy:   requestedBy,
	}

	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal create VPC event: %w", err)
	}

	log.Printf("[RDS-NATS] Publishing CreateVPC event for tenant %s, vpc %s", tenantID, vpcName)
	if err := p.nc.Publish(fmt.Sprintf("%s.network.vpc.create", p.prefix), eventData); err != nil {
		return fmt.Errorf("failed to publish create VPC event: %w", err)
	}

	return nil
}

// ReconcileVPCs triggers a global VPC reconciliation in the Network Service.
func (p *NATSPublisher) ReconcileVPCs() error {
	correlationID := uuid.New().String()
	req := reconcileVPCsRequest{
		CorrelationID: correlationID,
	}

	reqData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal reconcile request: %w", err)
	}

	log.Printf("[RDS-NATS] Requesting global VPC reconciliation")
	msg, err := p.nc.Request(fmt.Sprintf("%s.network.vpc.reconcile", p.prefix), reqData, 30*time.Second) // Long timeout for reconciliation
	if err != nil {
		return fmt.Errorf("NATS request failed: %w", err)
	}

	var resp reconcileVPCsResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return fmt.Errorf("failed to unmarshal reconcile response: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("VPC reconciliation failed: %s", resp.Error)
	}

	log.Printf("[RDS-NATS] VPC reconciliation completed: %s", resp.Message)
	return nil
}

// RequestInstanceToken asks the IAM service for a scoped JWT token for the metrics agent.
func (p *NATSPublisher) RequestInstanceToken(userID, instanceID string) (string, error) {
	correlationID := uuid.New().String()
	subject := fmt.Sprintf("%s.iam.token.generate", p.prefix)

	req := instanceTokenRequest{
		InstanceID: instanceID,
		UserID:     userID,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal instance token request: %w", err)
	}

	log.Printf("[RDS-NATS] [REQUEST] subject=%s correlation_id=%s user_id=%s instance_id=%s",
		subject, correlationID, userID, instanceID)

	msg, err := p.nc.Request(subject, data, 5*time.Second)
	if err != nil {
		log.Printf("[RDS-NATS] [ERROR] RequestInstanceToken failed: correlation_id=%s error=%v", correlationID, err)
		return "", fmt.Errorf("NATS request failed: %w", err)
	}

	var resp instanceTokenResponse
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return "", fmt.Errorf("failed to unmarshal instance token response: %w", err)
	}

	if resp.Error != "" {
		log.Printf("[RDS-NATS] [FAILURE] RequestInstanceToken: correlation_id=%s error=%s", correlationID, resp.Error)
		return "", fmt.Errorf("IAM service error: %s", resp.Error)
	}

	if resp.Token == "" {
		log.Printf("[RDS-NATS] [FAILURE] RequestInstanceToken: correlation_id=%s error=empty_token", correlationID)
		return "", fmt.Errorf("IAM service returned an empty token")
	}

	log.Printf("[RDS-NATS] [SUCCESS] Instance token received: correlation_id=%s instance_id=%s", correlationID, instanceID)
	return resp.Token, nil
}

// PublishScalingPolicy publishes a scaling policy creation event to the metrics service.
func (p *NATSPublisher) PublishScalingPolicy(tenantID string, policy domain.ScalingPolicyRequest) error {
	correlationID := uuid.New().String()
	subject := fmt.Sprintf("%s.metrics.scaling_policy.create", p.prefix)

	event := map[string]interface{}{
		"correlation_id": correlationID,
		"tenant_id":      tenantID,
		"policy":         policy,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal scaling policy event: %w", err)
	}

	log.Printf("[RDS-NATS] [REQUEST] subject=%s correlation_id=%s tenant_id=%s target_id=%s",
		subject, correlationID, tenantID, policy.TargetID)

	if err := p.nc.Publish(subject, data); err != nil {
		log.Printf("[RDS-NATS] [ERROR] Failed to publish scaling policy event: correlation_id=%s error=%v", correlationID, err)
		return fmt.Errorf("failed to publish scaling policy event: %w", err)
	}

	log.Printf("[RDS-NATS] [SUCCESS] Published scaling policy event: correlation_id=%s target_id=%s", correlationID, policy.TargetID)
	return nil
}

// GetScalingPolicies requests the scaling policies for a tenant from the metrics service.
func (p *NATSPublisher) GetScalingPolicies(tenantID string) ([]domain.ScalingPolicy, error) {
	correlationID := uuid.New().String()
	subject := fmt.Sprintf("%s.metrics.scaling_policy.list", p.prefix)

	req := map[string]string{
		"correlation_id": correlationID,
		"tenant_id":      tenantID,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal get policies request: %w", err)
	}

	log.Printf("[RDS-NATS] [REQUEST] subject=%s correlation_id=%s tenant_id=%s", subject, correlationID, tenantID)

	msg, err := p.nc.Request(subject, data, 5*time.Second)
	if err != nil {
		log.Printf("[RDS-NATS] [ERROR] GetScalingPolicies failed: correlation_id=%s error=%v", correlationID, err)
		return nil, fmt.Errorf("NATS request failed: %w", err)
	}

	var response struct {
		Policies []domain.ScalingPolicy `json:"policies"`
		Error    string                 `json:"error"`
	}
	if err := json.Unmarshal(msg.Data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policies response: %w", err)
	}

	if response.Error != "" {
		log.Printf("[RDS-NATS] [FAILURE] GetScalingPolicies: correlation_id=%s error=%s", correlationID, response.Error)
		return nil, fmt.Errorf("metrics service error: %s", response.Error)
	}

	log.Printf("[RDS-NATS] [SUCCESS] Retrieved %d policies: correlation_id=%s", len(response.Policies), correlationID)
	return response.Policies, nil
}

// UpdateScalingPolicy publishes an update event for a scaling policy.
func (p *NATSPublisher) UpdateScalingPolicy(tenantID, policyID string, req domain.UpdateScalingPolicyRequest) error {
	correlationID := uuid.New().String()
	subject := fmt.Sprintf("%s.metrics.scaling_policy.update", p.prefix)

	event := map[string]interface{}{
		"correlation_id": correlationID,
		"tenant_id":      tenantID,
		"policy_id":      policyID,
		"update":         req,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal update policy event: %w", err)
	}

	log.Printf("[RDS-NATS] [REQUEST] subject=%s correlation_id=%s tenant_id=%s policy_id=%s",
		subject, correlationID, tenantID, policyID)

	if err := p.nc.Publish(subject, data); err != nil {
		log.Printf("[RDS-NATS] [ERROR] Failed to publish update policy event: correlation_id=%s error=%v", correlationID, err)
		return fmt.Errorf("failed to publish update scaling policy event: %w", err)
	}

	log.Printf("[RDS-NATS] [SUCCESS] Published update policy event: correlation_id=%s policy_id=%s", correlationID, policyID)
	return nil
}

// ProvisionInstanceEvent is the payload sent to the EC2 service to provision a new VM.
// When Profile is "rds", the EC2 service will boot a VM pre-loaded with PostgreSQL
// and inject the Manifest data via cloud-init.
type ProvisionInstanceEvent struct {
	Profile    string                 `json:"profile"`
	Specs      map[string]int         `json:"specs"`
	UserID     string                 `json:"user_id"`
	ResourceID string                 `json:"resource_id"`
	StorageARN string                 `json:"storage_arn"`
	Manifest   map[string]interface{} `json:"manifest"`
	SessionID  string                 `json:"session_id"`
}

type m struct {
	Payload    string `json:"payload"`
	UserId     string `json:"userId"`
	InstanceID string `json:"instanceId"`
}

// ProvisionRDSInstance publishes a VM provisioning event to the EC2 service.
// This is a fire-and-forget publish: the EC2 service handles scheduling,
// host selection, and cloud-init injection. The SessionID (= database ID) is
// used to correlate the async callback when the VM becomes ready.



func (p *NATSPublisher) ProvisionRDSInstance(event domain.ProvisionInstanceEvent) (domain.EC2Response,error) {
	correlationID := uuid.New().String()
	// subject := fmt.Sprintf("%s.iam.token.generate",p.prefix)
	subject := fmt.Sprintf("%s.ec2.task.provision", p.prefix)

	data, err := json.Marshal(event)

	if err != nil {
		return domain.EC2Response{}, fmt.Errorf("failed to marshal ProvisionInstanceEvent: %w", err)
	}

	log.Printf("[RDS-NATS] [PROVISION] subject=%s correlation_id=%s session_id=%s profile=%s user_id=%s",
		subject, correlationID, event.SessionID, event.Profile, event.UserID)

	// if err := p.nc.Publish(subject, data); err != nil {
	reply, err := p.nc.Request(subject, data, time.Minute*3)
	if err != nil {
		log.Printf("[RDS-NATS] [ERROR] ProvisionRDSInstance failed: correlation_id=%s error=%v", correlationID, err)
		return domain.EC2Response{},fmt.Errorf("failed to publish ProvisionInstanceEvent: %w", err)
	}

	var ec2Response domain.EC2Response

	errr := json.Unmarshal(reply.Data, &ec2Response)
	if errr != nil {
		log.Printf("[RDS-NATS] [ERROR] ProvisionRDSInstance failed tounmarshalthe reply: correlation_id=%s error=%v", correlationID, err)
		return domain.EC2Response{}, fmt.Errorf("failed to publish ProvisionInstanceEvent: %w", err)
	}

	log.Printf("[RDS-NATS] [SUCCESS] Got a replyt: correlation_id=%s resource=%v", ec2Response.GatewayIP, ec2Response.GatewayPort)
	return ec2Response, nil
}

// DeleteScalingPolicy publishes a delete event for a scaling policy.
func (p *NATSPublisher) DeleteScalingPolicy(tenantID, policyID string) error {
	correlationID := uuid.New().String()
	subject := fmt.Sprintf("%s.metrics.scaling_policy.delete", p.prefix)

	event := map[string]interface{}{
		"correlation_id": correlationID,
		"tenant_id":      tenantID,
		"policy_id":      policyID,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal delete policy event: %w", err)
	}

	log.Printf("[RDS-NATS] [REQUEST] subject=%s correlation_id=%s tenant_id=%s policy_id=%s",
		subject, correlationID, tenantID, policyID)

	if err := p.nc.Publish(subject, data); err != nil {
		log.Printf("[RDS-NATS] [ERROR] Failed to publish delete policy event: correlation_id=%s error=%v", correlationID, err)
		return fmt.Errorf("failed to publish delete scaling policy event: %w", err)
	}

	log.Printf("[RDS-NATS] [SUCCESS] Published delete policy event: correlation_id=%s policy_id=%s", correlationID, policyID)
	return nil
}
