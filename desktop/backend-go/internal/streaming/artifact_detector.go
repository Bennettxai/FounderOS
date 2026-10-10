package streaming

import (
	"encoding/json"
	"regexp"
	"strings"
)

// ArtifactDetector detects and extracts artifacts and thinking blocks from streaming output
type ArtifactDetector struct {
	buffer         strings.Builder
	inArtifact     bool
	artifactBuffer strings.Builder
	startPattern   *regexp.Regexp
	
	inThinking     bool
	thinkingStart  *regexp.Regexp
	thinkingEnd    *regexp.Regexp
}

// NewArtifactDetector creates a new artifact detector
func NewArtifactDetector() *ArtifactDetector {
	return &ArtifactDetector{
		startPattern:  regexp.MustCompile("```artifact\\s*\\n?"),
		// Matches <thinking>, <thinkingg>, <thinkingng>, <think> - very permissive
		thinkingStart: regexp.MustCompile(`<think[a-z]*\s*>`), 
		thinkingEnd:   regexp.MustCompile(`</think[a-z]*\s*>`),
	}
}

// ProcessChunk processes a chunk of streaming output and returns events
func (d *ArtifactDetector) ProcessChunk(chunk string) []StreamEvent {
	// If we are already in a state, process accordingly
	if d.inArtifact {
		d.artifactBuffer.WriteString(chunk)
		return d.checkArtifactComplete()
	}
	if d.inThinking {
		return d.processThinkingContent(chunk)
	}

	// Buffer processing for normal content
	d.buffer.WriteString(chunk)
	content := d.buffer.String()

	return d.processNormalContent(content)
}

// processNormalContent handles content when not inside an artifact or thinking block
func (d *ArtifactDetector) processNormalContent(content string) []StreamEvent {
	var events []StreamEvent

	// Find indices of potential starts
	artifactLoc := d.startPattern.FindStringIndex(content)
	thinkingLoc := d.thinkingStart.FindStringIndex(content)

	// Determine which comes first, if any
	var firstMatchLoc []int
	isArtifact := false

	if artifactLoc != nil && thinkingLoc != nil {
		if artifactLoc[0] < thinkingLoc[0] {
			firstMatchLoc = artifactLoc
			isArtifact = true
		} else {
			firstMatchLoc = thinkingLoc
			isArtifact = false
		}
	} else if artifactLoc != nil {
		firstMatchLoc = artifactLoc
		isArtifact = true
	} else if thinkingLoc != nil {
		firstMatchLoc = thinkingLoc
		isArtifact = false
	}

	// No tags found
	if firstMatchLoc == nil {
		// Keep a buffer of 20 chars to handle split tags
		if len(content) > 20 {
			safeContent := content[:len(content)-20]
			events = append(events, StreamEvent{Type: EventTypeToken, Content: safeContent})
			d.buffer.Reset()
			d.buffer.WriteString(content[len(content)-20:])
		}
		return events
	}

	// Flush content before the tag
	if firstMatchLoc[0] > 0 {
		events = append(events, StreamEvent{Type: EventTypeToken, Content: content[:firstMatchLoc[0]]})
	}

	// Handle transition
	afterMarker := content[firstMatchLoc[1]:]

	d.buffer.Reset()

	if isArtifact {
		events = append(events, StreamEvent{Type: EventTypeArtifactStart})
		d.inArtifact = true
		d.artifactBuffer.Reset()
		d.artifactBuffer.WriteString(afterMarker)
		events = append(events, d.checkArtifactComplete()...)
	} else {
		// Thinking start
		events = append(events, StreamEvent{
			Type: EventTypeThinkingStart,
			Data: map[string]interface{}{"step": 1}, // Simple step tracking
		})
		d.inThinking = true
		// Don't buffer the tag itself ("<thinking>"), just the content after
		events = append(events, d.processThinkingContent(afterMarker)...)
	}

	return events
}

