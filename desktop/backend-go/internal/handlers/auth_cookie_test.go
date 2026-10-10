package handlers

import (
	"net/http"
	"testing"
)

func TestSessionCookieSameSiteSupportsPackagedDesktop(t *testing.T) {
	if got := sessionCookieSameSite(true); got != http.SameSiteNoneMode {
		t.Fatalf("production cookie must support app:// Electron requests, got %v", got)
	}
	if got := sessionCookieSameSite(false); got != http.SameSiteLaxMode {
		t.Fatalf("development cookie should remain SameSite=Lax, got %v", got)
	}
}
