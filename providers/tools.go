package providers

import (
	"fmt"

	"github.com/orchestra-mcp/framework/app/plugins"
)

// McpTools returns MCP tool definitions contributed by the WebSocket plugin.
func (p *SocketPlugin) McpTools() []plugins.McpToolDefinition {
	return []plugins.McpToolDefinition{
		{
			Name:        "list_ws_clients",
			Description: "List connected WebSocket clients",
			InputSchema: map[string]any{},
			Handler:     p.toolListClients,
		},
		{
			Name:        "ws_publish",
			Description: "Publish a message to a WebSocket channel",
			InputSchema: map[string]any{
				"channel": map[string]any{"type": "string", "description": "Channel name"},
				"data":    map[string]any{"type": "object", "description": "Message data"},
			},
			Handler: p.toolPublish,
		},
		{
			Name:        "list_ws_channels",
			Description: "List active WebSocket channels with subscriber counts",
			InputSchema: map[string]any{},
			Handler:     p.toolListChannels,
		},
	}
}

func (p *SocketPlugin) toolListClients(_ map[string]any) (any, error) {
	if p.service == nil {
		return nil, fmt.Errorf("websocket service not initialized")
	}
	clients := p.service.GetConnectedClients()
	infos := make([]any, 0, len(clients))
	for _, id := range clients {
		info, err := p.service.GetClientInfo(id)
		if err == nil {
			infos = append(infos, info)
		}
	}
	return map[string]any{
		"clients": infos,
		"count":   len(infos),
	}, nil
}

func (p *SocketPlugin) toolPublish(input map[string]any) (any, error) {
	if p.service == nil {
		return nil, fmt.Errorf("websocket service not initialized")
	}
	channel, _ := input["channel"].(string)
	if channel == "" {
		return nil, fmt.Errorf("channel is required")
	}
	data, _ := input["data"].(map[string]any)
	if data == nil {
		data = map[string]any{}
	}
	if err := p.service.Publish(channel, data); err != nil {
		return nil, err
	}
	return map[string]any{"published": true, "channel": channel}, nil
}

func (p *SocketPlugin) toolListChannels(_ map[string]any) (any, error) {
	if p.service == nil {
		return nil, fmt.Errorf("websocket service not initialized")
	}
	channels := p.service.GetChannels()
	result := make([]map[string]any, 0, len(channels))
	for name, count := range channels {
		result = append(result, map[string]any{
			"channel":     name,
			"subscribers": count,
		})
	}
	return map[string]any{"channels": result, "count": len(result)}, nil
}
