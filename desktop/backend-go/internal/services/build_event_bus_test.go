package services

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBuildEventBus_PublishAndSubscribe(t *testing.T) {
	logger := slog.Default()
	bus := NewBuildEventBus(logger)

	ctx := context.Background()
	appID := uuid.New()
	userID := uuid.New()

	// Subscribe to events
	sub := bus.Subscribe(ctx, userID, appID)
	assert.NotNil(t, sub)
	assert.NotEmpty(t, sub.ID)

	// Verify subscriber count
	assert.Equal(t, 1, bus.GetSubscriberCount())
	assert.Equal(t, 1, bus.GetSubscriberCountForApp(appID))

	// Publish an event
	event := BuildEvent{
		ID:              uuid.New(),
		AppID:           appID,
		EventType:       "build_progress",
		Phase:           "building",
		ProgressPercent: 50,
		StatusMessage:   "Building project...",
		Timestamp:       time.Now(),
	}
	bus.Publish(event)

	// Receive the event
	select {
	case received := <-sub.Events:
		assert.Equal(t, event.ID, received.ID)
		assert.Equal(t, event.AppID, received.AppID)
		assert.Equal(t, event.EventType, received.EventType)
		assert.Equal(t, event.ProgressPercent, received.ProgressPercent)
		assert.Equal(t, event.StatusMessage, received.StatusMessage)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	// Unsubscribe
	bus.Unsubscribe(sub.ID)
	assert.Equal(t, 0, bus.GetSubscriberCount())
}

func TestBuildEventBus_MultipleSubscribers(t *testing.T) {
	logger := slog.Default()
	bus := NewBuildEventBus(logger)

	ctx := context.Background()
	appID := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()

	// Subscribe two clients
	sub1 := bus.Subscribe(ctx, user1, appID)
	sub2 := bus.Subscribe(ctx, user2, appID)

	assert.Equal(t, 2, bus.GetSubscriberCount())
	assert.Equal(t, 2, bus.GetSubscriberCountForApp(appID))

	// Publish event
	event := BuildEvent{
		ID:              uuid.New(),
		AppID:           appID,
		EventType:       "build_started",
		ProgressPercent: 0,
		StatusMessage:   "Starting build...",
		Timestamp:       time.Now(),
	}
	bus.Publish(event)

	// Both subscribers should receive the event
	select {
	case received := <-sub1.Events:
		assert.Equal(t, event.ID, received.ID)
	case <-time.After(1 * time.Second):
		t.Fatal("sub1 timeout")
	}

	select {
	case received := <-sub2.Events:
		assert.Equal(t, event.ID, received.ID)
	case <-time.After(1 * time.Second):
		t.Fatal("sub2 timeout")
	}

	// Cleanup
	bus.Unsubscribe(sub1.ID)
	bus.Unsubscribe(sub2.ID)
	assert.Equal(t, 0, bus.GetSubscriberCount())
}

func TestBuildEventBus_FiltersByAppID(t *testing.T) {
	logger := slog.Default()
	bus := NewBuildEventBus(logger)

	ctx := context.Background()
	app1 := uuid.New()
	app2 := uuid.New()
	userID := uuid.New()

	// Subscribe to app1
	sub1 := bus.Subscribe(ctx, userID, app1)

	// Publish event for app2
	event := BuildEvent{
		ID:              uuid.New(),
		AppID:           app2,
		EventType:       "build_progress",
		ProgressPercent: 50,
		StatusMessage:   "Building...",
		Timestamp:       time.Now(),
	}
	bus.Publish(event)

	// Subscriber should NOT receive event for different app
	select {
	case <-sub1.Events:
		t.Fatal("received event for wrong app")
	case <-time.After(100 * time.Millisecond):
		// Expected timeout
	}

	// Publish event for app1
	event.AppID = app1
	bus.Publish(event)

	// Now subscriber SHOULD receive event
	select {
	case received := <-sub1.Events:
		assert.Equal(t, app1, received.AppID)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for event")
	}

	bus.Unsubscribe(sub1.ID)
}

func TestBuildEventBus_ContextCancellation(t *testing.T) {
	logger := slog.Default()
	bus := NewBuildEventBus(logger)

	ctx, cancel := context.WithCancel(context.Background())
	appID := uuid.New()
	userID := uuid.New()

	// Subscribe
	_ = bus.Subscribe(ctx, userID, appID)
	assert.Equal(t, 1, bus.GetSubscriberCount())

	// Cancel context
	cancel()

	// Wait for cleanup
	time.Sleep(100 * time.Millisecond)

	// Subscriber should be automatically removed
	assert.Equal(t, 0, bus.GetSubscriberCount())
}

func TestBuildEventBus_BufferedChannel(t *testing.T) {
	logger := slog.Default()
	bus := NewBuildEventBus(logger)

	ctx := context.Background()
	appID := uuid.New()
	userID := uuid.New()

	// Subscribe
	sub := bus.Subscribe(ctx, userID, appID)

	// Publish many events (more than buffer size)
	for i := 0; i < 150; i++ {
		event := BuildEvent{
			ID:              uuid.New(),
			AppID:           appID,
			EventType:       "build_progress",
			ProgressPercent: i,
			Timestamp:       time.Now(),
		}
		bus.Publish(event)
	}

	// Should receive at least 100 events (buffer size)
	receivedCount := 0
	timeout := time.After(2 * time.Second)

	for {
		select {
		case <-sub.Events:
			receivedCount++
		case <-timeout:
			// Expected timeout after draining buffer
			assert.GreaterOrEqual(t, receivedCount, 100, "should receive at least buffer size")
			bus.Unsubscribe(sub.ID)
			return
		}
	}
}

func TestFormatSSE(t *testing.T) {
	event := BuildEvent{
		ID:              uuid.New(),
		AppID:           uuid.New(),
		EventType:       "build_progress",
		Phase:           "building",
		ProgressPercent: 75,
		StatusMessage:   "Compiling TypeScript...",
		Timestamp:       time.Now(),
	}

	sseMessage := FormatSSE(event)

	// Should start with "data: " and end with "\n\n"
	assert.Contains(t, sseMessage, "data: ")
	assert.Contains(t, sseMessage, "\n\n")

	// Should contain JSON with event fields
	assert.Contains(t, sseMessage, "build_progress")
	assert.Contains(t, sseMessage, "75")
	assert.Contains(t, sseMessage, "Compiling TypeScript...")
}

func TestSendHeartbeat(t *testing.T) {
	heartbeat := SendHeartbeat()

	// Should be a comment line (starts with :) followed by \n\n
	assert.Equal(t, ": heartbeat\n\n", heartbeat)
}

// TestBuildEventBus_UnsubscribePublishRace verifies that concurrent Publish
// and Unsubscribe calls do not cause a send-on-closed-channel panic.
// This is the critical safety property: Unsubscribe must NOT close sub.Events.
func TestBuildEventBus_UnsubscribePublishRace(t *testing.T) {
	logger := slog.Default()
	bus := NewBuildEventBus(logger)

	ctx := context.Background()
	appID := uuid.New()
	userID := uuid.New()

	// Run many iterations to expose the race reliably
	const iterations = 500

	for i := 0; i < iterations; i++ {
		sub := bus.Subscribe(ctx, userID, appID)

		var wg sync.WaitGroup
		wg.Add(2)

		// Goroutine 1: publish events rapidly
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				bus.Publish(BuildEvent{
					ID:        uuid.New(),
					AppID:     appID,
					EventType: "build_progress",
					Timestamp: time.Now(),
				})
			}
		}()

		// Goroutine 2: unsubscribe concurrently
		go func() {
			defer wg.Done()
			bus.Unsubscribe(sub.ID)
		}()

		wg.Wait()
	}
	// If we reach here without panicking, the test passes.
}

