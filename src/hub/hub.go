package hub

import (
	"sync"

	"github.com/orchestra-mcp/socket/src/types"
	"github.com/rs/zerolog"
)

// MessageBridge publishes messages to other server instances.
// Defined here to avoid circular imports with the bridge package.
type MessageBridge interface {
	Publish(msg types.Message) error
	Available() bool
}

// Hub manages all WebSocket client connections and channel subscriptions.
type Hub struct {
	clients  map[string]*Client
	channels map[string]map[string]bool // channel -> set of clientIDs

	register   chan *Client
	unregister chan *Client
	incoming   chan types.Message
	broadcast  chan broadcastMsg
	localCast  chan broadcastMsg // messages from bridge, no re-publish

	handlers  map[string]types.MessageHandler
	onConnect []func(string)
	onDisconn []func(string)

	bridge MessageBridge
	mu     sync.RWMutex
	logger zerolog.Logger
	done   chan struct{}
}

type broadcastMsg struct {
	channel string
	msg     types.Message
}

// New creates a new Hub instance.
func New(logger zerolog.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		channels:   make(map[string]map[string]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		incoming:   make(chan types.Message, 256),
		broadcast:  make(chan broadcastMsg, 256),
		localCast:  make(chan broadcastMsg, 256),
		handlers:   make(map[string]types.MessageHandler),
		logger:     logger,
		done:       make(chan struct{}),
	}
}

// SetBridge attaches a cross-instance message bridge to the hub.
// When set, published messages are also forwarded to other instances.
func (h *Hub) SetBridge(b MessageBridge) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.bridge = b
}

// BroadcastToLocal delivers a message from the bridge to local subscribers only.
// It does not re-publish to Redis, preventing infinite loops.
func (h *Hub) BroadcastToLocal(msg types.Message) {
	h.localCast <- broadcastMsg{channel: msg.Channel, msg: msg}
}

// Run starts the hub event loop. Call in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.addClient(client)
		case client := <-h.unregister:
			h.removeClient(client)
		case msg := <-h.incoming:
			h.handleMessage(msg)
		case bm := <-h.broadcast:
			h.publishToBridge(bm.msg)
			h.broadcastToChannel(bm.channel, bm.msg)
		case bm := <-h.localCast:
			h.broadcastToChannel(bm.channel, bm.msg)
		case <-h.done:
			return
		}
	}
}

// Stop halts the hub event loop.
func (h *Hub) Stop() {
	close(h.done)
}

// Register queues a client for registration.
func (h *Hub) Register(c *Client) {
	h.register <- c
}

// Unregister queues a client for removal.
func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

func (h *Hub) addClient(c *Client) {
	h.mu.Lock()
	h.clients[c.ID] = c
	h.mu.Unlock()

	h.logger.Info().Str("client_id", c.ID).Msg("client registered")

	for _, cb := range h.onConnect {
		cb(c.ID)
	}
}

func (h *Hub) removeClient(c *Client) {
	h.mu.Lock()
	if _, ok := h.clients[c.ID]; !ok {
		h.mu.Unlock()
		return
	}
	delete(h.clients, c.ID)

	// Remove from all channel subscriptions.
	for ch, subs := range h.channels {
		delete(subs, c.ID)
		if len(subs) == 0 {
			delete(h.channels, ch)
		}
	}
	h.mu.Unlock()

	c.Close()
	h.logger.Info().Str("client_id", c.ID).Msg("client unregistered")

	for _, cb := range h.onDisconn {
		cb(c.ID)
	}
}
