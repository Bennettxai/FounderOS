import Database from 'better-sqlite3';
import { randomUUID } from 'node:crypto';
import fs from 'node:fs';
import { dirname, join } from 'node:path';
import { isValidCron } from '@/lib/cron';
import { labStore } from '@/lib/lab/store';
import { tradingRepo } from '@/lib/db-ext/trading';
import { clientConfirmationsRepo, clientWorkRepo } from '@/lib/db-ext/clients';
import { leadMagnetsRepo } from '@/lib/db-ext/lead-magnets';
import { usageReadingsRepo } from '@/lib/db-ext/usage';
import { commsDigestsRepo } from '@/lib/db-ext/comms-digest';
import { callRecordingsRepo } from '@/lib/db-ext/recordings';
import { cronRunsRepo, deliverableDecisionsRepo } from '@/lib/db-ext/agent-ops';
import {
  AgentCronSchema,
  AgentMessageSchema,
  AgentRunSchema,
  AgentSchema,
  AgentTaskSchema,
  AgentTaskStatusSchema,
  type AgentTaskInput,
  BroadcastReplySchema,
  BroadcastSchema,
  ContactTagSchema,
  DepartmentSchema,
  MetricSchema,
  PhaseSchema,
  RoadmapItemSchema,
  SocialAccountSchema,
  SocialSnapshotSchema,
  EmailListSnapshotSchema,
  SocialDmSchema,
  SocialDmSnapshotSchema,
  SocialDmMessageSchema,
  SocialPostSchema,
  FunnelContactSchema,
  FunnelTouchSchema,
  FunnelJourneySchema,
  PersonSchema,
  SopTaskSchema,
  WorkflowSchema,
  WorkflowVersionSchema,
  ConfigVersionSchema,
  SkillSchema,
  ToolSchema,
  type Agent,
  type AgentCron,
  type AgentMessage,
  type AgentRun,
  type AgentTask,
  type Broadcast,
  type BroadcastReply,
  type ContactTag,
  type Department,
  type Metric,
  type Phase,
  type RoadmapItem,
  type SocialAccount,
  type SocialPlatform,
  type SocialSnapshot,
  type EmailListSnapshot,
  type SocialDm,
  type SocialDmSnapshot,
  type SocialDmMessage,
  type SocialPost,
  type FunnelContact,
  type FunnelTouch,
  type FunnelJourney,
  type FunnelVenture,
  type Person,
  type SopTask,
  type Workflow,
  type WorkflowVersion,
  type ConfigVersion,
  type Skill,
  type Tool,
  BrandDealOverrideSchema,
  type BrandDealOverride,
  SetupCheckSchema,
  type SetupCheck,
} from '@/lib/schemas';

const DDL = `
CREATE TABLE IF NOT EXISTS departments (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  tagline TEXT NOT NULL DEFAULT '',
  color TEXT NOT NULL,
  "order" INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS agents (
  id TEXT PRIMARY KEY,
  department_id TEXT NOT NULL REFERENCES departments(id),
  name TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  tier TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  tools TEXT NOT NULL DEFAULT '[]'
);
CREATE TABLE IF NOT EXISTS tools (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  category TEXT NOT NULL,
  status TEXT NOT NULL,
  color TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS roadmap_items (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  quarter TEXT NOT NULL,
  status TEXT NOT NULL,
  department_id TEXT,
  description TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS metrics (
  id TEXT PRIMARY KEY,
  key TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL,
  value REAL NOT NULL,
  unit TEXT NOT NULL DEFAULT '',
  delta REAL NOT NULL DEFAULT 0,
  period TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS phases (
  id TEXT PRIMARY KEY,
  number INTEGER NOT NULL,
  title TEXT NOT NULL,
  items TEXT NOT NULL DEFAULT '[]'
);
CREATE TABLE IF NOT EXISTS agent_runs (
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT NOT NULL,
  ok INTEGER NOT NULL,
  summary TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS agent_messages (
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL,
  role TEXT NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  tool_calls TEXT NOT NULL DEFAULT '[]',
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS agent_chat_sessions (
  agent_id TEXT PRIMARY KEY,
  first_message_id TEXT NOT NULL,
  session_id TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS broadcasts (
  id TEXT PRIMARY KEY,
  message TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS agent_tasks (
  id TEXT PRIMARY KEY,
  agent_id TEXT,
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  detail TEXT NOT NULL DEFAULT '',
  due_at TEXT,
  priority TEXT NOT NULL DEFAULT 'normal',
  owner_kind TEXT NOT NULL DEFAULT 'agent',
  horizon TEXT,
  run_id TEXT
);
CREATE TABLE IF NOT EXISTS agent_crons (
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL,
  schedule TEXT NOT NULL,
  description TEXT NOT NULL,
  enabled INTEGER NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS contact_tags (
  person TEXT NOT NULL,
  channel TEXT NOT NULL,
  tag TEXT NOT NULL,
  tier INTEGER NOT NULL,
  PRIMARY KEY (person, channel)
);
CREATE TABLE IF NOT EXISTS social_accounts (
  platform TEXT PRIMARY KEY,
  handle TEXT NOT NULL,
  url TEXT,
  "order" INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS social_snapshots (
  platform TEXT NOT NULL,
  captured_at TEXT NOT NULL,
  followers INTEGER NOT NULL,
  source TEXT NOT NULL,
  PRIMARY KEY (platform, captured_at)
);
CREATE TABLE IF NOT EXISTS broadcast_replies (
  id TEXT PRIMARY KEY,
  broadcast_id TEXT NOT NULL REFERENCES broadcasts(id),
  agent_id TEXT NOT NULL,
  ok INTEGER NOT NULL,
  reply TEXT NOT NULL DEFAULT '',
  finished_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS email_list_snapshots (
  captured_at TEXT PRIMARY KEY,
  subscribers INTEGER NOT NULL,
  source TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS social_dms (
  platform TEXT PRIMARY KEY,
  count INTEGER NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS social_dm_snapshots (
  platform TEXT NOT NULL,
  captured_at TEXT NOT NULL,
  count INTEGER NOT NULL,
  source TEXT NOT NULL,
  PRIMARY KEY (platform, captured_at)
);
CREATE TABLE IF NOT EXISTS social_dm_messages (
  id TEXT PRIMARY KEY,
  platform TEXT NOT NULL,
  subscriber_id TEXT NOT NULL,
  name TEXT NOT NULL,
  handle TEXT,
  text TEXT NOT NULL,
  direction TEXT NOT NULL,
  tag TEXT,
  ts TEXT NOT NULL,
  source TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_social_dm_messages_ts ON social_dm_messages (ts);
CREATE TABLE IF NOT EXISTS social_posts (
  id TEXT PRIMARY KEY,
  caption TEXT NOT NULL,
  media_url TEXT,
  platforms TEXT NOT NULL,
  status TEXT NOT NULL,
  scheduled_for TEXT,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS people (
  id TEXT PRIMARY KEY,
  department_id TEXT NOT NULL REFERENCES departments(id),
  name TEXT NOT NULL,
  role TEXT NOT NULL,
  tools TEXT NOT NULL DEFAULT '[]'
);
CREATE TABLE IF NOT EXISTS sop_tasks (
  id TEXT PRIMARY KEY,
  department_id TEXT NOT NULL REFERENCES departments(id),
  title TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  steps TEXT NOT NULL DEFAULT '[]',
  assignee_kind TEXT NOT NULL,
  assignee_id TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS funnel_contacts (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  venture TEXT NOT NULL,
  status TEXT NOT NULL,
  product TEXT,
  amount_usd REAL,
  relationship TEXT NOT NULL DEFAULT 'warm',
  likelihood INTEGER NOT NULL DEFAULT 50,
  email TEXT,
  phone TEXT,
  person TEXT,
  company TEXT,
  role TEXT,
  linkedin TEXT,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS funnel_touches (
  id TEXT PRIMARY KEY,
  contact_id TEXT NOT NULL REFERENCES funnel_contacts(id),
  seq INTEGER NOT NULL,
  stage TEXT NOT NULL,
  channel TEXT NOT NULL,
  label TEXT NOT NULL,
  source TEXT NOT NULL,
  at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS workflows (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  subtitle TEXT NOT NULL DEFAULT '',
  revenue_usd INTEGER NOT NULL DEFAULT 0,
  ord INTEGER NOT NULL DEFAULT 0,
  steps TEXT NOT NULL DEFAULT '[]'
);
CREATE TABLE IF NOT EXISTS skills (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  category TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  owner_agent_id TEXT,
  status TEXT NOT NULL DEFAULT 'planned',
  tools TEXT NOT NULL DEFAULT '[]',
  markdown TEXT NOT NULL DEFAULT '',
  ord INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS setup_checks (
  id TEXT PRIMARY KEY,
  done_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS brand_deal_overrides (
  deal_id TEXT PRIMARY KEY,
  outcome TEXT NOT NULL DEFAULT 'open',
  amount_usd INTEGER,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS workflow_versions (
  id TEXT PRIMARY KEY,
  workflow_id TEXT NOT NULL,
  version INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  snapshot TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS workflow_versions_wf ON workflow_versions (workflow_id, version);
CREATE TABLE IF NOT EXISTS config_versions (
  id TEXT PRIMARY KEY,
  target TEXT NOT NULL,
  version INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  snapshot TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS config_versions_target ON config_versions (target, version);
`;

/* ── FounderOS UI merge tables (docs/projects/founderos-ui/SPEC.md §2.2, SCAFFOLD §2) ──
   Additive only: CREATE … IF NOT EXISTS, so opening an older database IS the migration
   (migrateFounderOs below). Read and written through the repos in lib/db-ext/*; zod
   schemas in lib/schemas.ts (fo/data block). None is seeded: a fresh install has no row
   in any of them (AC-11). Chats has no table: it reads agent_messages. */
export const FO_DDL = `
CREATE TABLE IF NOT EXISTS trading_snapshots (
  account_id TEXT NOT NULL,
  account_label TEXT NOT NULL,
  captured_at TEXT NOT NULL,
  account_value_usd REAL NOT NULL,
  buying_power_usd REAL NOT NULL,
  cash_usd REAL NOT NULL,
  day_pnl_usd REAL NOT NULL,
  total_pnl_usd REAL NOT NULL,
  source TEXT NOT NULL,
  PRIMARY KEY (account_id, captured_at)
);
CREATE TABLE IF NOT EXISTS trading_positions (
  account_id TEXT NOT NULL,
  captured_at TEXT NOT NULL,
  symbol TEXT NOT NULL,
  quantity REAL NOT NULL,
  avg_cost_usd REAL NOT NULL,
  market_value_usd REAL NOT NULL,
  unrealized_pnl_usd REAL NOT NULL,
  PRIMARY KEY (account_id, captured_at, symbol)
);
CREATE TABLE IF NOT EXISTS trading_activity (
  id TEXT PRIMARY KEY,
  at TEXT NOT NULL,
  account_id TEXT NOT NULL,
  agent TEXT NOT NULL,
  action TEXT NOT NULL,
  symbol TEXT NOT NULL,
  quantity REAL NOT NULL,
  price_usd REAL NOT NULL,
  rationale TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS trading_activity_by_at ON trading_activity (at);
CREATE TABLE IF NOT EXISTS trading_analysis (
  id TEXT PRIMARY KEY,
  at TEXT NOT NULL,
  account_id TEXT NOT NULL,
  agent TEXT NOT NULL,
  examined INTEGER NOT NULL DEFAULT 0,
  signals INTEGER NOT NULL DEFAULT 0,
  notes TEXT NOT NULL DEFAULT '',
  rows_json TEXT NOT NULL DEFAULT '[]'
);
CREATE TABLE IF NOT EXISTS trading_orders (
  id TEXT PRIMARY KEY,
  account_id TEXT NOT NULL,
  symbol TEXT NOT NULL,
  side TEXT NOT NULL,
  type TEXT NOT NULL,
  state TEXT NOT NULL,
  quantity REAL NOT NULL,
  filled_quantity REAL NOT NULL DEFAULT 0,
  dollar_amount_usd REAL,
  limit_price_usd REAL,
  placed_agent TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
-- one row by construction, so an agent can never read one of two sets
CREATE TABLE IF NOT EXISTS trading_limits (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  max_deployed_capital_usd REAL NOT NULL,
  max_notional_per_trade_usd REAL NOT NULL,
  max_position_pct_of_sleeve REAL NOT NULL,
  max_risk_pct_per_trade REAL NOT NULL,
  max_concurrent_positions INTEGER NOT NULL,
  max_trades_per_day INTEGER NOT NULL,
  min_sleeve_value_usd REAL NOT NULL,
  autopilot INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS client_confirmations (
  client_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  company TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL,
  retainer INTEGER NOT NULL DEFAULT 0,
  slack_channel TEXT,
  superset_project TEXT,
  confirmed_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS client_work (
  id TEXT PRIMARY KEY,
  client_id TEXT NOT NULL,
  brief TEXT NOT NULL,
  status TEXT NOT NULL,
  superset_workspace TEXT,
  created_at TEXT NOT NULL,
  finished_at TEXT,
  detail TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS client_work_by_client ON client_work (client_id, created_at);
CREATE TABLE IF NOT EXISTS lead_magnets (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  offer TEXT NOT NULL DEFAULT '',
  live_url TEXT NOT NULL,
  destination TEXT NOT NULL,
  campaign TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  went_live_at TEXT,
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS usage_readings (
  machine TEXT NOT NULL,
  plan TEXT NOT NULL,
  day TEXT NOT NULL,
  model TEXT NOT NULL,
  input INTEGER NOT NULL DEFAULT 0,
  output INTEGER NOT NULL DEFAULT 0,
  cache_write INTEGER NOT NULL DEFAULT 0,
  pushed_at TEXT NOT NULL,
  PRIMARY KEY (machine, plan, day, model)
);
CREATE TABLE IF NOT EXISTS comms_digests (
  id TEXT PRIMARY KEY,
  date TEXT NOT NULL,
  body_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS digest_reads (
  digest_id TEXT NOT NULL,
  entry_key TEXT NOT NULL DEFAULT '',
  read_at TEXT NOT NULL,
  PRIMARY KEY (digest_id, entry_key)
);
CREATE TABLE IF NOT EXISTS call_recordings (
  id TEXT PRIMARY KEY,
  source TEXT NOT NULL,
  external_id TEXT NOT NULL,
  title TEXT NOT NULL,
  started_at TEXT NOT NULL,
  duration_s INTEGER,
  participants_json TEXT NOT NULL DEFAULT '[]',
  summary TEXT NOT NULL DEFAULT '',
  transcript_url TEXT,
  UNIQUE (source, external_id)
);
CREATE TABLE IF NOT EXISTS deliverable_decisions (
  run_id TEXT PRIMARY KEY,
  decision TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS cron_runs (
  id TEXT PRIMARY KEY,
  cron_id TEXT NOT NULL,
  agent_id TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT NOT NULL,
  ok INTEGER NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  trigger TEXT NOT NULL DEFAULT 'schedule'
);
CREATE INDEX IF NOT EXISTS cron_runs_by_cron ON cron_runs (cron_id, started_at);
`;

