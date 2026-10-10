package handlers

import "testing"

func TestCalendarVisibilityPredicate(t *testing.T) {
	t.Run("workspace scope excludes personal and other workspace events", func(t *testing.T) {
		if got := calendarVisibilityPredicate(true, false); got != "workspace_id = $3" {
			t.Fatalf("scoped calendar predicate = %q, want workspace-only visibility", got)
		}
	})

	t.Run("workspace scope can opt into the current user's personal calendar", func(t *testing.T) {
		if got := calendarVisibilityPredicate(true, true); got != "(workspace_id = $3 OR (user_id = $4 AND workspace_id IS NULL))" {
			t.Fatalf("scoped calendar predicate = %q, want workspace plus personal visibility", got)
		}
	})

	t.Run("personal scope returns only the current user's events", func(t *testing.T) {
		if got := calendarVisibilityPredicate(false, false); got != "user_id = $3 AND workspace_id IS NULL" {
			t.Fatalf("personal calendar predicate = %q, want personal-only visibility", got)
		}
	})
}
