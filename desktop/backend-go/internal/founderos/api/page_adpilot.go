package api

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/connectors/foreplay"
	"github.com/rhl/businessos-backend/internal/founderos/pages/adpilot"
)

// /adpilot (spec 6.18): the campaign deck and Adscout's ad library, plus
// the five adscout routes from config/founderos/route-map.yaml. Page load
// reads local snapshots only: no model call and no Foreplay credit. Sync,
// mine and add-by-domain spend credits (GETs) and ask/mine may call the
// model, all only on an explicit user action.

func init() { RegisterPage(adpilotRegister) }

// adpilotEnv is everything the adpilot routes touch, injected so tests run
// on a temp store with a fake Foreplay and a fake model.
type adpilotEnv struct {
	Store         adpilot.Store
	CampaignsPath string
	// API returns a Foreplay client, or foreplay.ErrNotConfigured without a
	// key. It never makes a request itself.
	API func() (adpilot.API, error)
	LLM adpilot.LLM
	Now func() time.Time
}

func adpilotEnvFor(d *Deps) *adpilotEnv {
	fp := foreplay.New(d.Resolver)
	return &adpilotEnv{
		Store:         adpilot.DefaultStore(),
		CampaignsPath: adpilot.CampaignDataPath(),
		API: func() (adpilot.API, error) {
			c, err := fp.Client()
			if err != nil {
				return nil, err
			}
			return c, nil
		},
		LLM: adpilot.ClaudeCLI{},
		Now: time.Now,
	}
}

func adpilotRegister(s *gin.RouterGroup, d *Deps) { adpilotRoutes(s, adpilotEnvFor(d)) }

const adpilotNotConfigured = "Foreplay not configured: set FOREPLAY_API_KEY in .env.local"

type adpilotCredits struct {
	Remaining float64 `json:"remaining"`
	Total     float64 `json:"total"`
}

type adpilotLibrary struct {
	Wall       []adpilot.WallAd     `json:"wall"`
	Watchlist  []adpilot.WatchEntry `json:"watchlist"`
	Saved      []adpilot.SavedAd    `json:"saved"`
	Signals    []adpilot.Signal     `json:"signals"`
	Credits    *adpilotCredits      `json:"credits"`
	LastSyncAt *string              `json:"lastSyncAt"`
	// Configured is FOREPLAY_API_KEY presence: without it the library can
	// be read but not refreshed or searched.
	Configured bool `json:"configured"`
}

type adpilotPayload struct {
	Deck    adpilot.Deck      `json:"deck"`
	Library adpilotLibrary    `json:"library"`
	Errors  map[string]string `json:"errors"`
}

func adpilotFail(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"ok": false, "error": msg})
}

// adpilotBody decodes a JSON object body into a key → raw map; nil when the
// body is not a JSON object.
func adpilotBody(c *gin.Context) map[string]json.RawMessage {
	raw, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
	if err != nil {
		return nil
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return nil
	}
	return m
}

// adpilotString reads a string field with a rune-length window; ok=false
// when absent, not a string, or out of bounds.
func adpilotString(m map[string]json.RawMessage, k string, min, max int) (string, bool) {
	v, has := m[k]
	if !has {
		return "", false
	}
	var s string
	if json.Unmarshal(v, &s) != nil {
		return "", false
	}
	if n := len([]rune(s)); n < min || n > max {
		return "", false
	}
	return s, true
}

