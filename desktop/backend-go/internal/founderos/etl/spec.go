// Package etl copies FounderOS v1's SQLite databases into the founderos_* tables
// of BusinessOS Postgres (spec criterion 3.3). The mapping, workspace rules,
// date rules and load order are docs/founderos/table-map.md; this file encodes
// that document as data.
package etl

import (
	"fmt"
	"strings"
)

// Source database files, relative to the --from directory.
const (
	FileOS     = "founderos-os.db"
	FileBank   = "bank.db"
	FileLedger = "ledger.db"
	FilePaykit = "paykit.db"
	FileState  = "state.db" // slack-bridge state.db
)

// The operator workspace slugs (cmd/founderos-bootstrap creates them, criterion 2.2).
const (
	WSLaunchpadCohort = "launchpad-cohort"
	WSVantage         = "vantage"
	WSFounderOS       = "founderos"
	WSPersonal        = "personal"
)

// Slugs is every workspace the ETL writes to; all must exist before a run.
var Slugs = []string{WSLaunchpadCohort, WSVantage, WSFounderOS, WSPersonal}

// DroppedTables are retired in FounderOS v1: counted, never loaded. Empty
// since migration 165: roadmap_items, phases and domains (dropped 2026-09-23)
// load again for /os/roadmap and /os/reference.
var DroppedTables = []string{}

type kind int

const (
	kText     kind = iota // TEXT NOT NULL
	kTextN                // TEXT NULL
	kInt                  // INTEGER NOT NULL
	kIntN                 // INTEGER NULL
	kBig                  // BIGINT NOT NULL
	kFloat                // DOUBLE PRECISION NOT NULL
	kFloatN               // DOUBLE PRECISION NULL
	kBool                 // 0/1 -> BOOLEAN NOT NULL
	kJSONArr              // JSON array -> JSONB
	kJSONStrs             // JSON string[] -> JSONB
	kJSONObj              // JSON object -> JSONB
	kTS                   // instant -> TIMESTAMPTZ NOT NULL (D1, D2, no-offset UTC)
	kTSN                  // instant -> TIMESTAMPTZ NULL ('' -> NULL, D6)
	kDate                 // YYYY-MM-DD -> DATE (D3)
	kMonth                // YYYY-MM -> DATE, first of month (D4)
)

// pgType is the cast used for parameters and key arrays.
func (k kind) pgType() string {
	switch k {
	case kText, kTextN:
		return "text"
	case kInt, kIntN:
		return "integer"
	case kBig:
		return "bigint"
	case kFloat, kFloatN:
		return "double precision"
	case kBool:
		return "boolean"
	case kJSONArr, kJSONStrs, kJSONObj:
		return "jsonb"
	case kTS, kTSN:
		return "timestamptz"
	case kDate, kMonth:
		return "date"
	}
	return "text"
}

func (k kind) nullable() bool {
	return k == kTextN || k == kIntN || k == kFloatN || k == kTSN
}

// col maps one source column to one target column. src "" means the column is
// computed by the table's prep hook. rowidCol reads the source rowid.
type col struct {
	src    string
	dst    string
	k      kind
	def    any  // value used when the source column does not exist (older DB)
	hasDef bool // def is meaningful (nil def + hasDef = NULL)
}

const rowidCol = "__rowid"

func c(src, dst string, k kind) col { return col{src: src, dst: dst, k: k} }
func same(name string, k kind) col  { return col{src: name, dst: name, k: k} }
func withDef(name string, k kind, def any) col {
	return col{src: name, dst: name, k: k, def: def, hasDef: true}
}
func computed(dst string, k kind) col { return col{dst: dst, k: k} }

type srcRef struct {
	file  string
	table string
}

// tableSpec is one target table.
type tableSpec struct {
	target  string
	sources []srcRef // more than one = union; later sources win on a key collision
	group   string   // one transaction per group (= per source file)
	cols    []col
	key     []string // target key columns (upsert conflict target + mirror delete)
	ws      string   // constant workspace slug, when wsFn is nil
	wsFn    func(raw map[string]any, out map[string]any, st *runState) (string, error)
	prep    func(raw map[string]any, out map[string]any, st *runState, tr *TableReport) error
	// newerOnly keeps the source's upsert guard: a row only moves forward in
	// this column (vsl_snapshots.captured_at).
	newerOnly string
}

