package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// parseBlockquote parses blockquote blocks
func (s *BlockMapperService) parseBlockquote(lines []string, lineNum int) (*Block, int) {
	var content strings.Builder
	consumed := 0

	for i := lineNum; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, ">") && trimmed != "" {
			break
		}
		if trimmed == "" {
			consumed++
			break
		}

		// Remove > prefix
		lineContent := strings.TrimPrefix(trimmed, ">")
		lineContent = strings.TrimPrefix(lineContent, " ")
		content.WriteString(lineContent)
		content.WriteString("\n")
		consumed++
	}

	return &Block{
		ID:      uuid.New().String(),
		Type:    BlockTypeBlockquote,
		Content: strings.TrimSpace(content.String()),
		Hash:    s.hashContent(content.String()),
	}, consumed
}

// parseList parses list blocks
func (s *BlockMapperService) parseList(lines []string, lineNum int) (*Block, int) {
	block := &Block{
		ID:       uuid.New().String(),
		Type:     BlockTypeList,
		Children: make([]*Block, 0),
	}

	consumed := 0
	isOrdered := s.isOrderedListItem(strings.TrimSpace(lines[lineNum]))
	block.Metadata = map[string]interface{}{"ordered": isOrdered}

	for i := lineNum; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			consumed++
			break
		}
		if !s.isListItem(trimmed) {
			break
		}

		// Parse list item
		itemContent := s.extractListItemContent(trimmed)
		isTask, taskComplete := s.isTaskItem(itemContent)

		item := &Block{
			ID:      uuid.New().String(),
			Type:    BlockTypeListItem,
			Content: itemContent,
			Hash:    s.hashContent(itemContent),
		}

		if isTask {
			item.Type = BlockTypeTask
			item.Metadata = map[string]interface{}{"completed": taskComplete}
		}

		block.Children = append(block.Children, item)
		consumed++
	}

	// Build content summary
	var contentBuilder strings.Builder
	for _, child := range block.Children {
		contentBuilder.WriteString(child.Content)
		contentBuilder.WriteString("\n")
	}
	block.Content = strings.TrimSpace(contentBuilder.String())
	block.Hash = s.hashContent(block.Content)

	return block, consumed
}

// parseTable parses markdown tables
func (s *BlockMapperService) parseTable(lines []string, lineNum int) (*Block, int) {
	block := &Block{
		ID:       uuid.New().String(),
		Type:     BlockTypeTable,
		Children: make([]*Block, 0),
	}

	consumed := 0
	var contentBuilder strings.Builder

	for i := lineNum; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" || !strings.HasPrefix(trimmed, "|") {
			break
		}

		// Skip separator row
		if s.isTableSeparator(trimmed) {
			consumed++
			continue
		}

		// Parse table row
		cells := s.parseTableRow(trimmed)
		row := &Block{
			ID:       uuid.New().String(),
			Type:     BlockTypeTableRow,
			Children: make([]*Block, 0),
		}

		for _, cell := range cells {
			row.Children = append(row.Children, &Block{
				ID:      uuid.New().String(),
				Type:    BlockTypeTableCell,
				Content: cell,
				Hash:    s.hashContent(cell),
			})
		}

		block.Children = append(block.Children, row)
		contentBuilder.WriteString(trimmed)
		contentBuilder.WriteString("\n")
		consumed++
	}

	block.Content = strings.TrimSpace(contentBuilder.String())
	block.Hash = s.hashContent(block.Content)

	return block, consumed
}

// parseMathBlock parses $$ math blocks
func (s *BlockMapperService) parseMathBlock(lines []string, lineNum int) (*Block, int) {
	var content strings.Builder
	consumed := 1 // First $$

	for i := lineNum + 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "$$" {
			consumed++
			break
		}
		content.WriteString(lines[i])
		content.WriteString("\n")
		consumed++
	}

	return &Block{
		ID:      uuid.New().String(),
		Type:    BlockTypeMath,
		Content: strings.TrimSpace(content.String()),
		Hash:    s.hashContent(content.String()),
	}, consumed
}

// parseCallout parses Obsidian-style callouts
func (s *BlockMapperService) parseCallout(lines []string, lineNum int) (*Block, int) {
	firstLine := lines[lineNum]

	// Extract callout type
	typeMatch := regexp.MustCompile(`>\s*\[!(\w+)\]`).FindStringSubmatch(firstLine)
	calloutType := "note"
	if len(typeMatch) > 1 {
		calloutType = typeMatch[1]
	}

	var content strings.Builder
	consumed := 0

	for i := lineNum; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, ">") && trimmed != "" {
			break
		}
		if trimmed == "" {
			consumed++
			break
		}

		lineContent := strings.TrimPrefix(trimmed, ">")
		lineContent = strings.TrimPrefix(lineContent, " ")
		content.WriteString(lineContent)
		content.WriteString("\n")
		consumed++
	}

	return &Block{
		ID:      uuid.New().String(),
		Type:    BlockTypeCallout,
		Content: strings.TrimSpace(content.String()),
		Metadata: map[string]interface{}{
			"callout_type": calloutType,
		},
		Hash: s.hashContent(content.String()),
	}, consumed
}

// parseHTMLBlock parses raw HTML blocks
func (s *BlockMapperService) parseHTMLBlock(lines []string, lineNum int) (*Block, int) {
	var content strings.Builder
	consumed := 0

	// Find the tag name
	tagMatch := regexp.MustCompile(`<(\w+)`).FindStringSubmatch(lines[lineNum])
	if len(tagMatch) < 2 {
		return s.parseParagraph(lines, lineNum)
	}
	tagName := tagMatch[1]
	closingTag := fmt.Sprintf("</%s>", tagName)

	for i := lineNum; i < len(lines); i++ {
		content.WriteString(lines[i])
		content.WriteString("\n")
		consumed++

		if strings.Contains(lines[i], closingTag) {
			break
		}
	}

	return &Block{
		ID:      uuid.New().String(),
		Type:    BlockTypeHTML,
		Content: strings.TrimSpace(content.String()),
		Hash:    s.hashContent(content.String()),
	}, consumed
}

// parseParagraph parses regular paragraph text
func (s *BlockMapperService) parseParagraph(lines []string, lineNum int) (*Block, int) {
	var content strings.Builder
	consumed := 0

	for i := lineNum; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			consumed++
			break
		}

		// Check if next block type starts
		if strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "```") ||
			strings.HasPrefix(trimmed, ">") ||
			strings.HasPrefix(trimmed, "|") ||
			s.isListItem(trimmed) ||
			s.isHorizontalRule(trimmed) {
			break
		}

		content.WriteString(lines[i])
		content.WriteString("\n")
		consumed++
	}

	return &Block{
		ID:      uuid.New().String(),
		Type:    BlockTypeParagraph,
		Content: strings.TrimSpace(content.String()),
		Hash:    s.hashContent(content.String()),
	}, consumed
}
