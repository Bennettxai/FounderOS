package api

// /usage (spec 6.11): FounderOS v1 app/api/usage/route.ts. On the bridge every
// seat is a machine's push: the collectors' device pushes (live, in memory)
// plus rows /api/usage/push stored in founderos_usage_snapshots and
// founderos_ollama_snapshots (founderos). The backend parses no device files,
// so a plan no machine reported reads null (unknown), never zero.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/pages/usage"
)

func init() { RegisterPage(usageRegister) }

// usageBoardNames maps board seat ids to agent names, best-effort: an
// unreachable board keeps the ids (FounderOS v1 boardNames). A var so tests
// never reach the network.
var usageBoardNames = func(ctx context.Context, d *Deps) map[string]string {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	agents, err := paperclip.New(d.Resolver).Agents(ctx)
	if err != nil {
		return nil
	}
	out := map[string]string{}
	for _, a := range agents {
		out[a.ID] = a.Name
	}
	return out
}

func usageRegister(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/usage", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, usageBoard(ctx, d, time.Now()))
	})
}

func usageBoard(ctx context.Context, d *Deps, now time.Time) usage.Board {
	errs := map[string]string{}
	var seats []usage.Seat
	var lanes []usage.Ollama

	if d.Devices == nil {
		errs["push"] = "no device receiver"
	} else {
		if rs, err := d.Devices.UsageSeats(ctx); err != nil {
			errs["push"] = err.Error()
		} else {
			for _, r := range rs {
				seats = append(seats, usage.Seat{SeatUsage: r.Data, ReceiverStale: r.Stale})
			}
		}
		if rs, err := d.Devices.OllamaSnapshots(ctx); err != nil {
			errs["ollamaPush"] = err.Error()
		} else {
			for _, r := range rs {
				lanes = append(lanes, usage.Ollama{OllamaSnapshot: r.Data, ReceiverStale: r.Stale})
			}
		}
	}

	storedSeats, storedLanes, err := usageStored(ctx, d)
	if err != nil {
		errs["stored"] = err.Error()
	}
	seats = append(seats, storedSeats...)
	lanes = append(lanes, storedLanes...)

	var names map[string]string
	if len(seats) > 0 {
		names = usageBoardNames(ctx, d)
	}
	b := usage.Build(seats, lanes, names, now)
	if len(errs) > 0 {
		b.Errors = errs
	}
	return b
}

// usageStored reads the seats and Ollama lanes /api/usage/push stored for the
// FounderOS workspace. Every stored row is a push, whatever its payload says.
func usageStored(ctx context.Context, d *Deps) ([]usage.Seat, []usage.Ollama, error) {
	if d.Pool == nil {
		return nil, nil, errors.New("no database: stored usage pushes unreadable")
	}
	const q = `SELECT s.payload FROM %s s JOIN workspaces w ON w.id = s.workspace_id WHERE w.slug = 'founderos' ORDER BY s.captured_at`
	var seats []usage.Seat
	rows, err := d.Pool.Query(ctx, fmt.Sprintf(q, "founderos_usage_snapshots"))
	if err != nil {
		return nil, nil, err
	}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			rows.Close()
			return nil, nil, err
		}
		var s devicepush.SeatUsage
		if json.Unmarshal(raw, &s) != nil || s.ID == "" || (s.Kind != "claude" && s.Kind != "codex") {
			continue // a malformed row is skipped, as the Zod parse would drop it
		}
		s.Source = "push"
		seats = append(seats, usage.Seat{SeatUsage: s})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var lanes []usage.Ollama
	rows, err = d.Pool.Query(ctx, fmt.Sprintf(q, "founderos_ollama_snapshots"))
	if err != nil {
		return seats, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return seats, nil, err
		}
		var o devicepush.OllamaSnapshot
		if json.Unmarshal(raw, &o) != nil || o.ID == "" || (o.Lane.State != "up" && o.Lane.State != "down") {
			continue
		}
		lanes = append(lanes, usage.Ollama{OllamaSnapshot: o})
	}
	return seats, lanes, rows.Err()
}
