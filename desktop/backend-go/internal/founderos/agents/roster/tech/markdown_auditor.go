package tech

import (
	"context"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/optimalengine"
)

// StaleAfter is how long a workspace may go without a new or changed source
// before the auditor calls it stale.
const StaleAfter = 30 * 24 * time.Hour

// MarkdownAuditor audits the knowledge the engines hold: link health,
// orphans, duplicate and missing titles, stale and missing workspaces, and
// whether the full-text index still matches the stored sources.
type MarkdownAuditor struct {
	Engines EngineSource
	Client  *http.Client
	Now     func() time.Time
}

func (a *MarkdownAuditor) Meta() agents.Meta {
	return agents.Meta{
		ID:           "markdown-auditor",
		Name:         "Markdown Auditor",
		Description:  "Link health, orphans, duplicate titles and store-vs-index drift across the knowledge base.",
		DepartmentID: "dept-tech",
	}
}

type AuditFinding struct {
	Kind     string `json:"kind"`
	Severity string `json:"severity"` // warn | err
	Detail   string `json:"detail"`
	Count    int    `json:"count,omitempty"`
}

type WorkspaceAudit struct {
	Workspace string `json:"workspace"`
	Engine    string `json:"engine"`
	Sources   int    `json:"sources"`
	Links     int    `json:"links"`
	Resolved  int    `json:"resolved"`
	Orphans   int    `json:"orphans"`
	Newest    string `json:"newest,omitempty"`

	broken, orphans, untitled []string
	dupes                     []string
	stale                     bool
	pages                     []page
}

type KnowledgeAudit struct {
	Sources    int                 `json:"sources"`
	Links      int                 `json:"links"`
	Resolved   int                 `json:"resolved"`
	Broken     int                 `json:"broken"`
	Orphans    int                 `json:"orphans"`
	Workspaces []WorkspaceAudit    `json:"workspaces"`
	Unaudited  map[string][]string `json:"unaudited,omitempty"` // unstaged engine → workspaces
	Findings   []AuditFinding      `json:"findings"`
	Summary    string              `json:"summary"`
}

// engineAudit is what one engine contributed.
type engineAudit struct {
	workspaces []WorkspaceAudit
	findings   []AuditFinding
}

func (a *MarkdownAuditor) Run(ctx context.Context) (agents.Result, error) {
	v, err := resolveView(a.Engines)
	if err != nil {
		return agents.Result{OK: false, Summary: err.Error()}, nil
	}
	if len(v.engines) == 0 {
		return agents.Result{OK: false, Summary: noEnginesSummary}, nil
	}
	now := time.Now
	if a.Now != nil {
		now = a.Now
	}
	client := newClient(a.Client)

	per := make([]engineAudit, len(v.engines))
	var wg sync.WaitGroup
	for i, e := range v.engines {
		wg.Add(1)
		go func(i int, e optimalengine.Engine) {
			defer wg.Done()
			per[i] = auditEngine(ctx, engineAPI{e: e, client: client}, v.homed[e.Name], now())
		}(i, e)
	}
	wg.Wait()

	audit := &KnowledgeAudit{Unaudited: v.unstaged}
	var hard []AuditFinding
	for _, p := range per {
		audit.Workspaces = append(audit.Workspaces, p.workspaces...)
		hard = append(hard, p.findings...)
	}
	linkPass(audit.Workspaces)
	audit.Findings = append(hard, workspaceFindings(audit)...)
	reachable := true
	for _, f := range hard {
		if f.Kind == "unreachable" {
			reachable = false
		}
	}
	if audit.Sources == 0 && reachable {
		audit.Findings = append(audit.Findings, AuditFinding{Kind: "empty-store", Severity: "err",
			Detail: "no sources in any audited workspace: the import never ran or the engines were reset"})
	}
	sortFindings(audit.Findings)

	audit.Summary = knowledgeSummary(audit, v)
	var errs []string
	for _, f := range audit.Findings {
		if f.Severity == "err" {
			errs = append(errs, f.Detail)
		}
	}
	summary := audit.Summary
	if len(errs) > 0 {
		summary += " · " + strings.Join(errs, " | ")
	}
	return agents.Result{OK: len(errs) == 0 && audit.Sources > 0, Summary: summary, Data: audit}, nil
}