/** Every table FO_DDL creates (tests assert each exists and starts empty). */
export const FO_TABLES = [
  'trading_snapshots', 'trading_positions', 'trading_activity', 'trading_analysis', 'trading_orders', 'trading_limits',
  'client_confirmations', 'client_work', 'lead_magnets', 'usage_readings', 'comms_digests', 'digest_reads',
  'call_recordings', 'deliverable_decisions', 'cron_runs',
] as const;

/**
 * Bring an existing database up to the FounderOS-merge schema. Idempotent:
 * every statement is IF NOT EXISTS, and a cron_runs table made by the
 * 2026-10-06 parts bin (no `trigger` column) gains it in place.
 */
export function migrateFounderOs(db: InstanceType<typeof Database>): void {
  // The parts bin's digest tables had another shape (payload / key). Keep any such rows
  // under *_v0 rather than dropping them, then create the current shape.
  const colsOf = (t: string) => (db.prepare(`PRAGMA table_info(${t})`).all() as { name: string }[]).map((c) => c.name);
  for (const [table, mustHave] of [['comms_digests', 'body_json'], ['digest_reads', 'digest_id']] as const) {
    const cols = colsOf(table);
    if (cols.length > 0 && !cols.includes(mustHave) && colsOf(`${table}_v0`).length === 0) {
      db.exec(`ALTER TABLE ${table} RENAME TO ${table}_v0`);
    }
  }
  db.exec(FO_DDL);
  const cols = (db.prepare('PRAGMA table_info(cron_runs)').all() as { name: string }[]).map((c) => c.name);
  if (!cols.includes('trigger')) db.exec("ALTER TABLE cron_runs ADD COLUMN trigger TEXT NOT NULL DEFAULT 'schedule'");
}

/* ── Content Lab tables (docs/projects/content-lab/SPEC.md §6, CONTRACT.md) ──────────
   Additive only: CREATE … IF NOT EXISTS, never altered in place. Read and written
   through lib/lab/store.ts (labStore); nothing else touches lab_* directly. Measures,
   profiles and trend snapshots are derived and can be rebuilt from posts, metrics and
   dissections. Exported so the sample preview can build the same schema in memory. */
/* ── Content Lab v2 tables (S0; docs/projects/content-lab/v2/SPEC.md 3.4, 08.11.2) ──────────
   Appended to LAB_DDL below. Additive only (CREATE ... IF NOT EXISTS). Each table names its
   DDL owner section. Columns added to v1 tables are NOT here: they are LAB_V2_COLUMNS, applied
   by the one migrateLabV2 (indexes on those columns are created there too, after the column). */
export const LAB_V2_DDL = `
-- 04 section 8
CREATE TABLE IF NOT EXISTS lab_controls (
  post_id TEXT PRIMARY KEY,
  role TEXT NOT NULL,
  stratum TEXT NOT NULL,
  sampling_prob REAL NOT NULL,
  eligible_n INTEGER NOT NULL DEFAULT 0,
  drawn_n INTEGER NOT NULL DEFAULT 0,
  seed TEXT NOT NULL,
  verdict_age_h REAL NOT NULL,
  verdict_ratio REAL,
  drawn_at TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'selected',
  measure_version TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS lab_controls_stratum ON lab_controls (stratum);
-- 05 section 16
CREATE TABLE IF NOT EXISTS lab_events (
  id TEXT PRIMARY KEY,
  key TEXT NOT NULL,
  name TEXT NOT NULL,
  vendor TEXT,
  aliases_json TEXT NOT NULL DEFAULT '[]',
  status TEXT NOT NULL DEFAULT 'candidate',
  merged_into TEXT,
  dismiss_reason TEXT,
  novelty TEXT,
  novelty_source TEXT,
  class TEXT NOT NULL DEFAULT 'unknown',
  class_basis TEXT,
  class_json TEXT NOT NULL DEFAULT '{}',
  wiki_article TEXT,
  attention_json TEXT NOT NULL DEFAULT '[]',
  t0 TEXT,
  t0_rule TEXT,
  t0_ping_id TEXT,
  t0_revised_at TEXT,
  confirmed_at TEXT,
  first_seen_at TEXT NOT NULL,
  source_count INTEGER NOT NULL DEFAULT 0,
  sources_json TEXT NOT NULL DEFAULT '[]',
  wave_n INTEGER NOT NULL DEFAULT 1,
  wave_started_at TEXT,
  window_state TEXT,
  window_opens_at TEXT,
  window_closing_at TEXT,
  window_closes_at TEXT,
  window_changed_at TEXT,
  window_reason TEXT,
  window_log_json TEXT NOT NULL DEFAULT '[]',
  cluster_id TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS lab_events_key ON lab_events (key) WHERE status NOT IN ('merged','dismissed');
CREATE INDEX IF NOT EXISTS lab_events_state ON lab_events (status, window_state, t0);
CREATE TABLE IF NOT EXISTS lab_event_pings (
  id TEXT PRIMARY KEY,
  source TEXT NOT NULL,
  family TEXT NOT NULL,
  external_id TEXT NOT NULL,
  event_id TEXT,
  match_rule TEXT,
  name TEXT,
  name_key TEXT,
  title TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL DEFAULT '',
  author TEXT,
  vendor TEXT,
  source_time TEXT,
  first_seen_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  prev_seen_at TEXT,
  metric_first REAL,
  metric_prev REAL,
  metric_last REAL,
  rank INTEGER,
  velocity_per_h REAL,
  robust_z REAL,
  is_launch INTEGER NOT NULL DEFAULT 0,
  is_baseline INTEGER NOT NULL DEFAULT 0,
  qualifies INTEGER NOT NULL DEFAULT 0,
  qualify_reason TEXT,
  fake_star INTEGER,
  fake_star_json TEXT,
  lexicon INTEGER NOT NULL DEFAULT 0,
  dry INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS lab_event_pings_event ON lab_event_pings (event_id, first_seen_at);
CREATE INDEX IF NOT EXISTS lab_event_pings_source ON lab_event_pings (source, first_seen_at);
CREATE TABLE IF NOT EXISTS lab_event_coverage (
  event_id TEXT NOT NULL,
  post_id TEXT NOT NULL,
  source_id TEXT NOT NULL,
  platform TEXT NOT NULL,
  is_owned INTEGER NOT NULL DEFAULT 0,
  posted_at TEXT NOT NULL,
  hours_since_t0 REAL NOT NULL,
  wave_n INTEGER NOT NULL DEFAULT 1,
  wave_position TEXT NOT NULL,
  creator_rank INTEGER,
  match_rule TEXT NOT NULL,
  confidence TEXT NOT NULL,
  confirmed INTEGER,
  angle TEXT NOT NULL DEFAULT 'other',
  angle_source TEXT NOT NULL DEFAULT 'rule',
  ratio REAL,
  ratio_age_h REAL,
  is_breakout INTEGER NOT NULL DEFAULT 0,
  computed_at TEXT NOT NULL,
  PRIMARY KEY (event_id, post_id)
);
CREATE INDEX IF NOT EXISTS lab_event_coverage_post ON lab_event_coverage (post_id);
CREATE TABLE IF NOT EXISTS lab_brief (
  id TEXT PRIMARY KEY,
  day TEXT NOT NULL UNIQUE,
  composed_at TEXT NOT NULL,
  items_json TEXT NOT NULL DEFAULT '[]',
  dedupe_keys_json TEXT NOT NULL DEFAULT '[]',
  text TEXT NOT NULL DEFAULT '',
  sent_at TEXT,
  sent_via TEXT,
  send_error TEXT,
  dry INTEGER NOT NULL DEFAULT 0
);
-- 06 section 12
CREATE TABLE IF NOT EXISTS lab_predictions (
  id TEXT PRIMARY KEY,
  origin TEXT NOT NULL,
  predictor TEXT NOT NULL,
  predictor_version TEXT NOT NULL,
  kind TEXT NOT NULL,
  script_id TEXT,
  script_version INTEGER,
  post_id TEXT,
  source_id TEXT,
  platform TEXT,
  surface TEXT,
  ref_age_h REAL,
  as_of TEXT NOT NULL,
  frozen_at TEXT NOT NULL,
  backtest_origin TEXT,
  band TEXT,
  score REAL,
  ratio_point REAL, ratio_lo REAL, ratio_hi REAL,
  views_point REAL, views_lo REAL, views_hi REAL,
  p_breakout REAL,
  interval_level REAL,
  baseline_7d_views REAL,
  baseline_n INTEGER,
  inputs_json TEXT NOT NULL DEFAULT '{}',
  note TEXT NOT NULL DEFAULT '',
  payload_sha256 TEXT NOT NULL,
  late INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'frozen',
  void_reason TEXT,
  voided_at TEXT,
  outcome_views_7d REAL,
  outcome_ratio_7d REAL,
  outcome_band TEXT,
  outcome_is_breakout INTEGER,
  outcome_method TEXT,
  hit_band INTEGER,
  in_interval INTEGER,
  abs_log_err REAL,
  test_id TEXT,
  pair_right INTEGER,
  graded_at TEXT,
  dry INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS lab_predictions_post ON lab_predictions (post_id);
CREATE INDEX IF NOT EXISTS lab_predictions_pred ON lab_predictions (predictor, origin, status);
CREATE INDEX IF NOT EXISTS lab_predictions_origin ON lab_predictions (origin, backtest_origin);
CREATE UNIQUE INDEX IF NOT EXISTS lab_predictions_live_prepost ON lab_predictions (predictor, script_id)
  WHERE origin = 'live' AND kind = 'pre_post' AND script_id IS NOT NULL AND status <> 'void';
CREATE UNIQUE INDEX IF NOT EXISTS lab_predictions_live_prepost_post ON lab_predictions (predictor, post_id)
  WHERE origin = 'live' AND kind = 'pre_post' AND script_id IS NULL AND status <> 'void';
CREATE UNIQUE INDEX IF NOT EXISTS lab_predictions_live_early ON lab_predictions (predictor, post_id, ref_age_h)
  WHERE origin = 'live' AND kind = 'early' AND predictor <> 'comment_intent';
CREATE UNIQUE INDEX IF NOT EXISTS lab_predictions_live_ci ON lab_predictions (post_id)
  WHERE origin = 'live' AND predictor = 'comment_intent';
CREATE TABLE IF NOT EXISTS lab_evals (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  predictor TEXT NOT NULL,
  predictor_version TEXT NOT NULL,
  scope TEXT NOT NULL,
  population TEXT NOT NULL,
  platform TEXT NOT NULL,
  window_weeks INTEGER NOT NULL,
  computed_date TEXT NOT NULL,
  computed_at TEXT NOT NULL,
  harness_version TEXT NOT NULL,
  origins INTEGER NOT NULL DEFAULT 0,
  n_posts INTEGER NOT NULL DEFAULT 0,
  n_creators INTEGER NOT NULL DEFAULT 0,
  abstained INTEGER NOT NULL DEFAULT 0,
  n_pairs INTEGER NOT NULL DEFAULT 0,
  pairwise REAL, pairwise_lo REAL, pairwise_hi REAL,
  spearman REAL, spearman_lo REAL, spearman_hi REAL,
  log_mae REAL, log_mae_baseline REAL, mae_delta_lo REAL, mae_delta_hi REAL,
  coverage REAL, coverage_lo REAL, coverage_hi REAL, coverage_n INTEGER NOT NULL DEFAULT 0,
  pr_auc REAL, pr_auc_lo REAL, pr_auc_hi REAL, base_rate REAL, positives INTEGER NOT NULL DEFAULT 0,
  band_hits_json TEXT NOT NULL DEFAULT '{}',
  verdict_forecast TEXT, verdict_ranking TEXT, verdict_classifier TEXT,
  dry INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS lab_evals_key ON lab_evals (predictor, scope, population, platform, window_weeks, computed_date, dry);
CREATE TABLE IF NOT EXISTS lab_retention (
  post_id TEXT NOT NULL,
  platform TEXT NOT NULL,
  source TEXT NOT NULL,
  captured_at TEXT NOT NULL,
  age_h REAL NOT NULL,
  partial INTEGER NOT NULL DEFAULT 0,
  curve_json TEXT,
  views INTEGER,
  avg_view_duration_s REAL,
  avg_view_pct REAL,
  reach INTEGER,
  likes INTEGER,
  comments INTEGER,
  saved INTEGER,
  shares INTEGER,
  total_interactions INTEGER,
  avg_watch_time_ms REAL,
  total_watch_time_ms REAL,
  skip_rate REAL,
  raw_json TEXT NOT NULL DEFAULT '{}',
  dry INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (post_id, captured_at)
);
CREATE INDEX IF NOT EXISTS lab_retention_post ON lab_retention (post_id, age_h);
CREATE TABLE IF NOT EXISTS lab_tests (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  title TEXT NOT NULL,
  hypothesis TEXT NOT NULL DEFAULT '',
  variable TEXT NOT NULL,
  feature_key TEXT,
  expected_sign INTEGER,
  script_id TEXT,
  platform TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'planned',
  arms_json TEXT NOT NULL DEFAULT '[]',
  judge_age_h REAL,
  primary_metric TEXT NOT NULL,
  result TEXT,
  winner_arm TEXT,
  observed_sign INTEGER,
  confirms INTEGER,
  judge_detail_json TEXT NOT NULL DEFAULT '{}',
  notes TEXT NOT NULL DEFAULT '',
  plan_week TEXT,
  started_at TEXT,
  judged_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  dry INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS lab_tests_kind ON lab_tests (kind, status);
CREATE INDEX IF NOT EXISTS lab_tests_feature ON lab_tests (feature_key);
-- 07 7.11, 7.13.3
CREATE TABLE IF NOT EXISTS lab_embeddings (
  subject_kind TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  model TEXT NOT NULL,
  dim INTEGER NOT NULL,
  text_hash TEXT NOT NULL,
  vec BLOB NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (subject_kind, subject_id, model)
);
CREATE INDEX IF NOT EXISTS lab_embeddings_kind_model ON lab_embeddings (subject_kind, model);
CREATE TABLE IF NOT EXISTS lab_comments (
  id TEXT PRIMARY KEY,
  post_id TEXT NOT NULL,
  platform TEXT NOT NULL,
  author_hash TEXT NOT NULL,
  text TEXT NOT NULL,
  likes INTEGER,
  replies INTEGER,
  is_reply INTEGER NOT NULL DEFAULT 0,
  by_creator INTEGER NOT NULL DEFAULT 0,
  creator_heart INTEGER,
  posted_at TEXT,
  captured_at TEXT NOT NULL,
  intent TEXT NOT NULL,
  intent_rule TEXT,
  run_id TEXT
);
CREATE INDEX IF NOT EXISTS lab_comments_post ON lab_comments (post_id);
CREATE INDEX IF NOT EXISTS lab_comments_author ON lab_comments (author_hash);
-- 09 09.6
CREATE TABLE IF NOT EXISTS lab_forecasts (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  subject TEXT NOT NULL,
  frozen_at TEXT NOT NULL,
  inputs_json TEXT NOT NULL,
  predicted REAL NOT NULL,
  lo REAL, hi REAL,
  expected_items INTEGER,
  actual REAL,
  actual_items INTEGER,
  status TEXT NOT NULL DEFAULT 'open',
  graded_at TEXT,
  dry INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS lab_forecasts_kind ON lab_forecasts (kind, frozen_at);
CREATE INDEX IF NOT EXISTS lab_forecasts_subject ON lab_forecasts (subject);
`;

