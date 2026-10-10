package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"time"
)

// httpClient is shared so we keep one cookie jar across calls in a single
// invocation. The jar is loaded once from --cookie if provided.
var httpClient = func() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar, Timeout: 30 * time.Second}
}()

// remoteGet does an authenticated GET against the configured base URL and
// JSON-decodes the response into out.
func remoteGet(opts globalOpts, path string, out any) error {
	req, err := buildRequest(http.MethodGet, opts, path, nil)
	if err != nil {
		return err
	}
	return doRequest(req, out)
}

// remotePost JSON-encodes body, POSTs it, decodes the response.
func remotePost(opts globalOpts, path string, body, out any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	req, err := buildRequest(http.MethodPost, opts, path, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return doRequest(req, out)
}

func buildRequest(method string, opts globalOpts, path string, body io.Reader) (*http.Request, error) {
	base, err := url.Parse(opts.remote)
	if err != nil {
		return nil, fmt.Errorf("invalid --remote: %w", err)
	}
	rel, err := base.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("bad path %q: %w", path, err)
	}
	req, err := http.NewRequest(method, rel.String(), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if opts.cookie != "" {
		applyCookieJar(opts.cookie, rel, req)
	}
	return req, nil
}

func doRequest(req *http.Request, out any) error {
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode >= 400 {
		// Surface the server's error payload verbatim; it's almost always
		// a clear JSON object the operator wants to see.
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(raw))
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("decode: %w; body=%s", err, string(raw))
		}
	}
	return nil
}

// applyCookieJar reads a Netscape-format cookie file and seeds the request
// with any cookies whose Domain matches the request URL. We don't use
// cookiejar.Jar.SetCookies here because the file format is plain-text and
// portable across CLIs (curl, wget) that share the same jar.
func applyCookieJar(path string, target *url.URL, req *http.Request) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := newLineScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			continue
		}
		domain, name, value := fields[0], fields[5], fields[6]
		if !strings.HasSuffix(target.Hostname(), strings.TrimPrefix(domain, ".")) {
			continue
		}
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
}
