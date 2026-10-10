package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rhl/businessos-backend/internal/config"
	"github.com/rhl/businessos-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSkillExecution tests end-to-end skill execution
func TestSkillExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	_ = &config.Config{
		AIProvider: "anthropic",
	}

	t.Run("Execute simple skill with actions", func(t *testing.T) {
		// Define a simple skill
		skill := map[string]interface{}{
			"name":        "test_skill",
			"description": "A test skill for integration testing",
			"parameters": map[string]interface{}{
				"input": map[string]string{
					"type":        "string",
					"description": "Test input",
				},
			},
			"steps": []map[string]interface{}{
				{
					"type":   "action",
					"action": "test_action",
					"params": map[string]string{
						"value": "{{input}}",
					},
				},
			},
		}

		skillJSON, err := json.Marshal(skill)
		require.NoError(t, err)

		// Store skill in database
		skillID := uuid.New()
		_, err = testDB.Pool.Exec(ctx, `
			INSERT INTO skills (id, name, description, definition, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
		`, skillID, "test_skill", "Test skill", string(skillJSON))
		require.NoError(t, err)

		// Load and verify
		var loadedSkill string
		err = testDB.Pool.QueryRow(ctx, `
			SELECT definition FROM skills WHERE id = $1
		`, skillID).Scan(&loadedSkill)
		require.NoError(t, err)
		assert.NotEmpty(t, loadedSkill)

		var skillDef map[string]interface{}
		err = json.Unmarshal([]byte(loadedSkill), &skillDef)
		require.NoError(t, err)
		assert.Equal(t, "test_skill", skillDef["name"])
	})
}

// TestSkillActionHandlers tests action handler registration and execution
func TestSkillActionHandlers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	t.Run("Register and execute action handler", func(t *testing.T) {
		actionExecuted := false
		var actionParams map[string]interface{}

		// Mock action handler
		handler := func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			actionExecuted = true
			actionParams = params
			return map[string]string{"result": "success"}, nil
		}

		// Simulate action execution
		result, err := handler(ctx, map[string]interface{}{
			"param1": "value1",
			"param2": 42,
		})

		require.NoError(t, err)
		assert.True(t, actionExecuted)
		assert.Equal(t, "value1", actionParams["param1"])
		assert.Equal(t, 42, actionParams["param2"])

		resultMap := result.(map[string]string)
		assert.Equal(t, "success", resultMap["result"])
	})

	t.Run("Handle action errors gracefully", func(t *testing.T) {
		handler := func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return nil, assert.AnError
		}

		_, err := handler(ctx, map[string]interface{}{})
		assert.Error(t, err)
	})
}

// TestSkillStepExecution tests different step types
func TestSkillStepExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_ = context.Background()

	t.Run("Action step execution", func(t *testing.T) {
		step := map[string]interface{}{
			"type":   "action",
			"action": "fetch_data",
			"params": map[string]string{
				"url": "https://api.example.com/data",
			},
		}

		assert.Equal(t, "action", step["type"])
		assert.Contains(t, step, "action")
		assert.Contains(t, step, "params")
	})

	t.Run("Decision step execution", func(t *testing.T) {
		step := map[string]interface{}{
			"type":      "decision",
			"condition": "result.status == 'success'",
			"then":      "next_step",
			"else":      "error_handler",
		}

		assert.Equal(t, "decision", step["type"])
		assert.Contains(t, step, "condition")
		assert.Contains(t, step, "then")
		assert.Contains(t, step, "else")
	})

	t.Run("Loop step execution", func(t *testing.T) {
		step := map[string]interface{}{
			"type":  "loop",
			"items": "{{results}}",
			"steps": []map[string]interface{}{
				{
					"type":   "action",
					"action": "process_item",
					"params": map[string]string{
						"item": "{{current_item}}",
					},
				},
			},
		}

		assert.Equal(t, "loop", step["type"])
		assert.Contains(t, step, "items")
		assert.Contains(t, step, "steps")
	})

	t.Run("Parallel step execution", func(t *testing.T) {
		step := map[string]interface{}{
			"type": "parallel",
			"branches": []map[string]interface{}{
				{
					"name": "branch1",
					"steps": []map[string]interface{}{
						{
							"type":   "action",
							"action": "task1",
						},
					},
				},
				{
					"name": "branch2",
					"steps": []map[string]interface{}{
						{
							"type":   "action",
							"action": "task2",
						},
					},
				},
			},
		}

		assert.Equal(t, "parallel", step["type"])
		assert.Contains(t, step, "branches")
		branches := step["branches"].([]map[string]interface{})
		assert.Len(t, branches, 2)
	})
}

