package docker

import (
	"fmt"
	"net"
	"sync"
)

const startPort = 21000

type PortAllocator struct {
	mu   sync.Mutex
	next int
}

func NewPortAllocator() *PortAllocator {
	return &PortAllocator{next: startPort}
}

// Acquire finds the next available port starting from where we left off
func (p *PortAllocator) Acquire() (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for port := p.next; port < 65535; port++ {
		if isPortFree(port) {
			p.next = port + 1 // next call starts after this one
			return port, nil
		}
	}
	return 0, fmt.Errorf("no free ports available in range %d-65535", startPort)
}

func isPortFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}