package handlers

import (
	"regexp"
	"strings"
)

// AgentMention represents a parsed @agent mention
type AgentMention struct {
	AgentName string
	StartPos  int
	EndPos    int
}

// parseAgentMentions extracts @agent-name mentions from a message
func parseAgentMentions(message string) []AgentMention {
	var mentions []AgentMention
	mentionPattern := regexp.MustCompile(`@([a-z0-9][a-z0-9-]*[a-z0-9]|[a-z0-9])`)

	matches := mentionPattern.FindAllStringSubmatchIndex(message, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			mentions = append(mentions, AgentMention{
				AgentName: message[match[2]:match[3]],
				StartPos:  match[0],
				EndPos:    match[1],
			})
		}
	}
	return mentions
}

// stripMentions removes @mentions from message for cleaner processing
func stripMentions(message string, mentions []AgentMention) string {
	if len(mentions) == 0 {
		return message
	}
	result := message
	// Remove in reverse order to preserve indices
	for i := len(mentions) - 1; i >= 0; i-- {
		m := mentions[i]
		result = result[:m.StartPos] + result[m.EndPos:]
	}
	return strings.TrimSpace(result)
}
