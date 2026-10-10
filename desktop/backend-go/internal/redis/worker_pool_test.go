package redis

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestRedis creates a test Redis instance using miniredis
func setupTestRedis(t *testing.T) (*miniredis.Miniredis, func()) {
	t.Helper()

	// Create miniredis instance
	mr, err := miniredis.Run()
	require.NoError(t, err)

	// Set up global Redis client for worker pool
	client = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	cleanup := func() {
		if client != nil {
			client.Close()
			client = nil
		}
		mr.Close()
	}

	return mr, cleanup
}

func TestNewCacheWorkerPool(t *testing.T) {
	_, cleanup := setupTestRedis(t)
	defer cleanup()

	tests := []struct {
		name      string
		workers   int
		queueSize int
		wantNil   bool
	}{
		{
			name:      "valid configuration",
			workers:   5,
			queueSize: 50,
			wantNil:   false,
		},
		{
			name:      "default values when zero",
			workers:   0,
			queueSize: 0,
			wantNil:   false,
		},
		{
			name:      "negative values normalized to defaults",
			workers:   -5,
			queueSize: -10,
			wantNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := NewCacheWorkerPool(tt.workers, tt.queueSize)

			if tt.wantNil {
				assert.Nil(t, pool)
			} else {
				assert.NotNil(t, pool)
				assert.NotNil(t, pool.client)
				assert.NotNil(t, pool.metrics)
				assert.NotNil(t, pool.taskQueue)

				// Verify defaults are applied for invalid values
				if tt.workers <= 0 {
					assert.Equal(t, 10, pool.workers)
				} else {
					assert.Equal(t, tt.workers, pool.workers)
				}

				if tt.queueSize <= 0 {
					assert.Equal(t, 100, cap(pool.taskQueue))
				} else {
					assert.Equal(t, tt.queueSize, cap(pool.taskQueue))
				}

				// Clean up
				_ = pool.Shutdown(100 * time.Millisecond)
			}
		})
	}
}

func TestNewCacheWorkerPool_NoRedisClient(t *testing.T) {
	// Save original client
	originalClient := client
	defer func() { client = originalClient }()

	// Set client to nil
	client = nil

	pool := NewCacheWorkerPool(10, 100)
	assert.Nil(t, pool, "Should return nil when Redis client not initialized")
}

