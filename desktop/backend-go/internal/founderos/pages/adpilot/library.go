package adpilot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/foreplay"
)

// ISO is JavaScript's Date.toISOString(): the timestamp shape every store
// file already carries.
func ISO(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

func jsNum(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

func strp(s string) *string { return &s }

// ---- facts over one ad (lib/foreplay/signals.ts, mine.ts) -------------------

const daySeconds = 86400

// DaysRunning is whole days running; a missing duration reads 0.
func DaysRunning(a foreplay.Ad) int {
	if a.RunningDuration == nil {
		return 0
	}
	return int(math.Floor(a.RunningDuration.Seconds / daySeconds))
}

// Transcript filler that isn't a hook: silence-only captures and stubs.
var junkHooks = regexp.MustCompile(`(?i)^(thanks for watching|you|\.+|music)\W*$`)
var sentenceEnd = regexp.MustCompile(`[.!?]`)

// HookLine is the first transcript sentence: the ad's hook, when it has one.
func HookLine(a foreplay.Ad) *string {
	first := ""
	if len(a.TimestampedTranscription) > 0 {
		first = strings.TrimSpace(a.TimestampedTranscription[0].Sentence)
	}
	line := first
	if line == "" && a.FullTranscription != nil {
		line = strings.TrimSpace(sentenceEnd.Split(*a.FullTranscription, 2)[0])
	}
	if line == "" || utf8.RuneCountInString(line) < 8 || junkHooks.MatchString(line) {
		return nil
	}
	return &line
}

var entities = strings.NewReplacer("&#039;", "'", "&#39;", "'", "&quot;", `"`, "&amp;", "&")
var brTag = regexp.MustCompile(`(?i)<br\s*/?>`)

// TextOpener is the first line of the ad copy, entities decoded.
func TextOpener(a foreplay.Ad) *string {
	if a.Description == nil {
		return nil
	}
	desc := brTag.ReplaceAllString(entities.Replace(*a.Description), "\n")
	for _, l := range strings.Split(desc, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			if utf8.RuneCountInString(l) >= 8 {
				return &l
			}
			return nil
		}
	}
	return nil
}

// BestHook prefers the spoken hook, then the text opener.
func BestHook(a foreplay.Ad) (hook *string, source *string) {
	if h := HookLine(a); h != nil {
		return h, strp("spoken")
	}
	if h := TextOpener(a); h != nil {
		return h, strp("text")
	}
	return nil, nil
}

// ---- the wall (lib/foreplay/wall.ts) ---------------------------------------

type TranscriptBeat struct {
	T float64 `json:"t"`
	S string  `json:"s"`
}

// WallAd is the flat, serializable ad every AdPilot card, the dossier and
// the saved swipe file render.
type WallAd struct {
	ID          string             `json:"id"`
	Brand       string             `json:"brand"`
	BrandID     *string            `json:"brandId"`
	Thumbnail   *string            `json:"thumbnail"`
	Video       *string            `json:"video"`
	Image       *string            `json:"image"`
	Format      *string            `json:"format"`
	Live        bool               `json:"live"`
	DaysRunning int                `json:"daysRunning"`
	Hook        *string            `json:"hook"`
	HookSource  *string            `json:"hookSource"`
	Transcript  []TranscriptBeat   `json:"transcript"`
	Drivers     map[string]float64 `json:"drivers"`
	CTAType     *string            `json:"ctaType"`
	LinkURL     *string            `json:"linkUrl"`
}

func ToWallAd(a foreplay.Ad, brandName string) WallAd {
	hook, src := BestHook(a)
	brand := brandName
	if brand == "" {
		brand = "unknown"
		if a.Name != nil {
			brand = *a.Name
		}
	}
	beats := []TranscriptBeat{}
	for _, l := range a.TimestampedTranscription {
		if s := strings.TrimSpace(l.Sentence); s != "" {
			beats = append(beats, TranscriptBeat{T: l.StartTime, S: s})
		}
	}
	drivers := a.EmotionalDrivers
	if drivers == nil {
		drivers = map[string]float64{}
	}
	return WallAd{
		ID: a.ID, Brand: brand, BrandID: a.BrandID, Thumbnail: a.Thumbnail, Video: a.Video, Image: a.Image,
		Format: a.DisplayFormat, Live: a.Live != nil && *a.Live, DaysRunning: DaysRunning(a),
		Hook: hook, HookSource: src, Transcript: beats, Drivers: drivers, CTAType: a.CTAType, LinkURL: a.LinkURL,
	}
}

// ---- signals (lib/foreplay/signals.ts) --------------------------------------

type Signal struct {
	Type      string  `json:"type"` // new_launch | winner | killed | velocity_spike | velocity_drop
	BrandID   string  `json:"brandId"`
	BrandName string  `json:"brandName"`
	AdID      *string `json:"adId,omitempty"`
	Message   string  `json:"message"`
	At        string  `json:"at"`
}

var WinnerThresholdDays = []int{21, 30, 60}

func truncRunes(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// DiffBrandAds compares one brand's previous and fresh ads.
func DiffBrandAds(brandID, brandName string, prev, next []foreplay.Ad, at string) []Signal {
	before := map[string]foreplay.Ad{}
	for _, a := range prev {
		before[a.ID] = a
	}
	live := func(a foreplay.Ad) bool { return a.Live != nil && *a.Live }
	out := []Signal{}
	for _, a := range next {
		days := DaysRunning(a)
		label := "ad"
		if h := HookLine(a); h != nil {
			label = `"` + truncRunes(*h, 80) + `"`
		} else if a.Headline != nil && strings.TrimSpace(*a.Headline) != "" {
			label = strings.TrimSpace(*a.Headline)
		} else if a.DisplayFormat != nil && strings.TrimSpace(*a.DisplayFormat) != "" {
			label = strings.TrimSpace(*a.DisplayFormat)
		}
		sig := func(typ, msg string) Signal {
			return Signal{Type: typ, BrandID: brandID, BrandName: brandName, AdID: strp(a.ID), At: at, Message: msg}
		}
		old, seen := before[a.ID]
		if !seen {
			// A brand added to the watchlist arrives with its whole back
			// catalog: only a young live ad is a launch.
			if live(a) && days <= 7 {
				out = append(out, sig("new_launch", fmt.Sprintf("%s launched %s", brandName, label)))
			}
			continue
		}
		daysBefore := DaysRunning(old)
		for _, th := range WinnerThresholdDays {
			if live(a) && daysBefore < th && days >= th {
				out = append(out, sig("winner", fmt.Sprintf("%s: %s crossed %d days and is still live", brandName, label, th)))
			}
		}
		if live(old) && a.Live != nil && !*a.Live {
			out = append(out, sig("killed", fmt.Sprintf("%s killed %s after %d days", brandName, label, days)))
		}
	}
	return out
}

// VelocitySignal: a >=30% move in active ads against the trailing 7-day
// mean (needs >=4 days of history and a base of at least 3).
func VelocitySignal(brandID, brandName string, series []foreplay.AnalyticsDay, at string) *Signal {
	sorted := append([]foreplay.AnalyticsDay(nil), series...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Date < sorted[j].Date })
	if len(sorted) < 4 {
		return nil
	}
	latest := sorted[len(sorted)-1]
	from := len(sorted) - 8
	if from < 0 {
		from = 0
	}
	window := sorted[from : len(sorted)-1]
	mean := 0.0
	for _, d := range window {
		mean += d.ActiveCount
	}
	mean /= float64(len(window))
	if mean < 3 {
		return nil
	}
	delta := (latest.ActiveCount - mean) / mean
	if math.Abs(delta) < 0.3 {
		return nil
	}
	pct := int(math.Floor(delta*100 + 0.5))
	typ, sign := "velocity_drop", ""
	if delta > 0 {
		typ = "velocity_spike"
	}
	if pct > 0 {
		sign = "+"
	}
	return &Signal{Type: typ, BrandID: brandID, BrandName: brandName, At: at,
		Message: fmt.Sprintf("%s active ads %s%d%% vs 7-day mean (%s live on %s)", brandName, sign, pct, jsNum(latest.ActiveCount), latest.Date)}
}

// Digest is one line per signal for run summaries.
func Digest(signals []Signal) string {
	if len(signals) == 0 {
		return "No changes across the watchlist."
	}
	msgs := make([]string, len(signals))
	for i, s := range signals {
		msgs[i] = s.Message
	}
	return strings.Join(msgs, " · ")
}

// ---- the store (lib/foreplay/store.ts, watchlist.ts, saved.ts) ------------

// Store is Adscout's snapshot directory, read and written. Reads of files
// that don't exist are empty (never synced); unreadable or corrupt files
// are errors so the page can say so.
type Store struct{ Dir string }

// DefaultStore is the Foreplay connector's store dir (ADSCOUT_STORE_DIR,
// else data/ad-intel), so both read the same snapshots.
func DefaultStore() Store { return Store{Dir: foreplay.DefaultStore().Dir} }

func (s Store) Foreplay() foreplay.Store { return foreplay.Store{Dir: s.Dir} }

var unsafeChars = regexp.MustCompile(`[^A-Za-z0-9_-]`)

func brandFile(prefix, id string) string {
	return prefix + "-" + unsafeChars.ReplaceAllString(id, "_") + ".json"
}

func (s Store) read(name string, into any) (bool, error) {
	raw, err := os.ReadFile(filepath.Join(s.Dir, name))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return false, fmt.Errorf("adscout store %s: %w", name, err)
	}
	return true, nil
}

