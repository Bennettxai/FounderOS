package comms

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
)

// This file ports FounderOS v1 lib/comms-digest.ts: the 9am report's ranking of
// who the operator actually needs to respond to. Everything here is pure; the
// connectors are gathered in digest_run.go.
//
// The operator, 2026-08-18, verbatim priority: "people I have calls with are the
// most important ... WhatsApp chats are also pretty important from people like
// my students and my family ... Group chats are also gonna kind of not be as
// important ... The brand deal stuff is a second priority ... Messages from
// companies and software are not important. People are important."
//
// The rule that matters most: a real human NEVER falls into noise. Only
// detected bulk/automated senders do, because that tier is also the
// unsubscribe list.

// Tier is a digest priority tier.
type Tier string

const (
	TierCall      Tier = "call"
	TierClient    Tier = "client"
	TierPeople    Tier = "people"
	TierBrandDeal Tier = "branddeal"
	TierGroup     Tier = "group"
	TierNoise     Tier = "noise"
)

// TierOrder is the priority order, highest first; the index is the rank.
var TierOrder = []Tier{TierCall, TierClient, TierPeople, TierBrandDeal, TierGroup, TierNoise}

const (
	// WindowHours decides what is NEW in a report.
	WindowHours = 24
	// CarryMaxDays is how long an unanswered entry keeps riding along.
	CarryMaxDays = 30
	// ReadRetentionDays must exceed CarryMaxDays, or a cleared entry would
	// come back the morning after its read key was pruned.
	ReadRetentionDays = CarryMaxDays + 15
)

const isoMillis = "2006-01-02T15:04:05.000Z"

// Item is lib/comms.ts CommsItem: one message from one channel.
type Item struct {
	Source  string `json:"source"`
	Title   string `json:"title"`
	Preview string `json:"preview"`
	TS      string `json:"ts"`
	Sender  string `json:"sender,omitempty"`
	ReplyTo string `json:"replyTo,omitempty"`
	Account string `json:"account,omitempty"`
}

// DigestContext is what the ranking knows about the operator's world.
type DigestContext struct {
	// MeetingTitles are calendar event titles: "people I have calls with".
	MeetingTitles []string
	// ClientNames are paying customers (Stripe).
	ClientNames []string
	Students    []string
	Family      []string
	Now         time.Time
}

// Entry is one row of the report (DigestEntry).
type Entry struct {
	Tier        Tier   `json:"tier"`
	Rank        int    `json:"rank"`
	Reason      string `json:"reason"`
	Source      string `json:"source"`
	Sender      string `json:"sender"`
	Title       string `json:"title"`
	Preview     string `json:"preview"`
	TS          string `json:"ts"`
	ReplyTo     string `json:"replyTo,omitempty"`
	Account     string `json:"account,omitempty"`
	Carried     bool   `json:"carried,omitempty"`
	FirstSeenAt string `json:"firstSeenAt,omitempty"`
}

// Unsubscribe is a bulk sender worth leaving (UnsubscribeCandidate).
type Unsubscribe struct {
	Sender string `json:"sender"`
	Count  int    `json:"count"`
	Reason string `json:"reason"`
}

// Digest is the report (CommsDigest).
type Digest struct {
	GeneratedAt  string        `json:"generatedAt"`
	WindowHours  int           `json:"windowHours"`
	Entries      []Entry       `json:"entries"`
	Unsubscribes []Unsubscribe `json:"unsubscribes"`
	Counts       map[Tier]int  `json:"counts"`
	Total        int           `json:"total"`
	// NeedsReply is everything above group chatter.
	NeedsReply int `json:"needsReply"`
}

// Classification is a tier with its plain-language reason.
type Classification struct {
	Tier   Tier
	Reason string
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func norm(s string) string {
	return strings.TrimSpace(nonAlnum.ReplaceAllString(strings.ToLower(s), " "))
}

// stopWords never identify a PERSON. His own brands are here deliberately:
// the calendar is full of "Vantage".
var stopWords = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`the and com net org call with meeting sync chat zoom meet
		re fwd gmail group inc llc ltd team alex
		vantage founderos founder launchpad launchpads agency cohort core
		sales tfos ops support admin invited you new via linkedin`) {
		stopWords[w] = true
	}
}

