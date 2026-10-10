package social

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/pages/pagekit"
)

// This file ports lib/social-volume.ts: the /social and /social/[platform]
// slab view models. Nothing here invents a number: a channel with no reading
// is an unknown meter labelled offline (the bridge's null frac, where the operator
// OS drew an empty 0), and one snapshot is a reading, not a trend.

var hues = []string{pagekit.Ramp1, pagekit.Ramp2, pagekit.Ramp3, pagekit.Ramp4}

var mixKeys = [][2]string{{"instagram", "IG"}, {"tiktok", "TT"}, {"twitter", "X"}, {"youtube", "YT"}, {"linkedin", "LI"}}

const meterCount = 4

func jsRound(f float64) float64 { return math.Floor(f + 0.5) }

func pctText(n float64, suffix string) string {
	sign := ""
	if n >= 0 {
		sign = "+"
	}
	digits := 1
	if math.Abs(n) < 10 {
		digits = 2
	}
	return sign + pagekit.Fixed(n, digits) + "% " + suffix
}

func day(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", s)
	return t, err == nil
}

// Channel is one audience channel for the Audience Volume card; Value nil =
// no reading.
type Channel struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value *int   `json:"value"`
}

// ThreadState is what the volume needs from a DM thread.
type ThreadState struct {
	Name      string
	Unreplied bool
	Demo      bool // seeded (seed-dummy), labelled as such
}

// PostDay is one post's UTC date and its cross-post platforms.
type PostDay struct {
	Date      string   `json:"date"`
	Platforms []string `json:"platforms"`
}

type Insight struct {
	Value    int     `json:"value"`
	Headline string  `json:"headline"`
	Body     string  `json:"body"`
	Frac     float64 `json:"frac"`
}

type SocialVolume struct {
	Headline      int             `json:"headline"`
	Chips         []pagekit.Chip  `json:"chips"`
	Caption       string          `json:"caption"`
	Meters        []pagekit.Meter `json:"meters"`
	Foot          string          `json:"foot"`
	Series        []pagekit.Point `json:"series"`
	PostsInWindow int             `json:"postsInWindow"`
	Mix           []pagekit.Point `json:"mix"`
	Insight       Insight         `json:"insight"`
}

type SocialVolumeInput struct {
	Channels []Channel
	Total    int
	Growth7  *float64
	Leader   string
	Threads  []ThreadState
	Queued   int
	PostDays []PostDay
	Today    string // YYYY-MM-DD (UTC)
	Days     int
}

