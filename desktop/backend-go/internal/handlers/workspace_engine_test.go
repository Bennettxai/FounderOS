package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPingEngineUsesAPIHealthBeforeLegacyHealth(t *testing.T) {
	var hitAPIHealth bool
	var hitLegacyHealth bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			hitAPIHealth = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"up"}`))
		case "/health":
			hitLegacyHealth = true
			w.WriteHeader(http.StatusNotFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	reachable, status, message := pingEngine(context.Background(), server.URL, "")
	if !reachable {
		t.Fatalf("expected reachable engine, got status=%d message=%q", status, message)
	}
	if status != http.StatusOK {
		t.Fatalf("expected 200 status, got %d", status)
	}
	if !hitAPIHealth {
		t.Fatal("expected /api/health to be called")
	}
	if hitLegacyHealth {
		t.Fatal("did not expect /health fallback after /api/health succeeded")
	}
}

func TestPingEngineFallsBackToLegacyHealth(t *testing.T) {
	var hitLegacyHealth bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/health":
			w.WriteHeader(http.StatusNotFound)
		case "/health":
			hitLegacyHealth = true
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	reachable, status, message := pingEngine(context.Background(), server.URL, "")
	if !reachable {
		t.Fatalf("expected reachable engine, got status=%d message=%q", status, message)
	}
	if status != http.StatusOK {
		t.Fatalf("expected 200 status, got %d", status)
	}
	if !hitLegacyHealth {
		t.Fatal("expected /health fallback to be called")
	}
}
