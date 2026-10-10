package etl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/security"
)

// VaultProvider is the credential_vault provider_id holding every proposal's
// StatiCrypt access code as one encrypted JSON object {proposal_id: code}.
const VaultProvider = "founderos-staticrypt"

// moveAccessCodes mirrors the proposal access codes into credential_vault. The
// row is only rewritten when the decrypted content differs, so a re-run is a
// no-op. Values are never logged or reported, only counted.
func moveAccessCodes(ctx context.Context, pool *pgxpool.Pool, opts Options, ws map[string]workspaceRef, codes map[string]string, proposalsLoaded bool) (VaultReport, error) {
	vr := VaultReport{Provider: VaultProvider, Codes: len(codes), Action: "none"}
	if !proposalsLoaded {
		return vr, nil
	}
	user := opts.VaultUserID
	if user == "" {
		user = ws[WSFounderOS].owner
	}
	vr.UserID = user

	var existing []byte
	err := pool.QueryRow(ctx, `SELECT encrypted_data FROM credential_vault WHERE user_id = $1 AND provider_id = $2`, user, VaultProvider).Scan(&existing)
	hasRow := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return vr, fmt.Errorf("read credential_vault: %w", err)
	}

	if len(codes) == 0 {
		if !hasRow {
			return vr, nil
		}
		if _, err := pool.Exec(ctx, `DELETE FROM credential_vault WHERE user_id = $1 AND provider_id = $2`, user, VaultProvider); err != nil {
			return vr, fmt.Errorf("delete stale vault row: %w", err)
		}
		vr.Action = "deleted"
		return vr, nil
	}

	if opts.EncryptionKey == "" {
		return vr, fmt.Errorf("%d proposal access codes need TOKEN_ENCRYPTION_KEY (the backend's key) to be stored in credential_vault; they are never stored in plain text", len(codes))
	}
	enc, err := security.NewTokenEncryption(opts.EncryptionKey)
	if err != nil {
		return vr, fmt.Errorf("TOKEN_ENCRYPTION_KEY: %w", err)
	}

	if hasRow {
		if plain, err := enc.DecryptBytes(existing); err == nil {
			var have map[string]string
			if json.Unmarshal([]byte(plain), &have) == nil && maps.Equal(have, codes) {
				vr.Action = "unchanged"
				return vr, nil
			}
		}
	}

	var userExists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM "user" WHERE id = $1)`, user).Scan(&userExists); err != nil {
		return vr, err
	}
	if !userExists {
		return vr, fmt.Errorf("vault owner %q is not a BusinessOS user; pass --vault-user", user)
	}

	payload, err := json.Marshal(codes)
	if err != nil {
		return vr, err
	}
	cipher, err := enc.EncryptBytes(string(payload))
	if err != nil {
		return vr, err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return vr, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after commit
	if _, err := tx.Exec(ctx, `
		INSERT INTO credential_vault (user_id, provider_id, credential_type, encrypted_data, encryption_version, metadata)
		VALUES ($1, $2, 'api_key', $3, 1, '{"source":"founderos-os proposals"}'::jsonb)
		ON CONFLICT (user_id, provider_id) DO UPDATE SET
			credential_type = EXCLUDED.credential_type,
			encrypted_data = EXCLUDED.encrypted_data,
			encryption_version = EXCLUDED.encryption_version,
			metadata = EXCLUDED.metadata,
			updated_at = now(),
			last_rotated_at = now()`, user, VaultProvider, cipher); err != nil {
		return vr, fmt.Errorf("write credential_vault: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return vr, err
	}
	if hasRow {
		vr.Action = "updated"
	} else {
		vr.Action = "inserted"
	}
	return vr, nil
}
