package social

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pagekit"
)

// This file ports lib/newsletters.ts and lib/newsletter-volume.ts: the
// /social/beehiiv view model. Seeded issues (id seed-*) are labelled a
// preview, never passed off as sends.

// SeedNewsletters is FounderOS v1's SEED_NEWSLETTERS, newest first: the page's
// fallback while Beehiiv returns no posts.
var SeedNewsletters = []beehiiv.Newsletter{
	{ID: "seed-4", Title: "Ship the system, not the hustle", PublishedAt: "2026-07-14T15:00:00.000Z", Recipients: 15400, Delivered: 15150, DeliveryRate: 98.38, Opens: 3520, OpenRate: 23.24, Clicks: 158, ClickRate: 4.6, Unsubscribes: 88, UnsubscribeRate: 0.58, SpamReports: 0, WebViews: 6},
	{ID: "seed-3", Title: "What 30 days of operator OS looks like", PublishedAt: "2026-07-07T15:00:00.000Z", Recipients: 14600, Delivered: 14380, DeliveryRate: 98.49, Opens: 3180, OpenRate: 22.12, Clicks: 141, ClickRate: 4.28, Unsubscribes: 79, UnsubscribeRate: 0.55, SpamReports: 1, WebViews: 4},
	{ID: "seed-2", Title: "The duct-tape stack is costing you", PublishedAt: "2026-06-30T15:00:00.000Z", Recipients: 13900, Delivered: 13690, DeliveryRate: 98.49, Opens: 3010, OpenRate: 21.99, Clicks: 128, ClickRate: 4.05, Unsubscribes: 84, UnsubscribeRate: 0.61, SpamReports: 0, WebViews: 5},
	{ID: "seed-1", Title: "Why AI does not know you yet", PublishedAt: "2026-06-23T15:00:00.000Z", Recipients: 13100, Delivered: 12880, DeliveryRate: 98.32, Opens: 2760, OpenRate: 21.43, Clicks: 112, ClickRate: 3.81, Unsubscribes: 90, UnsubscribeRate: 0.7, SpamReports: 1, WebViews: 3},
}

type NewsletterInsight struct {
	Display  string  `json:"display"`
	Headline string  `json:"headline"`
	Body     string  `json:"body"`
	Frac     float64 `json:"frac"`
}

type NewsletterVolume struct {
	Headline        *int              `json:"headline"`
	Chips           []pagekit.Chip    `json:"chips"`
	Caption         string            `json:"caption"`
	Meters          []pagekit.Meter   `json:"meters"`
	Foot            string            `json:"foot"`
	Sends           []pagekit.Point   `json:"sends"`
	OpenRates       []pagekit.Point   `json:"openRates"`
	AvgOpenRate     *float64          `json:"avgOpenRate"`
	TotalRecipients float64           `json:"totalRecipients"`
	TotalClicks     float64           `json:"totalClicks"`
	Unsubscribes    float64           `json:"unsubscribes"`
	SpamReports     float64           `json:"spamReports"`
	Insight         NewsletterInsight `json:"insight"`
}

const matrixCols = 6

func shortUTC(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}
	return pagekit.ShortDate(t.UTC())
}

func pctOf(f float64, digits int) string { return pagekit.Fixed(f*100, digits) + "%" }

