package api

import (
	"testing"
	"unicode/utf8"
)

// clip cuts by characters, like JS slice on these strings: a byte cut can
// split a UTF-8 sequence and ship invalid text to the page.
func TestClipNeverSplitsACharacter(t *testing.T) {
	s := "Café — Sîrghie 🙂 ok"
	for n := 0; n <= len(s)+1; n++ {
		if got := clip(s, n); !utf8.ValidString(got) {
			t.Fatalf("clip(%d) = %q is not valid UTF-8", n, got)
		}
	}
	if got := clip("héllo", 2); got != "hé" {
		t.Fatalf("clip = %q, want hé", got)
	}
	if got := clip("short", 140); got != "short" {
		t.Fatalf("clip = %q", got)
	}
}
