// Package terminal provides pub/sub integration for horizontal scaling
package terminal

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Channel names for terminal pub/sub
const (
	ChannelTerminalOutput = "terminal:output"
	ChannelTerminalResize = "terminal:resize"
	ChannelSessionEvents  = "terminal:sessions"
)

// TerminalPubSub manages Redis pub/sub for terminal session broadcasting
type TerminalPubSub struct {
	client     *redis.Client
	instanceID string
	handlers   map[string][]PubSubHandler
	subs       map[string]*redis.PubSub
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// PubSubMessage represents a terminal pub/sub message
type PubSubMessage struct {
	Type      string      `json:"type"`                // Message type
	SessionID string      `json:"session_id"`          // Terminal session ID
	UserID    string      `json:"user_id,omitempty"`   // User ID
	Data      interface{} `json:"data"`                // Payload
	Timestamp int64       `json:"timestamp"`           // Unix timestamp
	SourceID  string      `json:"source_id,omitempty"` // Source instance (avoid echo)
}

// PubSubHandler handles incoming pub/sub messages
type PubSubHandler func(msg *PubSubMessage)

// SessionEvent represents a session lifecycle event
type SessionEvent struct {
	Event     string `json:"event"` // created, closed, resize
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	Cols      int    `json:"cols,omitempty"`
	Rows      int    `json:"rows,omitempty"`
}

// NewTerminalPubSub creates a new terminal pub/sub manager
func NewTerminalPubSub(client *redis.Client, instanceID string) *TerminalPubSub {
	ctx, cancel := context.WithCancel(context.Background())
	return &TerminalPubSub{
		client:     client,
		instanceID: instanceID,
		handlers:   make(map[string][]PubSubHandler),
		subs:       make(map[string]*redis.PubSub),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Subscribe adds a handler for a channel
func (p *TerminalPubSub) Subscribe(channel string, handler PubSubHandler) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.handlers[channel] = append(p.handlers[channel], handler)

	if _, exists := p.subs[channel]; exists {
		return nil
	}

	sub := p.client.Subscribe(p.ctx, channel)
	p.subs[channel] = sub

	go p.receiveMessages(channel, sub)
	slog.Info("TerminalPubSub: subscribed", "channel", channel)
	return nil
}

// Publish sends a message to a channel
func (p *TerminalPubSub) Publish(ctx context.Context, channel string, msg *PubSubMessage) error {
	msg.SourceID = p.instanceID
	msg.Timestamp = time.Now().Unix()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return p.client.Publish(ctx, channel, data).Err()
}

// PublishSessionEvent broadcasts a session event
func (p *TerminalPubSub) PublishSessionEvent(ctx context.Context, event, sessionID, userID string) error {
	return p.Publish(ctx, ChannelSessionEvents, &PubSubMessage{
		Type:      event,
		SessionID: sessionID,
		UserID:    userID,
		Data: SessionEvent{
			Event:     event,
			SessionID: sessionID,
			UserID:    userID,
		},
	})
}

// PublishResize broadcasts a terminal resize event
func (p *TerminalPubSub) PublishResize(ctx context.Context, sessionID string, cols, rows int) error {
	return p.Publish(ctx, ChannelTerminalResize, &PubSubMessage{
		Type:      "resize",
		SessionID: sessionID,
		Data: map[string]int{
			"cols": cols,
			"rows": rows,
		},
	})
}

// PublishOutput broadcasts terminal output (for session sharing/observation)
func (p *TerminalPubSub) PublishOutput(ctx context.Context, sessionID, userID string, output []byte) error {
	return p.Publish(ctx, ChannelTerminalOutput, &PubSubMessage{
		Type:      "output",
		SessionID: sessionID,
		UserID:    userID,
		Data:      string(output),
	})
}

func (p *TerminalPubSub) receiveMessages(channel string, sub *redis.PubSub) {
	ch := sub.Channel()

	for {
		select {
		case <-p.ctx.Done():
			return
		case redisMsg, ok := <-ch:
			if !ok {
				return
			}

			var msg PubSubMessage
			if err := json.Unmarshal([]byte(redisMsg.Payload), &msg); err != nil {
				slog.Error("TerminalPubSub: unmarshal error", "error", err)
				continue
			}

			// Skip messages from this instance
			if msg.SourceID == p.instanceID {
				continue
			}

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
func (p *TerminalPubSub) Close() error {
	p.cancel()

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, sub := range p.subs {
		sub.Close()
	}

	p.subs = make(map[string]*redis.PubSub)
	p.handlers = make(map[string][]PubSubHandler)

	slog.Info("TerminalPubSub: closed")
	return nil
}

// InstanceID returns this instance's identifier
func (p *TerminalPubSub) InstanceID() string {
	return p.instanceID
}

// IsAvailable returns true if pub/sub is enabled
func (p *TerminalPubSub) IsAvailable() bool {
	return p.client != nil
}