func adpilotRoutes(s *gin.RouterGroup, env *adpilotEnv) {
	s.GET("/pages/adpilot", func(c *gin.Context) { c.JSON(http.StatusOK, adpilotPage(env)) })

	s.POST("/pages/adscout/ask", func(c *gin.Context) { adpilotAsk(c, env) })
	s.POST("/pages/adscout/mine", func(c *gin.Context) { adpilotMine(c, env) })

	s.GET("/pages/adscout/saved", func(c *gin.Context) {
		saved, err := env.Store.ReadSaved()
		if err != nil {
			adpilotFail(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "saved": saved})
	})
	s.POST("/pages/adscout/saved", func(c *gin.Context) {
		raw, _ := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
		ad, err := adpilot.ParseWallAd(raw)
		if err != nil {
			adpilotFail(c, http.StatusBadRequest, "a full ad snapshot is required")
			return
		}
		saved, err := env.Store.SaveAd(ad, env.Now())
		if err != nil {
			adpilotFail(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "saved": saved})
	})
	s.DELETE("/pages/adscout/saved", func(c *gin.Context) {
		id, ok := adpilotString(adpilotBody(c), "adId", 1, 1<<10)
		if !ok {
			adpilotFail(c, http.StatusBadRequest, "adId required")
			return
		}
		saved, err := env.Store.UnsaveAd(id)
		if err != nil {
			adpilotFail(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "saved": saved})
	})

	s.POST("/pages/adscout/sync", func(c *gin.Context) {
		api, err := env.API()
		if err != nil {
			adpilotFail(c, http.StatusServiceUnavailable, adpilotNotConfigured)
			return
		}
		res, err := adpilot.RunSyncCycle(c.Request.Context(), api, env.Store, env.Now())
		if err != nil {
			adpilotFail(c, http.StatusBadGateway, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "brands": res.Brands, "adsSeen": res.AdsSeen, "signals": res.Signals,
			"digest": res.Digest, "apiCalls": res.APICalls, "remainingCredits": res.RemainingCredits})
	})

	s.GET("/pages/adscout/watchlist", func(c *gin.Context) {
		list, err := env.Store.ReadWatchEntries()
		if err != nil {
			adpilotFail(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "watchlist": list})
	})
	s.POST("/pages/adscout/watchlist", func(c *gin.Context) { adpilotWatchAdd(c, env) })
	s.DELETE("/pages/adscout/watchlist", func(c *gin.Context) {
		id, ok := adpilotString(adpilotBody(c), "brandId", 1, 1<<10)
		if !ok {
			adpilotFail(c, http.StatusBadRequest, "brandId required")
			return
		}
		list, err := env.Store.RemoveWatchEntry(id)
		if err != nil {
			adpilotFail(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "watchlist": list})
	})
}

// adpilotPage is the page payload. Each source fails on its own: an
// unreadable store or campaign file is named in errors, never read as empty.
func adpilotPage(env *adpilotEnv) adpilotPayload {
	p := adpilotPayload{Errors: map[string]string{}}
	cs, synced, err := adpilot.ReadCampaignFile(env.CampaignsPath)
	if err != nil {
		p.Errors["campaigns"] = err.Error()
	}
	p.Deck = adpilot.BuildDeck(cs, synced)

	lib := adpilotLibrary{Wall: []adpilot.WallAd{}, Watchlist: []adpilot.WatchEntry{}, Saved: []adpilot.SavedAd{}, Signals: []adpilot.Signal{}}
	var storeErrs []string
	note := func(err error) {
		if err != nil {
			storeErrs = append(storeErrs, err.Error())
		}
	}
	if w, err := env.Store.Wall(80); err == nil {
		lib.Wall = w
	} else {
		note(err)
	}
	if w, err := env.Store.ReadWatchEntries(); err == nil {
		lib.Watchlist = w
	} else {
		note(err)
	}
	if sv, err := env.Store.ReadSaved(); err == nil {
		lib.Saved = sv
	} else {
		note(err)
	}
	if sg, err := env.Store.ReadSignals(); err == nil {
		if len(sg) > 60 {
			sg = sg[:60]
		}
		lib.Signals = sg
	} else {
		note(err)
	}
	if u, err := env.Store.ReadUsage(); err == nil {
		if u != nil {
			lib.Credits = &adpilotCredits{Remaining: u.RemainingCredits, Total: u.TotalCredits}
		}
	} else {
		note(err)
	}
	if m, err := env.Store.ReadMeta(); err == nil {
		if m.LastSyncAt != nil && *m.LastSyncAt != "" {
			lib.LastSyncAt = m.LastSyncAt
		}
	} else {
		note(err)
	}
	if len(storeErrs) > 0 {
		p.Errors["store"] = strings.Join(storeErrs, "; ")
	}
	_, keyErr := env.API()
	lib.Configured = keyErr == nil
	p.Library = lib
	return p
}

