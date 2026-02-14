package providers

import "github.com/orchestra-mcp/framework/app/plugins"

// Compile-time interface assertions.
var (
	_ plugins.Plugin         = (*SocketPlugin)(nil)
	_ plugins.HasConfig      = (*SocketPlugin)(nil)
	_ plugins.HasFeatureFlag = (*SocketPlugin)(nil)
	_ plugins.HasRoutes      = (*SocketPlugin)(nil)
	_ plugins.HasMcpTools    = (*SocketPlugin)(nil)
	_ plugins.HasServices    = (*SocketPlugin)(nil)
)
