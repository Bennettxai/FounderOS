package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// replayBufferSize is the number of events retained per app for reconnecting subscribers.
const replayBufferSize = 100

// criticalEventTypes are events that must never be silently dropped.
// These signal terminal states — if lost, the client SSE stream hangs forever.
var criticalEventTypes = map[string]bool{
	"generation_complete": true,
	"error":               true,
	"generation_failed":   true,
}

// EventPersister is an optional interface for persisting critical events to durable storage.
// Implement this to survive server restarts during generation.
type EventPersister interface {
	PersistEvent(ctx context.Context, event BuildEvent) error
}

// BuildEvent represents a build progress event to be streamed to clients
type BuildEvent struct {
	ID              uuid.UUID              `json:"id"`
	AppID           uuid.UUID              `json:"app_id"`
	WorkspaceID     *uuid.UUID             `json:"workspace_id,omitempty"`
	EventType       string                 `json:"event_type"`
	Phase           string                 `json:"phase,omitempty"`
	ProgressPercent int                    `json:"progress_percent"`
	StatusMessage   string                 `json:"status_message,omitempty"`
	Data            map[string]interface{} `json:"data,omitempty"`
	Timestamp       time.Time              `json:"timestamp"`
}

// BuildEventSubscriber represents a client subscribed to build events
type BuildEventSubscriber struct {
	ID        string
	AppID     uuid.UUID
	UserID    uuid.UUID
	Events    chan BuildEvent
	ctx       context.Context
	cancelFn  context.CancelFunc
	done      chan struct{} // closed when subscriber is unsubscribed
	closeOnce sync.Once     // ensures done is closed only once
}

// Done returns a channel that is closed when the subscriber is unsubscribed.
// SSE handlers can select on this to detect clean shutdown.
func (s *BuildEventSubscriber) Done() <-chan struct{} {
	return s.done
}

// BuildEventBus manages pub/sub for build progress events
type BuildEventBus struct {
	subscribers map[string]*BuildEventSubscriber // subscriber ID -> subscriber
	mu          sync.RWMutex
	logger      *slog.Logger
	persister   EventPersister // Optional — nil means no persistence

	// Per-app event replay buffer (last replayBufferSize events) for reconnect support
	replayMu     sync.RWMutex
	replayBuffer map[uuid.UUID][]BuildEvent

	// Delivery metrics (atomic for lock-free reads)
	totalPublished atomic.Int64
	totalDelivered atomic.Int64
	totalDropped   atomic.Int64
	criticalDrops  atomic.Int64
}

// NewBuildEventBus creates a new build event bus
func NewBuildEventBus(logger *slog.Logger) *BuildEventBus {
	if logger == nil {
		logger = slog.Default()
	}
	return &BuildEventBus{
		subscribers:  make(map[string]*BuildEventSubscriber),
		replayBuffer: make(map[uuid.UUID][]BuildEvent),
		logger:       logger.With("component", "build_event_bus"),
	}
}