func auditEngine(ctx context.Context, api engineAPI, homed []string, now time.Time) engineAudit {
	var out engineAudit
	name := api.e.Name
	present, err := api.workspaces(ctx)
	if err != nil {
		out.findings = append(out.findings, AuditFinding{Kind: "unreachable", Severity: "err",
			Detail: fmt.Sprintf("%s unreachable, %s unaudited: %v", name, strings.Join(homed, ", "), err)})
		return out
	}
	if st, err := api.audit(ctx); err != nil {
		out.findings = append(out.findings, AuditFinding{Kind: "audit-unreadable", Severity: "warn",
			Detail: fmt.Sprintf("%s storage audit unreadable: %v", name, err)})
	} else {
		if c, found := st.check("fts_parity"); found && !c.OK {
			out.findings = append(out.findings, AuditFinding{Kind: "index-drift", Severity: "err",
				Detail: fmt.Sprintf("%s: the full-text index no longer matches the stored sources (%s), so search is not answering from what is stored", name, c.detail())})
		}
		if c, found := st.check("workspace_scope"); found && !c.OK {
			out.findings = append(out.findings, AuditFinding{Kind: "legacy-scope", Severity: "warn",
				Detail: fmt.Sprintf("%s holds rows in the retired default workspace (%s)", name, c.detail())})
		}
	}
	for _, slug := range homed {
		if !present[slug] {
			out.findings = append(out.findings, AuditFinding{Kind: "missing-workspace", Severity: "err",
				Detail: fmt.Sprintf("%s is routed to %s by the topology but does not exist there", slug, name)})
			continue
		}
		sigs, err := api.signals(ctx, slug)
		if err != nil {
			out.findings = append(out.findings, AuditFinding{Kind: "unreachable", Severity: "err",
				Detail: fmt.Sprintf("%s on %s could not be exported: %v", slug, name, err)})
			continue
		}
		out.workspaces = append(out.workspaces, auditWorkspace(slug, name, sigs, now))
	}
	return out
}

// ---- one workspace -----------------------------------------------------------

var (
	wikilinkRe  = regexp.MustCompile(`\[\[([^\]|#]+)(?:#[^\]|]*)?(?:\|[^\]]*)?\]\]`)
	inlineCode  = regexp.MustCompile("`[^`\n]*`")
	datePrefix  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)
	whitespaces = regexp.MustCompile(`\s+`)
)

type page struct {
	title string
	name  string // uri basename without .md
	short string // name without the engine's date prefix
	links []string
}

func pageOf(s signal) page {
	p := strings.TrimSuffix(s.URI, ".md")
	if i := strings.Index(p, "://"); i >= 0 {
		p = p[i+3:]
	}
	name := path.Base(p)
	return page{title: s.Title, name: name, short: datePrefix.ReplaceAllString(name, ""), links: wikilinks(s.Content)}
}

// wikilinks are the [[targets]] a human wrote: code fences and inline code
// are not prose, and frontmatter is not body.
func wikilinks(content string) []string {
	body := content
	if strings.HasPrefix(body, "---") {
		if end := strings.Index(body[3:], "\n---"); end >= 0 {
			rest := body[3+end+4:]
			if nl := strings.Index(rest, "\n"); nl >= 0 {
				body = rest[nl+1:]
			} else {
				body = ""
			}
		}
	}
	var kept []string
	inFence := ""
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if inFence != "" {
			if strings.HasPrefix(t, inFence) {
				inFence = ""
			}
			continue
		}
		if strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~") {
			inFence = t[:3]
			continue
		}
		kept = append(kept, line)
	}
	text := inlineCode.ReplaceAllString(strings.Join(kept, "\n"), "")
	var out []string
	for _, m := range wikilinkRe.FindAllStringSubmatch(text, -1) {
		target := strings.TrimSpace(m[1])
		if target == "" || len(target) > 120 || strings.ContainsAny(target, "{}\"\n\r") {
			continue
		}
		out = append(out, target)
	}
	return out
}

