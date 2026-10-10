package collect

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
)

const (
	walkCap        = 5000
	noteContentCap = 20_000
)

// walkVault visits the vault like the TS: dot entries skipped, and the walk
// stops descending once more than walkCap entries have been visited.
func walkVault(dir string, visited *int, onFile func(full string)) {
	if *visited > walkCap {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		*visited++
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		full := filepath.Join(dir, e.Name())
		if e.IsDir() {
			walkVault(full, visited, onFile)
		} else if strings.HasSuffix(e.Name(), ".md") {
			onFile(full)
		}
	}
}

// localeLess approximates String.prototype.localeCompare for paths:
// case-insensitive first, lowercase before uppercase on a tie.
func localeLess(a, b string) bool {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	if la != lb {
		return la < lb
	}
	return a > b
}

// ReadVaultNotes reads the vault's markdown notes (incl. the Claude Archive)
// as {path, content} rows: vault-relative paths, content capped, never an
// error. A missing vault or a macOS TCC denial is an empty list.
func ReadVaultNotes(vault string) []devicepush.VaultNote {
	notes := []devicepush.VaultNote{}
	visited := 0
	walkVault(vault, &visited, func(full string) {
		raw, err := os.ReadFile(full)
		if err != nil {
			return // unreadable file: skip it
		}
		rel, err := filepath.Rel(vault, full)
		if err != nil {
			return
		}
		notes = append(notes, devicepush.VaultNote{Path: filepath.ToSlash(rel), Content: truncRunes(string(raw), noteContentCap)})
	})
	sort.SliceStable(notes, func(i, j int) bool { return localeLess(notes[i].Path, notes[j].Path) })
	return notes
}

// ObsidianStatus counts the vault's markdown notes. Documents/ is TCC
// protected, so a denied read says so rather than reading as an empty vault.
func ObsidianStatus(vault, home string) connectors.Status {
	m := devicepush.Metas[devicepush.SourceObsidian]
	s := connectors.Status{ID: m.ID, Name: m.Name, Kind: m.Kind}
	if _, err := os.Stat(vault); err != nil {
		s.State = connectors.StateNotConfigured
		s.Detail = "Vault not found at " + vault + " — set OBSIDIAN_VAULT to override."
		return s
	}
	if _, err := os.ReadDir(vault); err != nil {
		s.State = connectors.StateError
		s.Detail = `Vault exists but macOS denied access — grant the terminal "Files and Folders → Documents" in System Settings → Privacy.`
		return s
	}
	notes, visited := 0, 0
	walkVault(vault, &visited, func(string) { notes++ })
	shown := vault
	if home != "" {
		shown = strings.Replace(vault, home, "~", 1)
	}
	s.State = connectors.StateConnected
	s.Detail = commas(notes) + " markdown notes (incl. Claude Archive) at " + shown
	s.Meta = map[string]any{"notes": notes}
	return s
}
