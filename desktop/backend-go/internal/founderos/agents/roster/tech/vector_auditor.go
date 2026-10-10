package tech

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
)

// criticalChecks are the engine audit checks retrieval cannot survive: a
// failure in any of them fails the run. The rest (backups, fixtures, the
// optional RLM runtime) are warnings for this auditor.
var criticalChecks = map[string]bool{
	"sqlite_integrity": true,
	"migrations":       true,
	"logical_stores":   true,
	"fts_parity":       true,
	"vector_integrity": true,
}

// VectorAuditor is the engines' doctor: storage audit, embedding coverage and
// the claims-to-facts pipeline, per staged engine.
type VectorAuditor struct {
	Engines EngineSource
	Client  *http.Client
}

func (a *VectorAuditor) Meta() agents.Meta {
	return agents.Meta{
		ID:           "vector-auditor",
		Name:         "Vector Auditor",
		Description:  "Optimal Engine doctor: storage audit, embedding coverage, claims vs facts, health score.",
		DepartmentID: "dept-tech",
	}
}

func (a *VectorAuditor) Run(ctx context.Context) (agents.Result, error) {
	v, err := resolveView(a.Engines)
	if err != nil {
		return agents.Result{OK: false, Summary: err.Error()}, nil
	}
	if len(v.engines) == 0 {
		return agents.Result{OK: false, Summary: noEnginesSummary}, nil
	}
	ovs := overview(ctx, v, newClient(a.Client), true)

	ok := true
	segments := make([]string, 0, len(ovs)+1)
	for _, o := range ovs {
		seg, good := vectorSegment(o)
		ok = ok && good
		segments = append(segments, seg)
	}
	if note := v.unstagedNote(); note != "" {
		segments = append(segments, note)
	}
	return agents.Result{OK: ok, Summary: strings.Join(segments, " | "), Data: map[string]any{"engines": ovs, "unstaged": v.unstaged}}, nil
}

func vectorSegment(o EngineOverview) (string, bool) {
	if !o.Reachable {
		return fmt.Sprintf("%s: offline — %s", o.Engine, o.Error), false
	}
	ok := o.Health == "up"
	var parts, warnings []string

	head := o.Engine + ": health "
	if !ok {
		head += fmt.Sprintf("%q, audit ", o.Health)
	}
	switch {
	case o.AuditError != "":
		ok = false
		parts = append(parts, head+"unknown · storage audit unreadable ("+o.AuditError+")")
	case o.Audit != nil:
		failing := o.Audit.failing()
		score := int(math.Round(float64(len(o.Audit.Checks)-len(failing)) * 100 / float64(len(o.Audit.Checks))))
		line := fmt.Sprintf("%s%d/100 · %d checks, %d failing", head, score, len(o.Audit.Checks), len(failing))
		if len(failing) > 0 {
			names := make([]string, len(failing))
			for i, c := range failing {
				names[i] = c.Name
				if criticalChecks[c.Name] {
					ok = false
					names[i] += " (" + clip(c.detail(), 80) + ")"
				}
			}
			line += ": " + strings.Join(names, ", ")
		}
		parts = append(parts, line)
	}

	if o.Counts == nil {
		ok = false
		parts = append(parts, "table counts unreadable ("+o.Error+")")
	} else {
		ctxs, vecs := o.count("contexts"), o.count("vectors")
		parts = append(parts, fmt.Sprintf("embeddings %s/%s contexts · %s chunk vectors", itoa(vecs), itoa(ctxs), itoa(o.count("chunk_embeddings"))))
		if ctxs != nil && vecs != nil && *ctxs > *vecs {
			warnings = append(warnings, fmt.Sprintf("%d context(s) without an embedding", *ctxs-*vecs))
		}
		claims, facts := o.count("claims"), o.count("facts")
		line := fmt.Sprintf("claims %s / facts %s", itoa(claims), itoa(facts))
		if len(o.Pending) > 0 {
			pend := make([]string, 0, len(o.Workspaces))
			for _, w := range o.Workspaces {
				if n, seen := o.Pending[w]; seen {
					pend = append(pend, w+" "+itoa(n))
				}
			}
			line += " (pending: " + strings.Join(pend, ", ") + ")"
		}
		parts = append(parts, line)
		if claims != nil && facts != nil && *claims > 0 && *facts == 0 {
			warnings = append(warnings, "no claim promoted to a fact yet")
		}
	}
	if o.PendingErr != "" {
		warnings = append(warnings, "pending claims unreadable ("+o.PendingErr+")")
	}
	if len(warnings) > 0 {
		parts = append(parts, "warnings: "+strings.Join(warnings, "; "))
	}
	return strings.Join(parts, " · "), ok
}
