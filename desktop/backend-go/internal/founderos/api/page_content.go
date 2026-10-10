package api

// /os/content and /os/content/lead-magnets (spec 6.14): the page models for
// FounderOS v1 app/content/page.tsx and app/content/lead-magnets/page.tsx, plus
// the lead-magnet writes the page makes. POST /api/lead-magnets itself is a
// compat route (the content-gen skill); the page's form posts to
// /pages/content/lead-magnets instead, through the same validation.

import (
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/zernio"
	"github.com/rhl/businessos-backend/internal/founderos/pages/content"
	"github.com/rhl/businessos-backend/internal/founderos/pages/pagekit"
)

func init() { RegisterPage(registerContent) }

const (
	contentWindowDays   = 30
	contentWindowWeeks  = 12
	contentRecentPosts  = 8
	contentZernioBudget = 10 * time.Second
)

// contentPostSource is the Zernio read the page needs (zernio.Connector).
type contentPostSource interface {
	RecentPosts(ctx context.Context, limit int) ([]zernio.Post, error)
	PostDays(ctx context.Context) ([]zernio.PostDay, error)
}

var (
	contentZernio = func(d *Deps) contentPostSource { return zernio.New(d.Resolver) }
	contentNow    = time.Now
)

// contentCrewMember is a crew agent plus whether the bridge runtime can run
// it, so the Run button says so instead of failing blind.
type contentCrewMember struct {
	content.Agent
	Runnable bool `json:"runnable"`
}

func registerContent(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/content", func(c *gin.Context) { contentPage(c, d) })
	s.GET("/pages/content/lead-magnets", func(c *gin.Context) { contentLeadMagnetList(c, d) })
	s.POST("/pages/content/lead-magnets", func(c *gin.Context) { contentLeadMagnetCreate(c, d) })
	s.PATCH("/pages/lead-magnets/:id", func(c *gin.Context) { contentLeadMagnetPatch(c, d) })
	s.DELETE("/pages/lead-magnets/:id", func(c *gin.Context) { contentLeadMagnetDelete(c, d) })
}

// contentWorkspace resolves a slug, answering 503 when the bridge has not
// been bootstrapped (an empty page would read as "nothing there").
func contentWorkspace(c *gin.Context, d *Deps, slug string) (string, bool) {
	ws, err := pagekit.WorkspaceID(c.Request.Context(), d.Pool, slug)
	if err != nil {
		code := http.StatusInternalServerError
		if errors.Is(err, pagekit.ErrNoWorkspace) {
			code = http.StatusServiceUnavailable
		}
		c.JSON(code, gin.H{"error": err.Error()})
		return "", false
	}
	return ws, true
}

