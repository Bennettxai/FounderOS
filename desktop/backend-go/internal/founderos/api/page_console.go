package api

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/rhl/businessos-backend/internal/founderos/guard"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/rhl/businessos-backend/internal/founderos/connectors"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/devicepush"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/email"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/paperclip"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/payments"
	"github.com/rhl/businessos-backend/internal/founderos/connectors/stripe"
	"github.com/rhl/businessos-backend/internal/founderos/memory"
	"github.com/rhl/businessos-backend/internal/founderos/pages/console"
	"github.com/rhl/businessos-backend/internal/founderos/topology"
)

// The operator console (spec 6.1, FounderOS v1 app/page.tsx):
//
//	GET  /pages/console    the composed console (pulse row, volume, activity,
//	                       inbound mix, attention, done today, brain core)
//	POST /pages/interject  the home composer (FounderOS v1 /api/interject)
func init() { RegisterPage(registerConsole) }

// consoleSources are the console's live readings; tests swap them.
type consoleSources struct {
	statuses  func(context.Context) []connectors.Status
	brain     func(context.Context) console.Brain
	roster    func(context.Context, time.Time) (*console.Roster, []console.Run, []console.Run, error)
	feed      func(context.Context) ([]console.Item, []console.SourceState)
	charges   func(context.Context) ([]console.Charge, error)
	interject console.InterjectDeps
}

var consoleSourcesFor = consoleDefaultSources