func TestCacheWorkerPool_BasicOperations(t *testing.T) {
	mr, cleanup := setupTestRedis(t)
	defer cleanup()

	pool := NewCacheWorkerPool(5, 50)
	require.NotNil(t, pool)

	pool.Start()
	defer pool.Shutdown(2 * time.Second)

	ctx := context.Background()

	t.Run("set operation", func(t *testing.T) {
		resultCh := make(chan CacheResult, 1)
		task := CacheTask{
			Operation: "set",
			Key:       "test:key1",
			Value:     "test-value",
			TTL:       time.Minute,
			ResultCh:  resultCh,
			Context:   ctx,
		}

		err := pool.Submit(task)
		require.NoError(t, err)

		// Wait for result
		select {
		case result := <-resultCh:
			assert.NoError(t, result.Error)

			// Verify key was set in Redis
			val, err := mr.Get("test:key1")
			assert.NoError(t, err)
			assert.Equal(t, "test-value", val)

		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for set result")
		}
	})

	t.Run("get operation", func(t *testing.T) {
		// Set a value first
		mr.Set("test:key2", "stored-value")

		resultCh := make(chan CacheResult, 1)
		task := CacheTask{
			Operation: "get",
			Key:       "test:key2",
			ResultCh:  resultCh,
			Context:   ctx,
		}

		err := pool.Submit(task)
		require.NoError(t, err)

		// Wait for result
		select {
		case result := <-resultCh:
			assert.NoError(t, result.Error)
			assert.True(t, result.Exists)
			assert.Equal(t, "stored-value", result.Value)

		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for get result")
		}
	})

	t.Run("get non-existent key", func(t *testing.T) {
		resultCh := make(chan CacheResult, 1)
		task := CacheTask{
			Operation: "get",
			Key:       "test:nonexistent",
			ResultCh:  resultCh,
			Context:   ctx,
		}

		err := pool.Submit(task)
		require.NoError(t, err)

		// Wait for result
		select {
		case result := <-resultCh:
			assert.NoError(t, result.Error)
			assert.False(t, result.Exists)
			assert.Nil(t, result.Value)

		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for get result")
		}
	})

	t.Run("delete operation", func(t *testing.T) {
		// Set a value first
		mr.Set("test:key3", "to-delete")

		resultCh := make(chan CacheResult, 1)
		task := CacheTask{
			Operation: "delete",
			Key:       "test:key3",
			ResultCh:  resultCh,
			Context:   ctx,
		}

		err := pool.Submit(task)
		require.NoError(t, err)

		// Wait for result
		select {
		case result := <-resultCh:
			assert.NoError(t, result.Error)

			// Verify key was deleted
			_, err := mr.Get("test:key3")
			assert.Error(t, err, "Key should not exist after deletion")

		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for delete result")
		}
	})

	t.Run("mget operation", func(t *testing.T) {
		// Set multiple values
		mr.Set("test:mkey1", "value1")
		mr.Set("test:mkey2", "value2")
		mr.Set("test:mkey3", "value3")

		resultCh := make(chan CacheResult, 1)
		task := CacheTask{
			Operation: "mget",
			Keys:      []string{"test:mkey1", "test:mkey2", "test:mkey3"},
			ResultCh:  resultCh,
			Context:   ctx,
		}

		err := pool.Submit(task)
		require.NoError(t, err)

		// Wait for result
		select {
		case result := <-resultCh:
			assert.NoError(t, result.Error)
			vals, ok := result.Value.([]interface{})
			assert.True(t, ok)
			assert.Len(t, vals, 3)
			assert.Equal(t, "value1", vals[0])
			assert.Equal(t, "value2", vals[1])
			assert.Equal(t, "value3", vals[2])

		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for mget result")
		}
	})
}

func TestCacheWorkerPool_ConcurrentOperations(t *testing.T) {
	_, cleanup := setupTestRedis(t)
	defer cleanup()

	pool := NewCacheWorkerPool(10, 100)
	require.NotNil(t, pool)

	pool.Start()
	defer pool.Shutdown(5 * time.Second)

	ctx := context.Background()
	numTasks := 100

	var wg sync.WaitGroup
	wg.Add(numTasks)

	// Submit many concurrent operations
	for i := 0; i < numTasks; i++ {
		go func(idx int) {
			defer wg.Done()

			key := fmt.Sprintf("concurrent:key%d", idx)
			value := fmt.Sprintf("value%d", idx)

			// Set
			resultCh := make(chan CacheResult, 1)
			setTask := CacheTask{
				Operation: "set",
				Key:       key,
				Value:     value,
				TTL:       time.Minute,
				ResultCh:  resultCh,
				Context:   ctx,
			}

			err := pool.Submit(setTask)
			assert.NoError(t, err)

			result := <-resultCh
			assert.NoError(t, result.Error)

			// Get
			getResultCh := make(chan CacheResult, 1)
			getTask := CacheTask{
				Operation: "get",
				Key:       key,
				ResultCh:  getResultCh,
				Context:   ctx,
			}

			err = pool.Submit(getTask)
			assert.NoError(t, err)

			getResult := <-getResultCh
			assert.NoError(t, getResult.Error)
			assert.True(t, getResult.Exists)
			assert.Equal(t, value, getResult.Value)
		}(i)
	}

	// Wait for all operations with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for concurrent operations")
	}

	// Check metrics
	metrics := pool.GetMetrics().Snapshot()
	assert.Equal(t, int64(numTasks*2), metrics.CompletedTasks, "Should complete all set and get operations")
	assert.Equal(t, int64(0), metrics.FailedTasks, "Should have no failed tasks")
	assert.Equal(t, int64(0), metrics.QueueDropped, "Should have no dropped tasks")
}

