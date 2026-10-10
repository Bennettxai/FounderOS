package ventures

import "testing"

func TestVenturesMatchFounderosOS(t *testing.T) {
	if len(All) != 2 || All[0].ID != "vantage" || All[1].ID != "launchpad-cohort" {
		t.Fatalf("ventures = %+v", All)
	}
	if All[0].Color != "#00ffaa" || All[1].Color != "#d9263f" {
		t.Fatal("brand colors")
	}
	set := AgentSet("launchpad-cohort")
	for _, a := range []string{"dmflow-mcp", "flexpay-financing", "conductor", "data-agent"} {
		if !set[a] {
			t.Errorf("launchpad-cohort should include %s", a)
		}
	}
	if got := ForAgent("conductor"); len(got) != 2 {
		t.Errorf("shared ops serve both ventures, got %d", len(got))
	}
	if got := ForAgent("vantage-sales"); len(got) != 1 || got[0].ID != "vantage" {
		t.Errorf("vantage-sales → %+v", got)
	}
}
