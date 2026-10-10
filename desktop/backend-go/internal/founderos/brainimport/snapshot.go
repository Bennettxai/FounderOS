package brainimport

import (
	"database/sql"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/topology"

	_ "modernc.org/sqlite"
)

// FromSnapshot turns an engine snapshot's source packages back into ingest
// items, routed by topology. Replaying through /api/ingest lets the staging
// engine rebuild its own claims, chunks, and embeddings instead of us editing
// its tables. It returns the items and how many packages had no text.
func FromSnapshot(path string, topo *topology.Topology) ([]Item, int, error) {
	db, err := sql.Open("sqlite", "file:"+(&url.URL{Path: path}).EscapedPath()+"?mode=ro")
	if err != nil {
		return nil, 0, err
	}
	defer db.Close()
	rows, err := db.Query(`SELECT id, coalesce(workspace_id, ''), coalesce(raw_text, '')
		FROM source_packages ORDER BY created_at, id`)
	if err != nil {
		// Older fixtures have no created_at; fall back to id order.
		rows, err = db.Query(`SELECT id, coalesce(workspace_id, ''), coalesce(raw_text, '')
			FROM source_packages ORDER BY id`)
		if err != nil {
			return nil, 0, err
		}
	}
	defer rows.Close()

	var items []Item
	var texts []string
	empty := 0
	for rows.Next() {
		var id, ws, text string
		if err := rows.Scan(&id, &ws, &text); err != nil {
			return nil, 0, err
		}
		if strings.TrimSpace(text) == "" {
			empty++
			continue
		}
		rel := "source_packages/" + id
		it, err := NewItem(topo, rel, strings.TrimPrefix(ws, "default:"), text)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, it)
		texts = append(texts, strings.TrimSpace(text))
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	orphans, err := orphanContexts(db, topo, texts)
	if err != nil {
		return nil, 0, err
	}
	return append(items, orphans...), empty, nil
}

// orphanContexts returns live contexts whose text is not carried by any
// source package (files the engine indexed directly). Without them a replay
// would silently drop that knowledge.
func orphanContexts(db *sql.DB, topo *topology.Topology, sourceTexts []string) ([]Item, error) {
	packageTitles := map[string]bool{}
	for _, t := range sourceTexts {
		if k := TitleKey(h1(t)); k != "" && k != "untitled" {
			packageTitles[k] = true
		}
	}
	var hasContexts int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='contexts'`).Scan(&hasContexts); err != nil || hasContexts == 0 {
		return nil, err
	}
	uriCol := "''"
	var hasURI int
	_ = db.QueryRow(`SELECT count(*) FROM pragma_table_info('contexts') WHERE name='uri'`).Scan(&hasURI)
	if hasURI > 0 {
		uriCol = "coalesce(uri,'')"
	}
	rows, err := db.Query(`SELECT id, coalesce(workspace_id,''), coalesce(title,''), coalesce(genre,''),
		coalesce(node,''), coalesce(content,''), ` + uriCol + ` FROM contexts WHERE archived_at IS NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Item
	for rows.Next() {
		var id, ws, title, genre, node, content, uri string
		if err := rows.Scan(&id, &ws, &title, &genre, &node, &content, &uri); err != nil {
			return nil, err
		}
		body := strings.TrimSpace(content)
		if body == "" || coveredBy(body, sourceTexts) {
			continue
		}
		if TitleKey(title) == "" || TitleKey(title) == "untitled" {
			title = titleFromURI(uri)
		}
		// A context whose title names one of the snapshot's own source
		// packages is a reformatted copy of it, not separate knowledge.
		if k := TitleKey(title); k != "" && packageTitles[k] {
			continue
		}
		it, err := NewItem(topo, "contexts/"+id, strings.TrimPrefix(ws, "default:"), content)
		if err != nil {
			return nil, err
		}
		if title != "" {
			it.Title = title
		}
		if genre != "" {
			it.Genre = genre
		}
		if node != "" {
			it.Node = node
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func coveredBy(body string, sourceTexts []string) bool {
	for _, t := range sourceTexts {
		if t != "" && (strings.Contains(body, t) || strings.Contains(t, body)) {
			return true
		}
	}
	return false
}

var datePrefix = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-`)

func titleFromURI(uri string) string {
	base := strings.TrimSuffix(path.Base(uri), path.Ext(uri))
	base = datePrefix.ReplaceAllString(base, "")
	if base == "" || base == "." || base == "/" {
		return ""
	}
	return strings.ReplaceAll(base, "-", " ")
}