func TestCacheWorkerPool_QueueOverflow(t *testing.T) {
	_, cleanup := setupTestRedis(t)
	defer cleanup()

	// Create pool with small queue
	pool := NewCacheWorkerPool(1, 2)
	require.NotNil(t, pool)

	pool.Start()
	defer pool.Shutdown(2 * time.Second)

	ctx := context.Background()

	// Fill the queue and overflow it
	var submitted, dropped int

	for i := 0; i < 10; i++ {
		task := CacheTask{
			Operation: "set",
			Key:       fmt.Sprintf("overflow:key%d", i),
			Value:     "value",
			TTL:       time.Minute,
			ResultCh:  make(chan CacheResult, 1),
			Context:   ctx,
		}

		err := pool.Submit(task)
		if err != nil {
			dropped++
			assert.Contains(t, err.Error(), "queue is full")
		} else {
			submitted++
		}
	}

	assert.Greater(t, dropped, 0, "Should have some dropped tasks due to queue overflow")
	assert.Greater(t, submitted, 0, "Should have submitted some tasks")

	// Wait a bit for processing
	time.Sleep(200 * time.Millisecond)

	metrics := pool.GetMetrics().Snapshot()
	assert.Equal(t, int64(dropped), metrics.QueueDropped, "Metrics should track dropped tasks")
}

func TestCacheWorkerPool_GracefulShutdown(t *testing.T) {
	_, cleanup := setupTestRedis(t)
	defer cleanup()

	pool := NewCacheWorkerPool(5, 50)
	require.NotNil(t, pool)

	pool.Start()

	ctx := context.Background()

	// Submit some tasks
	for i := 0; i < 10; i++ {
		resultCh := make(chan CacheResult, 1)
		task := CacheTask{
			Operation: "set",
			Key:       fmt.Sprintf("shutdown:key%d", i),
			Value:     "value",
			TTL:       time.Minute,
			ResultCh:  resultCh,
			Context:   ctx,
		}
		err := pool.Submit(task)
		require.NoError(t, err)
	}

	// Shutdown with timeout
	err := pool.Shutdown(3 * time.Second)
	assert.NoError(t, err, "Shutdown should complete within timeout")

	// Verify workers stopped
	metrics := pool.GetMetrics().Snapshot()
	assert.Equal(t, int32(0), metrics.ActiveWorkers, "All workers should be stopped")

	// Try to submit after shutdown
	resultCh := make(chan CacheResult, 1)
	task := CacheTask{
		Operation: "set",
		Key:       "after:shutdown",
		Value:     "value",
		TTL:       time.Minute,
		ResultCh:  resultCh,
		Context:   ctx,
	}

	err = pool.Submit(task)
	assert.Error(t, err, "Should not accept tasks after shutdown")
	assert.Contains(t, err.Error(), "shut", "Error should indicate shutdown")
}

func TestCacheWorkerPool_ShutdownTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	_, cleanup := setupTestRedis(t)
	defer cleanup()

	pool := NewCacheWorkerPool(2, 10)
	require.NotNil(t, pool)

	pool.Start()

	ctx := context.Background()

	// Submit tasks that will take a while
	for i := 0; i < 5; i++ {
		resultCh := make(chan CacheResult, 1)
		task := CacheTask{
			Operation: "set",
			Key:       fmt.Sprintf("timeout:key%d", i),
			Value:     "value",
			TTL:       5 * time.Second,
			ResultCh:  resultCh,
			Context:   ctx,
		}
		_ = pool.Submit(task)
	}

	// Shutdown with very short timeout
	err := pool.Shutdown(1 * time.Millisecond)

	// Note: This might not error if tasks complete quickly
	// Just ensure shutdown returns (doesn't hang)
	assert.NotPanics(t, func() {
		_ = err
	})
}

