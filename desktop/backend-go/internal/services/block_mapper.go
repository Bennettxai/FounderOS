package services

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BlockMapperService converts markdown content to structured blocks
type BlockMapperService struct {
	db     *sql.DB
	logger *slog.Logger
}

// Block represents a structured content block
type Block struct {
	ID         string                 `json:"id"`
	Type       BlockType              `json:"type"`
	Content    string                 `json:"content"`
	RawContent string                 `json:"raw_content,omitempty"`
	Language   string                 `json:"language,omitempty"`
	Level      int                    `json:"level,omitempty"`
	Children   []*Block               `json:"children,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Hash       string                 `json:"hash"`
	StartLine  int                    `json:"start_line,omitempty"`
	EndLine    int                    `json:"end_line,omitempty"`
}

// BlockType represents the type of content block
type BlockType string

const (
	BlockTypeParagraph   BlockType = "paragraph"
	BlockTypeHeading     BlockType = "heading"
	BlockTypeCode        BlockType = "code"
	BlockTypeCodeInline  BlockType = "code_inline"
	BlockTypeBlockquote  BlockType = "blockquote"
	BlockTypeList        BlockType = "list"
	BlockTypeListItem    BlockType = "list_item"
	BlockTypeTable       BlockType = "table"
	BlockTypeTableRow    BlockType = "table_row"
	BlockTypeTableCell   BlockType = "table_cell"
	BlockTypeImage       BlockType = "image"
	BlockTypeLink        BlockType = "link"
	BlockTypeHR          BlockType = "horizontal_rule"
	BlockTypeHTML        BlockType = "html"
	BlockTypeThinking    BlockType = "thinking"
	BlockTypeArtifact    BlockType = "artifact"
	BlockTypeCallout     BlockType = "callout"
	BlockTypeMath        BlockType = "math"
	BlockTypeFrontmatter BlockType = "frontmatter"
	BlockTypeTask        BlockType = "task"
	BlockTypeFootnote    BlockType = "footnote"
)

// BlockDocument represents a parsed document with blocks
type BlockDocument struct {
	ID          string                 `json:"id"`
	SourceID    string                 `json:"source_id,omitempty"`
	Title       string                 `json:"title,omitempty"`
	Blocks      []*Block               `json:"blocks"`
	Outline     []*OutlineEntry        `json:"outline,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Hash        string                 `json:"hash"`
	TotalBlocks int                    `json:"total_blocks"`
	CreatedAt   time.Time              `json:"created_at"`
}

// OutlineEntry represents a heading in the document outline
type OutlineEntry struct {
	ID       string          `json:"id"`
	Level    int             `json:"level"`
	Title    string          `json:"title"`
	BlockID  string          `json:"block_id"`
	Children []*OutlineEntry `json:"children,omitempty"`
}

// BlockMapperOptions configures parsing behavior
type BlockMapperOptions struct {
	ExtractOutline     bool `json:"extract_outline"`
	PreserveRawContent bool `json:"preserve_raw_content"`
	ParseCodeBlocks    bool `json:"parse_code_blocks"`
	ParseTables        bool `json:"parse_tables"`
	ParseThinking      bool `json:"parse_thinking"`
	ParseArtifacts     bool `json:"parse_artifacts"`
	IncludeLineNumbers bool `json:"include_line_numbers"`
}

// DefaultBlockMapperOptions returns default parsing options
func DefaultBlockMapperOptions() *BlockMapperOptions {
	return &BlockMapperOptions{
		ExtractOutline:     true,
		PreserveRawContent: false,
		ParseCodeBlocks:    true,
		ParseTables:        true,
		ParseThinking:      true,
		ParseArtifacts:     true,
		IncludeLineNumbers: true,
	}
}

// NewBlockMapperService creates a new block mapper service
func NewBlockMapperService(db *sql.DB, logger *slog.Logger) *BlockMapperService {
	return &BlockMapperService{
		db:     db,
		logger: logger,
	}
}

