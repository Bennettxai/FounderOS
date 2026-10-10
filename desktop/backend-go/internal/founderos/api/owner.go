package api

import (
	"context"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rhl/businessos-backend/internal/middleware"
)

// OwnerOnly admits only the operator to /api/founderos: the owner of the FounderOS
// workspace, or an email in FOUNDEROS_OWNER_EMAIL (comma-separated). Any other
// signed-in BusinessOS user gets 403. It fails closed: with no database and no
// allowlist match, nobody gets in.
func OwnerOnly(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := middleware.GetCurrentUser(c)
		if user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "sign in required"})
			return
		}
		for _, e := range strings.Split(os.Getenv("FOUNDEROS_OWNER_EMAIL"), ",") {
			if e = strings.TrimSpace(e); e != "" && strings.EqualFold(e, user.Email) {
				c.Next()
				return
			}
		}
		if d != nil && d.Pool != nil {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
			defer cancel()
			var owns bool
			err := d.Pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM workspaces WHERE slug = 'founderos' AND owner_id = $1)`, user.ID).Scan(&owns)
			if err == nil && owns {
				c.Next()
				return
			}
		}
		// Not the owner of the founderos (HQ) workspace: on a fresh install
		// nobody is yet. setup tells the /os layout to show its setup panel.
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"setup": true,
			"error": "Founder OS isn't set up for this account yet. Run: make demo OWNER=" + user.Email,
		})
	}
}
