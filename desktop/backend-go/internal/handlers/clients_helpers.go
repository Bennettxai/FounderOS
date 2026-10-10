package handlers

import (
	"strings"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// stringToClientType converts a string to sqlc.Clienttype
func stringToClientType(t string) sqlc.Clienttype {
	typeMap := map[string]sqlc.Clienttype{
		"company":    sqlc.ClienttypeCompany,
		"individual": sqlc.ClienttypeIndividual,
	}
	if enum, ok := typeMap[strings.ToLower(t)]; ok {
		return enum
	}
	return sqlc.ClienttypeCompany
}

// stringToClientStatus converts a string to sqlc.Clientstatus
func stringToClientStatus(s string) sqlc.Clientstatus {
	typeMap := map[string]sqlc.Clientstatus{
		"lead":     sqlc.ClientstatusLead,
		"prospect": sqlc.ClientstatusProspect,
		"active":   sqlc.ClientstatusActive,
		"inactive": sqlc.ClientstatusInactive,
		"churned":  sqlc.ClientstatusChurned,
	}
	if enum, ok := typeMap[strings.ToLower(s)]; ok {
		return enum
	}
	return sqlc.ClientstatusActive
}

// stringToInteractionType converts a string to sqlc.Interactiontype
func stringToInteractionType(t string) sqlc.Interactiontype {
	typeMap := map[string]sqlc.Interactiontype{
		"call":    sqlc.InteractiontypeCall,
		"email":   sqlc.InteractiontypeEmail,
		"meeting": sqlc.InteractiontypeMeeting,
		"note":    sqlc.InteractiontypeNote,
	}
	if enum, ok := typeMap[strings.ToLower(t)]; ok {
		return enum
	}
	return sqlc.InteractiontypeNote
}
