package services

import (
	"context"
	"sync"
	"time"
)

type SSEEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// clientState tracks health and metrics for each client channel
type clientState struct {
	ch           chan SSEEvent
	lastActivity time.Time
	failedSends  int
}

// SSEBroadcaster manages SSE connections per user with health checking and auto-cleanup
type SSEBroadcaster struct {
	mu      sync.RWMutex
	clients map[string]map[chan SSEEvent]*clientState

	// Configuration
	sendTimeout     time.Duration // Timeout for non-blocking sends
	maxFailedSends  int           // Max failed sends before marking unhealthy
	cleanupInterval time.Duration // How often to run cleanup
	inactiveTimeout time.Duration // Time before inactive client is removed

	// Cleanup goroutine
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewSSEBroadcaster creates a new broadcaster with default settings
func NewSSEBroadcaster() *SSEBroadcaster {
	return NewSSEBroadcasterWithConfig(
		100*time.Millisecond, // sendTimeout
		3,                    // maxFailedSends
		30*time.Second,       // cleanupInterval
		5*time.Minute,        // inactiveTimeout
	)
}

// NewSSEBroadcasterWithConfig creates a broadcaster with custom configuration
func NewSSEBroadcasterWithConfig(sendTimeout time.Duration, maxFailedSends int, cleanupInterval, inactiveTimeout time.Duration) *SSEBroadcaster {
	ctx, cancel := context.WithCancel(context.Background())
	b := &SSEBroadcaster{
		clients:         make(map[string]map[chan SSEEvent]*clientState),
		sendTimeout:     sendTimeout,
		maxFailedSends:  maxFailedSends,
		cleanupInterval: cleanupInterval,
		inactiveTimeout: inactiveTimeout,
		ctx:             ctx,
		cancel:          cancel,
	}

	// Start background cleanup goroutine
	b.wg.Add(1)
	go b.cleanupLoop()

	return b
}

// maxSSEConnectionsPerUser is the maximum number of concurrent SSE connections
// allowed per user. Connections beyond this limit are rejected to prevent
// resource exhaustion from runaway clients or connection storms.
const maxSSEConnectionsPerUser = 10

// Subscribe creates a channel for a user.
// Returns nil if the per-user connection limit has been reached.
func (b *SSEBroadcaster) Subscribe(userID string) chan SSEEvent {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.clients[userID] == nil {
		b.clients[userID] = make(map[chan SSEEvent]*clientState)
	}

	if len(b.clients[userID]) >= maxSSEConnectionsPerUser {
		return nil
	}

	ch := make(chan SSEEvent, 16)
	b.clients[userID][ch] = &clientState{
		ch:           ch,
		lastActivity: time.Now(),
		failedSends:  0,
	}
	return ch
}

// Unsubscribe removes a client channel
func (b *SSEBroadcaster) Unsubscribe(userID string, ch chan SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if channels, ok := b.clients[userID]; ok {
		if state, exists := channels[ch]; exists {
			delete(channels, ch)
			close(state.ch)
			if len(channels) == 0 {
				delete(b.clients, userID)
			}
		}
	}
}

// SendToUser sends an event to all connections for a user with timeout and health tracking
func (b *SSEBroadcaster) SendToUser(userID string, event SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if channels, ok := b.clients[userID]; ok {
		var toRemove []chan SSEEvent
		for ch, state := range channels {
			sent := b.trySendWithTimeout(ch, event)
			if sent {
				state.lastActivity = time.Now()
				state.failedSends = 0
			} else {
				state.failedSends++
				// Mark for removal if exceeded max failed sends
				if state.failedSends >= b.maxFailedSends {
					toRemove = append(toRemove, ch)
				}
			}
		}

		// Remove unhealthy clients
		for _, ch := range toRemove {
			if state, exists := channels[ch]; exists {
				delete(channels, ch)
				close(state.ch)
			}
		}

		// Clean up empty user map
		if len(channels) == 0 {
			delete(b.clients, userID)
		}
	}
}

// SendToAll broadcasts to all connected users with health tracking
func (b *SSEBroadcaster) SendToAll(event SSEEvent) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var usersToCleanup []string
	for userID, channels := range b.clients {
		var toRemove []chan SSEEvent
		for ch, state := range channels {
			sent := b.trySendWithTimeout(ch, event)
			if sent {
				state.lastActivity = time.Now()
				state.failedSends = 0
			} else {
				state.failedSends++
				// Mark for removal if exceeded max failed sends
				if state.failedSends >= b.maxFailedSends {
					toRemove = append(toRemove, ch)
				}
			}
		}

		// Remove unhealthy clients
		for _, ch := range toRemove {
			if state, exists := channels[ch]; exists {
				delete(channels, ch)
				close(state.ch)
			}
		}

		// Mark user for cleanup if no clients left
		if len(channels) == 0 {
			usersToCleanup = append(usersToCleanup, userID)
		}
	}

	// Clean up empty users
	for _, userID := range usersToCleanup {
		delete(b.clients, userID)
	}
}

// trySendWithTimeout attempts to send with timeout, returns true if successful
func (b *SSEBroadcaster) trySendWithTimeout(ch chan SSEEvent, event SSEEvent) bool {
	timer := time.NewTimer(b.sendTimeout)
	defer timer.Stop()

	select {
	case ch <- event:
		return true
	case <-timer.C:
		return false
	default:
		// Channel full, non-blocking send failed
		return false
	}
}

// cleanupLoop periodically removes inactive clients
func (b *SSEBroadcaster) cleanupLoop() {
	defer b.wg.Done()
	ticker := time.NewTicker(b.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			b.cleanupInactiveClients()
		}
	}
}

// cleanupInactiveClients removes clients that haven't received messages in a while
func (b *SSEBroadcaster) cleanupInactiveClients() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	var usersToCleanup []string

	for userID, channels := range b.clients {
		var toRemove []chan SSEEvent
		for ch, state := range channels {
			if now.Sub(state.lastActivity) > b.inactiveTimeout {
				toRemove = append(toRemove, ch)
			}
		}

		// Remove inactive clients
		for _, ch := range toRemove {
			if state, exists := channels[ch]; exists {
				delete(channels, ch)
				close(state.ch)
			}
		}

		// Mark user for cleanup if no clients left
		if len(channels) == 0 {
			usersToCleanup = append(usersToCleanup, userID)
		}
	}

	// Clean up empty users
	for _, userID := range usersToCleanup {
		delete(b.clients, userID)
	}
}

// IsConnected checks if a user has active connections
func (b *SSEBroadcaster) IsConnected(userID string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	channels, ok := b.clients[userID]
	return ok && len(channels) > 0
}

// ConnectedCount returns number of connected users
func (b *SSEBroadcaster) ConnectedCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// ClientCount returns total number of client connections across all users
func (b *SSEBroadcaster) ClientCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	count := 0
	for _, channels := range b.clients {
		count += len(channels)
	}
	return count
}

// Close shuts down the broadcaster and cleanup goroutine
func (b *SSEBroadcaster) Close() {
	b.cancel()
	b.wg.Wait()

	// Close all client channels
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, channels := range b.clients {
		for _, state := range channels {
			close(state.ch)
		}
	}
	b.clients = make(map[string]map[chan SSEEvent]*clientState)
}