func (s Store) write(name string, v any) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := filepath.Join(s.Dir, "."+name+".tmp")
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(s.Dir, name))
}

func (s Store) ReadBrandAds(id string) ([]foreplay.Ad, error) { return s.Foreplay().ReadBrandAds(id) }

// adJSON keeps every field the API returned (a spec bump never drops data
// on its way to the store), with the typed live flag laid over it.
func adJSON(a foreplay.Ad) (json.RawMessage, error) {
	if len(a.Raw) == 0 {
		return json.Marshal(a)
	}
	var m map[string]any
	if err := json.Unmarshal(a.Raw, &m); err != nil {
		return nil, err
	}
	if a.Live != nil {
		m["live"] = *a.Live
	} else {
		delete(m, "live")
	}
	return json.Marshal(m)
}

func (s Store) WriteBrandAds(id string, ads []foreplay.Ad) error {
	out := make([]json.RawMessage, 0, len(ads))
	for _, a := range ads {
		raw, err := adJSON(a)
		if err != nil {
			return err
		}
		out = append(out, raw)
	}
	return s.write(brandFile("ads", id), out)
}

func (s Store) ReadAnalytics(id string) ([]foreplay.AnalyticsDay, error) {
	return s.Foreplay().ReadAnalytics(id)
}

