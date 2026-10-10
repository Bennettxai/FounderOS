package api

// /os/integrations, the Connections board (spec 6.10). Ports FounderOS v1
// app/integrations/page.tsx's data, GET/POST/DELETE /api/connections/connect
// and POST /api/admin/keys/test.
//
// Keys are written ONLY to the bridge's own env.local (d.Resolver.EnvLocal,
// ~/.founderos-bridge/env.local), only under names the catalog or the key slots
// declare, and no value is ever sent back. That is local configuration, not an
// outbound side effect, so FOUNDEROS_WRITES does not gate it.

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/pages/integrations"
)

func init() { RegisterPage(registerIntegrations) }

func registerIntegrations(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/connections", integrationsBoard(d))
	s.POST("/pages/connections/connect", integrationsConnect(d))
	s.DELETE("/pages/connections/connect", integrationsDisconnect(d))
	s.POST("/pages/admin/keys/test", integrationsTestKey(d))
}

const integrationsMaxKeyLen = 4096 // a base64 RSA private key is ~2.2KB

func integrationsBoard(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
		defer cancel()
		statuses := d.Board.Statuses(ctx)
		// What the resolver would use: planted keys over env.local.
		saved := integrations.ReadEnv(d.Resolver.EnvLocal)
		if d.Resolver.Planted != "" {
			for k, v := range integrations.ReadEnv(d.Resolver.Planted) {
				saved[k] = v
			}
		}
		catalog := integrations.ConnectionCatalog(statuses, saved)
		resolve := func(name string) string { return d.Resolver.Resolve(name) }
		now := time.Now().UnixMilli()
		oauth := map[string]*integrations.OAuthReadiness{}
		for _, e := range catalog {
			if r := integrations.Readiness(e.Slug, resolve, now); r != nil {
				oauth[e.Slug] = r
			}
		}
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, gin.H{
			"connections": statuses,
			"catalog":     catalog,
			"categories":  integrations.ByCategory(integrations.Catalog),
			"volume":      integrations.Volume(statuses, catalog),
			"keys":        integrations.SlotStatuses(resolve),
			"oauth":       oauth,
		})
	}
}

type integrationsConnectBody struct {
	Slug   string            `json:"slug"`
	Values map[string]string `json:"values"`
	Slot   string            `json:"slot"`
	Value  *string           `json:"value"`
}

func integrationsBad(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": msg})
}

func integrationsSafeValue(v string) bool {
	return strings.TrimSpace(v) != "" && !strings.ContainsAny(v, "\r\n") && len(v) <= integrationsMaxKeyLen
}

func integrationsReadBody(c *gin.Context, out any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)).Decode(out); err != nil {
		integrationsBad(c, "bad body")
		return false
	}
	return true
}

// integrationsEnvLocal is where the Connections page saves keys: the planted
// file the resolver reads first (so a saved key actually takes effect), else
// env.local. The writers refuse symlinks, so the symlinked dev env.local is
// never written through (security review #4).
func integrationsEnvLocal(c *gin.Context, d *Deps) (string, bool) {
	if d.Resolver.Planted != "" {
		return d.Resolver.Planted, true
	}
	if d.Resolver.EnvLocal == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "error": "no ~/.founderos/.env is configured (FOUNDEROS_ENV_LOCAL)"})
		return "", false
	}
	return d.Resolver.EnvLocal, true
}

// connectKeys saves a tile's connect keys ({slug, values}) or one API-key
// slot ({slot, value}).
func integrationsConnect(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var b integrationsConnectBody
		if !integrationsReadBody(c, &b) {
			return
		}
		path, ok := integrationsEnvLocal(c, d)
		if !ok {
			return
		}
		if b.Slot != "" {
			slot, known := integrations.SlotFor(b.Slot)
			if !known {
				integrationsBad(c, "unknown key slot: "+b.Slot)
				return
			}
			if b.Value == nil || !integrationsSafeValue(*b.Value) {
				integrationsBad(c, "unsafe value")
				return
			}
			if err := integrations.UpsertEnv(path, map[string]string{slot.EnvVar: strings.TrimSpace(*b.Value)}); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "could not write ~/.founderos/.env"})
				return
			}
			c.Header("Cache-Control", "no-store")
			c.JSON(http.StatusOK, gin.H{"ok": true, "envVar": slot.EnvVar})
			return
		}

		entry, known := integrations.Find(b.Slug)
		if !known {
			integrationsBad(c, "unknown integration")
			return
		}
		keys := integrations.ConnectKeysFor(entry)
		if len(keys) == 0 {
			integrationsBad(c, entry.Name+" does not connect with a pasted key")
			return
		}
		allowed := map[string]bool{}
		for _, k := range keys {
			allowed[k] = true
		}
		if len(b.Values) == 0 {
			integrationsBad(c, "unexpected key name")
			return
		}
		values := map[string]string{}
		for k, v := range b.Values {
			if !allowed[k] {
				integrationsBad(c, "unexpected key name")
				return
			}
			if !integrationsSafeValue(v) {
				integrationsBad(c, "unsafe value")
				return
			}
			values[k] = strings.TrimSpace(v)
		}
		if err := integrations.UpsertEnv(path, values); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "could not write ~/.founderos/.env"})
			return
		}
		saved := integrations.ReadEnv(path)
		keySaved := true
		for _, k := range keys {
			if saved[k] == "" {
				keySaved = false
			}
		}
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, gin.H{"ok": true, "keySaved": keySaved, "partial": !keySaved})
	}
}

