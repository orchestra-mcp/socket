package bridge

import "github.com/orchestra-mcp/socket/src/types"

// Bridge defines the interface for cross-instance message broadcasting.
// Implementations relay messages between multiple server instances.
type Bridge interface {
	// Publish sends a message to all other instances via the bridge.
	Publish(msg types.Message) error

	// Start begins listening for messages from other instances.
	Start() error

	// Stop shuts down the bridge connection.
	Stop() error

	// Available reports whether the bridge is connected and operational.
	Available() bool
}

// BroadcastTarget is implemented by the Hub to receive messages from the bridge.
type BroadcastTarget interface {
	BroadcastToLocal(msg types.Message)
}
