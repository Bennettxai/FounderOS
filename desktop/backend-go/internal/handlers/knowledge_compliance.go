package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rhl/businessos-backend/internal/middleware"
	"github.com/rhl/businessos-backend/internal/services"
)

// DeleteCloudCopy removes ALL of a workspace's synced rows from the cloud copy
// (knowledge_documents), implementing the user's right-to-be-forgotten / data
// deletion. Cloud sync is opt-in; this is how a user withdraws data they
// previously chose to sync. Local engine files on the user's machine are never
// touched by this endpoint - only the cloud copy is deleted.
//
// It writes to the cloud DB directly when this machine owns the sync
// (CLOUD_DATABASE_URL set), otherwise this backend IS the cloud and it deletes
// from its own pool - mirroring KnowledgeSources / GetStorage. After deleting
// the documents it zeroes the workspace's storage accounting row so the usage
// meter reflects the deletion.
//
//	DELETE /api/knowledge/cloud?workspace=<slug>
//	-> { workspace, deleted: <count> }
func (h *KnowledgeHandler) DeleteCloudCopy(c *gin.Context) {
	user := middleware.GetCurrentUser(c)
	if user == nil {
		RespondUnauthorizedErr(c, "authentication required")
		return
	}
	slug := c.Query("workspace")
	if strings.TrimSpace(slug) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspace required"})
		return
	}
	// Guard the slug the same way the file-serving handlers do, so a caller can
	// never smuggle a traversal token into the DB predicate.
	if strings.ContainsAny(slug, "/\\") || strings.Contains(slug, "..") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid workspace"})
		return
	}
	ctx := c.Request.Context()

	deleted := int64(0)
	var runErr error
	forbidden := false
	del := func(pool *pgxpool.Pool) {
		if pool == nil {
			return
		}
		// Multi-tenant gate: only a member of this workspace may delete its
		// shared cloud copy. Destructive + cross-tenant, so fail-closed.
		if !h.callerMemberOfSlug(c, pool, slug) {
			forbidden = true
			return
		}
		// Resolve the workspace UUID (if any docs carry it) before deleting, so
		// we can zero the storage accounting row afterwards.
		var wsID string
		hasWsID := pool.QueryRow(ctx,
			`SELECT workspace_id::text FROM knowledge_documents WHERE workspace_slug=$1 AND workspace_id IS NOT NULL LIMIT 1`, slug).Scan(&wsID) == nil

		tag, err := pool.Exec(ctx, `DELETE FROM knowledge_documents WHERE workspace_slug=$1`, slug)
		if err != nil {
			runErr = err
			return
		}
		deleted = tag.RowsAffected()

		if hasWsID && strings.TrimSpace(wsID) != "" {
			// Best-effort: reflect the deletion in the usage meter. A missing
			// workspace_storage table/row is not an error for a delete.
			_, _ = pool.Exec(ctx,
				`UPDATE workspace_storage SET bytes_used=0, updated_at=now() WHERE workspace_id=$1`, wsID)
		}
	}

	if cloudURL := services.CloudDatabaseURL(); cloudURL != "" {
		cfg, err := pgxpool.ParseConfig(cloudURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "parse cloud url: " + err.Error()})
			return
		}
		// Supabase transaction pooler (:6543) doesn't support prepared statements.
		cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
		cloud, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "connect cloud: " + err.Error()})
			return
		}
		defer cloud.Close()
		del(cloud)
	} else {
		del(h.pool)
	}

	if forbidden {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this workspace"})
		return
	}
	if runErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete cloud copy: " + runErr.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"workspace": slug,
		"deleted":   deleted,
	})
}
