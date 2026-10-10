package brainimport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestEnsureWorkspacesCreatesOnlyMissingHomedWorkspaces(t *testing.T) {
	var mu sync.Mutex
	created := map[string][]string{}
	newEngine := func(name string, existing ...string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer k-"+name {
				http.Error(w, `{"error":"missing_api_key"}`, http.StatusUnauthorized)
				return
			}
			switch r.Method {
			case http.MethodGet:
				var ws []map[string]string
				for _, slug := range existing {
					ws = append(ws, map[string]string{"id": "default:" + slug, "slug": slug})
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"workspaces": ws})
			case http.MethodPost:
				var body map[string]string
				_ = json.NewDecoder(r.Body).Decode(&body)
				mu.Lock()
				created[name] = append(created[name], body["slug"]+"="+body["name"])
				mu.Unlock()
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{}`))
			}
		}))
		t.Cleanup(srv.Close)
		return srv
	}
	hub := newEngine("hub", "founderos")
	mac := newEngine("macbook")
	engines := map[string]Endpoint{"hub": {URL: hub.URL, Key: "k-hub"}, "macbook": {URL: mac.URL, Key: "k-macbook"}}

	if err := EnsureWorkspaces(topo(t), engines); err != nil {
		t.Fatal(err)
	}
	if got := created["hub"]; len(got) != 1 || got[0] != "vantage=Vantage" {
		t.Errorf("hub created %v, want [vantage=Vantage]", got)
	}
	if got := created["macbook"]; len(got) != 1 || got[0] != "personal=Personal" {
		t.Errorf("macbook created %v, want [personal=Personal]", got)
	}
}