// Subscribe creates a new subscription for build events for a specific app.
// Returns a subscriber that can receive events via the Events channel.
// If events were published before this call (e.g. on reconnect), they are
// replayed asynchronously from the per-app ring buffer.
func (b *BuildEventBus) Subscribe(ctx context.Context, userID, appID uuid.UUID) *BuildEventSubscriber {
	// Create cancellable context for this subscription
	subCtx, cancel := context.WithCancel(ctx)

	subscriber := &BuildEventSubscriber{
		ID:       uuid.New().String(),
		AppID:    appID,
		UserID:   userID,
		Events:   make(chan BuildEvent, 100), // Buffered channel to prevent blocking
		ctx:      subCtx,
		cancelFn: cancel,
		done:     make(chan struct{}),
	}

	b.mu.Lock()
	b.subscribers[subscriber.ID] = subscriber
	subCount := len(b.subscribers)
	b.mu.Unlock()

	b.logger.Info("client subscribed to build events",
		"subscriber_id", subscriber.ID,
		"user_id", userID,
		"app_id", appID,
		"total_subscribers", subCount,
	)

	// Replay missed events from ring buffer (for reconnecting clients)
	b.replayMu.RLock()
	missed := make([]BuildEvent, len(b.replayBuffer[appID]))
	copy(missed, b.replayBuffer[appID])
	b.replayMu.RUnlock()

	if len(missed) > 0 {
		go func() {
			for _, evt := range missed {
				select {
				case subscriber.Events <- evt:
				case <-subscriber.done:
					return
				case <-subscriber.ctx.Done():
					return
				}
			}
		}()
		b.logger.Info("replayed missed events to reconnecting subscriber",
			"subscriber_id", subscriber.ID,
			"app_id", appID,
			"events_replayed", len(missed),
		)
	}

	// Start cleanup goroutine
	go b.cleanupOnContextDone(subscriber)

	return subscriber
}

// Unsubscribe removes a subscriber from the bus.
//
// IMPORTANT: The Events channel is NOT closed here. Closing it while Publish
// may be holding a reference (after releasing RLock but before the send) would
// cause a send-on-closed-channel panic. Instead, we signal the done channel so
// that Publish's select cases can observe the unsubscription safely. The Events
// channel is garbage-collected when all references drop.
func (b *BuildEventBus) Unsubscribe(subscriberID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if sub, exists := b.subscribers[subscriberID]; exists {
		// Signal done exactly once — safe to call multiple times
		sub.closeOnce.Do(func() { close(sub.done) })
		// Cancel the derived context
		sub.cancelFn()
		// Remove from map; Events channel is intentionally NOT closed here
		delete(b.subscribers, subscriberID)

		b.logger.Info("client unsubscribed from build events",
			"subscriber_id", subscriberID,
			"app_id", sub.AppID,
			"total_subscribers", len(b.subscribers),
		)
	}
}

// Publish broadcasts a build event to all subscribers for the given app.
// Critical events (generation_complete, error, generation_failed) use a
// blocking send with a 5-second timeout to ensure they are never silently
// dropped — losing these causes SSE streams to hang forever.
//
// Subscribers are collected under RLock, then the lock is released before
// sending. This prevents livelock: if a subscriber's context cancels during
// the blocking send, cleanupOnContextDone calls Unsubscribe (needs write
// lock). Holding RLock while blocking would deadlock.
//
// Each select also checks sub.done so that an unsubscribed subscriber
// (whose context may still be live briefly) is skipped immediately without
// a send-on-closed-channel panic.
func (b *BuildEventBus) Publish(event BuildEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	isCritical := criticalEventTypes[event.EventType]
	b.totalPublished.Add(1)

	// Collect matching subscribers under read lock, then release before sending.
	b.mu.RLock()
	var targets []*BuildEventSubscriber
	for _, sub := range b.subscribers {
		if sub.AppID == event.AppID {
			targets = append(targets, sub)
		}
	}
	totalSubs := len(b.subscribers)
	b.mu.RUnlock()

	subscriberCount := 0
	droppedCount := 0

	for _, sub := range targets {
		if isCritical {
			// Critical events: blocking send with timeout — must not be lost
			select {
			case sub.Events <- event:
				subscriberCount++
			case <-sub.done:
				// Subscriber was unsubscribed between RUnlock and here — safe exit
			case <-sub.ctx.Done():
				b.logger.Debug("skipping cancelled subscriber for critical event",
					"subscriber_id", sub.ID,
					"app_id", event.AppID,
					"event_type", event.EventType,
				)
			case <-time.After(5 * time.Second):
				b.logger.Error("CRITICAL: timed out sending critical event to subscriber — stream may hang",
					"subscriber_id", sub.ID,
					"app_id", event.AppID,
					"event_type", event.EventType,
				)
				droppedCount++
				b.criticalDrops.Add(1)
			}
		} else {
			// Non-critical events: best-effort delivery
			select {
			case sub.Events <- event:
				subscriberCount++
			case <-sub.done:
				// Subscriber was unsubscribed — safe exit, no panic
			case <-sub.ctx.Done():
				b.logger.Debug("skipping cancelled subscriber",
					"subscriber_id", sub.ID,
					"app_id", event.AppID,
				)
			default:
				droppedCount++
			}
		}
	}

	b.totalDelivered.Add(int64(subscriberCount))
	b.totalDropped.Add(int64(droppedCount))

	logLevel := slog.LevelInfo
	if droppedCount > 0 {
		logLevel = slog.LevelWarn
	}
	b.logger.Log(context.Background(), logLevel, "published build event",
		"app_id", event.AppID,
		"event_type", event.EventType,
		"phase", event.Phase,
		"progress", event.ProgressPercent,
		"critical", isCritical,
		"subscribers_notified", subscriberCount,
		"dropped", droppedCount,
		"total_subscribers", totalSubs,
	)

	// Store in per-app ring buffer for reconnect replay.
	// Copy to a new slice to release the old backing array for GC.
	b.replayMu.Lock()
	buf := b.replayBuffer[event.AppID]
	buf = append(buf, event)
	if len(buf) > replayBufferSize {
		trimmed := make([]BuildEvent, replayBufferSize)
		copy(trimmed, buf[len(buf)-replayBufferSize:])
		buf = trimmed
	}
	b.replayBuffer[event.AppID] = buf
	b.replayMu.Unlock()

	// Persist critical events asynchronously for durability
	if isCritical && b.persister != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := b.persister.PersistEvent(ctx, event); err != nil {
				b.logger.Error("failed to persist critical event",
					"event_type", event.EventType,
					"app_id", event.AppID,
					"error", err,
				)
			}
		}()
	}
}

