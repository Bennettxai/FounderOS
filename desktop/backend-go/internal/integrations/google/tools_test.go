package google

import (
	"testing"

	"github.com/rhl/businessos-backend/internal/config"
)

func TestResolveToolRedirectURLKeepsCalendarCallbackSeparate(t *testing.T) {
	cfg := &config.Config{
		GoogleRedirectURI:            "http://localhost:8801/api/v1/auth/oauth/google/callback",
		GoogleIntegrationRedirectURI: "http://localhost:8801/api/integrations/google/callback",
		GoogleCalendarRedirectURI:    "http://localhost:8801/api/integrations/google_calendar/callback",
	}

	got := resolveToolRedirectURL(cfg, "google_calendar")
	want := "http://localhost:8801/api/integrations/google_calendar/callback"
	if got != want {
		t.Fatalf("resolveToolRedirectURL() = %q, want %q", got, want)
	}
}

func TestGrantsToolAccessUsesActualToolScopes(t *testing.T) {
	tests := []struct {
		name   string
		toolID string
		scopes []string
		want   bool
	}{
		{
			name:   "calendar rejects profile-only token",
			toolID: "google_calendar",
			scopes: []string{"openid", "https://www.googleapis.com/auth/userinfo.email"},
			want:   false,
		},
		{
			name:   "calendar accepts full calendar grant",
			toolID: "google_calendar",
			scopes: []string{"https://www.googleapis.com/auth/calendar"},
			want:   true,
		},
		{
			name:   "gmail token cannot authorize calendar",
			toolID: "google_calendar",
			scopes: []string{"https://mail.google.com/"},
			want:   false,
		},
		{
			name:   "gmail accepts full gmail grant",
			toolID: "google_gmail",
			scopes: []string{"https://mail.google.com/"},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := grantsToolAccess(tt.toolID, tt.scopes); got != tt.want {
				t.Fatalf("grantsToolAccess() = %v, want %v", got, tt.want)
			}
		})
	}
}
