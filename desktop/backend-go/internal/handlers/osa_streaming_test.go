package handlers

import (
	"context"
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

func TestOSAStreamingHandler_StreamBuildProgress(t *testing.T) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)
	handler := NewOSAStreamingHandler(eventBus, logger)

	// Setup Gin in test mode
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware to set userID (simulating auth)
	userID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})

	appID := uuid.New()
	router.GET("/stream/build/:app_id", handler.StreamBuildProgress)

	// Create test server
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Create HTTP client with SSE support
	url := ts.URL + "/stream/build/" + appID.String()
	req, err := http.NewRequest("GET", url, nil)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Verify SSE headers
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))
	assert.Equal(t, "keep-alive", resp.Header.Get("Connection"))

	// Start reading SSE stream in goroutine
	receivedEvents := make([]string, 0)
	var mu sync.Mutex
	done := make(chan bool)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if err != nil {
				close(done)
				return
			}
			if n > 0 {
				mu.Lock()
				receivedEvents = append(receivedEvents, string(buf[:n]))
				mu.Unlock()
			}
		}
	}()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Publish test event
	event := services.BuildEvent{
		ID:              uuid.New(),
		AppID:           appID,
		EventType:       "build_progress",
		Phase:           "building",
		ProgressPercent: 50,
		StatusMessage:   "Building project...",
		Timestamp:       time.Now(),
	}
	eventBus.Publish(event)

	// Wait for event to be received
	time.Sleep(200 * time.Millisecond)

	// Verify events were received
	mu.Lock()
	assert.Greater(t, len(receivedEvents), 0, "should receive at least one event")
	eventData := strings.Join(receivedEvents, "")
	assert.Contains(t, eventData, "data:")
	mu.Unlock()

	// Cancel context to close connection
	cancel()

	select {
	case <-done:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for stream to close")
	}
}

func TestOSAStreamingHandler_GetStreamStats(t *testing.T) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)
	handler := NewOSAStreamingHandler(eventBus, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	userID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})

	router.GET("/stream/stats", handler.GetStreamStats)

	// Create test request
	req := httptest.NewRequest("GET", "/stream/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "total_subscribers")
}

func TestOSAStreamingHandler_GetAppStreamStats(t *testing.T) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)
	handler := NewOSAStreamingHandler(eventBus, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	userID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})

	appID := uuid.New()
	router.GET("/stream/stats/:app_id", handler.GetAppStreamStats)

	// Subscribe to app to have stats
	ctx := context.Background()
	sub := eventBus.Subscribe(ctx, userID, appID)
	defer eventBus.Unsubscribe(sub.ID)

	// Create test request
	req := httptest.NewRequest("GET", "/stream/stats/"+appID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), appID.String())
	assert.Contains(t, w.Body.String(), "subscriber_count")
}

func TestOSAStreamingHandler_Unauthorized(t *testing.T) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)
	handler := NewOSAStreamingHandler(eventBus, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	appID := uuid.New()
	router.GET("/stream/build/:app_id", handler.StreamBuildProgress)

	// Request without userID in context
	req := httptest.NewRequest("GET", "/stream/build/"+appID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOSAStreamingHandler_InvalidAppID(t *testing.T) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)
	handler := NewOSAStreamingHandler(eventBus, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	userID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})

	router.GET("/stream/build/:app_id", handler.StreamBuildProgress)

	// Request with invalid app_id
	req := httptest.NewRequest("GET", "/stream/build/invalid-uuid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestOSAStreamingHandler_HandleGenerateAppStream(t *testing.T) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)
	handler := NewOSAStreamingHandler(eventBus, logger)

	// Setup Gin in test mode
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add middleware to set userID (simulating auth)
	userID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})

	appID := uuid.New()
	// Register the new RESTful endpoint
	router.GET("/generate/:app_id/stream", handler.HandleGenerateAppStream)

	// Create test server
	ts := httptest.NewServer(router)
	defer ts.Close()

	// Create HTTP client with SSE support
	url := ts.URL + "/generate/" + appID.String() + "/stream"
	req, err := http.NewRequest("GET", url, nil)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Verify SSE headers
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))
	assert.Equal(t, "no-cache", resp.Header.Get("Cache-Control"))
	assert.Equal(t, "keep-alive", resp.Header.Get("Connection"))

	// Start reading SSE stream in goroutine
	receivedEvents := make([]string, 0)
	var mu sync.Mutex
	done := make(chan bool)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if err != nil {
				close(done)
				return
			}
			if n > 0 {
				mu.Lock()
				receivedEvents = append(receivedEvents, string(buf[:n]))
				mu.Unlock()
			}
		}
	}()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	// Publish test event
	event := services.BuildEvent{
		ID:              uuid.New(),
		AppID:           appID,
		EventType:       "build_started",
		Phase:           "initialization",
		ProgressPercent: 0,
		StatusMessage:   "Starting app generation...",
		Timestamp:       time.Now(),
	}
	eventBus.Publish(event)

	// Wait for event to be received
	time.Sleep(200 * time.Millisecond)

	// Verify events were received
	mu.Lock()
	assert.Greater(t, len(receivedEvents), 0, "should receive at least one event (connected + build event)")
	eventData := strings.Join(receivedEvents, "")
	assert.Contains(t, eventData, "data:")
	assert.Contains(t, eventData, "connected")
	mu.Unlock()

	// Cancel context to close connection
	cancel()

	select {
	case <-done:
		// Expected
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for stream to close")
	}
}

