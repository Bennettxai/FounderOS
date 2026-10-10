package api

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush/collect"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pgtest"
)

func usageGet(t *testing.T, d *Deps) (int, map[string]json.RawMessage) {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/founderos/pages/usage", nil)
	req.Header.Set("Cookie", "session=ok")
	router(t, d).ServeHTTP(w, req)
	var body map[string]json.RawMessage
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return w.Code, body
}

func usageNoBoardNames(t *testing.T) {
	prev := usageBoardNames
	usageBoardNames = func(context.Context, *Deps) map[string]string { return nil }
	t.Cleanup(func() { usageBoardNames = prev })
}

func TestUsageRequiresSession(t *testing.T) {
	usageNoBoardNames(t)
	w := httptest.NewRecorder()
	router(t, &Deps{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/founderos/pages/usage", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestUsageWithNoPushesReadsUnknownNotZero(t *testing.T) {
	usageNoBoardNames(t)
	d := &Deps{Devices: devicepush.NewReceiver(devicepush.NewMemStore(), "alexs-macbook-pro")}
	code, body := usageGet(t, d)
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	for _, k := range []string{"claude", "codex", "ollama"} {
		if string(body[k]) != "null" {
			t.Fatalf("%s = %s, want null (no machine reported it)", k, body[k])
		}
	}
	// no database: the stored pushes are unreachable and the board says so
	var errs map[string]string
	_ = json.Unmarshal(body["errors"], &errs)
	if errs["stored"] == "" {
		t.Fatalf("errors = %s, want a stored-pushes error with no database", body["errors"])
	}
}

func TestUsageCombinesDevicePushedSeatsIntoOnePlanPerProvider(t *testing.T) {
	usageNoBoardNames(t)
	recv := devicepush.NewReceiver(devicepush.NewMemStore(), "alexs-macbook-pro", "mini")
	now := time.Now().UTC()
	day := now.In(time.Local).Format("2006-01-02") // buckets are local days, as in FounderOS v1 lib/usage.ts dayKey
	plan := "Claude Max 20x"
	push := func(dev string, in float64, withOllama bool) {
		p := devicepush.Payload{Device: dev, Label: dev, CapturedAt: now.Format(time.RFC3339), Usage: []devicepush.SeatUsage{{
			ID: "claude-" + dev, Kind: "claude", Label: "Claude · " + dev, CapturedAt: now.Format(time.RFC3339),
			Days: []devicepush.DayBucket{{Day: day, Tot: devicepush.Tot{In: in}}}, ByModel: map[string]devicepush.Tot{}, Plan: &plan,
		}}}
		if withOllama {
			p.Ollama = &devicepush.OllamaSnapshot{Kind: "ollama", ID: "ollama-" + dev, Label: "Ollama · " + dev, CapturedAt: now.Format(time.RFC3339),
				Lane: devicepush.OllamaLane{State: "up", Models: []devicepush.OllamaModel{}}}
		}
		if err := recv.Accept(context.Background(), p); err != nil {
			t.Fatal(err)
		}
	}
	push("alexs-macbook-pro", 100, true)
	push("mini", 50, false)

	code, body := usageGet(t, &Deps{Devices: recv})
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	var claude struct {
		Plan     *string `json:"plan"`
		Days     []struct{ In float64 }
		Machines []struct {
			Source string `json:"source"`
			Stale  bool   `json:"stale"`
		} `json:"machines"`
	}
	if err := json.Unmarshal(body["claude"], &claude); err != nil {
		t.Fatalf("claude = %s: %v", body["claude"], err)
	}
	if claude.Plan == nil || *claude.Plan != plan || len(claude.Days) != 7 || claude.Days[6].In != 150 { // the 7-day window ends today
		t.Fatalf("claude = %s", body["claude"])
	}
	if len(claude.Machines) != 2 || claude.Machines[0].Source != "push" || claude.Machines[0].Stale {
		t.Fatalf("machines = %s", body["claude"])
	}
	if string(body["codex"]) != "null" {
		t.Fatalf("codex = %s, want null", body["codex"])
	}
	var ollama struct {
		State    string `json:"state"`
		Machines []any  `json:"machines"`
	}
	_ = json.Unmarshal(body["ollama"], &ollama)
	if ollama.State != "up" || len(ollama.Machines) != 1 {
		t.Fatalf("ollama = %s", body["ollama"])
	}
}

func usageThrowawayDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	admin := pgtest.AdminURL()
	ctx := context.Background()
	ap, err := pgxpool.New(ctx, admin)
	if err == nil {
		err = ap.Ping(ctx)
	}
	if err != nil {
		t.Skipf("bridge Postgres unreachable (%v)", err)
	}
	name := fmt.Sprintf("founderos_usage_test_%d", rand.New(rand.NewSource(time.Now().UnixNano())).Intn(1_000_000))
	if _, err := ap.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = ap.Exec(context.Background(), "DROP DATABASE IF EXISTS "+name+" WITH (FORCE)")
		ap.Close()
	})
	pool, err := pgxpool.New(ctx, pgtest.DatabaseURL(name))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pgtest.RunMigrations(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return pool
}

