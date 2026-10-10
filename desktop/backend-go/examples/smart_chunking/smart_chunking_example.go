package main

import (
	"context"
	"fmt"
	"log"

	"github.com/rhl/businessos-backend/internal/services"
)

// Example demonstrates how to use SmartChunkingService
func main() {
	// Create the service
	svc := services.NewSmartChunkingService()
	ctx := context.Background()

	// Example 1: Chunk a Markdown document
	markdownContent := `# Introduction to Smart Chunking

Smart chunking is an essential technique for RAG (Retrieval Augmented Generation) systems.

## Why Smart Chunking Matters

When dealing with large documents, we need to:
- Split content into manageable pieces
- Preserve semantic boundaries
- Maintain context through overlap
- Respect document structure

## Implementation Details

Our implementation handles multiple document types and provides configurable options.

### Key Features

1. Automatic boundary detection
2. Overlap strategy for context preservation
3. Token-aware chunking
4. Metadata preservation`

	opts := services.ChunkOptions{
		ChunkSize:    512,
		OverlapRatio: 0.2,
		PreserveCode: true,
		MinChunkSize: 128,
	}

	chunks, err := svc.ChunkMarkdown(ctx, markdownContent, "doc-001", opts)
	if err != nil {
		log.Fatalf("Error chunking markdown: %v", err)
	}

	fmt.Printf("Markdown Document: Created %d chunks\n", len(chunks))
	for i, chunk := range chunks {
		fmt.Printf("Chunk %d: %d tokens, heading: %v\n",
			i+1, chunk.TokenCount, chunk.Metadata["heading"])
	}

	// Example 2: Chunk Go code
	goCode := `package main

import (
	"fmt"
	"log"
)

// main is the entry point
func main() {
	fmt.Println("Hello, World!")
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

// run executes the main logic
func run() error {
	data := processData()
	return saveData(data)
}

// processData processes the input
func processData() []string {
	return []string{"item1", "item2", "item3"}
}

// saveData saves the processed data
func saveData(data []string) error {
	for _, item := range data {
		fmt.Println(item)
	}
	return nil
}

// Config holds application configuration
type Config struct {
	Host string
	Port int
}

// NewConfig creates a new configuration
func NewConfig() *Config {
	return &Config{
		Host: "localhost",
		Port: 8080,
	}
}`

	codeChunks, err := svc.ChunkCode(ctx, goCode, "go", "doc-002", opts)
	if err != nil {
		log.Fatalf("Error chunking code: %v", err)
	}

	fmt.Printf("\nGo Code: Created %d chunks\n", len(codeChunks))
	for i, chunk := range codeChunks {
		fmt.Printf("Chunk %d: %d tokens, type: %v, name: %v\n",
			i+1, chunk.TokenCount, chunk.Metadata["block_type"], chunk.Metadata["name"])
	}

	// Example 3: Chunk plain text
	plainText := `This is a plain text document with multiple paragraphs.

The first paragraph introduces the topic and provides context.

The second paragraph goes into more detail about the subject matter. It contains several sentences that build upon each other.

The third paragraph concludes the document and summarizes the key points. It may also provide recommendations or next steps.

Finally, we have a closing paragraph that wraps everything up.`

	textChunks, err := svc.ChunkPlainText(ctx, plainText, "doc-003", opts)
	if err != nil {
		log.Fatalf("Error chunking text: %v", err)
	}

	fmt.Printf("\nPlain Text: Created %d chunks\n", len(textChunks))
	for i, chunk := range textChunks {
		fmt.Printf("Chunk %d: %d tokens\n", i+1, chunk.TokenCount)
	}

	// Example 4: Get statistics
	stats := svc.GetChunkStatistics(chunks)
	fmt.Printf("\nStatistics for Markdown chunks:\n")
	fmt.Printf("  Total chunks: %v\n", stats["count"])
	fmt.Printf("  Total tokens: %v\n", stats["total_tokens"])
	fmt.Printf("  Avg tokens: %v\n", stats["avg_tokens"])
	fmt.Printf("  Min tokens: %v\n", stats["min_tokens"])
	fmt.Printf("  Max tokens: %v\n", stats["max_tokens"])

	// Example 5: Validate chunks
	fmt.Printf("\nValidating chunks:\n")
	for i, chunk := range chunks {
		valid, msg := svc.ValidateChunk(chunk, opts)
		if !valid {
			fmt.Printf("  Chunk %d is invalid: %s\n", i+1, msg)
		} else {
			fmt.Printf("  Chunk %d is valid\n", i+1)
		}
	}

	// Example 6: Optimize chunks
	optimized := svc.OptimizeChunks(chunks, opts)
	fmt.Printf("\nOptimization: %d chunks -> %d chunks\n", len(chunks), len(optimized))
}

// ExampleChunkingStrategies demonstrates different chunking strategies
func ExampleChunkingStrategies() {
	svc := services.NewSmartChunkingService()
	ctx := context.Background()

	document := `# Technical Documentation

This is a technical document that needs to be chunked for RAG.

## Configuration Options

The system supports various configuration options...`

	// Strategy 1: Small chunks with high overlap (for precise retrieval)
	preciseOpts := services.ChunkOptions{
		ChunkSize:    256,
		OverlapRatio: 0.3, // 30% overlap
		MinChunkSize: 64,
	}

	preciseChunks, _ := svc.ChunkMarkdown(ctx, document, "doc", preciseOpts)
	fmt.Printf("Precise strategy: %d chunks with high overlap\n", len(preciseChunks))

	// Strategy 2: Large chunks with low overlap (for context-rich retrieval)
	contextOpts := services.ChunkOptions{
		ChunkSize:    1024,
		OverlapRatio: 0.1, // 10% overlap
		MinChunkSize: 256,
	}

	contextChunks, _ := svc.ChunkMarkdown(ctx, document, "doc", contextOpts)
	fmt.Printf("Context strategy: %d chunks with more context\n", len(contextChunks))

	// Strategy 3: Balanced approach (default)
	balancedOpts := services.DefaultChunkOptions()
	balancedChunks, _ := svc.ChunkMarkdown(ctx, document, "doc", balancedOpts)
	fmt.Printf("Balanced strategy: %d chunks with balanced settings\n", len(balancedChunks))
}