func (t *tableSpec) colByDst(dst string) (col, bool) {
	for _, cl := range t.cols {
		if cl.dst == dst {
			return cl, true
		}
	}
	return col{}, false
}

// Specs lists every target table in FK load order (table-map.md "ETL notes").
func Specs() []*tableSpec {
	fromOS := func(tbl string) []srcRef { return []srcRef{{FileOS, tbl}} }
	specs := []*tableSpec{
		// 2. Parents first.
		{target: "founderos_departments", sources: fromOS("departments"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("name", kText), same("slug", kText), same("tagline", kText), same("color", kText), c("order", "ord", kInt)}},
		{target: "founderos_agents", sources: fromOS("agents"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("department_id", kText), same("name", kText), same("role", kText), same("status", kText),
				same("tier", kText), same("description", kText), same("model", kText), same("tools", kJSONStrs),
				withDef("parent_id", kTextN, nil), withDef("instance", kText, "builtin")}},
		{target: "founderos_people", sources: fromOS("people"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("department_id", kText), same("name", kText), same("role", kText), same("tools", kJSONStrs)}},
		{target: "founderos_sop_tasks", sources: fromOS("sop_tasks"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("department_id", kText), same("title", kText), same("summary", kText), same("steps", kJSONStrs),
				same("assignee_kind", kText), same("assignee_id", kText)}},
		{target: "founderos_tools", sources: fromOS("tools"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("name", kText), same("category", kText), same("status", kText), same("color", kText), same("description", kText)}},
		{target: "founderos_skills", sources: fromOS("skills"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("name", kText), same("category", kText), same("description", kText), same("owner_agent_id", kTextN),
				same("status", kText), same("tools", kJSONStrs), withDef("markdown", kText, ""), same("ord", kInt)}},
		{target: "founderos_personas", sources: fromOS("personas"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("ord", kInt), same("name", kText), same("archetype", kText), same("tagline", kText), same("summary", kText),
				same("accent", kText), same("north_star", kText), same("pillars", kJSONArr), same("connectors", kJSONStrs), same("metrics", kJSONStrs),
				same("brain_use", kText), same("signature_play", kText)}},
		{target: "founderos_workflows", sources: fromOS("workflows"), group: FileOS, key: []string{"id"}, wsFn: workflowWS,
			cols: []col{same("id", kText), same("name", kText), same("subtitle", kText), same("revenue_usd", kInt), same("ord", kInt), same("steps", kJSONArr)}},
		// /os/roadmap and /os/reference (migration 165): phases before the
		// roadmap rows that name them; phase_id and department_id are soft refs.
		{target: "founderos_phases", sources: fromOS("phases"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("number", kInt), same("title", kText), same("items", kJSONStrs)}},
		{target: "founderos_roadmap_items", sources: fromOS("roadmap_items"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("title", kText), same("quarter", kText), same("status", kText), same("department_id", kTextN),
				same("description", kText), withDef("phase_id", kTextN, nil)}},
		{target: "founderos_domains", sources: fromOS("domains"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("number", kInt), same("title", kText), same("color", kText), same("items", kJSONStrs)}},
		{target: "founderos_metrics", sources: fromOS("metrics"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("key", kText), same("label", kText), same("value", kFloat), same("unit", kText), same("delta", kFloat), same("period", kText)}},

		// Agent runtime.
		{target: "founderos_agent_runs", sources: fromOS("agent_runs"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("agent_id", kText), same("started_at", kTS), same("finished_at", kTS), same("ok", kBool), same("summary", kText),
				withDef("model", kTextN, nil), withDef("tokens_in", kIntN, nil), withDef("tokens_out", kIntN, nil), withDef("cost_usd", kFloatN, nil)}},
		{target: "founderos_agent_messages", sources: fromOS("agent_messages"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("agent_id", kText), same("role", kText), same("content", kText), same("tool_calls", kJSONArr), same("created_at", kTS)}},
		{target: "founderos_agent_tasks", sources: fromOS("agent_tasks"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("agent_id", kText), same("title", kText), same("status", kText), same("created_at", kTS), same("updated_at", kTS)}},
		{target: "founderos_agent_crons", sources: fromOS("agent_crons"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("agent_id", kText), same("schedule", kText), same("description", kText), same("enabled", kBool), same("created_at", kTS)}},
		{target: "founderos_cron_runs", sources: fromOS("cron_runs"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("cron_id", kText), same("agent_id", kText), same("started_at", kTS), same("finished_at", kTSN), same("ok", kBool), same("summary", kText)}},
		// 3. broadcasts -> replies.
		{target: "founderos_broadcasts", sources: fromOS("broadcasts"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("message", kText), same("created_at", kTS)}},
		{target: "founderos_broadcast_replies", sources: fromOS("broadcast_replies"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("broadcast_id", kText), same("agent_id", kText), same("ok", kBool), same("reply", kText), same("finished_at", kTS)}},

		// 4. funnel contacts -> touches; proposals -> deliverable decisions.
		{target: "founderos_funnel_contacts", sources: fromOS("funnel_contacts"), group: FileOS, key: []string{"id"}, wsFn: funnelContactWS,
			cols: []col{same("id", kText), same("name", kText), same("venture", kText), same("status", kText), same("product", kTextN), same("amount_usd", kFloatN),
				withDef("relationship", kText, "warm"), withDef("likelihood", kInt, int64(50)), withDef("email", kTextN, nil), withDef("phone", kTextN, nil),
				withDef("person", kTextN, nil), withDef("company", kTextN, nil), withDef("role", kTextN, nil), withDef("linkedin", kTextN, nil), same("created_at", kTS)}},
		{target: "founderos_funnel_touches", sources: fromOS("funnel_touches"), group: FileOS, key: []string{"id"}, wsFn: funnelTouchWS,
			cols: []col{same("id", kText), same("contact_id", kText), same("seq", kInt), same("stage", kText), same("channel", kText), same("label", kText),
				same("source", kText), same("at", kDate)}},
		// The retired CRM archive (Attio + GoHighLevel journeys, restored in
		// FounderOS v1 2026-09-26): one row per journey, kept whole as JSON, in
		// the journey's venture workspace (migration 164).
		{target: "founderos_funnel_archive", sources: fromOS("funnel_archive"), group: FileOS, key: []string{"id"}, wsFn: funnelArchiveWS,
			cols: []col{same("id", kText), same("venture", kText), same("journey", kJSONObj)}},
		{target: "founderos_proposals", sources: fromOS("proposals"), group: FileOS, key: []string{"id"}, wsFn: proposalWS, prep: prepProposal,
			cols: []col{same("id", kText), same("client", kText), same("brand", kText), same("url", kText), same("status", kText), same("amount_usd", kFloatN),
				same("notes", kText), same("created_at", kTS), withDef("origin", kText, "seed"), computed("has_access_code", kBool)}},
		{target: "founderos_deliverable_decisions", sources: fromOS("deliverable_decisions"), group: FileOS, key: []string{"id"}, wsFn: decisionWS,
			cols: []col{same("id", kText), same("decision", kText), same("decided_at", kTS), same("decided_revision", kText), same("note", kText)}},

		// 5. Everything else.
		{target: "founderos_comms_digests", sources: fromOS("comms_digests"), group: FileOS, ws: WSPersonal, key: []string{"id"},
			cols: []col{same("id", kText), same("generated_at", kTS), same("payload", kJSONObj)}},
		{target: "founderos_digest_reads", sources: fromOS("digest_reads"), group: FileOS, ws: WSPersonal, key: []string{"key"},
			cols: []col{same("key", kText), same("read_at", kTS)}},
		{target: "founderos_contact_tags", sources: fromOS("contact_tags"), group: FileOS, ws: WSPersonal, key: []string{"person", "channel"},
			cols: []col{same("person", kText), same("channel", kText), same("tag", kText), same("tier", kInt)}},
		{target: "founderos_plaud_ingests", sources: fromOS("plaud_ingests"), group: FileOS, ws: WSFounderOS, key: []string{"file_id"},
			cols: []col{same("file_id", kText), same("title", kText), same("recorded_at", kTSN), same("ingested_at", kTS), same("via", kText), same("slug", kText), same("claims", kInt)}},
		{target: "founderos_social_accounts", sources: fromOS("social_accounts"), group: FileOS, ws: WSPersonal, key: []string{"platform"},
			cols: []col{same("platform", kText), same("handle", kText), same("url", kTextN), c("order", "ord", kInt)}},
		{target: "founderos_social_snapshots", sources: fromOS("social_snapshots"), group: FileOS, ws: WSPersonal, key: []string{"platform", "captured_on"},
			cols: []col{same("platform", kText), c("captured_at", "captured_on", kDate), same("followers", kInt), same("source", kText)}},
		{target: "founderos_social_dms", sources: fromOS("social_dms"), group: FileOS, ws: WSPersonal, key: []string{"platform"},
			cols: []col{same("platform", kText), same("count", kInt), same("updated_at", kTS)}},
		{target: "founderos_social_dm_snapshots", sources: fromOS("social_dm_snapshots"), group: FileOS, ws: WSPersonal, key: []string{"platform", "captured_on"},
			cols: []col{same("platform", kText), c("captured_at", "captured_on", kDate), same("count", kInt), same("source", kText)}},
		{target: "founderos_social_dm_messages", sources: fromOS("social_dm_messages"), group: FileOS, ws: WSPersonal, key: []string{"id"},
			cols: []col{same("id", kText), same("platform", kText), same("subscriber_id", kText), same("name", kText), same("handle", kTextN), same("text", kText),
				same("direction", kText), same("tag", kTextN), same("ts", kTS), same("source", kText)}},
		{target: "founderos_social_posts", sources: fromOS("social_posts"), group: FileOS, ws: WSPersonal, key: []string{"id"},
			cols: []col{same("id", kText), same("caption", kText), same("media_url", kTextN), same("platforms", kJSONStrs), same("status", kText),
				same("scheduled_for", kTSN), same("created_at", kTS)}},
		{target: "founderos_email_list_snapshots", sources: fromOS("email_list_snapshots"), group: FileOS, ws: WSPersonal, key: []string{"captured_on"},
			cols: []col{c("captured_at", "captured_on", kDate), same("subscribers", kInt), same("source", kText), withDef("publication_id", kText, ""),
				withDef("metric", kText, ""), withDef("quality", kText, "ok")}},
		{target: "founderos_email_list_sync_misses", sources: fromOS("email_list_sync_misses"), group: FileOS, ws: WSPersonal, key: []string{"attempted_on"},
			cols: []col{c("attempted_at", "attempted_on", kDate), same("reason", kText)}},
		{target: "founderos_lead_magnets", sources: fromOS("lead_magnets"), group: FileOS, ws: WSPersonal, key: []string{"id"},
			cols: []col{same("id", kText), same("name", kText), same("offer", kText), same("url", kText), same("status", kText), same("captures", kText),
				same("destination", kText), same("source", kText), same("launched_at", kText), same("notes", kText), withDef("origin", kText, "seed")}},
		{target: "founderos_brand_deals", sources: fromOS("brand_deals"), group: FileOS, ws: WSPersonal, key: []string{"id"},
			cols: []col{same("id", kText), same("brand", kText), same("status", kText), same("tier", kTextN), same("deal_value_usd", kFloatN),
				same("budget_usd", kFloatN), same("amount_agreed_usd", kFloatN), same("suggested_rate_usd", kFloatN), same("paid_in_full", kBool),
				same("deadline", kTSN), same("follow_up_date", kTSN), same("contact_name", kTextN), same("contact_email", kTextN), same("main_channel", kTextN),
				same("video_type", kTextN), same("source", kTextN), same("icp_fit", kTextN), same("notion_url", kText), same("last_edited", kTS), same("seeded", kBool)}},
		{target: "founderos_client_work", sources: fromOS("client_work"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), c("clientId", "client_id", kText), same("brief", kText), same("status", kText), c("createdAt", "created_at", kTS),
				c("workspaceId", "superset_workspace_id", kTextN), same("detail", kTextN)}},
		{target: "founderos_vsl_snapshots", sources: fromOS("vsl_snapshots"), group: FileOS, ws: WSLaunchpadCohort, key: []string{"video_id", "date_from", "date_to"},
			newerOnly: "captured_at",
			cols:      []col{same("video_id", kText), same("date_from", kDate), same("date_to", kDate), same("captured_at", kTS), same("payload", kJSONObj)}},
		{target: "founderos_trading_snapshots", sources: fromOS("trading_snapshots"), group: FileOS, ws: WSPersonal, key: []string{"account_id", "captured_at"},
			cols: []col{withDef("account_id", kText, "individual"), withDef("account_label", kText, "Individual"), same("captured_at", kTS),
				same("account_value_usd", kFloat), same("buying_power_usd", kFloat), same("cash_usd", kFloat), same("day_pnl_usd", kFloat),
				same("total_pnl_usd", kFloat), same("source", kText)}},
		{target: "founderos_trading_positions", sources: fromOS("trading_positions"), group: FileOS, ws: WSPersonal, key: []string{"account_id", "captured_at", "symbol"},
			cols: []col{withDef("account_id", kText, "individual"), same("captured_at", kTS), same("symbol", kText), same("quantity", kFloat),
				same("avg_cost_usd", kFloat), same("market_value_usd", kFloat), same("unrealized_pnl_usd", kFloat)}},
		{target: "founderos_trading_orders", sources: fromOS("trading_orders"), group: FileOS, ws: WSPersonal, key: []string{"id"},
			cols: []col{same("id", kText), same("account_id", kText), same("symbol", kText), same("side", kText), same("type", kText), same("state", kText),
				same("quantity", kFloat), same("filled_quantity", kFloat), same("dollar_amount_usd", kFloatN), same("limit_price_usd", kFloatN),
				same("placed_agent", kText), same("created_at", kTS)}},
		{target: "founderos_trading_analysis", sources: fromOS("trading_analysis"), group: FileOS, ws: WSPersonal, key: []string{"id"},
			cols: []col{same("id", kText), same("at", kTS), same("account_id", kText), same("agent", kText), same("examined", kInt), same("signals", kInt),
				same("notes", kText), same("rows", kJSONArr)}},
		{target: "founderos_trading_activity", sources: fromOS("trading_activity"), group: FileOS, ws: WSPersonal, key: []string{"id"},
			cols: []col{same("id", kText), same("at", kTS), withDef("account_id", kText, "individual"), same("agent", kText), same("action", kText),
				same("symbol", kText), same("quantity", kFloat), same("price_usd", kFloat), same("rationale", kText), same("status", kText)}},
		{target: "founderos_trading_limits", sources: fromOS("trading_limits"), group: FileOS, ws: WSPersonal, key: []string{"workspace_id"}, prep: prepTradingLimits,
			cols: []col{same("max_notional_per_trade_usd", kFloat), same("max_position_pct_of_sleeve", kFloat), same("max_risk_pct_per_trade", kFloat),
				same("max_concurrent_positions", kInt), same("max_trades_per_day", kInt), same("min_sleeve_value_usd", kFloat),
				same("max_deployed_capital_usd", kFloat), withDef("autopilot", kBool, int64(0)), same("updated_at", kTS)}},
		{target: "founderos_metric_snapshots", sources: fromOS("metric_snapshots"), group: FileOS, ws: WSFounderOS, key: []string{"metric_id", "captured_at"},
			cols: []col{same("metric_id", kText), same("captured_at", kTS), same("value", kFloat)}},
		{target: "founderos_usage_snapshots", sources: fromOS("usage_snapshots"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("captured_at", kTS), same("payload", kJSONObj)}},
		{target: "founderos_ollama_snapshots", sources: fromOS("ollama_snapshots"), group: FileOS, ws: WSFounderOS, key: []string{"id"},
			cols: []col{same("id", kText), same("captured_at", kTS), same("payload", kJSONObj)}},
		{target: "founderos_seed_meta", sources: fromOS("seed_meta"), group: FileOS, ws: WSFounderOS, key: []string{"key"},
			cols: []col{same("key", kText), same("value", kText)}},

		// Side databases.
		{target: "founderos_bank_summaries", sources: []srcRef{{FileBank, "bank_summaries"}}, group: FileBank, key: []string{"account", "month"}, wsFn: bankWS,
			cols: []col{same("account", kText), same("business", kText), same("month", kMonth), same("credits_cents", kBig), same("debits_cents", kBig), same("net_cents", kBig)}},
		{target: "founderos_ledger_rows", sources: []srcRef{{FileLedger, "ledger_rows"}}, group: FileLedger, key: []string{"hash"}, wsFn: ledgerWS, prep: prepLedger,
			cols: []col{same("hash", kText), same("date", kDate), same("description", kText), same("amount_cents", kBig), same("direction", kText),
				same("category", kText), withDef("card", kText, "platinum")}},
		{target: "founderos_paykit_customer_snapshots", sources: []srcRef{{FilePaykit, "paykit_customer_snapshots"}}, group: FilePaykit,
			key: []string{"account", "captured_on", "customer_id"}, wsFn: paykitWS,
			cols: []col{same("account", kText), same("captured_on", kDate), same("customer_id", kText), same("total_spent_cents", kBig),
				same("total_transactions", kInt), same("last_transaction_date", kTextN), same("source", kText)}},
		{target: "founderos_slack_bridge_jobs", sources: []srcRef{{FileOS, "slack_bridge_jobs"}, {FileState, "slack_bridge_jobs"}}, group: FileState,
			ws: WSFounderOS, key: []string{"id"},
			cols: []col{c(rowidCol, "seq", kBig), same("id", kText), same("payload", kJSONObj), same("state", kText)}},
		{target: "founderos_slack_bridge_sessions", sources: []srcRef{{FileOS, "slack_bridge_sessions"}, {FileState, "slack_bridge_sessions"}}, group: FileState,
			ws: WSFounderOS, key: []string{"key"}, prep: prepSlackSession,
			cols: []col{same("key", kText), same("payload", kJSONObj)}},
	}
	return specs
}

