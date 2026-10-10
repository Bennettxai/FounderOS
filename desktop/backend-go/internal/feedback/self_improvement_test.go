package feedback

import (
	"context"
	"testing"
)

// TestNewSelfImprovementEngine verifies the constructor succeeds with and
// without a Redis client.
func TestNewSelfImprovementEngine(t *testing.T) {
	t.Run("nil redis", func(t *testing.T) {
		engine := NewSelfImprovementEngine(nil, nil)
		if engine == nil {
			t.Fatal("expected non-nil engine")
		}
		if engine.alpha != 0.15 {
			t.Errorf("expected alpha=0.15, got %f", engine.alpha)
		}
		if engine.gamma != 0.85 {
			t.Errorf("expected gamma=0.85, got %f", engine.gamma)
		}
		if engine.qTable == nil {
			t.Fatal("expected non-nil qTable")
		}
	})

	t.Run("logger is set", func(t *testing.T) {
		engine := NewSelfImprovementEngine(nil, nil)
		if engine.logger == nil {
			t.Fatal("expected non-nil logger")
		}
	})
}

// TestShouldAutoApply verifies the auto-apply threshold logic.
func TestShouldAutoApply(t *testing.T) {
	cases := []struct {
		name       string
		confidence float64
		impact     float64
		want       bool
	}{
		{"both above threshold", 9.0, 0.25, true},
		{"confidence high, impact exact", 9.5, 0.25, true},
		{"confidence exact, impact high", 9.0, 0.9, true},
		{"confidence just below", 8.99, 0.25, false},
		{"impact just below", 9.0, 0.24, false},
		{"both below", 5.0, 0.1, false},
		{"zero values", 0.0, 0.0, false},
		{"confidence max, impact zero", 10.0, 0.0, false},
		{"confidence zero, impact max", 0.0, 1.0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := ImprovementSuggestion{
				Confidence: tc.confidence,
				Impact:     tc.impact,
			}
			got := shouldAutoApply(s)
			if got != tc.want {
				t.Errorf("shouldAutoApply(confidence=%.2f, impact=%.2f) = %v, want %v",
					tc.confidence, tc.impact, got, tc.want)
			}
		})
	}
}

// TestDefaultRewardWeights ensures all fields are non-zero and
// LatencyOptimization is 0.8 while the rest are 1.0.
func TestDefaultRewardWeights(t *testing.T) {
	w := defaultRewardWeights()

	fields := map[ImprovementType]float64{
		ImprovementTypePromptRefinement:    w.PromptRefinement,
		ImprovementTypeContextExpansion:    w.ContextExpansion,
		ImprovementTypeReasoningDepth:      w.ReasoningDepth,
		ImprovementTypeToolSelection:       w.ToolSelection,
		ImprovementTypeResponseFormat:      w.ResponseFormat,
		ImprovementTypeErrorRecovery:       w.ErrorRecovery,
		ImprovementTypeLatencyOptimization: w.LatencyOptimization,
	}

	for impType, val := range fields {
		if val <= 0 {
			t.Errorf("weight for %s must be > 0, got %f", impType, val)
		}
	}

	if w.LatencyOptimization != 0.8 {
		t.Errorf("LatencyOptimization weight: want 0.8, got %f", w.LatencyOptimization)
	}

	for _, impType := range []ImprovementType{
		ImprovementTypePromptRefinement,
		ImprovementTypeContextExpansion,
		ImprovementTypeReasoningDepth,
		ImprovementTypeToolSelection,
		ImprovementTypeResponseFormat,
		ImprovementTypeErrorRecovery,
	} {
		if fields[impType] != 1.0 {
			t.Errorf("weight for %s: want 1.0, got %f", impType, fields[impType])
		}
	}
}

// TestGenerateSuggestions verifies the engine produces one suggestion per
// improvement type when Redis is nil (fallback path).
func TestGenerateSuggestions(t *testing.T) {
	ctx := context.Background()
	engine := NewSelfImprovementEngine(nil, nil)

	suggestions, err := engine.GenerateSuggestions(ctx,
		"tenant-abc", "orchestrator", "user asked about project planning", "positive")
	if err != nil {
		t.Fatalf("GenerateSuggestions returned error: %v", err)
	}

	if len(suggestions) != len(allImprovementTypes) {
		t.Errorf("expected %d suggestions, got %d", len(allImprovementTypes), len(suggestions))
	}

	seenTypes := make(map[ImprovementType]bool)
	for _, s := range suggestions {
		if s.ID == "" {
			t.Error("suggestion ID must not be empty")
		}
		if s.TenantID != "tenant-abc" {
			t.Errorf("unexpected tenant_id %q", s.TenantID)
		}
		if s.Confidence < 0 || s.Confidence > 10 {
			t.Errorf("confidence %f out of [0,10] range for type %s", s.Confidence, s.Type)
		}
		if s.Impact < 0 || s.Impact > 1 {
			t.Errorf("impact %f out of [0,1] range for type %s", s.Impact, s.Type)
		}
		if s.CreatedAt.IsZero() {
			t.Errorf("CreatedAt must not be zero for type %s", s.Type)
		}
		seenTypes[s.Type] = true
	}

	for _, impType := range allImprovementTypes {
		if !seenTypes[impType] {
			t.Errorf("missing suggestion for type %s", impType)
		}
	}
}

