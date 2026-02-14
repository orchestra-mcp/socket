package providers

import (
	"github.com/orchestra-mcp/framework/app/plugins"
	"github.com/orchestra-mcp/socket/config"
	"github.com/orchestra-mcp/socket/src/bridge"
	"github.com/orchestra-mcp/socket/src/hub"
	"github.com/orchestra-mcp/socket/src/service"
)

// SocketPlugin implements the Orchestra plugin interface for WebSocket.
type SocketPlugin struct {
	active  bool
	ctx     *plugins.PluginContext
	cfg     *config.SocketConfig
	hub     *hub.Hub
	service *service.Service
	bridge  bridge.Bridge
}

// NewSocketPlugin creates a new WebSocket plugin instance.
func NewSocketPlugin() *SocketPlugin { return &SocketPlugin{} }

func (p *SocketPlugin) ID() string             { return "orchestra/socket" }
func (p *SocketPlugin) Name() string           { return "WebSocket" }
func (p *SocketPlugin) Version() string        { return "0.1.0" }
func (p *SocketPlugin) Dependencies() []string { return nil }
func (p *SocketPlugin) IsActive() bool         { return p.active }
func (p *SocketPlugin) FeatureFlag() string    { return "websocket" }
func (p *SocketPlugin) ConfigKey() string      { return "socket" }

func (p *SocketPlugin) DefaultConfig() map[string]any {
	return map[string]any{
		"max_connections": 1000,
		"ping_interval":   30,
		"write_timeout":   10,
		"read_buffer":     1024,
		"write_buffer":    1024,
	}
}

// Activate initializes the hub, service, and starts the event loop.
func (p *SocketPlugin) Activate(ctx *plugins.PluginContext) error {
	p.ctx = ctx
	p.cfg = config.DefaultConfig()
	p.hub = hub.New(ctx.Logger)
	p.service = service.New(p.hub, ctx.Logger)

	go p.hub.Run()

	// Attempt Redis bridge connection (non-fatal if unavailable).
	p.initBridge(ctx)

	p.active = true
	ctx.Logger.Info().Str("plugin", p.ID()).Msg("websocket plugin activated")
	return nil
}

// initBridge tries to start the Redis pub/sub bridge.
// If Redis is not reachable, the hub runs in standalone mode.
func (p *SocketPlugin) initBridge(ctx *plugins.PluginContext) {
	cfg := bridge.RedisConfigFromEnv()
	rb := bridge.NewRedisBridge(cfg, p.hub, ctx.Logger)

	if err := rb.Start(); err != nil {
		ctx.Logger.Warn().Err(err).Msg("redis bridge unavailable, running standalone")
		return
	}

	p.bridge = rb
	p.hub.SetBridge(rb)
	ctx.Logger.Info().Str("redis_addr", cfg.Addr).Msg("redis bridge connected")
}

// Deactivate stops the bridge and hub event loop.
func (p *SocketPlugin) Deactivate() error {
	if p.bridge != nil {
		if err := p.bridge.Stop(); err != nil {
			p.ctx.Logger.Error().Err(err).Msg("bridge stop error")
		}
		p.bridge = nil
	}
	if p.hub != nil {
		p.hub.Stop()
	}
	p.active = false
	return nil
}

// Services exposes the WebSocket service for dependency injection.
func (p *SocketPlugin) Services() []plugins.ServiceDefinition {
	return []plugins.ServiceDefinition{
		{
			ID:      "socket.service",
			Factory: func() any { return p.service },
		},
	}
}
