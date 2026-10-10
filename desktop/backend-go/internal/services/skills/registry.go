package skills

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

// Registry manages the collection of available skills
type Registry struct {
	mu     sync.RWMutex
	skills map[string]Skill
}

// NewRegistry creates a new skill registry
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]Skill),
	}
}

// Register adds a skill to the registry
func (r *Registry) Register(skill Skill) error {
	if skill == nil {
		return fmt.Errorf("cannot register nil skill")
	}

	name := skill.Name()
	if name == "" {
		return fmt.Errorf("skill name cannot be empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[name]; exists {
		return fmt.Errorf("skill %q is already registered", name)
	}

	r.skills[name] = skill
	slog.Info("skill registered", "name", name, "description", skill.Description())
	return nil
}

// Unregister removes a skill from the registry
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[name]; !exists {
		return fmt.Errorf("skill %q is not registered", name)
	}

	delete(r.skills, name)
	slog.Info("skill unregistered", "name", name)
	return nil
}

// Get retrieves a skill by name
func (r *Registry) Get(name string) (Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skill, exists := r.skills[name]
	if !exists {
		return nil, fmt.Errorf("skill %q not found", name)
	}

	return skill, nil
}

// List returns metadata for all registered skills
func (r *Registry) List() []SkillMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata := make([]SkillMetadata, 0, len(r.skills))
	for _, skill := range r.skills {
		metadata = append(metadata, SkillMetadata{
			Name:        skill.Name(),
			Description: skill.Description(),
			Schema:      skill.Schema(),
		})
	}

	return metadata
}

// Execute runs a skill by name with the given parameters
func (r *Registry) Execute(ctx context.Context, name string, params map[string]interface{}) (*SkillExecutionResult, error) {
	skill, err := r.Get(name)
	if err != nil {
		return nil, err
	}

	slog.Info("executing skill", "name", name)

	result, err := skill.Execute(ctx, params)
	if err != nil {
		slog.Error("skill execution failed", "name", name, "error", err)
		return &SkillExecutionResult{
			SkillName: name,
			Success:   false,
			Error:     err.Error(),
		}, err
	}

	slog.Info("skill execution succeeded", "name", name)
	return &SkillExecutionResult{
		SkillName: name,
		Success:   true,
		Result:    result,
	}, nil
}

// HasSkill checks if a skill is registered
func (r *Registry) HasSkill(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.skills[name]
	return exists
}

// Count returns the number of registered skills
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.skills)
}
