package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// ============================================================================
// COMMS CONTACTS SEARCH — merged contact suggestions for compose autocomplete
// ============================================================================
//
// Backs the recipient autocomplete in the Email tab compose modal. Frontend
// contract is in frontend/src/lib/api/comms/contacts.ts and types.ts:
//
//   GET /api/comms/contacts/search?q=<query>&limit=8
//   → ContactSuggestion[] = { email, name?, source: "crm"|"microsoft"|"hubspot"|"frequency" }
//
// Sources are merged across `clients` (BO's CRM contact table — note: the
// audit's "crm_contacts" actually maps to `clients`), `microsoft_contacts`,
// `hubspot_contacts`, plus a "frequency" source synthesized from distinct
// recipients/senders in the user's `emails` table.
//
// Each source is queried independently so a single broken table doesn't
// break the endpoint. Results are de-duplicated by email (case-insensitive),
// preferring the source with the most explicit name and earliest priority
// in the {crm > microsoft > hubspot > frequency} order.

type contactSuggestion struct {
	Email  string `json:"email"`
	Name   string `json:"name,omitempty"`
	Source string `json:"source"` // "crm" | "microsoft" | "hubspot" | "frequency"
}

// SearchContacts returns up to `limit` matched contacts across the four
// sources. Empty `q` returns []. q matches email, name, or company fields
// via case-insensitive prefix-anywhere (`ILIKE %q%`).
func (h *commsHandler) SearchContacts(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusOK, []contactSuggestion{})
		return
	}

	limit := parseIntDefault(c.Query("limit"), 8, 1)
	if limit > 50 {
		limit = 50
	}

	ctx := c.Request.Context()
	pat := "%" + q + "%"

	// Each source is independent — log+continue on error rather than
	// 500-ing the whole endpoint when one table or column happens to be
	// missing on a fresh install.
	var all []contactSuggestion
	all = append(all, h.searchClients(ctx, userID, pat, limit)...)
	all = append(all, h.searchMicrosoftContacts(ctx, userID, pat, limit)...)
	all = append(all, h.searchHubspotContacts(ctx, userID, pat, limit)...)
	all = append(all, h.searchFrequency(ctx, userID, pat, limit)...)

	// De-duplicate by lowercased email, preferring earlier sources.
	seen := make(map[string]int, len(all))
	merged := make([]contactSuggestion, 0, len(all))
	for _, s := range all {
		if s.Email == "" {
			continue
		}
		key := strings.ToLower(s.Email)
		if idx, ok := seen[key]; ok {
			// Already saw this email — prefer the existing entry's source
			// (which is earlier in priority), but fill a missing name.
			if merged[idx].Name == "" && s.Name != "" {
				merged[idx].Name = s.Name
			}
			continue
		}
		seen[key] = len(merged)
		merged = append(merged, s)
		if len(merged) >= limit {
			break
		}
	}

	c.JSON(http.StatusOK, merged)
}

// searchClients queries the BO CRM contact table (`clients`).
func (h *commsHandler) searchClients(ctx context.Context, userID, pat string, limit int) []contactSuggestion {
	rows, err := h.pool.Query(ctx, `
		SELECT email, name
		FROM clients
		WHERE user_id = $1
		  AND email IS NOT NULL AND email <> ''
		  AND (email ILIKE $2 OR name ILIKE $2)
		ORDER BY name NULLS LAST
		LIMIT $3
	`, userID, pat, limit)
	if err != nil {
		slog.Info("comms contacts: clients query failed", "error", err)
		return nil
	}
	defer rows.Close()

	var out []contactSuggestion
	for rows.Next() {
		var email, name *string
		if err := rows.Scan(&email, &name); err != nil {
			continue
		}
		s := contactSuggestion{Source: "crm"}
		if email != nil {
			s.Email = *email
		}
		if name != nil {
			s.Name = *name
		}
		if s.Email != "" {
			out = append(out, s)
		}
	}
	return out
}