// WriteAnalytics merges by date so history outlives the API's window.
func (s Store) WriteAnalytics(id string, days []foreplay.AnalyticsDay) error {
	prev, err := s.ReadAnalytics(id)
	if err != nil {
		return err
	}
	byDate := map[string]foreplay.AnalyticsDay{}
	for _, d := range prev {
		byDate[d.Date] = d
	}
	for _, d := range days {
		byDate[d.Date] = d
	}
	out := make([]foreplay.AnalyticsDay, 0, len(byDate))
	for _, d := range byDate {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date < out[j].Date })
	return s.write(brandFile("analytics", id), out)
}

func (s Store) ReadSignals() ([]Signal, error) {
	out := []Signal{}
	_, err := s.read("signals.json", &out)
	return out, err
}

// AppendSignals prepends to the rolling log, capped.
func (s Store) AppendSignals(signals []Signal, limit int) error {
	if len(signals) == 0 {
		return nil
	}
	prev, err := s.ReadSignals()
	if err != nil {
		return err
	}
	all := append(append([]Signal{}, signals...), prev...)
	if len(all) > limit {
		all = all[:limit]
	}
	return s.write("signals.json", all)
}

func (s Store) WriteUsage(u foreplay.Usage) error     { return s.write("usage.json", u) }
func (s Store) WriteMeta(m foreplay.StoreMeta) error  { return s.write("meta.json", m) }
func (s Store) ReadUsage() (*foreplay.Usage, error)   { return s.Foreplay().ReadUsage() }
func (s Store) ReadMeta() (foreplay.StoreMeta, error) { return s.Foreplay().ReadMeta() }