// ---------------------------------------------------------------------------
// Per-row workspace rules (table-map.md "Per-row workspace rules").
// ---------------------------------------------------------------------------

func workflowWS(raw, _ map[string]any, _ *runState) (string, error) {
	id, _ := raw["id"].(string)
	switch {
	case strings.HasPrefix(id, "wf-vantage"):
		return WSVantage, nil
	case strings.HasPrefix(id, "wf-lc"):
		return WSLaunchpadCohort, nil
	}
	return WSFounderOS, nil
}

func ventureWS(v any, what string) (string, error) {
	s, _ := v.(string)
	switch s {
	case "vantage":
		return WSVantage, nil
	case "launchpad-cohort":
		return WSLaunchpadCohort, nil
	}
	return "", fmt.Errorf("%s %q is not vantage or launchpad-cohort", what, s)
}

func funnelArchiveWS(raw, _ map[string]any, _ *runState) (string, error) {
	return ventureWS(raw["venture"], "venture")
}

func funnelContactWS(raw, out map[string]any, st *runState) (string, error) {
	ws, err := ventureWS(raw["venture"], "venture")
	if err != nil {
		return "", err
	}
	st.contactWS[out["id"].(string)] = ws
	return ws, nil
}

func funnelTouchWS(raw, _ map[string]any, st *runState) (string, error) {
	cid, _ := raw["contact_id"].(string)
	ws, ok := st.contactWS[cid]
	if !ok {
		return "", fmt.Errorf("touch references unknown contact %q", cid)
	}
	return ws, nil
}

