package services

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// =====================================================================
// TYPES
// =====================================================================

// JobHandler is a function that processes a job payload
type JobHandler func(ctx context.Context, payload map[string]interface{}) (interface{}, error)

// JobWorker represents a background job worker
type JobWorker struct {
	service      *BackgroundJobsService
	workerID     string
	pollInterval time.Duration
	handlers     map[string]JobHandler
	handlersMu   sync.RWMutex
	stopChan     chan struct{}
	wg           sync.WaitGroup
	running      bool
	runningMu    sync.Mutex
}

// =====================================================================
// CONSTRUCTOR
// =====================================================================

// NewJobWorker creates a new job worker
func NewJobWorker(service *BackgroundJobsService, workerID string, pollInterval time.Duration) *JobWorker {
	if pollInterval == 0 {
		pollInterval = 5 * time.Second // Default poll interval
	}

	return &JobWorker{
		service:      service,
		workerID:     workerID,
		pollInterval: pollInterval,
		handlers:     make(map[string]JobHandler),
		stopChan:     make(chan struct{}),
		running:      false,
	}
}

// =====================================================================
// HANDLER REGISTRATION
// =====================================================================

// RegisterHandler registers a job handler for a specific job type
func (w *JobWorker) RegisterHandler(jobType string, handler JobHandler) {
	w.handlersMu.Lock()
	defer w.handlersMu.Unlock()

	w.handlers[jobType] = handler
	slog.Info("Job handler registered", "worker_id", w.workerID, "job_type", jobType)
}

// GetHandler retrieves a handler for a job type
func (w *JobWorker) getHandler(jobType string) (JobHandler, bool) {
	w.handlersMu.RLock()
	defer w.handlersMu.RUnlock()

	handler, exists := w.handlers[jobType]
	return handler, exists
}

// =====================================================================
// WORKER LIFECYCLE
// =====================================================================

// Start begins the worker loop
func (w *JobWorker) Start(ctx context.Context) error {
	w.runningMu.Lock()
	if w.running {
		w.runningMu.Unlock()
		return fmt.Errorf("worker already running")
	}
	w.running = true
	w.runningMu.Unlock()

	slog.InfoContext(ctx, "Worker starting",
		"worker_id", w.workerID,
		"poll_interval", w.pollInterval,
	)

	w.wg.Add(1)
	go w.workerLoop(ctx)

	return nil
}

// Stop gracefully stops the worker
func (w *JobWorker) Stop() error {
	w.runningMu.Lock()
	if !w.running {
		w.runningMu.Unlock()
		return fmt.Errorf("worker not running")
	}
	w.runningMu.Unlock()

	slog.Info("Worker stopping", "worker_id", w.workerID)

	close(w.stopChan)
	w.wg.Wait()

	w.runningMu.Lock()
	w.running = false
	w.runningMu.Unlock()

	slog.Info("Worker stopped", "worker_id", w.workerID)
	return nil
}

// IsRunning returns whether the worker is currently running
func (w *JobWorker) IsRunning() bool {
	w.runningMu.Lock()
	defer w.runningMu.Unlock()
	return w.running
}

// =====================================================================
// WORKER LOOP
// =====================================================================

// workerLoop is the main worker processing loop
func (w *JobWorker) workerLoop(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	slog.InfoContext(ctx, "Worker loop started", "worker_id", w.workerID)

	for {
		select {
		case <-w.stopChan:
			slog.InfoContext(ctx, "Worker loop stopped", "worker_id", w.workerID)
			return

		case <-ticker.C:
			slog.InfoContext(ctx, "Worker tick", "worker_id", w.workerID)
			// Try to acquire and process a job
			if err := w.processNextJob(ctx); err != nil {
				// Log error but continue running
				slog.ErrorContext(ctx, "Error processing job",
					"worker_id", w.workerID,
					"error", err,
				)
			}

		case <-ctx.Done():
			slog.ErrorContext(ctx, "Context cancelled!", "worker_id", w.workerID, "error", ctx.Err())
			return
		}
	}
}

// processNextJob attempts to acquire and process the next available job
func (w *JobWorker) processNextJob(ctx context.Context) error {
	// Acquire next job
	job, err := w.service.AcquireJob(ctx, w.workerID)
	if err != nil {
		return fmt.Errorf("acquire job: %w", err)
	}

	// No job available
	if job == nil {
		return nil
	}

	// Process the job
	return w.ProcessJob(ctx, job)
}

// =====================================================================
// JOB PROCESSING
// =====================================================================

