package appgen

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	sdk "github.com/severity1/claude-agent-sdk-go"
)

type workerImpl struct {
	agentType        AgentType
	progressCallback ProgressCallback
}

func NewWorker(agentType AgentType, progressCallback ProgressCallback) Worker {
	return &workerImpl{
		agentType:        agentType,
		progressCallback: progressCallback,
	}
}

func (w *workerImpl) Execute(ctx context.Context, task Task) (*AgentResult, error) {
	startTime := time.Now()

	slog.InfoContext(ctx, "worker executing",
		"agent", w.agentType,
		"task", task.ID,
	)

	w.emitProgress(task.ID, "starting", fmt.Sprintf("Starting %s agent", w.agentType), 0)

	prompt := w.createPrompt(task)

	w.emitProgress(task.ID, "in_progress", fmt.Sprintf("Generating %s code", w.agentType), 30)

	// 5-minute timeout per individual agent API call
	// Increased from 3min to prevent database/test agents from timing out
	// (PR #34: Outer queue timeout is 10min, allowing safe retry window)
	apiCtx, apiCancel := context.WithTimeout(ctx, 5*time.Minute)
	defer apiCancel()

	var responseContent strings.Builder

	err := sdk.WithClient(apiCtx, func(client sdk.Client) error {
		if err := client.Query(apiCtx, prompt); err != nil {
			return err
		}

		msgChan := client.ReceiveMessages(apiCtx)
		for message := range msgChan {
			if message == nil {
				break
			}

			switch msg := message.(type) {
			case *sdk.AssistantMessage:
				for _, block := range msg.Content {
					if textBlock, ok := block.(*sdk.TextBlock); ok {
						responseContent.WriteString(textBlock.Text)
					}
				}
			case *sdk.ResultMessage:
				if msg.IsError {
					return fmt.Errorf("query failed")
				}
				return nil
			}
		}
		return nil
	},
		sdk.WithModel(string(sdk.AgentModelSonnet)),
		sdk.WithMaxTurns(5),
	)

	if err != nil {
		w.emitProgress(task.ID, "failed", fmt.Sprintf("Agent failed: %v", err), 0)
		return &AgentResult{
			TaskID:    task.ID,
			AgentType: w.agentType,
			Success:   false,
			Error:     err.Error(),
			Duration:  time.Since(startTime),
		}, err
	}

	// Parse response and extract code blocks
	output := responseContent.String()

	// DEBUG: Log raw output length and first 500 chars to understand what we're parsing
	slog.InfoContext(ctx, "DEBUG: raw model output",
		"agent", w.agentType,
		"output_length", len(output),
		"output_preview", truncateForLog(output, 500),
	)

	codeBlocks := parseCodeBlocksFromResponse(output)

	// DEBUG: Log parsing results
	slog.InfoContext(ctx, "DEBUG: code blocks parsed",
		"agent", w.agentType,
		"blocks_found", len(codeBlocks),
		"file_paths", getMapKeys(codeBlocks),
	)

	// Convert code blocks map to slice of file paths
	var filesCreated []string
	for filePath := range codeBlocks {
		filesCreated = append(filesCreated, filePath)
	}

	result := &AgentResult{
		TaskID:       task.ID,
		AgentType:    w.agentType,
		Success:      true,
		Output:       output,
		FilesCreated: filesCreated,
		CodeBlocks:   codeBlocks, // Store parsed code blocks
		Duration:     time.Since(startTime),
	}

	slog.InfoContext(ctx, "worker completed",
		"agent", w.agentType,
		"files_extracted", len(filesCreated),
	)

	w.emitProgress(task.ID, "completed", fmt.Sprintf("%s agent completed (%d files)", w.agentType, len(filesCreated)), 100)

	slog.InfoContext(ctx, "worker completed",
		"agent", w.agentType,
		"duration", result.Duration,
	)

	return result, nil
}

func (w *workerImpl) Type() AgentType {
	return w.agentType
}

