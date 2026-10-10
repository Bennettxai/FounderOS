package metrics

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLiveSeparatesMeasurementsFromGaps(t *testing.T) {
	r := Readers{
		Unread: func(context.Context) (Read, error) { return Read{Value: f(3031), Source: "4 inboxes · IMAP"}, nil },
		Stripe: func(context.Context) (Read, error) { return Read{}, errors.New("stripe down") },
		BrainPages: func(context.Context) (Read, error) {
			return Read{Value: f(1423), Source: "Optimal Engine · 2 engines"}, nil
		},
		AgentRuns:   func(ctx context.Context) (Read, error) { <-ctx.Done(); return Read{}, ctx.Err() },
		ReadTimeout: 50 * time.Millisecond,
	}
	ms := Live(context.Background(), r)
	by := map[string]Metric{}
	for _, m := range ms {
		by[m.Key] = m
	}
	if len(ms) != 4 {
		t.Fatalf("want the four pulse metrics, got %d", len(ms))
	}
	if m := by["unread_total"]; !m.Live || *m.Value != 3031 || m.ID != "metric-unread" {
		t.Errorf("unread = %+v", m)
	}
	if m := by["stripe_available"]; m.Live || m.Value != nil || m.Source != "Stripe balance read failed: stripe down" {
		t.Errorf("a failed read must be a gap with its reason, never 0: %+v", m)
	}
	if m := by["agent_runs"]; m.Live || m.Source != "agent run count timed out after 0.05s" {
		t.Errorf("a hung read must time out into a gap: %+v", m)
	}
	if m := by["brain_pages"]; !m.Live || m.Label != "Brain Pages" {
		t.Errorf("brain = %+v", m)
	}
	live, pending := Split(ms)
	if len(live) != 2 || len(pending) != 2 || pending[0]["reason"] == nil {
		t.Fatalf("split live=%v pending=%v", live, pending)
	}
}

func TestUnreadCountRules(t *testing.T) {
	n := 5
	if _, err := UnreadFrom(nil); err == nil {
		t.Fatal("no inbox configured must be a gap")
	}
	if _, err := UnreadFrom([]Inbox{{Err: "auth"}, {Err: "auth"}}); err == nil {
		t.Fatal("all inboxes failing must be a gap")
	}
	r, err := UnreadFrom([]Inbox{{Unread: &n}, {Err: "auth"}})
	if err != nil || *r.Value != 5 || r.Source != "2 inboxes · IMAP · 1 failing, count is a floor" {
		t.Fatalf("partial = %+v %v", r, err)
	}
}

func f(v float64) *float64 { return &v }
