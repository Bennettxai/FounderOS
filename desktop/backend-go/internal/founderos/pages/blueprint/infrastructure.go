package blueprint

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// The machines and the long-running services of the FounderOS v2 demo stack:
// the SvelteKit frontend, the Go backend, Postgres, Redis and the bundled
// Optimal Engine. FounderOS v1 hand-listed a workstation and a production host
// (lib/blueprint/compile.ts HOSTS); v2 keeps that shape and adds the services
// the stack is made of. Everything here is generic: no private hosts, no
// addresses, no credentials.
//
// Honesty rule: nothing here claims 'live' from a hand-written line. A
// service is live only when its port answers on the host doing the compile.
// Ports come from the backend's own environment (FRONTEND_URL, SERVER_PORT,
// DATABASE_URL, REDIS_URL, OPTIMAL_ENGINE_URL), defaulting to the stock ones.

const (
	HostWorkstation = "host-workstation"
	HostServer      = "host-server"
)

// infraNode is a Node plus where it runs and how to verify it there.
type infraNode struct {
	Node
	host  string // host id; empty for managed cloud services
	probe int    // loopback port probed when the compile runs on host
	app   string // /Applications bundle checked when the compile runs on host
}

func svc(id, name, host, blurb string, f Facts, icon string, probe int) infraNode {
	if probe != 0 {
		f.Set("port", strconv.Itoa(probe))
	}
	return infraNode{Node: Node{ID: id, Kind: KindService, Name: name, Layer: 4, Status: StatusConfigured, Blurb: blurb, Facts: f, Icon: icon}, host: host, probe: probe}
}

func infraHosts() []Node {
	return []Node{
		{ID: HostWorkstation, Kind: KindHost, Name: "Workstation", Layer: 4, Status: StatusLive,
			Blurb: "Where the OS runs and is built: scripts/dev-local.sh starts the Go backend, the SvelteKit frontend and the bundled Optimal Engine here, beside Postgres and Redis.",
			Facts: facts("start", "scripts/dev-local.sh", "role", "demo + development"), Icon: "laptop"},
		{ID: HostServer, Kind: KindHost, Name: "Production host", Layer: 4, Status: StatusDesigned,
			Blurb: "docker-compose.production.yml runs the same stack as containers on a dedicated host. Not deployed from this demo, so nothing is probed there.",
			Facts: facts("compose", "docker-compose.production.yml"), Icon: "server"},
	}
}

// portOf reads the port from a URL-ish env value, else the fallback. Only the
// port is kept: a database URL's credentials never reach the graph.
func portOf(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	if n, err := strconv.Atoi(raw); err == nil && n > 0 {
		return n
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fallback
	}
	if n, err := strconv.Atoi(u.Port()); err == nil && n > 0 {
		return n
	}
	return fallback
}

func infraNodes(env func(string) string) []infraNode {
	const WS = HostWorkstation
	return []infraNode{
		svc("service-frontend", "SvelteKit frontend", WS, "The /os pages in the Monolith Signal theme; talks to the backend over /api/founderos.",
			facts("runtime", "SvelteKit 2 · Svelte 5"), "layout-dashboard", portOf(env("FRONTEND_URL"), 5173)),
		svc("service-backend", "Go backend", WS, "Serves /api/founderos: the pages, the Connections board, the agent runtime and the cron scheduler.",
			facts("runtime", "Go · gin"), "activity", portOf(env("SERVER_PORT"), 8001)),
		svc("service-postgres", "Postgres", WS, "Postgres 16 + pgvector: the founderos_* tables, loaded from the seeded demo data and scoped by workspace.",
			facts("tables", "founderos_*"), "database", portOf(env("DATABASE_URL"), 5432)),
		svc("service-redis", "Redis", WS, "Sessions and queues for the backend.",
			facts(), "database", portOf(env("REDIS_URL"), 6379)),
		svc("service-optimal-engine", "Optimal Engine", WS, "The bundled Elixir memory engine: every workspace's governed memory, claims and chunks, per the engine topology.",
			facts("runtime", "Elixir", "topology", "config/founderos/engine-topology.yaml"), "brain", portOf(env("OPTIMAL_ENGINE_URL"), 4200)),
	}
}

func infraEdges() []Edge {
	return []Edge{
		{From: "service-frontend", To: "service-backend", Kind: "uses", Via: "/api/founderos"},
		{From: "service-backend", To: "service-postgres", Kind: "reads", Via: "founderos_*"},
		{From: "service-backend", To: "service-redis", Kind: "uses", Via: "sessions"},
		{From: "service-backend", To: "store-brain", Kind: "reads", Via: "memory"},
		{From: "store-brain", To: "service-optimal-engine", Kind: "reads", Via: "every workspace"},
	}
}

// CurrentHostID says which machine is compiling: FOUNDEROS_OS_HOST_ID, else
// the workstation, because the compiling backend is the one serving the demo.
func CurrentHostID(env func(string) string) string {
	if o := strings.TrimSpace(env("FOUNDEROS_OS_HOST_ID")); o != "" {
		return o
	}
	return HostWorkstation
}

// ProbeLoopback reports whether a loopback port accepts a connection.
func ProbeLoopback(port int) bool {
	c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = c.Close()
	return true
}

// AppExists checks /Applications for a bundle.
func AppExists(bundle string) bool {
	_, err := os.Stat(filepath.Join("/Applications", bundle))
	return err == nil
}

// resolveInfrastructure turns the inventory into graph nodes and edges,
// probing only the compiling host.
func resolveInfrastructure(o Options) ([]Node, []Edge) {
	inv := infraNodes(o.Env)
	nodes := make([]Node, len(inv))
	for i, in := range inv {
		n := in.Node
		n.Facts = in.Facts.Clone()
		switch {
		case in.host != "" && in.host == o.HostID:
			if in.probe != 0 {
				if o.Probe != nil && o.Probe(in.probe) {
					n.Status = StatusLive
					n.Facts.Set("probe", fmt.Sprintf(":%d answered on this host", in.probe))
				} else {
					n.Status = StatusConfigured
					n.Facts.Set("probe", fmt.Sprintf(":%d did not answer on this host", in.probe))
				}
			} else if in.app != "" {
				if o.AppExists != nil && o.AppExists(in.app) {
					n.Status = StatusLive
				} else {
					n.Status = StatusNotConfigured
				}
			}
		case in.host != "":
			n.Facts.Set("verified", "not from this host")
		}
		nodes[i] = n
	}
	edges := infraEdges()
	for _, in := range inv {
		if in.host != "" {
			edges = append(edges, Edge{From: in.ID, To: in.host, Kind: "runs-on"})
		}
	}
	return nodes, edges
}
