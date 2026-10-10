package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// applyOverlap applies overlap between consecutive chunks
func (s *SmartChunkingService) applyOverlap(chunks []Chunk, overlapTokens int) []Chunk {
	if len(chunks) <= 1 || overlapTokens == 0 {
		return chunks
	}

	for i := 1; i < len(chunks); i++ {
		prevChunk := chunks[i-1]
		currentChunk := chunks[i]

		// Get overlap from previous chunk
		overlapContent := s.extractOverlap(prevChunk.Content, overlapTokens)
		if overlapContent != "" {
			// Prepend overlap to current chunk
			currentChunk.Content = overlapContent + "\n...\n" + currentChunk.Content
			currentChunk.TokenCount = s.estimateTokens(currentChunk.Content)
			currentChunk.Metadata["has_overlap"] = true
			currentChunk.Metadata["overlap_tokens"] = s.estimateTokens(overlapContent)
			chunks[i] = currentChunk
		}
	}

	return chunks
}

// extractOverlap extracts the last N tokens from content
func (s *SmartChunkingService) extractOverlap(content string, targetTokens int) string {
	// Split into sentences and take last few
	sentences := s.splitSentences(content)
	if len(sentences) == 0 {
		return ""
	}

	var overlap []string
	tokens := 0

	// Work backwards from end
	for i := len(sentences) - 1; i >= 0; i-- {
		sentenceTokens := s.estimateTokens(sentences[i])
		if tokens+sentenceTokens > targetTokens && len(overlap) > 0 {
			break
		}
		overlap = append([]string{sentences[i]}, overlap...)
		tokens += sentenceTokens
	}

	return strings.Join(overlap, " ")
}

// createChunk creates a chunk with ID and metadata
func (s *SmartChunkingService) createChunk(content string, position int, parentDocID string, metadata map[string]interface{}) Chunk {
	content = strings.TrimSpace(content)
	tokenCount := s.estimateTokens(content)

	// Generate unique chunk ID
	chunkID := s.generateChunkID(parentDocID, position, content)

	// Add system metadata
	metadata["position"] = position
	metadata["token_count"] = tokenCount
	metadata["char_count"] = len(content)

	return Chunk{
		ID:          chunkID,
		Content:     content,
		TokenCount:  tokenCount,
		Position:    position,
		Metadata:    metadata,
		ParentDocID: parentDocID,
	}
}

// generateChunkID generates a unique ID for a chunk
func (s *SmartChunkingService) generateChunkID(parentDocID string, position int, content string) string {
	// Hash based on parent, position, and content
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s:%d:%s", parentDocID, position, content)))
	hash := hex.EncodeToString(hasher.Sum(nil))
	return hash[:16] // Use first 16 chars
}

// estimateTokens estimates token count for text
func (s *SmartChunkingService) estimateTokens(text string) int {
	// Simple estimation: count characters and divide by chars per token
	// More accurate would be to use a tokenizer like tiktoken
	return int(float64(len(text)) / s.charsPerToken)
}

// copyMetadata creates a copy of metadata map
func (s *SmartChunkingService) copyMetadata(metadata map[string]interface{}) map[string]interface{} {
	copy := make(map[string]interface{})
	for k, v := range metadata {
		copy[k] = v
	}
	return copy
}

// chunkCodeLines splits code into chunks by lines
func (s *SmartChunkingService) chunkCodeLines(content string, startPosition int, parentDocID string, options ChunkOptions, baseMetadata map[string]interface{}) []Chunk {
	var chunks []Chunk
	position := startPosition

	lines := strings.Split(content, "\n")
	var currentChunk []string
	currentTokens := 0

	for _, line := range lines {
		lineTokens := s.estimateTokens(line)

		if currentTokens+lineTokens > options.ChunkSize && len(currentChunk) > 0 {
			chunkContent := strings.Join(currentChunk, "\n")
			metadata := s.copyMetadata(baseMetadata)
			metadata["line_count"] = len(currentChunk)
			chunk := s.createChunk(chunkContent, position, parentDocID, metadata)
			chunks = append(chunks, chunk)
			position++

			currentChunk = []string{}
			currentTokens = 0
		}

		currentChunk = append(currentChunk, line)
		currentTokens += lineTokens
	}

	if len(currentChunk) > 0 {
		chunkContent := strings.Join(currentChunk, "\n")
		metadata := s.copyMetadata(baseMetadata)
		metadata["line_count"] = len(currentChunk)
		chunk := s.createChunk(chunkContent, position, parentDocID, metadata)
		chunks = append(chunks, chunk)
	}

	return chunks
}
