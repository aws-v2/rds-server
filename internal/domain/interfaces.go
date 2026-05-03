package domain

// ContainerConfig represents the configuration for creating a Docker container
type ContainerConfig struct {
	Name         string
	Image        string
   HostPort      int    // externally bound port on the host (1000, 1001, ...)
    ContainerPort int    // internal port inside the container (always 5432 for postgres)
    // Remove or repurpose the old Port field
	User         string
	Password     string
	OwnerID      string
	Environment  map[string]string
	Labels       map[string]string
	VolumeSource string
	VolumeDest   string
	PrivateIP    string // e.g. "10.1.2.5" — allocated by network service
	BridgeName   string // e.g. "br-vpc-19b72a7e" — tenant's VPC bridge
	Gateway      string // e.g. "10.1.2.1"
}

// ContainerInfo represents information about a running container
type ContainerInfo struct {
	ID        string
	Status    string
	Port      int
	CPUShares int64
	Memory    int64
}

// ContainerStats represents the resource usage statistics of a container
type ContainerStats struct {
	CPUPercentage    float64
	MemoryUsageBytes int64
	MemoryLimitBytes int64
}
