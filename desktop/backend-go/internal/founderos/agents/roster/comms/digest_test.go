package comms

import (
	"encoding/json"
	"slices"
	"testing"
	"time"
)

// Ported from FounderOS v1 tests/comms-digest.test.ts and
// tests/digest-stacking.test.ts: the ranking is the operator's stated priority
// (calls, clients, students/family, brand deals, group chats, companies).

var digestNow = time.Date(2026, 8, 18, 9, 0, 0, 0, time.UTC)

func isoAt(t time.Time) string { return t.UTC().Format(isoMillis) }

func mkItem(over func(*Item)) Item {
	it := Item{Source: "email", Title: "Subject", Preview: "body text", TS: isoAt(digestNow.Add(-time.Hour)), Sender: "someone@example.com"}
	if over != nil {
		over(&it)
	}
	return it
}

func mkCtx(over func(*DigestContext)) DigestContext {
	c := DigestContext{Now: digestNow}
	if over != nil {
		over(&c)
	}
	return c
}

func gmailItem(inbox, sender, subject string) Item {
	return mkItem(func(i *Item) { i.Sender, i.Title, i.Preview = sender, inbox+" — "+sender, subject })
}

func TestBulkDetection(t *testing.T) {
	for _, s := range []string{"noreply@stripe.com", "no-reply@x.com", "notifications@github.com", "mailer@foo.io"} {
		if !IsBulkSender(s, "", "") {
			t.Errorf("%s should be bulk", s)
		}
	}
	if IsBulkSender("morganhale@hale-advisory.example", "Re: the audit", "Sounds good") {
		t.Error("a real person is never bulk")
	}
	if !IsBulkSender("hello@somesaas.com", "Product update", "To unsubscribe click here") {
		t.Error("unsubscribe copy marks bulk")
	}
}

func TestRealEmailShape(t *testing.T) {
	g := func(s, subj string) Item { return gmailItem("Launchpad Cohort", s, subj) }
	cases := []struct {
		it   Item
		ctx  DigestContext
		want Tier
	}{
		{g("TikTok Shop", "Your order has shipped"), mkCtx(nil), TierNoise},
		{g("HighLevel", "Weekly newsletter: 5 tips"), mkCtx(nil), TierNoise},
		{g("Morgan Hale", "Re: the two week audit"), mkCtx(func(c *DigestContext) { c.MeetingTitles = []string{"Alex <> Morgan Hale"} }), TierCall},
		{g("Partnerships", "Sponsorship opportunity"), mkCtx(nil), TierBrandDeal},
	}
	for _, c := range cases {
		if got := Classify(c.it, c.ctx).Tier; got != c.want {
			t.Errorf("%s / %s: got %s want %s", c.it.Sender, c.it.Preview, got, c.want)
		}
	}
	out := UnsubscribeCandidates([]Item{g("TikTok Shop", "Your receipt"), g("TikTok Shop", "Your receipt"), g("Morgan", "Re: audit")}, mkCtx(nil))
	if len(out) != 1 || out[0].Sender != "TikTok Shop" || out[0].Count != 2 {
		t.Fatalf("unsubscribes = %+v", out)
	}
}

func TestGroupChats(t *testing.T) {
	if !IsGroupChat(mkItem(func(i *Item) { i.Source, i.Sender = "whatsapp", "FounderOS Cohort 1 (12)" })) {
		t.Error("member count marks a group")
	}
	if IsGroupChat(mkItem(func(i *Item) { i.Source, i.Sender = "whatsapp", "Mom" })) {
		t.Error("one-to-one is not a group")
	}
}