// BuildSocialVolume is socialVolume.
func BuildSocialVolume(x SocialVolumeInput) SocialVolume {
	days := x.Days
	if days == 0 {
		days = 30
	}
	live := 0
	for _, c := range x.Channels {
		if c.Value != nil {
			live++
		}
	}
	var owed []string
	for _, t := range x.Threads {
		if t.Unreplied {
			owed = append(owed, t.Name)
		}
	}

	chips := []pagekit.Chip{}
	if x.Growth7 != nil && !math.IsNaN(*x.Growth7) && !math.IsInf(*x.Growth7, 0) {
		tone := "ok"
		if *x.Growth7 < 0 {
			tone = "err"
		}
		chips = append(chips, pagekit.Chip{Tone: tone, Text: pctText(*x.Growth7, "7d")})
	}
	if len(owed) > 0 {
		chips = append(chips, pagekit.Chip{Tone: "warn", Text: fmt.Sprintf("%d need reply", len(owed))})
	}
	if x.Queued > 0 {
		chips = append(chips, pagekit.Chip{Tone: "accent", Text: fmt.Sprintf("%d queued", x.Queued)})
	}

	caption := "no channels reporting yet"
	if live > 0 {
		s := "s"
		if live == 1 {
			s = ""
		}
		caption = fmt.Sprintf("across %d live channel%s", live, s)
		if x.Leader != "" {
			caption += " · " + x.Leader + " leads 7-day growth"
		}
	}

	ranked := append([]Channel(nil), x.Channels...)
	val := func(c Channel) int {
		if c.Value == nil {
			return -1
		}
		return *c.Value
	}
	sort.SliceStable(ranked, func(i, j int) bool { return val(ranked[i]) > val(ranked[j]) })
	shown := ranked
	if len(shown) > meterCount {
		shown = shown[:meterCount]
	}
	meters := []pagekit.Meter{}
	for i, c := range shown {
		if c.Value == nil {
			meters = append(meters, pagekit.Unknown(c.Label, "offline", hues[i]))
			continue
		}
		f := 0.0
		if x.Total > 0 {
			f = pagekit.Clamp01(float64(*c.Value) / float64(x.Total))
		}
		meters = append(meters, pagekit.Known(c.Label, f, fmt.Sprintf("%s · %d%%", pagekit.Thousands(*c.Value), int(jsRound(f*100))), hues[i]))
	}
	foot := "share of total reach"
	if rest := len(x.Channels) - len(shown); rest > 0 {
		s := "s"
		if rest == 1 {
			s = ""
		}
		foot += fmt.Sprintf(" · %d smaller channel%s", rest, s)
	}

	end, _ := day(x.Today)
	index := map[string]int{}
	series := make([]pagekit.Point, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := end.AddDate(0, 0, -i)
		index[d.Format("2006-01-02")] = len(series)
		series = append(series, pagekit.Point{Label: pagekit.ShortDate(d)})
	}
	mixCounts := map[string]int{}
	posts := 0
	for _, p := range x.PostDays {
		i, ok := index[p.Date]
		if !ok {
			continue
		}
		series[i].Count++
		posts++
		seen := map[string]bool{}
		for _, pl := range p.Platforms {
			if !seen[pl] {
				seen[pl] = true
				mixCounts[pl]++
			}
		}
	}
	mix := make([]pagekit.Point, 0, len(mixKeys))
	for _, k := range mixKeys {
		mix = append(mix, pagekit.Point{Label: k[1], Count: float64(mixCounts[k[0]])})
	}

	threads := len(x.Threads)
	in := Insight{Value: len(owed), Headline: "Inbox clear."}
	switch {
	case len(owed) > 0:
		in.Headline = fmt.Sprintf("%d of %d Instagram threads need a reply.", len(owed), threads)
		names := owed
		if len(names) > 3 {
			names = names[:3]
		}
		in.Body = strings.Join(names, " · ")
	case threads > 0:
		in.Body = fmt.Sprintf("%d threads, every one answered.", threads)
	default:
		in.Body = "No Instagram threads yet."
	}
	if threads > 0 {
		in.Frac = float64(len(owed)) / float64(threads)
	}

	return SocialVolume{Headline: x.Total, Chips: chips, Caption: caption, Meters: meters, Foot: foot, Series: series, PostsInWindow: posts, Mix: mix, Insight: in}
}

// ---- one platform ---------------------------------------------------------------

type PlatformInsight struct {
	Value    int     `json:"value"`
	Display  string  `json:"display"`
	Headline string  `json:"headline"`
	Body     string  `json:"body"`
	Frac     float64 `json:"frac"`
}

type PlatformVolume struct {
	Headline    *int            `json:"headline"`
	Chips       []pagekit.Chip  `json:"chips"`
	Caption     string          `json:"caption"`
	Meters      []pagekit.Meter `json:"meters"`
	Foot        string          `json:"foot"`
	Series      []pagekit.Point `json:"series"`
	Intervals   int             `json:"intervals"`
	GainedDays  int             `json:"gainedDays"`
	DippedDays  int             `json:"dippedDays"`
	TrackedDays int             `json:"trackedDays"`
	Net         *int            `json:"net"`
	Insight     PlatformInsight `json:"insight"`
}

func signed(n int) string {
	switch {
	case n > 0:
		return "+" + pagekit.Thousands(n)
	case n < 0:
		return "-" + pagekit.Thousands(-n)
	}
	return "0"
}

func isoLabel(s string) string {
	t, ok := day(s)
	if !ok {
		return s
	}
	return pagekit.ShortDate(t)
}

