package brain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

// DumpInput is POST /pages/brain/dump's body (FounderOS v1 DumpSchema).
type DumpInput struct {
	Text   string   `json:"text"`
	Title  string   `json:"title,omitempty"`
	Folder string   `json:"folder"`
	Tags   []string `json:"tags"`
}

var folderRe = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

// Validate trims the input, derives a missing title from the first seven
// words, and rejects empty text or a folder that is not one plain segment.
func (d DumpInput) Validate() (DumpInput, error) {
	out := DumpInput{Text: strings.TrimSpace(d.Text), Title: strings.TrimSpace(d.Title), Folder: d.Folder, Tags: []string{}}
	if out.Text == "" {
		return out, errors.New("brain dump is empty")
	}
	if !folderRe.MatchString(d.Folder) {
		return out, fmt.Errorf("invalid folder: %q: one top-level folder, no slashes", d.Folder)
	}
	if out.Title == "" {
		words := strings.Fields(out.Text)
		if len(words) > 7 {
			words = words[:7]
		}
		out.Title = strings.Join(words, " ")
	}
	for _, t := range d.Tags {
		if t = strings.TrimPrefix(strings.TrimSpace(t), "#"); t != "" {
			out.Tags = append(out.Tags, t)
		}
	}
	return out, nil
}

// SlugifyTitle: kebab-case, accents folded, noise stripped, "untitled" if empty.
func SlugifyTitle(title string) string {
	var b strings.Builder
	for _, r := range norm.NFKD.String(strings.ToLower(title)) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case r == '_' || unicode.IsSpace(r):
			b.WriteByte(' ')
		case r == '-' || r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			b.WriteRune(r)
		}
	}
	slug := strings.Join(strings.Fields(b.String()), "-")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "untitled"
	}
	return slug
}

// RelPath is the brain-store path the note would have had; it stays the
// dump's human handle in the response.
func (d DumpInput) RelPath(now time.Time) string {
	return d.Folder + "/" + now.UTC().Format("2006-01-02") + "-" + SlugifyTitle(d.Title) + ".md"
}

// Document is the markdown the engine ingests: the same frontmatter and H1
// FounderOS v1 wrote to the brain-store file.
func (d DumpInput) Document(now time.Time) string {
	return strings.Join([]string{
		"---",
		"created: " + now.UTC().Format(time.RFC3339),
		"source: founderos-os-brain-dump",
		"tags: [" + strings.Join(d.Tags, ", ") + "]",
		"---",
		"",
		"# " + d.Title,
		"",
		d.Text,
		"",
	}, "\n")
}

// Workspace routes the dump like the brain importer routes a store page: by
// folder. A venture tag (a business workspace slug) moves a note out of the
// default workspace, but never out of a personal folder.
func (d DumpInput) Workspace(topo *topology.Topology) string {
	ws := topo.WorkspaceForBrainPath(d.Folder + "/note.md")
	if ws != topo.BrainStore.Default {
		return ws
	}
	for _, t := range d.Tags {
		for _, w := range topo.Workspaces {
			if w.Class == topology.ClassBusiness && strings.EqualFold(w.Slug, t) {
				return w.Slug
			}
		}
	}
	return ws
}

// Node is the engine node the capture lands in (the importer's mapping).
func (d DumpInput) Node() string {
	if d.Folder == "inbox" || d.Folder == "archive" {
		return "inbox"
	}
	return "knowledge-base"
}
