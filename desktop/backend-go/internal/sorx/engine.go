// Package sorx implements the Sorx skill execution engine.
// Sorx (System of Reasoning) executes skills using connected integrations.
package sorx

import (
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/carrier"
)

// Engine is the Sorx skill execution engine.
type Engine struct {
	pool    *pgxpool.Pool
	logger  *slog.Logger
	carrier *carrier.Client

	// Execution tracking
	executions sync.Map // map[string]*Execution

	// Skill registry
	skills sync.Map // map[string]*SkillDefinition

	// Event bus for async communication
	events chan Event
	done   chan struct{}
}

// NewEngine creates a new Sorx engine.
func NewEngine(pool *pgxpool.Pool, logger *slog.Logger) *Engine {
	e := &Engine{
		pool:   pool,
		logger: logger,
		events: make(chan Event, 100),
		done:   make(chan struct{}),
	}

	// Register built-in skills
	e.registerBuiltinSkills()

	// Start event processor
	go e.processEvents()

	return e
}

// Close shuts down the engine gracefully.
func (e *Engine) Close() {
	close(e.done)
	close(e.events)
}

// SetCarrierClient wires a CARRIER AMQP client into the engine for Tier 3-4
// skill routing. This setter exists because main.go constructs the engine
// before the carrier client is ready; call it once during startup before any
// skills are executed.
//
// Passing nil disables CARRIER routing; Tier 3-4 calls will fall back to the
// local Groq LLM.
func (e *Engine) SetCarrierClient(c *carrier.Client) {
	e.carrier = c
	setEngineCarrier(c)
	e.logger.Info("carrier client registered with sorx engine",
		"connected", c != nil && c.IsConnected())
}

// RegisterSkill adds a skill to the registry.
func (e *Engine) RegisterSkill(skill *SkillDefinition) {
	e.skills.Store(skill.ID, skill)
	e.logger.Info("Registered skill", "skill_id", skill.ID, "name", skill.Name)
}

// ListSkills returns all registered skills.
func (e *Engine) ListSkills() []*SkillDefinition {
	var skills []*SkillDefinition
	e.skills.Range(func(key, value interface{}) bool {
		skills = append(skills, value.(*SkillDefinition))
		return true
	})
	return skills
}

// timePtr returns a pointer to a time.Time value.
func timePtr(t time.Time) *time.Time {
	return &t
}