func norm(s string) string    { return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(s)), ".md") }
func slugify(s string) string { return whitespaces.ReplaceAllString(norm(s), "-") }

// resolver finds a link's page the way FounderOS v1's brain-graph resolver
// does (slug, basename, slugified, title), plus two keys the engine needs:
// its date-prefixed file names and a slugified title.
func resolver(pages []page) func(string) (int, bool) {
	keys := map[string]int{}
	add := func(k string, i int) {
		if _, taken := keys[k]; !taken && k != "" { // first page wins: stable
			keys[k] = i
		}
	}
	for i, p := range pages {
		add(norm(p.name), i)
		add(norm(p.short), i)
	}
	for i, p := range pages {
		add(norm(p.title), i)
		add(slugify(p.title), i)
	}
	return func(target string) (int, bool) {
		for _, k := range []string{norm(target), path.Base(norm(target)), slugify(target), path.Base(slugify(target))} {
			if i, ok := keys[k]; ok {
				return i, true
			}
		}
		return 0, false
	}
}

func auditWorkspace(slug, engine string, sigs []signal, now time.Time) WorkspaceAudit {
	w := WorkspaceAudit{Workspace: slug, Engine: engine, Sources: len(sigs)}
	pages := make([]page, len(sigs))
	for i, s := range sigs {
		pages[i] = pageOf(s)
	}
	w.pages = pages

	byTitle := map[string][]int{}
	var order []string
	for i, p := range pages {
		k := strings.ToLower(strings.TrimSpace(p.title))
		if k == "" {
			continue
		}
		if _, seen := byTitle[k]; !seen {
			order = append(order, k)
		}
		byTitle[k] = append(byTitle[k], i)
	}
	for _, k := range order {
		if n := len(byTitle[k]); n > 1 {
			w.dupes = append(w.dupes, fmt.Sprintf("%s (×%d)", strings.TrimSpace(pages[byTitle[k][0]].title), n))
		}
	}
	for _, p := range pages {
		t := strings.TrimSpace(p.title)
		if t == "" || t == p.name || t == p.short {
			w.untitled = append(w.untitled, p.short)
		}
	}

	var newest time.Time
	for _, s := range sigs {
		for _, raw := range []string{s.ModifiedAt, s.CreatedAt} {
			if ts, err := time.Parse(time.RFC3339Nano, raw); err == nil && ts.After(newest) {
				newest = ts
			}
		}
	}
	if !newest.IsZero() {
		w.Newest = newest.UTC().Format(time.RFC3339)
		w.stale = now.Sub(newest) > StaleAfter
	}
	return w
}

// linkPass resolves every workspace's wikilinks: in its own workspace
// first, then anywhere in the audited knowledge, because the brain-store was
// one corpus before placement split it across workspaces and engines.
func linkPass(ws []WorkspaceAudit) {
	type ref struct{ w, p int }
	var all []page
	var refs []ref
	local := make([]func(string) (int, bool), len(ws))
	linked := make([][]bool, len(ws))
	for wi := range ws {
		local[wi] = resolver(ws[wi].pages)
		linked[wi] = make([]bool, len(ws[wi].pages))
		for pi, p := range ws[wi].pages {
			all = append(all, p)
			refs = append(refs, ref{wi, pi})
		}
	}
	global := resolver(all)
	for wi := range ws {
		w := &ws[wi]
		for pi, p := range w.pages {
			for _, l := range p.links {
				w.Links++
				target := ref{wi, 0}
				if j, ok := local[wi](l); ok {
					target.p = j
				} else if g, ok := global(l); ok {
					target = refs[g]
				} else {
					w.broken = append(w.broken, p.short+" → "+l)
					continue
				}
				w.Resolved++
				if target != (ref{wi, pi}) {
					linked[wi][pi], linked[target.w][target.p] = true, true
				}
			}
		}
	}
	for wi := range ws {
		w := &ws[wi]
		for pi, p := range w.pages {
			if !linked[wi][pi] {
				w.orphans = append(w.orphans, p.short)
			}
		}
		w.Orphans = len(w.orphans)
	}
}