// ExampleDocumentTypeDetection shows automatic document type handling
func ExampleDocumentTypeDetection() {
	svc := services.NewSmartChunkingService()
	ctx := context.Background()
	opts := services.DefaultChunkOptions()

	documents := map[string]services.DocumentType{
		"# Markdown\n\nContent": services.DocTypeMarkdown,
		"func main() {}":        services.DocTypeCode,
		"Plain text content.":   services.DocTypePlainText,
		`{"key": "value"}`:      services.DocTypeJSON,
	}

	for content, docType := range documents {
		chunks, err := svc.ChunkDocument(ctx, content, docType, "doc", opts)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}
		fmt.Printf("Document type %s: %d chunks\n", docType, len(chunks))
	}
}

// ExampleIntegrationWithRAG shows how to integrate with RAG pipeline
func ExampleIntegrationWithRAG() {
	svc := services.NewSmartChunkingService()
	ctx := context.Background()

	// Simulate a document from a RAG pipeline
	document := `# API Documentation

## Authentication

Use JWT tokens for authentication.

## Endpoints

### GET /api/users
Returns a list of users.

### POST /api/users
Creates a new user.`

	opts := services.ChunkOptions{
		ChunkSize:    512,
		OverlapRatio: 0.2,
		MinChunkSize: 128,
	}

	// Chunk the document
	chunks, err := svc.ChunkDocument(ctx, document, services.DocTypeMarkdown, "api-docs-v1", opts)
	if err != nil {
		log.Fatalf("Chunking failed: %v", err)
	}

	// In a real RAG pipeline, each chunk would be:
	// 1. Embedded using an embedding model
	// 2. Stored in a vector database with metadata
	// 3. Retrieved during query time based on similarity

	fmt.Printf("Created %d chunks for RAG indexing\n", len(chunks))
	for _, chunk := range chunks {
		fmt.Printf("Chunk ID: %s\n", chunk.ID)
		fmt.Printf("  Tokens: %d\n", chunk.TokenCount)
		fmt.Printf("  Position: %d\n", chunk.Position)
		fmt.Printf("  Metadata: %v\n", chunk.Metadata)
		fmt.Printf("  Content preview: %.100s...\n\n", chunk.Content)
	}
}

// ExampleChunkingWithMetadata demonstrates metadata preservation
func ExampleChunkingWithMetadata() {
	svc := services.NewSmartChunkingService()
	ctx := context.Background()
	opts := services.DefaultChunkOptions()

	// Chunk a document
	content := `## Section 1

Important information here.

## Section 2

More details in this section.`

	chunks, _ := svc.ChunkMarkdown(ctx, content, "doc-with-metadata", opts)

	fmt.Println("Chunk Metadata:")
	for i, chunk := range chunks {
		fmt.Printf("\nChunk %d:\n", i+1)
		fmt.Printf("  ID: %s\n", chunk.ID)
		fmt.Printf("  Parent Doc: %s\n", chunk.ParentDocID)
		fmt.Printf("  Position: %d\n", chunk.Position)
		fmt.Printf("  Metadata:\n")
		for key, value := range chunk.Metadata {
			fmt.Printf("    %s: %v\n", key, value)
		}
	}
}

// ExamplePerformanceOptimization shows optimization techniques
func ExamplePerformanceOptimization() {
	svc := services.NewSmartChunkingService()
	ctx := context.Background()
	opts := services.DefaultChunkOptions()

	// Create a moderately large document
	var content string
	for i := 0; i < 50; i++ {
		content += fmt.Sprintf("## Section %d\n\nContent for section %d.\n\n", i, i)
	}

	// Initial chunking
	chunks, _ := svc.ChunkDocument(ctx, content, services.DocTypeMarkdown, "large-doc", opts)
	fmt.Printf("Initial: %d chunks\n", len(chunks))

	// Get statistics before optimization
	statsBefore := svc.GetChunkStatistics(chunks)
	fmt.Printf("Before optimization:\n")
	fmt.Printf("  Avg tokens: %v\n", statsBefore["avg_tokens"])
	fmt.Printf("  Min tokens: %v\n", statsBefore["min_tokens"])
	fmt.Printf("  Max tokens: %v\n", statsBefore["max_tokens"])

	// Optimize chunks (merge small chunks, normalize whitespace)
	optimized := svc.OptimizeChunks(chunks, opts)
	fmt.Printf("\nOptimized: %d chunks\n", len(optimized))

	// Get statistics after optimization
	statsAfter := svc.GetChunkStatistics(optimized)
	fmt.Printf("After optimization:\n")
	fmt.Printf("  Avg tokens: %v\n", statsAfter["avg_tokens"])
	fmt.Printf("  Min tokens: %v\n", statsAfter["min_tokens"])
	fmt.Printf("  Max tokens: %v\n", statsAfter["max_tokens"])
}