// integrationsDisconnect makes the key stop resolving, not just leave one
// file: it is removed from planted.env and env.local (never through a symlink)
// and from the process env (loadPlanted copied planted keys there at boot).
// A key that still resolves afterwards, e.g. from the symlinked dev
// env.local, is reported as such (409) instead of a fake disconnect.
func integrationsDisconnect(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var b integrationsConnectBody
		if !integrationsReadBody(c, &b) {
			return
		}
		entry, known := integrations.Find(b.Slug)
		if !known {
			integrationsBad(c, "unknown integration")
			return
		}
		if d.Resolver.Planted == "" && d.Resolver.EnvLocal == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"ok": false, "error": "no ~/.founderos/.env is configured (FOUNDEROS_ENV_LOCAL)"})
			return
		}
		keys := integrations.ConnectKeysFor(entry)
		for _, path := range []string{d.Resolver.Planted, d.Resolver.EnvLocal} {
			if path == "" || integrationsIsSymlink(path) {
				continue
			}
			if _, err := os.Stat(path); err != nil {
				continue // nothing to remove; do not create the file
			}
			if err := integrations.RemoveEnv(path, keys); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": "could not write ~/.founderos/.env"})
				return
			}
		}
		for _, k := range keys {
			_ = os.Unsetenv(k)
		}
		var still, from []string
		for _, k := range keys {
			if d.Resolver.Resolve(k) == "" {
				continue
			}
			still = append(still, k)
			from = append(from, k+" from "+integrationsKeySource(d, k))
		}
		c.Header("Cache-Control", "no-store")
		if len(still) > 0 {
			c.JSON(http.StatusConflict, gin.H{
				"ok": false, "keySaved": true, "stillResolves": still,
				"error": "still resolves: " + strings.Join(from, ", ") + "; remove it there",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "keySaved": false})
	}
}

func integrationsIsSymlink(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

// integrationsKeySource names where a key resolves from, in Resolve's order.
func integrationsKeySource(d *Deps, name string) string {
	for _, f := range []struct{ path, label string }{{d.Resolver.Planted, "planted keys"}, {d.Resolver.EnvLocal, "~/.founderos/.env"}} {
		if f.path == "" || integrations.ReadEnv(f.path)[name] == "" {
			continue
		}
		if integrationsIsSymlink(f.path) {
			return f.label + " (a symlink FounderOS will not write through)"
		}
		return f.label
	}
	return "the process env"
}

// testKey proves a key still works: it runs the slot's connector check (that
// one connector only) and reports how long the far end took. Nothing about
// the secret comes back.
func integrationsTestKey(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var b struct {
			EnvVar string `json:"envVar"`
		}
		if !integrationsReadBody(c, &b) {
			return
		}
		slot, known := integrations.SlotFor(b.EnvVar)
		switch {
		case !known:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown key slot: " + b.EnvVar})
			return
		case slot.ConnectorID == "":
			c.JSON(http.StatusBadRequest, gin.H{"error": slot.EnvVar + " has no live connector to test"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
		defer cancel()
		started := time.Now()
		st, found := d.Board.StatusOf(ctx, slot.ConnectorID)
		ms := time.Since(started).Milliseconds()
		if !found {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown connector: " + slot.ConnectorID})
			return
		}
		c.Header("Cache-Control", "no-store")
		c.JSON(http.StatusOK, gin.H{"ok": st.State == "connected", "state": st.State, "detail": st.Detail, "ms": ms})
	}
}