// BuildPlatformVolume is platformVolume: a day's change is the delta from
// the snapshot before it; dips sit at zero on the gains line.
func BuildPlatformVolume(label string, followers *int, g Growth, snapshots []Snapshot, today string, days int) PlatformVolume {
	if days == 0 {
		days = 30
	}
	snaps := append([]Snapshot(nil), snapshots...)
	sort.SliceStable(snaps, func(i, j int) bool { return snaps[i].CapturedAt < snaps[j].CapturedAt })
	end, _ := day(today)
	startKey := end.AddDate(0, 0, -(days - 1)).Format("2006-01-02")

	index := map[string]int{}
	series := make([]pagekit.Point, 0, days)
	for i := days - 1; i >= 0; i-- {
		d := end.AddDate(0, 0, -i)
		index[d.Format("2006-01-02")] = len(series)
		series = append(series, pagekit.Point{Label: pagekit.ShortDate(d)})
	}

	intervals, gained, dipped, net := 0, 0, 0, 0
	bestLabel, bestGain := "", 0
	for i := 1; i < len(snaps); i++ {
		at, ok := index[snaps[i].CapturedAt]
		if !ok {
			continue
		}
		delta := snaps[i].Followers - snaps[i-1].Followers
		intervals++
		net += delta
		if delta > 0 {
			gained++
			series[at].Count += float64(delta)
			if bestLabel == "" || delta > bestGain {
				bestLabel, bestGain = series[at].Label, delta
			}
		} else if delta < 0 {
			dipped++
		}
	}
	tracked := map[string]bool{}
	peak := -1
	for _, s := range snaps {
		if s.CapturedAt >= startKey && s.CapturedAt <= today {
			tracked[s.CapturedAt] = true
		}
		if s.Followers > peak {
			peak = s.Followers
		}
	}

	chips := []pagekit.Chip{}
	for _, w := range []struct {
		v *float64
		l string
	}{{g.D7, "7d"}, {g.D30, "30d"}} {
		if w.v != nil && !math.IsNaN(*w.v) && !math.IsInf(*w.v, 0) {
			tone := "ok"
			if *w.v < 0 {
				tone = "err"
			}
			chips = append(chips, pagekit.Chip{Tone: tone, Text: pctText(*w.v, w.l)})
		}
	}

	n := len(snaps)
	plural := func(n int) string {
		if n == 1 {
			return ""
		}
		return "s"
	}
	caption := "no follower reading yet"
	if followers != nil {
		caption = fmt.Sprintf("%s followers · %d snapshot%s", label, n, plural(n))
		if n > 0 {
			caption += " since " + isoLabel(snaps[0].CapturedAt)
		}
	}

	meters := []pagekit.Meter{}
	if intervals > 0 {
		meters = append(meters,
			pagekit.Known(fmt.Sprintf("Days gained (%d/%d)", gained, intervals), float64(gained)/float64(intervals), fmt.Sprintf("%d of %d", gained, intervals), pagekit.OK),
			pagekit.Known(fmt.Sprintf("Days dipped (%d/%d)", dipped, intervals), float64(dipped)/float64(intervals), fmt.Sprintf("%d of %d", dipped, intervals), pagekit.Err))
	}
	if t := len(tracked); t > 0 {
		f := math.Min(1, float64(t)/float64(days))
		meters = append(meters, pagekit.Known(fmt.Sprintf("Tracked days (%d/%d)", t, days), f, fmt.Sprintf("%d%%", int(jsRound(f*100))), pagekit.Ramp1))
	}
	if followers != nil && peak > 0 {
		meters = append(meters, pagekit.Known("Of all-time peak", pagekit.Clamp01(float64(*followers)/float64(peak)),
			fmt.Sprintf("%s of %s", pagekit.Thousands(*followers), pagekit.Thousands(peak)), pagekit.Accent))
	}

	foot := "no snapshots yet"
	if n > 0 {
		latest := snaps[n-1]
		foot = fmt.Sprintf("%d snapshot%s · latest %s · %s", n, plural(n), isoLabel(latest.CapturedAt), latest.Source)
	}

	var in PlatformInsight
	var netPtr *int
	if intervals == 0 {
		in = PlatformInsight{Display: "none", Headline: "Not enough history for a trend yet.", Body: "Two snapshots make a line; the next sync adds one."}
	} else {
		netPtr = &net
		in = PlatformInsight{Value: net, Display: signed(net), Frac: float64(gained) / float64(intervals)}
		switch {
		case net > 0:
			in.Headline = fmt.Sprintf("Gained %s followers in the last %d days.", pagekit.Thousands(net), days)
		case net < 0:
			in.Headline = fmt.Sprintf("Lost %s followers in the last %d days.", pagekit.Thousands(-net), days)
		default:
			in.Headline = fmt.Sprintf("Flat over the last %d days.", days)
		}
		in.Body = fmt.Sprintf("up on %d of %d tracked days", gained, intervals)
		if bestLabel != "" {
			in.Body += fmt.Sprintf(" · best day %s (+%s)", bestLabel, pagekit.Thousands(bestGain))
		}
	}

	return PlatformVolume{Headline: followers, Chips: chips, Caption: caption, Meters: meters, Foot: foot, Series: series,
		Intervals: intervals, GainedDays: gained, DippedDays: dipped, TrackedDays: len(tracked), Net: netPtr, Insight: in}
}
