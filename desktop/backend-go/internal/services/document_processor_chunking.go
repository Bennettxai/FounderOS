package services

import (
	"regexp"
	"strings"
)

// ============================================================================
// Chunking
// ============================================================================

// chunkDocument splits document into chunks for RAG
func (p *DocumentProcessor) chunkDocument(text string, opts ChunkingOptions) []DocumentChunk {
	if len(text) == 0 {
		return nil
	}

	var currentSection string

	// Split by headers if enabled
	if opts.SplitOnHeaders {
		return p.chunkByHeaders(text, opts)
	}
	return p.chunkBySize(text, opts, currentSection)
}

// chunkByHeaders splits text at markdown headers
func (p *DocumentProcessor) chunkByHeaders(text string, opts ChunkingOptions) []DocumentChunk {
	var chunks []DocumentChunk

	// Regex for markdown headers
	headerPattern := regexp.MustCompile(`(?m)^(#{1,6})\s+(.+)$`)

	// Find all headers
	matches := headerPattern.FindAllStringSubmatchIndex(text, -1)

	if len(matches) == 0 {
		// No headers, chunk by size
		return p.chunkBySize(text, opts, "")
	}

	// Process sections between headers
	for i, match := range matches {
		headerStart := match[0]
		headerEnd := match[1]
		titleStart := match[4]
		titleEnd := match[5]

		sectionTitle := text[titleStart:titleEnd]

		// Determine section end
		var sectionEnd int
		if i+1 < len(matches) {
			sectionEnd = matches[i+1][0]
		} else {
			sectionEnd = len(text)
		}

		sectionContent := text[headerEnd:sectionEnd]
		sectionContent = strings.TrimSpace(sectionContent)

		if len(sectionContent) == 0 {
			continue
		}

		// If section is too large, chunk it further
		if len(sectionContent) > opts.MaxChunkSize {
			subChunks := p.chunkBySize(sectionContent, opts, sectionTitle)
			for _, sc := range subChunks {
				sc.StartChar += headerStart
				sc.EndChar += headerStart
				chunks = append(chunks, sc)
			}
		} else {
			chunks = append(chunks, DocumentChunk{
				Content:      sectionContent,
				TokenCount:   p.estimateTokens(sectionContent),
				StartChar:    headerStart,
				EndChar:      sectionEnd,
				SectionTitle: sectionTitle,
				ChunkType:    "text",
			})
		}
	}

	// Handle text before first header
	if len(matches) > 0 && matches[0][0] > 0 {
		preContent := strings.TrimSpace(text[:matches[0][0]])
		if len(preContent) > 0 {
			preChunks := p.chunkBySize(preContent, opts, "")
			chunks = append(preChunks, chunks...)
		}
	}

	return chunks
}

// chunkBySize splits text into fixed-size chunks with overlap
func (p *DocumentProcessor) chunkBySize(text string, opts ChunkingOptions, sectionTitle string) []DocumentChunk {
	var chunks []DocumentChunk

	if len(text) <= opts.MaxChunkSize {
		chunks = append(chunks, DocumentChunk{
			Content:      text,
			TokenCount:   p.estimateTokens(text),
			StartChar:    0,
			EndChar:      len(text),
			SectionTitle: sectionTitle,
			ChunkType:    "text",
		})
		return chunks
	}

	// Split into sentences for better boundaries
	sentences := p.splitIntoSentences(text)

	var currentChunk strings.Builder
	var chunkStart int
	currentStart := 0

	for _, sentence := range sentences {
		// If adding this sentence would exceed max size, finalize current chunk
		if currentChunk.Len()+len(sentence) > opts.MaxChunkSize && currentChunk.Len() > 0 {
			chunks = append(chunks, DocumentChunk{
				Content:      currentChunk.String(),
				TokenCount:   p.estimateTokens(currentChunk.String()),
				StartChar:    chunkStart,
				EndChar:      currentStart,
				SectionTitle: sectionTitle,
				ChunkType:    "text",
			})

			// Start new chunk with overlap
			overlapText := p.getOverlapText(currentChunk.String(), opts.ChunkOverlap)
			currentChunk.Reset()
			currentChunk.WriteString(overlapText)
			chunkStart = currentStart - len(overlapText)
		}

		currentChunk.WriteString(sentence)
		currentStart += len(sentence)
	}

	// Add final chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, DocumentChunk{
			Content:      currentChunk.String(),
			TokenCount:   p.estimateTokens(currentChunk.String()),
			StartChar:    chunkStart,
			EndChar:      len(text),
			SectionTitle: sectionTitle,
			ChunkType:    "text",
		})
	}

	return chunks
}

// splitIntoSentences splits text into sentences
func (p *DocumentProcessor) splitIntoSentences(text string) []string {
	// Simple sentence splitting - could be improved with NLP
	pattern := regexp.MustCompile(`([.!?]+[\s]+)`)
	parts := pattern.Split(text, -1)
	delimiters := pattern.FindAllString(text, -1)

	var sentences []string
	for i, part := range parts {
		if i < len(delimiters) {
			sentences = append(sentences, part+delimiters[i])
		} else {
			sentences = append(sentences, part)
		}
	}

	return sentences
}

// getOverlapText returns the last N characters for overlap
func (p *DocumentProcessor) getOverlapText(text string, overlap int) string {
	if len(text) <= overlap {
		return text
	}

	// Try to find a sentence boundary within overlap range
	overlapText := text[len(text)-overlap:]

	// Find first sentence start
	sentenceStart := strings.Index(overlapText, ". ")
	if sentenceStart > 0 && sentenceStart < overlap/2 {
		return overlapText[sentenceStart+2:]
	}

	return overlapText
}
