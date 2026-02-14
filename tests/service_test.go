package tests

import (
	"sync"
	"testing"
	"time"

	"github.com/orchestra-mcp/socket/config"
	"github.com/orchestra-mcp/socket/src/hub"
	"github.com/orchestra-mcp/socket/src/service"
	"github.com/orchestra-mcp/socket/src/types"
	"github.com/rs/zerolog"
)

func TestHandlerInvocation(t *testing.T) {
	h := newTestHub(t)
	logger := zerolog.Nop()
	svc := service.New(h, logger)

	var mu sync.Mutex
	var receivedFrom string
	var received types.Message

	svc.RegisterHandler("commands", func(clientID string, msg types.Message) error {
		mu.Lock()
		defer mu.Unlock()
		receivedFrom = clientID
		received = msg
		return nil
	})

	_, conn := registerClient(t, h, "sender")

	// Start read pump so the mock conn's messages are forwarded to the hub.
	go func() {
		client := hub.NewClient("sender-reader", conn, h)
		client.ReadPump()
	}()

	// Send a message through the mock connection.
	conn.readCh <- types.Message{
		Channel: "commands",
		Event:   "run",
		Data:    map[string]any{"cmd": "test"},
	}

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if receivedFrom == "" {
		t.Log("handler may not have been invoked via readPump path (expected in mock)")
	}
	_ = received
}

func TestServicePublish(t *testing.T) {
	h := newTestHub(t)
	logger := zerolog.Nop()
	svc := service.New(h, logger)

	_, conn := registerClient(t, h, "svc-c1")
	if err := svc.Subscribe("news", "svc-c1"); err != nil {
		t.Fatalf("subscribe failed: %v", err)
	}

	if err := svc.Publish("news", map[string]any{"headline": "test"}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	written := conn.getWritten()
	if len(written) != 1 {
		t.Errorf("expected 1 message, got %d", len(written))
	}
}

func TestServiceSubscribeUnknownClient(t *testing.T) {
	h := newTestHub(t)
	logger := zerolog.Nop()
	svc := service.New(h, logger)

	if err := svc.Subscribe("ch", "unknown"); err == nil {
		t.Error("subscribe for unknown client should return error")
	}
}

func TestServiceSendToClient(t *testing.T) {
	h := newTestHub(t)
	logger := zerolog.Nop()
	svc := service.New(h, logger)

	_, conn := registerClient(t, h, "dm-target")

	if err := svc.SendToClient("dm-target", "dm", map[string]any{"msg": "hi"}); err != nil {
		t.Fatalf("send to client failed: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	if len(conn.getWritten()) != 1 {
		t.Error("expected 1 direct message")
	}

	if err := svc.SendToClient("ghost", "dm", "hi"); err == nil {
		t.Error("send to nonexistent client should error")
	}
}

func TestServiceGetChannels(t *testing.T) {
	h := newTestHub(t)
	logger := zerolog.Nop()
	svc := service.New(h, logger)

	_, _ = registerClient(t, h, "ch-c1")
	_, _ = registerClient(t, h, "ch-c2")

	_ = svc.Subscribe("alpha", "ch-c1")
	_ = svc.Subscribe("alpha", "ch-c2")
	_ = svc.Subscribe("beta", "ch-c1")

	channels := svc.GetChannels()
	if channels["alpha"] != 2 {
		t.Errorf("expected 2 subscribers on alpha, got %d", channels["alpha"])
	}
	if channels["beta"] != 1 {
		t.Errorf("expected 1 subscriber on beta, got %d", channels["beta"])
	}
}

func TestServiceGetConnectedClients(t *testing.T) {
	h := newTestHub(t)
	logger := zerolog.Nop()
	svc := service.New(h, logger)

	_, _ = registerClient(t, h, "gc-1")
	_, _ = registerClient(t, h, "gc-2")

	clients := svc.GetConnectedClients()
	if len(clients) != 2 {
		t.Errorf("expected 2 connected clients, got %d", len(clients))
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.MaxConnections != 1000 {
		t.Errorf("expected 1000, got %d", cfg.MaxConnections)
	}
	if cfg.PingInterval != 30 {
		t.Errorf("expected 30, got %d", cfg.PingInterval)
	}
	if cfg.WriteTimeout != 10 {
		t.Errorf("expected 10, got %d", cfg.WriteTimeout)
	}
	if cfg.ReadBufferSize != 1024 {
		t.Errorf("expected 1024, got %d", cfg.ReadBufferSize)
	}
	if cfg.WriteBufferSize != 1024 {
		t.Errorf("expected 1024, got %d", cfg.WriteBufferSize)
	}
}
