package services

import (
	"sync"
	"testing"
	"time"
)

func TestSSEBroadcaster_NonBlockingSend(t *testing.T) {
	b := NewSSEBroadcasterWithConfig(
		50*time.Millisecond, // sendTimeout
		2,                   // maxFailedSends
		1*time.Second,       // cleanupInterval
		2*time.Second,       // inactiveTimeout
	)
	defer b.Close()

	userID := "test-user"
	ch := b.Subscribe(userID)

	// Fill the buffer (capacity 16)
	for i := 0; i < 16; i++ {
		ch <- SSEEvent{Type: "fill", Data: i}
	}

	// This send should not block even though channel is full
	start := time.Now()
	b.SendToUser(userID, SSEEvent{Type: "test", Data: "should-not-block"})
	elapsed := time.Since(start)

	if elapsed > 100*time.Millisecond {
		t.Errorf("Send took too long: %v (expected < 100ms)", elapsed)
	}
}

func TestSSEBroadcaster_UnhealthyClientRemoval(t *testing.T) {
	b := NewSSEBroadcasterWithConfig(
		10*time.Millisecond, // sendTimeout
		2,                   // maxFailedSends (will be removed after 2 failures)
		100*time.Millisecond,
		1*time.Second,
	)
	defer b.Close()

	userID := "test-user"
	ch := b.Subscribe(userID)

	// Fill the buffer so sends fail
	for i := 0; i < 16; i++ {
		ch <- SSEEvent{Type: "fill", Data: i}
	}

	// First failed send
	b.SendToUser(userID, SSEEvent{Type: "test1", Data: "fail1"})
	if !b.IsConnected(userID) {
		t.Error("Client should still be connected after 1 failed send")
	}

	// Second failed send - should trigger removal
	b.SendToUser(userID, SSEEvent{Type: "test2", Data: "fail2"})
	if b.IsConnected(userID) {
		t.Error("Client should be removed after 2 failed sends")
	}
}

func TestSSEBroadcaster_HealthyClientPreserved(t *testing.T) {
	b := NewSSEBroadcasterWithConfig(
		50*time.Millisecond,
		3,
		100*time.Millisecond,
		1*time.Second,
	)
	defer b.Close()

	userID := "test-user"
	ch := b.Subscribe(userID)

	// Consume messages to keep channel healthy
	go func() {
		for range ch {
			// Drain channel
		}
	}()

	// Send multiple messages
	for i := 0; i < 10; i++ {
		b.SendToUser(userID, SSEEvent{Type: "test", Data: i})
	}

	time.Sleep(100 * time.Millisecond)

	if !b.IsConnected(userID) {
		t.Error("Healthy client should remain connected")
	}
}

func TestSSEBroadcaster_InactiveClientCleanup(t *testing.T) {
	b := NewSSEBroadcasterWithConfig(
		50*time.Millisecond,
		3,
		200*time.Millisecond, // cleanup every 200ms
		300*time.Millisecond, // inactive after 300ms
	)
	defer b.Close()

	userID := "test-user"
	ch := b.Subscribe(userID)

	// Don't consume or send any messages
	_ = ch

	if !b.IsConnected(userID) {
		t.Fatal("Client should be initially connected")
	}

	// Wait for cleanup to run (need > inactiveTimeout + cleanupInterval)
	time.Sleep(600 * time.Millisecond)

	if b.IsConnected(userID) {
		t.Error("Inactive client should be cleaned up")
	}
}

func TestSSEBroadcaster_SendToAll(t *testing.T) {
	b := NewSSEBroadcaster()
	defer b.Close()

	// Subscribe multiple users
	users := []string{"user1", "user2", "user3"}
	channels := make([]chan SSEEvent, len(users))
	for i, userID := range users {
		channels[i] = b.Subscribe(userID)
	}

	// Consume messages
	var wg sync.WaitGroup
	received := make([]int, len(users))
	for i, ch := range channels {
		wg.Add(1)
		go func(idx int, ch chan SSEEvent) {
			defer wg.Done()
			for range ch {
				received[idx]++
			}
		}(i, ch)
	}

	// Send to all
	b.SendToAll(SSEEvent{Type: "broadcast", Data: "test"})

	// Give time for messages to be received
	time.Sleep(50 * time.Millisecond)

	// Unsubscribe all
	for i, userID := range users {
		b.Unsubscribe(userID, channels[i])
	}

	wg.Wait()

	// Verify all users received the message
	for i, count := range received {
		if count != 1 {
			t.Errorf("User %d received %d messages, expected 1", i, count)
		}
	}
}

func TestSSEBroadcaster_ConcurrentAccess(t *testing.T) {
	b := NewSSEBroadcaster()
	defer b.Close()

	var wg sync.WaitGroup
	numGoroutines := 10

	// Concurrent subscribes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			userID := "user-" + string(rune('0'+id))
			ch := b.Subscribe(userID)
			defer b.Unsubscribe(userID, ch)

			// Drain messages
			go func() {
				for range ch {
				}
			}()

			// Send some messages
			for j := 0; j < 5; j++ {
				b.SendToUser(userID, SSEEvent{Type: "test", Data: j})
			}
		}(i)
	}

	wg.Wait()

	// All clients should be unsubscribed
	if b.ConnectedCount() != 0 {
		t.Errorf("Expected 0 connected users, got %d", b.ConnectedCount())
	}
}

func TestSSEBroadcaster_ClientCount(t *testing.T) {
	b := NewSSEBroadcaster()
	defer b.Close()

	// Subscribe multiple clients for same user
	userID := "test-user"
	ch1 := b.Subscribe(userID)
	ch2 := b.Subscribe(userID)

	if b.ConnectedCount() != 1 {
		t.Errorf("Expected 1 connected user, got %d", b.ConnectedCount())
	}

	if b.ClientCount() != 2 {
		t.Errorf("Expected 2 client connections, got %d", b.ClientCount())
	}

	b.Unsubscribe(userID, ch1)

	if b.ConnectedCount() != 1 {
		t.Errorf("Expected 1 connected user after partial unsubscribe, got %d", b.ConnectedCount())
	}

	if b.ClientCount() != 1 {
		t.Errorf("Expected 1 client connection after partial unsubscribe, got %d", b.ClientCount())
	}

	b.Unsubscribe(userID, ch2)

	if b.ConnectedCount() != 0 {
		t.Errorf("Expected 0 connected users after full unsubscribe, got %d", b.ConnectedCount())
	}
}

func TestSSEBroadcaster_Close(t *testing.T) {
	b := NewSSEBroadcaster()

	// Subscribe some clients
	ch1 := b.Subscribe("user1")
	ch2 := b.Subscribe("user2")

	// Close broadcaster
	b.Close()

	// Channels should be closed
	select {
	case _, ok := <-ch1:
		if ok {
			t.Error("Channel should be closed after Close()")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Channel should be closed immediately")
	}

	select {
	case _, ok := <-ch2:
		if ok {
			t.Error("Channel should be closed after Close()")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Channel should be closed immediately")
	}

	// No clients should remain
	if b.ConnectedCount() != 0 {
		t.Errorf("Expected 0 connected users after Close(), got %d", b.ConnectedCount())
	}
}
