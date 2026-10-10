package social

import (
	"sort"
	"strings"
)

// Platforms are the five tracked accounts (SocialPlatformSchema), in order.
var Platforms = []string{"instagram", "tiktok", "twitter", "youtube", "linkedin"}

var Labels = map[string]string{
	"instagram": "Instagram",
	"tiktok":    "TikTok",
	"twitter":   "X",
	"youtube":   "YouTube",
	"linkedin":  "LinkedIn",
}

// Pop-out chart colors (lib/social.ts PLATFORM_COLORS).
var Colors = map[string]string{
	"instagram": "#e1306c",
	"tiktok":    "#25f4ee",
	"twitter":   "#f2f2f2",
	"youtube":   "#ff4d4d",
	"linkedin":  "#0a85c2",
}

const (
	EmailColor = "#8b7cff"
	AllColor   = "#3df08c"
	DMColor    = "#5ec9f8"
)

// Tracked reports whether p is one of the five platforms.
func Tracked(p string) bool {
	for _, x := range Platforms {
		if x == p {
			return true
		}
	}
	return false
}

// Label is PLATFORM_LABELS with the page's capitalising fallback.
func Label(p string) string {
	if l, ok := Labels[p]; ok {
		return l
	}
	if p == "" {
		return p
	}
	return strings.ToUpper(p[:1]) + p[1:]
}

type Account struct {
	Platform string  `json:"platform"`
	Handle   string  `json:"handle"`
	URL      *string `json:"url"`
	Order    int     `json:"order"`
}

type Snapshot struct {
	Platform   string `json:"platform"`
	CapturedAt string `json:"capturedAt"`
	Followers  int    `json:"followers"`
	Source     string `json:"source"`
}

type DM struct {
	Platform  string `json:"platform"`
	Count     int    `json:"count"`
	UpdatedAt string `json:"updatedAt"`
}

type DMSnapshot struct {
	Platform   string `json:"platform"`
	CapturedAt string `json:"capturedAt"`
	Count      int    `json:"count"`
	Source     string `json:"source"`
}

type DMMessage struct {
	ID           string  `json:"id"`
	Platform     string  `json:"platform"`
	SubscriberID string  `json:"subscriberId"`
	Name         string  `json:"name"`
	Handle       *string `json:"handle"`
	Text         string  `json:"text"`
	Direction    string  `json:"direction"`
	Tag          *string `json:"tag"`
	TS           string  `json:"ts"`
	Source       string  `json:"source"`
}

type EmailSnapshot struct {
	CapturedAt    string `json:"capturedAt"`
	Subscribers   int    `json:"subscribers"`
	Source        string `json:"source"`
	PublicationID string `json:"publicationId"`
	Metric        string `json:"metric"`
	Quality       string `json:"quality"`
}

type Post struct {
	ID           string   `json:"id"`
	Caption      string   `json:"caption"`
	MediaURL     *string  `json:"mediaUrl"`
	Platforms    []string `json:"platforms"`
	Status       string   `json:"status"`
	ScheduledFor *string  `json:"scheduledFor"`
	CreatedAt    string   `json:"createdAt"`
}

// Data is everything the social pages read from Postgres, in the repo's
// orders: accounts by order, snapshots oldest first per platform, DM
// messages newest first, email snapshots oldest first (quality 'ok' only),
// posts newest first.
type Data struct {
	Accounts    []Account
	Snapshots   map[string][]Snapshot
	DMs         []DM
	DMSnapshots []DMSnapshot
	DMMessages  []DMMessage
	Email       []EmailSnapshot
	Posts       []Post
}

func (x *Data) snaps(p string) []Snapshot {
	if x.Snapshots == nil {
		return nil
	}
	return x.Snapshots[p]
}

func toPoints(s []Snapshot) []GrowthPoint {
	out := make([]GrowthPoint, 0, len(s))
	for _, v := range s {
		out = append(out, GrowthPoint{CapturedAt: v.CapturedAt, Value: float64(v.Followers)})
	}
	return out
}

