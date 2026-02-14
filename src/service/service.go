package service

import (
	"fmt"
	"time"

	"github.com/orchestra-mcp/socket/src/hub"
	"github.com/orchestra-mcp/socket/src/types"
	"github.com/rs/zerolog"
)

// Service provides the high-level WebSocket pub/sub API.
type Service struct {
	hub    *hub.Hub
	logger zerolog.Logger
}

// New creates a new WebSocket service backed by the given hub.
func New(h *hub.Hub, logger zerolog.Logger) *Service {
	return &Service{hub: h, logger: logger}
}

// Hub returns the underlying hub.
func (s *Service) Hub() *hub.Hub { return s.hub }

// RegisterHandler registers a message handler for a channel.
func (s *Service) RegisterHandler(channel string, handler types.MessageHandler) {
	s.hub.RegisterHandler(channel, handler)
	s.logger.Debug().Str("channel", channel).Msg("handler registered")
}

// Publish sends a message to all subscribers of a channel.
func (s *Service) Publish(channel string, data any) error {
	dataMap, ok := data.(map[string]any)
	if !ok {
		dataMap = map[string]any{"value": data}
	}
	msg := types.Message{
		Channel:   channel,
		Event:     "message",
		Data:      dataMap,
		Timestamp: time.Now(),
	}
	s.hub.Publish(channel, msg)
	return nil
}

// Subscribe adds a client to a channel.
func (s *Service) Subscribe(channel, clientID string) error {
	if ok := s.hub.Subscribe(channel, clientID); !ok {
		return fmt.Errorf("client %s not found", clientID)
	}
	s.logger.Debug().
		Str("client_id", clientID).
		Str("channel", channel).
		Msg("subscribed")
	return nil
}

// Unsubscribe removes a client from a channel.
func (s *Service) Unsubscribe(channel, clientID string) error {
	if ok := s.hub.Unsubscribe(channel, clientID); !ok {
		return fmt.Errorf("channel %s or client %s not found", channel, clientID)
	}
	s.logger.Debug().
		Str("client_id", clientID).
		Str("channel", channel).
		Msg("unsubscribed")
	return nil
}

// OnConnection registers a callback for new connections.
func (s *Service) OnConnection(cb func(clientID string)) {
	s.hub.OnConnection(cb)
}

// OnDisconnection registers a callback for disconnections.
func (s *Service) OnDisconnection(cb func(clientID string)) {
	s.hub.OnDisconnection(cb)
}

// GetConnectedClients returns IDs of all connected clients.
func (s *Service) GetConnectedClients() []string {
	return s.hub.ConnectedClients()
}

// SendToClient sends a message directly to a specific client.
func (s *Service) SendToClient(clientID, channel string, data any) error {
	dataMap, ok := data.(map[string]any)
	if !ok {
		dataMap = map[string]any{"value": data}
	}
	msg := types.Message{
		Channel:   channel,
		Event:     "message",
		Data:      dataMap,
		Timestamp: time.Now(),
	}
	if ok := s.hub.SendToClient(clientID, msg); !ok {
		return fmt.Errorf("client %s not found or buffer full", clientID)
	}
	return nil
}

// GetChannels returns active channels with subscriber counts.
func (s *Service) GetChannels() map[string]int {
	return s.hub.Channels()
}

// GetClientInfo returns info for a connected client, or error.
func (s *Service) GetClientInfo(clientID string) (*types.ClientInfo, error) {
	info := s.hub.ClientInfo(clientID)
	if info == nil {
		return nil, fmt.Errorf("client %s not found", clientID)
	}
	return info, nil
}
