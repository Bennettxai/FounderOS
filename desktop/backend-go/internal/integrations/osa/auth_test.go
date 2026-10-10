package osa

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAuthToken(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	secret := "test-secret-key-min-32-bytes-long"

	tests := []struct {
		name        string
		userID      uuid.UUID
		workspaceID *uuid.UUID
		wantErr     bool
	}{
		{
			name:        "with workspace ID",
			userID:      userID,
			workspaceID: &workspaceID,
			wantErr:     false,
		},
		{
			name:        "without workspace ID",
			userID:      userID,
			workspaceID: nil,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateAuthToken(tt.userID, tt.workspaceID, secret)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				// Verify token can be parsed
				claims, err := ValidateAuthToken(token, secret)
				require.NoError(t, err)
				assert.Equal(t, tt.userID.String(), claims.UserID)

				if tt.workspaceID != nil {
					assert.Equal(t, tt.workspaceID.String(), claims.WorkspaceID)
				} else {
					assert.Empty(t, claims.WorkspaceID)
				}
			}
		})
	}
}

func TestValidateAuthToken(t *testing.T) {
	userID := uuid.New()
	workspaceID := uuid.New()
	secret := "test-secret-key-min-32-bytes-long"

	t.Run("valid token", func(t *testing.T) {
		token, err := GenerateAuthToken(userID, &workspaceID, secret)
		require.NoError(t, err)

		claims, err := ValidateAuthToken(token, secret)

		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, userID.String(), claims.UserID)
		assert.Equal(t, workspaceID.String(), claims.WorkspaceID)
		assert.Equal(t, "BusinessOS", claims.Issuer)
		assert.Equal(t, userID.String(), claims.Subject)
	})

	t.Run("invalid token - wrong secret", func(t *testing.T) {
		token, err := GenerateAuthToken(userID, nil, secret)
		require.NoError(t, err)

		claims, err := ValidateAuthToken(token, "wrong-secret")

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("invalid token - malformed", func(t *testing.T) {
		claims, err := ValidateAuthToken("not-a-valid-token", secret)

		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("invalid token - expired", func(t *testing.T) {
		// Create an expired token
		claims := Claims{
			UserID: userID.String(),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
				NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
				Issuer:    "BusinessOS",
				Subject:   userID.String(),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		signedToken, err := token.SignedString([]byte(secret))
		require.NoError(t, err)

		parsedClaims, err := ValidateAuthToken(signedToken, secret)

		assert.Error(t, err)
		assert.Nil(t, parsedClaims)
	})

	t.Run("invalid token - wrong signing method", func(t *testing.T) {
		// Create token with RSA (asymmetric) instead of HMAC (symmetric)
		// This should definitely fail our validation
		claims := Claims{
			UserID: userID.String(),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Issuer:    "BusinessOS",
			},
		}

		// Create a simple RSA private key for testing (this will fail HMAC validation)
		// We'll create a token that looks valid but uses wrong algorithm
		token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		signedToken, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		parsedClaims, err := ValidateAuthToken(signedToken, secret)

		assert.Error(t, err)
		assert.Nil(t, parsedClaims)
	})
}

func TestTokenExpiration(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret-key-min-32-bytes-long"

	token, err := GenerateAuthToken(userID, nil, secret)
	require.NoError(t, err)

	claims, err := ValidateAuthToken(token, secret)
	require.NoError(t, err)

	// Verify expiration is set to 15 minutes from now
	expiresAt := claims.ExpiresAt.Time
	expectedExpiry := time.Now().Add(15 * time.Minute)

	// Allow 1 second tolerance for test execution time
	assert.WithinDuration(t, expectedExpiry, expiresAt, 1*time.Second)
}

func TestTokenIssuerAndSubject(t *testing.T) {
	userID := uuid.New()
	secret := "test-secret-key-min-32-bytes-long"

	token, err := GenerateAuthToken(userID, nil, secret)
	require.NoError(t, err)

	claims, err := ValidateAuthToken(token, secret)
	require.NoError(t, err)

	assert.Equal(t, "BusinessOS", claims.Issuer)
	assert.Equal(t, userID.String(), claims.Subject)
	assert.Equal(t, userID.String(), claims.UserID)
}
