package comms

import (
	"regexp"
	"strings"
	"testing"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
)

// Ported from FounderOS v1 tests/newsletter-agent.test.ts, plus the summary
// line and notes the TS pins through the agent.
func send(over func(*beehiiv.Newsletter)) beehiiv.Newsletter {
	n := beehiiv.Newsletter{ID: "n1", Title: "Issue", PublishedAt: "2026-08-01T00:00:00.000Z", Recipients: 1000, Delivered: 1000,
		DeliveryRate: 100, Opens: 400, OpenRate: 40, Clicks: 50, ClickRate: 5, Unsubscribes: 5, UnsubscribeRate: 0.5}
	if over != nil {
		over(&n)
	}
	return n
}

func TestBriefNoSendsIsStatedPlainly(t *testing.T) {
	b := BuildNewsletterBrief(nil)
	if b.Sends != 0 || b.Confident || b.BestBySubject != nil || b.BestByBody != nil || b.Worst != nil {
		t.Fatalf("%+v", b)
	}
	if !regexp.MustCompile(`(?i)no sends|nothing to learn`).MatchString(strings.Join(b.Notes, " ")) {
		t.Fatalf("notes = %q", b.Notes)
	}
	if b.RecentTitles == nil || len(b.RecentTitles) != 0 {
		t.Fatal("recentTitles is an empty list, never null")
	}
	if got := BriefSummary(b); got != "No Beehiiv sends on record yet, so the draft has no performance history behind it." {
		t.Fatalf("summary = %q", got)
	}
}

func TestBriefThinHistoryRefusesToCallAPattern(t *testing.T) {
	b := BuildNewsletterBrief([]beehiiv.Newsletter{send(func(n *beehiiv.Newsletter) { n.ID = "a" }), send(func(n *beehiiv.Newsletter) { n.ID = "b" })})
	if b.Sends != 2 || b.Confident || b.MedianOpenRate != 40 {
		t.Fatalf("%+v", b)
	}
	if b.Notes[0] != "Only 2 sends on record, which is too few to call a pattern. Treat the numbers below as anecdotes, not evidence." {
		t.Fatalf("notes = %q", b.Notes)
	}
	if got := BriefSummary(b); got != "2 sends · median 40% open / 5% click · only 2 sends, treat as anecdote" {
		t.Fatalf("summary = %q", got)
	}
	one := BuildNewsletterBrief([]beehiiv.Newsletter{send(nil)})
	if !strings.HasPrefix(one.Notes[0], "Only 1 send on record,") {
		t.Fatalf("singular = %q", one.Notes[0])
	}
}

func TestBriefEnoughHistoryIsConfidentAndFlagsTheClickWinner(t *testing.T) {
	var many []beehiiv.Newsletter
	for i := range MinSendsForConfidence {
		many = append(many, send(func(n *beehiiv.Newsletter) {
			n.ID, n.OpenRate, n.ClickRate = "n"+string(rune('0'+i)), float64(30+i), float64(3+i)
		}))
	}
	b := BuildNewsletterBrief(many)
	if !b.Confident {
		t.Fatalf("%+v", b)
	}
	if got := BriefSummary(b); got != "5 sends · median 32% open / 5% click · enough history to aim at" {
		t.Fatalf("summary = %q", got)
	}
	// 7% clicks against a 5% median is not 1.5x; 9% is.
	many[4].ClickRate = 9
	b = BuildNewsletterBrief(many)
	if len(b.Notes) != 1 || b.Notes[0] != `"Issue" pulled 9% clicks against a 5% median. Whatever it asked readers to do, that ask works.` {
		t.Fatalf("notes = %q", b.Notes)
	}
}

