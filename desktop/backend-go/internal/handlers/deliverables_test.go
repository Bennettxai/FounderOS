package handlers

import "testing"

func TestNormalizeDeliverableKind(t *testing.T) {
	tests := map[string]string{
		"report":  "report",
		" VIDEO ": "video",
		"unknown": "other",
		"":        "other",
	}
	for input, expected := range tests {
		if actual := normalizeDeliverableKind(input); actual != expected {
			t.Fatalf("normalizeDeliverableKind(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestNormalizeDeliverableStatus(t *testing.T) {
	tests := map[string]string{
		"delivered":     "delivered",
		" IN_PROGRESS ": "in_progress",
		"unknown":       "draft",
		"":              "draft",
	}
	for input, expected := range tests {
		if actual := normalizeDeliverableStatus(input); actual != expected {
			t.Fatalf("normalizeDeliverableStatus(%q) = %q, want %q", input, actual, expected)
		}
	}
}