func TestUsageMergesStoredPushesAndKeepsTheNewestPerSeat(t *testing.T) {
	usageNoBoardNames(t)
	pool := usageThrowawayDB(t)
	ctx := context.Background()
	var ws, other string
	if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ('FounderOS','founderos','u1') RETURNING id::text`).Scan(&ws); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO workspaces (name, slug, owner_id) VALUES ('Personal','personal','u1') RETURNING id::text`).Scan(&other); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	day := now.In(time.Local).Format("2006-01-02") // buckets are local days, as in FounderOS v1 lib/usage.ts dayKey
	seatJSON := func(id, kind string, captured time.Time, in float64) string {
		b, _ := json.Marshal(devicepush.SeatUsage{ID: id, Kind: kind, Label: id, Source: "push", CapturedAt: captured.Format(time.RFC3339),
			Days: []devicepush.DayBucket{{Day: day, Tot: devicepush.Tot{In: in}}}, ByModel: map[string]devicepush.Tot{}})
		return string(b)
	}
	ins := func(wsID, id, payload string, at time.Time) {
		if _, err := pool.Exec(ctx, `INSERT INTO founderos_usage_snapshots (id, workspace_id, captured_at, payload) VALUES ($1,$2,$3,$4)`, id, wsID, at, payload); err != nil {
			t.Fatal(err)
		}
	}
	// a stored codex seat, a stored (older) copy of the device's claude seat,
	// and a row in another workspace that must not leak in
	ins(ws, "codex-mini", seatJSON("codex-mini", "codex", now.Add(-time.Hour), 7), now.Add(-time.Hour))
	ins(ws, "claude-alexs-macbook-pro", seatJSON("claude-alexs-macbook-pro", "claude", now.Add(-2*time.Hour), 999), now.Add(-2*time.Hour))
	ins(other, "claude-elsewhere", seatJSON("claude-elsewhere", "claude", now, 1), now)
	olJSON, _ := json.Marshal(devicepush.OllamaSnapshot{Kind: "ollama", ID: "ollama-mini", Label: "Ollama · mini", CapturedAt: now.Format(time.RFC3339),
		Lane: devicepush.OllamaLane{State: "down", Models: []devicepush.OllamaModel{}}})
	if _, err := pool.Exec(ctx, `INSERT INTO founderos_ollama_snapshots (id, workspace_id, captured_at, payload) VALUES ('ollama-mini',$1,$2,$3)`, ws, now, string(olJSON)); err != nil {
		t.Fatal(err)
	}

	recv := devicepush.NewReceiver(devicepush.NewMemStore(), "alexs-macbook-pro")
	if err := recv.Accept(ctx, devicepush.Payload{Device: "alexs-macbook-pro", CapturedAt: now.Format(time.RFC3339), Usage: []devicepush.SeatUsage{{
		ID: "claude-alexs-macbook-pro", Kind: "claude", Label: "Claude · MacBook", CapturedAt: now.Format(time.RFC3339),
		Days: []devicepush.DayBucket{{Day: day, Tot: devicepush.Tot{In: 100}}}, ByModel: map[string]devicepush.Tot{},
	}}}); err != nil {
		t.Fatal(err)
	}

	code, body := usageGet(t, &Deps{Pool: pool, Devices: recv})
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	var plan struct {
		Days     []struct{ In float64 }
		Machines []any
	}
	_ = json.Unmarshal(body["claude"], &plan)
	if len(plan.Machines) != 1 || len(plan.Days) != 7 || plan.Days[6].In != 100 {
		t.Fatalf("claude = %s, want only the device's fresh reading", body["claude"])
	}
	_ = json.Unmarshal(body["codex"], &plan)
	if len(plan.Machines) != 1 || len(plan.Days) != 7 || plan.Days[6].In != 7 {
		t.Fatalf("codex = %s, want the stored push", body["codex"])
	}
	var ol struct{ State string }
	_ = json.Unmarshal(body["ollama"], &ol)
	if ol.State != "down" {
		t.Fatalf("ollama = %s", body["ollama"])
	}
	if len(body["errors"]) != 0 && string(body["errors"]) != "null" {
		t.Fatalf("errors = %s", body["errors"])
	}
}

