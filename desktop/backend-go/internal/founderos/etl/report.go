package etl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SourceCount is one source table's row count.
type SourceCount struct {
	File  string `json:"file"`
	Table string `json:"table"`
	Rows  int    `json:"rows"`
}

// TableReport is one target table's outcome.
type TableReport struct {
	Target  string        `json:"target"`
	Sources []SourceCount `json:"sources"`
	// Skipped is set when no source file holding the table was present: the
	// target is left untouched, because an absent source is unknown, not empty.
	Skipped    string `json:"skipped,omitempty"`
	SourceRows int    `json:"source_rows"`
	TargetRows int    `json:"target_rows"`
	Inserted   int    `json:"inserted"`
	Updated    int    `json:"updated"`
	Deleted    int    `json:"deleted"`
	Unchanged  int    `json:"unchanged"`
	// Conversions (table-map.md date rules).
	D2DateOnly      int      `json:"d2_date_only"`
	D6BlankToNull   int      `json:"d6_blank_to_null"`
	NoOffsetAsUTC   int      `json:"no_offset_as_utc"`
	KeyCollisions   int      `json:"key_collisions"`
	DefaultedColumn []string `json:"defaulted_columns,omitempty"`
	Match           bool     `json:"match"`
}

// DroppedReport records a retired table that is counted but not loaded.
type DroppedReport struct {
	Table      string `json:"table"`
	SourceRows int    `json:"source_rows"`
}

// VaultReport records the proposal access-code move to credential_vault.
type VaultReport struct {
	Provider string `json:"provider"`
	UserID   string `json:"user_id,omitempty"`
	Codes    int    `json:"codes"`
	Action   string `json:"action"` // inserted | updated | unchanged | deleted | none
}

// TradingLimitsReport states the one intentional value change.
type TradingLimitsReport struct {
	SourceRows        int    `json:"source_rows"`
	SourceAutopilot   *bool  `json:"source_autopilot"`
	ImportedAutopilot bool   `json:"imported_autopilot"`
	Note              string `json:"note"`
}

// Report is the per-run count report (criterion 3.3).
type Report struct {
	GeneratedAt   time.Time           `json:"generated_at"`
	From          string              `json:"from"`
	Database      string              `json:"database"`
	SourceFiles   map[string]string   `json:"source_files"`
	Workspaces    map[string]string   `json:"workspaces"`
	Tables        []*TableReport      `json:"tables"`
	Dropped       []DroppedReport     `json:"dropped"`
	StateDBOther  map[string]int      `json:"state_db_other_tables,omitempty"`
	Vault         VaultReport         `json:"vault"`
	TradingLimits TradingLimitsReport `json:"trading_limits"`
	PendingNulled int                 `json:"slack_sessions_pending_nulled"`
	OK            bool                `json:"ok"`
	Problems      []string            `json:"problems,omitempty"`
}

// Changes is the total of inserted, updated and deleted rows (0 = no-op run).
func (r *Report) Changes() int {
	n := 0
	for _, t := range r.Tables {
		n += t.Inserted + t.Updated + t.Deleted
	}
	if r.Vault.Action == "inserted" || r.Vault.Action == "updated" || r.Vault.Action == "deleted" {
		n++
	}
	return n
}

// Table returns the report row for a target table.
func (r *Report) Table(target string) *TableReport {
	for _, t := range r.Tables {
		if t.Target == target {
			return t
		}
	}
	return nil
}

// Write saves the report as <dir>/<base>.json and <dir>/<base>.md.
func (r *Report) Write(dir, base string) (string, string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", err
	}
	jp := filepath.Join(dir, base+".json")
	mp := filepath.Join(dir, base+".md")
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", "", err
	}
	if err := os.WriteFile(jp, append(b, '\n'), 0o644); err != nil {
		return "", "", err
	}
	if err := os.WriteFile(mp, []byte(r.Markdown()), 0o644); err != nil {
		return "", "", err
	}
	return jp, mp, nil
}