// ParseMarkdown converts markdown content to a BlockDocument
func (s *BlockMapperService) ParseMarkdown(ctx context.Context, content string, opts *BlockMapperOptions) (*BlockDocument, error) {
	if opts == nil {
		opts = DefaultBlockMapperOptions()
	}

	doc := &BlockDocument{
		ID:        uuid.New().String(),
		Blocks:    make([]*Block, 0),
		Outline:   make([]*OutlineEntry, 0),
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
	}

	// Calculate document hash
	hash := sha256.Sum256([]byte(content))
	doc.Hash = hex.EncodeToString(hash[:])

	lines := strings.Split(content, "\n")
	lineNum := 0

	for lineNum < len(lines) {
		block, consumed := s.parseBlock(lines, lineNum, opts)
		if block != nil {
			if opts.IncludeLineNumbers {
				block.StartLine = lineNum + 1
				block.EndLine = lineNum + consumed
			}
			doc.Blocks = append(doc.Blocks, block)
		}
		if consumed == 0 {
			consumed = 1
		}
		lineNum += consumed
	}

	// Extract outline from headings
	if opts.ExtractOutline {
		doc.Outline = s.extractOutline(doc.Blocks)
	}

	// Extract title from first heading
	for _, b := range doc.Blocks {
		if b.Type == BlockTypeHeading && b.Level == 1 {
			doc.Title = b.Content
			break
		}
	}

	doc.TotalBlocks = len(doc.Blocks)

	return doc, nil
}

// parseBlock parses a single block starting at lineNum
func (s *BlockMapperService) parseBlock(lines []string, lineNum int, opts *BlockMapperOptions) (*Block, int) {
	if lineNum >= len(lines) {
		return nil, 0
	}

	line := lines[lineNum]
	trimmedLine := strings.TrimSpace(line)

	// Skip empty lines
	if trimmedLine == "" {
		return nil, 1
	}

	// Frontmatter (YAML)
	if lineNum == 0 && trimmedLine == "---" {
		return s.parseFrontmatter(lines, lineNum)
	}

	// Thinking tags
	if opts.ParseThinking && strings.HasPrefix(trimmedLine, "<thinking") {
		return s.parseThinking(lines, lineNum)
	}

	// Artifact tags
	if opts.ParseArtifacts && strings.HasPrefix(trimmedLine, "<artifact") {
		return s.parseArtifact(lines, lineNum)
	}

	// Code blocks
	if opts.ParseCodeBlocks && strings.HasPrefix(trimmedLine, "```") {
		return s.parseCodeBlock(lines, lineNum)
	}

	// Headings
	if strings.HasPrefix(trimmedLine, "#") {
		return s.parseHeading(line), 1
	}

	// Horizontal rule
	if s.isHorizontalRule(trimmedLine) {
		return &Block{
			ID:   uuid.New().String(),
			Type: BlockTypeHR,
			Hash: s.hashContent("---"),
		}, 1
	}

	// Blockquote
	if strings.HasPrefix(trimmedLine, ">") {
		return s.parseBlockquote(lines, lineNum)
	}

	// Lists
	if s.isListItem(trimmedLine) {
		return s.parseList(lines, lineNum)
	}

	// Tables
	if opts.ParseTables && strings.HasPrefix(trimmedLine, "|") {
		return s.parseTable(lines, lineNum)
	}

	// Math blocks
	if strings.HasPrefix(trimmedLine, "$$") {
		return s.parseMathBlock(lines, lineNum)
	}

	// Callouts (Obsidian-style)
	if strings.HasPrefix(trimmedLine, "> [!") {
		return s.parseCallout(lines, lineNum)
	}

	// HTML blocks
	if strings.HasPrefix(trimmedLine, "<") && !strings.HasPrefix(trimmedLine, "<thinking") && !strings.HasPrefix(trimmedLine, "<artifact") {
		return s.parseHTMLBlock(lines, lineNum)
	}

	// Default: paragraph
	return s.parseParagraph(lines, lineNum)
}

// extractOutline builds an outline from heading blocks
func (s *BlockMapperService) extractOutline(blocks []*Block) []*OutlineEntry {
	outline := make([]*OutlineEntry, 0)
	stack := make([]*OutlineEntry, 0)

	for _, b := range blocks {
		if b.Type != BlockTypeHeading {
			continue
		}

		entry := &OutlineEntry{
			ID:       uuid.New().String(),
			Level:    b.Level,
			Title:    b.Content,
			BlockID:  b.ID,
			Children: make([]*OutlineEntry, 0),
		}

		// Find parent
		for len(stack) > 0 && stack[len(stack)-1].Level >= b.Level {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			outline = append(outline, entry)
		} else {
			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, entry)
		}

		stack = append(stack, entry)
	}

	return outline
}
