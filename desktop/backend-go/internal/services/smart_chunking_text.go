package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// ChunkPlainText chunks plain text respecting natural boundaries
func (s *SmartChunkingService) ChunkPlainText(ctx context.Context, content string, parentDocID string, options ChunkOptions) ([]Chunk, error) {
	// Apply defaults if not set
	if options.ChunkSize == 0 {
		options.ChunkSize = 512
	}
	if options.OverlapRatio == 0 {
		options.OverlapRatio = 0.2
	}
	if options.MinChunkSize == 0 {
		options.MinChunkSize = 128
	}

	// Validate options
	if options.ChunkSize < options.MinChunkSize {
		return nil, fmt.Errorf("chunk_size (%d) must be >= min_chunk_size (%d)", options.ChunkSize, options.MinChunkSize)
	}
	if options.OverlapRatio < 0 || options.OverlapRatio >= 1 {
		return nil, fmt.Errorf("overlap_ratio must be between 0 and 1")
	}

	// Split by paragraphs (double newline)
	paragraphs := s.splitParagraphs(content)

	position := 0
	chunks := s.chunkParagraphs(paragraphs, position, parentDocID, options, map[string]interface{}{
		"type": "plain_text",
	})

	// Apply overlap between chunks
	overlapSize := int(float64(options.ChunkSize) * options.OverlapRatio)
	if options.OverlapRatio > 0 && len(chunks) > 1 {
		chunks = s.applyOverlap(chunks, overlapSize)
	}

	return chunks, nil
}

// ChunkStructured chunks structured data (JSON, XML)
func (s *SmartChunkingService) ChunkStructured(ctx context.Context, content string, format string, parentDocID string, options ChunkOptions) ([]Chunk, error) {
	// For structured data, split by top-level elements/objects
	// This is a simplified implementation - production might use proper parsers

	var chunks []Chunk
	position := 0

	// Simple line-based splitting for now
	lines := strings.Split(content, "\n")
	currentChunk := []string{}
	currentTokens := 0

	for _, line := range lines {
		lineTokens := s.estimateTokens(line)

		if currentTokens+lineTokens > options.ChunkSize && len(currentChunk) > 0 {
			// Create chunk from accumulated lines
			chunkContent := strings.Join(currentChunk, "\n")
			chunk := s.createChunk(chunkContent, position, parentDocID, map[string]interface{}{
				"type":   "structured",
				"format": format,
			})
			chunks = append(chunks, chunk)
			position++

			// Start new chunk with overlap
			overlapLines := int(float64(len(currentChunk)) * options.OverlapRatio)
			if overlapLines > 0 {
				currentChunk = currentChunk[len(currentChunk)-overlapLines:]
				currentTokens = s.estimateTokens(strings.Join(currentChunk, "\n"))
			} else {
				currentChunk = []string{}
				currentTokens = 0
			}
		}

		currentChunk = append(currentChunk, line)
		currentTokens += lineTokens
	}

	// Add remaining content as final chunk
	if len(currentChunk) > 0 {
		chunkContent := strings.Join(currentChunk, "\n")
		chunk := s.createChunk(chunkContent, position, parentDocID, map[string]interface{}{
			"type":   "structured",
			"format": format,
		})
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// splitParagraphs splits text into paragraphs
func (s *SmartChunkingService) splitParagraphs(content string) []string {
	// Split by double newline or more
	paragraphRegex := regexp.MustCompile(`\n\s*\n`)
	paragraphs := paragraphRegex.Split(content, -1)

	var result []string
	for _, p := range paragraphs {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// chunkParagraphs combines paragraphs into chunks
func (s *SmartChunkingService) chunkParagraphs(paragraphs []string, startPosition int, parentDocID string, options ChunkOptions, baseMetadata map[string]interface{}) []Chunk {
	var chunks []Chunk
	position := startPosition

	var currentChunk []string
	currentTokens := 0

	for _, para := range paragraphs {
		paraTokens := s.estimateTokens(para)

		// If single paragraph exceeds chunk size, split by sentences
		if paraTokens > options.ChunkSize {
			// Flush current chunk
			if len(currentChunk) > 0 {
				chunkContent := strings.Join(currentChunk, "\n\n")
				metadata := s.copyMetadata(baseMetadata)
				metadata["paragraph_count"] = len(currentChunk)
				chunk := s.createChunk(chunkContent, position, parentDocID, metadata)
				chunks = append(chunks, chunk)
				position++
				currentChunk = []string{}
				currentTokens = 0
			}

			// Split large paragraph by sentences
			sentences := s.splitSentences(para)
			sentenceChunks := s.chunkSentences(sentences, position, parentDocID, options, baseMetadata)
			chunks = append(chunks, sentenceChunks...)
			position += len(sentenceChunks)
			continue
		}

		// Would adding this paragraph exceed chunk size?
		if currentTokens+paraTokens > options.ChunkSize && len(currentChunk) > 0 {
			// Create chunk from accumulated paragraphs
			chunkContent := strings.Join(currentChunk, "\n\n")
			metadata := s.copyMetadata(baseMetadata)
			metadata["paragraph_count"] = len(currentChunk)
			chunk := s.createChunk(chunkContent, position, parentDocID, metadata)
			chunks = append(chunks, chunk)
			position++

			// Start new chunk
			currentChunk = []string{}
			currentTokens = 0
		}

		currentChunk = append(currentChunk, para)
		currentTokens += paraTokens
	}

	// Add remaining content as final chunk
	if len(currentChunk) > 0 {
		chunkContent := strings.Join(currentChunk, "\n\n")
		metadata := s.copyMetadata(baseMetadata)
		metadata["paragraph_count"] = len(currentChunk)
		chunk := s.createChunk(chunkContent, position, parentDocID, metadata)
		chunks = append(chunks, chunk)
	}

	return chunks
}

// chunkSentences combines sentences into chunks
func (s *SmartChunkingService) chunkSentences(sentences []string, startPosition int, parentDocID string, options ChunkOptions, baseMetadata map[string]interface{}) []Chunk {
	var chunks []Chunk
	position := startPosition

	var currentChunk []string
	currentTokens := 0

	for _, sentence := range sentences {
		sentenceTokens := s.estimateTokens(sentence)

		if currentTokens+sentenceTokens > options.ChunkSize && len(currentChunk) > 0 {
			chunkContent := strings.Join(currentChunk, " ")
			metadata := s.copyMetadata(baseMetadata)
			metadata["sentence_count"] = len(currentChunk)
			chunk := s.createChunk(chunkContent, position, parentDocID, metadata)
			chunks = append(chunks, chunk)
			position++

			currentChunk = []string{}
			currentTokens = 0
		}

		currentChunk = append(currentChunk, sentence)
		currentTokens += sentenceTokens
	}

	if len(currentChunk) > 0 {
		chunkContent := strings.Join(currentChunk, " ")
		metadata := s.copyMetadata(baseMetadata)
		metadata["sentence_count"] = len(currentChunk)
		chunk := s.createChunk(chunkContent, position, parentDocID, metadata)
		chunks = append(chunks, chunk)
	}

	return chunks
}

// splitSentences splits text into sentences
func (s *SmartChunkingService) splitSentences(text string) []string {
	// Simple sentence splitting - production might use NLP library
	sentenceRegex := regexp.MustCompile(`[.!?]+\s+`)
	sentences := sentenceRegex.Split(text, -1)

	var result []string
	for _, s := range sentences {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