/** 06 2.4(b): a live prediction's frozen fields, post link and grade are written once; live non-dry rows are never deleted. Applied after the tables (part of LAB_DDL). */
export const LEDGER_TRIGGERS_DDL = `
CREATE TRIGGER IF NOT EXISTS lab_predictions_frozen_cols
BEFORE UPDATE OF id, origin, predictor, predictor_version, kind, script_id, script_version, source_id, platform, surface,
  ref_age_h, as_of, frozen_at, backtest_origin, band, score, ratio_point, ratio_lo, ratio_hi, views_point,
  views_lo, views_hi, p_breakout, interval_level, baseline_7d_views, baseline_n, inputs_json, note, payload_sha256
ON lab_predictions WHEN OLD.origin = 'live'
BEGIN SELECT RAISE(ABORT, 'lab_predictions: frozen fields are immutable'); END;
CREATE TRIGGER IF NOT EXISTS lab_predictions_post_once
BEFORE UPDATE OF post_id, late ON lab_predictions
WHEN OLD.origin = 'live' AND OLD.post_id IS NOT NULL
BEGIN SELECT RAISE(ABORT, 'lab_predictions: post link is set once'); END;
CREATE TRIGGER IF NOT EXISTS lab_predictions_grade_once
BEFORE UPDATE OF outcome_views_7d, outcome_ratio_7d, outcome_band, outcome_is_breakout, outcome_method,
  hit_band, in_interval, abs_log_err, test_id, pair_right, graded_at
ON lab_predictions WHEN OLD.origin = 'live' AND OLD.graded_at IS NOT NULL
BEGIN SELECT RAISE(ABORT, 'lab_predictions: a grade is written once'); END;
CREATE TRIGGER IF NOT EXISTS lab_predictions_status_flow
BEFORE UPDATE OF status ON lab_predictions
WHEN OLD.origin = 'live' AND NOT (
  (OLD.status = 'frozen' AND NEW.status IN ('frozen','graded','ungradable','void')) OR
  (OLD.status IN ('graded','ungradable') AND NEW.status IN (OLD.status,'void')) OR
  (OLD.status = 'void' AND NEW.status = 'void'))
BEGIN SELECT RAISE(ABORT, 'lab_predictions: illegal status change'); END;
CREATE TRIGGER IF NOT EXISTS lab_predictions_no_delete
BEFORE DELETE ON lab_predictions WHEN OLD.origin = 'live' AND OLD.dry = 0
BEGIN SELECT RAISE(ABORT, 'lab_predictions: live predictions are never deleted'); END;
`;

/** 09.6 lab_forecasts_lock: no UPDATE once a row is graded or void; a late row may be updated once, to graded. */
export const LAB_FORECASTS_TRIGGER_DDL = `
CREATE TRIGGER IF NOT EXISTS lab_forecasts_lock
BEFORE UPDATE ON lab_forecasts
WHEN OLD.status IN ('graded','void') OR (OLD.status = 'late' AND NEW.status <> 'graded')
BEGIN SELECT RAISE(ABORT, 'lab_forecasts: a graded forecast is immutable'); END;
`;
/* ── end Content Lab v2 tables ── */

