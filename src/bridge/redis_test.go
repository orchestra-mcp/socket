package bridge

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/orchestra-mcp/socket/src/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBroadcastTarget records messages forwarded from the bridge.
type mockBroadcastTarget struct {
	received []types.Message
}

func (m *mockBroadcastTarget) BroadcastToLocal(msg types.Message) {
	m.received = append(m.received, msg)
}

func TestRedisEnvelopeSerialization(t *testing.T) {
	msg := types.Message{
		Channel:   "sync",
		Event:     "update",
		Data:      map[string]any{"key": "value"},
		ClientID:  "client-1",
		Timestamp: time.Now().Truncate(time.Second),
	}

	env := redisEnvelope{
		InstanceID: "instance-abc",
		Message:    msg,
	}

	data, err := json.Marshal(env)
	require.NoError(t, err)

	var decoded redisEnvelope
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, env.InstanceID, decoded.InstanceID)
	assert.Equal(t, msg.Channel, decoded.Message.Channel)
	assert.Equal(t, msg.Event, decoded.Message.Event)
	assert.Equal(t, msg.ClientID, decoded.Message.ClientID)
	assert.Equal(t, "value", decoded.Message.Data["key"])
}

func TestRedisEnvelopeRoundTrip(t *testing.T) {
	msg := types.Message{
		Channel:   "presence",
		Event:     "join",
		Data:      map[string]any{"user": "alice", "count": float64(5)},
		Timestamp: time.Now().Truncate(time.Millisecond),
	}

	env := redisEnvelope{
		InstanceID: "node-1",
		Message:    msg,
	}

	data, err := json.Marshal(env)
	require.NoError(t, err)

	var out redisEnvelope
	require.NoError(t, json.Unmarshal(data, &out))

	assert.Equal(t, "node-1", out.InstanceID)
	assert.Equal(t, "presence", out.Message.Channel)
	assert.Equal(t, "join", out.Message.Event)
	assert.Equal(t, "alice", out.Message.Data["user"])
	assert.Equal(t, float64(5), out.Message.Data["count"])
}

func TestDefaultRedisConfig(t *testing.T) {
	cfg := DefaultRedisConfig()
	assert.Equal(t, "localhost:6379", cfg.Addr)
	assert.Empty(t, cfg.Password)
	assert.Equal(t, 0, cfg.DB)
	assert.Equal(t, "orchestra:ws:", cfg.Prefix)
}

func TestRedisConfigFromEnv(t *testing.T) {
	t.Setenv("REDIS_ADDR", "redis.example.com:6380")
	t.Setenv("REDIS_PASSWORD", "secret")
	t.Setenv("REDIS_DB", "3")
	t.Setenv("REDIS_WS_PREFIX", "test:ws:")

	cfg := RedisConfigFromEnv()
	assert.Equal(t, "redis.example.com:6380", cfg.Addr)
	assert.Equal(t, "secret", cfg.Password)
	assert.Equal(t, 3, cfg.DB)
	assert.Equal(t, "test:ws:", cfg.Prefix)
}

func TestRedisConfigFromEnvDefaults(t *testing.T) {
	// No env vars set, should return defaults.
	cfg := RedisConfigFromEnv()
	assert.Equal(t, "localhost:6379", cfg.Addr)
	assert.Empty(t, cfg.Password)
	assert.Equal(t, 0, cfg.DB)
	assert.Equal(t, "orchestra:ws:", cfg.Prefix)
}

func TestRedisConfigFromEnvInvalidDB(t *testing.T) {
	t.Setenv("REDIS_DB", "not-a-number")

	cfg := RedisConfigFromEnv()
	assert.Equal(t, 0, cfg.DB) // falls back to default
}

func TestRedisBridgeAvailableFalseBeforeStart(t *testing.T) {
	target := &mockBroadcastTarget{}
	cfg := DefaultRedisConfig()
	rb := NewRedisBridge(cfg, target, testLogger())
	assert.False(t, rb.Available())
}

func TestRedisBridgeInstanceIDUnique(t *testing.T) {
	target := &mockBroadcastTarget{}
	cfg := DefaultRedisConfig()
	b1 := NewRedisBridge(cfg, target, testLogger())
	b2 := NewRedisBridge(cfg, target, testLogger())
	assert.NotEqual(t, b1.instanceID, b2.instanceID)
}

func testLogger() zerolog.Logger {
	return zerolog.Nop()
}
