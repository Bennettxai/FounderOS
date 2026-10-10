// Package devicepush is the backend half of spec §4.18: device-local sources
// (WhatsApp, Wispr, Obsidian, the local stack, and Claude/Codex/Ollama usage)
// are read by founderos-collector on each Mac and PUSHED here, the same model as
// FounderOS v1's scripts/push-usage.mjs. The backend never reads a device path.
//
// This file holds the pushed payload. Field names and JSON shapes follow the
// FounderOS v1 schemas the pages already consume (CommsItem, WisprNote,
// UsageSnapshotSchema, OllamaSnapshotSchema), so a page port can use them as is.
package devicepush

import "github.com/rhl/businessos-backend/internal/founderos/connectors"

// Source ids a collector may push.
const (
	SourceWhatsApp    = "whatsapp"
	SourceWispr       = "wispr"
	SourceObsidian    = "obsidian"
	SourceLocalStack  = "local-stack"
	SourceClaudeUsage = "claude-usage"
	SourceCodexUsage  = "codex-usage"
	SourceOllamaUsage = "ollama-usage"
)

// Metas are the device-local rows of the Connections board, with FounderOS v1's
// exact ids, names and kinds (lib/connectors/index.ts and each module).
var Metas = map[string]connectors.Meta{
	SourceWhatsApp:   {ID: "whatsapp", Name: "WhatsApp", Kind: connectors.KindSocial},
	SourceWispr:      {ID: "wispr", Name: "Wispr Flow", Kind: connectors.KindLocal},
	SourceObsidian:   {ID: "obsidian", Name: "Obsidian Vault", Kind: connectors.KindKnowledge},
	SourceLocalStack: {ID: "local-stack", Name: "Local Stack", Kind: connectors.KindLocal},
}

// Payload is one collector tick from one Mac.
type Payload struct {
	// Device is a slug ("alexs-macbook-pro", "mini").
	Device string `json:"device"`
	Label  string `json:"label"`
	// CapturedAt is the collector's clock; the receiver's own time is what
	// staleness is judged by.
	CapturedAt string `json:"capturedAt"`
	// Statuses are the Connections-board rows as computed on the device.
	Statuses []connectors.Status `json:"statuses,omitempty"`
	WhatsApp *WhatsAppData       `json:"whatsapp,omitempty"`
	Wispr    *WisprData          `json:"wispr,omitempty"`
	Obsidian *ObsidianData       `json:"obsidian,omitempty"`
	// Usage carries the Claude and Codex seats (UsageSnapshotSchema).
	Usage  []SeatUsage     `json:"usage,omitempty"`
	Ollama *OllamaSnapshot `json:"ollama,omitempty"`
}

// Chat is a CommsItem from the WhatsApp lane (lib/comms.ts).
type Chat struct {
	Source  string `json:"source"`
	Title   string `json:"title"`
	Sender  string `json:"sender,omitempty"`
	ReplyTo string `json:"replyTo,omitempty"`
	Preview string `json:"preview"`
	TS      string `json:"ts"`
	Unread  int    `json:"unread"`
}

type WhatsAppData struct {
	Chats []Chat `json:"chats"`
}

// WisprNote mirrors WisprNoteSchema.
type WisprNote struct {
	ID        string  `json:"id"`
	Kind      string  `json:"kind"` // dictation | note | todo
	Text      string  `json:"text"`
	App       *string `json:"app"`
	TS        string  `json:"ts"`
	WordCount *int    `json:"wordCount"`
}

type WisprData struct {
	Notes []WisprNote `json:"notes"`
}

// VaultNote is a BrainNote-shaped vault row: vault-relative path, capped content.
type VaultNote struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type ObsidianData struct {
	Notes []VaultNote `json:"notes"`
}

// ---- usage (lib/usage.ts) ----------------------------------------------------

type OfficialWindow struct {
	UsedPercent   float64 `json:"usedPercent"`
	WindowMinutes int     `json:"windowMinutes"`
	ResetsAt      *string `json:"resetsAt"`
}

type Official struct {
	Session *OfficialWindow `json:"session,omitempty"`
	Weekly  *OfficialWindow `json:"weekly,omitempty"`
}

type Tot struct {
	In         float64 `json:"in"`
	Out        float64 `json:"out"`
	CacheWrite float64 `json:"cacheWrite"`
	CacheRead  float64 `json:"cacheRead"`
}

type DayBucket struct {
	Day string `json:"day"`
	Tot
}

// LaneTots is burn per lane: board, sessions, terminal, automation.
type LaneTots struct {
	Board      Tot `json:"board"`
	Sessions   Tot `json:"sessions"`
	Terminal   Tot `json:"terminal"`
	Automation Tot `json:"automation"`
}

type Windows struct {
	Hour    LaneTots `json:"hour"`
	Session LaneTots `json:"session"`
	Day     LaneTots `json:"day"`
	Week    LaneTots `json:"week"`
}

type TopBurner struct {
	Source string  `json:"source"`
	Label  string  `json:"label"`
	Burn   float64 `json:"burn"`
}

type Breakdown struct {
	Windows Windows     `json:"windows"`
	Top     []TopBurner `json:"top"`
}

// SeatUsage mirrors UsageSnapshotSchema: one machine's reading of a plan.
type SeatUsage struct {
	ID           string         `json:"id"`
	Kind         string         `json:"kind"` // claude | codex
	Label        string         `json:"label"`
	Source       string         `json:"source"` // local | push
	CapturedAt   string         `json:"capturedAt"`
	Days         []DayBucket    `json:"days"`
	ByModel      map[string]Tot `json:"byModel"`
	LastActivity *string        `json:"lastActivity"`
	Official     *Official      `json:"official"`
	Note         string         `json:"note,omitempty"`
	Breakdown    *Breakdown     `json:"breakdown,omitempty"`
	Plan         *string        `json:"plan"`
}

type RequestCounts struct {
	Chat  int `json:"chat"`
	Embed int `json:"embed"`
}

type RequestWindows struct {
	Hour    RequestCounts `json:"hour"`
	Session RequestCounts `json:"session"`
	Day     RequestCounts `json:"day"`
	Week    RequestCounts `json:"week"`
}

type OllamaModel struct {
	Name  string `json:"name"`
	Cloud bool   `json:"cloud"`
}

// OllamaLane mirrors OllamaLaneSchema.
type OllamaLane struct {
	State    string          `json:"state"` // up | down
	Plan     *string         `json:"plan"`
	Models   []OllamaModel   `json:"models"`
	Requests *RequestWindows `json:"requests"`
	Note     string          `json:"note"`
}

// OllamaSnapshot mirrors OllamaSnapshotSchema.
type OllamaSnapshot struct {
	Kind       string     `json:"kind"` // always "ollama"
	ID         string     `json:"id"`
	Label      string     `json:"label"`
	CapturedAt string     `json:"capturedAt"`
	Lane       OllamaLane `json:"lane"`
}