func registerConsole(s *gin.RouterGroup, d *Deps) {
	s.GET("/pages/console", func(c *gin.Context) {
		src := consoleSourcesFor(d)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 14*time.Second)
		defer cancel()
		now := time.Now()
		in := console.Input{Now: now}
		var wg sync.WaitGroup
		wg.Add(5)
		go func() { defer wg.Done(); in.Connections = src.statuses(ctx) }()
		go func() { defer wg.Done(); in.Brain = src.brain(ctx) }()
		go func() {
			defer wg.Done()
			r, recent, window, err := src.roster(ctx, now)
			if err != nil {
				in.RosterErr = err.Error()
				return
			}
			in.Roster, in.Recent, in.Window = r, recent, window
		}()
		go func() { defer wg.Done(); in.Feed, in.Sources = src.feed(ctx) }()
		go func() {
			defer wg.Done()
			ch, err := src.charges(ctx)
			if errors.Is(err, stripe.ErrNotConfigured) {
				in.ChargesNotConfigured = true
				return
			}
			if err != nil {
				in.ChargesErr = err.Error()
				return
			}
			in.Charges = ch
		}()
		wg.Wait()
		c.JSON(http.StatusOK, console.Build(in))
	})

	s.POST("/pages/interject", func(c *gin.Context) {
		var body struct {
			Text  string `json:"text"`
			Route string `json:"route"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10)).Decode(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "invalid JSON"})
			return
		}
		text := strings.TrimSpace(body.Text)
		if text == "" || len([]rune(text)) > 10_000 {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "text must be 1 to 10000 characters"})
			return
		}
		switch body.Route {
		case "", "task", "agent", "note":
		default:
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "route must be task, agent or note"})
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
		defer cancel()
		rc := console.PerformInterject(ctx, text, body.Route, consoleSourcesFor(d).interject, time.Now())
		// A landed interject is a 200; one that could not land is a 502 so the
		// composer shows the failure instead of a fake receipt.
		code := http.StatusOK
		if !rc.OK {
			code = http.StatusBadGateway
			if strings.Contains(rc.Error, guard.ErrWritesDisabled.Error()) {
				code = http.StatusConflict // refused by the bridge guard, not a failure upstream
			}
		}
		c.JSON(code, rc)
	})
}

// ---- production sources -------------------------------------------------------

func consoleEngineRefs() ([]console.EngineRef, *topology.Topology) {
	topo, err := topology.LoadRepo()
	if err != nil {
		return nil, nil
	}
	homes := map[string][]string{}
	for _, w := range topo.Workspaces {
		homes[w.Home] = append(homes[w.Home], w.Slug)
	}
	endpoints := map[string]console.EngineRef{}
	for _, e := range TopologyEngines() {
		endpoints[e.Name] = console.EngineRef{Name: e.Name, URL: e.URL, Key: e.Key}
	}
	var names []string
	for name, e := range topo.Engines {
		if !e.Deferred && len(homes[name]) > 0 {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	refs := make([]console.EngineRef, 0, len(names))
	for _, name := range names {
		ref := endpoints[name]
		ref.Name, ref.Homes = name, homes[name]
		refs = append(refs, ref)
	}
	return refs, topo
}

// consoleInterject lands tasks on the Paperclip board (a guarded write:
// refused while FOUNDEROS_WRITES=0) and notes in the Optimal Engine.
type consoleInterject struct{ d *Deps }

func (ci consoleInterject) CreateTask(ctx context.Context, title, description string) (string, string, error) {
	issue, err := paperclip.New(ci.d.Resolver).CreateIssue(ctx, title, description)
	if err != nil {
		return "", "", err
	}
	if issue == nil {
		return "", "", errors.New("the board returned no issue")
	}
	return issue.Identifier, "", nil
}

func (ci consoleInterject) CaptureNote(ctx context.Context, n console.NoteInput) (string, error) {
	refs, topo := consoleEngineRefs()
	if topo == nil {
		return "", errors.New("engine topology unreadable")
	}
	eps := map[string]memory.Endpoint{}
	for _, r := range refs {
		if r.URL != "" {
			eps[r.Name] = memory.Endpoint{URL: r.URL, Key: r.Key}
		}
	}
	// Interjected notes are general business knowledge: the brain-store
	// default workspace (founderos), the same place GBrain's inbox went.
	return memory.New(topo, eps).Capture(ctx, memory.Capture{Workspace: topo.BrainStore.Default, Title: n.Title, Text: n.Text, Genre: "note"})
}

func consoleDefaultSources(d *Deps) consoleSources {
	return consoleSources{
		statuses: func(ctx context.Context) []connectors.Status {
			if d.Board == nil {
				return nil
			}
			return d.Board.Statuses(ctx)
		},
		brain: func(ctx context.Context) console.Brain {
			refs, _ := consoleEngineRefs()
			return console.ProbeEngines(ctx, refs)
		},
		roster: func(ctx context.Context, now time.Time) (*console.Roster, []console.Run, []console.Run, error) {
			return console.ReadRoster(ctx, d.Pool, now)
		},
		feed:      func(ctx context.Context) ([]console.Item, []console.SourceState) { return consoleFeed(ctx, d) },
		charges:   func(ctx context.Context) ([]console.Charge, error) { return consoleCharges(ctx, d) },
		interject: consoleInterject{d},
	}
}

func consoleCharges(ctx context.Context, d *Deps) ([]console.Charge, error) {
	snap, err := payments.New(d.Resolver).StripeSnapshot(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]console.Charge, 0, len(snap.RecentCharges))
	for _, c := range snap.RecentCharges {
		out = append(out, console.Charge{Amount: c.Amount, Currency: c.Currency, Description: c.Description, Created: c.Created})
	}
	return out, nil
}

// consoleFeed is gatherCommsFeed: WhatsApp (pushed), email, Slack, each
// lane reporting whether it answered. A failed lane is surfaced, never
// silently read as "no messages".
func consoleFeed(ctx context.Context, d *Deps) ([]console.Item, []console.SourceState) {
	lanes := []string{"whatsapp", "email", "slack"}
	items := make([][]console.Item, len(lanes))
	states := make([]console.SourceState, len(lanes))
	var wg sync.WaitGroup
	wg.Add(len(lanes))
	go func() {
		defer wg.Done()
		st := console.SourceState{Source: "whatsapp", State: "ok"}
		if d.Devices == nil {
			st.State, st.Detail = "not_configured", "no device receiver"
		} else if r, err := d.Devices.WhatsAppChats(ctx); errors.Is(err, devicepush.ErrNoPush) {
			st.State, st.Detail = "not_configured", "no Mac has pushed WhatsApp yet"
		} else if err != nil {
			st.State, st.Detail = "error", err.Error()
		} else {
			if r.Stale {
				st.State, st.Detail = "stale", "last push from "+r.Label+" is stale"
			}
			chats := r.Data
			if len(chats) > 15 {
				chats = chats[:15]
			}
			for _, ch := range chats {
				items[0] = append(items[0], console.Item{Source: "whatsapp", Title: ch.Title, TS: ch.TS})
			}
		}
		states[0] = st
	}()
	go func() {
		defer wg.Done()
		st := console.SourceState{Source: "email", State: "ok"}
		mails, err := depsEmail(d).LatestEmails(ctx, 5)
		switch {
		case errors.Is(err, email.ErrNotConfigured):
			st.State, st.Detail = "not_configured", "no inbox configured"
		case err != nil:
			st.State, st.Detail = "error", err.Error()
		}
		for _, m := range mails {
			items[1] = append(items[1], console.Item{Source: "email", Title: m.Title, TS: m.TS})
		}
		states[1] = st
	}()
	go func() {
		defer wg.Done()
		st := console.SourceState{Source: "slack", State: "ok"}
		if d.Resolver.Resolve("SLACK_BOT_TOKEN") == "" {
			st.State, st.Detail = "not_configured", "SLACK_BOT_TOKEN not set"
		} else if msgs, err := depsSlack(d).RecentMessages(ctx, 15); err != nil {
			st.State, st.Detail = "error", err.Error()
		} else {
			for _, m := range msgs {
				items[2] = append(items[2], console.Item{Source: "slack", Title: "#" + m.Channel + " — " + m.User, TS: slackTS(m.TS)})
			}
		}
		states[2] = st
	}()
	wg.Wait()
	var all []console.Item
	for _, it := range items {
		all = append(all, it...)
	}
	return console.MergeFeed(all, 40), states
}

// slackTS turns Slack's "1727712000.000200" into RFC 3339.
func slackTS(ts string) string {
	sec := strings.SplitN(ts, ".", 2)[0]
	var n int64
	for _, c := range sec {
		if c < '0' || c > '9' {
			return ""
		}
		n = n*10 + int64(c-'0')
	}
	if n == 0 {
		return ""
	}
	return time.Unix(n, 0).UTC().Format(time.RFC3339)
}