func proposalWS(raw, out map[string]any, st *runState) (string, error) {
	ws, err := ventureWS(raw["brand"], "brand")
	if err != nil {
		return "", err
	}
	st.proposalWS[out["id"].(string)] = ws
	return ws, nil
}

func decisionWS(raw, _ map[string]any, st *runState) (string, error) {
	id, _ := raw["id"].(string)
	if pid, ok := strings.CutPrefix(id, "proposal:"); ok {
		if ws, ok := st.proposalWS[pid]; ok {
			return ws, nil
		}
	}
	return WSFounderOS, nil
}

// bankWS: Vantage -> vantage; General Operations and anything else ->
// launchpad-cohort (the general-business bucket is LC's, lib/cards.ts).
func bankWS(raw, _ map[string]any, _ *runState) (string, error) {
	if b, _ := raw["business"].(string); b == "Vantage" {
		return WSVantage, nil
	}
	return WSLaunchpadCohort, nil
}

// NormalizeCardID mirrors FounderOS v1 lib/cards.ts normalizeCardId.
func NormalizeCardID(raw any) string {
	s, ok := raw.(string)
	if !ok {
		return "platinum"
	}
	t := strings.ToLower(strings.TrimSpace(s))
	switch t {
	case "gold", "platinum", "blue":
		return t
	case "business":
		return "platinum"
	case "vantage":
		return "blue"
	}
	return "platinum"
}

