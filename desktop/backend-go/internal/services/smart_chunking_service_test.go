package services

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestNewSmartChunkingService(t *testing.T) {
	svc := NewSmartChunkingService()
	if svc == nil {
		t.Fatal("expected service to be created")
	}
	if svc.charsPerToken != 4.0 {
		t.Errorf("expected charsPerToken to be 4.0, got %f", svc.charsPerToken)
	}
}

func TestDefaultChunkOptions(t *testing.T) {
	opts := DefaultChunkOptions()
	if opts.ChunkSize != 512 {
		t.Errorf("expected chunk size 512, got %d", opts.ChunkSize)
	}
	if opts.OverlapRatio != 0.2 {
		t.Errorf("expected overlap ratio 0.2, got %f", opts.OverlapRatio)
	}
	if opts.PreserveCode != true {
		t.Error("expected PreserveCode to be true")
	}
	if opts.MinChunkSize != 128 {
		t.Errorf("expected min chunk size 128, got %d", opts.MinChunkSize)
	}
}

func TestChunkPlainText(t *testing.T) {
	svc := NewSmartChunkingService()
	ctx := context.Background()

	content := `This is paragraph one. It has multiple sentences. This is the third sentence.

This is paragraph two. It is separate from paragraph one.

This is paragraph three. It contains important information.`

	opts := ChunkOptions{
		ChunkSize:    100, // Small size to force multiple chunks
		OverlapRatio: 0.2,
		MinChunkSize: 20,
	}

	chunks, err := svc.ChunkPlainText(ctx, content, "doc123", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}

	// Verify each chunk has required fields
	for i, chunk := range chunks {
		if chunk.ID == "" {
			t.Errorf("chunk %d has empty ID", i)
		}
		if chunk.Content == "" {
			t.Errorf("chunk %d has empty content", i)
		}
		if chunk.ParentDocID != "doc123" {
			t.Errorf("chunk %d has wrong parent doc ID: %s", i, chunk.ParentDocID)
		}
		if chunk.TokenCount == 0 {
			t.Errorf("chunk %d has zero token count", i)
		}
		if chunk.Position != i {
			t.Errorf("chunk %d has wrong position: %d", i, chunk.Position)
		}
	}
}

func TestChunkMarkdown(t *testing.T) {
	svc := NewSmartChunkingService()
	ctx := context.Background()

	content := `# Main Title

This is the introduction paragraph.

## Section 1

Content for section 1. It has multiple sentences. This helps test chunking.

### Subsection 1.1

More detailed content here.

## Section 2

Content for section 2. Another paragraph of text.`

	opts := DefaultChunkOptions()
	chunks, err := svc.ChunkMarkdown(ctx, content, "doc456", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}

	// Check that chunks have markdown metadata
	foundHeading := false
	for _, chunk := range chunks {
		if heading, ok := chunk.Metadata["heading"]; ok && heading != "" {
			foundHeading = true
		}
		if chunkType, ok := chunk.Metadata["type"]; ok {
			if chunkType != "markdown_section" {
				t.Errorf("expected type to be markdown_section, got %v", chunkType)
			}
		}
	}

	if !foundHeading {
		t.Error("expected at least one chunk to have a heading")
	}
}

func TestChunkCode_Go(t *testing.T) {
	svc := NewSmartChunkingService()
	ctx := context.Background()

	content := `package main

import "fmt"

func main() {
	fmt.Println("Hello World")
}

func helper() string {
	return "helper"
}

type MyStruct struct {
	Field string
}`

	opts := DefaultChunkOptions()
	chunks, err := svc.ChunkCode(ctx, content, "go", "doc789", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}

	// Check that chunks have code metadata
	foundFunction := false
	for _, chunk := range chunks {
		if blockType, ok := chunk.Metadata["block_type"]; ok {
			if blockType == "function" {
				foundFunction = true
			}
		}
		if lang, ok := chunk.Metadata["language"]; ok {
			if lang != "go" {
				t.Errorf("expected language to be go, got %v", lang)
			}
		}
	}

	if !foundFunction {
		t.Error("expected at least one function block")
	}
}

