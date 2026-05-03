package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
)

type DBInstance struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Port        int       `json:"port"`
	User        string    `json:"user"`
	Password    string    `json:"password"`
	OwnerID     string    `json:"ownerId"`
	ContainerID string    `json:"containerId"`
	CreatedAt   time.Time `json:"createdAt"`
}

type DockerManager struct {
	client       *client.Client
	usedPorts    map[int]bool
	instances    map[string]*DBInstance
	mu           sync.RWMutex
	instanceFile string
}

func NewDockerManager(instanceFile string) (*DockerManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	dm := &DockerManager{
		client:       cli,
		usedPorts:    make(map[int]bool),
		instances:    make(map[string]*DBInstance),
		instanceFile: instanceFile,
	}

	// Load existing instances from file
	if err := dm.loadInstances(); err != nil {
		return nil, err
	}

	return dm, nil
}

func (m *DockerManager) loadInstances() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, err := os.Stat(m.instanceFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(m.instanceFile)
	if err != nil {
		return fmt.Errorf("failed to read instances file: %w", err)
	}

	var instances []*DBInstance
	if err := json.Unmarshal(data, &instances); err != nil {
		return fmt.Errorf("failed to unmarshal instances: %w", err)
	}

	for _, inst := range instances {
		m.instances[inst.ID] = inst
		m.usedPorts[inst.Port] = true
	}

	return nil
}

func (m *DockerManager) saveInstances() error {
	instanceList := make([]*DBInstance, 0, len(m.instances))
	for _, inst := range m.instances {
		instanceList = append(instanceList, inst)
	}

	data, err := json.MarshalIndent(instanceList, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal instances: %w", err)
	}

	if err := os.WriteFile(m.instanceFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write instances file: %w", err)
	}

	return nil
}

func (m *DockerManager) getNextAvailablePort() int {
	port := 5432
	for {
		if !m.usedPorts[port] {
			m.usedPorts[port] = true
			return port
		}
		port++
	}
}

func (m *DockerManager) CreatePostgres(ctx context.Context, name, user, pass, ownerID string) (*DBInstance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Pull postgres:17 image
	reader, err := m.client.ImagePull(ctx, "docker.io/library/postgres:17", image.PullOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to pull postgres:17 image: %w", err)
	}
	defer reader.Close()
	io.Copy(io.Discard, reader)

	// Get unique port
	hostPort := m.getNextAvailablePort()

	// Container configuration
	containerPort := "5432/tcp"
	config := &container.Config{
		Image: "postgres:17",
		Env: []string{
			fmt.Sprintf("POSTGRES_USER=%s", user),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", pass),
		},
		ExposedPorts: nat.PortSet{
			nat.Port(containerPort): struct{}{},
		},
		Labels: map[string]string{
			"ownerId": ownerID,
			"type":    "mini-rds",
		},
	}

	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			nat.Port(containerPort): []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: fmt.Sprintf("%d", hostPort),
				},
			},
		},
	}

	resp, err := m.client.ContainerCreate(ctx, config, hostConfig, nil, nil, name)
	if err != nil {
		m.usedPorts[hostPort] = false
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := m.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		m.usedPorts[hostPort] = false
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Create instance metadata
	instance := &DBInstance{
		ID:          uuid.New().String(),
		Name:        name,
		Port:        hostPort,
		User:        user,
		Password:    pass,
		OwnerID:     ownerID,
		ContainerID: resp.ID,
		CreatedAt:   time.Now(),
	}

	m.instances[instance.ID] = instance

	// Persist to file
	if err := m.saveInstances(); err != nil {
		return nil, fmt.Errorf("failed to save instances: %w", err)
	}

	return instance, nil
}

func (m *DockerManager) ListInstances(ownerID string) []*DBInstance {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*DBInstance
	for _, inst := range m.instances {
		if inst.OwnerID == ownerID {
			result = append(result, inst)
		}
	}
	return result
}

func (m *DockerManager) DeleteInstance(ctx context.Context, instanceID, ownerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, exists := m.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance not found")
	}

	if instance.OwnerID != ownerID {
		return fmt.Errorf("unauthorized: instance does not belong to user")
	}

	// Stop and remove container
	timeout := 10
	if err := m.client.ContainerStop(ctx, instance.ContainerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to stop container: %w", err)
	}

	if err := m.client.ContainerRemove(ctx, instance.ContainerID, container.RemoveOptions{}); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Free the port
	delete(m.usedPorts, instance.Port)

	// Remove from instances map
	delete(m.instances, instanceID)

	// Persist changes
	if err := m.saveInstances(); err != nil {
		return fmt.Errorf("failed to save instances: %w", err)
	}

	return nil
}
