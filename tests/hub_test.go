package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/orchestra-mcp/socket/src/hub"
	"github.com/orchestra-mcp/socket/src/types"
	"github.com/rs/zerolog"
)

// mockConn implements types.Conn for testing without a real WebSocket.
type mockConn struct {
	mu       sync.Mutex
	written  []any
	readCh   chan types.Message
	closed   bool
	closedCh chan struct{}
}

func newMockConn() *mockConn {
	return &mockConn{
		readCh:   make(chan types.Message, 16),
		closedCh: make(chan struct{}),
	}
}

func (m *mockConn) WriteJSON(v any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.written = append(m.written, v)
	return nil
}

func (m *mockConn) ReadJSON(v any) error {
	select {
	case msg := <-m.readCh:
		ptr, ok := v.(*types.Message)
		if ok {
			*ptr = msg
		}
		return nil
	case <-m.closedCh:
		return &closeError{}
	}
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.closed {
		m.closed = true
		close(m.closedCh)
	}
	return nil
}

func (m *mockConn) getWritten() []any {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]any, len(m.written))
	copy(cp, m.written)
	return cp
}

type closeError struct{}

func (e *closeError) Error() string { return "connection closed" }

// newTestHub creates a hub and starts its event loop in a goroutine.
func newTestHub(t *testing.T) *hub.Hub {
	t.Helper()
	logger := zerolog.Nop()
	h := hub.New(logger)
	go h.Run()
	t.Cleanup(func() { h.Stop() })
	return h
}

// registerClient creates, registers, and starts a mock client.
func registerClient(t *testing.T, h *hub.Hub, id string) (*hub.Client, *mockConn) {
	t.Helper()
	conn := newMockConn()
	client := hub.NewClient(id, conn, h)
	h.Register(client)
	go client.WritePump()
	// Allow registration to process.
	time.Sleep(20 * time.Millisecond)
	return client, conn
}

func TestHubRegisterAndUnregister(t *testing.T) {
	h := newTestHub(t)

	_, _ = registerClient(t, h, "client-1")
	_, _ = registerClient(t, h, "client-2")

	clients := h.ConnectedClients()
	if len(clients) != 2 {
		t.Fatalf("expected 2 clients, got %d", len(clients))
	}

	// Unregister one.
	info := h.ClientInfo("client-1")
	if info == nil {
		t.Fatal("expected client-1 info")
	}
	c1, _ := registerClient(t, h, "client-3")
	h.Unregister(c1)
	time.Sleep(20 * time.Millisecond)

	// client-3 should be gone, but client-1 and client-2 remain.
	if h.ClientInfo("client-3") != nil {
		t.Error("expected client-3 to be unregistered")
	}
}

func TestHubSubscribeAndUnsubscribe(t *testing.T) {
	h := newTestHub(t)
	_, _ = registerClient(t, h, "c1")

	if ok := h.Subscribe("events", "c1"); !ok {
		t.Fatal("subscribe should succeed for registered client")
	}

	channels := h.Channels()
	if channels["events"] != 1 {
		t.Errorf("expected 1 subscriber on events, got %d", channels["events"])
	}

	if ok := h.Subscribe("events", "nonexistent"); ok {
		t.Error("subscribe should fail for unregistered client")
	}

	h.Unsubscribe("events", "c1")
	channels = h.Channels()
	if _, ok := channels["events"]; ok {
		t.Error("expected events channel to be removed after last unsubscribe")
	}
}

func TestPublishToChannel(t *testing.T) {
	h := newTestHub(t)
	_, conn1 := registerClient(t, h, "c1")
	_, conn2 := registerClient(t, h, "c2")

	h.Subscribe("updates", "c1")
	h.Subscribe("updates", "c2")

	msg := types.Message{
		Channel: "updates",
		Event:   "test",
		Data:    map[string]any{"key": "value"},
	}
	h.Publish("updates", msg)

	// Allow broadcast to process.
	time.Sleep(50 * time.Millisecond)

	written1 := conn1.getWritten()
	written2 := conn2.getWritten()
	if len(written1) != 1 {
		t.Errorf("expected 1 message for c1, got %d", len(written1))
	}
	if len(written2) != 1 {
		t.Errorf("expected 1 message for c2, got %d", len(written2))
	}
}

func TestPublishDoesNotReachUnsubscribed(t *testing.T) {
	h := newTestHub(t)
	_, conn1 := registerClient(t, h, "c1")
	_, conn2 := registerClient(t, h, "c2")

	// Only c1 subscribes.
	h.Subscribe("private", "c1")

	msg := types.Message{Channel: "private", Event: "test"}
	h.Publish("private", msg)
	time.Sleep(50 * time.Millisecond)

	if len(conn1.getWritten()) != 1 {
		t.Error("c1 should receive the message")
	}
	if len(conn2.getWritten()) != 0 {
		t.Error("c2 should not receive the message")
	}
}

func TestSendToClient(t *testing.T) {
	h := newTestHub(t)
	_, conn := registerClient(t, h, "target")

	msg := types.Message{Channel: "dm", Event: "direct", Data: map[string]any{"hello": "world"}}
	if ok := h.SendToClient("target", msg); !ok {
		t.Fatal("send to existing client should succeed")
	}
	time.Sleep(20 * time.Millisecond)

	written := conn.getWritten()
	if len(written) != 1 {
		t.Fatalf("expected 1 message, got %d", len(written))
	}

	if ok := h.SendToClient("nonexistent", msg); ok {
		t.Error("send to nonexistent client should fail")
	}
}

func TestConnectionCallbacks(t *testing.T) {
	h := newTestHub(t)

	var connectedID string
	var disconnectedID string
	h.OnConnection(func(id string) { connectedID = id })
	h.OnDisconnection(func(id string) { disconnectedID = id })

	client, _ := registerClient(t, h, "cb-client")
	time.Sleep(20 * time.Millisecond)

	if connectedID != "cb-client" {
		t.Errorf("expected connected callback with cb-client, got %s", connectedID)
	}

	h.Unregister(client)
	time.Sleep(20 * time.Millisecond)

	if disconnectedID != "cb-client" {
		t.Errorf("expected disconnected callback with cb-client, got %s", disconnectedID)
	}
}

func TestClientInfo(t *testing.T) {
	h := newTestHub(t)

	_, _ = registerClient(t, h, "info-client")
	h.Subscribe("ch-a", "info-client")
	h.Subscribe("ch-b", "info-client")

	info := h.ClientInfo("info-client")
	if info == nil {
		t.Fatal("expected client info")
	}
	if info.ID != "info-client" {
		t.Errorf("expected ID info-client, got %s", info.ID)
	}
	if len(info.Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(info.Channels))
	}
}