func adpilotAsk(c *gin.Context, env *adpilotEnv) {
	q, ok := adpilotString(adpilotBody(c), "question", 3, 600)
	if !ok {
		adpilotFail(c, http.StatusBadRequest, "a question (3-600 chars) is required")
		return
	}
	wall, err := env.Store.Wall(30)
	if err != nil {
		adpilotFail(c, http.StatusInternalServerError, err.Error())
		return
	}
	sigs, err := env.Store.ReadSignals()
	if err != nil {
		adpilotFail(c, http.StatusInternalServerError, err.Error())
		return
	}
	if len(sigs) > 25 {
		sigs = sigs[:25]
	}
	msgs := make([]string, len(sigs))
	for i, s := range sigs {
		msgs[i] = s.Message
	}
	if len(wall) == 0 && len(msgs) == 0 {
		adpilotFail(c, http.StatusConflict, "The ad store is empty: add brands to the watchlist and sync first.")
		return
	}
	if env.LLM == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "unavailable": true, "error": "Analyst unavailable"})
		return
	}
	answer, err := env.LLM.Complete(c.Request.Context(), adpilot.AskPrompt(q, wall, msgs))
	if errors.Is(err, adpilot.ErrLLMUnavailable) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "unavailable": true, "error": "Analyst unavailable"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"ok": false, "unavailable": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "answer": answer})
}

func adpilotMine(c *gin.Context, env *adpilotEnv) {
	m := adpilotBody(c)
	concept, ok := adpilotString(m, "concept", 3, 300)
	bad := func() { adpilotFail(c, http.StatusBadRequest, "a concept (3-300 chars) is required") }
	if !ok {
		bad()
		return
	}
	opts := adpilot.MineOpts{}
	if v, has := m["minDays"]; has {
		var f float64
		if json.Unmarshal(v, &f) != nil || f != math.Trunc(f) || f < 1 || f > 365 {
			bad()
			return
		}
		opts.MinDays = int(f)
	}
	if _, has := m["format"]; has {
		f, ok := adpilotString(m, "format", 1, 10)
		if !ok || (f != "video" && f != "image") {
			bad()
			return
		}
		opts.Format = f
	}
	api, err := env.API()
	if err != nil {
		adpilotFail(c, http.StatusServiceUnavailable, adpilotNotConfigured)
		return
	}
	probes, by := adpilot.ExpandProbes(c.Request.Context(), env.LLM, concept)
	res := adpilot.MineConcept(c.Request.Context(), api, concept, probes, opts)
	winners := make([]adpilot.WallAd, 0, len(res.Winners))
	for _, w := range res.Winners {
		winners = append(winners, adpilot.ToWallAd(w.Ad, ""))
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "concept": res.Concept, "expandedBy": by, "probes": res.Probes,
		"pooled": res.Pooled, "apiCalls": res.APICalls, "winners": winners})
}

func adpilotWatchAdd(c *gin.Context, env *adpilotEnv) {
	m := adpilotBody(c)
	if id, ok := adpilotString(m, "brandId", 1, 1<<10); ok {
		name, ok := adpilotString(m, "name", 1, 1<<10)
		if !ok {
			adpilotFail(c, http.StatusBadRequest, "pass a domain, or a brandId + name")
			return
		}
		var avatar *string
		if v, has := m["avatar"]; has && json.Unmarshal(v, &avatar) != nil {
			adpilotFail(c, http.StatusBadRequest, "pass a domain, or a brandId + name")
			return
		}
		list, err := env.Store.AddWatchEntry(adpilot.WatchEntry{ID: id, Name: name, Avatar: avatar}, env.Now())
		if err != nil {
			adpilotFail(c, http.StatusInternalServerError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "watchlist": list})
		return
	}
	domain, ok := adpilotString(m, "domain", 3, 120)
	if !ok {
		adpilotFail(c, http.StatusBadRequest, "pass a domain, or a brandId + name")
		return
	}
	api, err := env.API()
	if err != nil {
		adpilotFail(c, http.StatusServiceUnavailable, adpilotNotConfigured)
		return
	}
	entry, err := adpilot.AddWatchDomain(c.Request.Context(), api, env.Store, domain, env.Now())
	if err != nil {
		adpilotFail(c, http.StatusBadGateway, err.Error())
		return
	}
	if entry == nil {
		adpilotFail(c, http.StatusNotFound, "no Foreplay brand found for "+domain)
		return
	}
	list, err := env.Store.ReadWatchEntries()
	if err != nil {
		adpilotFail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "added": entry, "watchlist": list})
}
