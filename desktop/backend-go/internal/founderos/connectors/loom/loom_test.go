package loom

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

// Shaped like GET https://www.loom.com/v1/oembed?url=...
const oembedFixture = `{"type":"video","version":"1.0","html":"<iframe src=\"https://www.loom.com/embed/abc123\"></iframe>","height":null,"width":null,"provider_name":"Loom","provider_url":"https://www.loom.com","thumbnail_height":720,"thumbnail_width":1280,"thumbnail_url":"https://cdn.loom.com/sessions/thumbnails/abc123-00001.gif","duration":184.5,"title":"FounderOS v1 walkthrough","author_name":"Alex"}`

func connector(t *testing.T, h http.HandlerFunc) *Connector {
	c := New(connectors.Resolver{})
	if h != nil {
		srv := httptest.NewServer(h)
		t.Cleanup(srv.Close)
		c.OEmbedURL = srv.URL + "/v1/oembed"
	}
	return c
}

func TestMetaMatchesFounderosOS(t *testing.T) {
	if Meta.ID != "loom" || Meta.Name != "Loom" || Meta.Kind != connectors.KindCreative {
		t.Fatalf("Meta = %+v", Meta)
	}
}

func TestIsLoomURL(t *testing.T) {
	for u, want := range map[string]bool{
		"https://www.loom.com/share/abc123":        true,
		"https://loom.com/embed/abc123":            true,
		"http://WWW.LOOM.COM/share/abc":            true,
		"https://vimeo.com/123":                    false,
		"not a url":                                false,
		"javascript:alert(1)//loom.com/share/x":    false,
		"https://loom.com.evil.example/share/x":    false,
		"https://evil.example/?u=https://loom.com": false,
		"//www.loom.com/share/abc":                 false,
		"":                                         false,
	} {
		if got := IsLoomURL(u); got != want {
			t.Errorf("IsLoomURL(%q) = %v, want %v", u, got, want)
		}
	}
}

// Loom publishes no account API: the lane is link-level and needs no key, so
// there is no not_configured state. Any HTTP answer means the lane is up.
func TestStatusConnectedOnAnyAnswer(t *testing.T) {
	for _, code := range []int{200, 404} {
		var probe string
		c := connector(t, func(w http.ResponseWriter, r *http.Request) {
			probe = r.URL.Query().Get("url")
			w.WriteHeader(code)
		})
		s := c.Status(context.Background())
		if s.State != connectors.StateConnected || s.ID != "loom" {
			t.Fatalf("%d: status = %+v", code, s)
		}
		if !strings.Contains(strings.ToLower(s.Detail), "no account api") || strings.Contains(strings.ToUpper(s.Detail), "LOOM_API_KEY") {
			t.Errorf("detail = %q", s.Detail)
		}
		if s.Meta["scope"] != "link-level" || s.Meta["auth"] != "none required" {
			t.Errorf("meta = %v", s.Meta)
		}
		if probe != ProbeURL {
			t.Errorf("probe url = %q", probe)
		}
	}
}

func TestStatusErrorWhenUnreachable(t *testing.T) {
	c := New(connectors.Resolver{})
	c.OEmbedURL = "http://127.0.0.1:1/v1/oembed"
	s := c.Status(context.Background())
	if s.State != connectors.StateError || !strings.HasPrefix(s.Detail, "Loom oEmbed unreachable: ") {
		t.Fatalf("status = %+v", s)
	}
}

func TestVideoMeta(t *testing.T) {
	var asked string
	c := connector(t, func(w http.ResponseWriter, r *http.Request) {
		asked = r.URL.Query().Get("url")
		io.WriteString(w, oembedFixture)
	})
	m, err := c.VideoMeta(context.Background(), "https://www.loom.com/share/abc123")
	if err != nil {
		t.Fatal(err)
	}
	if m.Title != "FounderOS v1 walkthrough" || m.Author == nil || *m.Author != "Alex" ||
		m.DurationSeconds == nil || *m.DurationSeconds != 184.5 || m.ThumbnailURL == nil || !strings.Contains(*m.ThumbnailURL, "cdn.loom.com") {
		t.Fatalf("meta = %+v", m)
	}
	if asked != "https://www.loom.com/share/abc123" {
		t.Errorf("asked = %q", asked)
	}
}

func TestVideoMetaOptionalFieldsAreNil(t *testing.T) {
	c := connector(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"title":"Untitled","duration":"long"}`)
	})
	m, err := c.VideoMeta(context.Background(), "https://loom.com/share/x")
	if err != nil || m.Author != nil || m.ThumbnailURL != nil || m.DurationSeconds != nil {
		t.Fatalf("m=%+v err=%v", m, err)
	}
}

func TestVideoMetaUnavailable(t *testing.T) {
	calls := 0
	c := connector(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if strings.Contains(r.URL.Query().Get("url"), "private") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		io.WriteString(w, `{"type":"video"}`)
	})
	if _, err := c.VideoMeta(context.Background(), "https://vimeo.com/1"); !errors.Is(err, ErrNotLoomURL) {
		t.Errorf("non-Loom link: err = %v", err)
	}
	if calls != 0 {
		t.Errorf("a non-Loom link must never be fetched")
	}
	if m, err := c.VideoMeta(context.Background(), "https://www.loom.com/share/private"); err == nil || m != nil {
		t.Errorf("private video: m=%v err=%v", m, err)
	}
	if m, err := c.VideoMeta(context.Background(), "https://www.loom.com/share/notitle"); err == nil || m != nil {
		t.Errorf("no title: m=%v err=%v", m, err)
	}
}