// The official Claude gauge travels as numbers only: a collector that read it
// with an OAuth token pushes utilization, reset times and the plan name, and
// the board shows the freshest machine's gauge. The token itself never
// reaches the payload or this API's response.
func TestUsageShowsTheOfficialClaudeGaugeAndNeverTheToken(t *testing.T) {
	usageNoBoardNames(t)
	const token = "sk-ant-oat01-NEVER-LEAVE-THIS-MAC-api-test"
	gauge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(401)
			return
		}
		_, _ = w.Write([]byte(`{"five_hour":{"utilization":37,"resets_at":"2099-01-01T00:00:00.000Z"},"seven_day":{"utilization":64,"resets_at":null}}`))
	}))
	defer gauge.Close()

	now := time.Now().UTC()
	dir := t.TempDir()
	if err := os.MkdirAll(dir+"/p", 0o755); err != nil {
		t.Fatal(err)
	}
	line := fmt.Sprintf(`{"type":"assistant","timestamp":%q,"message":{"model":"claude-opus-5","usage":{"input_tokens":100,"output_tokens":10}}}`+"\n", now.Add(-time.Minute).Format("2006-01-02T15:04:05.000Z"))
	if err := os.WriteFile(dir+"/p/s.jsonl", []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	g := collect.NewClaudeGauge(func(context.Context) string { return token })
	g.URL = gauge.URL
	seat := collect.WithClaudeOfficial(collect.ScanClaudeProjects(dir, now, collect.NewScanCache(), collect.SeatInfo{ID: "claude-alexs-macbook-pro", Label: "Claude · mbp"}), g.Read(context.Background()))

	recv := devicepush.NewReceiver(devicepush.NewMemStore(), "alexs-macbook-pro")
	p := devicepush.Payload{Device: "alexs-macbook-pro", Label: "mbp", CapturedAt: now.Format(time.RFC3339), Usage: []devicepush.SeatUsage{seat}}
	raw, _ := json.Marshal(p)
	if strings.Contains(string(raw), token) {
		t.Fatal("the OAuth token reached the push payload")
	}
	if err := recv.Accept(context.Background(), p); err != nil {
		t.Fatalf("the receiver refused a seat carrying the official gauge: %v", err)
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/founderos/pages/usage", nil)
	req.Header.Set("Cookie", "session=ok")
	router(t, &Deps{Devices: recv}).ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if strings.Contains(w.Body.String(), token) || strings.Contains(w.Body.String(), "sk-ant-") {
		t.Fatal("the OAuth token reached the API response")
	}
	var body struct {
		Claude struct {
			Official struct {
				Session struct {
					UsedPercent float64 `json:"usedPercent"`
					ResetsAt    *string `json:"resetsAt"`
				} `json:"session"`
				Weekly struct {
					UsedPercent float64 `json:"usedPercent"`
				} `json:"weekly"`
			} `json:"official"`
			Note string `json:"note"`
		} `json:"claude"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Claude.Official.Session.UsedPercent != 37 || body.Claude.Official.Weekly.UsedPercent != 64 ||
		body.Claude.Official.Session.ResetsAt == nil || *body.Claude.Official.Session.ResetsAt != "2099-01-01T00:00:00.000Z" {
		t.Fatalf("claude = %s", w.Body.String())
	}
	if body.Claude.Note != "burn measured from transcripts; limit % is the official gauge from this login" {
		t.Errorf("note = %q", body.Claude.Note)
	}
}
