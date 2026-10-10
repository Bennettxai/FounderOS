package database

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// EnsureAuthTables creates the Better Auth tables if they don't exist.
// Uses IF NOT EXISTS so it's safe to call on every startup.
func EnsureAuthTables(ctx context.Context, pool *pgxpool.Pool) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS "user" (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255),
			email VARCHAR(255) UNIQUE NOT NULL,
			"emailVerified" BOOLEAN DEFAULT FALSE,
			image TEXT,
			"createdAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			"updatedAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS session (
			id VARCHAR(255) PRIMARY KEY,
			"userId" VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
			token VARCHAR(255) UNIQUE NOT NULL,
			"expiresAt" TIMESTAMP WITH TIME ZONE NOT NULL,
			"ipAddress" VARCHAR(45),
			"userAgent" TEXT,
			"createdAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			"updatedAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS account (
			id VARCHAR(255) PRIMARY KEY,
			"userId" VARCHAR(255) NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
			"accountId" VARCHAR(255) NOT NULL,
			"providerId" VARCHAR(255) NOT NULL,
			"accessToken" TEXT,
			"refreshToken" TEXT,
			"accessTokenExpiresAt" TIMESTAMP WITH TIME ZONE,
			"refreshTokenExpiresAt" TIMESTAMP WITH TIME ZONE,
			scope TEXT,
			password TEXT,
			"createdAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			"updatedAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS verification (
			id VARCHAR(255) PRIMARY KEY,
			identifier VARCHAR(255) NOT NULL,
			value VARCHAR(255) NOT NULL,
			"expiresAt" TIMESTAMP WITH TIME ZONE NOT NULL,
			"createdAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_session_user ON session("userId")`,
		`CREATE INDEX IF NOT EXISTS idx_session_token ON session(token)`,
		`CREATE INDEX IF NOT EXISTS idx_account_user ON account("userId")`,
		`CREATE INDEX IF NOT EXISTS idx_account_provider ON account("providerId", "accountId")`,
		`ALTER TABLE "user" ADD COLUMN IF NOT EXISTS platform_role VARCHAR(20) DEFAULT 'user'`,
		`UPDATE "user" SET platform_role = 'user' WHERE platform_role IS NULL`,
		`DO $$
		BEGIN
			ALTER TABLE "user"
				ADD CONSTRAINT user_platform_role_check
				CHECK (platform_role IN ('superadmin', 'admin', 'user'));
		EXCEPTION
			WHEN duplicate_object THEN NULL;
		END $$`,
	}

	for _, q := range queries {
		if _, err := pool.Exec(ctx, q); err != nil {
			slog.Error("failed to ensure auth table", "error", err)
			return err
		}
	}

	slog.Info("Better Auth tables verified")
	return nil
}