func tokens(s string) []string {
	var out []string
	for _, t := range strings.Split(norm(s), " ") {
		if len(t) >= 3 && !stopWords[t] {
			out = append(out, t)
		}
	}
	return out
}

var (
	bulkLocalparts = regexp.MustCompile(`(?i)^(no-?reply|do-?not-?reply|notifications?|alerts?|mailer|bounce|postmaster|support|info|hello|hi|team|news|newsletter|updates?|billing|receipts?|digest|marketing|noreply)$`)
	localDigits    = regexp.MustCompile(`[._-]?\d+$`)
	bulkSubject    = regexp.MustCompile(`(?i)\b(unsubscribe|newsletter|digest|webinar|invoice|receipt|your order|deployment|build (succeeded|failed)|password reset|verify your|security alert|sign-?in|new login|promo|% off|sale ends|limited time)\b`)
	bulkBody       = regexp.MustCompile(`(?i)\b(unsubscribe|manage (your )?(email )?preferences|opt ?out|view (this|in) browser|you (are )?receiv(ed|ing) this)\b`)
	bulkRelay      = regexp.MustCompile(`(?i)\bvia (linkedin|substack|medium|quora|facebook|nextdoor|meetup|eventbrite)\b`)
	bulkOrg        = regexp.MustCompile(`(?i)\b(ticketmaster|nextdoor|eventbrite|stubhub|club|theat(er|re)|arena|casino|store|shop|dealership|bank|airlines?|hotel|tickets?)\b|^trending on\b|^the .* (team|crew)$|^mail delivery|daemon|postmaster`)
	bulkMachine    = regexp.MustCompile(`(?i)\[[\w.-]+/[\w.-]+\]|\b(run (failed|succeeded)|workflow run|build (failed|passed)|pipeline|pull request|commit|merged in|alert:|incident|uptime|delivery status notification|undeliverable|recap of your meeting)\b`)

	groupCount = regexp.MustCompile(`\(\s*\d+\s*\)\s*$`)
	groupLabel = regexp.MustCompile(`(?i)\b(group|chat|crew|team|family|squad|community)\b`)

	cohortRe = regexp.MustCompile(`(?i)\b(cohort|founderos|founder os|deploy day|railway|gbrain|g-brain|module|lesson|homework|onboarding|sop|workspace)\b`)
	// ventureNames are the operator's own brands. They label inboxes and
	// channels (the email connector titles an item "Launchpad Cohort — sender"),
	// so on their own they never make a message a cohort question.
	ventureNames = regexp.MustCompile(`(?i)\blaunchpad[ -]?cohort\b`)
	brandRe      = regexp.MustCompile(`(?i)\b(sponsor|sponsorship|partnership|collab|collaboration|brand deal|ambassador|paid promo|ugc|affiliate)\b`)
	replyRe      = regexp.MustCompile(`(?i)\b(re:|following up|circling back|as discussed|per our call|proposal|quote|invoice attached|contract|next steps?|kick ?off)\b`)
)

// IsBulkSender reports an automated or company sender. Conservative: a false
// positive hides a client. Both text fields are searched, because the email
// connector puts the subject in preview and Slack puts the channel in title.
func IsBulkSender(sender, title, preview string) bool {
	if at := strings.Index(sender, "@"); at >= 0 {
		local := sender[:at]
		if local != "" && bulkLocalparts.MatchString(localDigits.ReplaceAllString(local, "")) {
			return true
		}
	}
	if bulkRelay.MatchString(sender) || bulkOrg.MatchString(sender) {
		return true
	}
	text := title + "\n" + preview
	return bulkMachine.MatchString(text) || bulkBody.MatchString(text) || bulkSubject.MatchString(text)
}

