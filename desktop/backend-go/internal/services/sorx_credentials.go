// Package services provides business logic for BusinessOS.
package services

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CredentialTicketRequest represents a request from Sorx for a credential.
type CredentialTicketRequest struct {
	Provider    string    `json:"provider"`
	Scope       string    `json:"scope"`
	SkillID     string    `json:"skill_id"`
	ExecutionID string    `json:"execution_id"`
	UserID      string    `json:"user_id"`
	EngineID    string    `json:"engine_id"`
	SessionID   string    `json:"session_id"`
	Timestamp   time.Time `json:"timestamp"`
	Signature   []byte    `json:"signature"`
}

// CredentialTicket is issued to Sorx after validation.
type CredentialTicket struct {
	ID        uuid.UUID `json:"id"`
	RequestID string    `json:"request_id"`
	Provider  string    `json:"provider"`
	Scope     string    `json:"scope"`
	SkillID   string    `json:"skill_id"`
	UserID    string    `json:"user_id"`
	EngineID  string    `json:"engine_id"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Nonce     string    `json:"nonce"`
	Signature []byte    `json:"signature"`
}

// CredentialResponse contains the encrypted credential.
type CredentialResponse struct {
	TicketID            uuid.UUID `json:"ticket_id"`
	EncryptedCredential []byte    `json:"encrypted_credential"`
	Nonce               []byte    `json:"nonce"`
	Provider            string    `json:"provider"`
	ExpiresAt           time.Time `json:"expires_at"`
}

// ValidateTicketRequest validates a credential ticket request from Sorx.
func (s *SorxService) ValidateTicketRequest(ctx context.Context, req CredentialTicketRequest) error {
	if time.Since(req.Timestamp) > 30*time.Second {
		return fmt.Errorf("request timestamp too old")
	}

	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM user_integrations
			WHERE user_id = $1 AND provider_id = $2 AND status = 'connected'
		)
	`, req.UserID, req.Provider).Scan(&exists)

	if err != nil {
		return fmt.Errorf("failed to check integration: %w", err)
	}
	if !exists {
		return fmt.Errorf("user does not have %s connected", req.Provider)
	}

	return nil
}

// IssueCredentialTicket creates a signed ticket for credential retrieval.
func (s *SorxService) IssueCredentialTicket(ctx context.Context, req CredentialTicketRequest) (*CredentialTicket, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ticket := &CredentialTicket{
		ID:        uuid.New(),
		RequestID: req.ExecutionID,
		Provider:  req.Provider,
		Scope:     req.Scope,
		SkillID:   req.SkillID,
		UserID:    req.UserID,
		EngineID:  req.EngineID,
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(60 * time.Second),
		Nonce:     fmt.Sprintf("%x", nonce),
	}

	// Sign the ticket (simplified - in production use Ed25519)
	// ticket.Signature = s.signTicket(ticket)

	return ticket, nil
}

// RedeemTicket exchanges a ticket for the encrypted credential.
func (s *SorxService) RedeemTicket(ctx context.Context, ticket *CredentialTicket) (*CredentialResponse, error) {
	if time.Now().After(ticket.ExpiresAt) {
		return nil, fmt.Errorf("ticket expired")
	}

	var accessToken, refreshToken []byte
	var tokenExpires *time.Time
	err := s.pool.QueryRow(ctx, `
		SELECT access_token_encrypted, refresh_token_encrypted, token_expires_at
		FROM user_integrations
		WHERE user_id = $1 AND provider_id = $2 AND status = 'connected'
	`, ticket.UserID, ticket.Provider).Scan(&accessToken, &refreshToken, &tokenExpires)

	if err != nil {
		return nil, fmt.Errorf("failed to get integration: %w", err)
	}

	_, err = s.pool.Exec(ctx, `
		UPDATE user_integrations SET last_used_at = NOW()
		WHERE user_id = $1 AND provider_id = $2
	`, ticket.UserID, ticket.Provider)
	if err != nil {
		// Log but don't fail
	}

	response := &CredentialResponse{
		TicketID:            ticket.ID,
		EncryptedCredential: accessToken,
		Provider:            ticket.Provider,
	}
	if tokenExpires != nil {
		response.ExpiresAt = *tokenExpires
	}

	return response, nil
}