// TestGenerateSuggestions_AutoApplyFlagSet verifies that after a Q-value
// update pushes a type above the auto-apply threshold, the flag is set.
func TestGenerateSuggestions_AutoApplyFlagSet(t *testing.T) {
	ctx := context.Background()
	engine := NewSelfImprovementEngine(nil, nil)

	// Drive a very high Q-value for prompt_refinement so confidence reaches 9+.
	ctxToken := contextToken("hot context")
	state := QState{
		Context:         ctxToken,
		AgentType:       "orchestrator",
		ImprovementType: ImprovementTypePromptRefinement,
	}
	// Repeated updates with reward=1.0 will converge Q toward 1.0;
	// confidence = Q*10 so Q >= 0.9 gives confidence >= 9.0.
	for i := 0; i < 60; i++ {
		if err := engine.UpdateQValue(ctx, state, 1.0); err != nil {
			t.Fatalf("UpdateQValue error: %v", err)
		}
	}

	suggestions, err := engine.GenerateSuggestions(ctx,
		"tenant-abc", "orchestrator", "hot context", "")
	if err != nil {
		t.Fatalf("GenerateSuggestions returned error: %v", err)
	}

	var found bool
	for _, s := range suggestions {
		if s.Type == ImprovementTypePromptRefinement {
			found = true
			if s.Confidence < 9.0 {
				t.Errorf("expected confidence >= 9.0, got %f", s.Confidence)
			}
			// impact = (confidence/10) * weight(1.0) >= 0.9*1.0 = 0.9 >= 0.25
			if !s.AutoApply {
				t.Errorf("expected AutoApply=true for high-Q suggestion")
			}
		}
	}
	if !found {
		t.Fatal("prompt_refinement suggestion not found")
	}
}

// TestUpdateQValue verifies Bellman update convergence.
func TestUpdateQValue(t *testing.T) {
	ctx := context.Background()
	engine := NewSelfImprovementEngine(nil, nil)

	state := QState{
		Context:         "ctx",
		AgentType:       "agent",
		ImprovementType: ImprovementTypeReasoningDepth,
	}

	// After many updates with reward=1.0, Q should converge toward 1.0.
	for i := 0; i < 100; i++ {
		if err := engine.UpdateQValue(ctx, state, 1.0); err != nil {
			t.Fatalf("UpdateQValue error on iteration %d: %v", i, err)
		}
	}

	key := stateKey(state)
	engine.mu.RLock()
	q := engine.qTable[key][state.ImprovementType]
	engine.mu.RUnlock()

	if q < 0.95 {
		t.Errorf("expected Q-value to converge near 1.0 after 100 updates, got %f", q)
	}
}

// TestEnqueueAndList verifies that Enqueue stores a task and List returns it
// for the correct tenant.
func TestEnqueueAndList(t *testing.T) {
	ctx := context.Background()
	engine := NewSelfImprovementEngine(nil, nil)
	queue := NewImprovementTaskQueue(engine, nil)

	suggestion := ImprovementSuggestion{
		Type:        ImprovementTypeContextExpansion,
		Description: "Expand context for better recall",
		Confidence:  7.0,
		Impact:      0.5,
		TenantID:    "tenant-xyz",
	}

	taskID, err := queue.Enqueue(ctx, "tenant-xyz", suggestion)
	if err != nil {
		t.Fatalf("Enqueue error: %v", err)
	}
	if taskID == "" {
		t.Fatal("expected non-empty task ID")
	}

	tasks, err := queue.List(ctx, "tenant-xyz")
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].ID != taskID {
		t.Errorf("task ID mismatch: want %s, got %s", taskID, tasks[0].ID)
	}
	if tasks[0].Status != TaskStatusPending {
		t.Errorf("expected status=pending, got %s", tasks[0].Status)
	}
	if tasks[0].TenantID != "tenant-xyz" {
		t.Errorf("expected tenant_id=tenant-xyz, got %s", tasks[0].TenantID)
	}
}

