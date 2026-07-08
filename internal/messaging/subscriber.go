package messaging

//
// the rds should be listeningto .lifecycle for a vm
// so that it can get the

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

// EC2EventHandler is the interface the subscriber expects to call when events occur.
type EC2EventHandler interface {
	HandleProvisionLifeCycle(ctx context.Context, event *InstanceLifecycleEventSub) error
}

type NATSSubscriber struct {
	nc      *nats.Conn
	profile string
	handler EC2EventHandler
}


func NewNATSSubscriber(url, user, password string, profile string, handler EC2EventHandler) (*NATSSubscriber, error) {
	opts := []nats.Option{
		nats.Name("EC2-Subscriber"),
	}
	if user != "" && password != "" {
		opts = append(opts, nats.UserInfo(user, password))
	}

	nc, err := nats.Connect(url, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS for subscriber: %w", err)
	}

	return &NATSSubscriber{
		nc:      nc,
		profile: profile,
		handler: handler,
	}, nil
}
type RDSMetadata struct {
	HostIp string `json:"host_ip"`
	InstancePrivateIp string `json:"instance_private_ip"`
	InstancePublicIp string `json:"instance_public_ip"`

	ResourceID    string            `json:"resource_id"`

}


type InstanceLifecyclePayload struct {
	IPAddress   string            `json:"ip_address"`
	VPCID       string            `json:"vpc_id"`
	ServicePort int               `json:"service_port"`
	Metadata    InstanceMetadata `json:"metadata"`
	InstanceStartedMetadata RDSMetadata`json:"instance_started_metadata"`

}

type InstanceLifecycleEvent struct {
	CorrelationID string                  `json:"correlation_id"`
	InstanceID    string                  `json:"instance_id"`
	EventType     string                  `json:"event_type"`
	Timestamp     string                  `json:"timestamp"`
	Payload       InstanceLifecyclePayload `json:"payload"`
	AgentURL    string           `json:"agent_url,omitempty"`
	SessionID string `json:"session_id"`

}


type InstanceMetadata struct {
	InstanceType string `json:"instance_type"`
	AMIID        string `json:"ami_id"`
}
 type InstanceLifecycleEventSub struct {
	CorrelationID string                  `json:"correlation_id"`
	ResourceID    string            `json:"resource_id"`
	InstanceID    string                  `json:"instance_id"`
	EventType     string                  `json:"event_type"`
	Stage     string                  `json:"stage"`
	Timestamp     string                  `json:"timestamp"`
	Payload       InstanceLifecyclePayload `json:"payload"`
	AgentURL    string           `json:"agent_url,omitempty"`
	SessionID string `json:"session_id"`

}


func (s *NATSSubscriber) Start() error {
	queueGroup := "rds-enforcers"


	// 2. Subscribe to VM Provision Events
	provisionSubject := fmt.Sprintf("%s.ec2.instance.lifecycle", s.profile)
	_, err := s.nc.QueueSubscribe(provisionSubject, queueGroup, func(msg *nats.Msg) {
		var event InstanceLifecycleEventSub
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("[NATS-SUB] [ERROR] Failed to unmarshal provision event: %v", err)
			return
		}
		

		log.Printf("[NATS-SUB] [INFO] Received provision event for corelation: %s", event.CorrelationID)

		go func() {
			if err := s.handler.HandleProvisionLifeCycle(context.Background(), &event); err != nil {
				log.Printf("[NATS-SUB] [ERROR] VM provision failed for profile %s: %v", event.EventType, err)
			}
		}()
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", provisionSubject, err)
	}

	log.Printf("[NATS-SUB] Successfully subscribed to %s and %s", provisionSubject)
	return nil
}

func (s *NATSSubscriber) Close() {
	if s.nc != nil {
		s.nc.Close()
	}
}
