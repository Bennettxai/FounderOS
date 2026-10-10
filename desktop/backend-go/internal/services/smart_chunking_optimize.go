package services

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// GetChunkStatistics returns statistics about chunks
func (s *SmartChunkingService) GetChunkStatistics(chunks []Chunk) map[string]interface{} {
	if len(chunks) == 0 {
		return map[string]interface{}{
			"count":        0,
			"total_tokens": 0,
			"avg_tokens":   0,
			"min_tokens":   0,
			"max_tokens":   0,
		}
	}

	totalTokens := 0
	minTokens := chunks[0].TokenCount
	maxTokens := chunks[0].TokenCount

	for _, chunk := range chunks {
		totalTokens += chunk.TokenCount
		if chunk.TokenCount < minTokens {
			minTokens = chunk.TokenCount
		}
		if chunk.TokenCount > maxTokens {
			maxTokens = chunk.TokenCount
		}
	}

	return map[string]interface{}{
		"count":        len(chunks),
		"total_tokens": totalTokens,
		"avg_tokens":   totalTokens / len(chunks),
		"min_tokens":   minTokens,
		"max_tokens":   maxTokens,
	}
}

// OptimizeChunks post-processes chunks to improve quality
func (s *SmartChunkingService) OptimizeChunks(chunks []Chunk, options ChunkOptions) []Chunk {
	var optimized []Chunk

	for i, chunk := range chunks {
		// Skip very small chunks unless it's the last one
		if chunk.TokenCount < options.MinChunkSize && i < len(chunks)-1 {
			// Merge with next chunk
			if i+1 < len(chunks) {
				nextChunk := chunks[i+1]
				mergedContent := chunk.Content + "\n\n" + nextChunk.Content
				mergedTokens := s.estimateTokens(mergedContent)

				if mergedTokens <= options.ChunkSize {
					// Create merged chunk
					merged := s.createChunk(mergedContent, chunk.Position, chunk.ParentDocID, chunk.Metadata)
					optimized = append(optimized, merged)
					// Skip next chunk since we merged it
					i++
					continue
				}
			}
		}

		// Remove excessive whitespace
		chunk.Content = s.normalizeWhitespace(chunk.Content)
		chunk.TokenCount = s.estimateTokens(chunk.Content)

		optimized = append(optimized, chunk)
	}

	return optimized
}

// normalizeWhitespace removes excessive whitespace
func (s *SmartChunkingService) normalizeWhitespace(text string) string {
	// Replace multiple spaces with single space
	text = regexp.MustCompile(`[ \t]+`).ReplaceAllString(text, " ")

	// Replace more than 2 newlines with 2 newlines
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")

	// Trim leading/trailing whitespace
	text = strings.TrimSpace(text)

	return text
}

// ValidateChunk validates a chunk meets quality criteria
func (s *SmartChunkingService) ValidateChunk(chunk Chunk, options ChunkOptions) (bool, string) {
	// Check content is not empty first (before size checks)
	if strings.TrimSpace(chunk.Content) == "" {
		return false, "chunk content is empty"
	}

	// Check content is not just whitespace or punctuation
	hasAlphanumeric := false
	for _, r := range chunk.Content {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			hasAlphanumeric = true
			break
		}
	}
	if !hasAlphanumeric {
		return false, "chunk contains no alphanumeric characters"
	}

	// Check minimum size
	if chunk.TokenCount < options.MinChunkSize {
		return false, fmt.Sprintf("chunk too small: %d tokens (min: %d)", chunk.TokenCount, options.MinChunkSize)
	}

	// Check maximum size
	if chunk.TokenCount > options.ChunkSize*2 {
		return false, fmt.Sprintf("chunk too large: %d tokens (max: %d)", chunk.TokenCount, options.ChunkSize*2)
	}

	return true, ""
}
