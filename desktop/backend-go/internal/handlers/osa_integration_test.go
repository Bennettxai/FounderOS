package handlers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/services"
	"github.com/stretchr/testify/assert"
)

// TestOSAIntegration_EventBusToSSE tests the complete flow:
// 1. Client connects to SSE endpoint
// 2. Event is published to event bus
// 3. SSE client receives the event
func TestOSAIntegration_EventBusToSSE(t *testing.T) {
	logger := slog.Default()

	// Create event bus
	eventBus := services.NewBuildEventBus(logger)

	// Create handlers
	streamingHandler := NewOSAStreamingHandler(eventBus, logger)

	// Setup router
	gin.SetMode(gin.TestMode)
	router := gin.New()

	userID := uuid.New()
	appID := uuid.New()

	// Auth middleware
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})

	// Register routes
	router.GET("/stream/build/:app_id", streamingHandler.StreamBuildProgress)

	// Create test server
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Step 1: Start SSE client
	sseURL := ts.URL + "/stream/build/" + appID.String()
	req, err := http.NewRequest("GET", sseURL, nil)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	// Collect received events
	receivedEvents := make([]services.BuildEvent, 0)
	var mu sync.Mutex
	sseReady := make(chan bool, 1)

	go func() {
		buf := make([]byte, 4096)
		connectionEstablished := false
		for {
			n, err := resp.Body.Read(buf)
			if err != nil {
				if err != io.EOF {
					logger.Error("SSE read error", "error", err)
				}
				return
			}
			if n > 0 {
				data := string(buf[:n])

				// Signal when connection is established
				if !connectionEstablished && strings.Contains(data, "connected") {
					connectionEstablished = true
					sseReady <- true
				}

				// Parse SSE data lines
				lines := strings.Split(data, "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "data: ") {
						jsonData := strings.TrimPrefix(line, "data: ")
						// Skip connected messages (they have "type" not "event_type")
						if strings.Contains(jsonData, `"type":"connected"`) {
							continue
						}
						var event services.BuildEvent
						if err := json.Unmarshal([]byte(jsonData), &event); err == nil {
							// Only add build events with non-nil AppID and actual EventType
							if event.AppID != uuid.Nil && event.EventType != "" {
								mu.Lock()
								receivedEvents = append(receivedEvents, event)
								mu.Unlock()
							}
						}
					}
				}
			}
		}
	}()

	// Wait for SSE connection to be established
	select {
	case <-sseReady:
		logger.Info("SSE connection established")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for SSE connection")
	}

	// Small delay to ensure subscription is fully registered in event bus
	time.Sleep(100 * time.Millisecond)

	// Step 2: Publish event directly to event bus (simulating webhook behavior)
	testEvent := services.BuildEvent{
		ID:              uuid.New(),
		AppID:           appID,
		EventType:       "build_progress",
		Phase:           "building",
		ProgressPercent: 50,
		StatusMessage:   "Installing dependencies...",
		Timestamp:       time.Now(),
	}
	eventBus.Publish(testEvent)

	// Step 3: Wait for SSE client to receive event
	deadline := time.After(3 * time.Second)
	eventReceived := false

	for !eventReceived {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for event to be received via SSE")
		case <-time.After(100 * time.Millisecond):
			mu.Lock()
			if len(receivedEvents) > 0 {
				// Verify event content
				event := receivedEvents[0]
				assert.Equal(t, appID, event.AppID)
				assert.Equal(t, "build_progress", event.EventType)
				assert.Equal(t, 50, event.ProgressPercent)
				eventReceived = true
			}
			mu.Unlock()
		}
	}

	// Cleanup
	cancel()
}

