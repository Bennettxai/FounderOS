package handlers

import (
	"strings"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// stringToNodeType converts a string to sqlc.NodeType
func stringToNodeType(t string) sqlc.Nodetype {
	typeMap := map[string]sqlc.Nodetype{
		"business":    sqlc.NodetypeBUSINESS,
		"project":     sqlc.NodetypePROJECT,
		"learning":    sqlc.NodetypeLEARNING,
		"operational": sqlc.NodetypeOPERATIONAL,
	}
	if enum, ok := typeMap[strings.ToLower(t)]; ok {
		return enum
	}
	return sqlc.NodetypeBUSINESS
}

// stringToNodeHealth converts a string to sqlc.Nodehealth
func stringToNodeHealth(h string) sqlc.Nodehealth {
	typeMap := map[string]sqlc.Nodehealth{
		"healthy":         sqlc.NodehealthHEALTHY,
		"needs_attention": sqlc.NodehealthNEEDSATTENTION,
		"critical":        sqlc.NodehealthCRITICAL,
		"not_started":     sqlc.NodehealthNOTSTARTED,
	}
	if enum, ok := typeMap[strings.ToLower(h)]; ok {
		return enum
	}
	return sqlc.NodehealthNOTSTARTED
}