// ---- roll-up -------------------------------------------------------------------

func list(items []string, n int) string {
	if len(items) <= n {
		return strings.Join(items, ", ")
	}
	return strings.Join(items[:n], ", ") + fmt.Sprintf(", +%d more", len(items)-n)
}

func workspaceFindings(a *KnowledgeAudit) []AuditFinding {
	var broken, orphans, dupes, untitled, stale, empty []string
	for _, w := range a.Workspaces {
		a.Sources += w.Sources
		a.Links += w.Links
		a.Resolved += w.Resolved
		a.Broken += len(w.broken)
		a.Orphans += w.Orphans
		pre := func(xs []string) []string {
			out := make([]string, len(xs))
			for i, x := range xs {
				out[i] = w.Workspace + ": " + x
			}
			return out
		}
		broken = append(broken, pre(w.broken)...)
		orphans = append(orphans, pre(w.orphans)...)
		dupes = append(dupes, pre(w.dupes)...)
		untitled = append(untitled, pre(w.untitled)...)
		if w.stale {
			stale = append(stale, fmt.Sprintf("%s (newest %s)", w.Workspace, w.Newest[:10]))
		}
		if w.Sources == 0 {
			empty = append(empty, w.Workspace)
		}
	}
	var out []AuditFinding
	add := func(kind string, items []string, format string) {
		if len(items) > 0 {
			out = append(out, AuditFinding{Kind: kind, Severity: "warn", Count: len(items), Detail: fmt.Sprintf(format, len(items), list(items, 5))})
		}
	}
	add("broken-link", broken, "%d wikilink(s) point at nothing: %s")
	add("orphan", orphans, "%d source(s) with no link in or out: %s")
	add("duplicate-title", dupes, "%d title(s) claimed by more than one source: %s")
	add("no-title", untitled, "%d source(s) with no title of their own, only a file name: %s")
	add("stale-workspace", stale, "%d workspace(s) with nothing new in 30 days: %s")
	add("empty-workspace", empty, "%d workspace(s) hold no sources: %s")
	return out
}

func sortFindings(fs []AuditFinding) {
	sort.SliceStable(fs, func(i, j int) bool { return fs[i].Severity == "err" && fs[j].Severity != "err" })
}

func knowledgeSummary(a *KnowledgeAudit, v *view) string {
	counts := make([]string, len(a.Workspaces))
	for i, w := range a.Workspaces {
		counts[i] = fmt.Sprintf("%s %d", w.Workspace, w.Sources)
	}
	s := fmt.Sprintf("%d sources in %d workspaces (%s) · %d/%d links resolve · %d orphan(s) · ",
		a.Sources, len(a.Workspaces), strings.Join(counts, ", "), a.Resolved, a.Links, a.Orphans)
	if len(a.Findings) == 0 {
		s += "no findings"
	} else {
		kinds := make([]string, len(a.Findings))
		for i, f := range a.Findings {
			kinds[i] = f.Kind
		}
		s += fmt.Sprintf("%d finding(s): %s", len(a.Findings), strings.Join(kinds, ", "))
	}
	if len(v.unstaged) > 0 {
		names := make([]string, 0, len(v.unstaged))
		for n := range v.unstaged {
			names = append(names, n)
		}
		sort.Strings(names)
		parts := make([]string, len(names))
		for i, n := range names {
			parts[i] = fmt.Sprintf("%s (%s not staged)", strings.Join(v.unstaged[n], ", "), n)
		}
		s += " · unaudited: " + strings.Join(parts, "; ")
	}
	return s
}
