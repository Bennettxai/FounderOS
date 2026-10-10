package appgen

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/appgen/hooks"
	sdk "github.com/severity1/claude-agent-sdk-go"
)

type orchestratorImpl struct {
	progressCallback ProgressCallback
	workers          map[AgentType]Worker
	hookRegistry     *hooks.HookRegistry
	circuitBreaker   *CircuitBreaker
	logger           *slog.Logger
	mu               sync.RWMutex
	wg               sync.WaitGroup
	ctx              context.Context
	cancel           context.CancelFunc
}

func NewOrchestrator(pool *pgxpool.Pool) Orchestrator {
	logger := slog.Default()
	registry := hooks.NewHookRegistry()

	// Create learning storage (PostgreSQL if pool provided, in-memory fallback)
	var learningStorage hooks.LearningStorage
	if pool != nil {
		learningStorage = hooks.NewPostgresLearningStorage(pool, logger)
		logger.Info("orchestrator initialized with PostgreSQL learning storage")
	} else {
		learningStorage = &memoryLearningStorage{}
		logger.Warn("orchestrator initialized with in-memory learning storage (no database provided)")
	}

	// Register OSA hooks for all lifecycle events
	learningHook := hooks.NewLearningCaptureHook(learningStorage, logger)
	contextHook := hooks.NewContextOptimizerHook(200000, logger) // 200K token limit
	errorHook := hooks.NewErrorRecoveryHook(3, logger)           // max 3 retry attempts

	// Register hooks for each lifecycle event
	registry.Register(hooks.PreAgentExecute, contextHook)
	registry.Register(hooks.PostAgentExecute, learningHook)
	registry.Register(hooks.PostAgentExecute, contextHook)
	registry.Register(hooks.OnAgentError, errorHook)
	registry.Register(hooks.OnComplete, learningHook)

	// Initialize circuit breaker for Claude API resilience
	cbConfig := DefaultCircuitBreakerConfig()
	circuitBreaker := NewCircuitBreaker(cbConfig, logger)
	logger.Info("circuit breaker initialized",
		"max_failures", cbConfig.MaxFailures,
		"timeout", cbConfig.Timeout,
		"reset_successes", cbConfig.ResetSuccesses,
	)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	return &orchestratorImpl{
		workers:        make(map[AgentType]Worker),
		hookRegistry:   registry,
		circuitBreaker: circuitBreaker,
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
	}
}

// memoryLearningStorage implements hooks.LearningStorage interface
// TODO: Replace with actual PostgreSQL implementation in Phase 4
type memoryLearningStorage struct {
	mu      sync.RWMutex
	records []hooks.LearningRecord
}

func (s *memoryLearningStorage) Save(ctx context.Context, record hooks.LearningRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, record)
	return nil
}

func (s *memoryLearningStorage) GetPatterns(ctx context.Context, agentType string, limit int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var patterns []string
	for _, record := range s.records {
		if record.AgentType == agentType {
			patterns = append(patterns, record.Patterns...)
			if len(patterns) >= limit {
				break
			}
		}
	}
	return patterns, nil
}

