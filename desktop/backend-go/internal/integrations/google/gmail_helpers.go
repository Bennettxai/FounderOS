package google

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"google.golang.org/api/gmail/v1"
)

func parseEmailAddress(addr string) (name, email string) {
	addr = strings.TrimSpace(addr)
	if strings.Contains(addr, "<") {
		parts := strings.Split(addr, "<")
		name = strings.TrimSpace(parts[0])
		name = strings.Trim(name, "\"")
		email = strings.TrimSuffix(parts[1], ">")
	} else {
		email = addr
	}
	return name, email
}

func parseEmailAddresses(addrs string) []EmailAddress {
	var result []EmailAddress
	if addrs == "" {
		return result
	}

	parts := strings.Split(addrs, ",")
	for _, part := range parts {
		name, email := parseEmailAddress(strings.TrimSpace(part))
		if email != "" {
			result = append(result, EmailAddress{Name: name, Email: email})
		}
	}
	return result
}

func extractBody(payload *gmail.MessagePart) (text, html string) {
	if payload.MimeType == "text/plain" && payload.Body != nil && payload.Body.Data != "" {
		decoded, _ := base64.URLEncoding.DecodeString(payload.Body.Data)
		text = string(decoded)
	} else if payload.MimeType == "text/html" && payload.Body != nil && payload.Body.Data != "" {
		decoded, _ := base64.URLEncoding.DecodeString(payload.Body.Data)
		html = string(decoded)
	}

	for _, part := range payload.Parts {
		partText, partHTML := extractBody(part)
		if partText != "" && text == "" {
			text = partText
		}
		if partHTML != "" && html == "" {
			html = partHTML
		}
	}

	return text, html
}

func extractAttachments(payload *gmail.MessagePart) []Attachment {
	var attachments []Attachment

	if payload.Filename != "" && payload.Body != nil && payload.Body.AttachmentId != "" {
		attachments = append(attachments, Attachment{
			ID:       payload.Body.AttachmentId,
			Filename: payload.Filename,
			MimeType: payload.MimeType,
			Size:     payload.Body.Size,
		})
	}

	for _, part := range payload.Parts {
		attachments = append(attachments, extractAttachments(part)...)
	}

	return attachments
}

func containsLabel(labels []string, label string) bool {
	for _, l := range labels {
		if l == label {
			return true
		}
	}
	return false
}

func parseEmailDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 -0700 (MST)",
		"2 Jan 2006 15:04:05 -0700",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}
