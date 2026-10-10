package agentspage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/pages/osdata"
)

// Agent deliverables: Paperclip agents write real files into their
// workspace `deliverables/` folders on the board host. The directory is
// PAPERCLIP_WORKSPACES_DIR; a box without it has no deliverables to show
// (honest empty), never an error.

var DeliverableMIME = map[string]string{
	".pdf": "application/pdf", ".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif",
	".svg": "image/svg+xml", ".html": "text/html; charset=utf-8", ".md": "text/markdown; charset=utf-8",
	".txt": "text/plain; charset=utf-8", ".csv": "text/csv; charset=utf-8", ".json": "application/json",
	".zip": "application/zip", ".mp4": "video/mp4", ".mp3": "audio/mpeg", ".webp": "image/webp",
}

type Deliverable struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Workspace  string   `json:"workspace"`
	SizeBytes  int64    `json:"sizeBytes"`
	ModifiedAt string   `json:"modifiedAt"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	AlsoAs     []string `json:"alsoAs,omitempty"`
}

const headBytes = 4096

var textual = map[string]bool{".md": true, ".txt": true, ".json": true, ".csv": true, ".html": true, ".svg": true}

func readHead(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, headBytes)
	n, _ := io.ReadFull(f, buf)
	return string(buf[:n])
}

var extRE = regexp.MustCompile(`(?i)\.[a-z0-9]+$`)

func stem(name string) string { return extRE.ReplaceAllString(name, "") }

// CollapseSiblings keeps one row for work written as several formats in the
// SAME workspace, preferring the readable one; order is preserved.
func CollapseSiblings(rows []Deliverable) []Deliverable {
	readable := []string{".md", ".txt", ".html", ".json", ".csv"}
	rank := func(name string) int {
		ext := strings.ToLower(filepath.Ext(name))
		for i, r := range readable {
			if r == ext {
				return i
			}
		}
		return len(readable)
	}
	groups := map[string][]Deliverable{}
	var keys []string
	for _, r := range rows {
		k := r.Workspace + "/" + stem(r.Name)
		if _, seen := groups[k]; !seen {
			keys = append(keys, k)
		}
		groups[k] = append(groups[k], r)
	}
	out := make([]Deliverable, 0, len(keys))
	for _, k := range keys {
		g := groups[k]
		best := 0
		for i := range g {
			if rank(g[i].Name) < rank(g[best].Name) {
				best = i
			}
		}
		w := g[best]
		for i := range g {
			if i != best {
				w.AlsoAs = append(w.AlsoAs, strings.TrimPrefix(strings.ToLower(filepath.Ext(g[i].Name)), "."))
			}
		}
		out = append(out, w)
	}
	return out
}

// ListDeliverables walks <base>/<workspace>/deliverables/*, newest first,
// siblings collapsed, capped at 200. A missing base is an empty list.
func ListDeliverables(base string) []Deliverable {
	out := []Deliverable{}
	if base == "" {
		return out
	}
	workspaces, err := os.ReadDir(base)
	if err != nil {
		return out
	}
	for _, ws := range workspaces {
		dir := filepath.Join(base, ws.Name(), "deliverables")
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, f := range files {
			full := filepath.Join(dir, f.Name())
			st, err := os.Stat(full)
			if err != nil || !st.Mode().IsRegular() {
				continue
			}
			var b Brief
			if textual[strings.ToLower(filepath.Ext(f.Name()))] {
				b = BriefFrom(f.Name(), readHead(full))
			} else {
				b = Brief{Title: TitleFromFilename(f.Name())}
			}
			out = append(out, Deliverable{
				ID: ws.Name() + "/" + f.Name(), Name: f.Name(), Workspace: ws.Name(), SizeBytes: st.Size(),
				ModifiedAt: isoMillis(st.ModTime().Round(time.Millisecond)), Title: b.Title, Summary: b.Summary,
			})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ModifiedAt > out[j].ModifiedAt })
	out = CollapseSiblings(out)
	if len(out) > 200 {
		out = out[:200]
	}
	return out
}

// ResolveDeliverable maps `<workspace>/<file>` to a path inside a
// deliverables folder, or "" for anything that tries to escape it.
func ResolveDeliverable(id, base string) string {
	parts := strings.Split(id, "/")
	if base == "" || len(parts) != 2 {
		return ""
	}
	for _, p := range parts {
		if p == "" || p == "." || p == ".." || strings.Contains(p, `\`) {
			return ""
		}
	}
	root, err := filepath.Abs(base)
	if err != nil {
		return ""
	}
	full := filepath.Join(root, parts[0], "deliverables", parts[1])
	if !strings.HasPrefix(full, root+string(filepath.Separator)) {
		return ""
	}
	return full
}

// ---- preview (lib/deliverable-preview.ts) -----------------------------------

const PreviewMaxChars = 200_000

var (
	textExts  = map[string]bool{".md": true, ".txt": true, ".json": true, ".csv": true, ".log": true, ".yml": true, ".yaml": true, ".html": true, ".htm": true}
	imageExts = map[string]bool{".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".svg": true}
)

// PreviewKind is text | image | pdf | binary (svg renders as an image).
func PreviewKind(name string) string {
	i := strings.LastIndex(name, ".")
	ext := ""
	if i > 0 {
		ext = strings.ToLower(name[i:])
	}
	switch {
	case imageExts[ext]:
		return "image"
	case textExts[ext]:
		return "text"
	case ext == ".pdf":
		return "pdf"
	}
	return "binary"
}

func ClampPreview(text string, max int) (string, bool) {
	r := []rune(text)
	if len(r) <= max {
		return text, false
	}
	return string(r[:max]), true
}

// ---- grouping (groupDeliverables) --------------------------------------------

type DeliverableItem struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Kind       string   `json:"kind"` // file | link
	URL        *string  `json:"url"`
	Meta       string   `json:"meta"`
	ModifiedAt string   `json:"modifiedAt"`
	SizeBytes  *int64   `json:"sizeBytes"`
	AccessCode string   `json:"accessCode"`
	Title      string   `json:"title"`
	Summary    string   `json:"summary"`
	AlsoAs     []string `json:"alsoAs,omitempty"`
	Revision   string   `json:"revision"`
}

type DeliverableGroup struct {
	Name  string            `json:"name"`
	Items []DeliverableItem `json:"items"`
}

var proposalBrands = []struct{ id, folder string }{
	{"vantage", "Vantage proposals"},
	{"launchpad-cohort", "Launchpad Cohort proposals"},
}

func usd(n float64) string {
	whole := strconv.FormatFloat(n, 'f', -1, 64)
	intPart, frac, _ := strings.Cut(whole, ".")
	var b strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 && c != '-' {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	if frac != "" {
		b.WriteString("." + frac)
	}
	return "$" + b.String()
}

// RevisionOf fingerprints an item's mutable state, so a decision reopens
// when the work changes (lib/deliverable-revision.ts). The bridge never
// exposes proposal access codes (they live in credential_vault), so the
// link fingerprint carries an empty code.
func RevisionOf(it DeliverableItem) string {
	if it.Kind == "link" {
		u := ""
		if it.URL != nil {
			u = *it.URL
		}
		return strings.Join([]string{"link", it.Meta, u, it.AccessCode}, "|")
	}
	size := ""
	if it.SizeBytes != nil {
		size = strconv.FormatInt(*it.SizeBytes, 10)
	}
	return strings.Join([]string{"file", it.ModifiedAt, size}, "|")
}

// GroupDeliverables folds proposals and agent files into ordered folders;
// empty folders are omitted.
func GroupDeliverables(files []Deliverable, proposals []osdata.Proposal) []DeliverableGroup {
	groups := []DeliverableGroup{}
	for _, brand := range proposalBrands {
		var items []DeliverableItem
		for _, p := range proposals {
			if p.Brand != brand.id {
				continue
			}
			meta := []string{}
			if p.Status != "" {
				meta = append(meta, p.Status)
			}
			if p.AmountUSD != nil {
				meta = append(meta, usd(*p.AmountUSD))
			}
			u := p.URL
			it := DeliverableItem{ID: "proposal:" + p.ID, Name: p.Client, Kind: "link", URL: &u, Meta: strings.Join(meta, " · "),
				ModifiedAt: p.CreatedAt, Title: p.Client}
			it.Revision = RevisionOf(it)
			items = append(items, it)
		}
		if len(items) > 0 {
			groups = append(groups, DeliverableGroup{Name: brand.folder, Items: items})
		}
	}
	if len(files) > 0 {
		items := make([]DeliverableItem, len(files))
		for i, f := range files {
			size := f.SizeBytes
			meta := f.Workspace
			if len(meta) > 8 {
				meta = meta[:8]
			}
			items[i] = DeliverableItem{ID: f.ID, Name: f.Name, Kind: "file", Meta: meta, ModifiedAt: f.ModifiedAt, SizeBytes: &size,
				Title: f.Title, Summary: f.Summary, AlsoAs: f.AlsoAs}
			items[i].Revision = RevisionOf(items[i])
		}
		groups = append(groups, DeliverableGroup{Name: "Agent files", Items: items})
	}
	return groups
}

// ---- approvals (lib/board-approvals.ts) --------------------------------------

type Classified struct {
	DeliverableItem
	Ask       string  `json:"ask"`
	NeedsYou  bool    `json:"needsYou"`
	Deadline  *string `json:"deadline"`
	Overdue   bool    `json:"overdue"`
	Label     string  `json:"label"`
	Action    string  `json:"action"`
	Person    bool    `json:"person"`
	Glyph     string  `json:"glyph"`
	GlyphTone string  `json:"glyphTone"`
	Why       string  `json:"why"`
}

var actionText = map[string]string{
	"staged":   "The agent already wrote this. Approve to send it. Dismiss to bin it.",
	"decision": "Only you can make this call. Approve to go ahead. Dismiss to drop it.",
	"gate":     "Work is paused until you answer. Approve to unblock it. Dismiss to keep it closed.",
	"request":  "Someone asked you for something. Approve to act on it. Dismiss to decline.",
	"draft":    "A draft is waiting on you. Approve to publish it. Dismiss to send it back.",
	"done":     "Finished work. Nothing is being asked of you.",
	"output":   "Reference output. Nothing is being asked of you.",
}

var whyText = map[string]string{
	"staged": "a reply is written and unsent", "decision": "nobody else can make this call", "gate": "work is paused behind it",
	"request": "someone asked you directly", "draft": "a draft is waiting to go out", "done": "finished, kept for the record", "output": "reference only",
}

var glyphs = map[string]string{"staged": "✉", "decision": "!", "gate": "◐", "request": "!", "draft": "!", "done": "✓", "output": "·"}

var (
	moneyRE    = regexp.MustCompile(`(?i)(invoice|refund|payment|stripe|pricing|quote|contract|proposal)`)
	doneRE     = regexp.MustCompile(`(?i)(UNBLOCKED|RESOLVED|FINAL|COMPLETE)`)
	deadlineRE = regexp.MustCompile(`(?i)DELIVER-BEFORE-(\d{2})(\d{2})Z`)
	askRules   = []struct {
		re         *regexp.Regexp
		kind, text string
	}{
		{regexp.MustCompile(`(?i)STAGED`), "staged", "staged · send it"},
		{regexp.MustCompile(`(?i)\bdecision\b|-decision-`), "decision", "your call"},
		{regexp.MustCompile(`(?i)-gate-|\bgate\b`), "gate", "gate"},
		{regexp.MustCompile(`(?i)-request-|\brequest\b`), "request", "request"},
		{regexp.MustCompile(`(?i)-draft-|\bdraft\b`), "draft", "draft to approve"},
	}
)

func deadlineFrom(name, modifiedAt string) *string {
	m := deadlineRE.FindStringSubmatch(name)
	if m == nil {
		return nil
	}
	day, err := time.Parse(time.RFC3339Nano, modifiedAt)
	if err != nil {
		return nil
	}
	h, _ := strconv.Atoi(m[1])
	mi, _ := strconv.Atoi(m[2])
	d := day.UTC()
	s := isoMillis(time.Date(d.Year(), d.Month(), d.Day(), h, mi, 0, 0, time.UTC))
	return &s
}

// Classify reads what an agent file is asking of the operator from its name.
func Classify(it DeliverableItem, now time.Time) Classified {
	c := Classified{DeliverableItem: it, Ask: "output", Label: "output"}
	c.Deadline = deadlineFrom(it.Name, it.ModifiedAt)
	done := doneRE.MatchString(it.Name)
	if done {
		c.Ask, c.Label = "done", "done"
	} else {
		for _, r := range askRules {
			if r.re.MatchString(it.Name) {
				c.Ask, c.Label = r.kind, r.text
				c.Person = r.kind == "staged" || r.kind == "request"
				break
			}
		}
	}
	if c.Ask == "output" && c.Deadline != nil && !done {
		c.Ask, c.Label = "request", "due"
	}
	c.NeedsYou = !done && c.Ask != "output"
	if c.Deadline != nil && !done {
		if dl, err := time.Parse(time.RFC3339Nano, *c.Deadline); err == nil && dl.Before(now) {
			c.Overdue = true
		}
	}
	money := moneyRE.MatchString(it.Name) || moneyRE.MatchString(it.Title)
	c.Action = actionText[c.Ask]
	c.Glyph = glyphs[c.Ask]
	if money && c.NeedsYou {
		c.Glyph = "$"
	}
	switch {
	case c.Overdue:
		c.GlyphTone = "err"
	case money && c.NeedsYou:
		c.GlyphTone = "warn"
	case c.Ask == "staged":
		c.GlyphTone = "ok"
	case c.Ask == "gate" || c.Ask == "decision":
		c.GlyphTone = "warn"
	default:
		c.GlyphTone = "dim"
	}
	c.Why = whyText[c.Ask]
	if c.Overdue {
		c.Why = "the deadline the agent set has passed"
	}
	return c
}

// NeedsYou is the queue asking for his decision: overdue first, then work
// for a person, then anything with a deadline, newest first inside a rank.
func NeedsYou(items []DeliverableItem, now time.Time) []Classified {
	out := []Classified{}
	for _, it := range items {
		if c := Classify(it, now); c.NeedsYou {
			out = append(out, c)
		}
	}
	rank := func(c Classified) int {
		switch {
		case c.Overdue:
			return 0
		case c.Person:
			return 1
		case c.Deadline != nil:
			return 2
		}
		return 3
	}
	sort.SliceStable(out, func(i, j int) bool {
		if ri, rj := rank(out[i]), rank(out[j]); ri != rj {
			return ri < rj
		}
		return out[i].ModifiedAt > out[j].ModifiedAt
	})
	return out
}

// ---- decisions (lib/deliverable-decisions.ts) ---------------------------------

type Decided struct {
	Item     DeliverableItem `json:"item"`
	Decision osdata.Decision `json:"decision"`
}

// DecisionMap keeps the newest call per id.
func DecisionMap(list []osdata.Decision) map[string]osdata.Decision {
	out := map[string]osdata.Decision{}
	for _, d := range list {
		if prev, ok := out[d.ID]; !ok || d.DecidedAt >= prev.DecidedAt {
			out[d.ID] = d
		}
	}
	return out
}

// PartitionByDecision splits his open calls from handled ones; a decision
// made against an older revision reopens the item.
func PartitionByDecision(items []DeliverableItem, decisions []osdata.Decision) (open []DeliverableItem, decided []Decided) {
	m := DecisionMap(decisions)
	open, decided = []DeliverableItem{}, []Decided{}
	for _, it := range items {
		d, ok := m[it.ID]
		rev := it.Revision
		if rev == "" {
			rev = RevisionOf(it)
		}
		if !ok || (d.DecidedRevision != "" && d.DecidedRevision != rev) {
			open = append(open, it)
			continue
		}
		decided = append(decided, Decided{Item: it, Decision: d})
	}
	return open, decided
}

// ---- brief (lib/deliverable-brief.ts) -----------------------------------------

type Brief struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

const (
	titleMax   = 120
	summaryMax = 200
)

var proseKeys = []string{"body", "comment", "text", "content", "markdown", "description"}

var (
	imgRE      = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	linkRE     = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	codeRE     = regexp.MustCompile("`{1,3}([^`]*)`{1,3}")
	boldRE     = regexp.MustCompile(`\*\*([^*]*)\*\*`)
	emRE       = regexp.MustCompile(`(^|[\s(])[*_]([^*_]+)[*_]`)
	leadRE     = regexp.MustCompile(`^(?:\s*[>#]+\s*)+`)
	spaceRE    = regexp.MustCompile(`\s+`)
	sentenceRE = regexp.MustCompile(`^(.*?[.!?])(\s|$)`)
	tableRE    = regexp.MustCompile(`^[-|: ]+$`)
	bulletRE   = regexp.MustCompile(`^[-*+]\s`)
	numberedRE = regexp.MustCompile(`^\d+\.\s`)
	greetRE    = regexp.MustCompile(`(?i)^(hi|hey|hello|dear|good (morning|afternoon|evening))\b[^.!?]{0,40},?$`)
	headingRE  = regexp.MustCompile(`^#{1,6}\s+\S`)
)

func plain(md string) string {
	s := strings.TrimSpace(md)
	s = imgRE.ReplaceAllString(s, "")
	s = linkRE.ReplaceAllString(s, "$1")
	s = codeRE.ReplaceAllString(s, "$1")
	s = boldRE.ReplaceAllString(s, "$1")
	s = emRE.ReplaceAllString(s, "$1$2")
	s = leadRE.ReplaceAllString(s, "")
	s = spaceRE.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimRight(string(r[:n-1]), " \t\n") + "…"
}

func furniture(line string) bool {
	t := strings.TrimSpace(line)
	return t == "" || strings.HasPrefix(t, "|") || tableRE.MatchString(t) || strings.HasPrefix(t, "```") ||
		strings.HasPrefix(t, "---") || strings.HasPrefix(t, "===") || bulletRE.MatchString(t) || numberedRE.MatchString(t)
}

var (
	fnStaged   = regexp.MustCompile(`(?i)^STAGED-`)
	fnDeliver  = regexp.MustCompile(`(?i)-?DELIVER-BEFORE-\d{4}Z`)
	fnDate     = regexp.MustCompile(`(?i)-?\d{4}-\d{2}-\d{2}(T\d{4}Z)?`)
	fnSuffix   = regexp.MustCompile(`(?i)-(COMMENT|PATCH|FINAL|RESOLVED|UNBLOCKED|VERIFIED)$`)
	fnTicket   = regexp.MustCompile(`(?i)^ben\d+-`)
	fnDashes   = regexp.MustCompile(`[-_]+`)
	jsonKeyFmt = `"%s"\s*:\s*"`
)

// TitleFromFilename turns agent shorthand into a readable fallback title.
func TitleFromFilename(name string) string {
	s := stem(name)
	s = fnStaged.ReplaceAllString(s, "")
	s = fnDeliver.ReplaceAllString(s, "")
	s = fnDate.ReplaceAllString(s, "")
	s = fnSuffix.ReplaceAllString(s, "")
	s = fnTicket.ReplaceAllString(s, "")
	s = strings.TrimSpace(fnDashes.ReplaceAllString(s, " "))
	if s == "" {
		return name
	}
	return s
}

// scrapeJSONString pulls a string value out of possibly-truncated JSON.
func scrapeJSONString(head, key string) string {
	loc := regexp.MustCompile(fmt.Sprintf(jsonKeyFmt, regexp.QuoteMeta(key))).FindStringIndex(head)
	if loc == nil {
		return ""
	}
	var raw strings.Builder
	for i := loc[1]; i < len(head); i++ {
		ch := head[i]
		if ch == '\\' && i+1 < len(head) {
			raw.WriteString(head[i : i+2])
			i++
			continue
		}
		if ch == '"' {
			break
		}
		raw.WriteByte(ch)
	}
	r := raw.String()
	for _, cut := range []int{0, 1, 2} {
		if cut > len(r) {
			break
		}
		var out string
		if json.Unmarshal([]byte(`"`+r[:len(r)-cut]+`"`), &out) == nil && strings.TrimSpace(out) != "" {
			return out
		}
	}
	return ""
}

func unwrap(name, head string) (string, string) {
	if !strings.HasSuffix(strings.ToLower(name), ".json") {
		return "", head
	}
	var rec map[string]any
	if json.Unmarshal([]byte(head), &rec) == nil && rec != nil {
		title := ""
		if s, ok := rec["title"].(string); ok {
			title = strings.TrimSpace(s)
		}
		for _, k := range proseKeys {
			if s, ok := rec[k].(string); ok && strings.TrimSpace(s) != "" {
				return title, s
			}
		}
		return title, ""
	}
	title := scrapeJSONString(head, "title")
	for _, k := range proseKeys {
		if doc := scrapeJSONString(head, k); doc != "" {
			return title, doc
		}
	}
	return title, ""
}

// BriefFrom reads a readable title and one line of summary out of a file head.
func BriefFrom(name, head string) Brief {
	jsonTitle, doc := unwrap(name, head)
	lines := strings.Split(doc, "\n")
	isHeading := func(l string) bool { return headingRE.MatchString(strings.TrimSpace(l)) }
	headingAt, proseAt := -1, -1
	var prose []string
	for i, l := range lines {
		if headingAt == -1 && isHeading(l) {
			headingAt = i
		}
		if !furniture(l) && !isHeading(l) && !greetRE.MatchString(strings.TrimSpace(l)) {
			if proseAt == -1 {
				proseAt = i
			}
			if p := plain(l); p != "" {
				prose = append(prose, p)
			}
		}
	}
	title, rest := "", ""
	switch {
	case jsonTitle != "":
		title, rest = plain(jsonTitle), strings.Join(prose, " ")
	case headingAt != -1 && (proseAt == -1 || headingAt < proseAt):
		title = plain(lines[headingAt])
		var keep []string
		for _, p := range prose {
			if p != title {
				keep = append(keep, p)
			}
		}
		rest = strings.Join(keep, " ")
	default:
		lede := ""
		if len(prose) > 0 {
			lede = prose[0]
		}
		title = lede
		if m := sentenceRE.FindStringSubmatch(lede); m != nil {
			title = m[1]
		}
		title = strings.TrimSpace(title)
		tail := ""
		if len(prose) > 1 {
			tail = strings.Join(prose[1:], " ")
		}
		rest = strings.TrimSpace(lede[min(len(title), len(lede)):] + " " + tail)
	}
	if title == "" {
		title = TitleFromFilename(name)
	}
	return Brief{Title: clip(title, titleMax), Summary: clip(rest, summaryMax)}
}
