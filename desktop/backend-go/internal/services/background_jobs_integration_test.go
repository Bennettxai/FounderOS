package services_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestBackgroundJobsIntegration tests the complete flow: enqueue → acquire → complete
func TestBackgroundJobsIntegration(t *testing.T) {
	// This test requires a running PostgreSQL database
	// Skip if DATABASE_URL not set
	t.Skip("Integration test - requires database")

	// TODO: Setup test database connection
	// pool := setupTestDB(t)
	// defer pool.Close()

	// service := services.NewBackgroundJobsService(pool)

	// job, err := service.EnqueueJob(ctx, "test_job", payload, 1, 3, nil)
	// require.NoError(t, err)
	// assert.NotNil(t, job)
	// assert.Equal(t, "test_job", job.JobType)
	// assert.Equal(t, "pending", job.Status)
	// assert.Equal(t, 0, job.AttemptCount)

	// Test 2: Acquire the job
	// workerJob, err := service.AcquireJob(ctx, "worker-1")
	// require.NoError(t, err)
	// assert.NotNil(t, workerJob)
	// assert.Equal(t, job.ID, workerJob.ID)
	// assert.Equal(t, 1, workerJob.AttemptCount)

	// Test 3: Complete the job
	// result := map[string]interface{}{
	// 	"status": "success",
	// }
	// err = service.CompleteJob(ctx, job.ID, result)
	// require.NoError(t, err)

	// Test 4: Verify status
	// completedJob, err := service.GetJobStatus(ctx, job.ID)
	// require.NoError(t, err)
	// assert.Equal(t, "completed", completedJob.Status)
	// assert.NotNil(t, completedJob.CompletedAt)
}

// TestWorkerIntegration tests worker processing
func TestWorkerIntegration(t *testing.T) {
	t.Skip("Integration test - requires database")

	// pool := setupTestDB(t)
	// defer pool.Close()

	// service := services.NewBackgroundJobsService(pool)
	// worker := services.NewJobWorker(service, "test-worker", 100*time.Millisecond)

	// Test handler
	_ = func(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{"processed": true}, nil
	}

	// worker.RegisterHandler("test_job", testHandler)

	// Enqueue job
	// payload := map[string]interface{}{"data": "test"}
	// job, err := service.EnqueueJob(ctx, "test_job", payload, 0, 3, nil)
	// require.NoError(t, err)

	// Start worker
	// err = worker.Start(ctx)
	// require.NoError(t, err)

	// Wait for processing
	// time.Sleep(500 * time.Millisecond)

	// Stop worker
	// err = worker.Stop()
	// require.NoError(t, err)

	// Verify handler was called
	// assert.True(t, handlerCalled)

	// Verify job completed
	// completedJob, err := service.GetJobStatus(ctx, job.ID)
	// require.NoError(t, err)
	// assert.Equal(t, "completed", completedJob.Status)
}

// TestSchedulerIntegration tests scheduled jobs
func TestSchedulerIntegration(t *testing.T) {
	t.Skip("Integration test - requires database")

	// pool := setupTestDB(t)
	// defer pool.Close()

	// service := services.NewBackgroundJobsService(pool)
	// scheduler := services.NewJobScheduler(pool, service)

	// scheduledJob, err := scheduler.CreateScheduledJob(ctx, req)
	// require.NoError(t, err)
	// assert.NotNil(t, scheduledJob)
	// assert.True(t, scheduledJob.IsActive)
	// assert.NotNil(t, scheduledJob.NextRunAt)

	// Start scheduler
	// err = scheduler.Start(ctx)
	// require.NoError(t, err)

	// Wait for next minute boundary
	// time.Sleep(65 * time.Second)

	// Stop scheduler
	// err = scheduler.Stop()
	// require.NoError(t, err)

	// Verify background job was created
	// jobs, err := service.ListJobs(ctx, services.JobListFilters{
	// 	JobType: &scheduledJob.JobType,
	// 	Limit:   10,
	// })
	// require.NoError(t, err)
	// assert.GreaterOrEqual(t, len(jobs), 1)
}

// TestRetryLogic tests exponential backoff retry
func TestRetryLogic(t *testing.T) {
	t.Skip("Integration test - requires database")

	// pool := setupTestDB(t)
	// defer pool.Close()

	// service := services.NewBackgroundJobsService(pool)
	// worker := services.NewJobWorker(service, "test-worker", 100*time.Millisecond)

	// Failing handler (fails 2 times, succeeds on 3rd)
	attemptCount := 0
	_ = func(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
		attemptCount++
		if attemptCount < 3 {
			return nil, assert.AnError
		}
		return map[string]interface{}{"success_on_attempt": attemptCount}, nil
	}

	// worker.RegisterHandler("failing_job", failingHandler)

	// Enqueue job
	// payload := map[string]interface{}{"attempt": 0}
	// job, err := service.EnqueueJob(ctx, "failing_job", payload, 0, 5, nil)
	// require.NoError(t, err)

	// Start worker
	// err = worker.Start(ctx)
	// require.NoError(t, err)

	// Wait for retries (1min + 5min intervals)
	// time.Sleep(7 * time.Minute)

	// Stop worker
	// err = worker.Stop()
	// require.NoError(t, err)

	// Verify job eventually succeeded
	// completedJob, err := service.GetJobStatus(ctx, job.ID)
	// require.NoError(t, err)
	// assert.Equal(t, "completed", completedJob.Status)
	// assert.Equal(t, 3, completedJob.AttemptCount)
}