// TestEnqueue_EmptyTenantID verifies that an empty tenantID is rejected.
func TestEnqueue_EmptyTenantID(t *testing.T) {
	ctx := context.Background()
	engine := NewSelfImprovementEngine(nil, nil)
	queue := NewImprovementTaskQueue(engine, nil)

	_, err := queue.Enqueue(ctx, "", ImprovementSuggestion{})
	if err == nil {
		t.Fatal("expected error for empty tenantID")
	}
}

// TestGetStatus verifies GetStatus returns the correct task and errors on
// unknown IDs.
func TestGetStatus(t *testing.T) {
	ctx := context.Background()
	engine := NewSelfImprovementEngine(nil, nil)
	queue := NewImprovementTaskQueue(engine, nil)

	suggestion := ImprovementSuggestion{
		Type:       ImprovementTypeErrorRecovery,
		Confidence: 6.0,
		Impact:     0.3,
		TenantID:   "t1",
	}
	taskID, _ := queue.Enqueue(ctx, "t1", suggestion)

	task, err := queue.GetStatus(ctx, taskID)
	if err != nil {
		t.Fatalf("GetStatus error: %v", err)
	}
	if task.ID != taskID {
		t.Errorf("ID mismatch: want %s, got %s", taskID, task.ID)
	}

	_, err = queue.GetStatus(ctx, "nonexistent-id")
	if err == nil {
		t.Fatal("expected error for unknown task ID")
	}
}

// TestProcess_AutoApplyEligible verifies that Process applies auto-eligible
// tasks and transitions them to TaskStatusApplied.
func TestProcess_AutoApplyEligible(t *testing.T) {
	ctx := context.Background()
	engine := NewSelfImprovementEngine(nil, nil)
	queue := NewImprovementTaskQueue(engine, nil)

	// Suggestion above auto-apply threshold.
	suggestion := ImprovementSuggestion{
		Type:       ImprovementTypePromptRefinement,
		Confidence: 9.5,
		Impact:     0.8,
		AutoApply:  true,
		TenantID:   "t2",
	}
	taskID, _ := queue.Enqueue(ctx, "t2", suggestion)

	if err := queue.Process(ctx); err != nil {
		t.Fatalf("Process error: %v", err)
	}

	task, err := queue.GetStatus(ctx, taskID)
	if err != nil {
		t.Fatalf("GetStatus error: %v", err)
	}
	if task.Status != TaskStatusApplied {
		t.Errorf("expected status=applied, got %s", task.Status)
	}
}

// TestProcess_NonAutoApplyRemainsPending verifies that Process leaves
// non-auto-eligible tasks in pending state.
func TestProcess_NonAutoApplyRemainsPending(t *testing.T) {
	ctx := context.Background()
	engine := NewSelfImprovementEngine(nil, nil)
	queue := NewImprovementTaskQueue(engine, nil)

	suggestion := ImprovementSuggestion{
		Type:       ImprovementTypeLatencyOptimization,
		Confidence: 4.0, // below 9.0 threshold
		Impact:     0.1, // below 0.25 threshold
		AutoApply:  false,
		TenantID:   "t3",
	}
	taskID, _ := queue.Enqueue(ctx, "t3", suggestion)

	if err := queue.Process(ctx); err != nil {
		t.Fatalf("Process error: %v", err)
	}

	task, _ := queue.GetStatus(ctx, taskID)
	if task.Status != TaskStatusPending {
		t.Errorf("expected status=pending for non-auto-apply task, got %s", task.Status)
	}
}

// TestList_TenantIsolation verifies that List returns only tasks for the
// requested tenant.
func TestList_TenantIsolation(t *testing.T) {
	ctx := context.Background()
	engine := NewSelfImprovementEngine(nil, nil)
	queue := NewImprovementTaskQueue(engine, nil)

	for _, tid := range []string{"alpha", "beta", "beta"} {
		_, err := queue.Enqueue(ctx, tid, ImprovementSuggestion{
			Type:     ImprovementTypeToolSelection,
			TenantID: tid,
		})
		if err != nil {
			t.Fatalf("Enqueue error for tenant %s: %v", tid, err)
		}
	}

	alphaTasks, _ := queue.List(ctx, "alpha")
	if len(alphaTasks) != 1 {
		t.Errorf("alpha: expected 1 task, got %d", len(alphaTasks))
	}

	betaTasks, _ := queue.List(ctx, "beta")
	if len(betaTasks) != 2 {
		t.Errorf("beta: expected 2 tasks, got %d", len(betaTasks))
	}
}