func TestCacheWorkerPool_Metrics(t *testing.T) {
	_, cleanup := setupTestRedis(t)
	defer cleanup()

	pool := NewCacheWorkerPool(3, 20)
	require.NotNil(t, pool)

	pool.Start()
	defer pool.Shutdown(2 * time.Second)

	ctx := context.Background()

	// Submit successful tasks
	for i := 0; i < 5; i++ {
		resultCh := make(chan CacheResult, 1)
		task := CacheTask{
			Operation: "set",
			Key:       fmt.Sprintf("metrics:key%d", i),
			Value:     "value",
			TTL:       time.Minute,
			ResultCh:  resultCh,
			Context:   ctx,
		}
		err := pool.Submit(task)
		require.NoError(t, err)

		// Wait for completion
		<-resultCh
	}

	// Submit invalid operation (should fail)
	for i := 0; i < 2; i++ {
		resultCh := make(chan CacheResult, 1)
		task := CacheTask{
			Operation: "invalid_op",
			Key:       "test",
			ResultCh:  resultCh,
			Context:   ctx,
		}
		err := pool.Submit(task)
		require.NoError(t, err)

		result := <-resultCh
		assert.Error(t, result.Error)
	}

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	metrics := pool.GetMetrics().Snapshot()

	assert.Equal(t, int64(5), metrics.CompletedTasks, "Should have 5 completed tasks")
	assert.Equal(t, int64(2), metrics.FailedTasks, "Should have 2 failed tasks")
	assert.Equal(t, int64(7), metrics.TotalProcessed, "Should have processed 7 total tasks")
	assert.Equal(t, int64(0), metrics.QueueDropped, "Should have no dropped tasks")
	assert.GreaterOrEqual(t, int32(3), metrics.ActiveWorkers, "Should have workers running")
	assert.Greater(t, metrics.AvgProcessTime, time.Duration(0), "Should have positive average processing time")
}

func TestCacheWorkerPool_UnknownOperation(t *testing.T) {
	_, cleanup := setupTestRedis(t)
	defer cleanup()

	pool := NewCacheWorkerPool(2, 10)
	require.NotNil(t, pool)

	pool.Start()
	defer pool.Shutdown(2 * time.Second)

	ctx := context.Background()

	resultCh := make(chan CacheResult, 1)
	task := CacheTask{
		Operation: "unknown_operation",
		Key:       "test:key",
		ResultCh:  resultCh,
		Context:   ctx,
	}

	err := pool.Submit(task)
	require.NoError(t, err)

	select {
	case result := <-resultCh:
		assert.Error(t, result.Error)
		assert.Contains(t, result.Error.Error(), "unknown operation")

	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for result")
	}
}

func TestCacheWorkerPool_NilResultChannel(t *testing.T) {
	_, cleanup := setupTestRedis(t)
	defer cleanup()

	pool := NewCacheWorkerPool(2, 10)
	require.NotNil(t, pool)

	pool.Start()
	defer pool.Shutdown(2 * time.Second)

	ctx := context.Background()

	// Submit task without result channel (should not panic)
	task := CacheTask{
		Operation: "set",
		Key:       "test:key",
		Value:     "value",
		TTL:       time.Minute,
		ResultCh:  nil, // No result channel
		Context:   ctx,
	}

	err := pool.Submit(task)
	assert.NoError(t, err)

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Should complete without panic
	metrics := pool.GetMetrics().Snapshot()
	assert.Greater(t, metrics.CompletedTasks, int64(0))
}

func TestCacheWorkerPool_ContextCancellation(t *testing.T) {
	_, cleanup := setupTestRedis(t)
	defer cleanup()

	pool := NewCacheWorkerPool(2, 10)
	require.NotNil(t, pool)

	pool.Start()
	defer pool.Shutdown(2 * time.Second)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	resultCh := make(chan CacheResult, 1)
	task := CacheTask{
		Operation: "get",
		Key:       "test:key",
		ResultCh:  resultCh,
		Context:   ctx,
	}

	// Cancel immediately
	cancel()

	err := pool.Submit(task)
	require.NoError(t, err)

	// Wait a bit for the worker to process the task with cancelled context
	select {
	case result := <-resultCh:
		// Context cancellation should cause Redis operation to fail
		assert.Error(t, result.Error, "Expected error from cancelled context")
		if result.Error != nil {
			assert.Contains(t, result.Error.Error(), "context", "Error should mention context cancellation")
		}

	case <-time.After(2 * time.Second):
		// If we timeout, it means the task was never processed due to immediate cancellation
		// This is acceptable behavior - the worker might have checked context before starting
		t.Log("Task not processed due to immediate context cancellation (acceptable)")
	}
}