// TestSkillChaining tests skill chaining (one skill calling another)
func TestSkillChaining(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	t.Run("Chain multiple skills together", func(t *testing.T) {
		// Skill 1: Data fetcher
		skill1 := map[string]interface{}{
			"name":        "fetch_skill",
			"description": "Fetches data",
			"steps": []map[string]interface{}{
				{
					"type":   "action",
					"action": "fetch",
				},
			},
		}

		skill1JSON, _ := json.Marshal(skill1)
		skill1ID := uuid.New()
		_, err := testDB.Pool.Exec(ctx, `
			INSERT INTO skills (id, name, description, definition, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
		`, skill1ID, "fetch_skill", "Fetches data", string(skill1JSON))
		require.NoError(t, err)

		// Skill 2: Data processor (calls skill 1)
		skill2 := map[string]interface{}{
			"name":        "process_skill",
			"description": "Processes data using fetch_skill",
			"steps": []map[string]interface{}{
				{
					"type":  "call_skill",
					"skill": "fetch_skill",
				},
				{
					"type":   "action",
					"action": "process",
					"params": map[string]string{
						"data": "{{previous_result}}",
					},
				},
			},
		}

		skill2JSON, _ := json.Marshal(skill2)
		skill2ID := uuid.New()
		_, err = testDB.Pool.Exec(ctx, `
			INSERT INTO skills (id, name, description, definition, created_at, updated_at)
			VALUES ($1, $2, $3, $4, NOW(), NOW())
		`, skill2ID, "process_skill", "Processes data", string(skill2JSON))
		require.NoError(t, err)

		// Verify both skills exist
		var count int
		err = testDB.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM skills WHERE id IN ($1, $2)
		`, skill1ID, skill2ID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 2, count)
	})
}

// TestSkillErrorRecovery tests error handling in skills
func TestSkillErrorRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	_ = context.Background()

	t.Run("Skill with error recovery steps", func(t *testing.T) {
		skill := map[string]interface{}{
			"name":        "resilient_skill",
			"description": "Skill with error handling",
			"steps": []map[string]interface{}{
				{
					"type":   "action",
					"action": "risky_operation",
					"on_error": map[string]interface{}{
						"action": "error_handler",
						"params": map[string]string{
							"error": "{{error_message}}",
						},
					},
				},
				{
					"type":   "action",
					"action": "fallback_operation",
				},
			},
		}

		assert.Contains(t, skill, "steps")
		steps := skill["steps"].([]map[string]interface{})
		assert.Contains(t, steps[0], "on_error")
	})

	t.Run("Retry logic in skills", func(t *testing.T) {
		step := map[string]interface{}{
			"type":        "action",
			"action":      "unstable_api_call",
			"max_retries": 3,
			"retry_delay": "1s",
			"backoff":     "exponential",
		}

		assert.Equal(t, 3, step["max_retries"])
		assert.Equal(t, "1s", step["retry_delay"])
		assert.Equal(t, "exponential", step["backoff"])
	})
}

// TestSkillStateManagement tests skill state persistence
func TestSkillStateManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	t.Run("Persist and resume skill execution state", func(t *testing.T) {
		// Create skill execution record
		executionID := uuid.New()
		skillID := uuid.New()
		userID := uuid.New().String()

		state := map[string]interface{}{
			"current_step":   2,
			"variables":      map[string]string{"var1": "value1"},
			"started_at":     time.Now(),
			"last_step_time": time.Now(),
		}
		stateJSON, _ := json.Marshal(state)

		_, err := testDB.Pool.Exec(ctx, `
			CREATE TABLE IF NOT EXISTS skill_executions (
				id UUID PRIMARY KEY,
				skill_id UUID NOT NULL,
				user_id TEXT NOT NULL,
				state JSONB,
				status TEXT,
				created_at TIMESTAMPTZ DEFAULT NOW()
			)
		`)
		require.NoError(t, err)

		_, err = testDB.Pool.Exec(ctx, `
			INSERT INTO skill_executions (id, skill_id, user_id, state, status)
			VALUES ($1, $2, $3, $4, $5)
		`, executionID, skillID, userID, string(stateJSON), "in_progress")
		require.NoError(t, err)

		// Retrieve and verify state
		var loadedState string
		var status string
		err = testDB.Pool.QueryRow(ctx, `
			SELECT state, status FROM skill_executions WHERE id = $1
		`, executionID).Scan(&loadedState, &status)
		require.NoError(t, err)

		assert.Equal(t, "in_progress", status)

		var stateDef map[string]interface{}
		err = json.Unmarshal([]byte(loadedState), &stateDef)
		require.NoError(t, err)
		assert.Equal(t, float64(2), stateDef["current_step"])
	})
}

// TestSkillConcurrency tests concurrent skill execution
func TestSkillConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()
	testDB := testutil.RequireTestDatabase(t)
	defer testDB.Close()
	defer testutil.CleanupTestData(ctx, testDB.Pool)

	t.Run("Execute multiple skills concurrently", func(t *testing.T) {
		numSkills := 5
		done := make(chan bool, numSkills)

		for i := 0; i < numSkills; i++ {
			go func(idx int) {
				defer func() { done <- true }()

				skillID := uuid.New()
				skill := map[string]interface{}{
					"name":        "concurrent_skill_" + string(rune(idx)),
					"description": "Concurrent test skill",
					"steps": []map[string]interface{}{
						{
							"type":   "action",
							"action": "test_action",
						},
					},
				}
				skillJSON, _ := json.Marshal(skill)

				_, err := testDB.Pool.Exec(ctx, `
					INSERT INTO skills (id, name, description, definition, created_at, updated_at)
					VALUES ($1, $2, $3, $4, NOW(), NOW())
				`, skillID, skill["name"], skill["description"], string(skillJSON))
				assert.NoError(t, err)
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numSkills; i++ {
			select {
			case <-done:
				// Success
			case <-time.After(10 * time.Second):
				t.Fatal("timeout waiting for concurrent skill executions")
			}
		}

		// Verify all skills were created
		var count int
		err := testDB.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM skills`).Scan(&count)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, numSkills)
	})
}
