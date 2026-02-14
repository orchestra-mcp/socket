package bridge

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/google/uuid"
	"github.com/orchestra-mcp/socket/src/types"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// redisEnvelope wraps a message with the originating instance ID
// so that a node can skip its own published messages.
type redisEnvelope struct {
	InstanceID string        `json:"instance_id"`
	Message    types.Message `json:"message"`
}

// RedisBridge relays WebSocket messages between server instances via Redis pub/sub.
type RedisBridge struct {
	client     *redis.Client
	prefix     string
	instanceID string
	hub        BroadcastTarget
	logger     zerolog.Logger

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
	active bool
}

// NewRedisBridge creates a bridge that uses Redis pub/sub for cross-instance messaging.
func NewRedisBridge(cfg *RedisConfig, hub BroadcastTarget, logger zerolog.Logger) *RedisBridge {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx, cancel := context.WithCancel(context.Background())

	return &RedisBridge{
		client:     client,
		prefix:     cfg.Prefix,
		instanceID: uuid.New().String(),
		hub:        hub,
		logger:     logger.With().Str("component", "redis-bridge").Logger(),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start subscribes to the Redis broadcast channel and begins relaying messages.
func (b *RedisBridge) Start() error {
	if err := b.client.Ping(b.ctx).Err(); err != nil {
		return err
	}

	channel := b.prefix + "broadcast"
	sub := b.client.Subscribe(b.ctx, channel)

	// Wait for subscription confirmation.
	if _, err := sub.Receive(b.ctx); err != nil {
		return err
	}

	b.mu.Lock()
	b.active = true
	b.mu.Unlock()

	b.wg.Add(1)
	go b.listen(sub)

	b.logger.Info().
		Str("instance_id", b.instanceID).
		Str("channel", channel).
		Msg("redis bridge started")
	return nil
}

// Publish sends a message to all other instances via Redis.
func (b *RedisBridge) Publish(msg types.Message) error {
	env := redisEnvelope{
		InstanceID: b.instanceID,
		Message:    msg,
	}
	data, err := json.Marshal(env)
	if err != nil {
		return err
	}
	channel := b.prefix + "broadcast"
	return b.client.Publish(b.ctx, channel, data).Err()
}

// Stop unsubscribes and closes the Redis connection.
func (b *RedisBridge) Stop() error {
	b.mu.Lock()
	b.active = false
	b.mu.Unlock()

	b.cancel()
	b.wg.Wait()
	return b.client.Close()
}

// Available reports whether the bridge is connected.
func (b *RedisBridge) Available() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.active
}

// listen reads messages from the Redis subscription and forwards to the local hub.
func (b *RedisBridge) listen(sub *redis.PubSub) {
	defer b.wg.Done()
	defer sub.Close()

	ch := sub.Channel()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			b.handleRedisMessage(msg)
		case <-b.ctx.Done():
			return
		}
	}
}

// handleRedisMessage decodes an envelope and forwards non-self messages to the hub.
func (b *RedisBridge) handleRedisMessage(msg *redis.Message) {
	var env redisEnvelope
	if err := json.Unmarshal([]byte(msg.Payload), &env); err != nil {
		b.logger.Error().Err(err).Msg("failed to decode redis message")
		return
	}

	// Skip messages that originated from this instance.
	if env.InstanceID == b.instanceID {
		return
	}

	b.logger.Debug().
		Str("from_instance", env.InstanceID).
		Str("channel", env.Message.Channel).
		Msg("relaying message from redis")

	b.hub.BroadcastToLocal(env.Message)
}
