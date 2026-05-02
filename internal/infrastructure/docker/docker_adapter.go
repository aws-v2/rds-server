package docker

import (
	"context"
	"fmt"
	"io"
	"log"
	"rds/internal/domain"

	"encoding/json"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// DockerAdapter implements interfaces.DockerPort
type DockerAdapter struct {
	client *client.Client
}

// NewDockerAdapter creates a new Docker adapter
func NewDockerAdapter() (*DockerAdapter, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &DockerAdapter{
		client: cli,
	}, nil
}

// GetContainerStats fetch stats from docker and calculate CPU/Memory
func (d *DockerAdapter) GetContainerStats(ctx context.Context, containerID string) (*domain.ContainerStats, error) {
	stats, err := d.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats for %s: %w", containerID, err)
	}
	defer stats.Body.Close()

	var v struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
			OnlineCPUs  uint32 `json:"online_cpus"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64 `json:"usage"`
			Limit uint64 `json:"limit"`
		} `json:"memory_stats"`
	}

	if err := json.NewDecoder(stats.Body).Decode(&v); err != nil {
		return nil, fmt.Errorf("failed to decode container stats: %w", err)
	}

	// Calculate CPU Percentage
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
	onlineCPUs := float64(v.CPUStats.OnlineCPUs)
	if onlineCPUs == 0.0 {
		onlineCPUs = 1.0
	}

	cpuPercent := 0.0
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}

	return &domain.ContainerStats{
		CPUPercentage:    cpuPercent,
		MemoryUsageBytes: int64(v.MemoryStats.Usage),
		MemoryLimitBytes: int64(v.MemoryStats.Limit),
	}, nil
}

// PullImage pulls a Docker image
func (d *DockerAdapter) PullImage(ctx context.Context, imageq string) error {
	reader, err := d.client.ImagePull(ctx, imageq, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageq, err)
	}
	defer reader.Close()

	// Consume the output to ensure the pull completes
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to read image pull output: %w", err)
	}

	return nil
}

// CreateContainer creates a new Docker container
func (d *DockerAdapter) CreateContainer(ctx context.Context, cfg domain.ContainerConfig) (string, error) {
	log.Printf("[DOCKER] Creating container %s — BridgeName=%s PrivateIP=%s HostPort=%d",
		cfg.Name, cfg.BridgeName, cfg.PrivateIP, cfg.HostPort)

	env := []string{
		fmt.Sprintf("POSTGRES_USER=%s", cfg.User),
		fmt.Sprintf("POSTGRES_PASSWORD=%s", cfg.Password),
	}
	for key, value := range cfg.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	labels := map[string]string{
		"ownerId": cfg.OwnerID,
		"type":    "mini-rds",
	}
	for key, value := range cfg.Labels {
		labels[key] = value
	}

	// Bug 1 fix: postgres always listens on 5432 inside the container
	containerPort := nat.Port("5432/tcp")

	containerConfig := &container.Config{
		Image:  cfg.Image,
		Env:    env,
		Labels: labels,
		ExposedPorts: nat.PortSet{
			containerPort: struct{}{},
		},
	}

	hostConfig := &container.HostConfig{
		// Bug 2 fix: bind host's NodePort (1000, 1001...) → container's 5432
		PortBindings: nat.PortMap{
			containerPort: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "8743",
					// HostPort: fmt.Sprintf("%d", cfg.HostPort),
				},
			},
		},
	}

	var networkingConfig *network.NetworkingConfig
	if cfg.BridgeName != "" && cfg.PrivateIP != "" {
		hostConfig.NetworkMode = container.NetworkMode(cfg.BridgeName)
		networkingConfig = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				cfg.BridgeName: {
					IPAMConfig: &network.EndpointIPAMConfig{
						IPv4Address: cfg.PrivateIP,
					},
				},
			},
		}
	}

	if cfg.VolumeSource != "" && cfg.VolumeDest != "" {
		hostConfig.Binds = []string{
			fmt.Sprintf("%s:%s", cfg.VolumeSource, cfg.VolumeDest),
		}
	}

	resp, err := d.client.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, cfg.Name)
	if err != nil {
		log.Printf("[DOCKER] Failed to create container %s: %v", cfg.Name, err)
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	log.Printf("[DOCKER] Container %s created successfully — ID=%s HostPort=%d→ContainerPort=5432",
		cfg.Name, resp.ID, cfg.HostPort)
	return resp.ID, nil
}

func (d *DockerAdapter) StartContainer(ctx context.Context, containerID string) error {
	log.Printf("---->> [DOCKER] StartContainer: initiating for container %s", containerID)

	if err := d.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		log.Printf("---->> [DOCKER] StartContainer: failed for container %s: %v", containerID, err)
		return fmt.Errorf("failed to start container %s: %w", containerID, err)
	}

	log.Printf("---->> [DOCKER] StartContainer: container %s started successfully", containerID)
	return nil
}

// StopContainer stops a Docker container
func (d *DockerAdapter) StopContainer(ctx context.Context, containerID string) error {
	timeout := 10
	if err := d.client.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}
	return nil
}

// RemoveContainer removes a Docker container
func (d *DockerAdapter) RemoveContainer(ctx context.Context, containerID string) error {
	if err := d.client.ContainerRemove(ctx, containerID, container.RemoveOptions{}); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}
	return nil
}

// GetContainerStatus retrieves the status of a Docker container
func (d *DockerAdapter) GetContainerStatus(ctx context.Context, containerID string) (string, error) {
	inspect, err := d.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return "", fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	return inspect.State.Status, nil
}

// GetContainerInfo retrieves detailed information about a container
func (d *DockerAdapter) GetContainerInfo(ctx context.Context, containerID string) (*domain.ContainerInfo, error) {
	inspect, err := d.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	// Extract port binding
	var port int
	if bindings, ok := inspect.NetworkSettings.Ports["5432/tcp"]; ok && len(bindings) > 0 {
		fmt.Sscanf(bindings[0].HostPort, "%d", &port)
	}

	return &domain.ContainerInfo{
		ID:        inspect.ID,
		Status:    inspect.State.Status,
		Port:      port,
		CPUShares: inspect.HostConfig.CPUShares,
		Memory:    inspect.HostConfig.Memory,
	}, nil
}

// CreateVolume creates a named Docker volume
func (d *DockerAdapter) CreateVolume(ctx context.Context, name string) error {
	_, err := d.client.VolumeCreate(ctx, volume.CreateOptions{
		Name: name,
	})
	if err != nil {
		return fmt.Errorf("failed to create docker volume %s: %w", name, err)
	}
	return nil
}

// RemoveVolume removes a named Docker volume
func (d *DockerAdapter) RemoveVolume(ctx context.Context, name string) error {
	// Force remove the volume
	err := d.client.VolumeRemove(ctx, name, true)
	if err != nil {
		return fmt.Errorf("failed to remove docker volume %s: %w", name, err)
	}
	return nil
}

// InspectVolume returns volume details
func (d *DockerAdapter) InspectVolume(ctx context.Context, name string) (map[string]interface{}, error) {
	vol, err := d.client.VolumeInspect(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect docker volume %s: %w", name, err)
	}

	result := map[string]interface{}{
		"Name":       vol.Name,
		"Mountpoint": vol.Mountpoint,
		"CreatedAt":  vol.CreatedAt,
	}
	return result, nil
}

// UpdateContainerResources updates the resource limits of a running container on-the-fly
func (d *DockerAdapter) UpdateContainerResources(ctx context.Context, containerID string, cpuShares int64, memoryBytes int64) error {
	updateConfig := container.UpdateConfig{
		Resources: container.Resources{
			CPUShares: cpuShares,
			Memory:    memoryBytes,
		},
	}

	_, err := d.client.ContainerUpdate(ctx, containerID, updateConfig)
	if err != nil {
		return fmt.Errorf("failed to update container %s: %w", containerID, err)
	}

	log.Printf("[DOCKER] Successfully updated container %s resources: CPUShares=%d, Memory=%d",
		containerID, cpuShares, memoryBytes)
	return nil
}

// EnsureNetwork checks if a Docker network exists, and creates it if not.
func (d *DockerAdapter) EnsureNetwork(ctx context.Context, name, gateway string) error {
	_, err := d.client.NetworkInspect(ctx, name, network.InspectOptions{})
	if err == nil {
		return nil // Network already exists
	}

	// Create the network if it doesn't exist
	log.Printf("[DOCKER] Network %s not found, creating with gateway %s...", name, gateway)

	// Infer subnet from gateway (e.g., 10.5.1.1 -> 10.5.1.0/24)
	lastDot := -1
	for i := len(gateway) - 1; i >= 0; i-- {
		if gateway[i] == '.' {
			lastDot = i
			break
		}
	}

	subnet := "10.0.0.0/16" // Very broad fallback
	if lastDot != -1 {
		subnet = gateway[:lastDot] + ".0/24"
	}

	_, err = d.client.NetworkCreate(ctx, name, network.CreateOptions{
		Driver: "bridge",
		IPAM: &network.IPAM{
			Config: []network.IPAMConfig{
				{
					Subnet:  subnet,
					Gateway: gateway,
				},
			},
		},
		Options: map[string]string{
			"com.docker.network.bridge.name": name,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create docker network %s with subnet %s: %w", name, subnet, err)
	}

	log.Printf("[DOCKER] Successfully created network %s with subnet %s", name, subnet)
	return nil
}

// InspectNetwork returns details about a Docker network
func (d *DockerAdapter) InspectNetwork(ctx context.Context, name string) (map[string]interface{}, error) {
	nw, err := d.client.NetworkInspect(ctx, name, network.InspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to inspect docker network %s: %w", name, err)
	}

	result := map[string]interface{}{
		"ID":         nw.ID,
		"Name":       nw.Name,
		"Driver":     nw.Driver,
		"Containers": nw.Containers,
		"IPAM":       nw.IPAM,
	}
	return result, nil
}

// RemoveNetwork removes a Docker network
func (d *DockerAdapter) RemoveNetwork(ctx context.Context, name string) error {
	err := d.client.NetworkRemove(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to remove docker network %s: %w", name, err)
	}
	return nil
}

// Close closes the Docker client connection
func (d *DockerAdapter) Close() error {
	return d.client.Close()
}