// SnapshotGrowth is growthFor(snapshots).
func SnapshotGrowth(s []Snapshot) Growth {
	p := toPoints(s)
	return Growth{D7: GrowthOver(p, 7, 0), D30: GrowthOver(p, 30, 0), D60: GrowthOver(p, 60, 0), AllTime: GrowthAllTime(p)}
}

// LiveAccount is one platform's live (or config) follower reading.
type LiveAccount struct {
	Platform  string
	Handle    string
	Followers *float64
}

// SyncRows is syncSocialSnapshots: today's snapshot for every tracked
// platform that has a follower count; others are skipped.
func SyncRows(accounts []LiveAccount, today, source string) []Snapshot {
	var out []Snapshot
	for _, a := range accounts {
		if !Tracked(a.Platform) || a.Followers == nil {
			continue
		}
		out = append(out, Snapshot{Platform: a.Platform, CapturedAt: today, Followers: int(*a.Followers + 0.5), Source: source})
	}
	return out
}

// ---- dashboard --------------------------------------------------------------

type FollowerPoint struct {
	Date      string `json:"date"`
	Followers int    `json:"followers"`
}

type PlatformStats struct {
	Platform  string          `json:"platform"`
	Handle    string          `json:"handle"`
	URL       *string         `json:"url"`
	Followers *int            `json:"followers"`
	Growth    Growth          `json:"growth"`
	Series    []FollowerPoint `json:"series"`
}

type Dashboard struct {
	TotalFollowers int             `json:"totalFollowers"`
	AsOf           *string         `json:"asOf"`
	Platforms      []PlatformStats `json:"platforms"`
}

func last90(s []Snapshot) []FollowerPoint {
	if len(s) > 90 {
		s = s[len(s)-90:]
	}
	out := make([]FollowerPoint, 0, len(s))
	for _, v := range s {
		out = append(out, FollowerPoint{Date: v.CapturedAt, Followers: v.Followers})
	}
	return out
}

// BuildDashboard is buildSocialDashboard.
func BuildDashboard(x *Data) Dashboard {
	d := Dashboard{Platforms: []PlatformStats{}}
	for _, a := range x.Accounts {
		s := x.snaps(a.Platform)
		ps := PlatformStats{Platform: a.Platform, Handle: a.Handle, URL: a.URL, Growth: SnapshotGrowth(s), Series: last90(s)}
		if len(s) > 0 {
			f := s[len(s)-1].Followers
			ps.Followers = &f
		}
		d.Platforms = append(d.Platforms, ps)
	}
	var asOf string
	for _, p := range Platforms {
		s := x.snaps(p)
		if len(s) == 0 {
			continue
		}
		latest := s[len(s)-1]
		d.TotalFollowers += latest.Followers
		if latest.CapturedAt > asOf {
			asOf = latest.CapturedAt
		}
	}
	if asOf != "" {
		d.AsOf = &asOf
	}
	return d
}

// TotalDMs sums the per-platform DM counts.
func TotalDMs(x *Data) int {
	n := 0
	for _, d := range x.DMs {
		n += d.Count
	}
	return n
}

// ---- email list -------------------------------------------------------------

// MaxBaselineLagDays bounds the email list's growth windows (FOS-659).
const MaxBaselineLagDays = 7

// EmailSnapshots is the email list's history as FounderOS v1 reads it
// (lib/email-list.ts): every snapshot, seeded Beehiiv rows included, until a
// live reading lands on top of them.
func EmailSnapshots(x *Data) []EmailSnapshot {
	return x.Email
}

type SubscriberPoint struct {
	Date        string `json:"date"`
	Subscribers int    `json:"subscribers"`
}

type EmailList struct {
	Subscribers *int              `json:"subscribers"`
	AsOf        *string           `json:"asOf"`
	Growth      Growth            `json:"growth"`
	Series      []SubscriberPoint `json:"series"`
}

