package config

// SocketConfig holds WebSocket server configuration.
type SocketConfig struct {
	MaxConnections  int `json:"max_connections"`
	PingInterval    int `json:"ping_interval_seconds"`
	WriteTimeout    int `json:"write_timeout_seconds"`
	ReadBufferSize  int `json:"read_buffer_size"`
	WriteBufferSize int `json:"write_buffer_size"`
}

// DefaultConfig returns the default WebSocket configuration.
func DefaultConfig() *SocketConfig {
	return &SocketConfig{
		MaxConnections:  1000,
		PingInterval:    30,
		WriteTimeout:    10,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
}
