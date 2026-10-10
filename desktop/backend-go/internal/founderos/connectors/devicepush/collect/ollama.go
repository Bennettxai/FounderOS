package collect

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

// The Ollama lane (lib/connectors/ollama-usage.ts): which plan is signed in
// (POST /api/me on the LOCAL server), which models are cloud vs local, and
// chat vs embedding requests per window from the server log. Ollama logs no
// tokens and ollama.com has no usage API, so this lane never invents them.

const (
	ollamaNote     = "Ollama logs requests, not tokens, and ollama.com has no usage API: plan % lives only at ollama.com/settings"
	ollamaTailSize = 4 * 1024 * 1024
)

func tailLines(file string) []string {
	f, err := os.Open(file)
	if err != nil {
		return nil
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil
	}
	n := fi.Size()
	if n > ollamaTailSize {
		n = ollamaTailSize
	}
	buf := make([]byte, n)
	if _, err := f.ReadAt(buf, fi.Size()-n); err != nil && err != io.EOF {
		return nil
	}
	return strings.Split(string(buf), "\n")
}

// OllamaRequestCounts reads server-1.log (last week's rotation) and
// server.log; nil when neither is readable.
func OllamaRequestCounts(dir string, now time.Time) *devicepush.RequestWindows {
	var lines []string
	found := false
	for _, name := range []string{"server-1.log", "server.log"} {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			found = true
			lines = append(lines, tailLines(p)...)
		}
	}
	if !found {
		return nil
	}
	c := CountOllamaRequests(lines, now)
	return &c
}

var v1Suffix = regexp.MustCompile(`/v1/?$`)

// ReadOllamaLane reads the local Ollama server. client nil means a plain
// client: these are requests to the Mac's own server, not outbound writes.
func ReadOllamaLane(ctx context.Context, client *http.Client, baseURL, logDir string, now time.Time) devicepush.OllamaLane {
	if client == nil {
		client = &http.Client{}
	}
	baseURL = v1Suffix.ReplaceAllString(baseURL, "")
	lane := devicepush.OllamaLane{State: "down", Models: []devicepush.OllamaModel{}, Note: ollamaNote, Requests: OllamaRequestCounts(logDir, now)}

	get := func(method, path string, timeout time.Duration, into any) bool {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, method, baseURL+path, nil)
		if err != nil {
			return false
		}
		res, err := client.Do(req)
		if err != nil {
			return false
		}
		defer res.Body.Close()
		return res.StatusCode >= 200 && res.StatusCode <= 299 && json.NewDecoder(res.Body).Decode(into) == nil
	}

	var tags struct {
		Models []map[string]any `json:"models"`
	}
	if !get(http.MethodGet, "/api/tags", 1500*time.Millisecond, &tags) {
		return lane
	}
	lane.State = "up"
	for _, m := range tags.Models {
		name, ok := m["name"].(string)
		if !ok {
			continue
		}
		remote, _ := m["remote_host"].(string)
		lane.Models = append(lane.Models, devicepush.OllamaModel{Name: name, Cloud: remote != "" || strings.HasSuffix(name, ":cloud")})
	}
	var me struct {
		Plan string `json:"plan"`
	}
	// not signed in to ollama.com, or offline: the plan stays unknown
	if get(http.MethodPost, "/api/me", 2500*time.Millisecond, &me) {
		lane.Plan = OllamaPlanName(me.Plan)
	}
	return lane
}