export const LAB_DDL = `
CREATE TABLE IF NOT EXISTS lab_sources (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  platform TEXT NOT NULL,
  handle TEXT NOT NULL,
  url TEXT,
  display_name TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT 'active',
  added_at TEXT NOT NULL,
  reason TEXT,
  score REAL
);
CREATE UNIQUE INDEX IF NOT EXISTS lab_sources_handle ON lab_sources (platform, handle);
CREATE TABLE IF NOT EXISTS lab_creator_snapshots (
  source_id TEXT NOT NULL,
  captured_at TEXT NOT NULL,
  followers INTEGER,
  following INTEGER,
  post_count INTEGER
);
CREATE INDEX IF NOT EXISTS lab_creator_snapshots_source ON lab_creator_snapshots (source_id, captured_at);
CREATE TABLE IF NOT EXISTS lab_posts (
  id TEXT PRIMARY KEY,
  source_id TEXT NOT NULL,
  platform TEXT NOT NULL,
  external_id TEXT NOT NULL,
  url TEXT NOT NULL,
  posted_at TEXT,
  caption TEXT NOT NULL DEFAULT '',
  content_type TEXT NOT NULL DEFAULT 'video',
  thumbnail_url TEXT,
  media_url TEXT,
  duration_s REAL,
  is_owned INTEGER NOT NULL DEFAULT 0,
  UNIQUE (platform, external_id)
);
CREATE INDEX IF NOT EXISTS lab_posts_source ON lab_posts (source_id, posted_at);
CREATE TABLE IF NOT EXISTS lab_post_metrics (
  post_id TEXT NOT NULL,
  captured_at TEXT NOT NULL,
  views INTEGER,
  likes INTEGER,
  comments INTEGER,
  shares INTEGER,
  saves INTEGER
);
CREATE INDEX IF NOT EXISTS lab_post_metrics_post ON lab_post_metrics (post_id, captured_at);
CREATE TABLE IF NOT EXISTS lab_transcripts (
  post_id TEXT PRIMARY KEY,
  text TEXT NOT NULL DEFAULT '',
  word_count INTEGER NOT NULL DEFAULT 0,
  wpm REAL,
  provider TEXT NOT NULL DEFAULT '',
  cost_usd REAL NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'none'
);
CREATE TABLE IF NOT EXISTS lab_dissections (
  post_id TEXT PRIMARY KEY,
  format TEXT NOT NULL,
  hook_form TEXT NOT NULL,
  emotional_trigger TEXT NOT NULL,
  hook_channel TEXT NOT NULL,
  hook_text TEXT NOT NULL DEFAULT '',
  first_frame_text TEXT NOT NULL DEFAULT '',
  structure_type TEXT NOT NULL,
  beats_json TEXT NOT NULL DEFAULT '[]',
  pacing TEXT NOT NULL,
  pillar TEXT NOT NULL,
  cta_type TEXT NOT NULL,
  tension_json TEXT,
  confidences_json TEXT NOT NULL DEFAULT '{}',
  quality_notes TEXT NOT NULL DEFAULT '',
  model_id TEXT NOT NULL,
  prompt_version TEXT NOT NULL,
  analysed_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS lab_post_topics (
  post_id TEXT NOT NULL,
  topic TEXT NOT NULL,
  angle TEXT NOT NULL DEFAULT '',
  is_primary INTEGER NOT NULL DEFAULT 0,
  cluster_id TEXT,
  embedding BLOB
);
CREATE INDEX IF NOT EXISTS lab_post_topics_post ON lab_post_topics (post_id);
CREATE INDEX IF NOT EXISTS lab_post_topics_cluster ON lab_post_topics (cluster_id);
CREATE TABLE IF NOT EXISTS lab_measures (
  post_id TEXT PRIMARY KEY,
  depth REAL,
  breakout REAL,
  breakout_lo REAL,
  breakout_hi REAL,
  baseline_n INTEGER NOT NULL DEFAULT 0,
  baseline_confidence TEXT NOT NULL DEFAULT 'low',
  reach REAL,
  velocity REAL,
  trajectory TEXT,
  quadrant TEXT,
  is_breakout INTEGER NOT NULL DEFAULT 0,
  computed_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS lab_clusters (
  id TEXT PRIMARY KEY,
  label TEXT NOT NULL,
  centroid BLOB,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS lab_trend_snapshots (
  cluster_id TEXT NOT NULL,
  day TEXT NOT NULL,
  posts INTEGER NOT NULL DEFAULT 0,
  breakouts INTEGER NOT NULL DEFAULT 0,
  creators INTEGER NOT NULL DEFAULT 0,
  platforms INTEGER NOT NULL DEFAULT 0,
  feed_mentions INTEGER NOT NULL DEFAULT 0,
  source_count INTEGER NOT NULL DEFAULT 0,
  momentum REAL,
  timing TEXT,
  PRIMARY KEY (cluster_id, day)
);
CREATE TABLE IF NOT EXISTS lab_trend_items (
  id TEXT PRIMARY KEY,
  feed TEXT NOT NULL,
  url TEXT NOT NULL,
  title TEXT NOT NULL,
  score REAL,
  comments INTEGER,
  sources_json TEXT NOT NULL DEFAULT '[]',
  source_count INTEGER NOT NULL DEFAULT 1,
  cluster_id TEXT,
  seen_at TEXT NOT NULL,
  acted_on_at TEXT
);
CREATE TABLE IF NOT EXISTS lab_profiles (
  id TEXT PRIMARY KEY,
  format_family TEXT NOT NULL,
  label TEXT NOT NULL DEFAULT '',
  version INTEGER NOT NULL,
  window_days INTEGER NOT NULL DEFAULT 90,
  posts_n INTEGER NOT NULL DEFAULT 0,
  breakouts_n INTEGER NOT NULL DEFAULT 0,
  features_json TEXT NOT NULL DEFAULT '[]',
  strand_json TEXT NOT NULL DEFAULT '[]',
  stats_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS lab_profiles_family ON lab_profiles (format_family, version);
CREATE TABLE IF NOT EXISTS lab_ideas (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  cluster_id TEXT,
  angle TEXT NOT NULL DEFAULT '',
  tension_json TEXT,
  format_family TEXT NOT NULL,
  strand_json TEXT NOT NULL DEFAULT '[]',
  hook_recs_json TEXT NOT NULL DEFAULT '[]',
  evidence_json TEXT NOT NULL DEFAULT '[]',
  timing TEXT,
  fit REAL NOT NULL DEFAULT 0,
  thin INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'open',
  dismiss_reason TEXT,
  created_at TEXT NOT NULL,
  acted_on_at TEXT
);
CREATE TABLE IF NOT EXISTS lab_scripts (
  id TEXT PRIMARY KEY,
  idea_id TEXT,
  title TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'draft',
  published_post_id TEXT,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS lab_script_versions (
  script_id TEXT NOT NULL,
  version INTEGER NOT NULL,
  body_json TEXT NOT NULL,
  gate_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  edit_diff TEXT,
  PRIMARY KEY (script_id, version)
);
CREATE TABLE IF NOT EXISTS lab_outcomes (
  script_id TEXT NOT NULL,
  post_id TEXT NOT NULL,
  captured_at TEXT NOT NULL,
  breakout REAL,
  depth REAL
);
CREATE TABLE IF NOT EXISTS lab_priors (
  key TEXT PRIMARY KEY,
  value_json TEXT NOT NULL,
  n INTEGER NOT NULL DEFAULT 0,
  supersedes_default INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS lab_runs (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  apify_run_id TEXT,
  dataset_id TEXT,
  actor TEXT,
  input_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL,
  items INTEGER,
  cost_usd REAL,
  dry INTEGER NOT NULL DEFAULT 0,
  started_at TEXT NOT NULL,
  finished_at TEXT,
  error TEXT
);
CREATE INDEX IF NOT EXISTS lab_runs_started ON lab_runs (started_at);
CREATE TABLE IF NOT EXISTS lab_spend (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  run_id TEXT,
  provider TEXT NOT NULL,
  units REAL NOT NULL DEFAULT 0,
  est_usd REAL,
  actual_usd REAL,
  dry INTEGER NOT NULL DEFAULT 0,
  at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS lab_spend_at ON lab_spend (provider, at);
${LAB_V2_DDL}${LEDGER_TRIGGERS_DDL}${LAB_FORECASTS_TRIGGER_DDL}`;
/* ── end Content Lab tables ── */

/* ── Content Lab v2 migration (S0; 08.11.2). The ONLY migrateLabV2 and the ONLY place a
   v1 lab table gains a column. A section that needs another column adds a row here. ── */

/** Raised when a schema change ships; lab_meta_version() must equal it (health `db.schema`). */
export const LAB_SCHEMA_VERSION = 2;

/** Every column v2 adds to a v1 lab table, one list for all sections (owner = spec section). */
export const LAB_V2_COLUMNS: { table: string; column: string; ddl: string; owner: '04' | '05' | '07' | '08' | '09' }[] = [
  // 04 section 8
  { table: 'lab_post_metrics', column: 'age_h', ddl: 'REAL', owner: '04' },
  { table: 'lab_post_metrics', column: 'target_age_h', ddl: 'REAL', owner: '04' },
  { table: 'lab_post_metrics', column: 'read_kind', ddl: "TEXT NOT NULL DEFAULT 'cycle'", owner: '04' },
  { table: 'lab_post_metrics', column: 'quotes', ddl: 'INTEGER', owner: '04' },
  { table: 'lab_posts', column: 'epoch', ddl: "TEXT NOT NULL DEFAULT 'current'", owner: '04' },
  { table: 'lab_posts', column: 'exo_reason', ddl: 'TEXT', owner: '04' },
  { table: 'lab_posts', column: 'exo_note', ddl: 'TEXT', owner: '04' },
  { table: 'lab_posts', column: 'exo_at', ddl: 'TEXT', owner: '04' },
  { table: 'lab_measures', column: 'measure_version', ddl: "TEXT NOT NULL DEFAULT 'measure-v1'", owner: '04' },
  { table: 'lab_measures', column: 'metric_basis', ddl: "TEXT NOT NULL DEFAULT 'views'", owner: '04' },
  { table: 'lab_measures', column: 'judged_age_h', ddl: 'REAL', owner: '04' },
  { table: 'lab_measures', column: 'basis_at_age', ddl: 'REAL', owner: '04' },
  { table: 'lab_measures', column: 'baseline_log', ddl: 'REAL', owner: '04' },
  { table: 'lab_measures', column: 'baseline_var', ddl: 'REAL', owner: '04' },
  { table: 'lab_measures', column: 'prior_log', ddl: 'REAL', owner: '04' },
  { table: 'lab_measures', column: 'shrink_b', ddl: 'REAL', owner: '04' },
  { table: 'lab_measures', column: 'baseline_source', ddl: 'TEXT', owner: '04' },
  { table: 'lab_measures', column: 'robust_z', ddl: 'REAL', owner: '04' },
  { table: 'lab_measures', column: 'volatility', ddl: 'REAL', owner: '04' },
  { table: 'lab_measures', column: 'volatility_class', ddl: 'TEXT', owner: '04' },
  { table: 'lab_measures', column: 'max_over_median', ddl: 'REAL', owner: '04' },
  { table: 'lab_measures', column: 'niche_ratio', ddl: 'REAL', owner: '04' },
  { table: 'lab_measures', column: 'dual_high', ddl: 'INTEGER NOT NULL DEFAULT 0', owner: '04' },
  { table: 'lab_measures', column: 'trajectory_class', ddl: 'TEXT', owner: '04' },
  { table: 'lab_measures', column: 'peak_day_share', ddl: 'REAL', owner: '04' },
  { table: 'lab_measures', column: 'exo_auto', ddl: 'INTEGER NOT NULL DEFAULT 0', owner: '04' },
  { table: 'lab_measures', column: 'shares_per_view', ddl: 'REAL', owner: '04' },
  { table: 'lab_measures', column: 'saves_per_view', ddl: 'REAL', owner: '04' },
  { table: 'lab_measures', column: 'x_action_score', ddl: 'REAL', owner: '04' },
  { table: 'lab_measures', column: 'followers_as_of', ddl: 'TEXT', owner: '04' },
  { table: 'lab_measures', column: 'ages_json', ddl: "TEXT NOT NULL DEFAULT '{}'", owner: '04' },
  { table: 'lab_profiles', column: 'method', ddl: "TEXT NOT NULL DEFAULT 'lift_wilson_v1'", owner: '04' },
  { table: 'lab_profiles', column: 'controls_n', ddl: 'INTEGER NOT NULL DEFAULT 0', owner: '04' },
  { table: 'lab_profiles', column: 'fdr_q', ddl: 'REAL', owner: '04' },
  // 05 section 16
  { table: 'lab_ideas', column: 'event_id', ddl: 'TEXT', owner: '05' },
  { table: 'lab_ideas', column: 'launch_angle', ddl: 'TEXT', owner: '05' },
  // 07 7.3.6, 7.14.4
  { table: 'lab_ideas', column: 'source', ddl: "TEXT NOT NULL DEFAULT 'sync'", owner: '07' },
  { table: 'lab_ideas', column: 'batch_id', ddl: 'TEXT', owner: '07' },
  { table: 'lab_ideas', column: 'seed_id', ddl: 'TEXT', owner: '07' },
  { table: 'lab_ideas', column: 'notes', ddl: 'TEXT', owner: '07' },
  { table: 'lab_ideas', column: 'hook_form', ddl: 'TEXT', owner: '07' },
  { table: 'lab_ideas', column: 'hook_trigger', ddl: 'TEXT', owner: '07' },
  { table: 'lab_ideas', column: 'vs_probability', ddl: 'REAL', owner: '07' },
  { table: 'lab_ideas', column: 'mmr_rank', ddl: 'INTEGER', owner: '07' },
  { table: 'lab_ideas', column: 'grounding_json', ddl: "TEXT NOT NULL DEFAULT '{}'", owner: '07' },
  { table: 'lab_ideas', column: 'prompt_version', ddl: 'TEXT', owner: '07' },
  { table: 'lab_dissections', column: 'frames_used', ddl: 'INTEGER NOT NULL DEFAULT 0', owner: '07' },
  { table: 'lab_dissections', column: 'on_screen_text_density', ddl: 'INTEGER', owner: '07' },
  { table: 'lab_dissections', column: 'face_first_frame', ddl: 'INTEGER', owner: '07' },
  { table: 'lab_dissections', column: 'sheet_path', ddl: 'TEXT', owner: '07' },
  // 08.6.4
  { table: 'lab_spend', column: 'line', ddl: 'TEXT', owner: '08' },
  { table: 'lab_spend', column: 'reserved_usd', ddl: 'REAL', owner: '08' },
  { table: 'lab_spend', column: 'actual_source', ddl: 'TEXT', owner: '08' },
  { table: 'lab_spend', column: 'events_json', ddl: 'TEXT', owner: '08' },
  { table: 'lab_spend', column: 'settled_at', ddl: 'TEXT', owner: '08' },
  { table: 'lab_runs', column: 'line', ddl: 'TEXT', owner: '08' },
  { table: 'lab_runs', column: 'job', ddl: 'TEXT', owner: '08' },
  { table: 'lab_runs', column: 'scheduled', ddl: 'INTEGER NOT NULL DEFAULT 0', owner: '08' },
  // 09 09.6
  { table: 'lab_runs', column: 'batch_id', ddl: 'TEXT', owner: '09' },
  { table: 'lab_runs', column: 'progress_json', ddl: 'TEXT', owner: '09' },
];

/** Indexes on LAB_V2_COLUMNS columns: created after the column exists, so not in LAB_DDL. */
export const LAB_V2_INDEXES = [
  'CREATE INDEX IF NOT EXISTS lab_post_metrics_age ON lab_post_metrics (post_id, age_h)', // 04
  'CREATE INDEX IF NOT EXISTS lab_ideas_batch ON lab_ideas (batch_id)', // 07
  'CREATE INDEX IF NOT EXISTS lab_runs_batch ON lab_runs (batch_id)', // 09
] as const;

/** The prior `schema_version` ({ v }); absent = 1 (a v1 database never migrated). */
export function lab_meta_version(db: InstanceType<typeof Database>): number {
  const row = db.prepare("SELECT value_json FROM lab_priors WHERE key = 'schema_version'").get() as { value_json: string } | undefined;
  if (!row) return 1;
  try {
    const v = Number((JSON.parse(row.value_json) as { v?: unknown }).v);
    return Number.isFinite(v) ? v : 1;
  } catch {
    return 1;
  }
}