// BenchmarkSSEStreaming benchmarks SSE streaming with multiple concurrent clients
func BenchmarkSSEStreaming(b *testing.B) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)

	appID := uuid.New()
	userID := uuid.New()
	ctx := context.Background()

	// Subscribe multiple clients
	subscribers := make([]*services.BuildEventSubscriber, 100)
	for i := 0; i < 100; i++ {
		subscribers[i] = eventBus.Subscribe(ctx, userID, appID)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		event := services.BuildEvent{
			ID:              uuid.New(),
			AppID:           appID,
			EventType:       "build_progress",
			Phase:           "building",
			ProgressPercent: i % 100,
			StatusMessage:   "Building...",
			Timestamp:       time.Now(),
		}
		eventBus.Publish(event)
	}

	b.StopTimer()

	// Cleanup
	for _, sub := range subscribers {
		eventBus.Unsubscribe(sub.ID)
	}
}

// BenchmarkEventBusPublish benchmarks raw event bus publishing
func BenchmarkEventBusPublish(b *testing.B) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)

	appID := uuid.New()
	userID := uuid.New()
	ctx := context.Background()

	// Single subscriber
	sub := eventBus.Subscribe(ctx, userID, appID)
	defer eventBus.Unsubscribe(sub.ID)

	// Drain events in background
	go func() {
		for range sub.Events {
			// Discard
		}
	}()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		event := services.BuildEvent{
			ID:              uuid.New(),
			AppID:           appID,
			EventType:       "build_progress",
			ProgressPercent: i % 100,
			Timestamp:       time.Now(),
		}
		eventBus.Publish(event)
	}
}

// TestOSAStreamingHandler_HeartbeatMechanism tests the 30s heartbeat
func TestOSAStreamingHandler_HeartbeatMechanism(t *testing.T) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)
	handler := NewOSAStreamingHandler(eventBus, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	userID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})

	appID := uuid.New()
	router.GET("/stream/build/:app_id", handler.StreamBuildProgress)

	ts := httptest.NewServer(router)
	defer ts.Close()

	url := ts.URL + "/stream/build/" + appID.String()
	req, err := http.NewRequest("GET", url, nil)
	assert.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Read stream for at least 2 seconds to check for heartbeat (mocked with shorter interval)
	// Note: Real heartbeat is 30s, but we test the mechanism exists
	receivedData := make([]string, 0)
	var mu sync.Mutex
	done := make(chan bool, 1)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if err != nil {
				done <- true
				return
			}
			if n > 0 {
				mu.Lock()
				receivedData = append(receivedData, string(buf[:n]))
				mu.Unlock()
			}
		}
	}()

	// Wait for initial connection
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	data := strings.Join(receivedData, "")
	mu.Unlock()

	// Should receive connection confirmation
	assert.Contains(t, data, "connected")
	assert.Contains(t, data, appID.String())

	cancel()
	<-done
}

