package terminal

import (
	"net/http"
	"testing"
)

func TestCheckWebSocketOriginAllowsLocalDesktopRunner(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "/api/terminal/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", "http://localhost:5273")

	if !checkWebSocketOrigin(request) {
		t.Fatal("expected BusinessOS local desktop runner origin to be allowed")
	}
}

func TestCheckWebSocketOriginRejectsExternalOrigin(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "/api/terminal/ws", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Origin", "http://evil.example")

	if checkWebSocketOrigin(request) {
		t.Fatal("expected external websocket origin to be rejected")
	}
}