func contentFail(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func contentPage(c *gin.Context, d *Deps) {
	ctx := c.Request.Context()
	agentsWS, ok := contentWorkspace(c, d, content.AgentsWorkspace)
	if !ok {
		return
	}
	lmWS, ok := contentWorkspace(c, d, content.LeadMagnetsWorkspace)
	if !ok {
		return
	}
	now := contentNow().UTC()
	today := pagekit.ISODay(now)

	all, err := content.Agents(ctx, d.Pool, agentsWS)
	if err != nil {
		contentFail(c, err)
		return
	}
	crew := content.ContentAgents(all)
	ids := make([]string, len(crew))
	members := make([]contentCrewMember, len(crew))
	for i, a := range crew {
		ids[i] = a.ID
		members[i] = contentCrewMember{Agent: a, Runnable: d.Agents != nil && d.Agents.Has(a.ID)}
	}
	runs, err := content.RunsSince(ctx, d.Pool, agentsWS, now.Add(-time.Duration(contentWindowDays+1)*24*time.Hour), ids)
	if err != nil {
		contentFail(c, err)
		return
	}
	magnets, err := content.LeadMagnets(ctx, d.Pool, lmWS)
	if err != nil {
		contentFail(c, err)
		return
	}

	// Zernio is soft: an outage makes the posting history unknown, never empty.
	zctx, cancel := context.WithTimeout(ctx, contentZernioBudget)
	defer cancel()
	src := contentZernio(d)
	posts, postsErr := src.RecentPosts(zctx, contentRecentPosts)
	days, daysErr := src.PostDays(zctx)
	if posts == nil || postsErr != nil {
		posts = []zernio.Post{}
	}
	var postsError any
	if postsErr != nil {
		postsError = postsErr.Error()
	}
	postsKnown := daysErr == nil
	var pipelineActive any // null when the history is unknown
	if postsKnown {
		active := map[string]bool{}
		for _, p := range days {
			if len(p.Platforms) > 0 {
				active[p.Date] = true
			}
		}
		pipelineActive = len(active)
	} else {
		days = nil
	}

	c.JSON(http.StatusOK, gin.H{
		"today":              today,
		"windowDays":         contentWindowDays,
		"crew":               members,
		"leadMagnets":        magnets,
		"posts":              posts,
		"postsError":         postsError,
		"postsKnown":         postsKnown,
		"pipelineActiveDays": pipelineActive,
		"volume": content.Volume(content.VolumeInput{
			Crew: crew, Runs: runs, LeadMagnets: magnets, RecentCount: len(posts),
			PostDays: days, PostsKnown: postsKnown, Today: today, Days: contentWindowDays,
		}),
	})
}

func contentLeadMagnetList(c *gin.Context, d *Deps) {
	ws, ok := contentWorkspace(c, d, content.LeadMagnetsWorkspace)
	if !ok {
		return
	}
	all, err := content.LeadMagnets(c.Request.Context(), d.Pool, ws)
	if err != nil {
		contentFail(c, err)
		return
	}
	today := pagekit.ISODay(contentNow())
	filter := content.FilterOf(c.Query("status"))
	c.JSON(http.StatusOK, gin.H{
		"today":   today,
		"weeks":   contentWindowWeeks,
		"filter":  filter,
		"filters": content.Filters,
		"rows":    content.FilterRows(all, filter),
		"volume":  content.MagnetVolume(all, today, contentWindowWeeks),
	})
}

func contentBody(c *gin.Context) ([]byte, bool) {
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body unreadable: " + err.Error()})
		return nil, false
	}
	return body, true
}

func contentWriteErr(c *gin.Context, err error) {
	switch {
	case content.IsValidation(err):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, content.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		contentFail(c, err)
	}
}

func contentLeadMagnetCreate(c *gin.Context, d *Deps) {
	body, ok := contentBody(c)
	if !ok {
		return
	}
	m, err := content.ParseCreate(body, pagekit.ISODay(contentNow()))
	if err != nil {
		contentWriteErr(c, err)
		return
	}
	ws, ok := contentWorkspace(c, d, content.LeadMagnetsWorkspace)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	taken, err := content.TakenIDs(ctx, d.Pool)
	if err != nil {
		contentFail(c, err)
		return
	}
	m.ID = content.UniqueID(m.Name, taken)
	if err := content.InsertLeadMagnet(ctx, d.Pool, ws, m); err != nil {
		contentFail(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"leadMagnet": m})
}

func contentLeadMagnetPatch(c *gin.Context, d *Deps) {
	body, ok := contentBody(c)
	if !ok {
		return
	}
	// A bad body is a 400 whether or not the row exists (FounderOS v1 order).
	if _, err := content.ApplyPatch(content.LeadMagnet{}, body); err != nil {
		contentWriteErr(c, err)
		return
	}
	ws, ok := contentWorkspace(c, d, content.LeadMagnetsWorkspace)
	if !ok {
		return
	}
	ctx := c.Request.Context()
	cur, err := content.LeadMagnetByID(ctx, d.Pool, ws, c.Param("id"))
	if err != nil {
		contentWriteErr(c, err)
		return
	}
	next, err := content.ApplyPatch(cur, body)
	if err == nil {
		err = content.UpdateLeadMagnet(ctx, d.Pool, ws, next)
	}
	if err != nil {
		contentWriteErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"leadMagnet": next})
}

func contentLeadMagnetDelete(c *gin.Context, d *Deps) {
	ws, ok := contentWorkspace(c, d, content.LeadMagnetsWorkspace)
	if !ok {
		return
	}
	id := c.Param("id")
	if err := content.DeleteLeadMagnet(c.Request.Context(), d.Pool, ws, id); err != nil {
		contentWriteErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "id": id})
}
