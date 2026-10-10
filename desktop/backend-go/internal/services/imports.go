package services

import (
	"time"

	"github.com/rhl/businessos-backend/internal/database/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// =============================================================================
// IMPORT SERVICE
// =============================================================================

type ImportService struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
}

func NewImportService(pool *pgxpool.Pool) *ImportService {
	return &ImportService{
		pool:    pool,
		queries: sqlc.New(pool),
	}
}

// =============================================================================
// CHATGPT EXPORT FORMAT
// =============================================================================

// ChatGPT exports conversations.json with this structure
type ChatGPTExport struct {
	Conversations []ChatGPTConversation `json:"conversations"`
}

type ChatGPTConversation struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	CreateTime  float64                `json:"create_time"` // Unix timestamp (can be float)
	UpdateTime  float64                `json:"update_time"`
	Mapping     map[string]ChatGPTNode `json:"mapping"`
	CurrentNode string                 `json:"current_node,omitempty"`
}

type ChatGPTNode struct {
	ID       string          `json:"id"`
	Message  *ChatGPTMessage `json:"message,omitempty"`
	Parent   *string         `json:"parent"`
	Children []string        `json:"children"`
}

type ChatGPTMessage struct {
	ID         string         `json:"id"`
	Author     ChatGPTAuthor  `json:"author"`
	CreateTime *float64       `json:"create_time"`
	Content    ChatGPTContent `json:"content"`
	Status     string         `json:"status,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type ChatGPTAuthor struct {
	Role     string         `json:"role"` // "user", "assistant", "system", "tool"
	Name     *string        `json:"name,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type ChatGPTContent struct {
	ContentType string `json:"content_type"` // "text", "code", etc.
	Parts       []any  `json:"parts"`        // Usually strings, but can be other types
}

// =============================================================================
// CLAUDE EXPORT FORMAT
// =============================================================================

// Claude exports conversations.json with this structure
type ClaudeExport struct {
	Conversations []ClaudeConversation `json:"conversations"`
}

type ClaudeConversation struct {
	UUID         string          `json:"uuid"`
	Name         string          `json:"name"`
	CreatedAt    string          `json:"created_at"` // ISO8601
	UpdatedAt    string          `json:"updated_at"`
	ChatMessages []ClaudeMessage `json:"chat_messages"`
	Model        string          `json:"model,omitempty"`
	Project      *ClaudeProject  `json:"project,omitempty"`
}

type ClaudeMessage struct {
	UUID      string       `json:"uuid"`
	Text      string       `json:"text"`
	Sender    string       `json:"sender"` // "human", "assistant"
	CreatedAt string       `json:"created_at,omitempty"`
	Files     []ClaudeFile `json:"files,omitempty"`
}

type ClaudeFile struct {
	FileName         string `json:"file_name"`
	FileType         string `json:"file_type"`
	FileSize         int64  `json:"file_size,omitempty"`
	ExtractedContent string `json:"extracted_content,omitempty"`
}

type ClaudeProject struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// =============================================================================
// NORMALIZED MESSAGE FORMAT
// =============================================================================

// NormalizedMessage is the standardized format we store in the database
type NormalizedMessage struct {
	Role      string         `json:"role"` // "user", "assistant", "system"
	Content   string         `json:"content"`
	Timestamp *time.Time     `json:"timestamp,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// =============================================================================
// IMPORT JOB INPUT/OUTPUT
// =============================================================================

type CreateImportJobInput struct {
	UserID           string
	SourceType       sqlc.ImportSourceType
	OriginalFilename string
	FileSizeBytes    int64
	ContentType      string
	TargetModule     string
	ImportOptions    map[string]any
}

type ImportProgress struct {
	TotalRecords     int
	ProcessedRecords int
	ImportedRecords  int
	SkippedRecords   int
	FailedRecords    int
	ProgressPercent  int
}

type ImportResult struct {
	JobID           uuid.UUID
	Status          sqlc.ImportStatus
	TotalRecords    int
	ImportedRecords int
	SkippedRecords  int
	FailedRecords   int
	Errors          []ImportError
}

type ImportError struct {
	RecordIndex int    `json:"record_index"`
	ExternalID  string `json:"external_id,omitempty"`
	Error       string `json:"error"`
}

// =============================================================================
// POINTER HELPERS
// =============================================================================

func ptr[T any](v T) *T {
	return &v
}

func ptrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
