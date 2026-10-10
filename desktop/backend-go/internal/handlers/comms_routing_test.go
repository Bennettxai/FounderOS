package handlers

import "testing"

func TestNormalizeRouteKey(t *testing.T) {
	tests := []struct {
		name       string
		scope      string
		externalID string
		want       string
		ok         bool
	}{
		{name: "account uses provider wildcard", scope: "account", externalID: "ignored", want: "*", ok: true},
		{name: "conversation trims id", scope: "conversation", externalID: " channel-1 ", want: "channel-1", ok: true},
		{name: "conversation requires id", scope: "conversation", externalID: " ", ok: false},
		{name: "unknown scope rejected", scope: "organization", externalID: "x", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := normalizeRouteKey(tt.scope, tt.externalID)
			if got != tt.want || ok != tt.ok {
				t.Fatalf("normalizeRouteKey(%q, %q) = (%q, %v), want (%q, %v)", tt.scope, tt.externalID, got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestCommunicationRouteIndexConversationOverridesAccount(t *testing.T) {
	account := communicationRoute{Provider: "slack", Scope: "account", WorkspaceID: "agency"}
	conversation := communicationRoute{Provider: "slack", Scope: "conversation", ExternalID: "channel-1", WorkspaceID: "miosa"}
	index := communicationRouteIndex{
		accounts: map[string]communicationRoute{"slack": account},
		conversations: map[string]communicationRoute{
			"slack\x00channel-1": conversation,
		},
	}

	got, ok := index.resolve("slack", "channel-1")
	if !ok || got.WorkspaceID != "miosa" {
		t.Fatalf("conversation route = (%+v, %v), want MIOSA override", got, ok)
	}

	got, ok = index.resolve("slack", "channel-2")
	if !ok || got.WorkspaceID != "agency" {
		t.Fatalf("account route = (%+v, %v), want Agency fallback", got, ok)
	}

	if _, ok = index.resolve("teams", "channel-1"); ok {
		t.Fatal("unassigned provider unexpectedly resolved")
	}
}

func TestFilterUnifiedEmailsForWorkspace(t *testing.T) {
	index := communicationRouteIndex{
		accounts: map[string]communicationRoute{
			"outlook": {Provider: "outlook", Scope: "account", WorkspaceID: "northstar"},
		},
		conversations: map[string]communicationRoute{
			"gmail\x00thread-northstar": {Provider: "gmail", Scope: "conversation", ExternalID: "thread-northstar", WorkspaceID: "northstar"},
			"gmail\x00thread-agency":    {Provider: "gmail", Scope: "conversation", ExternalID: "thread-agency", WorkspaceID: "agency"},
		},
	}
	emails := []unifiedEmail{
		{ID: "northstar-gmail", Provider: "gmail", ThreadID: "thread-northstar"},
		{ID: "agency-gmail", Provider: "gmail", ThreadID: "thread-agency"},
		{ID: "unrouted-gmail", Provider: "gmail", ThreadID: "thread-unrouted"},
		{ID: "northstar-outlook", Provider: "outlook", ThreadID: "outlook-conversation"},
	}

	got := filterUnifiedEmailsForWorkspace(emails, index, "northstar")
	if len(got) != 2 || got[0].ID != "northstar-gmail" || got[1].ID != "northstar-outlook" {
		t.Fatalf("filtered emails = %#v, want only explicitly Northstar-routed email", got)
	}
}
