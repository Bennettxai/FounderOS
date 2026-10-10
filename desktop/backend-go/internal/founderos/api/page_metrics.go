package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/payments"
	"github.com/rhl/businessos-backend/internal/founderos/pages/metrics"
)

// FounderOS v1 GET /api/metrics: the business-pulse four, read live.
func init() { RegisterPage(registerMetrics) }

func registerMetrics(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/metrics", func(c *gin.Context) {
		ms := metrics.Live(c.Request.Context(), metricReaders(d))
		live, pending := metrics.Split(ms)
		c.JSON(http.StatusOK, gin.H{"metrics": ms, "live": nonNil(live), "pending": nonNil(pending)})
	})
}

func nonNil(v []map[string]any) []map[string]any {
	if v == nil {
		return []map[string]any{}
	}
	return v
}

func metricReaders(d *Deps) metrics.Readers {
	return metrics.Readers{
		Unread: func(ctx context.Context) (metrics.Read, error) {
			counts, err := depsEmail(d).UnreadCounts(ctx)
			if err != nil {
				return metrics.Read{}, err
			}
			inboxes := make([]metrics.Inbox, len(counts))
			for i, c := range counts {
				inboxes[i] = metrics.Inbox{Unread: c.Unread, Err: c.Error}
			}
			return metrics.UnreadFrom(inboxes)
		},
		Stripe: func(ctx context.Context) (metrics.Read, error) {
			snap, err := payments.New(d.Resolver).StripeSnapshot(ctx)
			if err != nil {
				return metrics.Read{}, err
			}
			if len(snap.Available) == 0 {
				return metrics.Read{}, errors.New("Stripe returned no available balance bucket")
			}
			v := float64(snap.Available[0].Amount) / 100
			return metrics.Read{Value: &v, Source: "Stripe · " + strings.ToUpper(snap.Available[0].Currency)}, nil
		},
		BrainPages: func(ctx context.Context) (metrics.Read, error) { return readBrainPages(ctx, d) },
		AgentRuns: func(ctx context.Context) (metrics.Read, error) {
			if d.Pool == nil {
				return metrics.Read{}, errors.New("database unavailable")
			}
			var n int
			if err := d.Pool.QueryRow(ctx, `SELECT count(*) FROM founderos_agent_runs WHERE id NOT LIKE 'seed-run-%'`).Scan(&n); err != nil {
				return metrics.Read{}, err
			}
			v := float64(n)
			return metrics.Read{Value: &v, Source: "agent runtime · all time, excludes seeded demo runs"}, nil
		},
	}
}

// readBrainPages is swappable so page tests never reach the real engines.
var readBrainPages = brainPages

// brainPages sums source packages across the routed engines. GBrain's
// "brain-store pages" is retired; an engine that does not answer fails the
// read rather than undercounting.
func brainPages(ctx context.Context, d *Deps) (metrics.Read, error) {
	engines := TopologyEngines()
	if len(engines) == 0 {
		return metrics.Read{}, errors.New("no engines in the topology have an endpoint")
	}
	client := &http.Client{Timeout: 6 * time.Second}
	total := 0
	for _, e := range engines {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(e.URL, "/")+"/api/stores", nil)
		if err != nil {
			return metrics.Read{}, fmt.Errorf("engine %s: %w", e.Name, err)
		}
		if e.Key != "" {
			req.Header.Set("Authorization", "Bearer "+e.Key)
		}
		resp, err := client.Do(req)
		if err != nil {
			return metrics.Read{}, fmt.Errorf("engine %s: %w", e.Name, err)
		}
		var body struct {
			Stores []struct {
				ID          string         `json:"id"`
				TableCounts map[string]int `json:"table_counts"`
			} `json:"stores"`
		}
		err = json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != http.StatusOK {
			return metrics.Read{}, fmt.Errorf("engine %s: HTTP %d", e.Name, resp.StatusCode)
		}
		for _, st := range body.Stores {
			if st.ID == "relational" {
				total += st.TableCounts["source_packages"]
			}
		}
	}
	v := float64(total)
	return metrics.Read{Value: &v, Source: fmt.Sprintf("Optimal Engine · %d engines · source packages", len(engines))}, nil
}
