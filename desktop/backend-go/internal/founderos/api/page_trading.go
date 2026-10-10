package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/phantom"
	"github.com/rhl/businessos-backend/internal/founderos/pages/trading"
)

// /os/trading (spec 6.9): GET /api/founderos/pages/trading is FounderOS v1
// tradingPayload() — the same payload its server page and GET /api/trading
// served. The /api/trading/* pushes (snapshot, orders, analysis, activity,
// limits) are compat routes owned by the compat listener; this page only
// reads what they wrote to founderos_trading_* (personal workspace).
//
// Deps.Robinhood (the in-memory feed store behind the Connections board) is
// deliberately not merged: Postgres is where the compat feed lands, and a
// second source would double-count accounts.
func init() { RegisterPage(tradingRoutes) }

type tradingSources struct {
	store  func() trading.Store
	wallet trading.WalletFunc
	now    func() time.Time
}

func tradingRoutes(s *gin.RouterGroup, d *Deps) {
	st := trading.NewPgStore(d.Pool)
	tradingMount(s, tradingSources{
		store:  func() trading.Store { return st },
		wallet: trading.PhantomWallet(phantom.New(d.Resolver)),
		now:    time.Now,
	})
}

func tradingMount(s *gin.RouterGroup, src tradingSources) {
	s.GET("/pages/trading", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
		defer cancel()
		wallet := func(ctx context.Context) trading.WalletRead {
			wctx, wcancel := context.WithTimeout(ctx, 10*time.Second)
			defer wcancel()
			return src.wallet(wctx)
		}
		p, err := trading.Build(ctx, src.store(), wallet, src.now())
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "trading store unreadable: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, p)
	})
}