func TestClassify(t *testing.T) {
	type tc struct {
		name string
		it   Item
		ctx  DigestContext
		want Tier
	}
	cases := []tc{
		{"calendar outranks", mkItem(func(i *Item) { i.Sender, i.Title = "Morgan Hale", "Re: the two week audit" }),
			mkCtx(func(c *DigestContext) { c.MeetingTitles = []string{"Alex <> Morgan Hale — discovery"} }), TierCall},
		{"client", mkItem(func(i *Item) { i.Sender = "ops@harbor-residences.example" }),
			mkCtx(func(c *DigestContext) { c.ClientNames = []string{"Harbor Residences"} }), TierClient},
		{"student", mkItem(func(i *Item) { i.Source, i.Sender = "whatsapp", "Riley Novak" }),
			mkCtx(func(c *DigestContext) { c.Students = []string{"Riley Novak"} }), TierPeople},
		{"family", mkItem(func(i *Item) { i.Source, i.Sender = "whatsapp", "Mom" }),
			mkCtx(func(c *DigestContext) { c.Family = []string{"Mom"} }), TierPeople},
		{"cohort question in a group", mkItem(func(i *Item) {
			i.Source, i.Sender, i.Preview = "whatsapp", "Cohort 1 Group (18)", "quick question about the cohort call tomorrow"
		}), mkCtx(nil), TierPeople},
		{"group chatter", mkItem(func(i *Item) { i.Source, i.Sender, i.Preview = "whatsapp", "Family Group (6)", "lol" }), mkCtx(nil), TierGroup},
		{"brand deal", mkItem(func(i *Item) {
			i.Sender, i.Title = "partnerships@brandco.com", "Sponsorship opportunity for your channel"
		}), mkCtx(nil), TierBrandDeal},
		{"software", mkItem(func(i *Item) { i.Sender, i.Title = "noreply@vercel.com", "Deployment ready" }), mkCtx(nil), TierNoise},
		{"unknown human", mkItem(func(i *Item) { i.Sender, i.Title = "someguy@gmail.com", "Question about your program" }), mkCtx(nil), TierPeople},
		// live-data regressions: companies and software stay out of people
		{"CI mail", gmailItem("Personal", "Alex", "[alexdev/founderos] Run failed: CI - main"), mkCtx(nil), TierNoise},
		{"ticketing", gmailItem("Personal", "Ticketmaster", "Chick-fil-A Peach Bowl, 6LACK, Aflac Kickoff"), mkCtx(nil), TierNoise},
		{"venue", gmailItem("Personal", "Downtown Comedy Club", "THIS WEEK: Headliner Night + $13 Tickets"), mkCtx(nil), TierNoise},
		{"nextdoor", gmailItem("Personal", "Trending on Nextdoor", "I was sitting at a red light..."), mkCtx(nil), TierNoise},
		{"bounce", gmailItem("Personal", "Mail Delivery Subsystem", "Delivery Status Notification (Failure)"), mkCtx(nil), TierNoise},
		{"meeting bot", gmailItem("Personal", "Fathom", "Recap of your meeting with sam@example.com"), mkCtx(nil), TierNoise},
		{"person 1", gmailItem("Personal", "Frankie Dalton", "We are still posting everyday"), mkCtx(nil), TierPeople},
		{"person 2", gmailItem("Personal", "Sami K", "like later in the day"), mkCtx(nil), TierPeople},
		// strict name matching
		{"full name", mkItem(func(i *Item) { i.Sender = "Morgan Hale" }),
			mkCtx(func(c *DigestContext) { c.MeetingTitles = []string{"Alex <> Morgan Hale — audit"} }), TierCall},
		{"surname", mkItem(func(i *Item) { i.Sender = "kowalczyk@fancy.example" }),
			mkCtx(func(c *DigestContext) { c.MeetingTitles = []string{"Lena Kowalczyk intro"} }), TierCall},
		{"linkedin relay", mkItem(func(i *Item) { i.Sender, i.Preview = "Pat ROWAN via LinkedIn", "Why the IBM partnership matters" }), mkCtx(nil), TierNoise},
	}
	for _, c := range cases {
		got := Classify(c.it, c.ctx)
		if got.Tier != c.want {
			t.Errorf("%s: got %s (%s) want %s", c.name, got.Tier, got.Reason, c.want)
		}
		if got.Reason == "" {
			t.Errorf("%s: no reason", c.name)
		}
	}
	cal := mkCtx(func(c *DigestContext) { c.MeetingTitles = []string{"Vantage standup", "FounderOS cohort call"} })
	for _, s := range []string{"Core Vantage", "Vinni Vantage"} {
		if got := Classify(mkItem(func(i *Item) { i.Sender = s }), cal).Tier; got == TierCall {
			t.Errorf("%s shares one company word with the calendar and must not be a call", s)
		}
	}
}

