package hub

import (
	"github.com/orchestra-mcp/socket/src/types"
)

// RegisterHandler registers a handler for a channel.
func (h *Hub) RegisterHandler(channel string, handler types.MessageHandler) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.handlers[channel] = handler
}

// OnConnection registers a callback for new connections.
func (h *Hub) OnConnection(cb func(string)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onConnect = append(h.onConnect, cb)
}

// OnDisconnection registers a callback for disconnections.
func (h *Hub) OnDisconnection(cb func(string)) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.onDisconn = append(h.onDisconn, cb)
}

// ConnectedClients returns a list of connected client IDs.
func (h *Hub) ConnectedClients() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.clients))
	for id := range h.clients {
		ids = append(ids, id)
	}
	return ids
}

// ClientInfo returns info for a connected client, or nil.
func (h *Hub) ClientInfo(clientID string) *types.ClientInfo {
	h.mu.RLock()
	client, ok := h.clients[clientID]
	h.mu.RUnlock()
	if !ok {
		return nil
	}
	info := client.Info()
	return &info
}

// Channels returns channel names with their subscriber counts.
func (h *Hub) Channels() map[string]int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make(map[string]int, len(h.channels))
	for ch, subs := range h.channels {
		result[ch] = len(subs)
	}
	return result
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
