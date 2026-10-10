package handlers

import (
	"encoding/json"

	"github.com/google/uuid"
)

// dnsNamespace is the UUID DNS namespace used for deterministic file ID generation.
var dnsNamespace = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

// workflowFileTypes is the ordered list of metadata keys treated as workflow files.
var workflowFileTypes = []string{
	"analysis", "architecture", "code", "quality",
	"deployment", "monitoring", "strategy", "recommendations",
}

// parseMetadataJSON unmarshals raw JSON into dst. On failure dst is left unchanged.
func parseMetadataJSON(raw []byte, dst *map[string]interface{}) error {
	return json.Unmarshal(raw, dst)
}

// resolveWorkflowSearch returns the search arguments for queries that accept
// either a UUID or a workflow ID prefix.
func resolveWorkflowSearch(rawID string) (searchID interface{}, searchPrefix string) {
	searchPrefix = rawID + "%"
	if parsed, err := uuid.Parse(rawID); err == nil {
		return parsed, searchPrefix
	}
	return uuid.Nil, searchPrefix
}