func digestItems() ([]Item, DigestContext) {
	items := []Item{
		mkItem(func(i *Item) {
			i.Sender, i.Title, i.TS = "noreply@vercel.com", "Deploy done", isoAt(digestNow.Add(-time.Second))
		}),
		mkItem(func(i *Item) {
			i.Sender, i.Title, i.TS = "Morgan Hale", "Re: audit — one department first", isoAt(digestNow.Add(-2*time.Hour))
		}),
		mkItem(func(i *Item) {
			i.Source, i.Sender, i.Preview, i.TS = "whatsapp", "Cohort Group (18)", "lol", isoAt(digestNow.Add(-3*time.Second))
		}),
		mkItem(func(i *Item) {
			i.Source, i.Sender, i.Preview, i.TS = "whatsapp", "Riley Novak", "question on the lesson", isoAt(digestNow.Add(-4*time.Second))
		}),
	}
	ctx := mkCtx(func(c *DigestContext) {
		c.MeetingTitles = []string{"Alex <> Morgan Hale"}
		c.Students = []string{"Riley Novak"}
	})
	return items, ctx
}

func tiersOf(d Digest) []Tier {
	var out []Tier
	for _, e := range d.Entries {
		out = append(out, e.Tier)
	}
	return out
}

func sendersOf(d Digest) []string {
	var out []string
	for _, e := range d.Entries {
		out = append(out, e.Sender)
	}
	return out
}

func TestBuildDigest(t *testing.T) {
	items, ctx := digestItems()
	d := BuildDigest(items, ctx)
	if got := tiersOf(d); !slices.Equal(got, []Tier{TierCall, TierPeople, TierGroup, TierNoise}) {
		t.Fatalf("order = %v", got)
	}
	if d.Counts[TierCall] != 1 || d.Counts[TierNoise] != 1 || d.Total != 4 || d.NeedsReply != 2 {
		t.Fatalf("counts = %+v total %d needsReply %d", d.Counts, d.Total, d.NeedsReply)
	}
	if d.WindowHours != 24 || d.GeneratedAt != "2026-08-18T09:00:00.000Z" {
		t.Fatalf("window %d generatedAt %s", d.WindowHours, d.GeneratedAt)
	}
	stale := mkItem(func(i *Item) { i.Sender, i.TS = "old@friend.com", isoAt(digestNow.Add(-40*time.Hour)) })
	if slices.Contains(sendersOf(BuildDigest(append(items, stale), ctx)), "old@friend.com") {
		t.Fatal("only the trailing window counts")
	}
	if !slices.Equal(TierOrder, []Tier{"call", "client", "people", "branddeal", "group", "noise"}) {
		t.Fatalf("tier order = %v", TierOrder)
	}
}

func TestUnsubscribeCandidatesNeverAPerson(t *testing.T) {
	items := []Item{
		mkItem(func(i *Item) { i.Sender = "news@saas.com" }),
		mkItem(func(i *Item) { i.Sender = "news@saas.com" }),
		mkItem(func(i *Item) { i.Sender = "noreply@other.com" }),
		mkItem(func(i *Item) { i.Sender, i.Title = "morgan@hale-advisory.example", "Re: audit" }),
	}
	out := UnsubscribeCandidates(items, mkCtx(nil))
	if out[0].Sender != "news@saas.com" || out[0].Count != 2 || out[0].Reason != "2 messages in 24h" {
		t.Fatalf("first = %+v", out[0])
	}
	for _, u := range out {
		if u.Sender == "morgan@hale-advisory.example" {
			t.Fatal("a person reached the unsubscribe list")
		}
	}
}

// ---- stacking -------------------------------------------------------------

var stackNow = time.Date(2026, 8, 19, 9, 0, 0, 0, time.UTC)

func stackCtx() DigestContext { return DigestContext{Now: stackNow} }

