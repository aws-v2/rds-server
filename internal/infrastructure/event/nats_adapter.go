package event

import (
	"fmt"

	"github.com/nats-io/nats.go"
)

// NATSAdapter wraps the NATS connection for messaging
type NATSAdapter struct {
	conn *nats.Conn
}

// NewNATSAdapter creates a new NATS adapter
func NewNATSAdapter(url string) (*NATSAdapter, error) {
	conn, err := nats.Connect(url,
		nats.MaxReconnects(-1), // Unlimited reconnects
		nats.ReconnectWait(nats.DefaultReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				fmt.Printf("NATS disconnected: %v\n", err)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			fmt.Printf("NATS reconnected to %s\n", nc.ConnectedUrl())
		}),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &NATSAdapter{
		conn: conn,
	}, nil
}

// GetConnection returns the underlying NATS connection
func (n *NATSAdapter) GetConnection() *nats.Conn {
	return n.conn
}

// Publish publishes a message to a subject
func (n *NATSAdapter) Publish(subject string, data []byte) error {
	return n.conn.Publish(subject, data)
}

// Subscribe subscribes to a subject
func (n *NATSAdapter) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	return n.conn.Subscribe(subject, handler)
}

// Request sends a request and waits for a response
func (n *NATSAdapter) Request(subject string, data []byte) (*nats.Msg, error) {
	return n.conn.Request(subject, data, nats.DefaultTimeout)
}

// Close closes the NATS connection
func (n *NATSAdapter) Close() {
	if n.conn != nil {
		n.conn.Close()
	}
}

// IsConnected returns true if the NATS connection is active
func (n *NATSAdapter) IsConnected() bool {
	return n.conn != nil && n.conn.IsConnected()
}
