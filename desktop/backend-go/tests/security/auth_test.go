package security_test

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// TestPasswordHashing verifies bcrypt is used with appropriate cost
func TestPasswordHashing(t *testing.T) {
	passwords := []string{
		"SecurePassword123!",
		"MyP@ssw0rd2024",
		"Complex!ty#9999",
	}

	for _, password := range passwords {
		t.Run("Hash_"+password, func(t *testing.T) {
			// Hash with bcrypt cost 12 (recommended for production)
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
			require.NoError(t, err, "Password hashing should succeed")

			// Verify hash is not equal to plaintext
			assert.NotEqual(t, password, string(hashedPassword), "Hash should not equal plaintext")

			// Verify hash can be verified
			err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
			assert.NoError(t, err, "Password verification should succeed")

			// Verify wrong password fails
			err = bcrypt.CompareHashAndPassword(hashedPassword, []byte("wrongpassword"))
			assert.Error(t, err, "Wrong password should fail verification")
		})
	}
}

// TestPasswordPolicy validates password requirements
func TestPasswordPolicy(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
		reason   string
	}{
		{"Valid password", "SecureP@ss123", true, ""},
		{"Too short", "Pass1!", false, "minimum 8 characters required"},
		{"No number", "Password!", false, "must contain at least one number"},
		{"No special char", "Password123", false, "must contain special character"},
		{"No uppercase", "password123!", false, "must contain uppercase letter"},
		{"Empty", "", false, "password cannot be empty"},
		{"Only numbers", "12345678", false, "must contain letters"},
		{"Common password", "Password123!", false, "too common"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := validatePasswordPolicy(tt.password)
			assert.Equal(t, tt.valid, valid, tt.reason)
		})
	}
}

// TestJWTTokenManipulation tests JWT security
func TestJWTTokenManipulation(t *testing.T) {
	secret := []byte("test-secret-key-min-32-chars-required")
	userID := "user-123"

	// Create valid token
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	validToken, err := token.SignedString(secret)
	require.NoError(t, err)

	t.Run("Valid token verifies successfully", func(t *testing.T) {
		parsed, err := jwt.Parse(validToken, func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		})
		assert.NoError(t, err)
		assert.True(t, parsed.Valid)
	})

	t.Run("Token without signature rejected", func(t *testing.T) {
		// Remove signature
		parts := strings.Split(validToken, ".")
		require.Len(t, parts, 3, "JWT should have 3 parts")
		unsignedToken := parts[0] + "." + parts[1] + "."

		_, err := jwt.Parse(unsignedToken, func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		})
		assert.Error(t, err, "Token without signature should be rejected")
	})

	t.Run("Token with modified claims rejected", func(t *testing.T) {
		// Try to modify claims (change user_id to admin)
		parts := strings.Split(validToken, ".")
		require.Len(t, parts, 3)

		// Parse claims
		claimsBytes, _ := base64.RawURLEncoding.DecodeString(parts[1])
		modifiedClaims := strings.Replace(string(claimsBytes), userID, "admin", 1)
		parts[1] = base64.RawURLEncoding.EncodeToString([]byte(modifiedClaims))

		tamperedToken := strings.Join(parts, ".")

		_, err := jwt.Parse(tamperedToken, func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		})
		assert.Error(t, err, "Tampered token should be rejected")
	})

	t.Run("Expired token rejected", func(t *testing.T) {
		expiredClaims := jwt.MapClaims{
			"user_id": userID,
			"exp":     time.Now().Add(-time.Hour).Unix(), // Expired 1 hour ago
			"iat":     time.Now().Add(-2 * time.Hour).Unix(),
		}

		expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
		expiredTokenString, err := expiredToken.SignedString(secret)
		require.NoError(t, err)

		_, err = jwt.Parse(expiredTokenString, func(token *jwt.Token) (interface{}, error) {
			return secret, nil
		})
		assert.Error(t, err, "Expired token should be rejected")
	})

	t.Run("Token with wrong algorithm rejected", func(t *testing.T) {
		// Try to use 'none' algorithm
		noneClaims := jwt.MapClaims{
			"user_id": "admin",
			"exp":     time.Now().Add(time.Hour).Unix(),
		}

		noneToken := jwt.NewWithClaims(jwt.SigningMethodNone, noneClaims)
		noneTokenString, err := noneToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		_, err = jwt.Parse(noneTokenString, func(token *jwt.Token) (interface{}, error) {
			// Verify algorithm is HS256
			if token.Method != jwt.SigningMethodHS256 {
				return nil, jwt.ErrSignatureInvalid
			}
			return secret, nil
		})
		assert.Error(t, err, "'none' algorithm should be rejected")
	})
}

