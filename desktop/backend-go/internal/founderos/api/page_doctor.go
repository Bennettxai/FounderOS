package api

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/pages/doctor"
)

// /doctor (spec 6.19). FounderOS v1's Doctor reported GBrain; GBrain is retired,
// so this reports Optimal Engine health: every topology engine (GET only),
// plus the relational reads the page always had (the pillar radar and the
// memory agents' runs) from the founderos founderos_* tables.

func init() {
	RegisterPage(func(s *gin.RouterGroup, d *Deps) {
		s.GET("/pages/doctor", doctorPage(d))
	})
}

// doctorEngines is the engine list; tests swap it for fakes.
var doctorEngines = func() []doctor.Engine {
	var out []doctor.Engine
	for _, e := range TopologyEngines() {
		out = append(out, doctor.Engine{Name: e.Name, URL: e.URL, Key: e.Key})
	}
	// TopologyEngines walks a map; keep the page's rows in a stable order.
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// doctorMemoryAgents are the agents whose runs the Brain Runs card counts.
// Like v1's Doctor, that is the data agent alone (the brain's analyst).
var doctorMemoryAgents = []string{"data-agent"}

const doctorWindowDays = 14

type doctorLastRun struct {
	AgentID    string    `json:"agentId"`
	FinishedAt time.Time `json:"finishedAt"`
	OK         bool      `json:"ok"`
}

type doctorRelational struct {
	axes    []doctor.PillarAxis
	runs    []doctor.RunLite
	last    *doctorLastRun
	failure error
}

func doctorPage(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
		defer cancel()
		now := time.Now()

		readings := doctor.Read(ctx, connectors.HTTPClient(8*time.Second), doctorEngines())
		if readings == nil {
			readings = []doctor.EngineReading{}
		}
		rel := doctorReadRelational(ctx, d, now)

		checks := []doctor.Check{}
		for _, e := range readings {
			checks = append(checks, e.Checks...)
		}
		vol := doctor.BuildVolume(doctor.VolumeInput{
			Engines: readings, Axes: rel.axes, Runs: rel.runs, Now: now, Days: doctorWindowDays,
		})
		relational := gin.H{"ok": rel.failure == nil}
		if rel.failure != nil {
			relational["error"] = rel.failure.Error()
		}
		c.JSON(http.StatusOK, gin.H{
			"generatedAt": now.UTC(),
			"windowDays":  doctorWindowDays,
			"engines":     readings,
			"checks":      checks,
			"score":       doctor.Score(readings),
			"volume":      vol,
			"layers":      doctor.Layers(readings),
			"axes":        rel.axes,
			"relational":  relational,
			"brain":       gin.H{"agents": doctorMemoryAgents, "last": rel.last},
		})
	}
}

func doctorReadRelational(ctx context.Context, d *Deps, now time.Time) doctorRelational {
	if d.Pool == nil {
		return doctorRelational{failure: errors.New("Postgres is not connected")}
	}
	var ws string
	if err := d.Pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = 'founderos'`).Scan(&ws); err != nil {
		return doctorRelational{failure: errors.New("no founderos workspace (bootstrap not run): " + err.Error())}
	}
	fail := func(err error) doctorRelational { return doctorRelational{failure: err} }

	var depts []doctor.Department
	rows, err := d.Pool.Query(ctx, `SELECT id, name, color FROM founderos_departments WHERE workspace_id = $1 ORDER BY ord`, ws)
	if err != nil {
		return fail(err)
	}
	for rows.Next() {
		var x doctor.Department
		if err := rows.Scan(&x.ID, &x.Name, &x.Color); err != nil {
			rows.Close()
			return fail(err)
		}
		depts = append(depts, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fail(err)
	}

	var roster []doctor.Agent
	rows, err = d.Pool.Query(ctx, `SELECT id, department_id, status FROM founderos_agents WHERE workspace_id = $1`, ws)
	if err != nil {
		return fail(err)
	}
	for rows.Next() {
		var x doctor.Agent
		if err := rows.Scan(&x.ID, &x.DepartmentID, &x.Status); err != nil {
			rows.Close()
			return fail(err)
		}
		roster = append(roster, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fail(err)
	}

	var tasks []doctor.SopTask
	rows, err = d.Pool.Query(ctx, `SELECT department_id FROM founderos_sop_tasks WHERE workspace_id = $1`, ws)
	if err != nil {
		return fail(err)
	}
	for rows.Next() {
		var x doctor.SopTask
		if err := rows.Scan(&x.DepartmentID); err != nil {
			rows.Close()
			return fail(err)
		}
		tasks = append(tasks, x)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fail(err)
	}

	latest := map[string]time.Time{}
	rows, err = d.Pool.Query(ctx, `SELECT agent_id, max(finished_at) FROM founderos_agent_runs WHERE workspace_id = $1 GROUP BY agent_id`, ws)
	if err != nil {
		return fail(err)
	}
	for rows.Next() {
		var id string
		var t time.Time
		if err := rows.Scan(&id, &t); err != nil {
			rows.Close()
			return fail(err)
		}
		latest[id] = t
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fail(err)
	}

	out := doctorRelational{axes: doctor.PillarAxes(depts, roster, tasks, latest, now)}
	// One spare day so a local-day window never loses its first morning.
	since := now.AddDate(0, 0, -(doctorWindowDays + 1))
	rows, err = d.Pool.Query(ctx, `SELECT agent_id, finished_at, ok FROM founderos_agent_runs
		WHERE workspace_id = $1 AND agent_id = ANY($2) AND finished_at >= $3 ORDER BY finished_at DESC`, ws, doctorMemoryAgents, since)
	if err != nil {
		return fail(err)
	}
	for rows.Next() {
		var r doctorLastRun
		if err := rows.Scan(&r.AgentID, &r.FinishedAt, &r.OK); err != nil {
			rows.Close()
			return fail(err)
		}
		if out.last == nil {
			last := r
			out.last = &last
		}
		out.runs = append(out.runs, doctor.RunLite{FinishedAt: r.FinishedAt, OK: r.OK})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fail(err)
	}
	if out.last == nil {
		// Older than the window still answers "when did memory last run".
		var r doctorLastRun
		err := d.Pool.QueryRow(ctx, `SELECT agent_id, finished_at, ok FROM founderos_agent_runs
			WHERE workspace_id = $1 AND agent_id = ANY($2) ORDER BY finished_at DESC LIMIT 1`, ws, doctorMemoryAgents).Scan(&r.AgentID, &r.FinishedAt, &r.OK)
		if err == nil {
			out.last = &r
		}
	}
	return out
}