func ledgerWS(_, out map[string]any, _ *runState) (string, error) {
	switch out["card"] {
	case "gold":
		return WSPersonal, nil
	case "blue":
		return WSVantage, nil
	}
	return WSLaunchpadCohort, nil
}

func paykitWS(raw, _ map[string]any, _ *runState) (string, error) {
	switch a, _ := raw["account"].(string); a {
	case "paykit-lc":
		return WSLaunchpadCohort, nil
	case "paykit-vantage":
		return WSVantage, nil
	default:
		return "", fmt.Errorf("paykit account %q has no workspace rule", a)
	}
}

// ---------------------------------------------------------------------------
// Prep hooks: the value changes the map calls for.
// ---------------------------------------------------------------------------

// prepProposal moves access_code out of the row: the value goes to the vault
// payload, only its presence is stored in founderos_proposals.
func prepProposal(raw, out map[string]any, st *runState, _ *TableReport) error {
	code, _ := raw["access_code"].(string)
	out["has_access_code"] = code != ""
	if code != "" {
		st.accessCodes[out["id"].(string)] = code
	}
	return nil
}

func prepLedger(_, out map[string]any, _ *runState, _ *TableReport) error {
	out["card"] = NormalizeCardID(out["card"])
	return nil
}

// prepTradingLimits drops the singleton id (the PK is workspace_id) and forces
// autopilot OFF: the bridge never inherits a live trading switch.
func prepTradingLimits(_, out map[string]any, st *runState, _ *TableReport) error {
	on := out["autopilot"].(bool)
	st.sourceAutopilot = &on
	out["autopilot"] = false
	return nil
}

// prepSlackSession blanks pending: pending.code is a two-minute confirmation
// nonce, and copying it would carry a live auth token for no benefit.
func prepSlackSession(_, out map[string]any, st *runState, _ *TableReport) error {
	var obj map[string]any
	if err := jsonUnmarshal(out["payload"].(string), &obj); err != nil {
		return err
	}
	if p, ok := obj["pending"]; ok && p != nil {
		st.pendingNulled++
	}
	obj["pending"] = nil
	s, err := jsonMarshal(obj)
	if err != nil {
		return err
	}
	out["payload"] = s
	return nil
}

// ParseWorkspaceMap reads --workspace-map: comma-separated
// founderos-slug=businessos-slug-or-uuid pairs.
func ParseWorkspaceMap(s string) (map[string]string, error) {
	out := map[string]string{}
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		k, v, ok := strings.Cut(pair, "=")
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if !ok || v == "" {
			return nil, fmt.Errorf("workspace map entry %q is not slug=target", pair)
		}
		known := false
		for _, slug := range Slugs {
			known = known || slug == k
		}
		if !known {
			return nil, fmt.Errorf("workspace map names %q; the FounderOS slugs are %s", k, strings.Join(Slugs, ", "))
		}
		out[k] = v
	}
	return out, nil
}