// WatchEntry is one brand on Adscout's own watchlist.
type WatchEntry struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Domain  *string `json:"domain,omitempty"`
	Avatar  *string `json:"avatar,omitempty"`
	AddedAt string  `json:"addedAt"`
}

// ReadWatchEntries keeps the rows that validate and drops the rest.
func (s Store) ReadWatchEntries() ([]WatchEntry, error) {
	var rows []json.RawMessage
	if _, err := s.read("watchlist.json", &rows); err != nil {
		return nil, err
	}
	out := []WatchEntry{}
	for _, r := range rows {
		var e struct {
			ID      *string         `json:"id"`
			Name    *string         `json:"name"`
			Domain  json.RawMessage `json:"domain"`
			Avatar  json.RawMessage `json:"avatar"`
			AddedAt *string         `json:"addedAt"`
		}
		if json.Unmarshal(r, &e) != nil || e.ID == nil || *e.ID == "" || e.Name == nil || *e.Name == "" || e.AddedAt == nil {
			continue
		}
		w := WatchEntry{ID: *e.ID, Name: *e.Name, AddedAt: *e.AddedAt}
		if len(e.Domain) > 0 && json.Unmarshal(e.Domain, &w.Domain) != nil {
			continue
		}
		if len(e.Avatar) > 0 && json.Unmarshal(e.Avatar, &w.Avatar) != nil {
			continue
		}
		out = append(out, w)
	}
	return out, nil
}

func (s Store) WriteWatchEntries(list []WatchEntry) error { return s.write("watchlist.json", list) }

// AddWatchEntry adds by a known brand id; an id already present is a no-op.
func (s Store) AddWatchEntry(e WatchEntry, now time.Time) ([]WatchEntry, error) {
	list, err := s.ReadWatchEntries()
	if err != nil {
		return nil, err
	}
	for _, w := range list {
		if w.ID == e.ID {
			return list, nil
		}
	}
	e.AddedAt = ISO(now)
	list = append(list, e)
	return list, s.WriteWatchEntries(list)
}

func (s Store) RemoveWatchEntry(id string) ([]WatchEntry, error) {
	list, err := s.ReadWatchEntries()
	if err != nil {
		return nil, err
	}
	out := []WatchEntry{}
	for _, w := range list {
		if w.ID != id {
			out = append(out, w)
		}
	}
	return out, s.WriteWatchEntries(out)
}