// searchMicrosoftContacts unrolls the email_addresses JSONB array. We expect
// each element to be {address, type} — Microsoft Graph's standard shape.
func (h *commsHandler) searchMicrosoftContacts(ctx context.Context, userID, pat string, limit int) []contactSuggestion {
	rows, err := h.pool.Query(ctx, `
		SELECT display_name, email_addresses
		FROM microsoft_contacts
		WHERE user_id = $1
		  AND (display_name ILIKE $2 OR given_name ILIKE $2 OR surname ILIKE $2 OR email_addresses::text ILIKE $2)
		ORDER BY display_name NULLS LAST
		LIMIT $3
	`, userID, pat, limit)
	if err != nil {
		slog.Info("comms contacts: microsoft query failed", "error", err)
		return nil
	}
	defer rows.Close()

	var out []contactSuggestion
	for rows.Next() {
		var displayName *string
		var emailsRaw []byte
		if err := rows.Scan(&displayName, &emailsRaw); err != nil {
			continue
		}
		name := ""
		if displayName != nil {
			name = *displayName
		}
		// Try {address, type} shape first, then a plain string array.
		var typed []struct {
			Address string `json:"address"`
			Name    string `json:"name"`
		}
		if err := json.Unmarshal(emailsRaw, &typed); err == nil && len(typed) > 0 {
			for _, e := range typed {
				if e.Address != "" {
					n := name
					if n == "" {
						n = e.Name
					}
					out = append(out, contactSuggestion{Email: e.Address, Name: n, Source: "microsoft"})
				}
			}
			continue
		}
		var strs []string
		if err := json.Unmarshal(emailsRaw, &strs); err == nil {
			for _, e := range strs {
				if e != "" {
					out = append(out, contactSuggestion{Email: e, Name: name, Source: "microsoft"})
				}
			}
		}
	}
	return out
}

// searchHubspotContacts queries hubspot_contacts (schema.sql:1642).
func (h *commsHandler) searchHubspotContacts(ctx context.Context, userID, pat string, limit int) []contactSuggestion {
	rows, err := h.pool.Query(ctx, `
		SELECT email, first_name, last_name
		FROM hubspot_contacts
		WHERE user_id = $1
		  AND email IS NOT NULL AND email <> ''
		  AND (email ILIKE $2 OR first_name ILIKE $2 OR last_name ILIKE $2)
		ORDER BY last_name NULLS LAST, first_name NULLS LAST
		LIMIT $3
	`, userID, pat, limit)
	if err != nil {
		slog.Info("comms contacts: hubspot query failed", "error", err)
		return nil
	}
	defer rows.Close()

	var out []contactSuggestion
	for rows.Next() {
		var email, firstName, lastName *string
		if err := rows.Scan(&email, &firstName, &lastName); err != nil {
			continue
		}
		s := contactSuggestion{Source: "hubspot"}
		if email != nil {
			s.Email = *email
		}
		s.Name = strings.TrimSpace(deref(firstName) + " " + deref(lastName))
		if s.Email != "" {
			out = append(out, s)
		}
	}
	return out
}

// searchFrequency picks distinct sender emails the user has actually
// corresponded with, weighted by frequency. Acts as the long tail when no
// CRM/contact-book row exists for someone the user emails regularly.
func (h *commsHandler) searchFrequency(ctx context.Context, userID, pat string, limit int) []contactSuggestion {
	rows, err := h.pool.Query(ctx, `
		SELECT from_email, MAX(COALESCE(from_name, '')) AS sender_name, COUNT(*) AS freq
		FROM emails
		WHERE user_id = $1
		  AND from_email IS NOT NULL AND from_email <> ''
		  AND (from_email ILIKE $2 OR from_name ILIKE $2)
		GROUP BY from_email
		ORDER BY freq DESC
		LIMIT $3
	`, userID, pat, limit)
	if err != nil {
		slog.Info("comms contacts: frequency query failed", "error", err)
		return nil
	}
	defer rows.Close()

	var out []contactSuggestion
	for rows.Next() {
		var email, name string
		var freq int
		if err := rows.Scan(&email, &name, &freq); err != nil {
			continue
		}
		if email != "" {
			out = append(out, contactSuggestion{Email: email, Name: name, Source: "frequency"})
		}
	}
	return out
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
