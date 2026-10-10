package shared

import (
	"testing"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

func TestDailySeriesBucketsByLocalDayAndDropsTheFuture(t *testing.T) {
	now := time.Date(2026, 9, 30, 15, 0, 0, 0, time.Local)
	ts := []string{
		now.Add(-time.Hour).Format(time.RFC3339),
		now.Add(-2 * time.Hour).Format(time.RFC3339),
		now.AddDate(0, 0, -13).Format(time.RFC3339),
		now.AddDate(0, 0, -14).Format(time.RFC3339), // outside
		now.Add(time.Hour).Format(time.RFC3339),     // future
		"", "garbage",
	}
	s := DailySeries(ts, 14, now)
	if len(s) != 14 || s[13].Label != "Sep 30" || s[13].Count != 2 || s[0].Label != "Sep 17" || s[0].Count != 1 {
		t.Fatalf("series: %+v", s)
	}
	total := 0
	for _, p := range s {
		total += p.Count
	}
	if total != 3 {
		t.Fatalf("total %d", total)
	}
}

func TestShortLabelsDedupe(t *testing.T) {
	got := ShortLabels([]string{"Sales Agent", "Sales Ops", "Conductor", "  "}, 0)
	want := []string{"Sales", "Sales 2", "Conductor", "Agent"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
	if cut := ShortLabels([]string{"Marketing"}, 4); cut[0] != "Mark" {
		t.Fatalf("max: %v", cut)
	}
}

func TestPlural(t *testing.T) {
	if Plural(1, "task") != "1 task" || Plural(0, "task") != "0 tasks" || Plural(3, "run") != "3 runs" {
		t.Fatal("plural")
	}
}

func TestScheduledJobRowsBandsAndFlags(t *testing.T) {
	now := time.Date(2026, 9, 30, 10, 0, 0, 0, time.Local)
	last := now.Add(-25 * time.Hour).UTC().Format(time.RFC3339)
	fresh := now.Add(-30 * time.Minute).UTC().Format(time.RFC3339)
	f, tr := false, true
	crons := []osdata.Cron{
		{ID: "a", AgentID: "crm-pulse", Schedule: "0 9 * * *", Description: "Zeta overdue", Enabled: true},
		{ID: "b", AgentID: "ghost", Schedule: "*/30 * * * *", Description: "Alpha off", Enabled: false},
		{ID: "c", AgentID: "crm-pulse", Schedule: "0 9 * * *", Description: "Beta healthy", Enabled: true},
	}
	stats := map[string]osdata.CronStat{
		"a": {Runs: 3, OK: 2, LastRunAt: &last, LastOK: &f},
		"c": {Runs: 1, OK: 1, LastRunAt: &fresh, LastOK: &tr},
	}
	recent := map[string][]osdata.CronRun{"a": {{OK: false, Summary: "boom"}, {OK: true, Summary: "fine"}}}
	rows := ScheduledJobRows(crons, stats, map[string]string{"crm-pulse": "CRM Pulse"}, recent, now)
	if rows[0].ID != "a" || !rows[0].Overdue || rows[1].ID != "c" || rows[1].Overdue || rows[2].ID != "b" {
		t.Fatalf("bands: %+v", rows)
	}
	if rows[2].AgentName != "ghost" || !rows[2].UnknownAgent || rows[2].NextRunAt != nil {
		t.Fatalf("unknown agent / disabled: %+v", rows[2])
	}
	if rows[0].ScheduleLabel != agents.DescribeCron("0 9 * * *") || rows[0].NextRunAt == nil {
		t.Fatalf("label/next: %+v", rows[0])
	}
	if len(rows[0].History) != 2 || !rows[0].History[0] || rows[0].History[1] || rows[0].LastSummary == nil || *rows[0].LastSummary != "boom" {
		t.Fatalf("history oldest first: %+v", rows[0])
	}
	if rows[1].History == nil || rows[1].LastSummary != nil {
		t.Fatalf("no runs: %+v", rows[1])
	}
}
