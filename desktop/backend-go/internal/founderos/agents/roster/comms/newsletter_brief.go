package comms

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
)

// The port of FounderOS v1 lib/agents/newsletter-brief.ts: what the list has
// actually done, assembled before anything is written. The honesty rule
// matters more than the arithmetic: with two sends there is no trend, so the
// brief carries Confident and says out loud when it is guessing.

// IssueRef points at one past send.
type IssueRef struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	OpenRate  float64 `json:"openRate"`
	ClickRate float64 `json:"clickRate"`
}

// NewsletterBrief is the TS NewsletterBrief, field for field.
type NewsletterBrief struct {
	Sends int `json:"sends"`
	// Confident: enough history to draw a conclusion from.
	Confident       bool    `json:"confident"`
	MedianOpenRate  float64 `json:"medianOpenRate"`
	MedianClickRate float64 `json:"medianClickRate"`
	// BestBySubject is the best opens (what the subject line learns from),
	// BestByBody the best clicks (what the body and offer learn from).
	BestBySubject      *IssueRef `json:"bestBySubject"`
	BestByBody         *IssueRef `json:"bestByBody"`
	Worst              *IssueRef `json:"worst"`
	UnsubscribeWarning *string   `json:"unsubscribeWarning"`
	// RecentTitles, newest first, so the same idea is not pitched twice.
	RecentTitles []string `json:"recentTitles"`
	Notes        []string `json:"notes"`
}

// MinSendsForConfidence: below this many sends the numbers are anecdotes.
const MinSendsForConfidence = 5

const (
	unsubSpikeMultiple = 2.0 // latest unsubscribe rate vs the median...
	unsubSpikeFloor    = 1.0 // ...once it is meaningful in absolute terms too
	recentTitles       = 6
)

// round2 is Math.round(n*100)/100 (half rounds up, as in JS).
func round2(n float64) float64 { return math.Floor(n*100+0.5) / 100 }

// jsNum prints a number the way a JS template literal does for these values.
func jsNum(n float64) string { return strconv.FormatFloat(n, 'f', -1, 64) }

func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	s := append([]float64(nil), values...)
	sort.Float64s(s)
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return round2(s[mid])
	}
	return round2((s[mid-1] + s[mid]) / 2)
}

func ref(n beehiiv.Newsletter) *IssueRef {
	return &IssueRef{ID: n.ID, Title: n.Title, OpenRate: n.OpenRate, ClickRate: n.ClickRate}
}

// bestBy is the TS reduce: the first send with the highest pick wins a tie.
func bestBy(list []beehiiv.Newsletter, pick func(beehiiv.Newsletter) float64) *beehiiv.Newsletter {
	if len(list) == 0 {
		return nil
	}
	best := list[0]
	for _, n := range list[1:] {
		if pick(n) > pick(best) {
			best = n
		}
	}
	return &best
}

func rates(list []beehiiv.Newsletter, f func(beehiiv.Newsletter) float64) []float64 {
	out := make([]float64, len(list))
	for i, n := range list {
		out[i] = f(n)
	}
	return out
}

// BuildNewsletterBrief is buildNewsletterBrief.
func BuildNewsletterBrief(newsletters []beehiiv.Newsletter) NewsletterBrief {
	sends := len(newsletters)
	if sends == 0 {
		return NewsletterBrief{
			RecentTitles: []string{},
			Notes:        []string{"No sends on record yet, so there is nothing to learn from. Write to the audience the operator describes in the skill file, not to past performance."},
		}
	}
	notes := []string{}

	// Newest first; ISO timestamps compare correctly as strings, and the sort
	// is stable like Array.prototype.sort.
	byNewest := append([]beehiiv.Newsletter(nil), newsletters...)
	sort.SliceStable(byNewest, func(i, j int) bool { return byNewest[i].PublishedAt > byNewest[j].PublishedAt })

	openRate := func(n beehiiv.Newsletter) float64 { return n.OpenRate }
	clickRate := func(n beehiiv.Newsletter) float64 { return n.ClickRate }
	medianOpen := median(rates(newsletters, openRate))
	medianClick := median(rates(newsletters, clickRate))
	confident := sends >= MinSendsForConfidence

	if !confident {
		plural := "s"
		if sends == 1 {
			plural = ""
		}
		notes = append(notes, fmt.Sprintf("Only %d send%s on record, which is too few to call a pattern. Treat the numbers below as anecdotes, not evidence.", sends, plural))
	}

	// Opens judge the subject line; clicks judge whether the body earned the open.
	topOpen := bestBy(newsletters, openRate)
	topClick := bestBy(newsletters, clickRate)
	worst := bestBy(newsletters, func(n beehiiv.Newsletter) float64 { return -n.OpenRate })

	var warning *string
	latest := byNewest[0]
	medianUnsub := median(rates(newsletters, func(n beehiiv.Newsletter) float64 { return n.UnsubscribeRate }))
	if latest.UnsubscribeRate >= unsubSpikeFloor && latest.UnsubscribeRate > medianUnsub*unsubSpikeMultiple {
		w := fmt.Sprintf(`The last send ("%s") lost %s%% of the list against a %s%% median. Something in it cost subscribers, so do not repeat its angle without a reason.`,
			latest.Title, jsNum(latest.UnsubscribeRate), jsNum(medianUnsub))
		warning = &w
		notes = append(notes, w)
	}

	if confident && topClick != nil && topClick.ClickRate > medianClick*1.5 {
		notes = append(notes, fmt.Sprintf(`"%s" pulled %s%% clicks against a %s%% median. Whatever it asked readers to do, that ask works.`,
			topClick.Title, jsNum(topClick.ClickRate), jsNum(medianClick)))
	}

	titles := []string{}
	for i, n := range byNewest {
		if i == recentTitles {
			break
		}
		titles = append(titles, n.Title)
	}
	return NewsletterBrief{
		Sends:              sends,
		Confident:          confident,
		MedianOpenRate:     medianOpen,
		MedianClickRate:    medianClick,
		BestBySubject:      ref(*topOpen),
		BestByBody:         ref(*topClick),
		Worst:              ref(*worst),
		UnsubscribeWarning: warning,
		RecentTitles:       titles,
		Notes:              notes,
	}
}

// BriefSummary is briefSummary: the one-line run summary.
func BriefSummary(b NewsletterBrief) string {
	if b.Sends == 0 {
		return "No Beehiiv sends on record yet, so the draft has no performance history behind it."
	}
	conf := "enough history to aim at"
	if !b.Confident {
		conf = fmt.Sprintf("only %d sends, treat as anecdote", b.Sends)
	}
	spike := ""
	if b.UnsubscribeWarning != nil {
		spike = " · unsubscribe spike flagged"
	}
	return fmt.Sprintf("%d sends · median %s%% open / %s%% click · %s%s", b.Sends, jsNum(b.MedianOpenRate), jsNum(b.MedianClickRate), conf, spike)
}