// TestBuildEventBus_ReplayOnReconnect verifies that a new subscriber receives
// events that were published before it subscribed (reconnect scenario).
func TestBuildEventBus_ReplayOnReconnect(t *testing.T) {
	logger := slog.Default()
	bus := NewBuildEventBus(logger)

	ctx := context.Background()
	appID := uuid.New()
	userID := uuid.New()

	// Publish events BEFORE any subscriber exists
	const numEvents = 5
	publishedIDs := make([]uuid.UUID, numEvents)
	for i := 0; i < numEvents; i++ {
		id := uuid.New()
		publishedIDs[i] = id
		bus.Publish(BuildEvent{
			ID:              id,
			AppID:           appID,
			EventType:       "build_progress",
			ProgressPercent: i * 20,
			Timestamp:       time.Now(),
		})
	}

	// Now subscribe — simulates a client reconnecting mid-generation
	sub := bus.Subscribe(ctx, userID, appID)
	defer bus.Unsubscribe(sub.ID)

	// The subscriber should receive all replayed events
	received := make(map[uuid.UUID]bool)
	timeout := time.After(2 * time.Second)

	for len(received) < numEvents {
		select {
		case evt := <-sub.Events:
			received[evt.ID] = true
		case <-timeout:
			t.Fatalf("timeout: only received %d/%d replayed events", len(received), numEvents)
		}
	}

	// Verify all published event IDs were replayed
	for _, id := range publishedIDs {
		assert.True(t, received[id], "expected event %s to be replayed", id)
	}
}

