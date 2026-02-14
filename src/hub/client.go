package hub

import (
	"sync"
	"time"

	"github.com/orchestra-mcp/socket/src/types"
)

// Client wraps a WebSocket connection and manages message flow.
type Client struct {
	ID          string
	conn        types.Conn
	hub         *Hub
	Send        chan types.Message
	connectedAt time.Time
	channels    map[string]bool
	mu          sync.RWMutex
	done        chan struct{}
	closed      bool
}

// NewClient creates a new WebSocket client wrapper.
func NewClient(id string, conn types.Conn, h *Hub) *Client {
	return &Client{
		ID:          id,
		conn:        conn,
		hub:         h,
		Send:        make(chan types.Message, 256),
		connectedAt: time.Now(),
		channels:    make(map[string]bool),
		done:        make(chan struct{}),
	}
}

// Info returns metadata about this client.
func (c *Client) Info() types.ClientInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	channels := make([]string, 0, len(c.channels))
	for ch := range c.channels {
		channels = append(channels, ch)
	}
	return types.ClientInfo{
		ID:          c.ID,
		ConnectedAt: c.connectedAt,
		Channels:    channels,
	}
}

// AddChannel adds a channel subscription.
func (c *Client) AddChannel(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.channels[channel] = true
}

// RemoveChannel removes a channel subscription.
func (c *Client) RemoveChannel(channel string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.channels, channel)
}

// ReadPump reads messages from the WebSocket and routes to the hub.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		var msg types.Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			return
		}
		msg.ClientID = c.ID
		msg.Timestamp = time.Now()
		c.hub.incoming <- msg
	}
}

// WritePump writes messages from the send channel to the WebSocket.
func (c *Client) WritePump() {
	defer c.conn.Close()

	for {
		select {
		case msg, ok := <-c.Send:
			if !ok {
				return
			}
			if err := c.conn.WriteJSON(msg); err != nil {
				return
			}
		case <-c.done:
			return
		}
	}
}

// Close signals the client to stop its pumps.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.done)
		close(c.Send)
	}
}
