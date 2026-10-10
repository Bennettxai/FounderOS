package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	bd "github.com/rhl/businessos-backend/internal/founderos/connectors/branddeals"
	pbd "github.com/rhl/businessos-backend/internal/founderos/pages/branddeals"
)

// /brand-deals (spec 6.12): FounderOS v1 GET /api/brand-deals, served at the
// route-map bridge path. Notion stays the source of truth and the page stays
// read-only: this route only reads the Brand Deals Hub through the Notion
// connector (20-minute cache, stale fallback) and adds the slab's view.

// brandDealsFetcher is the connector's read, swapped for a fake in tests.
type brandDealsFetcher interface {
	FetchDeals(ctx context.Context) bd.Result
}

var (
	brandDealsSource = func(d *Deps) brandDealsFetcher { return depsBrandDeals(d) }
	brandDealsNow    = time.Now
)

// brandDealsPayload is the source route's body (deals, mode, detail,
// syncedAt) plus the precomputed view. View is null when Notion could not be
// read: an unreachable hub has no figures, never zeroes.
type brandDealsPayload struct {
	bd.Result
	View *pbd.View `json:"view"`
}

func init() { RegisterPage(brandDealsRoutes) }

func brandDealsRoutes(s *gin.RouterGroup, d *Deps) {
	// The shared per-Deps connector, so its cache outlives a request and the
	// analytics heartbeat's warm-up lands in it.
	src := brandDealsSource(d)
	s.GET("/pages/brand-deals", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
		defer cancel()
		c.JSON(http.StatusOK, brandDealsBody(src.FetchDeals(ctx), brandDealsNow()))
	})
}

func brandDealsBody(r bd.Result, now time.Time) brandDealsPayload {
	if r.Deals == nil {
		r.Deals = []bd.Deal{}
	}
	out := brandDealsPayload{Result: r}
	if r.Mode != bd.ModeError {
		v := pbd.BuildView(r.Deals, now)
		out.View = &v
	}
	return out
}