/**
 * The one v1 to v2 migration (08.11.2). Idempotent: every ADD COLUMN is guarded by
 * PRAGMA table_info, every backfill only touches rows still missing the value, and a second
 * run changes nothing. Needs LAB_DDL applied first (getDb, the sample world and labStore()
 * all do). Nothing is deleted. Ends by writing prior `schema_version` { v: LAB_SCHEMA_VERSION }.
 */
export function migrateLabV2(db: InstanceType<typeof Database>): void {
  const has = new Map<string, Set<string>>();
  const cols = (table: string) => {
    let set = has.get(table);
    if (!set) {
      set = new Set((db.prepare(`PRAGMA table_info(${table})`).all() as { name: string }[]).map((c) => c.name));
      has.set(table, set);
    }
    return set;
  };
  if (LAB_V2_COLUMNS.every((c) => cols(c.table).has(c.column)) && lab_meta_version(db) >= LAB_SCHEMA_VERSION) return;
  // IMMEDIATE takes the write lock first and the columns are re-read under it: `next build` runs
  // several prerender workers on one database, and a check made outside the lock let two of them
  // ALTER the same column ("duplicate column name: age_h").
  db.transaction(() => {
    has.clear();
    const missing = LAB_V2_COLUMNS.filter((c) => !cols(c.table).has(c.column));
    const versionNow = lab_meta_version(db);
    if (missing.length === 0 && versionNow >= LAB_SCHEMA_VERSION) return;
    for (const c of missing) {
      db.exec(`ALTER TABLE ${c.table} ADD COLUMN ${c.column} ${c.ddl}`);
      cols(c.table).add(c.column);
    }
    for (const sql of LAB_V2_INDEXES) db.exec(sql);
    // 04: ages of existing reads (only when the column was just added; negative ages stay null)
    if (missing.some((c) => c.table === 'lab_post_metrics' && c.column === 'age_h')) {
      db.exec(`UPDATE lab_post_metrics SET age_h = (
          SELECT (julianday(lab_post_metrics.captured_at) - julianday(p.posted_at)) * 24 FROM lab_posts p
          WHERE p.id = lab_post_metrics.post_id AND p.posted_at IS NOT NULL
            AND julianday(lab_post_metrics.captured_at) >= julianday(p.posted_at))
        WHERE age_h IS NULL`);
    }
    // 08.6.4: v1 spend rows were all sync work on Apify
    db.exec("UPDATE lab_spend SET line = CASE provider WHEN 'apify' THEN 'apify.sync' ELSE provider END WHERE line IS NULL");
    db.exec("UPDATE lab_spend SET actual_source = 'dry' WHERE dry = 1 AND actual_source IS NULL");
    db.exec("UPDATE lab_spend SET actual_source = 'provider', settled_at = at WHERE dry = 0 AND actual_usd IS NOT NULL AND actual_source IS NULL");
    if (versionNow < LAB_SCHEMA_VERSION) {
      db.prepare('INSERT OR REPLACE INTO lab_priors (key, value_json, n, supersedes_default, updated_at) VALUES (?, ?, 0, 0, ?)')
        .run('schema_version', JSON.stringify({ v: LAB_SCHEMA_VERSION }), new Date().toISOString());
    }
  }).immediate();
}
/* ── end Content Lab v2 migration ── */

/** Databases created before the hierarchy build lack these columns. */
function migrateAgentsTable(db: InstanceType<typeof Database>): void {
  const columns = new Set(
    (db.pragma('table_info(agents)') as { name: string }[]).map((c) => c.name),
  );
  if (!columns.has('parent_id')) db.exec('ALTER TABLE agents ADD COLUMN parent_id TEXT');
  if (!columns.has('instance')) db.exec("ALTER TABLE agents ADD COLUMN instance TEXT NOT NULL DEFAULT 'builtin'");
}

/** Databases created before the funnel-space build lack these columns. */
/** 2026-09-15: the canvas stores a target agent-owned ratio per workflow. */
function migrateWorkflowsTable(db: InstanceType<typeof Database>): void {
  const columns = new Set((db.prepare('PRAGMA table_info(workflows)').all() as { name: string }[]).map((c) => c.name));
  if (!columns.has('ratio_target')) db.exec('ALTER TABLE workflows ADD COLUMN ratio_target REAL');
}

function migrateFunnelContactsTable(db: InstanceType<typeof Database>): void {
  const columns = new Set(
    (db.pragma('table_info(funnel_contacts)') as { name: string }[]).map((c) => c.name),
  );
  if (!columns.has('relationship')) db.exec("ALTER TABLE funnel_contacts ADD COLUMN relationship TEXT NOT NULL DEFAULT 'warm'");
  if (!columns.has('likelihood')) db.exec('ALTER TABLE funnel_contacts ADD COLUMN likelihood INTEGER NOT NULL DEFAULT 50');
  if (!columns.has('email')) db.exec('ALTER TABLE funnel_contacts ADD COLUMN email TEXT');
  if (!columns.has('phone')) db.exec('ALTER TABLE funnel_contacts ADD COLUMN phone TEXT');
  // dossier identity (Round 15) — the human behind the deal
  for (const col of ['person', 'company', 'role', 'linkedin']) {
    if (!columns.has(col)) db.exec(`ALTER TABLE funnel_contacts ADD COLUMN ${col} TEXT`);
  }
}

/**
 * The tasks overhaul (docs/TASKS-REDESIGN.md) gave a task a body, a due date,
 * a priority, an owner lane and a horizon. Databases created before it have
 * none of them; every column carries a default so existing rows stay valid.
 * `agent_id` was NOT NULL and cannot be widened in place, so operator-owned
 * tasks on an old database write the sentinel '' and read back as null.
 */
function migrateAgentTasksTable(db: InstanceType<typeof Database>): void {
  const columns = new Set(
    (db.pragma('table_info(agent_tasks)') as { name: string }[]).map((c) => c.name),
  );
  if (columns.size === 0) return;
  if (!columns.has('detail')) db.exec("ALTER TABLE agent_tasks ADD COLUMN detail TEXT NOT NULL DEFAULT ''");
  if (!columns.has('due_at')) db.exec('ALTER TABLE agent_tasks ADD COLUMN due_at TEXT');
  if (!columns.has('priority')) db.exec("ALTER TABLE agent_tasks ADD COLUMN priority TEXT NOT NULL DEFAULT 'normal'");
  if (!columns.has('owner_kind')) db.exec("ALTER TABLE agent_tasks ADD COLUMN owner_kind TEXT NOT NULL DEFAULT 'agent'");
  if (!columns.has('horizon')) db.exec('ALTER TABLE agent_tasks ADD COLUMN horizon TEXT');
  if (!columns.has('run_id')) db.exec('ALTER TABLE agent_tasks ADD COLUMN run_id TEXT');
}

// Skills gained a `markdown` (SKILL.md) column after first ship. Add it, and
// clear the stale rows so the re-seed backfills each skill's doc.
function migrateSkillsTable(db: InstanceType<typeof Database>): void {
  const columns = new Set((db.pragma('table_info(skills)') as { name: string }[]).map((c) => c.name));
  if (columns.size > 0 && !columns.has('markdown')) {
    db.exec("ALTER TABLE skills ADD COLUMN markdown TEXT NOT NULL DEFAULT ''");
    db.exec('DELETE FROM skills');
  }
}

/** Tables whose pages were removed (SPEC S12): Reference Model and Personas. */
export const RETIRED_TABLES = ['domains', 'personas'] as const;

export type RetireOutcome = {
  table: string;
  action: 'absent' | 'dropped' | 'exported' | 'kept';
  rows: number;
  /** The JSON export holding the rows (written now, or an identical earlier one). */
  file?: string;
  reason?: string;
};

