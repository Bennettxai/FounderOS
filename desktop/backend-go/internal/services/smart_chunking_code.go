package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// CodeBlock represents a code block
type CodeBlock struct {
	Type    string // function, class, method, etc.
	Name    string
	Content string
}

// ChunkCode chunks code content respecting language syntax
func (s *SmartChunkingService) ChunkCode(ctx context.Context, content string, language string, parentDocID string, options ChunkOptions) ([]Chunk, error) {
	var chunks []Chunk

	// If language not specified, try to detect
	if language == "" {
		language = s.detectLanguage(content)
	}

	// Split by code blocks (functions, classes, etc.)
	codeBlocks := s.splitCodeBlocks(content, language)

	position := 0
	overlapSize := int(float64(options.ChunkSize) * options.OverlapRatio)

	for blockIdx, block := range codeBlocks {
		blockTokens := s.estimateTokens(block.Content)

		if blockTokens <= options.ChunkSize {
			// Block fits in one chunk
			chunk := s.createChunk(block.Content, position, parentDocID, map[string]interface{}{
				"type":       "code_block",
				"language":   language,
				"block_type": block.Type,
				"name":       block.Name,
				"block_idx":  blockIdx,
			})
			chunks = append(chunks, chunk)
			position++
		} else {
			// Block too large, split by lines while preserving context
			subChunks := s.chunkCodeLines(block.Content, position, parentDocID, options, map[string]interface{}{
				"type":       "code_block",
				"language":   language,
				"block_type": block.Type,
				"name":       block.Name,
				"block_idx":  blockIdx,
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

// splitCodeBlocks splits code into logical blocks
func (s *SmartChunkingService) splitCodeBlocks(content string, language string) []CodeBlock {
	// Simplified implementation - production would use proper parsers
	var blocks []CodeBlock

	switch strings.ToLower(language) {
	case "go", "golang":
		blocks = s.splitGoCode(content)
	case "python", "py":
		blocks = s.splitPythonCode(content)
	case "javascript", "js", "typescript", "ts":
		blocks = s.splitJSCode(content)
	default:
		// Fallback: split by blank lines
		blocks = s.splitByBlankLines(content)
	}

	return blocks
}

// splitGoCode splits Go code by functions and types
func (s *SmartChunkingService) splitGoCode(content string) []CodeBlock {
	var blocks []CodeBlock
	lines := strings.Split(content, "\n")

	funcRegex := regexp.MustCompile(`^func\s+(\w+)`)
	typeRegex := regexp.MustCompile(`^type\s+(\w+)`)

	var currentBlock *CodeBlock
	var currentLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for function declaration
		if matches := funcRegex.FindStringSubmatch(trimmed); matches != nil {
			if currentBlock != nil {
				currentBlock.Content = strings.Join(currentLines, "\n")
				blocks = append(blocks, *currentBlock)
			}
			currentBlock = &CodeBlock{
				Type: "function",
				Name: matches[1],
			}
			currentLines = []string{line}
		} else if matches := typeRegex.FindStringSubmatch(trimmed); matches != nil {
			if currentBlock != nil {
				currentBlock.Content = strings.Join(currentLines, "\n")
				blocks = append(blocks, *currentBlock)
			}
			currentBlock = &CodeBlock{
				Type: "type",
				Name: matches[1],
			}
			currentLines = []string{line}
		} else if currentBlock != nil {
			currentLines = append(currentLines, line)
		} else {
			// Code before first function/type
			if len(blocks) == 0 {
				currentBlock = &CodeBlock{
					Type: "header",
					Name: "package",
				}
				currentLines = []string{line}
			}
		}
	}

	// Add final block
	if currentBlock != nil {
		currentBlock.Content = strings.Join(currentLines, "\n")
		blocks = append(blocks, *currentBlock)
	}

	return blocks
}

// splitPythonCode splits Python code by functions and classes
func (s *SmartChunkingService) splitPythonCode(content string) []CodeBlock {
	var blocks []CodeBlock
	lines := strings.Split(content, "\n")

	funcRegex := regexp.MustCompile(`^def\s+(\w+)`)
	classRegex := regexp.MustCompile(`^class\s+(\w+)`)

	var currentBlock *CodeBlock
	var currentLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if matches := funcRegex.FindStringSubmatch(trimmed); matches != nil {
			if currentBlock != nil {
				currentBlock.Content = strings.Join(currentLines, "\n")
				blocks = append(blocks, *currentBlock)
			}
			currentBlock = &CodeBlock{
				Type: "function",
				Name: matches[1],
			}
			currentLines = []string{line}
		} else if matches := classRegex.FindStringSubmatch(trimmed); matches != nil {
			if currentBlock != nil {
				currentBlock.Content = strings.Join(currentLines, "\n")
				blocks = append(blocks, *currentBlock)
			}
			currentBlock = &CodeBlock{
				Type: "class",
				Name: matches[1],
			}
			currentLines = []string{line}
		} else if currentBlock != nil {
			currentLines = append(currentLines, line)
		} else {
			if len(blocks) == 0 {
				currentBlock = &CodeBlock{
					Type: "header",
					Name: "imports",
				}
				currentLines = []string{line}
			}
		}
	}

	if currentBlock != nil {
		currentBlock.Content = strings.Join(currentLines, "\n")
		blocks = append(blocks, *currentBlock)
	}

	return blocks
}

// splitJSCode splits JavaScript/TypeScript code
func (s *SmartChunkingService) splitJSCode(content string) []CodeBlock {
	var blocks []CodeBlock
	lines := strings.Split(content, "\n")

	funcRegex := regexp.MustCompile(`^(function|const|let|var)\s+(\w+)\s*[=\(]`)
	classRegex := regexp.MustCompile(`^class\s+(\w+)`)

	var currentBlock *CodeBlock
	var currentLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if matches := classRegex.FindStringSubmatch(trimmed); matches != nil {
			if currentBlock != nil {
				currentBlock.Content = strings.Join(currentLines, "\n")
				blocks = append(blocks, *currentBlock)
			}
			currentBlock = &CodeBlock{
				Type: "class",
				Name: matches[1],
			}
			currentLines = []string{line}
		} else if matches := funcRegex.FindStringSubmatch(trimmed); matches != nil {
			if currentBlock != nil {
				currentBlock.Content = strings.Join(currentLines, "\n")
				blocks = append(blocks, *currentBlock)
			}
			currentBlock = &CodeBlock{
				Type: "function",
				Name: matches[2],
			}
			currentLines = []string{line}
		} else if currentBlock != nil {
			currentLines = append(currentLines, line)
		} else {
			if len(blocks) == 0 {
				currentBlock = &CodeBlock{
					Type: "header",
					Name: "imports",
				}
				currentLines = []string{line}
			}
		}
	}

	if currentBlock != nil {
		currentBlock.Content = strings.Join(currentLines, "\n")
		blocks = append(blocks, *currentBlock)
	}

	return blocks
}

// splitByBlankLines splits content by blank lines
func (s *SmartChunkingService) splitByBlankLines(content string) []CodeBlock {
	var blocks []CodeBlock
	lines := strings.Split(content, "\n")

	var currentLines []string
	blockIdx := 0

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if len(currentLines) > 0 {
				blocks = append(blocks, CodeBlock{
					Type:    "block",
					Name:    fmt.Sprintf("block_%d", blockIdx),
					Content: strings.Join(currentLines, "\n"),
				})
				blockIdx++
				currentLines = []string{}
			}
		} else {
			currentLines = append(currentLines, line)
		}
	}

	if len(currentLines) > 0 {
		blocks = append(blocks, CodeBlock{
			Type:    "block",
			Name:    fmt.Sprintf("block_%d", blockIdx),
			Content: strings.Join(currentLines, "\n"),
		})
	}

	return blocks
}

// detectLanguage attempts to detect programming language
func (s *SmartChunkingService) detectLanguage(content string) string {
	// Simple heuristic-based detection
	content = strings.TrimSpace(content)

	// Go
	if strings.Contains(content, "package ") && strings.Contains(content, "func ") {
		return "go"
	}

	// Python
	if strings.Contains(content, "def ") && strings.Contains(content, ":") {
		return "python"
	}

	// JavaScript/TypeScript
	if strings.Contains(content, "function ") || strings.Contains(content, "const ") || strings.Contains(content, "=>") {
		return "javascript"
	}

	// Java
	if strings.Contains(content, "public class ") || strings.Contains(content, "private ") {
		return "java"
	}

	// C/C++
	if strings.Contains(content, "#include") || strings.Contains(content, "int main(") {
		return "c"
	}

	// Fallback
	return "unknown"
}