// IsGroupChat spots WhatsApp/Slack group threads.
func IsGroupChat(it Item) bool {
	if groupCount.MatchString(it.Sender) {
		return true
	}
	return groupLabel.MatchString(it.Sender) && it.Source != "email"
}

// nameHits is strict on purpose: two shared tokens (first + last name) or one
// distinctive token of six characters or more (a surname).
func nameHits(needle string, haystack []string) bool {
	nt := tokens(needle)
	if len(nt) == 0 {
		return false
	}
	for _, entry := range haystack {
		h := norm(entry)
		hits, long := 0, false
		for _, t := range nt {
			if strings.Contains(h, t) {
				hits++
				if len(t) >= 6 {
					long = true
				}
			}
		}
		if hits >= 2 || long {
			return true
		}
	}
	return false
}

// Classify ranks one message.
func Classify(it Item, ctx DigestContext) Classification {
	sender := it.Sender
	group := IsGroupChat(it)
	bulk := IsBulkSender(sender, it.Title, it.Preview)
	text := it.Title + "\n" + it.Preview

	if !bulk && sender != "" && nameHits(sender, ctx.MeetingTitles) {
		return Classification{TierCall, "on your calendar — you have a call with them"}
	}
	if !bulk && sender != "" && nameHits(sender, ctx.ClientNames) {
		return Classification{TierClient, "current client or prospect"}
	}
	if !bulk && !group && replyRe.MatchString(text) && nameHits(sender, append(slices.Clone(ctx.ClientNames), ctx.MeetingTitles...)) {
		return Classification{TierClient, "replying on a proposal or deal thread"}
	}
	// Machine mail (a CI run on the FounderOS repo, an alert) can name the
	// product without anyone asking a question.
	if cohortRe.MatchString(ventureNames.ReplaceAllString(text, " ")) && !bulkMachine.MatchString(text) {
		return Classification{TierPeople, "cohort question"}
	}
	if sender != "" && nameHits(sender, ctx.Students) {
		return Classification{TierPeople, "your student"}
	}
	if sender != "" && nameHits(sender, ctx.Family) {
		return Classification{TierPeople, "family"}
	}
	if group {
		return Classification{TierGroup, "group chat"}
	}
	if !bulk && brandRe.MatchString(text) {
		return Classification{TierBrandDeal, "brand deal or partnership"}
	}
	if bulk {
		return Classification{TierNoise, "automated or marketing sender"}
	}
	return Classification{TierPeople, "a person wrote to you"}
}

// UnsubscribeCandidates lists bulk senders in the window, noisiest first.
func UnsubscribeCandidates(items []Item, ctx DigestContext) []Unsubscribe {
	counts := map[string]int{}
	var order []string
	for _, it := range items {
		if Classify(it, ctx).Tier != TierNoise || it.Sender == "" {
			continue
		}
		if counts[it.Sender] == 0 {
			order = append(order, it.Sender)
		}
		counts[it.Sender]++
	}
	out := make([]Unsubscribe, 0, len(order))
	for _, s := range order {
		reason := "automated sender"
		if counts[s] > 1 {
			reason = fmt.Sprintf("%d messages in 24h", counts[s])
		}
		out = append(out, Unsubscribe{Sender: s, Count: counts[s], Reason: reason})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return localeLess(out[i].Sender, out[j].Sender)
	})
	return out
}

// localeLess approximates String.localeCompare: case-insensitive first.
func localeLess(a, b string) bool {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	if la != lb {
		return la < lb
	}
	return a < b
}

func parseTS(s string) (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, s)
	return t, err == nil
}

func rankOf(t Tier) int { return slices.Index(TierOrder, t) }

