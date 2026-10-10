package agents

import (
	"compress/gzip"
	"io"
	"os"
	"regexp"
	"testing"
	"time"
)

// FounderOS v1's seven agent crons (lib/seed.ts), with a minute each one must
// fire in (local time) and a neighbouring minute it must not.
var founderosCrons = []struct {
	id, agent, schedule string
	fires, quiet        string
}{
	{"cron-comms-digest-0900", "comms-digest", "0 9 * * *", "2026-10-03 09:00", "2026-10-03 09:01"},
	{"cron-plaud-ingest-30m", "sales-calls-data", "*/30 * * * *", "2026-10-03 14:30", "2026-10-03 14:15"},
	{"cron-stack-monitor-0700", "stack-monitor", "0 7 * * *", "2026-10-03 07:00", "2026-10-03 08:00"},
	{"cron-payments-pulse-0800", "payments-pulse", "0 8 * * *", "2026-10-03 08:00", "2026-10-03 07:59"},
	{"cron-client-onboarding-0830", "client-onboarding", "30 8 * * *", "2026-10-03 08:30", "2026-10-03 08:00"},
	{"cron-crm-pulse-0900", "crm-pulse", "0 9 * * 1-5", "2026-10-02 09:00", "2026-10-03 09:00"}, // Fri yes, Sat no
	{"cron-social-agent-1800", "social-agent", "0 18 * * *", "2026-10-03 18:00", "2026-10-03 06:00"},
}

func TestFounderosCronSchedulesFireWhenFounderosOSDoes(t *testing.T) {
	at := func(s string) time.Time { tm, _ := time.ParseInLocation("2006-01-02 15:04", s, CronZone); return tm }
	for _, c := range founderosCrons {
		if !ValidCron(c.schedule) {
			t.Errorf("%s: invalid schedule", c.id)
		}
		if !MatchesCron(c.schedule, at(c.fires)) {
			t.Errorf("%s must fire at %s", c.id, c.fires)
		}
		if MatchesCron(c.schedule, at(c.quiet)) {
			t.Errorf("%s must not fire at %s", c.id, c.quiet)
		}
	}
}

// The ETL fixture is FounderOS v1's own seed; the bridge's crons come from it,
// so the pinned list above must be exactly what that seed carries.
func TestSeedCarriesExactlyTheseCrons(t *testing.T) {
	f, err := os.Open("../etl/testdata/founderos-os.seed.sql.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(gz)
	re := regexp.MustCompile(`INSERT INTO agent_crons VALUES\('([^']+)','([^']+)','([^']+)'`)
	seen := map[string]string{}
	for _, m := range re.FindAllStringSubmatch(string(raw), -1) {
		seen[m[1]] = m[2] + "|" + m[3]
	}
	if len(seen) != len(founderosCrons) {
		t.Fatalf("seed has %d crons, pinned %d: %v", len(seen), len(founderosCrons), seen)
	}
	for _, c := range founderosCrons {
		if seen[c.id] != c.agent+"|"+c.schedule {
			t.Errorf("%s: seed %q, pinned %q", c.id, seen[c.id], c.agent+"|"+c.schedule)
		}
	}
}

// Crons are the operator's wall clock (America/Chicago), not the process's: under
// a UTC container the 09:00 digest must still fire at 09:00 Chicago.
func TestCronsFireOnChicagoTimeWhateverTheProcessZone(t *testing.T) {
	utc := time.Date(2026, 9, 30, 14, 0, 0, 0, time.UTC) // 09:00 CDT
	if !MatchesCron("0 9 * * *", utc) {
		t.Fatal("09:00 Chicago read from a UTC clock must match 0 9 * * *")
	}
	if MatchesCron("0 14 * * *", utc) {
		t.Fatal("14:00 UTC is not 14:00 on the operator's clock")
	}
	if CronZone.String() != "America/Chicago" {
		t.Fatalf("zone = %s", CronZone)
	}
}
