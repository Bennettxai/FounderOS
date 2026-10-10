// Package foreplay ports FounderOS v1's Foreplay connector
// (lib/connectors/foreplay.ts), the Foreplay public API client behind
// Adscout and the AdPilot ad library (lib/foreplay/client.ts), and the read
// side of Adscout's local snapshot store (lib/foreplay/store.ts).
//
// Status is key presence plus the store's last-synced credit meter. It never
// calls the API, because every call spends account credits.
package foreplay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
)

var Meta = connectors.Meta{ID: "foreplay", Name: "Foreplay", Kind: connectors.KindCreative}

const (
	KeyName        = "FOREPLAY_API_KEY"
	DefaultBaseURL = "https://public.api.foreplay.co"
	Timeout        = 20 * time.Second
)

var ErrNotConfigured = errors.New("foreplay: FOREPLAY_API_KEY is not set")

type Connector struct {
	res connectors.Resolver
	// Files are the fallback credential files (social-media, then clue-agent).
	Files   []string
	BaseURL string
	Store   Store
}

func New(res connectors.Resolver) *Connector {
	social, clue, _, _ := connectors.CredFiles()
	return &Connector{res: res, Files: []string{social, clue}, BaseURL: DefaultBaseURL, Store: DefaultStore()}
}

func (c *Connector) key() string { return c.res.Resolve(KeyName, c.Files...) }

// Client returns an API client for the resolved key, or ErrNotConfigured.
func (c *Connector) Client() (*Client, error) {
	key := c.key()
	if key == "" {
		return nil, ErrNotConfigured
	}
	return &Client{key: key, baseURL: strings.TrimRight(c.BaseURL, "/"), http: connectors.HTTPClient(Timeout)}, nil
}

func (c *Connector) Status(context.Context) connectors.Status {
	st := connectors.Status{ID: Meta.ID, Name: Meta.Name, Kind: Meta.Kind}
	if c.key() == "" {
		st.State = connectors.StateNotConfigured
		st.Detail = "Ad intelligence for Adscout (Spyder watchlist, Discovery search, sponsor dossiers). " + connectors.SetKeys("FOREPLAY_API_KEY")
		return st
	}
	usage, _ := c.Store.ReadUsage() // unreadable reads as unknown, never zero
	meta, _ := c.Store.ReadMeta()
	credits := "credits unknown until first sync"
	if usage != nil {
		credits = fmt.Sprintf("%s/%s credits", formatNumber(usage.RemainingCredits), formatNumber(usage.TotalCredits))
		st.Meta = map[string]any{"remaining": usage.RemainingCredits, "total": usage.TotalCredits}
	}
	synced := "never synced"
	if meta.LastSyncAt != nil && *meta.LastSyncAt != "" {
		at := *meta.LastSyncAt
		if len(at) > 16 {
			at = at[:16]
		}
		synced = "last sync " + strings.Replace(at, "T", " ", 1)
	}
	st.State = connectors.StateConnected
	st.Detail = credits + " · " + synced
	return st
}

// formatNumber matches Number.prototype.toLocaleString('en-US'): comma
// grouping and at most three fraction digits.
func formatNumber(f float64) string {
	s := strconv.FormatFloat(math.Abs(f), 'f', 3, 64)
	intPart, frac, _ := strings.Cut(s, ".")
	frac = strings.TrimRight(frac, "0")
	var b strings.Builder
	if f < 0 && s != "0.000" {
		b.WriteByte('-')
	}
	for i, r := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if frac != "" {
		b.WriteString("." + frac)
	}
	return b.String()
}

// ---- snapshot store ----------------------------------------------------------

// Store is Adscout's local snapshot directory (plain JSON). Everything
// downstream reads from here so credits are spent once per fact. A missing
// file is an empty store (never synced); a corrupt one is an error.
type Store struct {
	Dir string
}

// DefaultStore is ADSCOUT_STORE_DIR, else data/ad-intel under the working
// directory, as in FounderOS v1.
func DefaultStore() Store {
	if d := os.Getenv("ADSCOUT_STORE_DIR"); d != "" {
		return Store{Dir: d}
	}
	return Store{Dir: filepath.Join("data", "ad-intel")}
}

// StoreMeta is meta.json.
type StoreMeta struct {
	LastSyncAt    *string `json:"lastSyncAt"`
	LastSyncCalls int     `json:"lastSyncCalls"`
}

// readJSON reports found=false for a missing file.
func (s Store) readJSON(name string, into any) (bool, error) {
	raw, err := os.ReadFile(filepath.Join(s.Dir, name))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return false, fmt.Errorf("foreplay store %s: %w", name, err)
	}
	return true, nil
}

var unsafeChars = regexp.MustCompile(`[^A-Za-z0-9_-]`)

func brandFile(prefix, brandID string) string {
	return prefix + "-" + unsafeChars.ReplaceAllString(brandID, "_") + ".json"
}

// ReadUsage is the last synced credit meter; nil when never synced.
func (s Store) ReadUsage() (*Usage, error) {
	var u *Usage
	if _, err := s.readJSON("usage.json", &u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s Store) ReadMeta() (StoreMeta, error) {
	var m StoreMeta
	_, err := s.readJSON("meta.json", &m)
	return m, err
}

// ReadWatchlist is the synced Spyder watchlist.
func (s Store) ReadWatchlist() ([]SpyderBrand, error) {
	out := []SpyderBrand{}
	_, err := s.readJSON("watchlist.json", &out)
	return out, err
}

// ReadBrandAds is one brand's synced ads, the AdPilot ad library's source.
func (s Store) ReadBrandAds(brandID string) ([]Ad, error) {
	out := []Ad{}
	_, err := s.readJSON(brandFile("ads", brandID), &out)
	return out, err
}

// ReadAnalytics is one brand's accumulated daily series, oldest first.
func (s Store) ReadAnalytics(brandID string) ([]AnalyticsDay, error) {
	out := []AnalyticsDay{}
	_, err := s.readJSON(brandFile("analytics", brandID), &out)
	return out, err
}

func reason(err error) string {
	var ue *url.Error
	if errors.As(err, &ue) {
		if ue.Timeout() {
			return "request timed out"
		}
		return ue.Err.Error()
	}
	return err.Error()
}
