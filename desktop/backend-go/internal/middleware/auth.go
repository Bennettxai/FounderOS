package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// BetterAuthUser represents a user from Better Auth's user table
type BetterAuthUser struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	Image         *string   `json:"image"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

const (
	UserContextKey    = "user"
	SessionCookieName = "better-auth.session_token"

	// Session configuration
	SessionMaxAge         = 7 * 24 * time.Hour  // 7 days max session lifetime
	SessionRefreshWindow  = 24 * time.Hour      // Refresh if less than 24h remaining
	SessionAbsoluteMaxAge = 30 * 24 * time.Hour // 30 days absolute maximum
)

// verifySessionCookie validates the HMAC signature on a Better Auth session cookie.
// Format: {token}.{HMAC-SHA256(secret, token)}
// Returns the raw token if valid, empty string + false if the signature is invalid.
//
// Backwards compatibility: if no dot is present the cookie is treated as a raw
// token (dev-only migration path). In PRODUCTION both fallbacks (no dot, no
// SECRET_KEY) are REFUSED so a dev migration path can't become a silent prod
// auth bypass (attacker-supplied session id → valid session).
func verifySessionCookie(sessionCookie string) (string, bool) {
	isProd := os.Getenv("ENVIRONMENT") == "production"
	idx := strings.LastIndex(sessionCookie, ".")
	if idx == -1 || idx == len(sessionCookie)-1 {
		if isProd {
			return "", false
		}
		// Dev only: no signature present — accept raw token during migration.
		return sessionCookie, true
	}

	token := sessionCookie[:idx]
	secret := os.Getenv("SECRET_KEY")
	if secret == "" {
		if isProd {
			slog.Error("session HMAC verification refused: SECRET_KEY not configured in production")
			return "", false
		}
		// Dev only: no secret configured — cannot verify, accept raw token.
		slog.Warn("session HMAC verification skipped: SECRET_KEY not configured")
		return token, true
	}

	sig := sessionCookie[idx+1:]
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(token))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return "", false
	}
	return token, true
}

// AuthMiddleware validates Better Auth session from cookie
// Implements sliding window session refresh for better security
func AuthMiddleware(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get session token from cookie
		sessionCookie, err := c.Cookie(SessionCookieName)
		if err != nil || sessionCookie == "" {
			slog.Debug("AuthMiddleware: no session cookie found")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
			return
		}

		slog.Debug("AuthMiddleware: session cookie received", "length", len(sessionCookie))

		// URL decode in case it's encoded
		sessionCookie, err = url.QueryUnescape(sessionCookie)
		if err != nil {
			slog.Debug("AuthMiddleware: URL decode failed", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid session cookie"})
			return
		}

		slog.Debug("AuthMiddleware: cookie decoded successfully")

		// Verify HMAC signature and extract the raw session token.
		sessionToken, valid := verifySessionCookie(sessionCookie)
		if !valid {
			slog.Debug("AuthMiddleware: invalid session cookie signature")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid session signature"})
			return
		}

		slog.Debug("AuthMiddleware: token extracted", "hasSignature", strings.Contains(sessionCookie, "."))

		// Look up session in Better Auth's session table
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var user BetterAuthUser
		var sessionExpiresAt time.Time
		var sessionCreatedAt time.Time
		err = pool.QueryRow(ctx, `
			SELECT u.id, u.name, u.email, u."emailVerified", u.image, u."createdAt", u."updatedAt",
			       s."expiresAt", s."createdAt"
			FROM session s
			JOIN "user" u ON s."userId" = u.id
			WHERE s.token = $1 AND s."expiresAt" > NOW()
		`, sessionToken).Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.EmailVerified,
			&user.Image,
			&user.CreatedAt,
			&user.UpdatedAt,
			&sessionExpiresAt,
			&sessionCreatedAt,
		)

		if err != nil {
			slog.Debug("AuthMiddleware: session lookup failed", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session"})
			return
		}

		// Check absolute session lifetime (30 days from creation)
		if time.Since(sessionCreatedAt) > SessionAbsoluteMaxAge {
			slog.Info("AuthMiddleware: session exceeded absolute max age, requiring re-authentication",
				"userID", user.ID, "sessionAge", time.Since(sessionCreatedAt))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Session expired. Please sign in again.",
				"code":  "SESSION_ABSOLUTE_TIMEOUT",
			})
			return
		}

		// Sliding window refresh: extend session if within refresh window
		timeUntilExpiry := time.Until(sessionExpiresAt)
		if timeUntilExpiry < SessionRefreshWindow {
			// Session is close to expiry, refresh it
			newExpiresAt := time.Now().Add(SessionMaxAge)
			_, err := pool.Exec(ctx, `
				UPDATE session SET "expiresAt" = $1, "updatedAt" = NOW()
				WHERE token = $2
			`, newExpiresAt, sessionToken)
			if err != nil {
				slog.Warn("AuthMiddleware: failed to refresh session", "error", err, "userID", user.ID)
				// Continue anyway - session is still valid
			} else {
				slog.Debug("AuthMiddleware: session refreshed", "userID", user.ID, "newExpiry", newExpiresAt)
			}
		}

		slog.Debug("AuthMiddleware: user authenticated", "userID", user.ID, "email", user.Email)

		// Store user in context
		c.Set(UserContextKey, &user)
		// Also set user_id as string for integration handlers
		c.Set("user_id", user.ID)
		c.Next()
	}
}

// GetCurrentUser retrieves the authenticated user from context
func GetCurrentUser(c *gin.Context) *BetterAuthUser {
	user, exists := c.Get(UserContextKey)
	if !exists {
		return nil
	}
	if user == nil {
		return nil
	}
	return user.(*BetterAuthUser)
}

// RequireAuth ensures a user is authenticated, aborting with 401 if not.
// MUST be used AFTER AuthMiddleware or CachedAuthMiddleware in the middleware chain.
// This provides a clean separation: AuthMiddleware sets the user, RequireAuth enforces it.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := GetCurrentUser(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "UNAUTHENTICATED",
			})
			return
		}
		c.Next()
	}
}

// MustGetCurrentUser retrieves the authenticated user from context.
// ONLY use this in handlers protected by RequireAuth() middleware.
// Returns HTTP 500 and aborts if user is nil (indicates middleware misconfiguration).
// This prevents server crashes while still clearly indicating a programming error.
func MustGetCurrentUser(c *gin.Context) *BetterAuthUser {
	user := GetCurrentUser(c)
	if user == nil {
		// This should NEVER happen if RequireAuth() is properly configured
		// Log the error and return 500 instead of panicking to prevent server crash
		slog.Error("BUG: user not in context despite RequireAuth() middleware",
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
		)
		c.JSON(500, gin.H{
			"error": "Internal server error: authentication middleware misconfiguration",
		})
		c.Abort()
		return nil
	}
	return user
}

// OptionalAuthMiddleware allows unauthenticated requests but sets user if authenticated
func OptionalAuthMiddleware(pool *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionCookie, err := c.Cookie(SessionCookieName)
		if err != nil || sessionCookie == "" {
			c.Next()
			return
		}

		sessionCookie, err = url.QueryUnescape(sessionCookie)
		if err != nil {
			c.Next()
			return
		}

		sessionToken, valid := verifySessionCookie(sessionCookie)
		if !valid {
			slog.Debug("OptionalAuthMiddleware: invalid session cookie signature")
			c.Next()
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		var user BetterAuthUser
		err = pool.QueryRow(ctx, `
			SELECT u.id, u.name, u.email, u."emailVerified", u.image, u."createdAt", u."updatedAt"
			FROM session s
			JOIN "user" u ON s."userId" = u.id
			WHERE s.token = $1 AND s."expiresAt" > NOW()
		`, sessionToken).Scan(
			&user.ID,
			&user.Name,
			&user.Email,
			&user.EmailVerified,
			&user.Image,
			&user.CreatedAt,
			&user.UpdatedAt,
		)

		if err == nil {
			c.Set(UserContextKey, &user)
		}
		c.Next()
	}
}