// TestOSAIntegration_MultipleClients tests multiple SSE clients receiving the same event
func TestOSAIntegration_MultipleClients(t *testing.T) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)
	streamingHandler := NewOSAStreamingHandler(eventBus, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	userID := uuid.New()
	appID := uuid.New()

	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})

	router.GET("/stream/build/:app_id", streamingHandler.StreamBuildProgress)

	ts := httptest.NewServer(router)
	defer ts.Close()

	// Create 3 concurrent SSE clients
	numClients := 3
	clientsReady := make(chan bool, numClients)
	clientEvents := make([][]services.BuildEvent, numClients)
	var clientMutexes []*sync.Mutex

	for i := 0; i < numClients; i++ {
		clientEvents[i] = make([]services.BuildEvent, 0)
		clientMutexes = append(clientMutexes, &sync.Mutex{})
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for clientIdx := 0; clientIdx < numClients; clientIdx++ {
		idx := clientIdx
		go func() {
			url := ts.URL + "/stream/build/" + appID.String()
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return
			}
			req = req.WithContext(ctx)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			clientsReady <- true

			buf := make([]byte, 4096)
			for {
				n, err := resp.Body.Read(buf)
				if err != nil {
					return
				}
				if n > 0 {
					data := string(buf[:n])
					lines := strings.Split(data, "\n")
					for _, line := range lines {
						if strings.HasPrefix(line, "data: ") {
							jsonData := strings.TrimPrefix(line, "data: ")
							// Skip connected messages
							if strings.Contains(jsonData, `"type":"connected"`) {
								continue
							}
							var event services.BuildEvent
							if err := json.Unmarshal([]byte(jsonData), &event); err == nil {
								// Only add build events with non-nil AppID and actual EventType
								if event.AppID != uuid.Nil && event.EventType != "" {
									clientMutexes[idx].Lock()
									clientEvents[idx] = append(clientEvents[idx], event)
									clientMutexes[idx].Unlock()
								}
							}
						}
					}
				}
			}
		}()
	}

	// Wait for all clients to connect
	for i := 0; i < numClients; i++ {
		select {
		case <-clientsReady:
			// Client connected
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for client to connect")
		}
	}

	// Give clients time to fully establish connection
	time.Sleep(200 * time.Millisecond)

	// Publish a single event
	testEvent := services.BuildEvent{
		ID:              uuid.New(),
		AppID:           appID,
		EventType:       "build_complete",
		Phase:           "complete",
		ProgressPercent: 100,
		StatusMessage:   "Build completed successfully",
		Timestamp:       time.Now(),
	}
	eventBus.Publish(testEvent)

	// Wait for all clients to receive the event
	time.Sleep(500 * time.Millisecond)

	// Verify all clients received the event
	for i := 0; i < numClients; i++ {
		clientMutexes[i].Lock()
		assert.GreaterOrEqual(t, len(clientEvents[i]), 1,
			"client %d should have received at least one event", i)

		if len(clientEvents[i]) > 0 {
			event := clientEvents[i][0]
			assert.Equal(t, appID, event.AppID)
			assert.Equal(t, "build_complete", event.EventType)
			assert.Equal(t, 100, event.ProgressPercent)
		}
		clientMutexes[i].Unlock()
	}

	cancel()
}

// BenchmarkWebhookToSSELatency benchmarks end-to-end latency
func BenchmarkWebhookToSSELatency(b *testing.B) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)

	appID := uuid.New()
	userID := uuid.New()
	ctx := context.Background()

	// Subscribe
	sub := eventBus.Subscribe(ctx, userID, appID)
	defer eventBus.Unsubscribe(sub.ID)

	// Drain events
	go func() {
		for range sub.Events {
			// Measure in main loop
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()

		event := services.BuildEvent{
			ID:              uuid.New(),
			AppID:           appID,
			EventType:       "build_progress",
			ProgressPercent: i % 100,
			Timestamp:       time.Now(),
		}

		eventBus.Publish(event)

		// Wait for event
		select {
		case <-sub.Events:
			_ = time.Since(start)
		case <-time.After(1 * time.Second):
			b.Fatal("timeout")
		}
	}
}
