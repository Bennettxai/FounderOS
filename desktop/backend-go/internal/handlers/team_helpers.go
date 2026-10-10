package handlers

import (
	"strings"

	"github.com/rhl/businessos-backend/internal/database/sqlc"
)

// stringToMemberStatus converts a string to sqlc.Memberstatus.
func stringToMemberStatus(s string) sqlc.Memberstatus {
	typeMap := map[string]sqlc.Memberstatus{
		"available":  sqlc.MemberstatusAVAILABLE,
		"busy":       sqlc.MemberstatusBUSY,
		"overloaded": sqlc.MemberstatusOVERLOADED,
		"ooo":        sqlc.MemberstatusOOO,
	}
	if enum, ok := typeMap[strings.ToLower(s)]; ok {
		return enum
	}
	return sqlc.MemberstatusAVAILABLE
}
