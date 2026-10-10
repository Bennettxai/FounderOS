package pagekit

import (
	"encoding/json"
	"testing"
	"time"
)

func TestPlural(t *testing.T) {
	for _, c := range []struct {
		n         int
		one, many string
		want      string
	}{
		{1, "inbox", "inboxes", "1 inbox"},
		{0, "inbox", "inboxes", "0 inboxes"},
		{3, "thread", "", "3 threads"},
	} {
		if got := Plural(c.n, c.one, c.many); got != c.want {
			t.Errorf("Plural(%d,%q) = %q, want %q", c.n, c.one, got, c.want)
		}
	}
}

func TestThousandsMatchesEnUSLocale(t *testing.T) {
	for n, want := range map[int]string{0: "0", 999: "999", 1000: "1,000", 24814: "24,814", -2049: "-2,049", 1234567: "1,234,567"} {
		if got := Thousands(n); got != want {
			t.Errorf("Thousands(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestShortDateAndWeekday(t *testing.T) {
	d := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	if got := ShortDate(d); got != "Sep 3" {
		t.Fatalf("ShortDate = %q", got)
	}
	if got := Weekday(d); got != "Thu" {
		t.Fatalf("Weekday = %q", got)
	}
}

func TestMeterJSONKnownAndUnknown(t *testing.T) {
	raw, _ := json.Marshal([]Meter{Known("Email", 0.5, "2 unread", Ramp1), Unknown("Beehiiv", "offline", OK)})
	want := `[{"label":"Email","frac":0.5,"display":"2 unread","hue":"var(--bn-text-2)"},{"label":"Beehiiv","frac":null,"display":"offline","hue":"var(--bn-ok)"}]`
	if string(raw) != want {
		t.Fatalf("got %s\nwant %s", raw, want)
	}
}

func TestFracNeverDividesByZero(t *testing.T) {
	if Frac(3, 0) != 0 || Frac(1, 4) != 0.25 {
		t.Fatal("Frac")
	}
}

// The kit's DotMatrix keys columns by label; two issues sent the same day
// must not share one (Svelte throws each_key_duplicate and the page dies).
func TestUniqueLabelsNumbersCollisions(t *testing.T) {
	got := UniqueLabels([]Point{{Label: "Sep 3", Count: 1}, {Label: "Sep 3", Count: 2}, {Label: "Sep 4"}, {Label: "Sep 3"}})
	want := []string{"Sep 3", "Sep 3 2", "Sep 4", "Sep 3 3"}
	for i, p := range got {
		if p.Label != want[i] {
			t.Fatalf("labels = %+v", got)
		}
	}
	if got[1].Count != 2 {
		t.Fatal("counts kept")
	}
}
