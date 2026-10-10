package google

import "testing"

func TestGetSafeRedirectURLDefaultsToConnectorsOnConfiguredPort(t *testing.T) {
	t.Setenv("FRONTEND_URL", "")
	t.Setenv("FRONTEND_PORT", "5273")

	got := getSafeRedirectURL("", "google_calendar")
	want := "http://localhost:5273/connectors?connected=google_calendar"

	if got != want {
		t.Fatalf("getSafeRedirectURL() = %q, want %q", got, want)
	}
}

func TestGetSafeRedirectURLAllowsConfiguredFrontendURL(t *testing.T) {
	t.Setenv("FRONTEND_URL", "http://localhost:5273")
	t.Setenv("FRONTEND_PORT", "")

	requested := "http://localhost:5273/connectors"
	got := getSafeRedirectURL(requested, "google_calendar")
	want := "http://localhost:5273/connectors?connected=google_calendar"

	if got != want {
		t.Fatalf("getSafeRedirectURL() = %q, want %q", got, want)
	}
}

func TestGetSafeRedirectURLAllowsDesktopDeepLink(t *testing.T) {
	t.Setenv("FRONTEND_URL", "http://localhost:5273")
	t.Setenv("FRONTEND_PORT", "")

	got := getSafeRedirectURL("businessos://oauth", "google_calendar")
	want := "businessos://oauth?connected=google_calendar"

	if got != want {
		t.Fatalf("getSafeRedirectURL() = %q, want desktop deep link %q", got, want)
	}
}

func TestGetSafeRedirectURLBlocksUntrustedOrigin(t *testing.T) {
	t.Setenv("FRONTEND_URL", "http://localhost:5273")
	t.Setenv("FRONTEND_PORT", "")

	got := getSafeRedirectURL("https://evil.example/connectors", "google_calendar")
	want := "http://localhost:5273/connectors?connected=google_calendar"

	if got != want {
		t.Fatalf("getSafeRedirectURL() = %q, want safe default %q", got, want)
	}
}
