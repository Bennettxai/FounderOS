// Package redis provides Redis connection management with worker pool for cache operations
//
// # Worker Pool for Bounded Concurrency
//
// The CacheWorkerPool provides a bounded worker pool to prevent cache operations
// from overwhelming the system. This is especially useful when:
//   - Handling burst traffic that could create thousands of concurrent Redis operations
//   - Need to control resource usage (connections, memory, CPU)
//   - Want visibility into cache operation metrics
//
// # Integration with SessionStore
//
// Example: Enable worker pool for session operations
//
//	// Configure session store with worker pool
//	cfg := &SessionStoreConfig{
//	    KeyPrefix:       "session:",
//	    TTL:             24 * time.Hour,
//	    HMACSecret:      os.Getenv("REDIS_KEY_HMAC_SECRET"),
//	    UseWorkerPool:   true,  // Enable worker pool
//	    WorkerPoolSize:  10,    // 10 concurrent workers
//	    WorkerQueueSize: 100,   // Queue up to 100 operations
//	}
//
//	store, err := NewSessionStore(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer store.Close(5 * time.Second)
//
//	// Monitor metrics
//	if metrics := store.GetWorkerPoolMetrics(); metrics != nil {
//	    slog.Info("Active workers: , Queue size: , Completed", "arg0", // metrics.ActiveWorkers, "arg1", metrics.QueueSize, "arg2", metrics.CompletedTasks)
//	}
//
// # Environment Variables
//
//   - REDIS_WORKER_POOL_ENABLED: Set to "true" to enable worker pool (default: false)
//   - REDIS_WORKER_POOL_SIZE: Number of workers (default: 10, recommended: 5-20)
//   - REDIS_WORKER_QUEUE_SIZE: Queue size (default: 100, recommended: 50-500)
//
// # Direct Usage (Advanced)
//
// For direct worker pool usage outside SessionStore:
//
//	pool := NewCacheWorkerPool(10, 100)
//	pool.Start()
//	defer pool.Shutdown(5 * time.Second)
//
//	// Submit a cache operation
//	ctx := context.Background()
//	resultCh := make(chan CacheResult, 1)
//	task := CacheTask{
//	    Operation: "set",
//	    Key:       "user:123",
//	    Value:     userData,
//	    TTL:       time.Hour,
//	    ResultCh:  resultCh,
//	    Context:   ctx,
//	}
//
//	if err := pool.Submit(task); err != nil {
//	    slog.Info("Failed to submit task", "arg0", err)
//	    return
//	}
//
//	// Wait for result
//	result := <-resultCh
//	if result.Error != nil {
//	    slog.Info("Cache operation failed", "arg0", result.Error)
//	}
package redis

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheWorkerPool manages a bounded pool of workers for cache operations
// to prevent overwhelming the system with concurrent Redis operations.
//
// Features:
// - Bounded worker pool with configurable size
// - Queue for pending operations
// - Graceful shutdown with timeout
// - Metrics tracking (active workers, queue size, completed tasks)
// - Non-blocking task submission with error feedback
//
// Usage:
//
//	pool := NewCacheWorkerPool(10, 100)  // 10 workers, queue size 100
//	pool.Start()
//	defer pool.Shutdown(5 * time.Second)
//
//	// Submit a cache operation
//	resultCh := make(chan CacheResult, 1)
//	task := CacheTask{
//	    Operation: "get",
//	    Key:       "user:123",
//	    ResultCh:  resultCh,
//	}
//	if err := pool.Submit(task); err != nil {
//	    // Handle queue full error
//	}
//	result := <-resultCh
type CacheWorkerPool struct {
	workers   int
	taskQueue chan CacheTask
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	metrics   *WorkerMetrics
	client    *redis.Client

	// Shutdown state
	shutdownOnce sync.Once
	shutdownCh   chan struct{}
}

// CacheTask represents a single cache operation to be executed by a worker
type CacheTask struct {
	Operation string        // "get", "set", "delete", "mget", "pipeline"
	Key       string        // Redis key (for get, set, delete)
	Keys      []string      // Redis keys (for mget, pipeline operations)
	Value     interface{}   // Value to set (for set operations)
	TTL       time.Duration // TTL for set operations
	ResultCh  chan CacheResult
	Context   context.Context // Optional context for cancellation
}

// CacheResult contains the result of a cache operation
type CacheResult struct {
	Value  interface{} // Result value (string, []byte, []interface{} for mget)
	Exists bool        // Whether key exists (for get operations)
	Error  error       // Error if operation failed
}

// WorkerMetrics tracks worker pool performance metrics
type WorkerMetrics struct {
	ActiveWorkers   atomic.Int32 // Current number of active workers
	QueueSize       atomic.Int32 // Current queue size
	CompletedTasks  atomic.Int64 // Total completed tasks
	FailedTasks     atomic.Int64 // Total failed tasks
	TotalProcessed  atomic.Int64 // Total tasks processed (completed + failed)
	QueueDropped    atomic.Int64 // Tasks dropped due to full queue
	AvgProcessTime  atomic.Int64 // Average processing time in microseconds
	processingTimes []time.Duration
	timesMutex      sync.RWMutex
}