// Wall is every stored ad of every watched brand, live first, longest
// running first, capped at limit.
func (s Store) Wall(limit int) ([]WallAd, error) {
	brands, err := s.ReadWatchEntries()
	if err != nil {
		return nil, err
	}
	out := []WallAd{}
	for _, b := range brands {
		ads, err := s.ReadBrandAds(b.ID)
		if err != nil {
			return nil, err
		}
		for _, a := range ads {
			out = append(out, ToWallAd(a, b.Name))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Live != out[j].Live {
			return out[i].Live
		}
		return out[i].DaysRunning > out[j].DaysRunning
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

// SavedAd is one swipe-file entry: a full WallAd snapshot, so it survives
// its brand leaving the watchlist.
type SavedAd struct {
	Ad      WallAd `json:"ad"`
	SavedAt string `json:"savedAt"`
}

const savedCap = 200

func (s Store) ReadSaved() ([]SavedAd, error) {
	var rows []json.RawMessage
	if _, err := s.read("saved.json", &rows); err != nil {
		return nil, err
	}
	out := []SavedAd{}
	for _, r := range rows {
		var e struct {
			Ad      json.RawMessage `json:"ad"`
			SavedAt *string         `json:"savedAt"`
		}
		if json.Unmarshal(r, &e) != nil || e.SavedAt == nil || len(e.Ad) == 0 || e.Ad[0] != '{' {
			continue
		}
		var w WallAd
		if json.Unmarshal(e.Ad, &w) != nil {
			continue
		}
		out = append(out, SavedAd{Ad: w, SavedAt: *e.SavedAt})
	}
	return out, nil
}

func (s Store) writeSaved(list []SavedAd) error {
	if len(list) > savedCap {
		list = list[:savedCap]
	}
	return s.write("saved.json", list)
}

// SaveAd puts the ad at the top of the swipe file; saving twice is a no-op.
func (s Store) SaveAd(a WallAd, now time.Time) ([]SavedAd, error) {
	list, err := s.ReadSaved()
	if err != nil {
		return nil, err
	}
	for _, x := range list {
		if x.Ad.ID == a.ID {
			return list, nil
		}
	}
	next := append([]SavedAd{{Ad: a, SavedAt: ISO(now)}}, list...)
	if len(next) > savedCap {
		next = next[:savedCap]
	}
	return next, s.writeSaved(next)
}

func (s Store) UnsaveAd(id string) ([]SavedAd, error) {
	list, err := s.ReadSaved()
	if err != nil {
		return nil, err
	}
	out := []SavedAd{}
	for _, x := range list {
		if x.Ad.ID != id {
			out = append(out, x)
		}
	}
	return out, s.writeSaved(out)
}

// ---- Foreplay-spending cycles (sync.ts, mine.ts, watchlist.ts) ------------

// API is the slice of the Foreplay client the user-triggered cycles use.
// *foreplay.Client satisfies it; tests pass a fake, so no test spends.
type API interface {
	AdsByBrandID(ctx context.Context, brandIDs string, f foreplay.AdFilters) ([]foreplay.Ad, error)
	BrandAnalytics(ctx context.Context, id, startDate, endDate string) ([]foreplay.AnalyticsDay, error)
	Usage(ctx context.Context) (*foreplay.Usage, error)
	BrandsByDomain(ctx context.Context, domain string, limit int) ([]foreplay.SpyderBrand, error)
	DiscoveryAds(ctx context.Context, query string, f foreplay.AdFilters) ([]foreplay.Ad, error)
	CallCount() int64
}

type SyncResult struct {
	Brands           int      `json:"brands"`
	AdsSeen          int      `json:"adsSeen"`
	Signals          []Signal `json:"signals"`
	Digest           string   `json:"digest"`
	APICalls         int64    `json:"apiCalls"`
	RemainingCredits *float64 `json:"remainingCredits"`
}

// brandPage is the API's cap on brand-ads pages: newest + oldest cover <=150.
const brandPage = 75

// RunSyncCycle is one sync over the local watchlist: recent and
// longest-running live ads per brand, analytics, the signal diff, usage.
func RunSyncCycle(ctx context.Context, api API, s Store, now time.Time) (*SyncResult, error) {
	at := ISO(now)
	start := api.CallCount()
	brands, err := s.ReadWatchEntries()
	if err != nil {
		return nil, err
	}
	live, limit := true, brandPage
	all := []Signal{}
	seen := 0
	for _, b := range brands {
		prev, err := s.ReadBrandAds(b.ID)
		if err != nil {
			return nil, err
		}
		newest, err := api.AdsByBrandID(ctx, b.ID, foreplay.AdFilters{Live: &live, Limit: &limit})
		if err != nil {
			return nil, err
		}
		oldest, err := api.AdsByBrandID(ctx, b.ID, foreplay.AdFilters{Live: &live, Limit: &limit, Order: "oldest"})
		if err != nil {
			return nil, err
		}
		var order []string
		liveNow := map[string]foreplay.Ad{}
		for _, a := range append(append([]foreplay.Ad{}, newest...), oldest...) {
			if _, ok := liveNow[a.ID]; !ok {
				order = append(order, a.ID)
			}
			liveNow[a.ID] = a
		}
		next := make([]foreplay.Ad, 0, len(order))
		for _, id := range order {
			next = append(next, liveNow[id])
		}
		seen += len(next)

		// A previously-live ad missing from an untruncated live pull has
		// been turned off: surface it so the killed signal fires.
		if len(newest) < brandPage || len(oldest) < brandPage || len(next) < 2*brandPage-10 {
			off := false
			for _, old := range prev {
				if _, still := liveNow[old.ID]; old.Live != nil && *old.Live && !still {
					old.Live = &off
					next = append(next, old)
				}
			}
		}

		// Dead ads stay in the store so longevity history isn't lost.
		mergedOrder := []string{}
		merged := map[string]foreplay.Ad{}
		for _, a := range append(append([]foreplay.Ad{}, prev...), next...) {
			if _, ok := merged[a.ID]; !ok {
				mergedOrder = append(mergedOrder, a.ID)
			}
			merged[a.ID] = a
		}
		keep := make([]foreplay.Ad, 0, len(mergedOrder))
		for _, id := range mergedOrder {
			keep = append(keep, merged[id])
		}
		if err := s.WriteBrandAds(b.ID, keep); err != nil {
			return nil, err
		}
		all = append(all, DiffBrandAds(b.ID, b.Name, prev, next, at)...)

		series, err := api.BrandAnalytics(ctx, b.ID, "", "")
		if err == nil && len(series) > 0 {
			if err := s.WriteAnalytics(b.ID, series); err != nil {
				return nil, err
			}
			hist, err := s.ReadAnalytics(b.ID)
			if err != nil {
				return nil, err
			}
			if v := VelocitySignal(b.ID, b.Name, hist, at); v != nil {
				all = append(all, *v)
			}
		}
	}
	if err := s.AppendSignals(all, 500); err != nil {
		return nil, err
	}
	res := &SyncResult{Brands: len(brands), AdsSeen: seen, Signals: all, Digest: Digest(all)}
	// Usage is bookkeeping: a failed read never fails the cycle.
	if u, err := api.Usage(ctx); err == nil && u != nil {
		if err := s.WriteUsage(*u); err != nil {
			return nil, err
		}
		rc := u.RemainingCredits
		res.RemainingCredits = &rc
	}
	res.APICalls = api.CallCount() - start
	if err := s.WriteMeta(foreplay.StoreMeta{LastSyncAt: &at, LastSyncCalls: int(res.APICalls)}); err != nil {
		return nil, err
	}
	return res, nil
}

// NaiveProbes is the deterministic probe expansion when no model is there.
func NaiveProbes(concept string) []string {
	clean := strings.TrimSpace(concept)
	var words []string
	for _, w := range strings.Fields(clean) {
		if utf8.RuneCountInString(w) > 2 {
			words = append(words, w)
		}
	}
	probes := []string{clean}
	add := func(p string) {
		for _, x := range probes {
			if x == p {
				return
			}
		}
		probes = append(probes, p)
	}
	if len(words) > 2 {
		add(strings.Join(words[:3], " "))
	}
	if len(words) > 3 {
		add(strings.Join(words[len(words)-3:], " "))
	}
	if len(probes) > 4 {
		probes = probes[:4]
	}
	return probes
}

type MineOpts struct {
	MinDays    int
	PerProbe   int
	MaxWinners int
	Format     string
}

type ProbeStat struct {
	Query   string `json:"query"`
	Results int    `json:"results"`
}

type MinedAd struct {
	Ad          foreplay.Ad
	DaysRunning int
	Probes      []string
}

type MineResult struct {
	Concept  string
	Probes   []ProbeStat
	Pooled   int
	Winners  []MinedAd
	APICalls int64
}

// MineConcept pools Discovery results across probes, dedupes and ranks by
// longevity. One API call per probe; a failed probe counts as zero.
func MineConcept(ctx context.Context, api API, concept string, probes []string, o MineOpts) MineResult {
	if o.MinDays == 0 {
		o.MinDays = 21
	}
	if o.PerProbe == 0 {
		o.PerProbe = 15
	}
	if o.MaxWinners == 0 {
		o.MaxWinners = 24
	}
	start := api.CallCount()
	live := true
	var order []string
	pool := map[string]*MinedAd{}
	stats := []ProbeStat{}
	for _, q := range probes {
		minDays, per := o.MinDays, o.PerProbe
		ads, err := api.DiscoveryAds(ctx, q, foreplay.AdFilters{RunningDurationMinDays: &minDays, Live: &live, Limit: &per, DisplayFormat: o.Format})
		if err != nil {
			ads = nil
		}
		stats = append(stats, ProbeStat{Query: q, Results: len(ads)})
		for _, a := range ads {
			if hit, ok := pool[a.ID]; ok {
				hit.Probes = append(hit.Probes, q)
				continue
			}
			order = append(order, a.ID)
			pool[a.ID] = &MinedAd{Ad: a, DaysRunning: DaysRunning(a), Probes: []string{q}}
		}
	}
	winners := make([]MinedAd, 0, len(order))
	for _, id := range order {
		winners = append(winners, *pool[id])
	}
	sort.SliceStable(winners, func(i, j int) bool { return winners[i].DaysRunning > winners[j].DaysRunning })
	if len(winners) > o.MaxWinners {
		winners = winners[:o.MaxWinners]
	}
	return MineResult{Concept: concept, Probes: stats, Pooled: len(order), Winners: winners, APICalls: api.CallCount() - start}
}

var schemePrefix = regexp.MustCompile(`^https?://`)
var pathSuffix = regexp.MustCompile(`/.*$`)

// AddWatchDomain resolves a website to its biggest Foreplay brand page (one
// API call) and watches it; nil when Foreplay knows no page for it.
func AddWatchDomain(ctx context.Context, api API, s Store, domain string, now time.Time) (*WatchEntry, error) {
	clean := pathSuffix.ReplaceAllString(schemePrefix.ReplaceAllString(strings.ToLower(strings.TrimSpace(domain)), ""), "")
	pages, err := api.BrandsByDomain(ctx, clean, 0)
	if err != nil {
		return nil, err
	}
	if len(pages) == 0 {
		return nil, nil
	}
	count := func(b foreplay.SpyderBrand) int {
		if b.AdsCount == nil {
			return 0
		}
		return *b.AdsCount
	}
	best := pages[0]
	for _, p := range pages[1:] {
		if count(p) > count(best) {
			best = p
		}
	}
	e := WatchEntry{ID: best.ID, Name: best.Name, Domain: &clean, Avatar: best.Avatar}
	if _, err := s.AddWatchEntry(e, now); err != nil {
		return nil, err
	}
	e.AddedAt = ISO(now)
	return &e, nil
}

// ParseWallAd validates a client-sent WallAd snapshot (the saved route's
// WallAdShape): every key present, the right types, nullables null or typed.
func ParseWallAd(raw []byte) (WallAd, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil || m == nil {
		return WallAd{}, errors.New("a full ad snapshot is required")
	}
	isNull := func(k string) bool { return string(m[k]) == "null" }
	kind := map[string]string{
		"id": "string", "brand": "string", "brandId": "string?", "thumbnail": "string?", "video": "string?",
		"image": "string?", "format": "string?", "live": "bool", "daysRunning": "number", "hook": "string?",
		"hookSource": "string?", "transcript": "array", "drivers": "object", "ctaType": "string?", "linkUrl": "string?",
	}
	for k, want := range kind {
		v, ok := m[k]
		if !ok {
			return WallAd{}, fmt.Errorf("a full ad snapshot is required (missing %s)", k)
		}
		if strings.HasSuffix(want, "?") {
			if isNull(k) {
				continue
			}
			want = strings.TrimSuffix(want, "?")
		}
		var probe any
		if json.Unmarshal(v, &probe) != nil {
			return WallAd{}, fmt.Errorf("bad %s", k)
		}
		okType := false
		switch probe.(type) {
		case string:
			okType = want == "string"
		case bool:
			okType = want == "bool"
		case float64:
			okType = want == "number"
		case []any:
			okType = want == "array"
		case map[string]any:
			okType = want == "object"
		}
		if !okType {
			return WallAd{}, fmt.Errorf("a full ad snapshot is required (%s is not a %s)", k, want)
		}
	}
	var w WallAd
	if err := json.Unmarshal(raw, &w); err != nil {
		return WallAd{}, fmt.Errorf("a full ad snapshot is required: %w", err)
	}
	if w.ID == "" {
		return WallAd{}, errors.New("a full ad snapshot is required (empty id)")
	}
	if w.HookSource != nil && *w.HookSource != "spoken" && *w.HookSource != "text" {
		return WallAd{}, errors.New("hookSource must be spoken, text or null")
	}
	return w, nil
}
