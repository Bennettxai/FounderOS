// Package redis provides pub/sub messaging for cross-instance WebSocket broadcasting
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Channel names for different message types
const (
	ChannelTerminalOutput  = "terminal:output"
	ChannelTerminalInput   = "terminal:input"
	ChannelTerminalResize  = "terminal:resize"
	ChannelSessionEvents   = "session:events"
	ChannelSystemBroadcast = "system:broadcast"
)

// Message represents a pub/sub message
type Message struct {
	Type      string      `json:"type"`                 // Message type (output, input, resize, etc.)
	Channel   string      `json:"channel"`              // Target channel
	SessionID string      `json:"session_id,omitempty"` // Terminal session ID
	UserID    string      `json:"user_id,omitempty"`    // User ID
	Data      interface{} `json:"data"`                 // Message payload
	Timestamp int64       `json:"timestamp"`            // Unix timestamp
	SourceID  string      `json:"source_id,omitempty"`  // Source instance ID (to avoid echo)
}

// PubSub manages Redis pub/sub for cross-instance messaging
type PubSub struct {
	client     *redis.Client
	instanceID string
	handlers   map[string][]MessageHandler
	mu         sync.RWMutex
	subs       map[string]*redis.PubSub
	ctx        context.Context
	cancel     context.CancelFunc
}

// MessageHandler is a callback for handling pub/sub messages
type MessageHandler func(msg *Message)

// NewPubSub creates a new pub/sub manager
func NewPubSub(instanceID string) (*PubSub, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client not initialized - call Connect first")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &PubSub{
		client:     client,
		instanceID: instanceID,
		handlers:   make(map[string][]MessageHandler),
		subs:       make(map[string]*redis.PubSub),
		ctx:        ctx,
		cancel:     cancel,
	}, nil
}

// Subscribe subscribes to a channel with a message handler
func (p *PubSub) Subscribe(channel string, handler MessageHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Add handler
	p.handlers[channel] = append(p.handlers[channel], handler)

	// Check if already subscribed
	if _, exists := p.subs[channel]; exists {
		return nil
	}

	// Create subscription
	sub := p.client.Subscribe(p.ctx, channel)
	p.subs[channel] = sub

	// Start message receiver goroutine
	go p.receiveMessages(channel, sub)

	slog.Info("PubSub subscribed to channel", "channel", channel)
	return nil
}

// Unsubscribe removes a subscription from a channel
func (p *PubSub) Unsubscribe(channel string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if sub, exists := p.subs[channel]; exists {
		if err := sub.Close(); err != nil {
			return fmt.Errorf("failed to close subscription: %w", err)
		}
		delete(p.subs, channel)
		delete(p.handlers, channel)
	}

	return nil
}

// Publish sends a message to a channel
func (p *PubSub) Publish(ctx context.Context, channel string, msg *Message) error {
	// Set source ID to identify origin instance
	msg.SourceID = p.instanceID
	msg.Channel = channel

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return p.client.Publish(ctx, channel, data).Err()
}

// PublishTerminalOutput broadcasts terminal output across instances
func (p *PubSub) PublishTerminalOutput(ctx context.Context, sessionID, userID string, output []byte) error {
	return p.Publish(ctx, ChannelTerminalOutput, &Message{
		Type:      "output",
		SessionID: sessionID,
		UserID:    userID,
		Data:      string(output),
		Timestamp: nowUnix(),
	})
}

// PublishTerminalResize broadcasts terminal resize events
func (p *PubSub) PublishTerminalResize(ctx context.Context, sessionID string, cols, rows uint16) error {
	return p.Publish(ctx, ChannelTerminalResize, &Message{
		Type:      "resize",
		SessionID: sessionID,
		Data: map[string]uint16{
			"cols": cols,
			"rows": rows,
		},
		Timestamp: nowUnix(),
	})
}

// PublishSessionEvent broadcasts session events (create, delete, etc.)
func (p *PubSub) PublishSessionEvent(ctx context.Context, eventType, sessionID, userID string) error {
	return p.Publish(ctx, ChannelSessionEvents, &Message{
		Type:      eventType,
		SessionID: sessionID,
		UserID:    userID,
		Timestamp: nowUnix(),
	})
}

// PublishSystemBroadcast sends a system-wide message to all instances
func (p *PubSub) PublishSystemBroadcast(ctx context.Context, messageType string, data interface{}) error {
	return p.Publish(ctx, ChannelSystemBroadcast, &Message{
		Type:      messageType,
		Data:      data,
		Timestamp: nowUnix(),
	})
}

// receiveMessages handles incoming messages for a channel
func (p *PubSub) receiveMessages(channel string, sub *redis.PubSub) {
	ch := sub.Channel()

	for {
		select {
		case <-p.ctx.Done():
			return
		case redisMsg, ok := <-ch:
			if !ok {
				return
			}

			// Parse message
			var msg Message
			if err := json.Unmarshal([]byte(redisMsg.Payload), &msg); err != nil {
				slog.Error("PubSub failed to unmarshal message", "channel", channel, "error", err)
				continue
			}

			// Skip messages from this instance (avoid echo)
			if msg.SourceID == p.instanceID {
				continue
			}

			// Dispatch to handlers
			p.mu.RLock()
			handlers := p.handlers[channel]
			p.mu.RUnlock()

			for _, handler := range handlers {
				go handler(&msg)
			}
		}
	}
}

// Close shuts down all subscriptions
func (p *PubSub) Close() error {
	p.cancel()

	p.mu.Lock()
	defer p.mu.Unlock()

	for channel, sub := range p.subs {
		if err := sub.Close(); err != nil {
			slog.Error("PubSub error closing subscription", "channel", channel, "error", err)
		}
	}

	p.subs = make(map[string]*redis.PubSub)
	p.handlers = make(map[string][]MessageHandler)

	return nil
}

// InstanceID returns this instance's unique identifier
func (p *PubSub) InstanceID() string {
	return p.instanceID
}

// nowUnix returns current Unix timestamp
func nowUnix() int64 {
	return timeNowFunc().Unix()
}

// timeNowFunc is a variable for testing (can be mocked)
var timeNowFunc = func() interface{ Unix() int64 } {
	return realTime{}
}

type realTime struct{}

func (realTime) Unix() int64 {
	return int64(time.Now().Unix())
}

// Add missing time import handled at package level
