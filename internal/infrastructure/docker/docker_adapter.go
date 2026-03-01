package docker

import (
	"context"
	"fmt"
	"io"
	"rds/internal/domain"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// DockerAdapter implements domain.DockerPort
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
	// Prepare environment variables
	env := []string{
		fmt.Sprintf("POSTGRES_USER=%s", cfg.User),
		fmt.Sprintf("POSTGRES_PASSWORD=%s", cfg.Password),
	}
	for key, value := range cfg.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Prepare labels
	labels := map[string]string{
		"ownerId": cfg.OwnerID,
		"type":    "mini-rds",
	}
	for key, value := range cfg.Labels {
		labels[key] = value
	}

	// Container port configuration
	containerPort := "5432/tcp"
	containerConfig := &container.Config{
		Image: cfg.Image,
		Env:   env,
		ExposedPorts: nat.PortSet{
			nat.Port(containerPort): struct{}{},
		},
		Labels: labels,
	}

	// Host port configuration
	hostConfig := &container.HostConfig{
		PortBindings: nat.PortMap{
			nat.Port(containerPort): []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: fmt.Sprintf("%d", cfg.Port),
				},
			},
		},
	}

	if cfg.VolumeSource != "" && cfg.VolumeDest != "" {
		hostConfig.Binds = []string{
			fmt.Sprintf("%s:%s", cfg.VolumeSource, cfg.VolumeDest),
		}
	}

	// Create the container
	resp, err := d.client.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, cfg.Name)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	return resp.ID, nil
}

// StartContainer starts a Docker container
func (d *DockerAdapter) StartContainer(ctx context.Context, containerID string) error {
	if err := d.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %s: %w", containerID, err)
	}
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
		ID:     inspect.ID,
		Status: inspect.State.Status,
		Port:   port,
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

// Close closes the Docker client connection
func (d *DockerAdapter) Close() error {
	return d.client.Close()
}
