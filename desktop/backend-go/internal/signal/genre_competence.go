package signal

// GenreCompetence maps an agent type + genre combination to context strategy.
type GenreCompetence struct {
	Agent        string   // agent type identifier
	Genre        Genre    // which genre this competence applies to
	ContextHints []string // what context to pull: "project", "client", "documents", "metrics", "team", "memory", "knowledge"
	DocTypes     []string // eligible document types for this genre
}

// CompetenceRegistry maps (AgentType, Genre) to competence entries.
type CompetenceRegistry struct {
	entries map[string]map[Genre]*GenreCompetence
}

// NewCompetenceRegistry creates a pre-populated CompetenceRegistry with defaults.
// Since the Orchestrator handles all traffic (self-routing), defaults are broad.
func NewCompetenceRegistry() *CompetenceRegistry {
	cr := &CompetenceRegistry{
		entries: make(map[string]map[Genre]*GenreCompetence),
	}

	// Default competence for orchestrator (handles all genres)
	orchestratorGenres := map[Genre]*GenreCompetence{
		GenreDirect: {
			Agent:        "orchestrator",
			Genre:        GenreDirect,
			ContextHints: []string{"project", "client", "team", "tasks"},
			DocTypes:     []string{"proposal", "plan", "sop"},
		},
		GenreInform: {
			Agent:        "orchestrator",
			Genre:        GenreInform,
			ContextHints: []string{"documents", "metrics", "knowledge"},
			DocTypes:     []string{"report", "brief"},
		},
		GenreCommit: {
			Agent:        "orchestrator",
			Genre:        GenreCommit,
			ContextHints: []string{"project", "tasks", "client"},
			DocTypes:     []string{"plan", "agreement"},
		},
		GenreDecide: {
			Agent:        "orchestrator",
			Genre:        GenreDecide,
			ContextHints: []string{"metrics", "project", "client"},
			DocTypes:     []string{"framework", "comparison"},
		},
		GenreExpress: {
			Agent:        "orchestrator",
			Genre:        GenreExpress,
			ContextHints: []string{"memory", "client"},
			DocTypes:     nil,
		},
	}
	cr.entries["orchestrator"] = orchestratorGenres

	// Document agent competence
	cr.entries["document"] = map[Genre]*GenreCompetence{
		GenreDirect: {
			Agent:        "document",
			Genre:        GenreDirect,
			ContextHints: []string{"project", "client", "documents", "knowledge"},
			DocTypes:     []string{"proposal", "sop", "report", "brief", "framework", "guide"},
		},
		GenreInform: {
			Agent:        "document",
			Genre:        GenreInform,
			ContextHints: []string{"documents", "knowledge"},
			DocTypes:     []string{"report", "brief"},
		},
	}

	// Project agent competence
	cr.entries["project"] = map[Genre]*GenreCompetence{
		GenreDirect: {
			Agent:        "project",
			Genre:        GenreDirect,
			ContextHints: []string{"project", "tasks", "team"},
			DocTypes:     []string{"plan", "roadmap"},
		},
		GenreCommit: {
			Agent:        "project",
			Genre:        GenreCommit,
			ContextHints: []string{"project", "tasks", "team", "client"},
			DocTypes:     []string{"plan", "agreement"},
		},
	}

	return cr
}

// Lookup returns the competence entry for the given agent+genre, or nil if not found.
// Falls back to "orchestrator" competence if the specific agent has no entry.
func (r *CompetenceRegistry) Lookup(agentType string, genre Genre) *GenreCompetence {
	if genreMap, ok := r.entries[agentType]; ok {
		if comp, ok := genreMap[genre]; ok {
			return comp
		}
	}
	// Fallback to orchestrator
	if genreMap, ok := r.entries["orchestrator"]; ok {
		if comp, ok := genreMap[genre]; ok {
			return comp
		}
	}
	return nil
}

// Register adds or replaces a competence entry.
func (r *CompetenceRegistry) Register(comp GenreCompetence) {
	if _, ok := r.entries[comp.Agent]; !ok {
		r.entries[comp.Agent] = make(map[Genre]*GenreCompetence)
	}
	r.entries[comp.Agent][comp.Genre] = &comp
}
