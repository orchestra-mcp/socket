package providers

import (
	"strings"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/orchestra-mcp/socket/src/hub"
	"github.com/valyala/fasthttp"
)

var upgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// RegisterRoutes registers the WebSocket info route via Fiber.
// The actual WebSocket upgrade uses FastHTTPHandler, registered
// at the app level since Fiber v3 does not expose *fasthttp.RequestCtx.
func (p *SocketPlugin) RegisterRoutes(group fiber.Router) {
	group.Get("/ws/info", p.handleInfo)
}

func (p *SocketPlugin) handleInfo(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"websocket":  true,
		"endpoint":   "/ws",
		"clients":    p.hub.ClientCount(),
		"channels":   len(p.hub.Channels()),
	})
}

// FastHTTPHandler returns a raw fasthttp handler for WebSocket upgrades.
// Register this on the fasthttp server at the "/ws" path.
func (p *SocketPlugin) FastHTTPHandler() fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		upgrade := string(ctx.Request.Header.Peek("Upgrade"))
		if !strings.EqualFold(upgrade, "websocket") {
			ctx.SetStatusCode(fasthttp.StatusUpgradeRequired)
			ctx.SetBodyString(`{"error":"upgrade_required","message":"WebSocket upgrade required"}`)
			return
		}

		clientID := uuid.New().String()
		h := p.hub
		logger := p.ctx.Logger

		err := upgrader.Upgrade(ctx, func(conn *websocket.Conn) {
			client := hub.NewClient(clientID, &fasthttpConn{conn}, h)
			h.Register(client)
			go client.WritePump()
			client.ReadPump()
		})
		if err != nil {
			logger.Error().Err(err).Msg("websocket upgrade failed")
		}
	}
}

// fasthttpConn wraps fasthttp/websocket.Conn to satisfy types.Conn.
type fasthttpConn struct {
	conn *websocket.Conn
}

func (f *fasthttpConn) WriteJSON(v any) error { return f.conn.WriteJSON(v) }
func (f *fasthttpConn) ReadJSON(v any) error  { return f.conn.ReadJSON(v) }
func (f *fasthttpConn) Close() error          { return f.conn.Close() }