// BuildDigest ranks the trailing window of items.
func BuildDigest(items []Item, ctx DigestContext) Digest {
	cutoff := ctx.Now.Add(-WindowHours * time.Hour)
	latest := ctx.Now.Add(time.Minute)
	var fresh []Item
	for _, it := range items {
		t, ok := parseTS(it.TS)
		if ok && !t.Before(cutoff) && !t.After(latest) {
			fresh = append(fresh, it)
		}
	}
	entries := make([]Entry, 0, len(fresh))
	for _, it := range fresh {
		c := Classify(it, ctx)
		sender := it.Sender
		if sender == "" {
			sender = "unknown"
		}
		entries = append(entries, Entry{
			Tier: c.Tier, Rank: rankOf(c.Tier), Reason: c.Reason, Source: it.Source, Sender: sender,
			Title: it.Title, Preview: it.Preview, TS: it.TS, ReplyTo: it.ReplyTo, Account: it.Account,
		})
	}
	d := Digest{
		GeneratedAt:  ctx.Now.UTC().Format(isoMillis),
		WindowHours:  WindowHours,
		Entries:      orderEntries(entries),
		Unsubscribes: UnsubscribeCandidates(fresh, ctx),
	}
	d.tally()
	return d
}

// orderEntries: tier first, then NEW above held-over, then newest.
func orderEntries(in []Entry) []Entry {
	out := slices.Clone(in)
	if out == nil {
		out = []Entry{}
	}
	ms := func(s string) int64 {
		t, _ := parseTS(s)
		return t.UnixMilli()
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Rank != b.Rank {
			return a.Rank < b.Rank
		}
		if a.Carried != b.Carried {
			return !a.Carried
		}
		return ms(a.TS) > ms(b.TS)
	})
	return out
}

// tally recounts so the header never disagrees with the rows.
func (d *Digest) tally() {
	d.Counts = map[Tier]int{}
	for _, t := range TierOrder {
		d.Counts[t] = 0
	}
	for _, e := range d.Entries {
		d.Counts[e.Tier]++
	}
	d.Total = len(d.Entries)
	d.NeedsReply = d.Counts[TierCall] + d.Counts[TierClient] + d.Counts[TierPeople] + d.Counts[TierBrandDeal]
}

// StackDigest puts this morning's report on top of what is left of the older
// ones: an entry leaves only when the operator cleared it, or after CarryMaxDays.
func StackDigest(fresh Digest, previous []Entry, cleared []string, now time.Time) Digest {
	clearedSet := map[string]bool{}
	for _, k := range cleared {
		clearedSet[k] = true
	}
	seen := map[string]bool{}
	for _, e := range fresh.Entries {
		seen[EntryKey(e.Source, e.Sender, e.TS)] = true
	}
	oldest := now.Add(-CarryMaxDays * 24 * time.Hour)
	var carried []Entry
	for _, e := range previous {
		key := EntryKey(e.Source, e.Sender, e.TS)
		if clearedSet[key] || seen[key] {
			continue
		}
		t, ok := parseTS(e.TS)
		if !ok || t.Before(oldest) {
			continue
		}
		seen[key] = true
		e.Carried = true
		if e.FirstSeenAt == "" {
			e.FirstSeenAt = e.TS
		}
		carried = append(carried, e)
	}
	out := fresh
	out.Entries = orderEntries(append(slices.Clone(fresh.Entries), carried...))
	out.tally()
	return out
}

// EntryKey is the stable id of one message: channel + sender + timestamp,
// deliberately not the tier. It matches the TS entryKey and the keys stored in
// founderos_digest_reads.
func EntryKey(source, sender, ts string) string { return source + "|" + sender + "|" + ts }

// PreviousEntries reads the entries out of a stored digest payload. A corrupt
// payload is an error, never an empty backlog.
func PreviousEntries(payload []byte) ([]Entry, error) {
	var r struct {
		Digest *struct {
			Entries []Entry `json:"entries"`
		} `json:"digest"`
	}
	if err := json.Unmarshal(payload, &r); err != nil {
		return nil, fmt.Errorf("stored digest unreadable: %w", err)
	}
	if r.Digest == nil {
		return []Entry{}, nil
	}
	return r.Digest.Entries, nil
}
