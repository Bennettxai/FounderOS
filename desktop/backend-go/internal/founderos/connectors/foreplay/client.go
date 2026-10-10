package foreplay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
)

// Client reads the Foreplay public API (lib/foreplay/client.ts). The API is
// read-only and every call burns account credits (10k/month plan), so the
// client counts its calls and callers cache what it returns in the Store.
type Client struct {
	key     string
	baseURL string
	http    *http.Client
	calls   atomic.Int64
}

// CallCount is the number of API requests made: the credit ledger.
func (c *Client) CallCount() int64 { return c.calls.Load() }

// Error is a failed call; Status is the HTTP status when there was one.
type Error struct {
	Path   string
	Msg    string
	Status int
}

func (e *Error) Error() string { return "Foreplay " + e.Path + ": " + e.Msg }

// ---- shapes ----------------------------------------------------------------

type TranscriptLine struct {
	StartTime float64 `json:"startTime"`
	EndTime   float64 `json:"endTime"`
	Sentence  string  `json:"sentence"`
}

type RunningDuration struct {
	Seconds float64 `json:"seconds"`
}

// Ad is one Foreplay ad. The fields Adscout depends on are typed; Raw keeps
// the whole object so a spec bump never drops data on its way to the store.
type Ad struct {
	ID                       string             `json:"id"`
	AdID                     *string            `json:"ad_id"`
	Name                     *string            `json:"name"` // publishing page name
	BrandID                  *string            `json:"brand_id"`
	Description              *string            `json:"description"`
	Headline                 *string            `json:"headline"`
	FullTranscription        *string            `json:"full_transcription"`
	TimestampedTranscription []TranscriptLine   `json:"timestamped_transcription"`
	EmotionalDrivers         map[string]float64 `json:"emotional_drivers"`
	DisplayFormat            *string            `json:"display_format"`
	PublisherPlatform        []string           `json:"publisher_platform"`
	Live                     *bool              `json:"live"`
	StartedRunning           *float64           `json:"started_running"` // epoch ms
	RunningDuration          *RunningDuration   `json:"running_duration"`
	Thumbnail                *string            `json:"thumbnail"`
	Video                    *string            `json:"video"`
	Image                    *string            `json:"image"`
	CTAType                  *string            `json:"cta_type"`
	CTATitle                 *string            `json:"cta_title"`
	LinkURL                  *string            `json:"link_url"`
	ProductCategory          *string            `json:"product_category"`
	MarketTarget             *string            `json:"market_target"`
	Raw                      json.RawMessage    `json:"-"`
}

func (a *Ad) UnmarshalJSON(b []byte) error {
	if err := requireKeys(b, "id"); err != nil {
		return err
	}
	type plain Ad
	var p plain
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	*a = Ad(p)
	a.Raw = append(json.RawMessage(nil), b...)
	return nil
}

type SpyderBrand struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	AdLibraryID    *string `json:"ad_library_id"`
	Avatar         *string `json:"avatar"`
	Category       *string `json:"category"`
	Websites       []any   `json:"websites"`
	AdsCount       *int    `json:"ads_count"`
	ActiveAdsCount *int    `json:"active_ads_count"`
}

func (s *SpyderBrand) UnmarshalJSON(b []byte) error {
	if err := requireKeys(b, "id", "name"); err != nil {
		return err
	}
	type plain SpyderBrand
	return json.Unmarshal(b, (*plain)(s))
}

// AnalyticsDay is one day of active/inactive totals and format split.
type AnalyticsDay struct {
	Date          string   `json:"date"`
	ActiveCount   float64  `json:"active_count"`
	InactiveCount *float64 `json:"inactive_count"`
	Video         *float64 `json:"video"`
	Image         *float64 `json:"image"`
	Carousel      *float64 `json:"carousel"`
}

func (d *AnalyticsDay) UnmarshalJSON(b []byte) error {
	if err := requireKeys(b, "date", "active_count"); err != nil {
		return err
	}
	type plain AnalyticsDay
	return json.Unmarshal(b, (*plain)(d))
}

type UsageUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type Usage struct {
	StartDate        string     `json:"start_date,omitempty"`
	EndDate          string     `json:"end_date,omitempty"`
	TotalCredits     float64    `json:"total_credits"`
	RemainingCredits float64    `json:"remaining_credits"`
	User             *UsageUser `json:"user,omitempty"`
}

func (u *Usage) UnmarshalJSON(b []byte) error {
	if err := requireKeys(b, "total_credits", "remaining_credits"); err != nil {
		return err
	}
	type plain Usage
	return json.Unmarshal(b, (*plain)(u))
}

// requireKeys fails when b is not an object or a required key is absent or
// null, the way the TS Zod schemas fail.
func requireKeys(b []byte, keys ...string) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil || m == nil {
		return fmt.Errorf("expected an object")
	}
	for _, k := range keys {
		if v, ok := m[k]; !ok || string(v) == "null" {
			return fmt.Errorf("missing required field %q", k)
		}
	}
	return nil
}

// AdFilters are the optional ad-listing filters; zero values are omitted.
type AdFilters struct {
	StartDate              string
	EndDate                string
	Live                   *bool
	DisplayFormat          string
	RunningDurationMinDays *int
	Limit                  *int
	Order                  string
}

