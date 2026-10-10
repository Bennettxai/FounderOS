package services

import (
	"fmt"
	"strings"
)

// BlocksToMarkdown converts blocks back to markdown
func (s *BlockMapperService) BlocksToMarkdown(blocks []*Block) string {
	var builder strings.Builder

	for _, block := range blocks {
		s.blockToMarkdown(&builder, block)
		builder.WriteString("\n")
	}

	return strings.TrimSpace(builder.String())
}

func (s *BlockMapperService) blockToMarkdown(builder *strings.Builder, block *Block) {
	switch block.Type {
	case BlockTypeParagraph:
		builder.WriteString(block.Content)
		builder.WriteString("\n")

	case BlockTypeHeading:
		builder.WriteString(strings.Repeat("#", block.Level))
		builder.WriteString(" ")
		builder.WriteString(block.Content)
		builder.WriteString("\n")

	case BlockTypeCode:
		builder.WriteString("```")
		builder.WriteString(block.Language)
		builder.WriteString("\n")
		builder.WriteString(block.Content)
		builder.WriteString("\n```\n")

	case BlockTypeBlockquote:
		lines := strings.Split(block.Content, "\n")
		for _, line := range lines {
			builder.WriteString("> ")
			builder.WriteString(line)
			builder.WriteString("\n")
		}

	case BlockTypeList:
		for i, child := range block.Children {
			if block.Metadata["ordered"] == true {
				builder.WriteString(fmt.Sprintf("%d. ", i+1))
			} else {
				builder.WriteString("- ")
			}
			builder.WriteString(child.Content)
			builder.WriteString("\n")
		}

	case BlockTypeTable:
		for i, row := range block.Children {
			builder.WriteString("|")
			for _, cell := range row.Children {
				builder.WriteString(" ")
				builder.WriteString(cell.Content)
				builder.WriteString(" |")
			}
			builder.WriteString("\n")

			// Add separator after header
			if i == 0 {
				builder.WriteString("|")
				for range row.Children {
					builder.WriteString(" --- |")
				}
				builder.WriteString("\n")
			}
		}

	case BlockTypeHR:
		builder.WriteString("---\n")

	case BlockTypeThinking:
		builder.WriteString("<thinking>\n")
		builder.WriteString(block.Content)
		builder.WriteString("\n</thinking>\n")

	case BlockTypeArtifact:
		builder.WriteString("<artifact")
		if id, ok := block.Metadata["identifier"].(string); ok {
			builder.WriteString(fmt.Sprintf(" identifier=\"%s\"", id))
		}
		if t, ok := block.Metadata["type"].(string); ok {
			builder.WriteString(fmt.Sprintf(" type=\"%s\"", t))
		}
		if title, ok := block.Metadata["title"].(string); ok {
			builder.WriteString(fmt.Sprintf(" title=\"%s\"", title))
		}
		builder.WriteString(">\n")
		builder.WriteString(block.Content)
		builder.WriteString("\n</artifact>\n")

	case BlockTypeMath:
		builder.WriteString("$$\n")
		builder.WriteString(block.Content)
		builder.WriteString("\n$$\n")

	case BlockTypeCallout:
		calloutType := "note"
		if t, ok := block.Metadata["callout_type"].(string); ok {
			calloutType = t
		}
		lines := strings.Split(block.Content, "\n")
		for i, line := range lines {
			if i == 0 {
				builder.WriteString(fmt.Sprintf("> [!%s] %s\n", calloutType, line))
			} else {
				builder.WriteString("> ")
				builder.WriteString(line)
				builder.WriteString("\n")
			}
		}

	default:
		builder.WriteString(block.Content)
		builder.WriteString("\n")
	}
}