function localDate(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

function rowKeys(rows: Record<string, unknown>[]): string {
  return rows
    .map((r) => (r.id !== undefined ? String(r.id) : JSON.stringify(r)))
    .sort()
    .join('\n');
}

/**
 * Retire tables nothing reads any more (SPEC S12 §5.3): never lose rows
 * silently, never leave orphan tables behind. Per table: absent → skip;
 * empty → DROP; with rows → export to `<db dir>/retired/<table>-<date>.json`
 * (reusing an earlier export that holds the same ids, so a restored old
 * snapshot does not duplicate files), read the file back, check the row
 * count, then DROP in a transaction. Any failure keeps the table and logs one
 * warning; this never throws, so the app always opens. An in-memory DB has
 * no directory to export to and is simply dropped.
 */
export function retireTables(
  db: InstanceType<typeof Database>,
  dbPath: string,
  opts: { tables?: readonly string[]; now?: Date; warn?: (msg: string) => void } = {},
): RetireOutcome[] {
  const { tables = RETIRED_TABLES, now = new Date(), warn = (m: string) => console.warn(m) } = opts;
  const inMemory = dbPath === ':memory:' || dbPath === '';
  const out: RetireOutcome[] = [];
  for (const table of tables) {
    try {
      const exists = db.prepare("SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?").get(table);
      if (!exists) {
        out.push({ table, action: 'absent', rows: 0 });
        continue;
      }
      const n = (db.prepare(`SELECT COUNT(*) AS n FROM "${table}"`).get() as { n: number }).n;
      const drop = db.transaction(() => db.exec(`DROP TABLE IF EXISTS "${table}"`));
      if (n === 0 || inMemory) {
        drop();
        out.push({ table, action: 'dropped', rows: n });
        continue;
      }
      const rows = db.prepare(`SELECT * FROM "${table}" ORDER BY rowid`).all() as Record<string, unknown>[];
      const keys = rowKeys(rows);
      const dir = join(dirname(dbPath), 'retired');
      fs.mkdirSync(dir, { recursive: true });
      const readRows = (file: string): Record<string, unknown>[] | null => {
        try {
          const parsed = JSON.parse(fs.readFileSync(file, 'utf8')) as { table?: string; rows?: unknown };
          return parsed.table === table && Array.isArray(parsed.rows) ? (parsed.rows as Record<string, unknown>[]) : null;
        } catch {
          return null;
        }
      };
      // An earlier export of exactly these rows (any date) is reused.
      let file = fs
        .readdirSync(dir)
        .filter((f) => f.startsWith(`${table}-`) && f.endsWith('.json'))
        .map((f) => join(dir, f))
        .find((f) => {
          const prev = readRows(f);
          return prev !== null && prev.length === n && rowKeys(prev) === keys;
        });
      if (!file) {
        const stem = `${table}-${localDate(now)}`;
        let candidate = join(dir, `${stem}.json`);
        for (let i = 2; fs.existsSync(candidate); i++) candidate = join(dir, `${stem}-${i}.json`);
        const payload = { table, retiredOn: now.toISOString(), reason: 'spec 12: page removed', rows };
        fs.writeFileSync(candidate, `${JSON.stringify(payload, null, 2)}\n`);
        file = candidate;
      }
      const check = readRows(file);
      if (!check || check.length !== n) throw new Error(`export ${file} did not read back ${n} rows`);
      drop();
      out.push({ table, action: 'exported', rows: n, file });
    } catch (err) {
      const reason = err instanceof Error ? err.message : String(err);
      warn(`[db] retire ${table} skipped: ${reason}`);
      out.push({ table, action: 'kept', rows: -1, reason });
    }
  }
  return out;
}

type AgentRow = {
  id: string;
  department_id: string;
  name: string;
  role: string;
  status: string;
  tier: string;
  description: string;
  model: string;
  tools: string;
  parent_id: string | null;
  instance: string;
};

function rowToAgent(row: AgentRow): Agent {
  return AgentSchema.parse({
    id: row.id,
    departmentId: row.department_id,
    name: row.name,
    role: row.role,
    status: row.status,
    tier: row.tier,
    description: row.description,
    model: row.model,
    tools: JSON.parse(row.tools),
    parentId: row.parent_id,
    instance: row.instance,
  });
}

export function openDb(path: string) {
  const db = new Database(path);
  db.pragma('journal_mode = WAL');
  db.exec(DDL);
  db.exec(LAB_DDL); // Content Lab (additive)
  migrateLabV2(db); // Content Lab v2 columns (idempotent, 08.11.2)
  migrateAgentsTable(db);
  migrateFunnelContactsTable(db);
  migrateSkillsTable(db);
  migrateAgentTasksTable(db);
  migrateWorkflowsTable(db);
  migrateFounderOs(db); // FounderOS UI merge tables (additive)
  retireTables(db, path);

  const departments = {
    all(): Department[] {
      return db
        .prepare('SELECT * FROM departments ORDER BY "order"')
        .all()
        .map((r) => DepartmentSchema.parse(r));
    },
    insert(d: Department): void {
      db.prepare(
        'INSERT OR REPLACE INTO departments (id, name, slug, tagline, color, "order") VALUES (?, ?, ?, ?, ?, ?)',
      ).run(d.id, d.name, d.slug, d.tagline, d.color, d.order);
    },
    deleteWhereIdNotIn(ids: string[]): void {
      // An empty keep-list means "keep nothing": SQL rejects `NOT IN ()`.
      if (ids.length === 0) { db.prepare('DELETE FROM departments').run(); return; }
      const placeholders = ids.map(() => '?').join(', ');
      db.prepare(`DELETE FROM departments WHERE id NOT IN (${placeholders})`).run(...ids);
    },
  };

  const agents = {
    all(): Agent[] {
      return (db.prepare('SELECT * FROM agents ORDER BY tier, name').all() as AgentRow[]).map(rowToAgent);
    },
    byDepartment(departmentId: string): Agent[] {
      return (
        db
          .prepare('SELECT * FROM agents WHERE department_id = ? ORDER BY tier, name')
          .all(departmentId) as AgentRow[]
      ).map(rowToAgent);
    },
    get(id: string): Agent | null {
      const r = db.prepare('SELECT * FROM agents WHERE id = ?').get(id) as AgentRow | undefined;
      return r ? rowToAgent(r) : null;
    },
    /** Failover (F9): move a stored agent onto another model. False when the id is unknown. */
    setModel(id: string, model: string): boolean {
      return db.prepare('UPDATE agents SET model = ? WHERE id = ?').run(model, id).changes > 0;
    },
    insert(a: Agent): void {
      db.prepare(
        'INSERT OR REPLACE INTO agents (id, department_id, name, role, status, tier, description, model, tools, parent_id, instance) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)',
      ).run(
        a.id, a.departmentId, a.name, a.role, a.status, a.tier, a.description, a.model,
        JSON.stringify(a.tools), a.parentId, a.instance,
      );
    },
    deleteWhereIdNotIn(ids: string[]): void {
      // An empty keep-list means "keep nothing": SQL rejects `NOT IN ()`.
      if (ids.length === 0) { db.prepare('DELETE FROM agents').run(); return; }
      const placeholders = ids.map(() => '?').join(', ');
      db.prepare(`DELETE FROM agents WHERE id NOT IN (${placeholders})`).run(...ids);
    },
  };

  const tools = {
    all(): Tool[] {
      return db
        .prepare('SELECT * FROM tools ORDER BY category, name')
        .all()
        .map((r) => ToolSchema.parse(r));
    },
    insert(t: Tool): void {
      db.prepare(
        'INSERT OR REPLACE INTO tools (id, name, category, status, color, description) VALUES (?, ?, ?, ?, ?, ?)',
      ).run(t.id, t.name, t.category, t.status, t.color, t.description);
    },
  };

  const roadmap = {
    all(): RoadmapItem[] {
      return db
        .prepare('SELECT * FROM roadmap_items ORDER BY quarter, title')
        .all()
        .map((r: any) =>
          RoadmapItemSchema.parse({
            id: r.id,
            title: r.title,
            quarter: r.quarter,
            status: r.status,
            departmentId: r.department_id,
            description: r.description,
          }),
        );
    },
    insert(item: RoadmapItem): void {
      db.prepare(
        'INSERT OR REPLACE INTO roadmap_items (id, title, quarter, status, department_id, description) VALUES (?, ?, ?, ?, ?, ?)',
      ).run(item.id, item.title, item.quarter, item.status, item.departmentId, item.description);
    },
  };

  const metrics = {
    all(): Metric[] {
      return db
        .prepare('SELECT * FROM metrics ORDER BY label')
        .all()
        .map((r) => MetricSchema.parse(r));
    },
    insert(m: Metric): void {
      db.prepare(
        'INSERT OR REPLACE INTO metrics (id, key, label, value, unit, delta, period) VALUES (?, ?, ?, ?, ?, ?, ?)',
      ).run(m.id, m.key, m.label, m.value, m.unit, m.delta, m.period);
    },
  };

  const phases = {
    all(): Phase[] {
      return db
        .prepare('SELECT * FROM phases ORDER BY number')
        .all()
        .map((r: any) => PhaseSchema.parse({ ...r, items: JSON.parse(r.items) }));
    },
    insert(p: Phase): void {
      db.prepare('INSERT OR REPLACE INTO phases (id, number, title, items) VALUES (?, ?, ?, ?)').run(
        p.id,
        p.number,
        p.title,
        JSON.stringify(p.items),
      );
    },
  };

  const rowToRun = (r: any): AgentRun =>
    AgentRunSchema.parse({
      id: r.id,
      agentId: r.agent_id,
      startedAt: r.started_at,
      finishedAt: r.finished_at,
      ok: Boolean(r.ok),
      summary: r.summary,
    });

  const agentRuns = {
    byAgent(agentId: string): AgentRun[] {
      return db
        .prepare('SELECT * FROM agent_runs WHERE agent_id = ? ORDER BY started_at DESC')
        .all(agentId)
        .map(rowToRun);
    },
    get(id: string): AgentRun | null {
      const r = db.prepare('SELECT * FROM agent_runs WHERE id = ?').get(id);
      return r ? rowToRun(r) : null;
    },
    recent(limit: number): AgentRun[] {
      return db
        .prepare('SELECT * FROM agent_runs ORDER BY started_at DESC, rowid DESC LIMIT ?')
        .all(limit)
        .map(rowToRun);
    },
    /** Lifetime run count, uncapped (SPEC C20): not `recent(n).length`, which stops at n. */
    count(): number {
      return (db.prepare('SELECT COUNT(*) AS n FROM agent_runs').get() as { n: number }).n;
    },
    /** The newest run (same order as `recent`), or null on an install with none. Feeds "last run N ago". */
    latest(): AgentRun | null {
      const r = db.prepare('SELECT * FROM agent_runs ORDER BY started_at DESC, rowid DESC LIMIT 1').get();
      return r ? rowToRun(r) : null;
    },
    insert(run: AgentRun): void {
      db.prepare(
        'INSERT OR REPLACE INTO agent_runs (id, agent_id, started_at, finished_at, ok, summary) VALUES (?, ?, ?, ?, ?, ?)',
      ).run(run.id, run.agentId, run.startedAt, run.finishedAt, run.ok ? 1 : 0, run.summary);
    },
  };

  const rowToMessage = (r: any): AgentMessage =>
    AgentMessageSchema.parse({
      id: r.id,
      agentId: r.agent_id,
      role: r.role,
      content: r.content,
      toolCalls: JSON.parse(r.tool_calls || '[]'),
      createdAt: r.created_at,
    });

  const agentMessages = {
    /** One opaque ID per persisted rolling thread, independent of MCP clients.
     * Clearing/replacing the history starts a new conversation; reopening this
     * database or retrying a failed turn keeps the same ID.
     */
    sessionId(agentId: string): string {
      return db.transaction(() => {
        const first = db.prepare('SELECT id FROM agent_messages WHERE agent_id = ? ORDER BY created_at ASC, rowid ASC LIMIT 1').get(agentId) as { id: string } | undefined;
        if (!first) throw new Error('A chat session needs a conversation');
        const saved = db.prepare('SELECT first_message_id, session_id FROM agent_chat_sessions WHERE agent_id = ?').get(agentId) as { first_message_id: string; session_id: string } | undefined;
        if (saved?.first_message_id === first.id) return saved.session_id;
        const id = randomUUID();
        db.prepare('INSERT OR REPLACE INTO agent_chat_sessions (agent_id, first_message_id, session_id) VALUES (?, ?, ?)').run(agentId, first.id, id);
        return id;
      }).immediate();
    },
    insert(m: AgentMessage): void {
      const parsed = AgentMessageSchema.parse(m);
      db.prepare(
        'INSERT OR REPLACE INTO agent_messages (id, agent_id, role, content, tool_calls, created_at) VALUES (?, ?, ?, ?, ?, ?)',
      ).run(parsed.id, parsed.agentId, parsed.role, parsed.content, JSON.stringify(parsed.toolCalls), parsed.createdAt);
    },
    /** Full conversation for one agent, oldest → newest (ready to replay). */
    byAgent(agentId: string): AgentMessage[] {
      return db
        .prepare('SELECT * FROM agent_messages WHERE agent_id = ? ORDER BY created_at ASC, rowid ASC')
        .all(agentId)
        .map(rowToMessage);
    },
    recent(limit: number): AgentMessage[] {
      return db
        .prepare('SELECT * FROM agent_messages ORDER BY created_at DESC, rowid DESC LIMIT ?')
        .all(limit)
        .map(rowToMessage);
    },
  };

  const rowToReply = (r: any): BroadcastReply =>
    BroadcastReplySchema.parse({
      id: r.id,
      broadcastId: r.broadcast_id,
      agentId: r.agent_id,
      ok: Boolean(r.ok),
      reply: r.reply,
      finishedAt: r.finished_at,
    });

  const broadcasts = {
    insert(b: { id: string; message: string; createdAt: string }): void {
      db.prepare('INSERT OR REPLACE INTO broadcasts (id, message, created_at) VALUES (?, ?, ?)').run(
        b.id, b.message, b.createdAt,
      );
    },
    insertReply(r: BroadcastReply): void {
      db.prepare(
        'INSERT OR REPLACE INTO broadcast_replies (id, broadcast_id, agent_id, ok, reply, finished_at) VALUES (?, ?, ?, ?, ?, ?)',
      ).run(r.id, r.broadcastId, r.agentId, r.ok ? 1 : 0, r.reply, r.finishedAt);
    },
    recent(limit: number): Broadcast[] {
      const rows = db
        .prepare('SELECT * FROM broadcasts ORDER BY created_at DESC, rowid DESC LIMIT ?')
        .all(limit) as { id: string; message: string; created_at: string }[];
      const replyStmt = db.prepare('SELECT * FROM broadcast_replies WHERE broadcast_id = ? ORDER BY agent_id');
      return rows.map((b) =>
        BroadcastSchema.parse({
          id: b.id,
          message: b.message,
          createdAt: b.created_at,
          replies: replyStmt.all(b.id).map(rowToReply),
        }),
      );
    },
  };

  const rowToTask = (r: any): AgentTask =>
    AgentTaskSchema.parse({
      id: r.id,
      // '' is the old NOT NULL schema's stand-in for "no agent" (see the migration)
      agentId: r.agent_id ? r.agent_id : null,
      title: r.title, status: r.status,
      createdAt: r.created_at, updatedAt: r.updated_at,
      detail: r.detail ?? '',
      dueAt: r.due_at ?? null,
      priority: r.priority ?? 'normal',
      ownerKind: r.owner_kind ?? 'agent',
      horizon: r.horizon ?? null,
      runId: r.run_id ?? null,
    });

  const agentTasks = {
    insert(input: AgentTaskInput): void {
      const t = AgentTaskSchema.parse(input);
      db.prepare(
        `INSERT OR REPLACE INTO agent_tasks
           (id, agent_id, title, status, created_at, updated_at, detail, due_at, priority, owner_kind, horizon, run_id)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
      ).run(
        t.id, t.agentId ?? '', t.title, t.status, t.createdAt, t.updatedAt,
        t.detail, t.dueAt, t.priority, t.ownerKind, t.horizon, t.runId,
      );
    },
    get(id: string): AgentTask | null {
      const r = db.prepare('SELECT * FROM agent_tasks WHERE id = ?').get(id);
      return r ? rowToTask(r) : null;
    },
    byAgent(agentId: string): AgentTask[] {
      return db
        .prepare('SELECT * FROM agent_tasks WHERE agent_id = ? ORDER BY created_at DESC, rowid DESC')
        .all(agentId)
        .map(rowToTask);
    },
    all(): AgentTask[] {
      return db.prepare('SELECT * FROM agent_tasks ORDER BY created_at DESC, rowid DESC').all().map(rowToTask);
    },
    setStatus(id: string, status: AgentTask['status'], updatedAt: string): void {
      AgentTaskStatusSchema.parse(status);
      db.prepare('UPDATE agent_tasks SET status = ?, updated_at = ? WHERE id = ?').run(status, updatedAt, id);
    },
    /** Partial edit of the fields the task drawer owns. Validates by round-
     *  tripping the whole task, so a bad patch can never reach the table. */
    update(
      id: string,
      patch: Partial<Pick<AgentTask, 'title' | 'detail' | 'dueAt' | 'priority' | 'horizon' | 'agentId' | 'ownerKind' | 'runId'>>,
      updatedAt: string,
    ): void {
      const current = this.get(id);
      if (!current) return;
      const next = AgentTaskSchema.parse({ ...current, ...patch, updatedAt });
      db.prepare(
        `UPDATE agent_tasks
            SET title = ?, detail = ?, due_at = ?, priority = ?, horizon = ?,
                agent_id = ?, owner_kind = ?, run_id = ?, updated_at = ?
          WHERE id = ?`,
      ).run(
        next.title, next.detail, next.dueAt, next.priority, next.horizon,
        next.agentId ?? '', next.ownerKind, next.runId, next.updatedAt, id,
      );
    },
    remove(id: string): void {
      db.prepare('DELETE FROM agent_tasks WHERE id = ?').run(id);
    },
  };

  const setupChecks = {
    set(id: string, doneAt: string): void {
      SetupCheckSchema.parse({ id, doneAt });
      db.prepare('INSERT OR REPLACE INTO setup_checks (id, done_at) VALUES (?, ?)').run(id, doneAt);
    },
    unset(id: string): void {
      db.prepare('DELETE FROM setup_checks WHERE id = ?').run(id);
    },
    all(): SetupCheck[] {
      return db
        .prepare('SELECT * FROM setup_checks')
        .all()
        .map((r: any) => SetupCheckSchema.parse({ id: r.id, doneAt: r.done_at }));
    },
  };

  const rowToBrandDealOverride = (r: any): BrandDealOverride =>
    BrandDealOverrideSchema.parse({
      dealId: r.deal_id, outcome: r.outcome, amountUsd: r.amount_usd ?? null, updatedAt: r.updated_at,
    });

  const brandDeals = {
    upsert(o: BrandDealOverride): void {
      BrandDealOverrideSchema.parse(o);
      db.prepare(
        'INSERT OR REPLACE INTO brand_deal_overrides (deal_id, outcome, amount_usd, updated_at) VALUES (?, ?, ?, ?)',
      ).run(o.dealId, o.outcome, o.amountUsd, o.updatedAt);
    },
    get(dealId: string): BrandDealOverride | null {
      const r = db.prepare('SELECT * FROM brand_deal_overrides WHERE deal_id = ?').get(dealId);
      return r ? rowToBrandDealOverride(r) : null;
    },
    all(): BrandDealOverride[] {
      return db.prepare('SELECT * FROM brand_deal_overrides').all().map(rowToBrandDealOverride);
    },
  };

  const rowToCron = (r: any): AgentCron =>
    AgentCronSchema.parse({
      id: r.id, agentId: r.agent_id, schedule: r.schedule, description: r.description,
      enabled: Boolean(r.enabled), createdAt: r.created_at,
    });

  const agentCrons = {
    insert(c: AgentCron): void {
      AgentCronSchema.parse(c);
      if (!isValidCron(c.schedule)) throw new Error(`invalid cron schedule: ${c.schedule}`);
      db.prepare(
        'INSERT OR REPLACE INTO agent_crons (id, agent_id, schedule, description, enabled, created_at) VALUES (?, ?, ?, ?, ?, ?)',
      ).run(c.id, c.agentId, c.schedule, c.description, c.enabled ? 1 : 0, c.createdAt);
    },
    byAgent(agentId: string): AgentCron[] {
      return db
        .prepare('SELECT * FROM agent_crons WHERE agent_id = ? ORDER BY created_at DESC, rowid DESC')
        .all(agentId)
        .map(rowToCron);
    },
    all(): AgentCron[] {
      return db.prepare('SELECT * FROM agent_crons ORDER BY created_at DESC, rowid DESC').all().map(rowToCron);
    },
    setEnabled(id: string, enabled: boolean): void {
      db.prepare('UPDATE agent_crons SET enabled = ? WHERE id = ?').run(enabled ? 1 : 0, id);
    },
    remove(id: string): void {
      db.prepare('DELETE FROM agent_crons WHERE id = ?').run(id);
    },
  };

  const contactTags = {
    upsert(t: ContactTag): void {
      ContactTagSchema.parse(t);
      db.prepare(
        'INSERT INTO contact_tags (person, channel, tag, tier) VALUES (?, ?, ?, ?) ON CONFLICT(person, channel) DO UPDATE SET tag = excluded.tag, tier = excluded.tier',
      ).run(t.person, t.channel, t.tag, t.tier);
    },
    all(): ContactTag[] {
      return (db.prepare('SELECT * FROM contact_tags ORDER BY tier, person').all() as ContactTag[]).map(
        (r) => ContactTagSchema.parse(r),
      );
    },
    byTier(tier: number): ContactTag[] {
      return (
        db.prepare('SELECT * FROM contact_tags WHERE tier = ? ORDER BY person').all(tier) as ContactTag[]
      ).map((r) => ContactTagSchema.parse(r));
    },
    remove(person: string, channel: string): void {
      db.prepare('DELETE FROM contact_tags WHERE person = ? AND channel = ?').run(person, channel);
    },
  };

  const rowToSnapshot = (r: any): SocialSnapshot =>
    SocialSnapshotSchema.parse({
      platform: r.platform,
      capturedAt: r.captured_at,
      followers: r.followers,
      source: r.source,
    });

  const social = {
    upsertAccount(a: SocialAccount): void {
      SocialAccountSchema.parse(a);
      db.prepare(
        'INSERT OR REPLACE INTO social_accounts (platform, handle, url, "order") VALUES (?, ?, ?, ?)',
      ).run(a.platform, a.handle, a.url, a.order);
    },
    accounts(): SocialAccount[] {
      return db
        .prepare('SELECT * FROM social_accounts ORDER BY "order"')
        .all()
        .map((r) => SocialAccountSchema.parse(r));
    },
    insertSnapshot(s: SocialSnapshot): void {
      SocialSnapshotSchema.parse(s);
      db.prepare(
        'INSERT OR REPLACE INTO social_snapshots (platform, captured_at, followers, source) VALUES (?, ?, ?, ?)',
      ).run(s.platform, s.capturedAt, s.followers, s.source);
    },
    snapshots(platform: SocialPlatform): SocialSnapshot[] {
      return db
        .prepare('SELECT * FROM social_snapshots WHERE platform = ? ORDER BY captured_at')
        .all(platform)
        .map(rowToSnapshot);
    },
    latest(): SocialSnapshot[] {
      return db
        .prepare(
          `SELECT * FROM social_snapshots s
           WHERE captured_at = (SELECT MAX(captured_at) FROM social_snapshots WHERE platform = s.platform)
           ORDER BY platform`,
        )
        .all()
        .map(rowToSnapshot);
    },
    upsertDm(d: SocialDm): void {
      SocialDmSchema.parse(d);
      db.prepare(
        'INSERT OR REPLACE INTO social_dms (platform, count, updated_at) VALUES (?, ?, ?)',
      ).run(d.platform, d.count, d.updatedAt);
    },
    dms(): SocialDm[] {
      return db
        .prepare(
          `SELECT d.platform, d.count, d.updated_at AS updatedAt FROM social_dms d
           LEFT JOIN social_accounts a ON a.platform = d.platform
           ORDER BY a."order"`,
        )
        .all()
        .map((r) => SocialDmSchema.parse(r));
    },
    insertDmSnapshot(s: SocialDmSnapshot): void {
      SocialDmSnapshotSchema.parse(s);
      db.prepare(
        'INSERT OR REPLACE INTO social_dm_snapshots (platform, captured_at, count, source) VALUES (?, ?, ?, ?)',
      ).run(s.platform, s.capturedAt, s.count, s.source);
    },
    dmSnapshots(platform?: SocialPlatform): SocialDmSnapshot[] {
      const rows = platform
        ? db
            .prepare('SELECT platform, captured_at AS capturedAt, count, source FROM social_dm_snapshots WHERE platform = ? ORDER BY captured_at')
            .all(platform)
        : db
            .prepare('SELECT platform, captured_at AS capturedAt, count, source FROM social_dm_snapshots ORDER BY platform, captured_at')
            .all();
      return rows.map((r) => SocialDmSnapshotSchema.parse(r));
    },
    // Individual DM messages (the inbox). Fed live by POST /api/webhooks/manychat;
    // seeded until then. Upsert by id so replayed webhooks don't duplicate.
    upsertDmMessage(m: SocialDmMessage): void {
      SocialDmMessageSchema.parse(m);
      db.prepare(
        `INSERT OR REPLACE INTO social_dm_messages
           (id, platform, subscriber_id, name, handle, text, direction, tag, ts, source)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
      ).run(m.id, m.platform, m.subscriberId, m.name, m.handle, m.text, m.direction, m.tag, m.ts, m.source);
    },
    dmMessages(platform?: SocialPlatform): SocialDmMessage[] {
      const cols =
        'id, platform, subscriber_id AS subscriberId, name, handle, text, direction, tag, ts, source';
      const rows = platform
        ? db.prepare(`SELECT ${cols} FROM social_dm_messages WHERE platform = ? ORDER BY ts DESC`).all(platform)
        : db.prepare(`SELECT ${cols} FROM social_dm_messages ORDER BY ts DESC`).all();
      return rows.map((r) => SocialDmMessageSchema.parse(r));
    },
  };

  const emailList = {
    insertSnapshot(s: EmailListSnapshot): void {
      EmailListSnapshotSchema.parse(s);
      db.prepare(
        'INSERT OR REPLACE INTO email_list_snapshots (captured_at, subscribers, source) VALUES (?, ?, ?)',
      ).run(s.capturedAt, s.subscribers, s.source);
    },
    // Drop seed-sourced rows so a re-seed is authoritative — the real Beehiiv
    // baseline replaces any retired dummy history. Live-synced snapshots
    // (source 'beehiiv') are preserved.
    deleteSeeded(): void {
      db.prepare("DELETE FROM email_list_snapshots WHERE source LIKE 'seed%'").run();
    },
    snapshots(): EmailListSnapshot[] {
      return db
        .prepare('SELECT captured_at AS capturedAt, subscribers, source FROM email_list_snapshots ORDER BY captured_at')
        .all()
        .map((r) => EmailListSnapshotSchema.parse(r));
    },
    latest(): EmailListSnapshot | null {
      const row = db
        .prepare('SELECT captured_at AS capturedAt, subscribers, source FROM email_list_snapshots ORDER BY captured_at DESC LIMIT 1')
        .get();
      return row ? EmailListSnapshotSchema.parse(row) : null;
    },
  };

  const rowToPost = (r: {
    id: string;
    caption: string;
    media_url: string | null;
    platforms: string;
    status: string;
    scheduled_for: string | null;
    created_at: string;
  }): SocialPost =>
    SocialPostSchema.parse({
      id: r.id,
      caption: r.caption,
      mediaUrl: r.media_url,
      platforms: JSON.parse(r.platforms),
      status: r.status,
      scheduledFor: r.scheduled_for,
      createdAt: r.created_at,
    });

  const socialPosts = {
    enqueue(p: SocialPost): void {
      SocialPostSchema.parse(p);
      db.prepare(
        `INSERT OR REPLACE INTO social_posts (id, caption, media_url, platforms, status, scheduled_for, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
      ).run(p.id, p.caption, p.mediaUrl, JSON.stringify(p.platforms), p.status, p.scheduledFor, p.createdAt);
    },
    all(): SocialPost[] {
      return db
        .prepare('SELECT * FROM social_posts ORDER BY created_at DESC')
        .all()
        .map((r) => rowToPost(r as Parameters<typeof rowToPost>[0]));
    },
    queued(): SocialPost[] {
      return db
        .prepare("SELECT * FROM social_posts WHERE status = 'queued' ORDER BY created_at DESC")
        .all()
        .map((r) => rowToPost(r as Parameters<typeof rowToPost>[0]));
    },
    /** Withdraw a post that is still queued (SPEC H40). One statement, so a
        post the agent picks up concurrently is never deleted mid-publish. */
    removeQueued(id: string): 'removed' | 'missing' | 'not_queued' {
      const res = db.prepare("DELETE FROM social_posts WHERE id = ? AND status = 'queued'").run(id);
      if (res.changes > 0) return 'removed';
      const exists = db.prepare('SELECT 1 FROM social_posts WHERE id = ?').get(id);
      return exists ? 'not_queued' : 'missing';
    },
  };

  const people = {
    all(): Person[] {
      return db
        .prepare('SELECT * FROM people ORDER BY department_id, name')
        .all()
        .map((r: any) =>
          PersonSchema.parse({
            id: r.id,
            departmentId: r.department_id,
            name: r.name,
            role: r.role,
            tools: JSON.parse(r.tools),
          }),
        );
    },
    insert(p: Person): void {
      PersonSchema.parse(p);
      db.prepare(
        'INSERT OR REPLACE INTO people (id, department_id, name, role, tools) VALUES (?, ?, ?, ?, ?)',
      ).run(p.id, p.departmentId, p.name, p.role, JSON.stringify(p.tools));
    },
    deleteWhereIdNotIn(ids: string[]): void {
      // An empty keep-list means "keep nothing": SQL rejects `NOT IN ()`.
      if (ids.length === 0) { db.prepare('DELETE FROM people').run(); return; }
      const placeholders = ids.map(() => '?').join(', ');
      db.prepare(`DELETE FROM people WHERE id NOT IN (${placeholders})`).run(...ids);
    },
  };

  const sopTasks = {
    all(): SopTask[] {
      return db
        .prepare('SELECT * FROM sop_tasks ORDER BY department_id, title')
        .all()
        .map((r: any) =>
          SopTaskSchema.parse({
            id: r.id,
            departmentId: r.department_id,
            title: r.title,
            summary: r.summary,
            steps: JSON.parse(r.steps),
            assigneeKind: r.assignee_kind,
            assigneeId: r.assignee_id,
          }),
        );
    },
    insert(t: SopTask): void {
      SopTaskSchema.parse(t);
      db.prepare(
        'INSERT OR REPLACE INTO sop_tasks (id, department_id, title, summary, steps, assignee_kind, assignee_id) VALUES (?, ?, ?, ?, ?, ?, ?)',
      ).run(t.id, t.departmentId, t.title, t.summary, JSON.stringify(t.steps), t.assigneeKind, t.assigneeId);
    },
    deleteWhereIdNotIn(ids: string[]): void {
      // An empty keep-list means "keep nothing": SQL rejects `NOT IN ()`.
      if (ids.length === 0) { db.prepare('DELETE FROM sop_tasks').run(); return; }
      const placeholders = ids.map(() => '?').join(', ');
      db.prepare(`DELETE FROM sop_tasks WHERE id NOT IN (${placeholders})`).run(...ids);
    },
  };

  const rowToWorkflow = (r: any): Workflow =>
    WorkflowSchema.parse({
      id: r.id,
      name: r.name,
      subtitle: r.subtitle,
      revenueUsd: r.revenue_usd,
      order: r.ord,
      steps: JSON.parse(r.steps),
      // only present when set, so rows written before the canvas read back
      // exactly as they were written
      ...(r.ratio_target != null ? { ratioTarget: r.ratio_target } : {}),
    });

  const rowToWorkflowVersion = (r: any): WorkflowVersion =>
    WorkflowVersionSchema.parse({ id: r.id, workflowId: r.workflow_id, version: r.version, createdAt: r.created_at, note: r.note, snapshot: JSON.parse(r.snapshot) });
  const workflowVersions = {
    list(workflowId: string): WorkflowVersion[] {
      return db.prepare('SELECT * FROM workflow_versions WHERE workflow_id = ? ORDER BY version DESC').all(workflowId).map(rowToWorkflowVersion);
    },
    get(id: string): WorkflowVersion | null {
      const r = db.prepare('SELECT * FROM workflow_versions WHERE id = ?').get(id) as any;
      return r ? rowToWorkflowVersion(r) : null;
    },
    latest(workflowId: string): WorkflowVersion | null {
      const r = db.prepare('SELECT * FROM workflow_versions WHERE workflow_id = ? ORDER BY version DESC LIMIT 1').get(workflowId) as any;
      return r ? rowToWorkflowVersion(r) : null;
    },
    /** Appends a version unless the snapshot equals the latest one. */
    record(w: Workflow, note = ''): WorkflowVersion | null {
      const snapshot = JSON.stringify(w);
      const last = workflowVersions.latest(w.id);
      if (last && JSON.stringify(last.snapshot) === snapshot) return null;
      const version = (last?.version ?? 0) + 1;
      const id = `${w.id}-v${version}`;
      const createdAt = new Date().toISOString();
      db.prepare('INSERT INTO workflow_versions (id, workflow_id, version, created_at, note, snapshot) VALUES (?, ?, ?, ?, ?, ?)').run(id, w.id, version, createdAt, note, snapshot);
      return { id, workflowId: w.id, version, createdAt, note, snapshot: JSON.parse(snapshot) };
    },
  };

  const rowToConfigVersion = (r: any): ConfigVersion =>
    ConfigVersionSchema.parse({ id: r.id, target: r.target, version: r.version, createdAt: r.created_at, note: r.note, snapshot: JSON.parse(r.snapshot) });
  const configVersions = {
    list(target: string): ConfigVersion[] {
      return db.prepare('SELECT * FROM config_versions WHERE target = ? ORDER BY version DESC').all(target).map(rowToConfigVersion);
    },
    get(id: string): ConfigVersion | null {
      const r = db.prepare('SELECT * FROM config_versions WHERE id = ?').get(id) as any;
      return r ? rowToConfigVersion(r) : null;
    },
    latest(target: string): ConfigVersion | null {
      const r = db.prepare('SELECT * FROM config_versions WHERE target = ? ORDER BY version DESC LIMIT 1').get(target) as any;
      return r ? rowToConfigVersion(r) : null;
    },
    /** Appends a version unless identical to the latest. */
    record(target: string, snapshotValue: unknown, note = ''): ConfigVersion {
      const snapshot = JSON.stringify(snapshotValue);
      const last = configVersions.latest(target);
      if (last && JSON.stringify(last.snapshot) === snapshot) return last;
      const version = (last?.version ?? 0) + 1;
      const id = `${target}-v${version}`;
      const createdAt = new Date().toISOString();
      db.prepare('INSERT INTO config_versions (id, target, version, created_at, note, snapshot) VALUES (?, ?, ?, ?, ?, ?)').run(id, target, version, createdAt, note, snapshot);
      return { id, target, version, createdAt, note, snapshot: JSON.parse(snapshot) };
    },
  };

  const workflows = {
    all(): Workflow[] {
      return db.prepare('SELECT * FROM workflows ORDER BY ord, name').all().map(rowToWorkflow);
    },
    /** Single workflow by id, or null when it doesn't exist — the builder's
     *  update/delete routes use this to tell "not found" from a real 500. */
    get(id: string): Workflow | null {
      const r = db.prepare('SELECT * FROM workflows WHERE id = ?').get(id) as any;
      return r ? rowToWorkflow(r) : null;
    },
    /** Create-or-replace, and the ONLY write path — so every save records a
     *  version (skipped when nothing changed, so re-running the seed does not
     *  pile up identical rows). `note` labels the version. */
    insert(w: Workflow, note = ''): void {
      WorkflowSchema.parse(w);
      db.prepare(
        'INSERT OR REPLACE INTO workflows (id, name, subtitle, revenue_usd, ord, steps, ratio_target) VALUES (?, ?, ?, ?, ?, ?, ?)',
      ).run(w.id, w.name, w.subtitle, w.revenueUsd, w.order, JSON.stringify(w.steps), w.ratioTarget ?? null);
      workflowVersions.record(w, note);
    },
    /** Deletes one workflow by id — the builder's delete affordance. Distinct
     *  from `deleteWhereIdNotIn`, which is the seed's bulk reconciliation. */
    remove(id: string): void {
      db.prepare('DELETE FROM workflows WHERE id = ?').run(id);
    },
    deleteWhereIdNotIn(ids: string[]): void {
      // An empty keep-list means "keep nothing": SQL rejects `NOT IN ()`.
      if (ids.length === 0) { db.prepare('DELETE FROM workflows').run(); return; }
      const placeholders = ids.map(() => '?').join(', ');
      db.prepare(`DELETE FROM workflows WHERE id NOT IN (${placeholders})`).run(...ids);
    },
  };

  const skills = {
    all(): Skill[] {
      return db
        .prepare('SELECT * FROM skills ORDER BY ord, name')
        .all()
        .map((r: any) =>
          SkillSchema.parse({
            id: r.id,
            name: r.name,
            category: r.category,
            description: r.description,
            ownerAgentId: r.owner_agent_id,
            status: r.status,
            tools: JSON.parse(r.tools),
            markdown: r.markdown,
            order: r.ord,
          }),
        );
    },
    insert(s: Skill): void {
      SkillSchema.parse(s);
      db.prepare(
        'INSERT OR REPLACE INTO skills (id, name, category, description, owner_agent_id, status, tools, markdown, ord) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)',
      ).run(s.id, s.name, s.category, s.description, s.ownerAgentId, s.status, JSON.stringify(s.tools), s.markdown, s.order);
    },
    deleteWhereIdNotIn(ids: string[]): void {
      // An empty keep-list means "keep nothing": SQL rejects `NOT IN ()`.
      if (ids.length === 0) { db.prepare('DELETE FROM skills').run(); return; }
      const placeholders = ids.map(() => '?').join(', ');
      db.prepare(`DELETE FROM skills WHERE id NOT IN (${placeholders})`).run(...ids);
    },
  };

  const rowToFunnelTouch = (r: any): FunnelTouch =>
    FunnelTouchSchema.parse({
      id: r.id,
      contactId: r.contact_id,
      seq: r.seq,
      stage: r.stage,
      channel: r.channel,
      label: r.label,
      source: r.source,
      at: r.at,
    });

  const funnel = {
    insertContact(c: FunnelContact): void {
      FunnelContactSchema.parse(c);
      db.prepare(
        'INSERT OR REPLACE INTO funnel_contacts (id, name, venture, status, product, amount_usd, relationship, likelihood, email, phone, person, company, role, linkedin, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)',
      ).run(c.id, c.name, c.venture, c.status, c.product, c.amountUsd, c.relationship, c.likelihood, c.email, c.phone, c.person, c.company, c.role, c.linkedin, c.createdAt);
    },
    insertTouch(t: FunnelTouch): void {
      FunnelTouchSchema.parse(t);
      db.prepare(
        'INSERT OR REPLACE INTO funnel_touches (id, contact_id, seq, stage, channel, label, source, at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)',
      ).run(t.id, t.contactId, t.seq, t.stage, t.channel, t.label, t.source, t.at);
    },
    /** Contacts with their touches in journey order, newest contact first. */
    journeys(venture?: FunnelVenture): FunnelJourney[] {
      const rows = (
        venture
          ? db.prepare('SELECT * FROM funnel_contacts WHERE venture = ? ORDER BY created_at DESC, id').all(venture)
          : db.prepare('SELECT * FROM funnel_contacts ORDER BY created_at DESC, id').all()
      ) as any[];
      const touchStmt = db.prepare('SELECT * FROM funnel_touches WHERE contact_id = ? ORDER BY seq');
      return rows.map((r) =>
        FunnelJourneySchema.parse({
          id: r.id,
          name: r.name,
          venture: r.venture,
          status: r.status,
          product: r.product,
          amountUsd: r.amount_usd,
          relationship: r.relationship,
          likelihood: r.likelihood,
          email: r.email,
          phone: r.phone,
          person: r.person,
          company: r.company,
          role: r.role,
          linkedin: r.linkedin,
          createdAt: r.created_at,
          touches: touchStmt.all(r.id).map(rowToFunnelTouch),
        }),
      );
    },
  };

  return {
    departments,
    agents,
    tools,
    roadmap,
    metrics,
    phases,
    agentRuns,
    agentMessages,
    agentTasks,
    agentCrons,
    brandDeals,
    setupChecks,
    broadcasts,
    contactTags,
    social,
    emailList,
    socialPosts,
    funnel,
    people,
    sopTasks,
    workflows,
    workflowVersions,
    configVersions,
    skills,
    /** Content Lab repository (lib/lab/store.ts). */
    lab: labStore(db),
    /* FounderOS UI merge repositories (lib/db-ext/*; tables in FO_DDL). */
    trading: tradingRepo(db),
    clientConfirmations: clientConfirmationsRepo(db),
    clientWork: clientWorkRepo(db),
    leadMagnets: leadMagnetsRepo(db),
    usageReadings: usageReadingsRepo(db),
    commsDigests: commsDigestsRepo(db),
    callRecordings: callRecordingsRepo(db),
    deliverableDecisions: deliverableDecisionsRepo(db),
    cronRuns: cronRunsRepo(db),
    /** The live journal mode, read from SQLite (S10 FR-17), e.g. "wal". */
    journalMode: (): string => String(db.pragma('journal_mode', { simple: true }) ?? ''),
    close: () => db.close(),
  };
}

export type GladosDb = ReturnType<typeof openDb>;