// cleanupOnContextDone removes subscriber when context is cancelled
func (b *BuildEventBus) cleanupOnContextDone(sub *BuildEventSubscriber) {
	<-sub.ctx.Done()
	b.Unsubscribe(sub.ID)
}

// ClearReplayBuffer removes replay events for an app.
// Call this after a generation_complete event so the buffer does not grow
// indefinitely and stale events are not replayed to future subscribers.
func (b *BuildEventBus) ClearReplayBuffer(appID uuid.UUID) {
	b.replayMu.Lock()
	defer b.replayMu.Unlock()
	delete(b.replayBuffer, appID)
}

// SetPersister sets the optional event persister for critical event durability.
// Safe to call at any time; nil disables persistence.
func (b *BuildEventBus) SetPersister(p EventPersister) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.persister = p
}

// GetMetrics returns event bus delivery metrics.
// criticalDrops should always be 0 in healthy operation.
func (b *BuildEventBus) GetMetrics() map[string]int64 {
	return map[string]int64{
		"total_published": b.totalPublished.Load(),
		"total_delivered": b.totalDelivered.Load(),
		"total_dropped":   b.totalDropped.Load(),
		"critical_drops":  b.criticalDrops.Load(),
	}
}

// GetSubscriberCount returns the current number of active subscribers
func (b *BuildEventBus) GetSubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscribers)
}

// GetSubscriberCountForApp returns the number of subscribers for a specific app
func (b *BuildEventBus) GetSubscriberCountForApp(appID uuid.UUID) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	count := 0
	for _, sub := range b.subscribers {
		if sub.AppID == appID {
			count++
		}
	}
	return count
}

// FormatSSE formats a BuildEvent as a Server-Sent Event message
func FormatSSE(event BuildEvent) string {
	data, err := json.Marshal(event)
	if err != nil {
		return ""
	}
	return "data: " + string(data) + "\n\n"
}

// SendHeartbeat sends a heartbeat/keep-alive event
func SendHeartbeat() string {
	return ": heartbeat\n\n"
}