func stackItem(sender string, age time.Duration) Item {
	return Item{Source: "email", Title: "Subject", Preview: "body text", TS: isoAt(stackNow.Add(-age)), Sender: sender}
}

func mkEntry(over func(*Entry)) Entry {
	e := Entry{Tier: TierPeople, Rank: 2, Reason: "tagged", Source: "email", Sender: "old@example.com",
		Title: "Yesterday", Preview: "still waiting on you", TS: isoAt(stackNow.Add(-48 * time.Hour))}
	if over != nil {
		over(&e)
	}
	return e
}

func freshDigest() Digest {
	return BuildDigest([]Item{stackItem("new@example.com", time.Hour)}, stackCtx())
}

func TestStackCarriesUnclearedAndDropsCleared(t *testing.T) {
	out := StackDigest(freshDigest(), []Entry{mkEntry(func(e *Entry) { e.Sender = "unanswered@example.com" })}, nil, stackNow)
	if s := sendersOf(out); !slices.Contains(s, "new@example.com") || !slices.Contains(s, "unanswered@example.com") || out.Total != 2 {
		t.Fatalf("stacked = %v total %d", s, out.Total)
	}
	done := mkEntry(func(e *Entry) { e.Sender = "answered@example.com" })
	out = StackDigest(freshDigest(), []Entry{done}, []string{EntryKey(done.Source, done.Sender, done.TS)}, stackNow)
	if slices.Contains(sendersOf(out), "answered@example.com") || out.Total != 1 {
		t.Fatal("a cleared entry came back")
	}
}

func TestStackDedupesAndFreshWins(t *testing.T) {
	overlap := stackItem("overlap@example.com", 2*time.Hour)
	today := BuildDigest([]Item{overlap}, stackCtx())
	out := StackDigest(today, []Entry{mkEntry(func(e *Entry) { e.Sender, e.TS = "overlap@example.com", overlap.TS })}, nil, stackNow)
	if out.Total != 1 || out.Entries[0].Carried {
		t.Fatalf("entries = %+v", out.Entries)
	}
}

func TestStackOrdering(t *testing.T) {
	out := StackDigest(BuildDigest([]Item{stackItem("new@example.com", 6*time.Hour)}, stackCtx()), []Entry{mkEntry(nil)}, nil, stackNow)
	if out.Entries[0].Sender != "new@example.com" || !out.Entries[1].Carried {
		t.Fatalf("new must sit above held-over: %+v", out.Entries)
	}
	group := Item{Source: "whatsapp", Sender: "Some Group", Title: "Group chat", Preview: "lol", TS: isoAt(stackNow.Add(-time.Hour))}
	client := mkEntry(func(e *Entry) { e.Tier, e.Rank, e.Sender = TierClient, 1, "bigclient@example.com" })
	out = StackDigest(BuildDigest([]Item{group}, stackCtx()), []Entry{client}, nil, stackNow)
	if out.Entries[0].Sender != "bigclient@example.com" {
		t.Fatal("tier outranks freshness")
	}
}

func TestStackFirstSeenIsStable(t *testing.T) {
	first := mkEntry(func(e *Entry) { e.Sender = "chaser@example.com" })
	once := StackDigest(freshDigest(), []Entry{first}, nil, stackNow)
	find := func(d Digest) Entry {
		for _, e := range d.Entries {
			if e.Sender == "chaser@example.com" {
				return e
			}
		}
		t.Fatal("chaser missing")
		return Entry{}
	}
	if find(once).FirstSeenAt != first.TS {
		t.Fatal("firstSeenAt not set to the original ts")
	}
	twice := StackDigest(freshDigest(), once.Entries, nil, stackNow.Add(24*time.Hour))
	if find(twice).FirstSeenAt != first.TS {
		t.Fatal("firstSeenAt drifted")
	}
}

