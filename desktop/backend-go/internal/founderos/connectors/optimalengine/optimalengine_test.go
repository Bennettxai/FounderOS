package optimalengine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

func engine(t *testing.T, key string, workspaces string, healthy bool) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+key {
			w.WriteHeader(401)
			return
		}
		switch r.URL.Path {
		case "/api/health":
			if !healthy {
				_, _ = w.Write([]byte(`{"status":"down"}`))
				return
			}
			_, _ = w.Write([]byte(`{"status":"up"}`))
		case "/api/workspaces":
			_, _ = w.Write([]byte(`{"workspaces":[` + workspaces + `]}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestStatusAcrossEngines(t *testing.T) {
	hub := engine(t, "h", `{"slug":"default"},{"slug":"founderos"},{"slug":"vantage"}`, true)
	mac := engine(t, "m", `{"slug":"personal"}`, true)
	c := New([]Engine{{Name: "hub", URL: hub.URL, Key: "h"}, {Name: "macbook", URL: mac.URL, Key: "m"}})
	st := c.Status(context.Background())
	if st.State != connectors.StateConnected || !strings.Contains(st.Detail, "2/2 engines up") || !strings.Contains(st.Detail, "3 workspaces") {
		t.Fatalf("status = %+v (the legacy 'default' workspace is not counted)", st)
	}
}

func TestOneEngineDownIsAnErrorNamingIt(t *testing.T) {
	hub := engine(t, "h", `{"slug":"founderos"}`, true)
	mac := engine(t, "m", ``, false)
	c := New([]Engine{{Name: "hub", URL: hub.URL, Key: "h"}, {Name: "macbook", URL: mac.URL, Key: "m"}})
	st := c.Status(context.Background())
	if st.State != connectors.StateError || !strings.Contains(st.Detail, "macbook") || !strings.Contains(st.Detail, "1/2") {
		t.Fatalf("status = %+v", st)
	}
	bad := New([]Engine{{Name: "hub", URL: hub.URL, Key: "wrong"}})
	if st := bad.Status(context.Background()); st.State != connectors.StateError || !strings.Contains(st.Detail, "401") {
		t.Fatalf("auth failure must read as error: %+v", st)
	}
}

func TestNoEnginesIsNotConfigured(t *testing.T) {
	if st := New(nil).Status(context.Background()); st.State != connectors.StateNotConfigured {
		t.Fatalf("status = %+v", st)
	}
}
