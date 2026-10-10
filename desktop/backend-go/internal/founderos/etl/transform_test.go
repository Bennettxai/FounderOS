package etl

import (
	"testing"
	"time"
)

func TestParseInstantRules(t *testing.T) {
	cases := []struct {
		in   string
		rule string
		want time.Time
	}{
		{"2026-09-24T21:00:06.920Z", "D1", time.Date(2026, 9, 24, 21, 0, 6, 920_000_000, time.UTC)},
		{"2026-08-09T15:15:50-05:00", "D1", time.Date(2026, 8, 9, 20, 15, 50, 0, time.UTC)},
		{"2026-08-12", "D2", time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)},
		{"2026-09-01T10:00:00", "no-offset", time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)},
		{"2026-09-01 10:00:00", "no-offset", time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		got, rule, err := ParseInstant(c.in)
		if err != nil {
			t.Fatalf("%s: %v", c.in, err)
		}
		if rule != c.rule || !got.Equal(c.want) {
			t.Errorf("%s: got %v (%s), want %v (%s)", c.in, got, rule, c.want, c.rule)
		}
	}
	if _, _, err := ParseInstant("yesterday"); err == nil {
		t.Error("unparseable instant accepted")
	}
}

func TestDayAndMonth(t *testing.T) {
	if d, err := ParseDay("2026-03-14"); err != nil || d != time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC) {
		t.Errorf("ParseDay = %v, %v", d, err)
	}
	if _, err := ParseDay("2026-03-14T00:00:00Z"); err == nil {
		t.Error("D3 must reject an instant")
	}
	if m, err := ParseMonth("2026-08"); err != nil || m != time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC) {
		t.Errorf("ParseMonth = %v, %v", m, err)
	}
}

func TestConvertBlankNullableInstantIsD6(t *testing.T) {
	cs := &convStats{}
	v, err := convert("", kTSN, cs)
	if err != nil || v != nil || cs.d6 != 1 {
		t.Fatalf("got %v, %v, d6=%d", v, err, cs.d6)
	}
	if _, err := convert("", kTS, cs); err == nil {
		t.Fatal("blank NOT NULL instant accepted")
	}
}

func TestConvertJSONShapes(t *testing.T) {
	cs := &convStats{}
	if _, err := convert(`["a","b"]`, kJSONStrs, cs); err != nil {
		t.Error(err)
	}
	if _, err := convert(`["a",1]`, kJSONStrs, cs); err == nil {
		t.Error("non-string element accepted in string[]")
	}
	if _, err := convert(`{"a":1}`, kJSONArr, cs); err == nil {
		t.Error("object accepted as array")
	}
	if _, err := convert(`[1`, kJSONArr, cs); err == nil {
		t.Error("broken JSON accepted")
	}
	if _, err := convert(`[]`, kJSONObj, cs); err == nil {
		t.Error("array accepted as object")
	}
}

func TestConvertBoolAndInts(t *testing.T) {
	cs := &convStats{}
	if v, _ := convert(int64(1), kBool, cs); v != true {
		t.Error("1 is not true")
	}
	if _, err := convert(int64(2), kBool, cs); err == nil {
		t.Error("2 accepted as a flag")
	}
	if _, err := convert(int64(1)<<40, kInt, cs); err == nil {
		t.Error("INTEGER overflow accepted")
	}
	if v, err := convert(int64(1)<<40, kBig, cs); err != nil || v != int64(1)<<40 {
		t.Error("BIGINT rejected")
	}
	if _, err := convert(nil, kText, cs); err == nil {
		t.Error("NULL accepted in NOT NULL")
	}
}