// BuildNewsletterVolume is newsletterVolume. The funnel meters are ratios of
// summed send stats, never averaged per-issue percentages.
func BuildNewsletterVolume(list []beehiiv.Newsletter, subscribers *int) NewsletterVolume {
	count := len(list)
	oldest := append([]beehiiv.Newsletter(nil), list...)
	sort.SliceStable(oldest, func(i, j int) bool { return oldest[i].PublishedAt < oldest[j].PublishedAt })
	seeded := count > 0
	var rec, del, opens, clicks, unsub, spam, rateSum float64
	for _, n := range list {
		if !strings.HasPrefix(n.ID, "seed-") {
			seeded = false
		}
		rec += n.Recipients
		del += n.Delivered
		opens += n.Opens
		clicks += n.Clicks
		unsub += n.Unsubscribes
		spam += n.SpamReports
		rateSum += n.OpenRate
	}
	var avg *float64
	if count > 0 {
		a := jsRound(rateSum/float64(count)*100) / 100
		avg = &a
	}
	plural := func(n int) string {
		if n == 1 {
			return ""
		}
		return "s"
	}

	v := NewsletterVolume{Headline: subscribers, Chips: []pagekit.Chip{}, Meters: []pagekit.Meter{}, Sends: []pagekit.Point{}, OpenRates: []pagekit.Point{},
		AvgOpenRate: avg, TotalRecipients: rec, TotalClicks: clicks, Unsubscribes: unsub, SpamReports: spam}
	if count > 0 {
		v.Chips = append(v.Chips, pagekit.Chip{Tone: "accent", Text: fmt.Sprintf("%d issue%s", count, plural(count))},
			pagekit.Chip{Tone: "ok", Text: pagekit.Fixed(*avg, 1) + "% avg open"})
	}
	if rec > 0 {
		v.Meters = append(v.Meters, pagekit.Known("Delivered", pagekit.Clamp01(del/rec), pctOf(del/rec, 1)+" of sends", pagekit.OK))
	}
	if del > 0 {
		v.Meters = append(v.Meters, pagekit.Known("Opened", pagekit.Clamp01(opens/del), pctOf(opens/del, 1)+" of delivered", pagekit.Ramp1))
	}
	if opens > 0 {
		v.Meters = append(v.Meters, pagekit.Known("Clicked", pagekit.Clamp01(clicks/opens), pctOf(clicks/opens, 1)+" of opens", pagekit.Ramp3))
	}
	if del > 0 {
		v.Meters = append(v.Meters, pagekit.Known("Unsubscribed", pagekit.Clamp01(unsub/del), pctOf(unsub/del, 2)+" of delivered", pagekit.Warn))
	}
	v.Foot = "no issues yet"
	if count > 0 {
		v.Foot = fmt.Sprintf("%d issue%s · %s sends", count, plural(count), pagekit.Thousands(int(rec)))
		if seeded {
			v.Foot += " · seeded preview"
		}
	}
	v.Caption = "no live subscriber count · add BEEHIIV_API_KEY"
	if subscribers != nil {
		v.Caption = "subscribers, live via Beehiiv"
	}
	for _, n := range oldest {
		v.Sends = append(v.Sends, pagekit.Point{Label: shortUTC(n.PublishedAt), Count: n.Recipients})
	}
	tail := oldest
	if len(tail) > matrixCols {
		tail = tail[len(tail)-matrixCols:]
	}
	for _, n := range tail {
		v.OpenRates = append(v.OpenRates, pagekit.Point{Label: shortUTC(n.PublishedAt), Count: n.OpenRate})
	}
	// two issues on one day must not share a DotMatrix column key
	v.OpenRates = pagekit.UniqueLabels(v.OpenRates)

	var best *beehiiv.Newsletter
	for i := range list {
		if best == nil || best.OpenRate < list[i].OpenRate {
			best = &list[i]
		}
	}
	if best == nil {
		v.Insight = NewsletterInsight{Display: "none", Headline: "No issues sent yet.", Body: "Open rates show here after the first send."}
	} else {
		v.Insight = NewsletterInsight{
			Display:  pagekit.Fixed(best.OpenRate, 1) + "%",
			Headline: "Best open rate: " + best.Title + ".",
			Body:     fmt.Sprintf("%s%% average across %d issue%s · %s", pagekit.Fixed(*avg, 1), count, plural(count), shortUTC(best.PublishedAt)),
			Frac:     pagekit.Clamp01(best.OpenRate / 100),
		}
	}
	return v
}