// TestBuildEventBus_DoneClosedAfterUnsubscribe verifies that the Done()
// channel is closed after Unsubscribe is called, allowing SSE handlers to
// detect clean shutdown without blocking on sub.Events.
func TestBuildEventBus_DoneClosedAfterUnsubscribe(t *testing.T) {
	logger := slog.Default()
	bus := NewBuildEventBus(logger)

	ctx := context.Background()
	appID := uuid.New()
	userID := uuid.New()

	sub := bus.Subscribe(ctx, userID, appID)

	// Done() channel must be open initially
	select {
	case <-sub.Done():
		t.Fatal("Done() channel should not be closed before Unsubscribe")
	default:
		// correct — channel is open
	}

	// Unsubscribe
	bus.Unsubscribe(sub.ID)

	// Done() channel must be closed now
	select {
	case <-sub.Done():
		// correct — channel was closed by Unsubscribe
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Done() channel was not closed after Unsubscribe")
	}

	// Calling Unsubscribe again must not panic (closeOnce guard)
	assert.NotPanics(t, func() {
		bus.Unsubscribe(sub.ID)
	})
}

// TestBuildEventBus_ClearReplayBuffer verifies that ClearReplayBuffer removes
// stored events so future subscribers do not receive stale replay data.
func TestBuildEventBus_ClearReplayBuffer(t *testing.T) {
	logger := slog.Default()
	bus := NewBuildEventBus(logger)

	ctx := context.Background()
	appID := uuid.New()
	userID := uuid.New()

	// Publish an event (goes into replay buffer)
	bus.Publish(BuildEvent{
		ID:        uuid.New(),
		AppID:     appID,
		EventType: "generation_complete",
		Timestamp: time.Now(),
	})

	// Clear the buffer (as we would after generation completes)
	bus.ClearReplayBuffer(appID)

	// New subscriber must NOT receive any replayed events
	sub := bus.Subscribe(ctx, userID, appID)
	defer bus.Unsubscribe(sub.ID)

	select {
	case evt := <-sub.Events:
		t.Fatalf("received unexpected replayed event after ClearReplayBuffer: %s", evt.EventType)
	case <-time.After(150 * time.Millisecond):
		// correct — no replay events delivered
	}
}