func TestChunkCode_Python(t *testing.T) {
	svc := NewSmartChunkingService()
	ctx := context.Background()

	content := `import sys

def main():
    print("Hello World")
    return 0

class MyClass:
    def __init__(self):
        self.value = 42

    def method(self):
        return self.value`

	opts := DefaultChunkOptions()
	chunks, err := svc.ChunkCode(ctx, content, "python", "doc999", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}

	// Verify python-specific parsing
	foundClass := false
	for _, chunk := range chunks {
		if blockType, ok := chunk.Metadata["block_type"]; ok {
			if blockType == "class" {
				foundClass = true
			}
		}
	}

	if !foundClass {
		t.Error("expected at least one class block")
	}
}

func TestChunkDocument_Routing(t *testing.T) {
	svc := NewSmartChunkingService()
	ctx := context.Background()
	opts := DefaultChunkOptions()

	testCases := []struct {
		name    string
		docType DocumentType
		content string
	}{
		{
			name:    "markdown",
			docType: DocTypeMarkdown,
			content: "# Title\n\nContent here.",
		},
		{
			name:    "code",
			docType: DocTypeCode,
			content: "func main() {}\n",
		},
		{
			name:    "plain text",
			docType: DocTypePlainText,
			content: "Just some plain text content.\n\nAnother paragraph.",
		},
		{
			name:    "json",
			docType: DocTypeJSON,
			content: `{"key": "value", "nested": {"a": 1}}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			chunks, err := svc.ChunkDocument(ctx, tc.content, tc.docType, "test-doc", opts)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(chunks) == 0 {
				t.Error("expected at least one chunk")
			}
		})
	}
}

func TestChunkOptions_Validation(t *testing.T) {
	svc := NewSmartChunkingService()
	ctx := context.Background()

	testCases := []struct {
		name        string
		opts        ChunkOptions
		shouldError bool
	}{
		{
			name: "valid options",
			opts: ChunkOptions{
				ChunkSize:    512,
				OverlapRatio: 0.2,
				MinChunkSize: 128,
			},
			shouldError: false,
		},
		{
			name: "chunk size less than min",
			opts: ChunkOptions{
				ChunkSize:    100,
				OverlapRatio: 0.2,
				MinChunkSize: 200,
			},
			shouldError: true,
		},
		{
			name: "invalid overlap ratio - negative",
			opts: ChunkOptions{
				ChunkSize:    512,
				OverlapRatio: -0.1,
				MinChunkSize: 128,
			},
			shouldError: true,
		},
		{
			name: "invalid overlap ratio - too large",
			opts: ChunkOptions{
				ChunkSize:    512,
				OverlapRatio: 1.5,
				MinChunkSize: 128,
			},
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.ChunkPlainText(ctx, "Some test content.", "doc", tc.opts)
			if tc.shouldError && err == nil {
				t.Error("expected error but got none")
			}
			if !tc.shouldError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	svc := NewSmartChunkingService()

	testCases := []struct {
		text           string
		expectedTokens int
	}{
		{"", 0},
		{"test", 1},
		{"this is a test", 3},
		{"a longer piece of text with many words", 9},
	}

	for _, tc := range testCases {
		tokens := svc.estimateTokens(tc.text)
		// Allow some variance due to estimation
		if tokens < tc.expectedTokens-1 || tokens > tc.expectedTokens+1 {
			t.Errorf("text %q: expected ~%d tokens, got %d", tc.text, tc.expectedTokens, tokens)
		}
	}
}

func TestDetectLanguage(t *testing.T) {
	svc := NewSmartChunkingService()

	testCases := []struct {
		content  string
		expected string
	}{
		{
			content:  "package main\n\nfunc main() {}",
			expected: "go",
		},
		{
			content:  "def main():\n    print('hello')",
			expected: "python",
		},
		{
			content:  "function test() { return true; }",
			expected: "javascript",
		},
		{
			content:  "public class Main { }",
			expected: "java",
		},
		{
			content:  "#include <stdio.h>\nint main() {}",
			expected: "c",
		},
		{
			content:  "some random text",
			expected: "unknown",
		},
	}

	for _, tc := range testCases {
		detected := svc.detectLanguage(tc.content)
		if detected != tc.expected {
			t.Errorf("content %q: expected %s, got %s", tc.content, tc.expected, detected)
		}
	}
}

func TestApplyOverlap(t *testing.T) {
	svc := NewSmartChunkingService()

	chunks := []Chunk{
		{
			ID:          "1",
			Content:     "First chunk content. This is sentence one. This is sentence two.",
			ParentDocID: "doc",
			Position:    0,
			Metadata:    map[string]interface{}{},
		},
		{
			ID:          "2",
			Content:     "Second chunk content. Different text here.",
			ParentDocID: "doc",
			Position:    1,
			Metadata:    map[string]interface{}{},
		},
	}

	overlapped := svc.applyOverlap(chunks, 10)

	if len(overlapped) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(overlapped))
	}

	// Second chunk should have overlap marker
	if !strings.Contains(overlapped[1].Content, "...") {
		t.Error("expected second chunk to contain overlap marker")
	}

	// Second chunk should have overlap metadata
	if hasOverlap, ok := overlapped[1].Metadata["has_overlap"]; !ok || !hasOverlap.(bool) {
		t.Error("expected second chunk to have has_overlap metadata")
	}
}

func TestGetChunkStatistics(t *testing.T) {
	svc := NewSmartChunkingService()

	chunks := []Chunk{
		{TokenCount: 100},
		{TokenCount: 200},
		{TokenCount: 150},
	}

	stats := svc.GetChunkStatistics(chunks)

	if stats["count"] != 3 {
		t.Errorf("expected count 3, got %v", stats["count"])
	}
	if stats["total_tokens"] != 450 {
		t.Errorf("expected total_tokens 450, got %v", stats["total_tokens"])
	}
	if stats["avg_tokens"] != 150 {
		t.Errorf("expected avg_tokens 150, got %v", stats["avg_tokens"])
	}
	if stats["min_tokens"] != 100 {
		t.Errorf("expected min_tokens 100, got %v", stats["min_tokens"])
	}
	if stats["max_tokens"] != 200 {
		t.Errorf("expected max_tokens 200, got %v", stats["max_tokens"])
	}
}

func TestGetChunkStatistics_Empty(t *testing.T) {
	svc := NewSmartChunkingService()

	stats := svc.GetChunkStatistics([]Chunk{})

	if stats["count"] != 0 {
		t.Errorf("expected count 0, got %v", stats["count"])
	}
}

func TestValidateChunk(t *testing.T) {
	svc := NewSmartChunkingService()
	opts := DefaultChunkOptions()

	testCases := []struct {
		name    string
		chunk   Chunk
		valid   bool
		errMsg  string
	}{
		{
			name: "valid chunk",
			chunk: Chunk{
				Content:    "This is valid content with enough text to meet requirements.",
				TokenCount: 150,
			},
			valid: true,
		},
		{
			name: "too small",
			chunk: Chunk{
				Content:    "Small",
				TokenCount: 50,
			},
			valid:  false,
			errMsg: "too small",
		},
		{
			name: "too large",
			chunk: Chunk{
				Content:    strings.Repeat("word ", 2000),
				TokenCount: 1500,
			},
			valid:  false,
			errMsg: "too large",
		},
		{
			name: "empty content",
			chunk: Chunk{
				Content:    "",
				TokenCount: 0,
			},
			valid:  false,
			errMsg: "empty",
		},
		{
			name: "only whitespace",
			chunk: Chunk{
				Content:    "   \n\n\t\t   ",
				TokenCount: 150,
			},
			valid:  false,
			errMsg: "empty",
		},
		{
			name: "only punctuation",
			chunk: Chunk{
				Content:    ".,!?;:()[]{}",
				TokenCount: 150,
			},
			valid:  false,
			errMsg: "no alphanumeric",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid, msg := svc.ValidateChunk(tc.chunk, opts)
			if valid != tc.valid {
				t.Errorf("expected valid=%v, got valid=%v (msg: %s)", tc.valid, valid, msg)
			}
			if !tc.valid && !strings.Contains(strings.ToLower(msg), tc.errMsg) {
				t.Errorf("expected error message to contain %q, got %q", tc.errMsg, msg)
			}
		})
	}
}

func TestNormalizeWhitespace(t *testing.T) {
	svc := NewSmartChunkingService()

	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "normal text",
			expected: "normal text",
		},
		{
			input:    "text  with   extra    spaces",
			expected: "text with extra spaces",
		},
		{
			input:    "text\n\n\n\n\nwith many newlines",
			expected: "text\n\nwith many newlines",
		},
		{
			input:    "  \n\n  leading and trailing  \n\n  ",
			expected: "leading and trailing",
		},
	}

	for _, tc := range testCases {
		result := svc.normalizeWhitespace(tc.input)
		if result != tc.expected {
			t.Errorf("input %q: expected %q, got %q", tc.input, tc.expected, result)
		}
	}
}

func TestSplitMarkdownSections(t *testing.T) {
	svc := NewSmartChunkingService()

	content := `# Title
Content under title

## Section 1
Content under section 1

### Subsection 1.1
Content under subsection

## Section 2
Content under section 2`

	sections := svc.splitMarkdownSections(content)

	if len(sections) == 0 {
		t.Fatal("expected at least one section")
	}

	// Check that we found headers
	foundTitle := false
	foundSection := false
	for _, section := range sections {
		if section.Heading == "Title" && section.Level == 1 {
			foundTitle = true
		}
		if section.Heading == "Section 1" && section.Level == 2 {
			foundSection = true
		}
	}

	if !foundTitle {
		t.Error("expected to find title heading")
	}
	if !foundSection {
		t.Error("expected to find section heading")
	}
}

func TestSplitParagraphs(t *testing.T) {
	svc := NewSmartChunkingService()

	content := `First paragraph.

Second paragraph.


Third paragraph with extra newlines.`

	paragraphs := svc.splitParagraphs(content)

	if len(paragraphs) != 3 {
		t.Errorf("expected 3 paragraphs, got %d", len(paragraphs))
	}

	for i, p := range paragraphs {
		if strings.TrimSpace(p) == "" {
			t.Errorf("paragraph %d is empty", i)
		}
	}
}

func TestGenerateChunkID(t *testing.T) {
	svc := NewSmartChunkingService()

	id1 := svc.generateChunkID("doc1", 0, "content")
	id2 := svc.generateChunkID("doc1", 0, "content")
	id3 := svc.generateChunkID("doc1", 1, "content")
	id4 := svc.generateChunkID("doc2", 0, "content")

	// Same inputs should generate same ID
	if id1 != id2 {
		t.Error("expected same IDs for same inputs")
	}

	// Different inputs should generate different IDs
	if id1 == id3 {
		t.Error("expected different IDs for different positions")
	}
	if id1 == id4 {
		t.Error("expected different IDs for different parent docs")
	}

	// Check ID format
	if len(id1) != 16 {
		t.Errorf("expected ID length 16, got %d", len(id1))
	}
}

func TestChunkLargeDocument(t *testing.T) {
	svc := NewSmartChunkingService()
	ctx := context.Background()

	// Create a large document
	var content strings.Builder
	for i := 0; i < 100; i++ {
		content.WriteString(fmt.Sprintf("This is paragraph %d. It contains multiple sentences. ", i))
		content.WriteString("Each paragraph has enough content to be meaningful. ")
		content.WriteString("We want to test chunking of large documents.\n\n")
	}

	opts := ChunkOptions{
		ChunkSize:    200,
		OverlapRatio: 0.2,
		MinChunkSize: 50,
	}

	chunks, err := svc.ChunkPlainText(ctx, content.String(), "large-doc", opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chunks) <= 1 {
		t.Error("expected large document to be split into multiple chunks")
	}

	// Verify chunks are roughly the right size
	for i, chunk := range chunks {
		if chunk.TokenCount > opts.ChunkSize*2 {
			t.Errorf("chunk %d is too large: %d tokens", i, chunk.TokenCount)
		}
		if chunk.TokenCount < opts.MinChunkSize && i < len(chunks)-1 {
			t.Errorf("chunk %d is too small: %d tokens", i, chunk.TokenCount)
		}
	}
}

func TestEmptyContent(t *testing.T) {
	svc := NewSmartChunkingService()
	ctx := context.Background()
	opts := DefaultChunkOptions()

	_, err := svc.ChunkDocument(ctx, "", DocTypePlainText, "doc", opts)
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func BenchmarkChunkPlainText(b *testing.B) {
	svc := NewSmartChunkingService()
	ctx := context.Background()
	opts := DefaultChunkOptions()

	content := strings.Repeat("This is a test sentence. ", 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.ChunkPlainText(ctx, content, "doc", opts)
	}
}

func BenchmarkChunkMarkdown(b *testing.B) {
	svc := NewSmartChunkingService()
	ctx := context.Background()
	opts := DefaultChunkOptions()

	var content strings.Builder
	for i := 0; i < 50; i++ {
		content.WriteString(fmt.Sprintf("## Section %d\n\nContent for section %d.\n\n", i, i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = svc.ChunkMarkdown(ctx, content.String(), "doc", opts)
	}
}