// ProcessJob processes a single job
func (w *JobWorker) ProcessJob(ctx context.Context, job *BackgroundJob) error {
	startTime := time.Now()

	slog.InfoContext(ctx, "Processing job",
		"worker_id", w.workerID,
		"job_id", job.ID,
		"job_type", job.JobType,
		"attempt", job.AttemptCount,
	)

	// Get handler for job type
	handler, exists := w.getHandler(job.JobType)
	if !exists {
		err := fmt.Errorf("no handler registered for job type: %s", job.JobType)
		slog.ErrorContext(ctx, "Job handler not found",
			"worker_id", w.workerID,
			"job_id", job.ID,
			"job_type", job.JobType,
		)
		// Mark job as failed
		return w.service.FailJob(ctx, job.ID, err.Error())
	}

	// Create job context with timeout
	// Ensure timeout is respected even if parent context is cancelled first
	jobTimeout := 10 * time.Minute
	var jobCtx context.Context
	var cancel context.CancelFunc

	// Check if parent context has a deadline
	if deadline, ok := ctx.Deadline(); ok {
		// Use the minimum of parent deadline and job timeout
		parentRemaining := time.Until(deadline)
		if parentRemaining < jobTimeout {
			jobCtx, cancel = context.WithDeadline(ctx, deadline)
		} else {
			jobCtx, cancel = context.WithTimeout(ctx, jobTimeout)
		}
	} else {
		// No parent deadline, use job timeout
		jobCtx, cancel = context.WithTimeout(ctx, jobTimeout)
	}
	defer cancel()

	// Execute handler
	result, err := handler(jobCtx, job.Payload)
	duration := time.Since(startTime)

	if err != nil {
		slog.ErrorContext(ctx, "Job execution failed",
			"worker_id", w.workerID,
			"job_id", job.ID,
			"job_type", job.JobType,
			"duration", duration,
			"error", err,
		)
		// Mark job as failed (will auto-retry if attempts remaining)
		return w.service.FailJob(ctx, job.ID, err.Error())
	}

	// Prepare result map
	resultMap := map[string]interface{}{
		"completed_at": time.Now(),
		"duration_ms":  duration.Milliseconds(),
	}

	// Add handler result if not nil
	if result != nil {
		resultMap["data"] = result
	}

	// Mark job as completed
	if err := w.service.CompleteJob(ctx, job.ID, resultMap); err != nil {
		slog.ErrorContext(ctx, "Failed to mark job as completed",
			"worker_id", w.workerID,
			"job_id", job.ID,
			"error", err,
		)
		return err
	}

	slog.InfoContext(ctx, "Job completed successfully",
		"worker_id", w.workerID,
		"job_id", job.ID,
		"job_type", job.JobType,
		"duration", duration,
	)

	return nil
}

// =====================================================================
// EXAMPLE JOB HANDLERS
// =====================================================================

// ExampleEmailSendHandler is an example job handler for sending emails
func ExampleEmailSendHandler(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	to, _ := payload["to"].(string)
	subject, _ := payload["subject"].(string)
	_ = payload["body"] // body for email content

	slog.InfoContext(ctx, "Sending email",
		"to", to,
		"subject", subject,
	)

	// Simulate email sending
	time.Sleep(1 * time.Second)

	// In real implementation, call email service here
	// err := emailService.Send(to, subject, body)

	return map[string]interface{}{
		"sent_at":  time.Now(),
		"to":       to,
		"subject":  subject,
		"provider": "smtp",
		"status":   "delivered",
	}, nil
}

// ExampleReportGenerateHandler is an example job handler for generating reports
func ExampleReportGenerateHandler(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	reportType, _ := payload["report_type"].(string)
	startDate, _ := payload["start_date"].(string)
	endDate, _ := payload["end_date"].(string)

	slog.InfoContext(ctx, "Generating report",
		"report_type", reportType,
		"start_date", startDate,
		"end_date", endDate,
	)

	// Simulate report generation
	time.Sleep(2 * time.Second)

	// In real implementation, generate report here
	// report, err := reportService.Generate(reportType, startDate, endDate)

	reportID := uuid.New().String()

	return map[string]interface{}{
		"report_id":    reportID,
		"report_type":  reportType,
		"file_url":     fmt.Sprintf("/reports/%s.pdf", reportID),
		"generated_at": time.Now(),
		"row_count":    1234,
	}, nil
}

// ExampleSyncCalendarHandler is an example job handler for calendar sync
func ExampleSyncCalendarHandler(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	userID, _ := payload["user_id"].(string)
	calendarID, _ := payload["calendar_id"].(string)

	slog.InfoContext(ctx, "Syncing calendar",
		"user_id", userID,
		"calendar_id", calendarID,
	)

	// Simulate calendar sync
	time.Sleep(3 * time.Second)

	// In real implementation, sync calendar here
	// events, err := calendarService.Sync(userID, calendarID)

	return map[string]interface{}{
		"synced_at":    time.Now(),
		"events_count": 42,
		"calendar_id":  calendarID,
		"status":       "success",
	}, nil
}

// ExampleFailingHandler is an example handler that fails (for testing retry logic)
func ExampleFailingHandler(ctx context.Context, payload map[string]interface{}) (interface{}, error) {
	// Type assert with proper error handling
	attemptFloat, ok := payload["attempt"].(float64)
	if !ok {
		// Default to 1 if attempt is not present or not a float64
		attemptFloat = 1
		slog.WarnContext(ctx, "Failing job handler: attempt field missing or invalid, defaulting to 1")
	}

	slog.InfoContext(ctx, "Failing job handler", "attempt", attemptFloat)

	// Fail first 2 attempts, succeed on 3rd
	if attemptFloat < 3 {
		return nil, fmt.Errorf("simulated failure (attempt %v)", attemptFloat)
	}

	return map[string]interface{}{
		"status":  "success_after_retries",
		"attempt": attemptFloat,
	}, nil
}
