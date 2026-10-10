// Package api mounts the FounderOS surface on the BusinessOS backend under
// /api/founderos: the Connections board, the bridge guard state, and the device
// collector's push endpoint. Pages and the FounderOS-compatible routes (spec
// §6) hang off the same Deps.
package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rhl/businessos-backend/internal/founderos/agents"
	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/arcads"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/beehiiv"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/docusign"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/fathomcalls"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/gcal"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/loom"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/manychat"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/metaads"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/miro"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/optimalengine"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/payments"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/plaud"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/robinhood"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/slack"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/trakyo"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/typeform"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/vidalytics"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/zernio"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

// Deps is everything the FounderOS routes need, built once at server start.
type Deps struct {
	Pool      *pgxpool.Pool
	Resolver  connectors.Resolver
	Board     *connectors.Registry
	Devices   *devicepush.Receiver
	Robinhood robinhood.Store
	Agents    *agents.Runtime
	Conductor *ConductorDeps // nil: built from Resolver on demand
	// Scheduler fires the operator's crons (guard-gated); the compat API's
	// /api/cron/tick drives the same one.
	Scheduler *agents.Scheduler
}

// Expected devices on the board: the MacBook and the mini collectors.
var expectedDevices = []string{"alexs-macbook-pro", "mini"}

