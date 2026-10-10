// Package metrics is the business-pulse four, read live (FounderOS v1
// lib/live-metrics.ts). A read produces a number or explains itself; a
// failed or hung read is a gap with its reason, never a zero.
package metrics

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type Read struct {
	Value  *float64
	Source string
}

type Metric struct {
	ID     string   `json:"id"`
	Key    string   `json:"key"`
	Label  string   `json:"label"`
	Unit   string   `json:"unit"`
	Value  *float64 `json:"value"`
	Source string   `json:"source"`
	Live   bool     `json:"live"`
}

type Readers struct {
	Unread, Stripe, BrainPages, AgentRuns func(context.Context) (Read, error)
	ReadTimeout                           time.Duration // default 8s
}

func guarded(ctx context.Context, label string, timeout time.Duration, read func(context.Context) (Read, error)) Read {
	if read == nil {
		return Read{Source: label + " not wired"}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	done := make(chan Read, 1)
	go func() {
		r, err := read(ctx)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return // the timeout branch reports it
			}
			done <- Read{Source: label + " failed: " + err.Error()}
			return
		}
		done <- r
	}()
	select {
	case r := <-done:
		return r
	case <-ctx.Done():
		return Read{Source: fmt.Sprintf("%s timed out after %ss", label, strconv.FormatFloat(timeout.Seconds(), 'f', -1, 64))}
	}
}

// Live reads all four concurrently; the slowest bounds the response.
func Live(ctx context.Context, r Readers) []Metric {
	if r.ReadTimeout <= 0 {
		r.ReadTimeout = 8 * time.Second
	}
	specs := []struct {
		id, key, label, unit, what string
		read                       func(context.Context) (Read, error)
	}{
		{"metric-unread", "unread_total", "Unread (all inboxes)", "emails", "inbox read", r.Unread},
		{"metric-brain", "brain_pages", "Brain Pages", "pages", "brain read", r.BrainPages},
		{"metric-balance", "stripe_available", "Stripe Available", "usd", "Stripe balance read", r.Stripe},
		{"metric-runs", "agent_runs", "Agent Runs Logged", "runs", "agent run count", r.AgentRuns},
	}
	out := make([]Metric, len(specs))
	var wg sync.WaitGroup
	for i := range specs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s := specs[i]
			rd := guarded(ctx, s.what, r.ReadTimeout, s.read)
			out[i] = Metric{ID: s.id, Key: s.key, Label: s.label, Unit: s.unit, Value: rd.Value, Source: rd.Source, Live: rd.Value != nil}
		}(i)
	}
	wg.Wait()
	return out
}

// Split pre-separates measurements from gaps for agent readers.
func Split(ms []Metric) (live, pending []map[string]any) {
	for _, m := range ms {
		if m.Live {
			live = append(live, map[string]any{"key": m.Key, "label": m.Label, "value": *m.Value, "unit": m.Unit, "source": m.Source})
		} else {
			pending = append(pending, map[string]any{"key": m.Key, "label": m.Label, "reason": m.Source})
		}
	}
	return
}

type Inbox struct {
	Unread *int
	Err    string
}

// UnreadFrom totals unread mail; a partial failure is reported as a floor.
func UnreadFrom(inboxes []Inbox) (Read, error) {
	if len(inboxes) == 0 {
		return Read{}, errors.New("no inbox configured (set INBOX_n_* in env.local)")
	}
	total, failing := 0, 0
	firstErr := ""
	for _, ib := range inboxes {
		if ib.Err != "" || ib.Unread == nil {
			failing++
			if firstErr == "" {
				firstErr = ib.Err
			}
			continue
		}
		total += *ib.Unread
	}
	if failing == len(inboxes) {
		return Read{}, fmt.Errorf("all %d inbox connections failed: %s", len(inboxes), firstErr)
	}
	plural := ""
	if len(inboxes) > 1 {
		plural = "es"
	}
	src := fmt.Sprintf("%d inbox%s · IMAP", len(inboxes), plural)
	if failing > 0 {
		src += fmt.Sprintf(" · %d failing, count is a floor", failing)
	}
	v := float64(total)
	return Read{Value: &v, Source: src}, nil
}
