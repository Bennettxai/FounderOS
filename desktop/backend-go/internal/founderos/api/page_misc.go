package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/trakyo"
)

// Small FounderOS v1 routes no single page owns: /api/tools, /api/contacts/tags,
// /api/trakyo/{workspace,content,links}. (/api/departments and /api/cron/run
// live with /org and /workflows.)
func init() { RegisterPage(registerMisc) }

// CONTACT_TIERS (FounderOS v1 lib/life-map.ts).
var contactTiers = []gin.H{
	{"tier": 1, "label": "Priority 1", "color": "#ef4444", "respond": "ASAP", "tags": []string{"client", "student"}},
	{"tier": 2, "label": "Priority 2", "color": "#eab308", "respond": "same day", "tags": []string{"brand", "partner", "lead"}},
	{"tier": 3, "label": "Priority 3", "color": "#22c55e", "respond": "when free", "tags": []string{"personal", "friend", "community"}},
}

func needPool(c *gin.Context, d *Deps) bool {
	if d.Pool == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "database unavailable"})
		return false
	}
	return true
}

func wsOf(c *gin.Context, d *Deps, slug string) (string, bool) {
	var id string
	if err := d.Pool.QueryRow(c.Request.Context(), `SELECT id::text FROM workspaces WHERE slug = $1`, slug).Scan(&id); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "workspace " + slug + " missing: run founderos-bootstrap"})
		return "", false
	}
	return id, true
}

func registerMisc(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/tools", func(c *gin.Context) {
		if !needPool(c, d) {
			return
		}
		rows, err := d.Pool.Query(c.Request.Context(), `SELECT id, name, category, status, color, description FROM founderos_tools ORDER BY name`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		out := []gin.H{}
		for rows.Next() {
			var id, name, cat, status, color, desc string
			if err := rows.Scan(&id, &name, &cat, &status, &color, &desc); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, gin.H{"id": id, "name": name, "category": cat, "status": status, "color": color, "description": desc})
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tools": out})
	})

	type tag struct {
		Person  string `json:"person"`
		Channel string `json:"channel"`
		Tag     string `json:"tag"`
		Tier    int    `json:"tier"`
	}
	s.GET("/pages/contacts/tags", func(c *gin.Context) {
		if !needPool(c, d) {
			return
		}
		rows, err := d.Pool.Query(c.Request.Context(), `SELECT person, channel, tag, tier FROM founderos_contact_tags ORDER BY person, channel`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		tags := []tag{}
		for rows.Next() {
			var t tag
			if err := rows.Scan(&t.Person, &t.Channel, &t.Tag, &t.Tier); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			tags = append(tags, t)
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"tiers": contactTiers, "tags": tags})
	})
	s.POST("/pages/contacts/tags", func(c *gin.Context) {
		var t tag
		if err := c.ShouldBindJSON(&t); err != nil || strings.TrimSpace(t.Person) == "" || strings.TrimSpace(t.Channel) == "" || strings.TrimSpace(t.Tag) == "" || t.Tier < 1 || t.Tier > 3 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "person, channel, tag required; tier 1–3"})
			return
		}
		if !needPool(c, d) {
			return
		}
		ws, ok := wsOf(c, d, "personal") // table map: contact tags are Personal
		if !ok {
			return
		}
		if _, err := d.Pool.Exec(c.Request.Context(), `INSERT INTO founderos_contact_tags (person, channel, workspace_id, tag, tier) VALUES ($1,$2,$3,$4,$5)
			ON CONFLICT (person, channel) DO UPDATE SET tag = EXCLUDED.tag, tier = EXCLUDED.tier`, t.Person, t.Channel, ws, t.Tag, t.Tier); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "tag": t})
	})
	s.DELETE("/pages/contacts/tags", func(c *gin.Context) {
		var body struct {
			Person  string `json:"person"`
			Channel string `json:"channel"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Person == "" || body.Channel == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "person and channel required"})
			return
		}
		if !needPool(c, d) {
			return
		}
		if _, err := d.Pool.Exec(c.Request.Context(), `DELETE FROM founderos_contact_tags WHERE person = $1 AND channel = $2`, body.Person, body.Channel); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	s.GET("/pages/trakyo/workspace", func(c *gin.Context) {
		c.Header("Cache-Control", "private, no-store")
		c.JSON(http.StatusOK, trakyo.New(d.Resolver).LinkWorkspace(c.Request.Context()))
	})
	s.POST("/pages/trakyo/content", func(c *gin.Context) {
		var in trakyo.ContentInput
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}
		out, err := trakyo.New(d.Resolver).CreateContent(c.Request.Context(), in)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, out)
	})
	s.POST("/pages/trakyo/links", func(c *gin.Context) {
		var in trakyo.LinkInput
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}
		out, err := trakyo.New(d.Resolver).CreateLink(c.Request.Context(), in)
		if err != nil {
			writeError(c, err)
			return
		}
		c.JSON(http.StatusCreated, out)
	})
}