// NewDeps wires the production dependencies.
func NewDeps(pool *pgxpool.Pool) *Deps {
	res := connectors.DefaultResolver()
	d := &Deps{
		Pool:      pool,
		Resolver:  res,
		Devices:   devicepush.NewReceiver(devicepush.NewMemStore(), expectedDevices...),
		Robinhood: robinhood.NewMemStore(),
	}
	// Device pushes persist in Postgres (FounderOS) so a restart keeps them.
	if pool != nil {
		var ws string
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = 'founderos'`).Scan(&ws); err == nil {
			d.Devices = devicepush.NewReceiver(devicepush.NewPgStore(pool, ws), expectedDevices...)
		}
		cancel()
	}
	// The device this bridge runs on: machine-local checks (the stack) read
	// its push only (devicepush.HostConnector).
	d.Devices.Host = hostDevice()
	d.Board = NewBoard(d)
	// Agent runs and crons belong to the FounderOS workspace (table map).
	// Without it (bootstrap not run yet) the agent routes answer 503.
	if pool != nil {
		var ws string
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := pool.QueryRow(ctx, `SELECT id::text FROM workspaces WHERE slug = 'founderos'`).Scan(&ws); err == nil {
			d.Agents = agents.New(agents.NewPgStore(pool, ws), AllAgents(d)...)
			// Fires nothing until FOUNDEROS_CRONS=1 (guard.RunCron).
			d.Scheduler = agents.NewScheduler(d.Agents, agents.NewPgCronSource(pool, ws))
			d.Scheduler.Start(context.Background())
		}
	}
	return d
}

type boardEntry struct {
	meta connectors.Meta
	make func(d *Deps) connectors.Connector
}

// board is FounderOS v1's Connections board (lib/connectors/index.ts CHECKS),
// in its order, with G-Brain replaced by the Optimal Engine row.
func board() []boardEntry {
	dev := func(source string) boardEntry {
		return boardEntry{devicepush.Metas[source], func(d *Deps) connectors.Connector { return d.Devices.Connector(devicepush.Metas[source].ID) }}
	}
	r := func(m connectors.Meta, f func(connectors.Resolver) connectors.Connector) boardEntry {
		return boardEntry{m, func(d *Deps) connectors.Connector { return f(d.Resolver) }}
	}
	return []boardEntry{
		{optimalengine.Meta, func(*Deps) connectors.Connector { return optimalengine.New(TopologyEngines()) }},
		r(paperclip.Meta, func(res connectors.Resolver) connectors.Connector { return paperclip.New(res) }),
		dev(devicepush.SourceWhatsApp),
		r(zernio.Meta, func(res connectors.Resolver) connectors.Connector { return zernio.New(res) }),
		r(beehiiv.Meta, func(res connectors.Resolver) connectors.Connector { return beehiiv.New(res) }),
		r(manychat.Meta, func(res connectors.Resolver) connectors.Connector { return manychat.New(res) }),
		r(trakyo.Meta, func(res connectors.Resolver) connectors.Connector { return trakyo.New(res) }),
		r(typeform.Meta, func(res connectors.Resolver) connectors.Connector { return typeform.New(res) }),
		r(fathomcalls.Meta, func(res connectors.Resolver) connectors.Connector { return fathomcalls.New(res) }),
		r(plaud.Meta, func(res connectors.Resolver) connectors.Connector { return plaud.New(res) }),
		r(docusign.Meta, func(res connectors.Resolver) connectors.Connector { return docusign.New(res) }),
		r(loom.Meta, func(res connectors.Resolver) connectors.Connector { return loom.New(res) }),
		r(vidalytics.Meta, func(res connectors.Resolver) connectors.Connector { return vidalytics.New(res) }),
		r(metaads.Meta, func(res connectors.Resolver) connectors.Connector { return metaads.New(res) }),
		r(arcads.Meta, func(res connectors.Resolver) connectors.Connector { return arcads.New(res) }),
		dev(devicepush.SourceWispr),
		dev(devicepush.SourceLocalStack),
		dev(devicepush.SourceObsidian),
		r(miro.Meta, func(res connectors.Resolver) connectors.Connector { return miro.New(res) }),
		r(email.Meta, func(res connectors.Resolver) connectors.Connector { return email.New(res) }),
		r(gcal.Meta, func(res connectors.Resolver) connectors.Connector { return gcal.New(res) }),
		r(slack.Meta, func(res connectors.Resolver) connectors.Connector { return slack.New(res) }),
		r(payments.Meta, func(res connectors.Resolver) connectors.Connector { return payments.New(res) }),
		{robinhood.Meta, func(d *Deps) connectors.Connector {
			c := robinhood.New(d.Resolver)
			c.Store = d.Robinhood
			return c
		}},
	}
}

// BoardIDs lists the board's connector ids without checking any of them.
func BoardIDs() []string {
	var ids []string
	for _, e := range board() {
		ids = append(ids, e.meta.ID)
	}
	return ids
}

// NewBoard builds the Connections board registry.
func NewBoard(d *Deps) *connectors.Registry {
	reg := connectors.NewRegistry()
	for _, e := range board() {
		reg.Register(e.meta, e.make(d))
	}
	return reg
}

var stagingPorts = map[string]int{"macbook": 4210, "hub": 4211, "mini": 4212}

// TopologyEngines resolves the engines workspaces are routed to: each
// engine's url_env/key_env, else the local staging engine and its key file.
// Engines with no endpoint (e.g. the mini before it is staged) are skipped.
func TopologyEngines() []optimalengine.Engine {
	topo, err := topology.LoadRepo()
	if err != nil {
		return nil
	}
	home, _ := os.UserHomeDir()
	used := map[string]bool{}
	for _, w := range topo.Workspaces {
		used[w.Home] = true
	}
	var out []optimalengine.Engine
	for name, e := range topo.Engines {
		if e.Deferred || !used[name] {
			continue
		}
		url, key := os.Getenv(e.URLEnv), os.Getenv(e.KeyEnv)
		keyFile := filepath.Join(home, ".founderos-bridge", "keys", "oe-"+name+".key")
		if url == "" {
			if _, err := os.Stat(keyFile); err != nil {
				continue // not staged on this machine
			}
			url = fmt.Sprintf("http://127.0.0.1:%d", stagingPorts[name])
		}
		if key == "" {
			raw, _ := os.ReadFile(keyFile)
			key = strings.TrimSpace(string(raw))
		}
		out = append(out, optimalengine.Engine{Name: name, URL: url, Key: key})
	}
	return out
}

// Register mounts /api/founderos/*. auth is BusinessOS's session middleware
// chain (session + RequireAuth).
func Register(api *gin.RouterGroup, d *Deps, auth ...gin.HandlerFunc) {
	b := api.Group("/founderos")

	// The device collector has no browser session; it presents its own token.
	b.POST("/device/push", devicePush(d))

	s := b.Group("")
	s.Use(auth...)
	s.GET("/connections", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
		defer cancel()
		c.JSON(http.StatusOK, gin.H{"connections": d.Board.Statuses(ctx)})
	})
	registerAgents(s, d)
	mountPages(s, d)
	s.GET("/guard", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"writes": guard.WritesEnabled(), "crons": guard.CronsEnabled(), "refused": guard.Refused()})
	})
}

func devicePush(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		want := os.Getenv("FOUNDEROS_DEVICE_TOKEN")
		if want == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "device push is closed: FOUNDEROS_DEVICE_TOKEN is not set"})
			return
		}
		got := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "bad device token"})
			return
		}
		var p devicepush.Payload
		if err := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 8<<20)).Decode(&p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "payload is not JSON: " + err.Error()})
			return
		}
		if d.Devices == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no device receiver"})
			return
		}
		if err := d.Devices.Accept(c.Request.Context(), p); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// machineRoutes authenticate with a bearer token, not a browser session, so
// double-submit CSRF does not apply (and would block every call).
var machineRoutes = map[string]bool{
	"/api/founderos/device/push":    true,
	"/api/v1/founderos/device/push": true, // RegisterRoutes also mounts under /api/v1
}

// CSRFExempt reports whether a request path is a FounderOS machine route.
func CSRFExempt(path string) bool { return machineRoutes[path] }

// BodyLimit is the request body cap for the operator routes that need more than the
// server's global 10MB (bank/card statement uploads); 0 means the default.
func BodyLimit(path string) int64 {
	p := strings.TrimPrefix(strings.TrimPrefix(path, "/api/v1"), "/api")
	switch p {
	case "/founderos/pages/finances/statements", "/founderos/pages/finances/bank-statement":
		return 25 << 20
	}
	return 0
}

// hostDevice is this machine's device id as founderos-collector pushes it.
func hostDevice() string {
	h, _ := os.Hostname()
	return hostDeviceFrom(connectors.DefaultResolver(), h)
}

// hostDeviceFrom is FOUNDEROS_HOST_DEVICE (process env, then the bridge's env
// files), else FOUNDEROS_COLLECTOR_DEVICE resolved exactly as this box's
// collector resolves it, else the hostname slug the collector defaults to.
func hostDeviceFrom(res connectors.Resolver, hostname string) string {
	if v := strings.TrimSpace(os.Getenv("FOUNDEROS_HOST_DEVICE")); v != "" {
		return v
	}
	for _, k := range []string{"FOUNDEROS_HOST_DEVICE", "FOUNDEROS_COLLECTOR_DEVICE"} {
		if v := strings.TrimSpace(res.Resolve(k)); v != "" {
			return v
		}
	}
	return devicepush.HostDeviceSlug(hostname)
}
