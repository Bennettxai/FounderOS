package feedback

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TaskStatus represents the lifecycle state of an ImprovementTask.
type TaskStatus string

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusApplying TaskStatus = "applying"
	TaskStatusApplied  TaskStatus = "applied"
	TaskStatusFailed   TaskStatus = "failed"
)

// ImprovementTask wraps an ImprovementSuggestion with queue bookkeeping.
type ImprovementTask struct {
	ID         string                `json:"id"`
	TenantID   string                `json:"tenant_id"`
	Suggestion ImprovementSuggestion `json:"suggestion"`
	Status     TaskStatus            `json:"status"`
	Error      string                `json:"error,omitempty"`
	CreatedAt  time.Time             `json:"created_at"`
	UpdatedAt  time.Time             `json:"updated_at"`
}

// ImprovementTaskQueue is an in-memory, concurrency-safe queue for
// ImprovementTasks.  Auto-eligible tasks (shouldAutoApply) are applied
// immediately during Process.
type ImprovementTaskQueue struct {
	mu     sync.RWMutex
	tasks  map[string]*ImprovementTask // taskID -> task
	engine *SelfImprovementEngine
	logger *slog.Logger
}

// NewImprovementTaskQueue constructs an ImprovementTaskQueue.
// engine is used to update Q-values when tasks are applied.
func NewImprovementTaskQueue(engine *SelfImprovementEngine, logger *slog.Logger) *ImprovementTaskQueue {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("component", "improvement_task_queue")
	return &ImprovementTaskQueue{
		tasks:  make(map[string]*ImprovementTask),
		engine: engine,
		logger: logger,
	}
}

// Enqueue adds a suggestion as a pending ImprovementTask and returns the
// assigned task ID.
func (q *ImprovementTaskQueue) Enqueue(
	ctx context.Context,
	tenantID string,
	suggestion ImprovementSuggestion,
) (string, error) {
	if tenantID == "" {
		return "", fmt.Errorf("task queue enqueue: tenantID must not be empty")
	}

	taskID := uuid.New().String()
	now := time.Now().UTC()

	task := &ImprovementTask{
		ID:         taskID,
		TenantID:   tenantID,
		Suggestion: suggestion,
		Status:     TaskStatusPending,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	q.mu.Lock()
	q.tasks[taskID] = task
	q.mu.Unlock()

	q.logger.InfoContext(ctx, "task enqueued",
		"task_id", taskID,
		"tenant_id", tenantID,
		"improvement_type", suggestion.Type,
		"auto_apply", suggestion.AutoApply)

	return taskID, nil
}

// Process iterates over all pending tasks and applies those that are
// auto-eligible.  Non-eligible tasks remain pending.
// Applied tasks transition to TaskStatusApplied; failures transition to
// TaskStatusFailed.
func (q *ImprovementTaskQueue) Process(ctx context.Context) error {
	// Collect pending task IDs under read-lock to avoid holding the lock
	// during potentially slow apply operations.
	q.mu.RLock()
	var pending []string
	for id, t := range q.tasks {
		if t.Status == TaskStatusPending {
			pending = append(pending, id)
		}
	}
	q.mu.RUnlock()

	for _, taskID := range pending {
		q.mu.Lock()
		task, ok := q.tasks[taskID]
		if !ok || task.Status != TaskStatusPending {
			q.mu.Unlock()
			continue
		}
		if !shouldAutoApply(task.Suggestion) {
			// Not eligible for automatic application; leave as pending.
			q.mu.Unlock()
			continue
		}
		task.Status = TaskStatusApplying
		task.UpdatedAt = time.Now().UTC()
		q.mu.Unlock()

		applyErr := q.apply(ctx, task)

		q.mu.Lock()
		task.UpdatedAt = time.Now().UTC()
		if applyErr != nil {
			task.Status = TaskStatusFailed
			task.Error = applyErr.Error()
			q.logger.ErrorContext(ctx, "task apply failed",
				"task_id", taskID,
				"improvement_type", task.Suggestion.Type,
				"error", applyErr)
		} else {
			task.Status = TaskStatusApplied
			q.logger.InfoContext(ctx, "task applied",
				"task_id", taskID,
				"improvement_type", task.Suggestion.Type)
		}
		q.mu.Unlock()
	}

	return nil
}

// GetStatus returns a snapshot of the task with the given ID.
func (q *ImprovementTaskQueue) GetStatus(ctx context.Context, taskID string) (*ImprovementTask, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	task, ok := q.tasks[taskID]
	if !ok {
		return nil, fmt.Errorf("task queue get status: task %q not found", taskID)
	}

	// Return a copy to prevent external mutation.
	copy := *task
	return &copy, nil
}

// List returns a snapshot of all tasks belonging to tenantID.
func (q *ImprovementTaskQueue) List(ctx context.Context, tenantID string) ([]ImprovementTask, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var out []ImprovementTask
	for _, t := range q.tasks {
		if t.TenantID == tenantID {
			out = append(out, *t)
		}
	}
	return out, nil
}

// apply executes the improvement suggestion and updates the Q-table with the
// resulting positive reward.
func (q *ImprovementTaskQueue) apply(ctx context.Context, task *ImprovementTask) error {
	if q.engine == nil {
		return fmt.Errorf("task queue apply: no engine configured")
	}

	// The reward is derived from the suggestion's impact score scaled to a
	// reasonable Q-learning reward magnitude.
	reward := task.Suggestion.Impact * 10.0

	state := QState{
		Context:         contextToken(task.Suggestion.Description),
		AgentType:       "",
		ImprovementType: task.Suggestion.Type,
	}

	if err := q.engine.UpdateQValue(ctx, state, reward); err != nil {
		return fmt.Errorf("task queue apply: update Q-value: %w", err)
	}

	return nil
}