// WorkerPoolConfig configures the cache worker pool
type WorkerPoolConfig struct {
	Workers   int // Number of concurrent workers
	QueueSize int // Size of task queue
}

// DefaultWorkerPoolConfig returns default worker pool configuration
func DefaultWorkerPoolConfig() *WorkerPoolConfig {
	return &WorkerPoolConfig{
		Workers:   10,  // Default: 10 concurrent workers
		QueueSize: 100, // Default: queue up to 100 operations
	}
}

// NewCacheWorkerPool creates a new cache worker pool
//
// Parameters:
//   - workers: Number of concurrent workers (recommended: 5-20 depending on load)
//   - queueSize: Size of task queue (recommended: 50-500 depending on burst capacity)
//
// Returns nil if Redis client is not initialized.
func NewCacheWorkerPool(workers, queueSize int) *CacheWorkerPool {
	if client == nil {
		slog.Warn("Redis client not initialized - worker pool creation skipped")
		return nil
	}

	if workers <= 0 {
		workers = 10
	}
	if queueSize <= 0 {
		queueSize = 100
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &CacheWorkerPool{
		workers:    workers,
		taskQueue:  make(chan CacheTask, queueSize),
		ctx:        ctx,
		cancel:     cancel,
		metrics:    &WorkerMetrics{},
		client:     client,
		shutdownCh: make(chan struct{}),
	}
}

// Start begins processing tasks with the configured number of workers
// This is non-blocking - workers run in background goroutines.
func (p *CacheWorkerPool) Start() {
	if p == nil {
		return
	}

	slog.Info("Starting cache worker pool",
		"workers", p.workers,
		"queue_size", cap(p.taskQueue))

	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker is the main worker loop that processes tasks from the queue
func (p *CacheWorkerPool) worker(id int) {
	defer p.wg.Done()
	p.metrics.ActiveWorkers.Add(1)
	defer p.metrics.ActiveWorkers.Add(-1)

	slog.Debug("Worker started", "worker_id", id)

	for {
		select {
		case <-p.ctx.Done():
			slog.Debug("Worker shutting down", "worker_id", id)
			return

		case task, ok := <-p.taskQueue:
			if !ok {
				slog.Debug("Task queue closed, worker exiting", "worker_id", id)
				return
			}

			// Process the task
			p.processTask(id, task)
		}
	}
}

// processTask executes a single cache task and sends the result
func (p *CacheWorkerPool) processTask(workerID int, task CacheTask) {
	startTime := time.Now()
	defer func() {
		processingTime := time.Since(startTime)
		p.updateProcessingTime(processingTime)
		p.metrics.TotalProcessed.Add(1)
		p.metrics.QueueSize.Add(-1)
	}()

	// Use task context if provided, otherwise use pool context
	ctx := task.Context
	if ctx == nil {
		ctx = p.ctx
	}

	var result CacheResult

	switch task.Operation {
	case "get":
		result = p.doGet(ctx, task.Key)
	case "set":
		result = p.doSet(ctx, task.Key, task.Value, task.TTL)
	case "delete":
		result = p.doDelete(ctx, task.Key)
	case "mget":
		result = p.doMGet(ctx, task.Keys)
	default:
		result = CacheResult{
			Error: fmt.Errorf("unknown operation: %s", task.Operation),
		}
	}

	// Update metrics
	if result.Error != nil {
		p.metrics.FailedTasks.Add(1)
	} else {
		p.metrics.CompletedTasks.Add(1)
	}

	// Send result if channel is provided
	if task.ResultCh != nil {
		select {
		case task.ResultCh <- result:
			// Result sent successfully
		case <-ctx.Done():
			// Context cancelled while sending result
			slog.Debug("Context cancelled while sending result", "worker_id", workerID)
		default:
			// Non-blocking send failed, receiver not ready
			slog.Warn("Result channel full or closed, dropping result",
				"worker_id", workerID,
				"operation", task.Operation,
				"key", task.Key)
		}
	}
}

// doGet performs a GET operation
func (p *CacheWorkerPool) doGet(ctx context.Context, key string) CacheResult {
	val, err := p.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return CacheResult{Exists: false, Error: nil}
	}
	if err != nil {
		return CacheResult{Error: fmt.Errorf("redis get failed: %w", err)}
	}
	return CacheResult{Value: val, Exists: true, Error: nil}
}

// doSet performs a SET operation
func (p *CacheWorkerPool) doSet(ctx context.Context, key string, value interface{}, ttl time.Duration) CacheResult {
	err := p.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		return CacheResult{Error: fmt.Errorf("redis set failed: %w", err)}
	}
	return CacheResult{Error: nil}
}