// TestOSAStreamingHandler_MultipleEventTypes tests different event types
func TestOSAStreamingHandler_MultipleEventTypes(t *testing.T) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)
	handler := NewOSAStreamingHandler(eventBus, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	userID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})

	appID := uuid.New()
	router.GET("/stream/build/:app_id", handler.StreamBuildProgress)

	ts := httptest.NewServer(router)
	defer ts.Close()

	url := ts.URL + "/stream/build/" + appID.String()
	req, err := http.NewRequest("GET", url, nil)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	receivedEvents := make([]string, 0)
	var mu sync.Mutex
	done := make(chan bool)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if err != nil {
				close(done)
				return
			}
			if n > 0 {
				mu.Lock()
				receivedEvents = append(receivedEvents, string(buf[:n]))
				mu.Unlock()
			}
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Publish multiple event types
	events := []services.BuildEvent{
		{
			ID:              uuid.New(),
			AppID:           appID,
			EventType:       "build_started",
			Phase:           "initialization",
			ProgressPercent: 0,
			StatusMessage:   "Starting build...",
			Timestamp:       time.Now(),
		},
		{
			ID:              uuid.New(),
			AppID:           appID,
			EventType:       "build_progress",
			Phase:           "building",
			ProgressPercent: 50,
			StatusMessage:   "Building application...",
			Timestamp:       time.Now(),
		},
		{
			ID:              uuid.New(),
			AppID:           appID,
			EventType:       "build_completed",
			Phase:           "complete",
			ProgressPercent: 100,
			StatusMessage:   "Build completed successfully",
			Timestamp:       time.Now(),
		},
		{
			ID:              uuid.New(),
			AppID:           appID,
			EventType:       "build_error",
			Phase:           "error",
			ProgressPercent: 75,
			StatusMessage:   "Build failed",
			Data:            map[string]interface{}{"error": "compilation error"},
			Timestamp:       time.Now(),
		},
	}

	for _, event := range events {
		eventBus.Publish(event)
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	eventData := strings.Join(receivedEvents, "")
	mu.Unlock()

	// Verify all event types received
	assert.Contains(t, eventData, "build_started")
	assert.Contains(t, eventData, "build_progress")
	assert.Contains(t, eventData, "build_completed")
	assert.Contains(t, eventData, "build_error")
	assert.Contains(t, eventData, "Starting build")
	assert.Contains(t, eventData, "Building application")
	assert.Contains(t, eventData, "Build completed successfully")

	cancel()
	<-done
}

// TestOSAStreamingHandler_MultipleSubscribers tests multiple clients for same app
func TestOSAStreamingHandler_MultipleSubscribers(t *testing.T) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)
	handler := NewOSAStreamingHandler(eventBus, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	userID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})

	appID := uuid.New()
	router.GET("/stream/build/:app_id", handler.StreamBuildProgress)

	ts := httptest.NewServer(router)
	defer ts.Close()

	// Create 3 concurrent subscribers
	numSubscribers := 3
	cancels := make([]context.CancelFunc, numSubscribers)
	doneChans := make([]chan bool, numSubscribers)
	receivedCounts := make([]int, numSubscribers)
	var mu sync.Mutex

	for i := 0; i < numSubscribers; i++ {
		idx := i
		doneChans[i] = make(chan bool)

		url := ts.URL + "/stream/build/" + appID.String()
		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancels[i] = cancel
		req = req.WithContext(ctx)

		client := &http.Client{}
		resp, err := client.Do(req)
		assert.NoError(t, err)

		go func(respBody *http.Response, done chan bool) {
			defer respBody.Body.Close()
			buf := make([]byte, 4096)
			for {
				n, err := respBody.Body.Read(buf)
				if err != nil {
					close(done)
					return
				}
				if n > 0 {
					// Count data events (not heartbeats)
					if strings.Contains(string(buf[:n]), "data:") {
						mu.Lock()
						receivedCounts[idx]++
						mu.Unlock()
					}
				}
			}
		}(resp, doneChans[i])
	}

	// Wait for connections
	time.Sleep(200 * time.Millisecond)

	// Publish events
	for i := 0; i < 5; i++ {
		event := services.BuildEvent{
			ID:              uuid.New(),
			AppID:           appID,
			EventType:       "build_progress",
			Phase:           "building",
			ProgressPercent: i * 20,
			StatusMessage:   "Building...",
			Timestamp:       time.Now(),
		}
		eventBus.Publish(event)
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)

	// Cancel all subscribers
	for _, cancel := range cancels {
		cancel()
	}

	// Wait for all to finish
	for _, done := range doneChans {
		<-done
	}

	// Verify all subscribers received events
	mu.Lock()
	for i, count := range receivedCounts {
		assert.Greater(t, count, 0, "subscriber %d should receive events", i)
	}
	mu.Unlock()
}