func emailPoints(s []EmailSnapshot) []GrowthPoint {
	out := make([]GrowthPoint, 0, len(s))
	for _, v := range s {
		out = append(out, GrowthPoint{CapturedAt: v.CapturedAt, Value: float64(v.Subscribers)})
	}
	return out
}

// BuildEmailList is buildEmailList.
func BuildEmailList(x *Data) EmailList {
	snaps := EmailSnapshots(x)
	p := emailPoints(snaps)
	e := EmailList{
		Growth: Growth{
			D7:      GrowthOver(p, 7, MaxBaselineLagDays),
			D30:     GrowthOver(p, 30, MaxBaselineLagDays),
			D60:     GrowthOver(p, 60, MaxBaselineLagDays),
			AllTime: GrowthAllTime(p),
		},
		Series: []SubscriberPoint{},
	}
	if n := len(snaps); n > 0 {
		latest := snaps[n-1]
		e.Subscribers, e.AsOf = &latest.Subscribers, &latest.CapturedAt
	}
	tail := snaps
	if len(tail) > 90 {
		tail = tail[len(tail)-90:]
	}
	for _, s := range tail {
		e.Series = append(e.Series, SubscriberPoint{Date: s.CapturedAt, Subscribers: s.Subscribers})
	}
	return e
}

// ---- audience ---------------------------------------------------------------

type SeriesPoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type LabelledSeries struct {
	Key    string        `json:"key"`
	Label  string        `json:"label"`
	Color  string        `json:"color"`
	Points []SeriesPoint `json:"points"`
}

func audienceChannels(x *Data) [][]GrowthPoint {
	var out [][]GrowthPoint
	for _, a := range x.Accounts {
		out = append(out, toPoints(x.snaps(a.Platform)))
	}
	return append(out, emailPoints(EmailSnapshots(x)))
}

func seriesPoints(p []GrowthPoint) []SeriesPoint {
	out := make([]SeriesPoint, 0, len(p))
	for _, v := range p {
		out = append(out, SeriesPoint{Date: v.CapturedAt, Value: v.Value})
	}
	return out
}

// AudienceGrowthPct aggregates growth across channels that reach back the
// whole window; nil when none qualifies. days <= 0 means all time.
func AudienceGrowthPct(x *Data, days int) *float64 {
	var cur, base float64
	q := 0
	for _, s := range audienceChannels(x) {
		var d *Delta
		if days <= 0 {
			if len(s) >= 2 && s[0].Value != 0 {
				d = &Delta{Current: s[len(s)-1].Value, Baseline: s[0].Value}
			}
		} else {
			d = WindowDelta(s, days, 0)
		}
		if d == nil {
			continue
		}
		cur += d.Current
		base += d.Baseline
		q++
	}
	if q == 0 || base == 0 {
		return nil
	}
	v := (cur - base) / base * 100
	return &v
}

func AudienceGrowth(x *Data) Growth {
	return Growth{D7: AudienceGrowthPct(x, 7), D30: AudienceGrowthPct(x, 30), D60: AudienceGrowthPct(x, 60), AllTime: AudienceGrowthPct(x, 0)}
}

// AudienceTotal sums each channel's latest value.
func AudienceTotal(x *Data) int {
	n := 0.0
	for _, s := range audienceChannels(x) {
		if len(s) > 0 {
			n += s[len(s)-1].Value
		}
	}
	return int(n)
}

// AudienceSeries is the "All audience" series and the per-channel ones.
func AudienceSeries(x *Data) (LabelledSeries, []LabelledSeries) {
	var channels []LabelledSeries
	for _, a := range x.Accounts {
		p := seriesPoints(toPoints(x.snaps(a.Platform)))
		if len(p) > 0 {
			channels = append(channels, LabelledSeries{Key: a.Platform, Label: Label(a.Platform), Color: Colors[a.Platform], Points: p})
		}
	}
	if p := seriesPoints(emailPoints(EmailSnapshots(x))); len(p) > 0 {
		channels = append(channels, LabelledSeries{Key: "email", Label: "Email List", Color: EmailColor, Points: p})
	}
	all := LabelledSeries{Key: "all", Label: "All audience", Color: AllColor, Points: seriesPoints(MergeSeriesSum(audienceChannels(x)))}
	return all, channels
}

