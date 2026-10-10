package services

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

func (s *BlockMapperService) isHorizontalRule(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) < 3 {
		return false
	}
	return regexp.MustCompile(`^[-*_]{3,}$`).MatchString(line)
}

func (s *BlockMapperService) isListItem(line string) bool {
	return regexp.MustCompile(`^(\s*[-*+]|\s*\d+\.)\s`).MatchString(line)
}

func (s *BlockMapperService) isOrderedListItem(line string) bool {
	return regexp.MustCompile(`^\d+\.\s`).MatchString(line)
}

func (s *BlockMapperService) extractListItemContent(line string) string {
	// Remove list marker
	return regexp.MustCompile(`^(\s*[-*+]|\s*\d+\.)\s*`).ReplaceAllString(line, "")
}

func (s *BlockMapperService) isTaskItem(content string) (bool, bool) {
	if strings.HasPrefix(content, "[ ] ") {
		return true, false
	}
	if strings.HasPrefix(content, "[x] ") || strings.HasPrefix(content, "[X] ") {
		return true, true
	}
	return false, false
}

func (s *BlockMapperService) isTableSeparator(line string) bool {
	return regexp.MustCompile(`^\|[\s:-]+\|$`).MatchString(line)
}

func (s *BlockMapperService) parseTableRow(line string) []string {
	// Remove leading and trailing |
	line = strings.Trim(line, "|")
	cells := strings.Split(line, "|")

	for i, cell := range cells {
		cells[i] = strings.TrimSpace(cell)
	}

	return cells
}

func (s *BlockMapperService) hashContent(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:8])
}
