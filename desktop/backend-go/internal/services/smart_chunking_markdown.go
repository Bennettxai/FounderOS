package services

import (
	"context"
	"regexp"
	"strings"
)

// MarkdownSection represents a markdown section
type MarkdownSection struct {
	Heading string
	Level   int
	Content string
}

// ChunkMarkdown chunks markdown content respecting structure
func (s *SmartChunkingService) ChunkMarkdown(ctx context.Context, content string, parentDocID string, options ChunkOptions) ([]Chunk, error) {
	var chunks []Chunk

	// Split by markdown sections (headers)
	sections := s.splitMarkdownSections(content)

	position := 0
	overlapSize := int(float64(options.ChunkSize) * options.OverlapRatio)

	for sectionIdx, section := range sections {
		// Check if section fits in one chunk
		sectionTokens := s.estimateTokens(section.Content)

		if sectionTokens <= options.ChunkSize {
			// Section fits in one chunk
			chunk := s.createChunk(section.Content, position, parentDocID, map[string]interface{}{
				"type":        "markdown_section",
				"heading":     section.Heading,
				"level":       section.Level,
				"section_idx": sectionIdx,
			})
			chunks = append(chunks, chunk)
			position++
		} else {
			// Section too large, split by paragraphs
			paragraphs := s.splitParagraphs(section.Content)
			subChunks := s.chunkParagraphs(paragraphs, position, parentDocID, options, map[string]interface{}{
				"type":        "markdown_section",
				"heading":     section.Heading,
				"level":       section.Level,
				"section_idx": sectionIdx,
			})
			chunks = append(chunks, subChunks...)
			position += len(subChunks)
		}
	}

	// Apply overlap between chunks
	if options.OverlapRatio > 0 && len(chunks) > 1 {
		chunks = s.applyOverlap(chunks, overlapSize)
	}

	return chunks, nil
}

// splitMarkdownSections splits markdown by headers
func (s *SmartChunkingService) splitMarkdownSections(content string) []MarkdownSection {
	var sections []MarkdownSection
	lines := strings.Split(content, "\n")

	var currentSection *MarkdownSection
	headerRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

	for _, line := range lines {
		matches := headerRegex.FindStringSubmatch(line)
		if matches != nil {
			// Save previous section
			if currentSection != nil {
				currentSection.Content = strings.TrimSpace(currentSection.Content)
				sections = append(sections, *currentSection)
			}

			// Start new section
			level := len(matches[1])
			heading := matches[2]
			currentSection = &MarkdownSection{
				Heading: heading,
				Level:   level,
				Content: line + "\n",
			}
		} else if currentSection != nil {
			currentSection.Content += line + "\n"
		} else {
			// Content before first header
			if len(sections) == 0 {
				currentSection = &MarkdownSection{
					Heading: "",
					Level:   0,
					Content: line + "\n",
				}
			}
		}
	}

	// Add final section
	if currentSection != nil {
		currentSection.Content = strings.TrimSpace(currentSection.Content)
		sections = append(sections, *currentSection)
	}

	return sections
}