// processThinkingContent handles content when inside a thinking block
func (d *ArtifactDetector) processThinkingContent(chunk string) []StreamEvent {
	var events []StreamEvent
	
	// We check for the end tag in the current chunk + potentially what matches end tag
	// But simpler approach: emit chunk as ThinkingChunk, but buffer check for end tag
	
	// Issue: ProcessChunk passes raw chunks. We can't easily detect end tag split across chunks 
	// without buffering. But for Thinking, we want to stream chunks ASAP.
	
	// Strategy: Buffer a small tail to check for end tag. Emit everything before the tail.
	// But `thinkingEnd` regex needs to match against the stream.
	
	d.buffer.WriteString(chunk)
	content := d.buffer.String()
	
	endLoc := d.thinkingEnd.FindStringIndex(content)
	
	if endLoc == nil {
		// No end tag found.
		// Emit safe content (keep tail in buffer in case end tag is starting)
		if len(content) > 20 {
			safeContent := content[:len(content)-20]
			events = append(events, StreamEvent{
				Type: EventTypeThinkingChunk,
				Data: map[string]interface{}{"content": safeContent},
			})
			d.buffer.Reset()
			d.buffer.WriteString(content[len(content)-20:])
		}
		return events
	}
	
	// End tag found
	// 1. Emit content before end tag
	if endLoc[0] > 0 {
		thinkingContent := content[:endLoc[0]]
		if len(thinkingContent) > 0 {
			events = append(events, StreamEvent{
				Type: EventTypeThinkingChunk,
				Data: map[string]interface{}{"content": thinkingContent},
			})
		}
	}
	
	// 2. Emit ThinkingEnd
	events = append(events, StreamEvent{
		Type: EventTypeThinkingEnd,
		Data: map[string]interface{}{"step": 1},
	})
	
	d.inThinking = false
	d.buffer.Reset()
	
	// 3. Process remaining content (after </thinking>) as normal content
	if endLoc[1] < len(content) {
		remaining := content[endLoc[1]:]
		d.buffer.WriteString(remaining) 
		// We recurse into processNormalContent effectively by returning control to ProcessChunk 
		// but since ProcessChunk checks state first, we need to handle it here or let next chunk handle it.
		// However, remaining content might contain artifact start.
		// Let's call processNormalContent with the remainder
		d.buffer.Reset() // Reset because processNormalContent expects string input and uses buffer itself (well, it resets buffer)
						 // Wait, processNormalContent uses d.buffer internally or just input?
						 // My implementation of processNormalContent uses d.buffer for output buffering.
						 // It takes `content` string.
		events = append(events, d.processNormalContent(remaining)...)
	}
	
	return events
}

// processArtifactContent handles content when inside an artifact
func (d *ArtifactDetector) processArtifactContent(chunk string) []StreamEvent {
	d.artifactBuffer.WriteString(chunk)
	return d.checkArtifactComplete()
}

// checkArtifactComplete checks if the artifact has ended and parses it
func (d *ArtifactDetector) checkArtifactComplete() []StreamEvent {
	var events []StreamEvent
	content := d.artifactBuffer.String()

	closingIdx := strings.LastIndex(content, "```")
	if closingIdx == -1 {
		return events
	}

	afterClosing := strings.TrimSpace(content[closingIdx+3:])
	if len(afterClosing) > 20 {
		return events
	}

	artifactJSON := strings.TrimSpace(content[:closingIdx])

	var artifact Artifact
	if err := json.Unmarshal([]byte(artifactJSON), &artifact); err != nil {
		events = append(events, StreamEvent{
			Type:    EventTypeArtifactError,
			Content: "Failed to parse artifact: " + err.Error(),
		})
		d.inArtifact = false
		d.buffer.Reset()
		d.buffer.WriteString(afterClosing)
		return events
	}

	events = append(events, StreamEvent{
		Type: EventTypeArtifactComplete,
		Data: artifact,
	})

	d.inArtifact = false
	d.artifactBuffer.Reset()
	d.buffer.Reset()
	d.buffer.WriteString(afterClosing)

	return events
}

// Flush returns any remaining buffered content
func (d *ArtifactDetector) Flush() []StreamEvent {
	var events []StreamEvent

	if d.inArtifact {
		events = append(events, StreamEvent{
			Type:    EventTypeArtifactError,
			Content: "Artifact was not properly closed",
		})
		events = append(events, StreamEvent{
			Type:    EventTypeToken,
			Content: d.artifactBuffer.String(),
		})
	} else if d.inThinking {
		// Flush remaining thinking buffer as content
		events = append(events, StreamEvent{
			Type: EventTypeThinkingChunk,
			Data: map[string]interface{}{"content": d.buffer.String()},
		})
		events = append(events, StreamEvent{
			Type: EventTypeThinkingEnd,
			Data: map[string]interface{}{"step": 1}, // Close it gracefully
		})
	} else if d.buffer.Len() > 0 {
		events = append(events, StreamEvent{
			Type:    EventTypeToken,
			Content: d.buffer.String(),
		})
	}

	d.buffer.Reset()
	d.artifactBuffer.Reset()
	d.inArtifact = false
	d.inThinking = false

	return events
}

// Reset resets the detector state
func (d *ArtifactDetector) Reset() {
	d.buffer.Reset()
	d.artifactBuffer.Reset()
	d.inArtifact = false
	d.inThinking = false
}

// IsInArtifact returns whether we're currently inside an artifact
func (d *ArtifactDetector) IsInArtifact() bool {
	return d.inArtifact
}