func TestWorkspaceRules(t *testing.T) {
	st := &runState{contactWS: map[string]string{}, proposalWS: map[string]string{"p1": WSVantage}}
	check := func(name string, got string, err error, want string) {
		t.Helper()
		if err != nil || got != want {
			t.Errorf("%s = %q, %v; want %q", name, got, err, want)
		}
	}
	g, err := workflowWS(map[string]any{"id": "wf-vantage-sales"}, nil, st)
	check("workflow vantage", g, err, WSVantage)
	g, err = workflowWS(map[string]any{"id": "wf-lc-delivery"}, nil, st)
	check("workflow lc", g, err, WSLaunchpadCohort)
	g, err = workflowWS(map[string]any{"id": "wf-user-1"}, nil, st)
	check("workflow other", g, err, WSFounderOS)

	g, err = decisionWS(map[string]any{"id": "proposal:p1"}, nil, st)
	check("decision proposal", g, err, WSVantage)
	g, err = decisionWS(map[string]any{"id": "ws-123/file.md"}, nil, st)
	check("decision superset", g, err, WSFounderOS)

	g, err = bankWS(map[string]any{"business": "Vantage"}, nil, st)
	check("bank vantage", g, err, WSVantage)
	g, err = bankWS(map[string]any{"business": "General Operations"}, nil, st)
	check("bank general", g, err, WSLaunchpadCohort)

	for raw, want := range map[string]string{"gold": WSPersonal, "blue": WSVantage, "platinum": WSLaunchpadCohort} {
		g, err = ledgerWS(nil, map[string]any{"card": raw}, st)
		check("ledger "+raw, g, err, want)
	}
	for raw, want := range map[any]string{"Business": "platinum", "vantage": "blue", " GOLD ": "gold", "weird": "platinum", nil: "platinum"} {
		if got := NormalizeCardID(raw); got != want {
			t.Errorf("NormalizeCardID(%v) = %s, want %s", raw, got, want)
		}
	}

	g, err = paykitWS(map[string]any{"account": "paykit-lc"}, nil, st)
	check("paykit aa", g, err, WSLaunchpadCohort)
	if _, err := paykitWS(map[string]any{"account": "other"}, nil, st); err == nil {
		t.Error("unknown paykit account accepted")
	}
	if _, err := funnelContactWS(map[string]any{"venture": "founderos"}, map[string]any{"id": "x"}, st); err == nil {
		t.Error("venture outside the enum accepted")
	}
	if _, err := funnelTouchWS(map[string]any{"contact_id": "nobody"}, nil, st); err == nil {
		t.Error("touch of an unknown contact accepted")
	}
}

func TestSpecsCover55TargetsAnd57SourceTables(t *testing.T) {
	specs := Specs()
	targets := map[string]bool{}
	sources := map[string]bool{}
	for _, sp := range specs {
		if targets[sp.target] {
			t.Fatalf("duplicate target %s", sp.target)
		}
		targets[sp.target] = true
		for _, s := range sp.sources {
			sources[s.file+":"+s.table] = true
		}
		if len(sp.key) == 0 {
			t.Errorf("%s has no key", sp.target)
		}
		for _, k := range sp.key {
			if _, ok := sp.colByDst(k); !ok && k != "workspace_id" {
				t.Errorf("%s key %s is not a column", sp.target, k)
			}
		}
		if sp.ws == "" && sp.wsFn == nil {
			t.Errorf("%s has no workspace rule", sp.target)
		}
	}
	for _, want := range wantFounderosTables {
		if !targets[want] {
			t.Errorf("no spec for %s", want)
		}
	}
	if len(targets) != 55 {
		t.Errorf("targets = %d, want 55", len(targets))
	}
	// 52 founderos-os.db tables (roadmap_items, phases and domains load since
	// migration 165, none is dropped), plus bank, ledger, paykit and the two
	// state.db tables = 57.
	if len(sources) != 57 {
		t.Errorf("source tables = %d, want 57", len(sources))
	}
}

func TestParseWorkspaceMap(t *testing.T) {
	m, err := ParseWorkspaceMap("personal=founderos-personal, founderos=0f9c2c7e-1111-4222-8333-944455556666")
	if err != nil || m[WSPersonal] != "founderos-personal" || m[WSFounderOS] != "0f9c2c7e-1111-4222-8333-944455556666" {
		t.Fatalf("got %v, %v", m, err)
	}
	if m, err := ParseWorkspaceMap(""); err != nil || len(m) != 0 {
		t.Fatalf("empty map: %v, %v", m, err)
	}
	for _, bad := range []string{"personal", "nosuch=x", "personal="} {
		if _, err := ParseWorkspaceMap(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