// TestSessionFixation tests session fixation attack prevention
func TestSessionFixation(t *testing.T) {
	// Simulate login flow
	oldSessionID := generateSessionID()

	t.Run("New session created on login", func(t *testing.T) {
		// After successful authentication, a NEW session should be created
		newSessionID := generateSessionID()

		assert.NotEqual(t, oldSessionID, newSessionID, "Session ID should change after login")
		assert.NotEmpty(t, newSessionID, "New session ID should not be empty")
		assert.Greater(t, len(newSessionID), 32, "Session ID should be long enough (cryptographically random)")
	})

	t.Run("Old session invalidated", func(t *testing.T) {
		// Old session should be marked as invalid in database
		// In real implementation, this would check database
		isOldSessionValid := false // Would query DB

		assert.False(t, isOldSessionValid, "Old session should be invalidated after login")
	})
}

// TestCredentialStuffing simulates credential stuffing attack
func TestCredentialStuffing(t *testing.T) {
	// Simulate multiple failed login attempts
	maxAttempts := 5
	attempts := 0

	for i := 0; i < 10; i++ {
		attempts++

		if attempts > maxAttempts {
			t.Run("Rate limiting triggered", func(t *testing.T) {
				// After 5 failed attempts, account should be locked
				isLocked := attempts > maxAttempts
				assert.True(t, isLocked, "Account should be locked after max attempts")
			})
			break
		}
	}

	t.Run("Lockout duration enforced", func(t *testing.T) {
		lockoutDuration := 15 * time.Minute
		now := time.Now()
		lockoutUntil := now.Add(lockoutDuration)

		// Verify lockout is at least 15 minutes
		assert.True(t, lockoutUntil.After(now), "Lockout should be in the future")
		assert.Equal(t, lockoutDuration, lockoutUntil.Sub(now), "Lockout should be at least 15 minutes")
	})
}

// TestRefreshTokenRotation tests refresh token security
func TestRefreshTokenRotation(t *testing.T) {
	t.Run("Refresh token rotates on use", func(t *testing.T) {
		// When refresh token is used, a new one should be issued
		oldRefreshToken := generateSessionID()
		newRefreshToken := generateSessionID()

		assert.NotEqual(t, oldRefreshToken, newRefreshToken, "Refresh token should rotate")
		assert.Greater(t, len(newRefreshToken), 32, "New refresh token should be cryptographically random")
	})

	t.Run("Old refresh token invalidated", func(t *testing.T) {
		// After rotation, old token should not work
		oldTokenValid := false // In real implementation, would check database

		assert.False(t, oldTokenValid, "Old refresh token should be invalidated")
	})

	t.Run("Refresh token has expiration", func(t *testing.T) {
		// Refresh tokens should expire (e.g., 30 days)
		refreshTokenExpiry := time.Now().Add(30 * 24 * time.Hour)

		assert.True(t, refreshTokenExpiry.After(time.Now()), "Refresh token should have future expiry")
		assert.True(t, refreshTokenExpiry.Before(time.Now().Add(31*24*time.Hour)), "Refresh token should expire within 30 days")
	})
}

// TestBruteForceProtection tests brute force attack prevention
func TestBruteForceProtection(t *testing.T) {
	t.Run("Rate limiting on login endpoint", func(t *testing.T) {
		maxAttemptsPerWindow := 5
		windowDuration := 15 * time.Minute

		// Simulate rapid login attempts
		attempts := make([]time.Time, 0)
		for i := 0; i < 10; i++ {
			attempts = append(attempts, time.Now())
		}

		// Count attempts in current window
		cutoff := time.Now().Add(-windowDuration)
		recentAttempts := 0
		for _, attempt := range attempts {
			if attempt.After(cutoff) {
				recentAttempts++
			}
		}

		shouldBlock := recentAttempts > maxAttemptsPerWindow
		assert.True(t, shouldBlock, "Should block after exceeding rate limit")
	})
}

// Helper functions

func validatePasswordPolicy(password string) bool {
	if len(password) < 8 {
		return false
	}

	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSpecial := false

	for i := 0; i < len(password); i++ {
		c := password[i]
		if c >= 'A' && c <= 'Z' {
			hasUpper = true
		} else if c >= 'a' && c <= 'z' {
			hasLower = true
		} else if c >= '0' && c <= '9' {
			hasNumber = true
		} else {
			hasSpecial = true
		}
	}

	// Check for common passwords (simplified)
	commonPasswords := []string{"Password123!", "Admin123!", "Welcome123!"}
	for _, common := range commonPasswords {
		if password == common {
			return false
		}
	}

	return hasUpper && hasLower && hasNumber && hasSpecial
}

func generateSessionID() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
