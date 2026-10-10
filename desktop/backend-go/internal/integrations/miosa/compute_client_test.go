package miosa

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestComputeClientListComputers(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/computers" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "cmp_1", "status": "active", "template_type": "business_os", "slug": "biz-1"},
			},
		})
	}))
	defer server.Close()

	client := NewComputeClient(server.URL, "test-key")
	computers, err := client.ListComputers(context.Background())
	if err != nil {
		t.Fatalf("ListComputers returned error: %v", err)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("Authorization header = %q", gotAuth)
	}
	if len(computers) != 1 || computers[0].ID != "cmp_1" || computers[0].Status != "active" || computers[0].Slug != "biz-1" {
		t.Fatalf("unexpected computers: %#v", computers)
	}
}

func TestComputeClientCreateComputer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/computers" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if body["name"] != "BusinessOS" || body["template_type"] != "business_os" || body["size"] != "small" {
			t.Fatalf("unexpected body: %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": "cmp_2", "status": "provisioning"})
	}))
	defer server.Close()

	client := NewComputeClient(server.URL, "test-key")
	computer, err := client.CreateComputer(context.Background(), "BusinessOS", "business_os", "small")
	if err != nil {
		t.Fatalf("CreateComputer returned error: %v", err)
	}
	if computer.ID != "cmp_2" || computer.Status != "provisioning" {
		t.Fatalf("unexpected computer: %#v", computer)
	}
}

func TestComputeClientReturnsStatusErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "platform unavailable"})
	}))
	defer server.Close()

	client := NewComputeClient(server.URL, "test-key")
	if _, err := client.ListComputers(context.Background()); err == nil {
		t.Fatal("expected ListComputers error")
	}
}

func TestComputeClientRequiresAPIKey(t *testing.T) {
	client := NewComputeClient("https://api.example.test", "")
	if _, err := client.ListComputers(context.Background()); err == nil {
		t.Fatal("expected missing API key error")
	}
}

func TestComputeClientNormalizesBaseURL(t *testing.T) {
	cases := map[string]string{
		"https://api.miosa.ai":        "https://api.miosa.ai/api/v1",
		"https://api.miosa.ai/api":    "https://api.miosa.ai/api/v1",
		"https://api.miosa.ai/api/v1": "https://api.miosa.ai/api/v1",
	}
	for in, want := range cases {
		if got := normalizeAPIBaseURL(in); got != want {
			t.Fatalf("normalizeAPIBaseURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestComputeClientStreamAndTerminalParsing(t *testing.T) {
	stream := parseStreamToken(map[string]interface{}{
		"token": map[string]interface{}{
			"token":      "stream-token",
			"expires_at": float64(123),
		},
	})
	if stream.Token != "stream-token" || stream.ExpiresAt != 123 {
		t.Fatalf("unexpected stream token: %#v", stream)
	}

	session := parseTerminalSession(map[string]interface{}{
		"session": map[string]interface{}{
			"session_id":  "term-1",
			"stream_auth": "auth-1",
			"expires_at":  int64(456),
		},
	})
	if session.SessionID != "term-1" || session.StreamAuth != "auth-1" || session.ExpiresAt != 456 {
		t.Fatalf("unexpected terminal session: %#v", session)
	}
}
