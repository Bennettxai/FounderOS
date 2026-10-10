package skills

import (
	"context"
	"fmt"
	"testing"
)

// mockSkill is a test implementation of the Skill interface
type mockSkill struct {
	name        string
	description string
	executeFunc func(ctx context.Context, params map[string]interface{}) (interface{}, error)
}

func (m *mockSkill) Name() string {
	return m.name
}

func (m *mockSkill) Description() string {
	return m.description
}

func (m *mockSkill) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, params)
	}
	return "mock result", nil
}

func (m *mockSkill) Schema() *SkillSchema {
	return nil
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name      string
		skill     Skill
		wantError bool
	}{
		{
			name: "register valid skill",
			skill: &mockSkill{
				name:        "test_skill",
				description: "A test skill",
			},
			wantError: false,
		},
		{
			name:      "register nil skill",
			skill:     nil,
			wantError: true,
		},
		{
			name: "register skill with empty name",
			skill: &mockSkill{
				name:        "",
				description: "No name skill",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			err := registry.Register(tt.skill)

			if (err != nil) != tt.wantError {
				t.Errorf("Register() error = %v, wantError %v", err, tt.wantError)
			}

			if !tt.wantError && tt.skill != nil {
				// Verify skill was registered
				if !registry.HasSkill(tt.skill.Name()) {
					t.Errorf("Skill %q was not registered", tt.skill.Name())
				}
			}
		})
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewRegistry()

	skill := &mockSkill{
		name:        "duplicate",
		description: "Test skill",
	}

	// First registration should succeed
	err := registry.Register(skill)
	if err != nil {
		t.Fatalf("First registration failed: %v", err)
	}

	// Second registration should fail
	err = registry.Register(skill)
	if err == nil {
		t.Error("Expected error when registering duplicate skill")
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	skill := &mockSkill{
		name:        "test_get",
		description: "Test get skill",
	}

	// Register skill
	err := registry.Register(skill)
	if err != nil {
		t.Fatalf("Failed to register skill: %v", err)
	}

	// Get existing skill
	retrieved, err := registry.Get("test_get")
	if err != nil {
		t.Errorf("Failed to get skill: %v", err)
	}
	if retrieved == nil {
		t.Error("Retrieved skill is nil")
	}
	if retrieved.Name() != "test_get" {
		t.Errorf("Retrieved skill name = %q, want %q", retrieved.Name(), "test_get")
	}

	// Get non-existent skill
	_, err = registry.Get("non_existent")
	if err == nil {
		t.Error("Expected error when getting non-existent skill")
	}
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// Register multiple skills
	skills := []*mockSkill{
		{name: "skill1", description: "First skill"},
		{name: "skill2", description: "Second skill"},
		{name: "skill3", description: "Third skill"},
	}

	for _, skill := range skills {
		err := registry.Register(skill)
		if err != nil {
			t.Fatalf("Failed to register skill %q: %v", skill.Name(), err)
		}
	}

	// List all skills
	metadata := registry.List()
	if len(metadata) != len(skills) {
		t.Errorf("List() returned %d skills, want %d", len(metadata), len(skills))
	}

	// Verify all skills are in the list
	nameMap := make(map[string]bool)
	for _, m := range metadata {
		nameMap[m.Name] = true
	}

	for _, skill := range skills {
		if !nameMap[skill.Name()] {
			t.Errorf("Skill %q not found in List() output", skill.Name())
		}
	}
}

func TestRegistry_Execute(t *testing.T) {
	registry := NewRegistry()

	// Register a skill with custom execute function
	executeCalled := false
	skill := &mockSkill{
		name:        "execute_test",
		description: "Test execute",
		executeFunc: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			executeCalled = true
			return "success", nil
		},
	}

	err := registry.Register(skill)
	if err != nil {
		t.Fatalf("Failed to register skill: %v", err)
	}

	// Execute the skill
	ctx := context.Background()
	result, err := registry.Execute(ctx, "execute_test", nil)
	if err != nil {
		t.Errorf("Execute() failed: %v", err)
	}

	if !executeCalled {
		t.Error("Skill execute function was not called")
	}

	if result == nil {
		t.Error("Execute() returned nil result")
	}

	if !result.Success {
		t.Error("Execute() result.Success = false, want true")
	}

	// Execute non-existent skill
	_, err = registry.Execute(ctx, "non_existent", nil)
	if err == nil {
		t.Error("Expected error when executing non-existent skill")
	}
}

func TestRegistry_ExecuteWithError(t *testing.T) {
	registry := NewRegistry()

	// Register a skill that returns an error
	expectedErr := fmt.Errorf("test error")
	skill := &mockSkill{
		name:        "error_skill",
		description: "Skill that errors",
		executeFunc: func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
			return nil, expectedErr
		},
	}

	err := registry.Register(skill)
	if err != nil {
		t.Fatalf("Failed to register skill: %v", err)
	}

	// Execute the skill
	ctx := context.Background()
	result, err := registry.Execute(ctx, "error_skill", nil)
	if err == nil {
		t.Error("Expected error from Execute()")
	}

	if result == nil {
		t.Error("Execute() should return result even on error")
	}

	if result.Success {
		t.Error("Execute() result.Success = true, want false")
	}

	if result.Error == "" {
		t.Error("Execute() result.Error is empty")
	}
}

func TestRegistry_Unregister(t *testing.T) {
	registry := NewRegistry()

	skill := &mockSkill{
		name:        "unregister_test",
		description: "Test unregister",
	}

	// Register skill
	err := registry.Register(skill)
	if err != nil {
		t.Fatalf("Failed to register skill: %v", err)
	}

	// Verify it's registered
	if !registry.HasSkill("unregister_test") {
		t.Error("Skill not found after registration")
	}

	// Unregister it
	err = registry.Unregister("unregister_test")
	if err != nil {
		t.Errorf("Unregister() failed: %v", err)
	}

	// Verify it's gone
	if registry.HasSkill("unregister_test") {
		t.Error("Skill still found after unregistration")
	}

	// Try to unregister non-existent skill
	err = registry.Unregister("non_existent")
	if err == nil {
		t.Error("Expected error when unregistering non-existent skill")
	}
}

func TestRegistry_Count(t *testing.T) {
	registry := NewRegistry()

	if registry.Count() != 0 {
		t.Errorf("New registry count = %d, want 0", registry.Count())
	}

	// Add skills
	for i := 0; i < 5; i++ {
		skill := &mockSkill{
			name:        fmt.Sprintf("skill%d", i),
			description: fmt.Sprintf("Skill %d", i),
		}
		err := registry.Register(skill)
		if err != nil {
			t.Fatalf("Failed to register skill: %v", err)
		}
	}

	if registry.Count() != 5 {
		t.Errorf("Registry count = %d, want 5", registry.Count())
	}
}

func TestRegistry_Concurrency(t *testing.T) {
	registry := NewRegistry()

	// Test concurrent registration
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(n int) {
			skill := &mockSkill{
				name:        fmt.Sprintf("concurrent_%d", n),
				description: fmt.Sprintf("Concurrent skill %d", n),
			}
			registry.Register(skill)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// All skills should be registered
	if registry.Count() != 10 {
		t.Errorf("Registry count = %d, want 10", registry.Count())
	}

	// Test concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			registry.List()
			registry.HasSkill("concurrent_0")
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