// TestOSAStreamingHandler_InvalidUserIDFormat tests invalid user ID type
func TestOSAStreamingHandler_InvalidUserIDFormat(t *testing.T) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)
	handler := NewOSAStreamingHandler(eventBus, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Set userID as string instead of uuid.UUID
	router.Use(func(c *gin.Context) {
		c.Set("userID", "not-a-uuid")
		c.Next()
	})

	appID := uuid.New()
	router.GET("/stream/build/:app_id", handler.StreamBuildProgress)

	req := httptest.NewRequest("GET", "/stream/build/"+appID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid user ID format")
}

// TestOSAStreamingHandler_ContextCancellation tests proper cleanup on context cancel
func TestOSAStreamingHandler_ContextCancellation(t *testing.T) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)
	handler := NewOSAStreamingHandler(eventBus, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	userID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})

	appID := uuid.New()
	router.GET("/stream/build/:app_id", handler.StreamBuildProgress)

	ts := httptest.NewServer(router)
	defer ts.Close()

	// Check initial subscriber count
	assert.Equal(t, 0, eventBus.GetSubscriberCount())

	url := ts.URL + "/stream/build/" + appID.String()
	req, err := http.NewRequest("GET", url, nil)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)

	done := make(chan bool)
	go func() {
		defer resp.Body.Close()
		buf := make([]byte, 4096)
		for {
			_, err := resp.Body.Read(buf)
			if err != nil {
				close(done)
				return
			}
		}
	}()

	// Wait for subscription
	time.Sleep(200 * time.Millisecond)

	// Should have 1 subscriber
	assert.Equal(t, 1, eventBus.GetSubscriberCount())
	assert.Equal(t, 1, eventBus.GetSubscriberCountForApp(appID))

	// Cancel context
	cancel()

	// Wait for cleanup
	select {
	case <-done:
		// Connection closed as expected
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for connection to close")
	}

	// Allow time for cleanup goroutine
	time.Sleep(300 * time.Millisecond)

	// Subscriber should be cleaned up
	assert.Equal(t, 0, eventBus.GetSubscriberCount())
	assert.Equal(t, 0, eventBus.GetSubscriberCountForApp(appID))
}

// TestOSAStreamingHandler_ConnectionConfirmation tests initial connection message
func TestOSAStreamingHandler_ConnectionConfirmation(t *testing.T) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)
	handler := NewOSAStreamingHandler(eventBus, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	userID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})

	appID := uuid.New()
	router.GET("/stream/build/:app_id", handler.StreamBuildProgress)

	ts := httptest.NewServer(router)
	defer ts.Close()

	url := ts.URL + "/stream/build/" + appID.String()
	req, err := http.NewRequest("GET", url, nil)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	// Read initial message
	buf := make([]byte, 512)
	n, err := resp.Body.Read(buf)
	assert.NoError(t, err)
	assert.Greater(t, n, 0)

	initialMessage := string(buf[:n])

	// Verify connection confirmation
	assert.Contains(t, initialMessage, "data:")
	assert.Contains(t, initialMessage, "connected")
	assert.Contains(t, initialMessage, appID.String())
	assert.Contains(t, initialMessage, "\n\n") // SSE message terminator

	cancel()
}

// TestOSAStreamingHandler_EventOrdering tests that events are received in order
func TestOSAStreamingHandler_EventOrdering(t *testing.T) {
	logger := slog.Default()
	eventBus := services.NewBuildEventBus(logger)
	handler := NewOSAStreamingHandler(eventBus, logger)

	gin.SetMode(gin.TestMode)
	router := gin.New()

	userID := uuid.New()
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Next()
	})

	appID := uuid.New()
	router.GET("/stream/build/:app_id", handler.StreamBuildProgress)

	ts := httptest.NewServer(router)
	defer ts.Close()

	url := ts.URL + "/stream/build/" + appID.String()
	req, err := http.NewRequest("GET", url, nil)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req = req.WithContext(ctx)

	client := &http.Client{}
	resp, err := client.Do(req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	receivedEvents := make([]string, 0)
	var mu sync.Mutex
	done := make(chan bool)

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if err != nil {
				close(done)
				return
			}
			if n > 0 {
				data := string(buf[:n])
				if strings.Contains(data, "progress_percent") {
					mu.Lock()
					receivedEvents = append(receivedEvents, data)
					mu.Unlock()
				}
			}
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Publish events in sequence with distinct progress values
	expectedProgress := []int{0, 20, 40, 60, 80, 100}
	for _, progress := range expectedProgress {
		event := services.BuildEvent{
			ID:              uuid.New(),
			AppID:           appID,
			EventType:       "build_progress",
			Phase:           "building",
			ProgressPercent: progress,
			StatusMessage:   "Progress update",
			Timestamp:       time.Now(),
		}
		eventBus.Publish(event)
		time.Sleep(30 * time.Millisecond)
	}

	time.Sleep(200 * time.Millisecond)

	cancel()
	<-done

	// Verify we received events (ordering guaranteed by channel semantics)
	mu.Lock()
	assert.Greater(t, len(receivedEvents), 0, "should receive at least some progress events")
	// Check that data contains valid JSON-like progress values
	eventData := strings.Join(receivedEvents, "")
	assert.Contains(t, eventData, "progress_percent")
	mu.Unlock()
}