func (f AdFilters) apply(q url.Values) {
	set := func(k, v string) {
		if v != "" {
			q.Set(k, v)
		}
	}
	set("start_date", f.StartDate)
	set("end_date", f.EndDate)
	if f.Live != nil {
		q.Set("live", strconv.FormatBool(*f.Live))
	}
	set("display_format", f.DisplayFormat)
	if f.RunningDurationMinDays != nil {
		q.Set("running_duration_min_days", strconv.Itoa(*f.RunningDurationMinDays))
	}
	if f.Limit != nil {
		q.Set("limit", strconv.Itoa(*f.Limit))
	}
	set("order", f.Order)
}

// ---- requests --------------------------------------------------------------

// request GETs path and returns the envelope's data.
func (c *Client) request(ctx context.Context, path string, q url.Values) (json.RawMessage, error) {
	u := c.baseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.key)
	c.calls.Add(1)
	res, err := c.http.Do(req)
	if err != nil {
		return nil, &Error{Path: path, Msg: reason(err)}
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	var obj map[string]json.RawMessage
	isObject := json.Unmarshal(raw, &obj) == nil && obj != nil
	if res.StatusCode < 200 || res.StatusCode > 299 {
		msg := fmt.Sprintf("HTTP %d", res.StatusCode)
		if m, ok := obj["message"]; ok {
			var s string
			if json.Unmarshal(m, &s) == nil {
				msg = s
			} else {
				msg = string(m)
			}
		}
		return nil, &Error{Path: path, Msg: msg, Status: res.StatusCode}
	}
	if !isObject {
		return nil, &Error{Path: path, Msg: "unrecognized response shape"}
	}
	return obj["data"], nil
}

func decode[T any](path string, data json.RawMessage) (T, error) {
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		return out, &Error{Path: path, Msg: "unexpected data: " + err.Error()}
	}
	return out, nil
}

func list[T any](ctx context.Context, c *Client, path string, q url.Values) ([]T, error) {
	data, err := c.request(ctx, path, q)
	if err != nil {
		return nil, err
	}
	out, err := decode[[]T](path, data)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, &Error{Path: path, Msg: "unexpected data: expected a list"}
	}
	return out, nil
}

func (c *Client) Usage(ctx context.Context) (*Usage, error) {
	data, err := c.request(ctx, "/api/usage", nil)
	if err != nil {
		return nil, err
	}
	u, err := decode[Usage]("/api/usage", data)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// SpyderBrands is the Spyder watchlist: the brands tracked in the Foreplay
// app, paged 10 at a time (the API maximum) up to 100.
func (c *Client) SpyderBrands(ctx context.Context) ([]SpyderBrand, error) {
	var out []SpyderBrand
	for offset := 0; offset < 100; offset += 10 {
		page, err := list[SpyderBrand](ctx, c, "/api/spyder/brands", url.Values{"limit": {"10"}, "offset": {strconv.Itoa(offset)}})
		if err != nil {
			return nil, err
		}
		out = append(out, page...)
		if len(page) < 10 {
			break
		}
	}
	return out, nil
}

func (c *Client) SpyderBrandAds(ctx context.Context, brandID string, f AdFilters) ([]Ad, error) {
	q := url.Values{"brand_id": {brandID}}
	f.apply(q)
	return list[Ad](ctx, c, "/api/spyder/brand/ads", q)
}

// BrandAnalytics is the daily active/inactive and format-split series.
// Empty dates are omitted.
func (c *Client) BrandAnalytics(ctx context.Context, id, startDate, endDate string) ([]AnalyticsDay, error) {
	q := url.Values{"id": {id}}
	AdFilters{StartDate: startDate, EndDate: endDate}.apply(q)
	return list[AnalyticsDay](ctx, c, "/api/brand/analytics", q)
}

// BrandsByDomain lists every brand page publishing for a domain; limit <= 0
// means the TS default of 10.
func (c *Client) BrandsByDomain(ctx context.Context, domain string, limit int) ([]SpyderBrand, error) {
	if limit <= 0 {
		limit = 10
	}
	return list[SpyderBrand](ctx, c, "/api/brand/getBrandsByDomain", url.Values{"domain": {domain}, "limit": {strconv.Itoa(limit)}})
}

// AdsByBrandID takes a comma-separated list of brand ids.
func (c *Client) AdsByBrandID(ctx context.Context, brandIDs string, f AdFilters) ([]Ad, error) {
	q := url.Values{"brand_ids": {brandIDs}}
	f.apply(q)
	return list[Ad](ctx, c, "/api/brand/getAdsByBrandId", q)
}

// DiscoveryAds searches the ~100M-ad Discovery database. Explicit spend:
// call it only on a user action.
func (c *Client) DiscoveryAds(ctx context.Context, query string, f AdFilters) ([]Ad, error) {
	q := url.Values{"query": {query}}
	f.apply(q)
	return list[Ad](ctx, c, "/api/discovery/ads", q)
}

func (c *Client) AdDetails(ctx context.Context, adID string) (*Ad, error) {
	data, err := c.request(ctx, "/api/ad", url.Values{"ad_id": {adID}})
	if err != nil {
		return nil, err
	}
	a, err := decode[Ad]("/api/ad", data)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// AdDuplicates lists ads reusing the same creative; group growth is the
// scaling signal.
func (c *Client) AdDuplicates(ctx context.Context, adID string) ([]Ad, error) {
	return list[Ad](ctx, c, "/api/ad/duplicates/"+url.PathEscape(adID), nil)
}