func TestStackCountsAndCap(t *testing.T) {
	out := StackDigest(freshDigest(), []Entry{mkEntry(func(e *Entry) { e.Tier, e.Rank = TierClient, 1 })}, nil, stackNow)
	if out.Total != len(out.Entries) || out.Counts[TierClient] != 1 || out.NeedsReply != 2 {
		t.Fatalf("counts %+v total %d needs %d", out.Counts, out.Total, out.NeedsReply)
	}
	ancient := mkEntry(func(e *Entry) {
		e.Sender, e.TS = "ancient@example.com", isoAt(stackNow.Add(-time.Duration(CarryMaxDays+1)*24*time.Hour))
	})
	recent := mkEntry(func(e *Entry) { e.Sender = "recent@example.com" })
	s := sendersOf(StackDigest(freshDigest(), []Entry{ancient, recent}, nil, stackNow))
	if slices.Contains(s, "ancient@example.com") || !slices.Contains(s, "recent@example.com") {
		t.Fatalf("carry cap: %v", s)
	}
	empty := StackDigest(BuildDigest(nil, stackCtx()), []Entry{mkEntry(func(e *Entry) { e.Sender = "waiting@example.com" })}, nil, stackNow)
	if empty.Total != 1 || empty.Entries[0].Sender != "waiting@example.com" {
		t.Fatal("an empty morning still shows the backlog")
	}
	if ReadRetentionDays <= CarryMaxDays {
		t.Fatal("read keys must outlive what they clear")
	}
}

// The payload the bridge stores must be the one FounderOS v1's /comms reads
// (and the ETL'd rows hold): same JSON keys, carry flags survive a round trip.
func TestPayloadRoundTripsInFounderosOSShape(t *testing.T) {
	stacked := StackDigest(BuildDigest([]Item{stackItem("new@example.com", time.Hour)}, stackCtx()),
		[]Entry{mkEntry(func(e *Entry) { e.Sender = "held@example.com" })}, nil, stackNow)
	raw, err := json.Marshal(RunResult{Digest: stacked, Sources: []SourceState{{Source: "email", OK: true, Count: 1}}})
	if err != nil {
		t.Fatal(err)
	}
	var generic map[string]any
	_ = json.Unmarshal(raw, &generic)
	dg := generic["digest"].(map[string]any)
	for _, k := range []string{"generatedAt", "windowHours", "entries", "unsubscribes", "counts", "total", "needsReply"} {
		if _, ok := dg[k]; !ok {
			t.Errorf("digest.%s missing", k)
		}
	}
	entries := dg["entries"].([]any)
	fresh := entries[0].(map[string]any)
	if _, has := fresh["carried"]; has {
		t.Error("a fresh entry must omit carried, as the TS does")
	}
	held := entries[1].(map[string]any)
	if held["carried"] != true || held["firstSeenAt"] == "" {
		t.Errorf("held = %v", held)
	}
	prev, err := PreviousEntries(raw)
	if err != nil || len(prev) != 2 {
		t.Fatalf("read back %d entries, err %v", len(prev), err)
	}
	next := StackDigest(BuildDigest(nil, stackCtx()), prev, nil, stackNow)
	if !slices.Contains(sendersOf(next), "held@example.com") {
		t.Fatal("cannot carry again from what was read back")
	}
}

func TestPreviousEntriesReadsTheTSPayload(t *testing.T) {
	// A payload exactly as FounderOS v1 wrote it (ETL'd rows carry these).
	ts := `{"digest":{"generatedAt":"2026-08-18T09:00:00.000Z","windowHours":24,"entries":[{"tier":"call","rank":0,"reason":"on your calendar — you have a call with them","source":"email","sender":"Morgan Hale","title":"Re","preview":"x","ts":"2026-08-18T08:00:00.000Z","replyTo":"y@z.com","account":"inbox-1"}],"unsubscribes":[],"counts":{"call":1,"client":0,"people":0,"branddeal":0,"group":0,"noise":0},"total":1,"needsReply":1},"sources":[]}`
	prev, err := PreviousEntries([]byte(ts))
	if err != nil || len(prev) != 1 || prev[0].Tier != TierCall || prev[0].ReplyTo != "y@z.com" || prev[0].Account != "inbox-1" {
		t.Fatalf("prev = %+v err %v", prev, err)
	}
	if _, err := PreviousEntries([]byte(`not json`)); err == nil {
		t.Fatal("a corrupt payload is an error, not an empty backlog")
	}
}