func (o *orchestratorImpl) CreatePlan(ctx context.Context, req AppRequest) (*Plan, error) {
	slog.InfoContext(ctx, "creating plan", "app", req.AppName)

	o.emitProgress(ProgressEvent{
		TaskID:    "planning",
		AgentType: "orchestrator",
		Status:    "in_progress",
		Message:   "Creating architectural plan...",
		Progress:  10,
		Timestamp: time.Now(),
	})

	prompt := fmt.Sprintf(`Create a detailed architectural plan for: %s

Description: %s
Features: %v

Decompose into 4 tasks:
1. Frontend (Svelte) → /workspace/frontend/
2. Backend (Go) → /workspace/backend/
3. Database (PostgreSQL) → /workspace/database/
4. Tests → /workspace/tests/

Respond with clear task descriptions.`, req.AppName, req.Description, req.Features)

	var responseContent strings.Builder

	err := sdk.WithClient(ctx, func(client sdk.Client) error {
		if err := client.Query(ctx, prompt); err != nil {
			return err
		}

		msgChan := client.ReceiveMessages(ctx)
		for message := range msgChan {
			if message == nil {
				break
			}

			switch msg := message.(type) {
			case *sdk.AssistantMessage:
				for _, block := range msg.Content {
					if textBlock, ok := block.(*sdk.TextBlock); ok {
						responseContent.WriteString(textBlock.Text)
					}
				}
			case *sdk.ResultMessage:
				if msg.IsError {
					return fmt.Errorf("query failed")
				}
				return nil
			}
		}
		return nil
	},
		sdk.WithModel(string(sdk.AgentModelOpus)),
		sdk.WithMaxTurns(5),
	)

	if err != nil {
		o.emitProgress(ProgressEvent{
			TaskID:    "planning",
			AgentType: "orchestrator",
			Status:    "failed",
			Message:   fmt.Sprintf("Plan creation failed: %v", err),
			Progress:  0,
			Timestamp: time.Now(),
		})
		return nil, fmt.Errorf("create plan failed: %w", err)
	}

	plan := &Plan{
		Architecture: responseContent.String(),
		Tasks: []Task{
			{ID: "task-frontend", Type: AgentFrontend, Description: "Create Svelte frontend", Workspace: "/workspace/frontend/", Priority: 1},
			{ID: "task-backend", Type: AgentBackend, Description: "Create Go backend", Workspace: "/workspace/backend/", Priority: 1},
			{ID: "task-database", Type: AgentDatabase, Description: "Create PostgreSQL migrations", Workspace: "/workspace/database/", Priority: 2},
			{ID: "task-test", Type: AgentTest, Description: "Create tests", Workspace: "/workspace/tests/", Priority: 3},
		},
		CreatedAt: time.Now(),
	}

	o.emitProgress(ProgressEvent{
		TaskID:    "planning",
		AgentType: "orchestrator",
		Status:    "completed",
		Message:   "Plan created",
		Progress:  100,
		Timestamp: time.Now(),
	})

	slog.InfoContext(ctx, "plan created", "tasks", len(plan.Tasks))
	return plan, nil
}

func (o *orchestratorImpl) Execute(ctx context.Context, plan *Plan) (*GeneratedApp, error) {
	startTime := time.Now()
	slog.InfoContext(ctx, "executing plan", "tasks", len(plan.Tasks))

	// Check if orchestrator is shutting down
	select {
	case <-o.ctx.Done():
		return nil, fmt.Errorf("orchestrator is shutting down")
	default:
	}

	o.initWorkers()

	var wg sync.WaitGroup
	resultsChan := make(chan *AgentResult, len(plan.Tasks))
	errorsChan := make(chan error, len(plan.Tasks))

	for _, task := range plan.Tasks {
		wg.Add(1)
		o.wg.Add(1) // Track for graceful shutdown
		go func(t Task) {
			defer wg.Done()
			defer o.wg.Done()

			taskStartTime := time.Now()

			// Check for shutdown before executing
			select {
			case <-o.ctx.Done():
				errorsChan <- fmt.Errorf("task cancelled due to shutdown")
				return
			case <-ctx.Done():
				errorsChan <- fmt.Errorf("task context cancelled")
				return
			default:
			}

			// Trigger PreAgentExecute hook
			preHookCtx := hooks.HookContext{
				AgentType:  t.Type,
				TaskID:     t.ID,
				Input:      t.Description,
				Metadata:   map[string]interface{}{"workspace": t.Workspace, "priority": t.Priority},
				TokensUsed: 0,
				StartTime:  taskStartTime,
			}
			if err := o.hookRegistry.Trigger(ctx, hooks.PreAgentExecute, preHookCtx); err != nil {
				o.logger.WarnContext(ctx, "PreAgentExecute hook failed", "error", err)
			}

			// Execute worker with circuit breaker protection
			worker := o.workers[t.Type]
			var result *AgentResult
			var err error

			circuitBreakerErr := o.circuitBreaker.Execute(ctx, func(cbCtx context.Context) error {
				result, err = worker.Execute(cbCtx, t)
				return err
			})

			// Circuit breaker itself failed (circuit open)
			if circuitBreakerErr != nil && result == nil {
				o.logger.ErrorContext(ctx, "circuit breaker prevented execution",
					"task", t.ID,
					"agent", t.Type,
					"error", circuitBreakerErr,
				)
				errorsChan <- fmt.Errorf("circuit breaker: %w", circuitBreakerErr)
				return
			}

			endTime := time.Now()

			if err != nil {
				// Trigger OnAgentError hook
				errorHookCtx := hooks.HookContext{
					AgentType:  t.Type,
					TaskID:     t.ID,
					Input:      t.Description,
					Error:      err,
					Metadata:   map[string]interface{}{"workspace": t.Workspace, "priority": t.Priority},
					TokensUsed: 0, // TODO: Get actual token count from result
					StartTime:  taskStartTime,
					EndTime:    endTime,
				}
				if hookErr := o.hookRegistry.Trigger(ctx, hooks.OnAgentError, errorHookCtx); hookErr != nil {
					o.logger.WarnContext(ctx, "OnAgentError hook failed", "error", hookErr)
				}

				errorsChan <- err
				return
			}

			// Trigger PostAgentExecute hook
			postHookCtx := hooks.HookContext{
				AgentType:  t.Type,
				TaskID:     t.ID,
				Input:      t.Description,
				Output:     result.Output,
				Metadata:   map[string]interface{}{"workspace": t.Workspace, "priority": t.Priority, "files_created": len(result.FilesCreated)},
				TokensUsed: 0, // TODO: Get actual token count from result
				StartTime:  taskStartTime,
				EndTime:    endTime,
			}
			if hookErr := o.hookRegistry.Trigger(ctx, hooks.PostAgentExecute, postHookCtx); hookErr != nil {
				o.logger.WarnContext(ctx, "PostAgentExecute hook failed", "error", hookErr)
			}

			resultsChan <- result
		}(task)
	}

	wg.Wait()
	close(resultsChan)
	close(errorsChan)

	var results []AgentResult
	for result := range resultsChan {
		results = append(results, *result)
	}

	var errors []error
	for err := range errorsChan {
		errors = append(errors, err)
	}

	app := &GeneratedApp{
		AppName:       plan.Tasks[0].Description,
		Results:       results,
		Success:       len(errors) == 0,
		TotalDuration: time.Since(startTime),
		CreatedAt:     time.Now(),
	}

	if len(errors) > 0 {
		app.ErrorMessage = fmt.Sprintf("%d tasks failed", len(errors))
	}

	// Trigger OnComplete hook with overall execution summary
	completeHookCtx := hooks.HookContext{
		AgentType: "orchestrator",
		TaskID:    "execution-complete",
		Input:     fmt.Sprintf("Plan execution with %d tasks", len(plan.Tasks)),
		Output:    app,
		Metadata: map[string]interface{}{
			"total_tasks":     len(plan.Tasks),
			"successful":      len(results),
			"failed":          len(errors),
			"total_duration":  app.TotalDuration.Seconds(),
			"circuit_breaker": o.circuitBreaker.GetMetrics(),
		},
		TokensUsed: 0, // TODO: Aggregate token counts from all agents
		StartTime:  startTime,
		EndTime:    time.Now(),
	}
	if len(errors) > 0 {
		completeHookCtx.Error = fmt.Errorf("execution completed with %d errors", len(errors))
	}

	if err := o.hookRegistry.Trigger(ctx, hooks.OnComplete, completeHookCtx); err != nil {
		o.logger.WarnContext(ctx, "OnComplete hook failed", "error", err)
	}

	slog.InfoContext(ctx, "execution complete",
		"duration", app.TotalDuration,
		"circuit_breaker_state", o.circuitBreaker.GetState(),
	)
	return app, nil
}