func TestBriefWinnersMediansAndWorst(t *testing.T) {
	b := BuildNewsletterBrief([]beehiiv.Newsletter{
		send(func(n *beehiiv.Newsletter) { n.ID, n.Title, n.OpenRate, n.ClickRate = "a", "Great subject", 62, 2 }),
		send(func(n *beehiiv.Newsletter) { n.ID, n.Title, n.OpenRate, n.ClickRate = "b", "Great body", 31, 11 }),
		send(func(n *beehiiv.Newsletter) { n.ID, n.Title, n.OpenRate, n.ClickRate = "c", "Middle", 40, 5 }),
	})
	if b.BestBySubject.Title != "Great subject" || b.BestByBody.Title != "Great body" || b.Worst.Title != "Great body" {
		t.Fatalf("%+v %+v %+v", b.BestBySubject, b.BestByBody, b.Worst)
	}
	if *b.BestByBody != (IssueRef{ID: "b", Title: "Great body", OpenRate: 31, ClickRate: 11}) {
		t.Fatalf("ref = %+v", *b.BestByBody)
	}
	odd := BuildNewsletterBrief([]beehiiv.Newsletter{
		send(func(n *beehiiv.Newsletter) { n.OpenRate = 10 }), send(func(n *beehiiv.Newsletter) { n.OpenRate = 40 }), send(func(n *beehiiv.Newsletter) { n.OpenRate = 900 }),
	})
	if odd.MedianOpenRate != 40 {
		t.Fatalf("median = %v", odd.MedianOpenRate)
	}
	even := BuildNewsletterBrief([]beehiiv.Newsletter{send(func(n *beehiiv.Newsletter) { n.OpenRate = 20 }), send(func(n *beehiiv.Newsletter) { n.OpenRate = 41.255 })})
	if even.MedianOpenRate != 30.63 {
		t.Fatalf("even median rounds to 2dp: %v", even.MedianOpenRate)
	}
	// Ties keep the first send, as Array.reduce does.
	tie := BuildNewsletterBrief([]beehiiv.Newsletter{
		send(func(n *beehiiv.Newsletter) { n.Title = "First" }), send(func(n *beehiiv.Newsletter) { n.Title = "Second" }),
	})
	if tie.BestBySubject.Title != "First" || tie.Worst.Title != "First" {
		t.Fatalf("tie = %+v %+v", tie.BestBySubject, tie.Worst)
	}
}

func TestBriefUnsubscribeSpikeOnTheLatestSend(t *testing.T) {
	b := BuildNewsletterBrief([]beehiiv.Newsletter{
		send(func(n *beehiiv.Newsletter) {
			n.ID, n.PublishedAt, n.UnsubscribeRate = "old1", "2026-07-01T00:00:00.000Z", 0.4
		}),
		send(func(n *beehiiv.Newsletter) {
			n.ID, n.Title, n.PublishedAt, n.UnsubscribeRate = "latest", "Hot take", "2026-08-01T00:00:00.000Z", 3.2
		}),
		send(func(n *beehiiv.Newsletter) {
			n.ID, n.PublishedAt, n.UnsubscribeRate = "old2", "2026-07-08T00:00:00.000Z", 0.5
		}),
	})
	want := `The last send ("Hot take") lost 3.2% of the list against a 0.5% median. Something in it cost subscribers, so do not repeat its angle without a reason.`
	if b.UnsubscribeWarning == nil || *b.UnsubscribeWarning != want {
		t.Fatalf("warning = %v", b.UnsubscribeWarning)
	}
	if !strings.HasSuffix(BriefSummary(b), " · unsubscribe spike flagged") {
		t.Fatalf("summary = %q", BriefSummary(b))
	}
	calm := BuildNewsletterBrief([]beehiiv.Newsletter{
		send(func(n *beehiiv.Newsletter) { n.PublishedAt, n.UnsubscribeRate = "2026-07-01T00:00:00.000Z", 0.4 }),
		send(func(n *beehiiv.Newsletter) { n.PublishedAt, n.UnsubscribeRate = "2026-08-01T00:00:00.000Z", 0.5 }),
	})
	if calm.UnsubscribeWarning != nil {
		t.Fatalf("a normal rate raises nothing: %v", *calm.UnsubscribeWarning)
	}
}

func TestBriefRecentTitlesNewestFirstCappedAtSix(t *testing.T) {
	var list []beehiiv.Newsletter
	for i := 1; i <= 8; i++ {
		list = append(list, send(func(n *beehiiv.Newsletter) {
			n.Title, n.PublishedAt = "T"+string(rune('0'+i)), "2026-08-0"+string(rune('0'+i))+"T00:00:00.000Z"
		}))
	}
	b := BuildNewsletterBrief(list)
	if strings.Join(b.RecentTitles, ",") != "T8,T7,T6,T5,T4,T3" {
		t.Fatalf("recent = %q", b.RecentTitles)
	}
}
