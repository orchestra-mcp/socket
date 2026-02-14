package hub

import (
	"github.com/orchestra-mcp/socket/src/types"
)

func (h *Hub) handleMessage(msg types.Message) {
	h.mu.RLock()
	handler, ok := h.handlers[msg.Channel]
	h.mu.RUnlock()

	if !ok {
		h.logger.Debug().Str("channel", msg.Channel).Msg("no handler")
		return
	}
	if err := handler(msg.ClientID, msg); err != nil {
		h.logger.Error().Err(err).Str("channel", msg.Channel).Msg("handler error")
	}
}

func (h *Hub) broadcastToChannel(channel string, msg types.Message) {
	h.mu.RLock()
	subs, ok := h.channels[channel]
	if !ok {
		h.mu.RUnlock()
		return
	}
	// Copy subscriber IDs to avoid holding lock during sends.
	ids := make([]string, 0, len(subs))
	for id := range subs {
		ids = append(ids, id)
	}
	h.mu.RUnlock()

	for _, id := range ids {
		h.mu.RLock()
		client, exists := h.clients[id]
		h.mu.RUnlock()
		if !exists {
			continue
		}
		select {
		case client.Send <- msg:
		default:
			h.logger.Warn().Str("client_id", id).Msg("send buffer full, dropping")
		}
	}
}

// publishToBridge forwards a message to the bridge if one is attached.
func (h *Hub) publishToBridge(msg types.Message) {
	h.mu.RLock()
	b := h.bridge
	h.mu.RUnlock()

	if b == nil || !b.Available() {
		return
	}
	if err := b.Publish(msg); err != nil {
		h.logger.Error().Err(err).Msg("bridge publish failed")
	}
}

// Publish sends a message to all subscribers of a channel.
func (h *Hub) Publish(channel string, msg types.Message) {
	h.broadcast <- broadcastMsg{channel: channel, msg: msg}
}

// Subscribe adds a client to a channel.
func (h *Hub) Subscribe(channel, clientID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[clientID]; !ok {
		return false
	}
	if h.channels[channel] == nil {
		h.channels[channel] = make(map[string]bool)
	}
	h.channels[channel][clientID] = true
	h.clients[clientID].AddChannel(channel)
	return true
}

// Unsubscribe removes a client from a channel.
func (h *Hub) Unsubscribe(channel, clientID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	subs, ok := h.channels[channel]
	if !ok {
		return false
	}
	delete(subs, clientID)
	if len(subs) == 0 {
		delete(h.channels, channel)
	}
	if c, ok := h.clients[clientID]; ok {
		c.RemoveChannel(channel)
	}
	return true
}

// SendToClient sends a message directly to a specific client.
func (h *Hub) SendToClient(clientID string, msg types.Message) bool {
	h.mu.RLock()
	client, ok := h.clients[clientID]
	h.mu.RUnlock()
	if !ok {
		return false
	}
	select {
	case client.Send <- msg:
		return true
	default:
		return false
	}
}
