// Package ventures is FounderOS v1 lib/ventures.ts: the two businesses, their
// focus lists (the operator's to edit), and which agents work each life area.
package ventures

type Venture struct {
	ID         string              `json:"id"`
	Label      string              `json:"label"`
	Kind       string              `json:"kind"`
	Color      string              `json:"color"`
	Detail     string              `json:"detail"`
	BrainTag   string              `json:"brainTag"` // the venture's Optimal Engine workspace slug
	Focus      []string            `json:"focus"`
	AreaAgents map[string][]string `json:"areaAgents"`
}

var (
	sharedOps       = []string{"conductor", "stack-monitor"}
	sharedKnowledge = []string{"data-agent", "markdown-auditor", "vector-auditor"}
)

var All = []Venture{
	{
		ID: "vantage", Label: "Vantage", Kind: "AI agency", Color: "#00ffaa",
		Detail:   "Client AI builds and delivery — the agency arm.",
		BrainTag: "vantage",
		Focus: []string{
			"Active client builds shipped on schedule",
			"Pipeline: leads in from Typeform, calls booked and held, proposals out",
			"Delivery quality — every handoff documented in the brain (Optimal Engine)",
		},
		AreaAgents: map[string][]string{
			"marketing":     {"social-agent", "postly-publisher", "reelkit-editor", "renderly-creative"},
			"sales":         {"vantage-sales", "vantage-paykit", "sales-agent", "sales-calls-data"},
			"communication": {"comms-agent", "gmail-worker", "slack-worker", "crm-pulse"},
			"finances":      {"payments-pulse", "stripe-sales", "processor-confirmation"},
			"knowledge":     sharedKnowledge,
			"operations":    sharedOps,
		},
	},
	{
		ID: "launchpad-cohort", Label: "Launchpad Cohort", Kind: "Mentorship program", Color: "#d9263f",
		Detail:   "The mentorship — students, curriculum, community.",
		BrainTag: "launchpad-cohort",
		Focus: []string{
			"Student results — track wins, unblock stuck students fast",
			"Content + newsletter cadence for enrollment",
			"Community pulse on WhatsApp; T1 response times hold",
		},
		AreaAgents: map[string][]string{
			"marketing":     {"social-agent", "adsmith-creative", "postly-publisher", "dmflow-mcp", "reelkit-editor"},
			"sales":         {"launchpad-cohort-sales", "paykit-sales", "sales-agent", "sales-calls-data"},
			"communication": {"whatsapp-worker", "gmail-worker", "comms-agent", "crm-pulse"},
			"finances":      {"payments-pulse", "stripe-sales", "flexpay-financing", "processor-confirmation"},
			"knowledge":     sharedKnowledge,
			"operations":    sharedOps,
		},
	},
}

func Get(id string) *Venture {
	for i := range All {
		if All[i].ID == id {
			return &All[i]
		}
	}
	return nil
}

// AgentSet is every agent serving a venture, across its life areas.
func AgentSet(id string) map[string]bool {
	out := map[string]bool{}
	if v := Get(id); v != nil {
		for _, agents := range v.AreaAgents {
			for _, a := range agents {
				out[a] = true
			}
		}
	}
	return out
}

// ForAgent lists the ventures an agent works for.
func ForAgent(agentID string) []Venture {
	var out []Venture
	for _, v := range All {
		if AgentSet(v.ID)[agentID] {
			out = append(out, v)
		}
	}
	return out
}