// Markdown renders the report for docs/founderos/.
func (r *Report) Markdown() string {
	var b strings.Builder
	status := "GREEN"
	if !r.OK {
		status = "RED"
	}
	fmt.Fprintf(&b, "# The operator ETL report: %s\n\n", status)
	fmt.Fprintf(&b, "- Generated: %s\n", r.GeneratedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "- From: `%s`\n", r.From)
	fmt.Fprintf(&b, "- Database: `%s`\n", r.Database)
	files := make([]string, 0, len(r.SourceFiles))
	for _, f := range []string{FileOS, FileBank, FileLedger, FilePaykit, FileState} {
		if p, ok := r.SourceFiles[f]; ok {
			files = append(files, fmt.Sprintf("`%s`", filepath.Base(filepath.Dir(p))+"/"+filepath.Base(p)))
		} else {
			files = append(files, fmt.Sprintf("%s (absent)", f))
		}
	}
	fmt.Fprintf(&b, "- Source files: %s\n", strings.Join(files, ", "))
	fmt.Fprintf(&b, "- Tables: %d target tables; source rows %d, target rows %d; changes this run: %d\n\n",
		len(r.Tables), r.sum(func(t *TableReport) int { return t.SourceRows }), r.sum(func(t *TableReport) int { return t.TargetRows }), r.Changes())

	b.WriteString("| Target table | Source | Source rows | Target rows | Ins | Upd | Del | D2 | D6 | No-offset | Match |\n")
	b.WriteString("|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|\n")
	for _, t := range r.Tables {
		var src []string
		for _, s := range t.Sources {
			src = append(src, fmt.Sprintf("%s:%s (%d)", strings.TrimSuffix(s.File, ".db"), s.Table, s.Rows))
		}
		if t.Skipped != "" {
			src = append(src, "skipped: "+t.Skipped)
		}
		match := "yes"
		if !t.Match {
			match = "**NO**"
		}
		if t.Skipped != "" {
			match = "skipped"
		}
		fmt.Fprintf(&b, "| %s | %s | %d | %d | %d | %d | %d | %d | %d | %d | %s |\n",
			t.Target, strings.Join(src, ", "), t.SourceRows, t.TargetRows, t.Inserted, t.Updated, t.Deleted,
			t.D2DateOnly, t.D6BlankToNull, t.NoOffsetAsUTC, match)
	}
	b.WriteString("\n## Dropped (retired, not loaded)\n\n")
	for _, d := range r.Dropped {
		fmt.Fprintf(&b, "- `%s`: %d source rows\n", d.Table, d.SourceRows)
	}
	b.WriteString("\n## Intentional value changes\n\n")
	ap := "no source row"
	if r.TradingLimits.SourceAutopilot != nil {
		ap = fmt.Sprintf("source %v", *r.TradingLimits.SourceAutopilot)
	}
	fmt.Fprintf(&b, "- `trading_limits.autopilot`: %s, imported **%v**. %s\n", ap, r.TradingLimits.ImportedAutopilot, r.TradingLimits.Note)
	fmt.Fprintf(&b, "- `proposals.access_code`: %d codes moved to `credential_vault` (provider `%s`, action %s); `founderos_proposals.has_access_code` records presence only.\n",
		r.Vault.Codes, r.Vault.Provider, r.Vault.Action)
	fmt.Fprintf(&b, "- `slack_bridge_sessions.payload.pending`: nulled on %d rows (live confirmation nonce).\n", r.PendingNulled)
	if len(r.StateDBOther) > 0 {
		b.WriteString("\n## state.db non-Slack tables (asserted empty)\n\n")
		fmt.Fprintf(&b, "%d tables checked, all empty.\n", len(r.StateDBOther))
	}
	if len(r.Problems) > 0 {
		b.WriteString("\n## Problems\n\n")
		for _, p := range r.Problems {
			fmt.Fprintf(&b, "- %s\n", p)
		}
	}
	return b.String()
}

func (r *Report) sum(f func(*TableReport) int) int {
	n := 0
	for _, t := range r.Tables {
		n += f(t)
	}
	return n
}