func (w *workerImpl) createPrompt(task Task) string {
	base := fmt.Sprintf("Task: %s\nWorkspace: %s\n\n", task.Description, task.Workspace)

	fileFormatInstructions := `

CRITICAL: For EVERY file you generate, you MUST use this exact code block format with the filepath after a colon:

` + "```" + `language:path/to/file.ext
// file content here
` + "```" + `

Example:
` + "```" + `typescript:src/components/App.svelte
<script lang="ts">
  let count = $state(0);
</script>
` + "```" + `

Always include the FULL relative file path after the language, separated by a colon. Every code block MUST have a filepath. Do NOT use code blocks without filepaths.
`

	switch w.agentType {
	case AgentFrontend:
		return base + `Create Svelte 5 frontend components for this application:
- TypeScript strict mode
- Tailwind CSS for styling
- Svelte 5 runes ($state, $derived, $effect)
- Production-ready, accessible components
- Include: main page component, reusable UI components, stores, types
- File paths should start with src/` + fileFormatInstructions

	case AgentBackend:
		return base + `Create Go backend code for this application:
- Gin HTTP framework
- Handler → Service → Repository pattern
- slog for structured logging
- Proper error handling with wrapped errors
- Include: main.go, handlers, services, models, routes
- File paths should start with cmd/ or internal/` + fileFormatInstructions

	case AgentDatabase:
		return base + `Create PostgreSQL database schema and migrations:
- CREATE TABLE statements with proper types
- Indexes for common query patterns
- Foreign key constraints
- Include: migration files only; do not invent seed/sample rows unless the user explicitly asks
- File paths should start with migrations/ or database/` + fileFormatInstructions

	case AgentTest:
		return base + `Create comprehensive tests for this application:
- Go unit tests for backend (testing package)
- Frontend component tests
- Target 80%+ coverage
- Test edge cases and error paths
- Include: test files matching source structure
- File paths should end with _test.go or .test.ts` + fileFormatInstructions

	default:
		return base + "Create the code for this task." + fileFormatInstructions
	}
}

func (w *workerImpl) emitProgress(taskID, status, message string, progress int) {
	if w.progressCallback != nil {
		w.progressCallback(ProgressEvent{
			TaskID:    taskID,
			AgentType: w.agentType,
			Status:    status,
			Message:   message,
			Progress:  progress,
			Timestamp: time.Now(),
		})
	}
}

// parseCodeBlocksFromResponse extracts code blocks from markdown-formatted response
func parseCodeBlocksFromResponse(text string) map[string]string {
	files := make(map[string]string)

	// Split by code fence markers
	lines := strings.Split(text, "\n")
	var currentFile string
	var currentContent []string
	inCodeBlock := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Check for code block start
		if strings.HasPrefix(line, "```") {
			if !inCodeBlock {
				// Start of code block
				inCodeBlock = true
				currentContent = []string{}
				currentFile = ""

				// Try to extract filename from code fence
				// Format: ```typescript:path/to/file.ts
				// or: ```typescript path/to/file.ts
				// or: ```go:internal/handler/user.go
				parts := strings.TrimPrefix(line, "```")
				parts = strings.TrimSpace(parts)

				// Look for colon separator (preferred format)
				if strings.Contains(parts, ":") {
					filenameParts := strings.SplitN(parts, ":", 2)
					if len(filenameParts) == 2 {
						currentFile = strings.TrimSpace(filenameParts[1])
					}
				} else if strings.Contains(parts, " ") {
					// Look for space separator
					filenameParts := strings.SplitN(parts, " ", 2)
					if len(filenameParts) == 2 {
						potentialPath := strings.TrimSpace(filenameParts[1])
						// Only use if it looks like a path (has / or .)
						if strings.Contains(potentialPath, "/") || strings.Contains(potentialPath, ".") {
							currentFile = potentialPath
						}
					}
				}

				// If no filename in fence, check next few lines for File: comment
				if currentFile == "" && i+1 < len(lines) {
					nextLine := strings.TrimSpace(lines[i+1])
					if strings.HasPrefix(nextLine, "// File:") {
						currentFile = strings.TrimSpace(strings.TrimPrefix(nextLine, "// File:"))
						i++ // Skip this comment line
					} else if strings.HasPrefix(nextLine, "# File:") {
						currentFile = strings.TrimSpace(strings.TrimPrefix(nextLine, "# File:"))
						i++ // Skip this comment line
					}
				}
			} else {
				// End of code block
				inCodeBlock = false
				if currentFile != "" && len(currentContent) > 0 {
					files[currentFile] = strings.Join(currentContent, "\n")
				}
				currentFile = ""
				currentContent = []string{}
			}
			continue
		}

		// Accumulate code block content
		if inCodeBlock {
			currentContent = append(currentContent, line)
		}
	}

	// Handle unclosed code block (edge case)
	if inCodeBlock && currentFile != "" && len(currentContent) > 0 {
		files[currentFile] = strings.Join(currentContent, "\n")
	}

	return files
}

// truncateForLog truncates a string for logging purposes
func truncateForLog(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "... [truncated]"
}

// getMapKeys returns the keys of a map as a slice
func getMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