func (o *orchestratorImpl) SetProgressCallback(callback ProgressCallback) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.progressCallback = callback
}

func (o *orchestratorImpl) emitProgress(event ProgressEvent) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	if o.progressCallback != nil {
		o.progressCallback(event)
	}
}

func (o *orchestratorImpl) initWorkers() {
	o.mu.Lock()
	defer o.mu.Unlock()

	if len(o.workers) > 0 {
		return
	}

	o.workers[AgentFrontend] = NewWorker(AgentFrontend, o.progressCallback)
	o.workers[AgentBackend] = NewWorker(AgentBackend, o.progressCallback)
	o.workers[AgentDatabase] = NewWorker(AgentDatabase, o.progressCallback)
	o.workers[AgentTest] = NewWorker(AgentTest, o.progressCallback)
}

// Shutdown gracefully shuts down the orchestrator
// Waits for all running workers to complete before returning
func (o *orchestratorImpl) Shutdown() error {
	o.logger.Info("orchestrator shutting down, waiting for workers to complete...")

	// Cancel context to signal all workers to stop
	o.cancel()

	// Wait for all workers to finish (with timeout)
	done := make(chan struct{})
	go func() {
		o.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		o.logger.Info("orchestrator shutdown complete, all workers finished")
		return nil
	case <-time.After(30 * time.Second):
		o.logger.Warn("orchestrator shutdown timeout, some workers may still be running")
		return fmt.Errorf("shutdown timeout after 30s")
	}
}

// GetCircuitBreakerMetrics returns current circuit breaker state
func (o *orchestratorImpl) GetCircuitBreakerMetrics() map[string]interface{} {
	return o.circuitBreaker.GetMetrics()
}