// doDelete performs a DELETE operation
func (p *CacheWorkerPool) doDelete(ctx context.Context, key string) CacheResult {
	err := p.client.Del(ctx, key).Err()
	if err != nil {
		return CacheResult{Error: fmt.Errorf("redis delete failed: %w", err)}
	}
	return CacheResult{Error: nil}
}

// doMGet performs a MGET operation (get multiple keys)
func (p *CacheWorkerPool) doMGet(ctx context.Context, keys []string) CacheResult {
	vals, err := p.client.MGet(ctx, keys...).Result()
	if err != nil {
		return CacheResult{Error: fmt.Errorf("redis mget failed: %w", err)}
	}
	return CacheResult{Value: vals, Error: nil}
}

// Submit submits a task to the worker pool
// Returns error if queue is full or pool is shutting down.
//
// This is non-blocking - returns immediately with error if queue is full.
func (p *CacheWorkerPool) Submit(task CacheTask) error {
	if p == nil {
		return fmt.Errorf("worker pool not initialized")
	}

	select {
	case <-p.ctx.Done():
		return fmt.Errorf("worker pool is shutting down")
	case <-p.shutdownCh:
		return fmt.Errorf("worker pool is shut down")
	default:
	}

	// Non-blocking send to queue
	select {
	case p.taskQueue <- task:
		p.metrics.QueueSize.Add(1)
		return nil
	default:
		// Queue is full
		p.metrics.QueueDropped.Add(1)
		return fmt.Errorf("task queue is full (size: %d)", cap(p.taskQueue))
	}
}

// Shutdown gracefully shuts down the worker pool
// Waits for all workers to finish processing current tasks.
//
// Parameters:
//   - timeout: Maximum time to wait for graceful shutdown
//
// Returns error if shutdown times out.
func (p *CacheWorkerPool) Shutdown(timeout time.Duration) error {
	if p == nil {
		return nil
	}

	var shutdownErr error
	p.shutdownOnce.Do(func() {
		slog.Info("Shutting down cache worker pool", "timeout", timeout)

		// Signal shutdown
		close(p.shutdownCh)

		// Stop accepting new tasks
		close(p.taskQueue)

		// Cancel context to stop workers
		p.cancel()

		// Wait for workers with timeout
		done := make(chan struct{})
		go func() {
			p.wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			slog.Info("Cache worker pool shutdown complete",
				"completed_tasks", p.metrics.CompletedTasks.Load(),
				"failed_tasks", p.metrics.FailedTasks.Load(),
				"dropped_tasks", p.metrics.QueueDropped.Load())
		case <-time.After(timeout):
			shutdownErr = fmt.Errorf("worker pool shutdown timed out after %v", timeout)
			slog.Error("Worker pool shutdown timeout",
				"timeout", timeout,
				"active_workers", p.metrics.ActiveWorkers.Load())
		}
	})

	return shutdownErr
}

// GetMetrics returns a snapshot of worker pool metrics
func (p *CacheWorkerPool) GetMetrics() *WorkerMetrics {
	if p == nil {
		return &WorkerMetrics{}
	}
	return p.metrics
}

// updateProcessingTime updates the average processing time metric
func (p *CacheWorkerPool) updateProcessingTime(duration time.Duration) {
	p.metrics.timesMutex.Lock()
	defer p.metrics.timesMutex.Unlock()

	// Keep only last 1000 processing times for rolling average
	maxSamples := 1000
	p.metrics.processingTimes = append(p.metrics.processingTimes, duration)
	if len(p.metrics.processingTimes) > maxSamples {
		p.metrics.processingTimes = p.metrics.processingTimes[1:]
	}

	// Calculate average
	var total time.Duration
	for _, t := range p.metrics.processingTimes {
		total += t
	}
	avg := total / time.Duration(len(p.metrics.processingTimes))
	p.metrics.AvgProcessTime.Store(int64(avg.Microseconds()))
}

// MetricsSnapshot returns a snapshot of current metrics
type MetricsSnapshot struct {
	ActiveWorkers  int32         `json:"active_workers"`
	QueueSize      int32         `json:"queue_size"`
	CompletedTasks int64         `json:"completed_tasks"`
	FailedTasks    int64         `json:"failed_tasks"`
	TotalProcessed int64         `json:"total_processed"`
	QueueDropped   int64         `json:"queue_dropped"`
	AvgProcessTime time.Duration `json:"avg_process_time_us"`
}

// Snapshot returns a point-in-time snapshot of metrics
func (m *WorkerMetrics) Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		ActiveWorkers:  m.ActiveWorkers.Load(),
		QueueSize:      m.QueueSize.Load(),
		CompletedTasks: m.CompletedTasks.Load(),
		FailedTasks:    m.FailedTasks.Load(),
		TotalProcessed: m.TotalProcessed.Load(),
		QueueDropped:   m.QueueDropped.Load(),
		AvgProcessTime: time.Duration(m.AvgProcessTime.Load()) * time.Microsecond,
	}
}
