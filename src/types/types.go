package types

import "time"

// Message is a WebSocket message.
type Message struct {
	Channel   string         `json:"channel"`
	Event     string         `json:"event"`
	Data      map[string]any `json:"data,omitempty"`
	ClientID  string         `json:"client_id,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
}

// MessageHandler handles incoming messages on a channel.
type MessageHandler func(clientID string, msg Message) error

// ClientInfo holds metadata about a connected WebSocket client.
type ClientInfo struct {
	ID          string    `json:"id"`
	ConnectedAt time.Time `json:"connected_at"`
	Channels    []string  `json:"channels"`
	UserAgent   string    `json:"user_agent,omitempty"`
}

// Conn abstracts a WebSocket connection for testability.
type Conn interface {
	WriteJSON(v any) error
	ReadJSON(v any) error
	Close() error
}