// ---- DMs ----------------------------------------------------------------------

func dmPoints(x *Data) []GrowthPoint {
	by := map[string][]GrowthPoint{}
	var order []string
	for _, s := range x.DMSnapshots {
		if _, ok := by[s.Platform]; !ok {
			order = append(order, s.Platform)
		}
		by[s.Platform] = append(by[s.Platform], GrowthPoint{CapturedAt: s.CapturedAt, Value: float64(s.Count)})
	}
	var list [][]GrowthPoint
	for _, p := range order {
		s := by[p]
		sort.SliceStable(s, func(i, j int) bool { return s[i].CapturedAt < s[j].CapturedAt })
		list = append(list, s)
	}
	return MergeSeriesSum(list)
}

// DMSeries totals DMs per day across platforms (carry-forward).
func DMSeries(x *Data) []SeriesPoint { return seriesPoints(dmPoints(x)) }

// DMGrowthPct is the DM total's growth; days <= 0 is all time.
func DMGrowthPct(x *Data, days int) *float64 {
	p := dmPoints(x)
	if days <= 0 {
		return GrowthAllTime(p)
	}
	return GrowthOver(p, days, 0)
}

func DMGrowth(x *Data) Growth {
	return Growth{D7: DMGrowthPct(x, 7), D30: DMGrowthPct(x, 30), D60: DMGrowthPct(x, 60), AllTime: DMGrowthPct(x, 0)}
}

// Thread is one subscriber's conversation, messages oldest first.
type Thread struct {
	SubscriberID string      `json:"subscriberId"`
	Name         string      `json:"name"`
	Handle       *string     `json:"handle"`
	Messages     []DMMessage `json:"messages"`
	Last         DMMessage   `json:"last"`
	Unreplied    bool        `json:"unreplied"`
	// Demo: every message is seeded (source seed-dummy), not a real DM.
	Demo bool `json:"demo"`
}

// DMThreads groups a platform's messages into threads, newest thread first;
// unreplied when the last message is inbound.
func DMThreads(x *Data, platform string) []Thread {
	groups := map[string][]DMMessage{}
	var order []string
	for _, m := range x.DMMessages {
		if m.Platform != platform {
			continue
		}
		if _, ok := groups[m.SubscriberID]; !ok {
			order = append(order, m.SubscriberID)
		}
		groups[m.SubscriberID] = append(groups[m.SubscriberID], m)
	}
	out := []Thread{}
	for _, id := range order {
		newest := groups[id]
		chrono := make([]DMMessage, len(newest))
		for i, m := range newest {
			chrono[len(newest)-1-i] = m
		}
		last := newest[0]
		demo := true
		for _, m := range newest {
			if m.Source != "seed-dummy" {
				demo = false
			}
		}
		out = append(out, Thread{SubscriberID: id, Name: last.Name, Handle: last.Handle, Messages: chrono, Last: last, Unreplied: last.Direction == "in", Demo: demo})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Last.TS > out[j].Last.TS })
	return out
}

// ---- one platform ---------------------------------------------------------------

type Detail struct {
	Account   Account    `json:"account"`
	Followers *int       `json:"followers"`
	Growth    Growth     `json:"growth"`
	Snapshots []Snapshot `json:"snapshots"`
}

// PlatformDetail is platformDetail: nil for an untracked platform or one
// with no account row.
func PlatformDetail(x *Data, platform string) *Detail {
	if !Tracked(platform) {
		return nil
	}
	for _, a := range x.Accounts {
		if a.Platform != platform {
			continue
		}
		s := x.snaps(platform)
		d := &Detail{Account: a, Growth: SnapshotGrowth(s), Snapshots: append([]Snapshot{}, s...)}
		if len(s) > 0 {
			f := s[len(s)-1].Followers
			d.Followers = &f
		}
		return d
	}
	return nil
}
