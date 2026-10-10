package services

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// parseFrontmatter parses YAML frontmatter
func (s *BlockMapperService) parseFrontmatter(lines []string, lineNum int) (*Block, int) {
	var content strings.Builder
	consumed := 1 // First ---

	for i := lineNum + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			consumed++
			break
		}
		content.WriteString(lines[i])
		content.WriteString("\n")
		consumed++
	}

	return &Block{
		ID:      uuid.New().String(),
		Type:    BlockTypeFrontmatter,
		Content: strings.TrimSpace(content.String()),
		Hash:    s.hashContent(content.String()),
	}, consumed
}

// parseThinking parses thinking tags
func (s *BlockMapperService) parseThinking(lines []string, lineNum int) (*Block, int) {
	var content strings.Builder
	consumed := 0
	inThinking := false

	for i := lineNum; i < len(lines); i++ {
		line := lines[i]
		consumed++

		if strings.Contains(line, "<thinking") {
			inThinking = true
			// Extract content after opening tag if on same line
			if idx := strings.Index(line, ">"); idx != -1 {
				content.WriteString(line[idx+1:])
				content.WriteString("\n")
			}
			continue
		}

		if strings.Contains(line, "</thinking>") {
			// Extract content before closing tag
			if idx := strings.Index(line, "</thinking>"); idx > 0 {
				content.WriteString(line[:idx])
			}
			break
		}

		if inThinking {
			content.WriteString(line)
			content.WriteString("\n")
		}
	}

	return &Block{
		ID:      uuid.New().String(),
		Type:    BlockTypeThinking,
		Content: strings.TrimSpace(content.String()),
		Hash:    s.hashContent(content.String()),
	}, consumed
}

// parseArtifact parses artifact tags
func (s *BlockMapperService) parseArtifact(lines []string, lineNum int) (*Block, int) {
	var content strings.Builder
	consumed := 0
	metadata := make(map[string]interface{})

	openingLine := lines[lineNum]

	// Extract attributes from opening tag
	if attrMatch := regexp.MustCompile(`identifier="([^"]+)"`).FindStringSubmatch(openingLine); len(attrMatch) > 1 {
		metadata["identifier"] = attrMatch[1]
	}
	if attrMatch := regexp.MustCompile(`type="([^"]+)"`).FindStringSubmatch(openingLine); len(attrMatch) > 1 {
		metadata["type"] = attrMatch[1]
	}
	if attrMatch := regexp.MustCompile(`title="([^"]+)"`).FindStringSubmatch(openingLine); len(attrMatch) > 1 {
		metadata["title"] = attrMatch[1]
	}
	if attrMatch := regexp.MustCompile(`language="([^"]+)"`).FindStringSubmatch(openingLine); len(attrMatch) > 1 {
		metadata["language"] = attrMatch[1]
	}

	for i := lineNum; i < len(lines); i++ {
		line := lines[i]
		consumed++

		if i == lineNum {
			// Check if content starts on same line
			if idx := strings.Index(line, ">"); idx != -1 && idx < len(line)-1 {
				content.WriteString(line[idx+1:])
				content.WriteString("\n")
			}
			continue
		}

		if strings.Contains(line, "</artifact>") {
			if idx := strings.Index(line, "</artifact>"); idx > 0 {
				content.WriteString(line[:idx])
			}
			break
		}

		content.WriteString(line)
		content.WriteString("\n")
	}

	return &Block{
		ID:       uuid.New().String(),
		Type:     BlockTypeArtifact,
		Content:  strings.TrimSpace(content.String()),
		Metadata: metadata,
		Hash:     s.hashContent(content.String()),
	}, consumed
}

// parseCodeBlock parses fenced code blocks
func (s *BlockMapperService) parseCodeBlock(lines []string, lineNum int) (*Block, int) {
	firstLine := strings.TrimSpace(lines[lineNum])
	language := strings.TrimPrefix(firstLine, "```")
	language = strings.TrimSpace(language)

	var content strings.Builder
	consumed := 1

	for i := lineNum + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "```" {
			consumed++
			break
		}
		content.WriteString(lines[i])
		content.WriteString("\n")
		consumed++
	}

	return &Block{
		ID:       uuid.New().String(),
		Type:     BlockTypeCode,
		Content:  strings.TrimSuffix(content.String(), "\n"),
		Language: language,
		Hash:     s.hashContent(content.String()),
	}, consumed
}

// parseHeading parses heading lines
func (s *BlockMapperService) parseHeading(line string) *Block {
	level := 0
	for _, c := range line {
		if c == '#' {
			level++
		} else {
			break
		}
	}

	content := strings.TrimSpace(strings.TrimLeft(line, "# "))

	return &Block{
		ID:      uuid.New().String(),
		Type:    BlockTypeHeading,
		Content: content,
		Level:   level,
		Hash:    s.hashContent(content),
	}
}
