package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"rds/internal/infrastructure/event"
	"rds/internal/interfaces"
	"sort"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

type ScalingAlarmPayload struct {
	InstanceID   string `json:"instance_id"`
	Action       string `json:"action"`        // SCALE_OUT or SCALE_IN
	ResourceType string `json:"resource_type"` // CPU or MEMORY
	NewLimit     int64  `json:"new_limit"`
	TenantID     string `json:"tenant_id"`
	Reason       string `json:"reason"`
}

type ScalingService struct {
	repo         interfaces.RepositoryPort
	dockerClient interfaces.DockerPort
	nats         *event.NATSAdapter
	natsPrefix   string
	stopChan     chan struct{}

	memoryTiers []int64 // bytes
	cpuTiers    []int64 // CPU shares
}

func NewScalingService(repo interfaces.RepositoryPort, dockerClient interfaces.DockerPort, nats *event.NATSAdapter, natsPrefix string) *ScalingService {
	return &ScalingService{
		repo:         repo,
		dockerClient: dockerClient,
		nats:         nats,
		natsPrefix:   natsPrefix,
		stopChan:     make(chan struct{}),
		// Strategy: Discrete tiers for stability
		memoryTiers: []int64{
			256 * 1024 * 1024,
			512 * 1024 * 1024,
			1024 * 1024 * 1024,
			2048 * 1024 * 1024,
			4096 * 1024 * 1024,
		},
		cpuTiers: []int64{
			256,
			512,
			1024,
			2048,
		},
	}
}

func (s *ScalingService) Start() {
	log.Println("[SCALING] Starting Vertical Scaling Service...")

	if s.nats != nil {
		s.nats.Subscribe(fmt.Sprintf("%s.rds.scale.out", s.natsPrefix), s.handleScalingMessage)
		s.nats.Subscribe(fmt.Sprintf("%s.rds.scale.in", s.natsPrefix), s.handleScalingMessage)
	}

}

func (s *ScalingService) handleScalingMessage(m *nats.Msg) {
	var payload ScalingAlarmPayload
	if err := json.Unmarshal(m.Data, &payload); err != nil {
		log.Printf("[SCALING] Error unmarshaling payload: %v", err)
		return
	}

	log.Printf("[SCALING] Received %s trigger for %s (%s): %s",
		payload.Action, payload.InstanceID, payload.ResourceType, payload.Reason)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// 1. Resolve container name
	containerName := fmt.Sprintf("claudedb-prod-%s", payload.InstanceID)

	// 2. Get current limits
	info, err := s.dockerClient.GetContainerInfo(ctx, containerName)
	if err != nil {
		log.Printf("[SCALING] Failed to info container %s: %v", containerName, err)
		return
	}

	var newCPUShares = info.CPUShares
	var newMemoryBytes = info.Memory

	// 3. Determine next tier
	if strings.ToUpper(payload.ResourceType) == "MEMORY" {
		newMemoryBytes = s.calculateNextTier(info.Memory, payload.Action, s.memoryTiers)
		if newMemoryBytes == info.Memory {
			log.Printf("[SCALING] Already at limit tier for MEMORY (%d bytes)", info.Memory)
			return
		}
	} else if strings.ToUpper(payload.ResourceType) == "CPU" {
		newCPUShares = s.calculateNextTier(info.CPUShares, payload.Action, s.cpuTiers)
		if newCPUShares == info.CPUShares {
			log.Printf("[SCALING] Already at limit tier for CPU (%d shares)", info.CPUShares)
			return
		}
	} else {
		log.Printf("[SCALING] Unknown resource type: %s", payload.ResourceType)
		return
	}

	// 4. Execute on-the-fly update
	err = s.dockerClient.UpdateContainerResources(ctx, containerName, newCPUShares, newMemoryBytes)
	if err != nil {
		log.Printf("[SCALING] FAILED to scale %s: %v", containerName, err)
		return
	}

	log.Printf("[SCALING] SUCCESS Scall %s: CPU %d -> %d | MEM %d -> %d",
		payload.Action, info.CPUShares, newCPUShares, info.Memory, newMemoryBytes)
}

func (s *ScalingService) calculateNextTier(current int64, action string, tiers []int64) int64 {
	sort.Slice(tiers, func(i, j int) bool { return tiers[i] < tiers[j] })

	if action == "SCALE_OUT" {
		for _, tier := range tiers {
			if tier > current {
				return tier
			}
		}
		return current // Already at max tier
	}

	if action == "SCALE_IN" {
		for i := len(tiers) - 1; i >= 0; i-- {
			if tiers[i] < current {
				return tiers[i]
			}
		}
		return current // Already at min tier
	}

	return current
}

func (s *ScalingService) Stop() {
	// Subscriptions are automatically closed when the connection is closed,
	// but we could explicitly unsubscribe if needed.
	close(s.stopChan)
}
