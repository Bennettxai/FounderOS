PRAGMA foreign_keys=OFF;
BEGIN TRANSACTION;
CREATE TABLE seed_meta (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
INSERT INTO seed_meta VALUES('seed_version','2026-09-30-alex-first-name');
CREATE TABLE departments (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  tagline TEXT NOT NULL DEFAULT '',
  color TEXT NOT NULL,
  "order" INTEGER NOT NULL
);
INSERT INTO departments VALUES('dept-sales','Sales','sales','Pipeline and deals.','#fafafa',1);
INSERT INTO departments VALUES('dept-marketing-growth','Marketing/Growth','marketing-growth','Publishing, content, attention.','#d4d4d4',2);
INSERT INTO departments VALUES('dept-tech','TECH','tech','AI & automations · Brain.','#a3a3a3',3);
INSERT INTO departments VALUES('dept-finance','Finances','finances','Every processor, one view.','#737373',4);
INSERT INTO departments VALUES('dept-comms','Communications','communications','Gmail, WhatsApp, Slack → one feed.','#525252',5);
INSERT INTO departments VALUES('dept-clients','Clients','clients','Every client, onboarded and served.','#d4d4d4',6);
CREATE TABLE agents (
  id TEXT PRIMARY KEY,
  department_id TEXT NOT NULL REFERENCES departments(id),
  name TEXT NOT NULL,
  role TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  tier TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  tools TEXT NOT NULL DEFAULT '[]'
, parent_id TEXT, instance TEXT NOT NULL DEFAULT 'builtin');
INSERT INTO agents VALUES('conductor','dept-tech','Conductor','Broadcast & Orchestration','active','lead','Fans your message out to every agent at once and checks which instance hosts (Clawline, Ollama, tmux) are available for future bindings.','fan-out runtime','["broadcast","clawline","tmux"]',NULL,'builtin');
INSERT INTO agents VALUES('comms-digest','dept-comms','Comms Digest','Morning Report · 09:00 daily','active','lead','Scrapes the last 24h across all four inboxes, WhatsApp and Slack and ranks who needs a reply: calls first, then clients, community members, brand deals, group chats, companies last. Also lists what to unsubscribe from.','rules + connectors','["comms-feed","calendar","ledger"]',NULL,'builtin');
INSERT INTO agents VALUES('comms-agent','dept-comms','Comms Agent','Unified Communications Instance','active','lead','Owns the unified /comms feed. Aggregates its three channel workers and reports which are live.','aggregate of workers','["comms-feed"]',NULL,'builtin');
INSERT INTO agents VALUES('gmail-worker','dept-comms','Gmail Worker','IMAP Inboxes ×4','planned','worker','Pulls unread counts and recent mail from up to four IMAP inboxes into /comms. Activates when INBOX_* creds land.','imapflow','["imap"]','comms-agent','builtin');
INSERT INTO agents VALUES('whatsapp-worker','dept-comms','WhatsApp Worker','Chat Monitor','active','worker','Reads the local WhatsApp ChatStorage (local team chats) into /comms. Works today.','local sqlite (read-only)','["whatsapp"]','comms-agent','builtin');
INSERT INTO agents VALUES('slack-worker','dept-comms','Slack Worker','Channel Digest','planned','worker','Latest messages across joined channels into /comms. Needs SLACK_BOT_TOKEN.','@slack/web-api','["slack"]','comms-agent','builtin');
INSERT INTO agents VALUES('social-agent','dept-marketing-growth','Social Agent','Social Media & Content Creation Instance','active','lead','Owns publishing and content production. Aggregates the Postly and Adsmith workers.','aggregate of workers','["postly","adsmith","reelkit","renderly","dmflow"]',NULL,'builtin');
INSERT INTO agents VALUES('brand-deal-agent','dept-sales','Brand Deal Agent','Vera · brand deal manager','active','worker','Negotiates as Vera, Alex’s brand deal manager: qualifies inbound, anchors and counters, chases unpaid invoices, and bumps stalled threads. A tested contact governor decides whether a thread may be touched at all (five bumps maximum, one revival per brand per six months). Drafts only, never sends.','rules + gateway','["ledger","imap"]','sales-agent','builtin');
INSERT INTO agents VALUES('newsletter-agent','dept-marketing-growth','Newsletter Agent','Issue drafting','active','worker','Reads newsletter send performance, builds a brief that is honest about how thin the history is, and drafts the next issue against the skill file. Drafts only, never schedules or sends.','rules + gateway','["newsletter"]','social-agent','builtin');
INSERT INTO agents VALUES('postly-publisher','dept-marketing-growth','Postly Publisher','Six-Platform Publishing','active','worker','Publishes and monitors six platforms under @founderos.ai via Postly. Live once the Postly key is set.','postly api','["postly"]','social-agent','builtin');
INSERT INTO agents VALUES('adsmith-creative','dept-marketing-growth','Adsmith Creative','UGC Ad Generation','active','worker','Generates UGC ads for Vantage (Veo/Sora/Kling) via the Adsmith API. Live once Adsmith auth is set.','adsmith api','["adsmith"]','social-agent','builtin');
INSERT INTO agents VALUES('reelkit-editor','dept-marketing-growth','Reelkit Editor','Social Editing Pipeline','active','worker','Editing and rendering pipeline for social media clips, captions, and promotional cuts.','reelkit pipeline','["reelkit","whisper"]','social-agent','builtin');
INSERT INTO agents VALUES('renderly-creative','dept-marketing-growth','Renderly Creative','AI Creative Studio','active','worker','Renderly creative generation for social assets, product shots, and campaign visuals.','renderly cli','["renderly"]','social-agent','builtin');
INSERT INTO agents VALUES('dmflow-mcp','dept-marketing-growth','DMFlow MCP','DM Automation','active','worker','DMFlow MCP/API lane for social DM automations, keyword flows, and lead capture.','dmflow api','["dmflow"]','social-agent','builtin');
INSERT INTO agents VALUES('sales-agent','dept-sales','Sales Agent','Deals & Pipeline Instance','active','lead','Owns the sales pillar. Aggregates CRM Pulse and reports the live Ledger deals pipeline.','aggregate of workers','["ledger","paykit","stripe","flexpay","recall","plaud"]',NULL,'builtin');
INSERT INTO agents VALUES('launchpad-cohort-sales','dept-sales','Launchpad Cohort','Sales Account Lane','planned','worker','Launchpad Cohort sales lane: offers, calls, payment confirmation, and CRM context.','account lane','["ledger","stripe","paykit"]','sales-agent','builtin');
INSERT INTO agents VALUES('vantage-sales','dept-sales','Vantage','Sales Account Lane','planned','worker','Vantage sales lane: account pipeline, PayKit context, payment confirmation, and call data.','account lane','["ledger","stripe","paykit"]','sales-agent','builtin');
INSERT INTO agents VALUES('paykit-sales','dept-finance','PayKit','Offer & Payment Platform','planned','worker','PayKit sales platform connection for offers and customer/payment context.','paykit api','["paykit"]','payments-pulse','builtin');
INSERT INTO agents VALUES('vantage-paykit','dept-sales','Vantage PayKit','Vantage PayKit Lane','planned','worker','PayKit lane specifically under Vantage for offer, payment, and customer context.','paykit api','["paykit"]','vantage-sales','builtin');
INSERT INTO agents VALUES('stripe-sales','dept-finance','Stripe','Sales Payment Processor','planned','worker','Stripe payment confirmation lane for sales workflows and account-level revenue checks.','stripe sdk','["stripe"]','payments-pulse','builtin');
INSERT INTO agents VALUES('processor-confirmation','dept-finance','Processor Confirm','Payment API Confirmation','planned','worker','APIs to payment processors for confirming paid, failed, disputed, and pending states.','processor registry','["stripe","paypal","square","whop","paykit"]','payments-pulse','builtin');
INSERT INTO agents VALUES('flexpay-financing','dept-finance','FlexPay Financing','Financing Options','planned','worker','FlexPay financing options lane for sales offers and payment-plan context.','flexpay api','["flexpay"]','payments-pulse','builtin');
INSERT INTO agents VALUES('sales-calls-data','dept-sales','Sales Calls Data','Call Intelligence','planned','worker','Sales calls data lane for recordings, notes, outcomes, and follow-up context: Recall on the calls, Plaud in the room.','recall + plaud + crm','["recall","plaud","ledger"]','sales-agent','builtin');
INSERT INTO agents VALUES('data-agent','dept-tech','Data Agent','Brain Analyst','active','lead','Bound to the Brain instance: analyzes markdown + vector storage health and surfaces ideas. Answers broadcasts by querying the brain.','optimal-engine CLI','["optimal-engine","brain-store","ollama","supabase"]',NULL,'builtin');
INSERT INTO agents VALUES('markdown-auditor','dept-tech','Markdown Auditor','brain-store Health','active','worker','Audits the knowledge base: broken wikilinks, orphan pages, duplicate titles, and whether the index search reads still matches the store on disk.','link audit','["brain-store"]','data-agent','builtin');
INSERT INTO agents VALUES('vector-auditor','dept-tech','Vector Auditor','pgvector / Supabase Health','active','worker','Runs optimal-engine doctor: connection to Supabase pgvector, embedding checks, health score. Works today.','optimal-engine doctor','["supabase","ollama"]','data-agent','builtin');
INSERT INTO agents VALUES('payments-pulse','dept-finance','Payments Pulse','Processor Monitor','planned','lead','Stripe balance + recent charges; PayPal/Square/Whop registered and awaiting keys.','stripe sdk','["stripe","paypal","square","whop"]',NULL,'builtin');
INSERT INTO agents VALUES('crm-pulse','dept-sales','Ledger CRM','ATTO / Ledger Deals Pipeline','active','worker','Vantage + LC deals from Ledger, key reused from the MCP config. Works today.','ledger api','["ledger"]','sales-agent','builtin');
INSERT INTO agents VALUES('stack-monitor','dept-tech','Stack Monitor','Local Stack Health','active','lead','Reelkit, Ollama, command-center, Clawline, tmux, whisper, ffmpeg, renderly, gh + Dictate Flow stats.','local checks','["reelkit","ollama","tmux","dictate"]',NULL,'builtin');
INSERT INTO agents VALUES('client-roster','dept-clients','Client Roster','Live Client List','active','lead','The single source of truth for who is a client: reconciles Ledger and PayKit against the funnel and keeps the roster current.','funnel + Ledger','["ledger","paykit"]',NULL,'builtin');
INSERT INTO agents VALUES('client-onboarding','dept-clients','Onboarding Agent','Closed-Won to Kickoff','planned','worker','Runs the onboarding SOP end to end when a deal closes: welcome pack, workspace setup, kickoff booked, handoff notes.','ledger + slack','["ledger","slack"]','client-roster','builtin');
INSERT INTO agents VALUES('client-success','dept-clients','Client Success','Service & Renewals','planned','worker','Keeps active clients served: check-in cadence, deliverable tracking from call notes (Recall) and in-person meeting recordings (Plaud), renewal and upsell flags.','recall + plaud + slack','["recall","plaud","slack"]','client-roster','builtin');
CREATE TABLE tools (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  category TEXT NOT NULL,
  status TEXT NOT NULL,
  color TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT ''
);
INSERT INTO tools VALUES('tool-optimal-engine','Optimal Engine','Knowledge','connected','#fafafa','Markdown brain-store plus a hosted vector backend and local embeddings.');
INSERT INTO tools VALUES('tool-brain-store','brain-store/','Knowledge','connected','#d4d4d4','Local markdown knowledge base on disk.');
INSERT INTO tools VALUES('tool-supabase','Supabase (Second Brain)','Knowledge','available','#a3a3a3','Roughly a thousand pages of chunked knowledge. A free tier pauses on idle: unpause from the dashboard when queries fail.');
INSERT INTO tools VALUES('tool-obsidian','Notes Vault','Knowledge','connected','#d4d4d4','Local notes vault. Direct filesystem access.');
INSERT INTO tools VALUES('tool-postly','Postly','Social','connected','#fafafa','Six platforms behind one publishing account (IG, TikTok, X…). Key comes from the environment.');
INSERT INTO tools VALUES('tool-dmflow','DMFlow','Social','connected','#fafafa','DM automation, live via the standalone DMFlow MCP. Keyword flows are still authored in the DMFlow UI: the public API has no flow authoring.');
INSERT INTO tools VALUES('tool-skool','Skool (via Playwright)','Social','connected','#a3a3a3','launchpad-cohort community, driven by the documented Playwright workflow.');
INSERT INTO tools VALUES('tool-ledger','Ledger','CRM & Revenue','connected','#fafafa','Vantage and Launchpad Cohort deals, read-scoped (query records, not lists).');
INSERT INTO tools VALUES('tool-paykit','PayKit','CRM & Revenue','planned','#d4d4d4','Offer/payment/customer context for Sales, including the Vantage PayKit lane.');
INSERT INTO tools VALUES('tool-flexpay','FlexPay','CRM & Revenue','planned','#a3a3a3','Financing options for sales offers and payment-plan context.');
INSERT INTO tools VALUES('tool-stripe','Stripe','CRM & Revenue','available','#d4d4d4','Full client implemented — balance + charges live once STRIPE_SECRET_KEY is set.');
INSERT INTO tools VALUES('tool-ghl','GoHighLevel','CRM & Revenue','planned','#525252','CLI wrapper scaffolded; no keys configured.');
INSERT INTO tools VALUES('tool-recall','Recall','CRM & Revenue','available','#a3a3a3','AI meeting notetaker. Needs RECALL_API_KEY for API access.');
INSERT INTO tools VALUES('tool-plaud','Plaud','CRM & Revenue','connected','#d4d4d4','Pocket voice recorder for the room: in-person client meetings, site walks, memos. Transcripts + AI notes over its API; pairs with Recall on the Recordings tab.');
INSERT INTO tools VALUES('tool-trakyo','Trakyo','CRM & Revenue','planned','#737373','Revenue attribution for Launchpad Cohort: content → booked calls → payments. Status-only until Trakyo ships a public API (TRAKYO_API_KEY).');
INSERT INTO tools VALUES('tool-reelkit','Reelkit Pipeline','Creative','connected','#fafafa','Local render pipeline with per-brand themes and a skill library.');
INSERT INTO tools VALUES('tool-renderly','Renderly CLI','Creative','connected','#d4d4d4','Authenticated CLI: generate / product-photoshoot / marketing-studio / soul-id.');
INSERT INTO tools VALUES('tool-adsmith','Adsmith','Creative','connected','#a3a3a3','UGC ads for Vantage (Veo/Sora/Kling). Basic auth from env.');
INSERT INTO tools VALUES('tool-whisper','Whisper (local)','Creative','connected','#737373','Local transcription CLI plus ffmpeg. Nothing leaves the host.');
INSERT INTO tools VALUES('tool-miro','Miro','Creative','connected','#a3a3a3','REST API with a token from the environment. Architecture boards live here.');
INSERT INTO tools VALUES('tool-canva-figma','Canva + Figma','Creative','available','#525252','Connected as session-scoped MCPs. A standalone API needs separate keys.');
INSERT INTO tools VALUES('tool-imap','Email (4 IMAP slots)','Comms','available','#d4d4d4','Client implemented for 4 inboxes — set INBOX_1..4_HOST/_USER/_PASS.');
INSERT INTO tools VALUES('tool-slack','Slack','Comms','available','#a3a3a3','Client implemented. Needs a bot token with channels:read/history scopes.');
INSERT INTO tools VALUES('tool-dictate','Dictate Flow','Comms','connected','#fafafa','Voice dictation. Its local SQLite history is read live.');
INSERT INTO tools VALUES('tool-whatsapp','WhatsApp','Comms','connected','#fafafa','Desktop app local ChatStorage.sqlite, read-only: local team chats.');
INSERT INTO tools VALUES('tool-command-center','Command Center (:4000)','Orchestration','available','#d4d4d4','Kanban, brand deals, sales calls, SOPs and dispatch. Start it with npm run dev.');
INSERT INTO tools VALUES('tool-clawline','Clawline Gateway','Orchestration','available','#737373','Dormant: gateway offline and token missing. Needs a reinstall.');
INSERT INTO tools VALUES('tool-tmux','tmux','Orchestration','connected','#a3a3a3','Multi-session orchestration. The dashboard reads the live session list.');
INSERT INTO tools VALUES('tool-ollama','Ollama','Orchestration','available','#a3a3a3','Local LLM server :11434, no auth. Start it to enable free local inference.');
INSERT INTO tools VALUES('tool-vercel','Vercel CLI','Orchestration','connected','#a3a3a3','Authenticated CLI. The deploy target for a public build.');
INSERT INTO tools VALUES('tool-gh','GitHub CLI','Orchestration','connected','#737373','Authenticated CLI for repos, issues and releases.');
INSERT INTO tools VALUES('tool-paypal','PayPal','Payments','planned','#a3a3a3','Registered in the processor registry; client lands when keys do.');
INSERT INTO tools VALUES('tool-square','Square','Payments','planned','#737373','Registered in the processor registry; client lands when keys do.');
INSERT INTO tools VALUES('tool-whop','Whop','Payments','planned','#525252','Registered in the processor registry; client lands when keys do.');
CREATE TABLE roadmap_items (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  quarter TEXT NOT NULL,
  status TEXT NOT NULL,
  department_id TEXT,
  description TEXT NOT NULL DEFAULT '',
  phase_id TEXT
);
INSERT INTO roadmap_items VALUES('rm-v1','FOUNDER OS v1 baseline','2026-Q2','done','dept-tech','Six views, SQLite repos, 32 tests.','phase-2');
INSERT INTO roadmap_items VALUES('rm-mono','Monochrome rebuild + real connectors','2026-Q2','done','dept-tech','Black & white theme; IMAP, Slack, Stripe, optimal-engine wired.','phase-1');
INSERT INTO roadmap_items VALUES('rm-optimal-engine','Brain provider live','2026-Q2','done','dept-tech','optimal-engine CLI doctor/query + brain-store local fallback.','phase-1');
INSERT INTO roadmap_items VALUES('rm-creds-email','Connect 4 email inboxes','2026-Q2','done','dept-comms','Four Gmail IMAP slots live on app passwords, feeding /comms.','phase-1');
INSERT INTO roadmap_items VALUES('rm-creds-slack','Connect Slack workspace','2026-Q2','done','dept-comms','Bot token reads channels + history for the per-client board.','phase-1');
INSERT INTO roadmap_items VALUES('rm-creds-payments','Connect payment processors','2026-Q2','done','dept-finance','Stripe live; PayKit, PayPal and Square in the registry.','phase-1');
INSERT INTO roadmap_items VALUES('rm-supabase','Revive Supabase Second Brain','2026-Q2','done','dept-tech','Free-tier project unpaused; optimal-engine hybrid queries resolve again.','phase-1');
INSERT INTO roadmap_items VALUES('rm-scheduler','Agent scheduler (cron runs)','2026-Q3','done','dept-tech','Seven schedules on a 60s tick with cron_runs history and catch-up.','phase-3');
INSERT INTO roadmap_items VALUES('rm-llm','LLM summarization layer','2026-Q3','done','dept-tech','Agent chat and digests through the AI Gateway, with model failover.','phase-3');
INSERT INTO roadmap_items VALUES('rm-host','Migrate to a dedicated host','2026-Q3','done','dept-tech','App, optimal-engine and agents run on the host; Supabase stays managed.','phase-4');
INSERT INTO roadmap_items VALUES('rm-embeddings','Own the embedding stack','2026-Q3','done','dept-tech','Brain moved onto local embeddings before the hosted vendor went away.','phase-1');
INSERT INTO roadmap_items VALUES('rm-call-archive','Archive every sales call','2026-Q3','done','dept-sales','CRM and notetaker transcripts exported into brain-store as one page each.','phase-2');
INSERT INTO roadmap_items VALUES('rm-recorders','Voice recorders into the brain','2026-Q3','done','dept-sales','Pocket recorder and Recall on /comms; transcripts file themselves into Brain.','phase-2');
INSERT INTO roadmap_items VALUES('rm-trading','Trading board','2026-Q3','done','dept-finance','Robinhood and Phantom sleeves, agent reasoning, orders and trade log.','phase-2');
INSERT INTO roadmap_items VALUES('rm-usage','Token burn board','2026-Q3','done','dept-tech','Live seat-by-seat spend after the August burn; other boxes push in.','phase-2');
INSERT INTO roadmap_items VALUES('rm-workers','Worker pool on the host','2026-Q3','now','dept-tech','Cheap model seats behind the Conductor. Hardening and gateway install left.','phase-3');
INSERT INTO roadmap_items VALUES('rm-statements','Statement ingestion','2026-Q3','now','dept-finance','Card and bank statements parsed into /finances instead of hand entry.','phase-1');
INSERT INTO roadmap_items VALUES('rm-railway','Move hosting to Railway','2026-Q3','now','dept-tech','Every app moving to one platform; the gated OS demo went first as the pilot.','phase-4');
INSERT INTO roadmap_items VALUES('rm-ui','Interaction rebrand','2026-Q3','now','dept-tech','Alex-led design pass over the whole OS now the integrations are live.','phase-2');
INSERT INTO roadmap_items VALUES('rm-auth','Auth + remote access','2026-Q4','next','dept-tech','Reach FOUNDER OS on the host from anywhere, safely.','phase-4');
INSERT INTO roadmap_items VALUES('rm-postiz','Replace Postly with Postiz','2026-Q4','next','dept-clients','Self-hosted scheduler with ungated post and channel analytics.','phase-1');
INSERT INTO roadmap_items VALUES('rm-board-embed','Board fully inside the OS','2026-Q4','later','dept-tech','Conductor and 40+ agents driven from the OS, SOPs running as real skills.','phase-3');
CREATE TABLE metrics (
  id TEXT PRIMARY KEY,
  key TEXT NOT NULL UNIQUE,
  label TEXT NOT NULL,
  value REAL NOT NULL,
  unit TEXT NOT NULL DEFAULT '',
  delta REAL NOT NULL DEFAULT 0,
  period TEXT NOT NULL DEFAULT ''
);
INSERT INTO metrics VALUES('metric-unread','unread_total','Unread (all inboxes)',0.0,'emails',0.0,'pending creds');
INSERT INTO metrics VALUES('metric-brain','brain_pages','Brain-store Pages',0.0,'pages',0.0,'run Data Agent');
INSERT INTO metrics VALUES('metric-balance','stripe_available','Stripe Available',0.0,'usd',0.0,'pending creds');
INSERT INTO metrics VALUES('metric-runs','agent_runs','Agent Runs Logged',0.0,'runs',0.0,'all time');
CREATE TABLE domains (
  id TEXT PRIMARY KEY,
  number INTEGER NOT NULL,
  title TEXT NOT NULL,
  color TEXT NOT NULL,
  items TEXT NOT NULL DEFAULT '[]'
);
INSERT INTO domains VALUES('brm-1',1,'Command & Memory','#fafafa','["Optimal Engine","brain-store markdown","Agent run history","Operator dashboard"]');
INSERT INTO domains VALUES('brm-2',2,'Email Operations','#d4d4d4','["Four IMAP inboxes","Unread triage","Per-inbox health","Digest (planned)"]');
INSERT INTO domains VALUES('brm-3',3,'Team Comms','#d4d4d4','["Slack channels","Message digests","Mention tracking (planned)"]');
INSERT INTO domains VALUES('brm-4',4,'Payments & Revenue','#a3a3a3','["Stripe balance + charges","PayPal / Square / Whop registry","Reconciliation (planned)"]');
INSERT INTO domains VALUES('brm-5',5,'Knowledge & Docs','#a3a3a3','["Notes vault","Local embeddings","Supabase Second Brain"]');
INSERT INTO domains VALUES('brm-6',6,'Agent Runtime','#737373','["Registry + run()","Persisted run log","Honest failure states"]');
INSERT INTO domains VALUES('brm-7',7,'Infrastructure','#737373','["Current host","dedicated host (next)","SQLite local","Supabase managed"]');
INSERT INTO domains VALUES('brm-8',8,'Security','#525252','[".env.local secrets (gitignored)","Read-only connector scopes","No keys in repo"]');
CREATE TABLE personas (
  id TEXT PRIMARY KEY,
  ord INTEGER NOT NULL,
  name TEXT NOT NULL,
  archetype TEXT NOT NULL,
  tagline TEXT NOT NULL,
  summary TEXT NOT NULL,
  accent TEXT NOT NULL,
  north_star TEXT NOT NULL,
  pillars TEXT NOT NULL DEFAULT '[]',
  connectors TEXT NOT NULL DEFAULT '[]',
  metrics TEXT NOT NULL DEFAULT '[]',
  brain_use TEXT NOT NULL,
  signature_play TEXT NOT NULL
);
INSERT INTO personas VALUES('persona-marketing-agency',1,'Agency Owner','Agency Operator','Runs 14 client retainers on a 4-person team without dropping a ball','A marketing/creative agency owner who lives in client delivery, new-business pipeline, and reporting. This OS variant turns a small team''s chaos into a watched factory line: every retainer''s scope, deadlines, deliverables, and monthly report run on rails so nothing slips and no client churns silently.','#3df08c','Net retainer MRR retained (revenue kept, not just won)','[{"name":"New Business","focus":"Lead-to-signed-retainer pipeline and proposals","agents":["Pipeline Bot","Proposal Drafter","Discovery Scheduler","Win-Loss Logger"]},{"name":"Client Delivery","focus":"Retainer scope, deadlines, and deliverables across every account","agents":["Account Pilot","Deadline Sentinel","Scope Guard","QA Reviewer"]},{"name":"Creative Studio","focus":"Asset production, brand consistency, and approvals","agents":["Brief Builder","Asset Wrangler","Brand Cop","Approval Chaser"]},{"name":"Reporting & Retention","focus":"Monthly client reports, health scores, and churn defense","agents":["Report Compiler","Health Scorer","Churn Sentinel","QBR Prep"]},{"name":"Agency Ops","focus":"Team utilization, billing, and margin per account","agents":["Capacity Planner","Invoice Runner","Margin Watch","Time Auditor"]}]','["HubSpot CRM","Asana","Slack","Google Analytics 4","Meta Ads Manager","Google Ads","Harvest","QuickBooks Online"]','["Net retainer MRR","Client health score","On-time delivery rate","Team utilization %","Gross margin per account"]','Every agent reads the shared Brain for each client''s brand guidelines, scope-of-work, past approvals, and account history, so proposals, creative, and reports all speak in the right voice without re-briefing.','On the 1st of each month, Report Compiler pulls GA4, Meta, and Google Ads numbers for all active retainers, drafts a branded report per client in their voice, and Health Scorer flags any account trending toward churn so the owner walks into every QBR already knowing who''s at risk and who to upsell.');
INSERT INTO personas VALUES('persona-brick-and-mortar',2,'Multi-Location Operator','Brick-and-Mortar Founder','6 locations, one floor view — labor, inventory, reviews, traffic.','A multi-unit restaurant + retail founder running six rooftops from one console. The OS turns POS, labor, inventory, and local-reputation feeds into a single floor view so every store runs to target whether or not the owner is on-site.','#5ec9f8','Same-store sales per labor hour, blended across all locations','[{"name":"Floor Ops","focus":"Staffing, shifts, and labor cost per location","agents":["Dispatch Bot","Shift Filler","Labor Cop"]},{"name":"Inventory & Cost","focus":"Stock, COGS, vendor invoices, waste","agents":["Count Keeper","Invoice Reconciler","Waste Watcher","Reorder Bot"]},{"name":"Local Reputation","focus":"Reviews, local search, and listings across units","agents":["Review Harvester","Reply Bot","Listing Sentry"]},{"name":"Foot Traffic & Sales","focus":"Door count, throughput, daypart sales, loyalty","agents":["Traffic Counter","Daypart Analyst","Loyalty Driver"]},{"name":"Back Office","focus":"Payments, payroll, cash, multi-entity P&L","agents":["Settlement Bot","Payroll Runner","P&L Closer"]}]','["Toast POS","7shifts","Square","MarginEdge","Google Business Profile","Gusto","QuickBooks Online","Brain"]','["Same-store sales (YoY)","Labor cost % of sales","Food cost % (COGS)","Avg Google rating + review velocity","Door count vs conversion to ticket"]','Every agent reads the shared Brain — par levels, vendor terms, recipe costs, opening checklists, and per-store playbooks — so a fix at one location becomes the standard at all six.','Before open, the OS pulls last night''s Toast sales and today''s 7shifts roster, forecasts each store''s covers by daypart, flags any location running over labor target, auto-pings Shift Filler to cover call-outs, and drops a one-line per-store readiness brief — staffed, stocked, and reviews replied — on the owner''s phone by 7am.');
INSERT INTO personas VALUES('persona-ecommerce-dtc',3,'DTC Brand Operator','DTC Operator','One operator running a Shopify brand on margin, not vibes.','A solo ecommerce/DTC brand owner running a Shopify storefront, paid acquisition, 3PL fulfillment, and email/SMS retention from one deck. This OS variant watches contribution margin per order in real time and lets agents kill losing ad sets, chase abandoned carts, and flag stockouts before they cost a launch.','#e1306c','Contribution margin after ad spend (per order, blended)','[{"name":"Acquisition","focus":"Paid ads + traffic — spend that buys profitable orders","agents":["Spend Sentinel","Creative Tester","Attribution Bot"]},{"name":"Storefront & CRO","focus":"Shopify catalog, PDP, checkout, conversion rate","agents":["Catalog Bot","CRO Tester","Cart Recovery Bot"]},{"name":"Retention","focus":"Email/SMS flows, LTV, repeat-purchase engine","agents":["Flow Builder","Winback Bot","Review Harvester"]},{"name":"Fulfillment & Ops","focus":"Inventory, 3PL, shipping SLAs, stockout defense","agents":["Stockout Sentinel","Dispatch Bot","Returns Triage"]},{"name":"Margin & Finance","focus":"Unit economics, COGS, blended CAC, cash","agents":["Margin Watch","CAC Auditor","Reconcile Bot"]}]','["Shopify","Meta Ads Manager","Google Ads","TikTok Ads","Klaviyo","Postscript","ShipBob (3PL)","Stripe"]','["Contribution margin / order","Blended ROAS","CAC vs 60-day LTV","Repeat-purchase rate","Days of inventory on hand"]','Every agent reads from one shared Brain holding SKU economics, brand voice, hero-creative learnings, and customer segments — so a winning ad angle instantly informs Klaviyo flows and PDP copy.','Spend Sentinel cross-checks each ad set''s spend against real contribution margin from Shopify+COGS overnight, auto-pauses anything underwater, reallocates budget to the profitable hero creative, and fires a Klaviyo/Postscript winback to yesterday''s abandoned carts — so the operator wakes up to a brand that already cut its losers and chased its money.');
INSERT INTO personas VALUES('persona-online-coach',4,'Personal Online Coach','Transformation Coach','Clients win, stay, and refer — the practice runs itself.','A 1:1 and group transformation coach (fitness/health/mindset) running a hybrid book of premium private clients plus cohort programs. This OS variant turns coaching into an operating system: every lead nurtured, every client onboarded, every check-in answered, every at-risk client caught, and every win turned into a referral or testimonial.','#ffc53d','Active retained clients × 90-day transformation rate','[{"name":"Acquisition","focus":"Turn leads and discovery calls into signed clients","agents":["Lead Catcher","Booking Bot","Close Tracker"]},{"name":"Onboarding & Delivery","focus":"Intake, program build, and getting clients moving day one","agents":["Intake Bot","Program Builder","Loom Recap Bot"]},{"name":"Accountability & Results","focus":"Check-ins, habit streaks, and tracking transformation data","agents":["Check-in Bot","Streak Sentinel","Results Logger"]},{"name":"Retention & Renewals","focus":"Catch churn risk, drive renewals, harvest wins","agents":["Churn Sentinel","Renewal Closer","Win Harvester"]},{"name":"Brand & Lead-Gen","focus":"Content, community, and referral engine that fills the pipeline","agents":["Content Bot","Community Bot","Referral Bot"]}]','["Trainerize","Calendly","Stripe","Kajabi","Typeform","WhatsApp","Loom","Circle"]','["Active retained clients","90-day transformation rate","Check-in response rate","Renewal rate","Referral / testimonial count"]','Every agent reads one shared client brain — goals, injuries, macros, milestones, and last-conversation context — so check-ins, recaps, and renewals all sound like the coach who actually knows them.','Every client check-in is auto-pulled, scored against their goals, and either gets a personalized voice/text reply drafted or fires a churn flag — so nobody goes silent, no win goes uncaptured, and every transformation becomes a testimonial and a referral ask.');
INSERT INTO personas VALUES('persona-saas-founder',5,'SaaS Founder OS','SaaS Founder','Solo indie SaaS founder running product, growth, and MRR on autopilot','A bootstrapped indie software founder running a self-serve SaaS solo. This OS variant turns the whole funnel — signup to expansion to churn-save — into a closed loop of named agents so one person operates like a Series A growth team.','#a855f7','Net MRR growth (new + expansion − churn)','[{"name":"Acquisition","focus":"Top-of-funnel growth loops, SEO, and trial signups","agents":["Loop Engine","SEO Crawler","Trial Capture"]},{"name":"Activation","focus":"Onboarding new signups to first value (aha moment)","agents":["Onboarding Bot","Aha Tracker","Setup Nudger"]},{"name":"Retention","focus":"Churn prediction and win-back before cancellation","agents":["Churn Sentinel","Win-Back Agent","Expansion Hunter"]},{"name":"Support","focus":"Inbox triage, ticket deflection, and docs from real tickets","agents":["Ticket Triage","Deflection Bot","Docs Synth"]},{"name":"Revenue Ops","focus":"MRR, dunning, and every billing event in one view","agents":["MRR Pulse","Dunning Agent","Churn Accountant"]}]','["Stripe Billing","Intercom","PostHog","Customer.io","Linear","Ahrefs","Slack","Brain"]','["Net MRR","Trial-to-paid conversion %","Net revenue retention","Logo + revenue churn %","Activation rate (time-to-aha)"]','Every agent reads and writes the same Brain core — feature decisions, churn reasons, support answers, and onboarding playbooks — so a support reply, a docs page, and a win-back email all speak with one product memory.','Churn Sentinel watches PostHog usage decay + Stripe billing signals, scores at-risk accounts nightly, then fires Win-Back Agent to send a personalized Customer.io save offer and opens a Linear issue on the feature gap that caused the churn — a full detect-save-fix loop with no founder in the seat.');
INSERT INTO personas VALUES('persona-real-estate',6,'Real Estate Team Lead','Broker-Operator','Runs a producing team: leads in, listings up, deals closed.','A team lead / managing broker running a high-volume residential team. This OS variant turns a leaky lead-to-close pipeline into a machine: leads get worked in 60 seconds, listings get launched and syndicated, showings stay booked, and every transaction crosses the finish line with nothing falling through.','#22c55e','Net closed GCI per quarter (commission off closed-and-funded deals)','[{"name":"Lead Gen & Conversion","focus":"Inbound buyer/seller leads to booked appointments","agents":["Speed-to-Lead Bot","Drip Nurturer","ISA Caller"]},{"name":"Listings & Marketing","focus":"List, price, launch, and syndicate every property","agents":["Listing Launcher","CMA Pricer","Syndication Bot","Open House Bot"]},{"name":"Showings & Transactions","focus":"Showings booked and deals driven to close","agents":["Showing Scheduler","Deal Driver","Compliance Hawk"]},{"name":"Team & Accountability","focus":"Agent production, splits, and pipeline coverage","agents":["Roster Tracker","Lead Router","Split Calculator"]},{"name":"Client Care & Database","focus":"Past clients, reviews, and repeat/referral business","agents":["Review Harvester","SOI Reactivator","Closing Concierge"]}]','["MLS / RESO Web API","Follow Up Boss","dotloop","ShowingTime","Zillow Premier Agent","BombBomb"]','["Speed-to-lead (avg first-touch time)","Lead-to-appointment conversion %","Active listings + days on market","Pending-to-close ratio","GCI pipeline (pending + closed)"]','One shared Brain holds every lead''s history, property notes, pricing comps, and past-client relationships so any agent answering a call or text already knows the full context.','A Zillow lead hits the system at 11pm: Speed-to-Lead Bot texts within 60 seconds, qualifies budget and timeline against Brain, routes the hot one to the on-duty agent and books the showing in ShowingTime before the lead ever calls a competitor.');
INSERT INTO personas VALUES('persona-course-creator',7,'Course Creator / Info-Product Operator','Course Creator','Cohorts, evergreen funnels, and student wins on autopilot.','An info-product seller running paid cohorts and an always-on evergreen funnel. This OS variant orchestrates launches, keeps students completing the material, and harvests testimonials and affiliate revenue — so enrollments compound without the founder living in the dashboard.','#f59e0b','Enrollment revenue x student completion rate (revenue that actually finishes the course)','[{"name":"Funnel & Launch","focus":"Evergreen funnel + live launch carts, urgency, and cart-open ops","agents":["Cart Sentinel","Launch Conductor","Webinar Bot","Deadline Driver"]},{"name":"Audience & Content","focus":"Top-of-funnel content, lead magnets, and list growth that feeds the funnel","agents":["Lead-Magnet Bot","Content Repurposer","List Builder"]},{"name":"Student Success","focus":"Onboarding, completion nudges, community engagement, testimonial harvest","agents":["Onboarding Bot","Completion Sentinel","Win Harvester","Community Pulse"]},{"name":"Money & Recovery","focus":"Checkout revenue, payment plans, failed-payment dunning, refund/chargeback watch","agents":["Checkout Pulse","Dunning Bot","Refund Watch"]},{"name":"Affiliates & Partners","focus":"Affiliate recruiting, swipe copy, commission tracking, JV launch ops","agents":["Affiliate Recruiter","Commission Bot","Swipe Smith"]}]','["Kajabi","ThriveCart","ConvertKit","Circle","Deadline Funnel","Stripe","Zoom","PartnerStack"]','["Cart conversion rate","Course completion rate","Email list growth + EPC","Refund / chargeback rate","Affiliate-driven revenue %"]','Every agent reads from one shared Brain holding the course curriculum, swipe copy, student FAQs, objection-handling scripts, and past launch numbers — so onboarding nudges, sales replies, and affiliate swipe all speak in one consistent voice.','An evergreen launch loop: Deadline Driver opens a personal-deadline cart the moment a lead finishes the webinar, Cart Sentinel fires ConvertKit sequences and Dunning Bot rescues failed payment-plan charges, while Completion Sentinel pushes the new student to module 3 and Win Harvester captures the testimonial the instant they post a win in Circle — turning one enrollment into the proof that sells the next.');
INSERT INTO personas VALUES('persona-local-service',8,'Field Service OS','Home-Services Operator','Owner of the truck-and-tools shop. Every job booked, dispatched, paid.','A local HVAC/plumbing/contractor owner running a fleet of trucks. This OS variant turns the phone-and-whiteboard chaos of a service business into a closed loop: a missed call becomes a booked job, a booked job becomes a routed tech, a finished job becomes a paid invoice and a 5-star review, and every customer rolls into a recurring maintenance plan.','#0a85c2','Booked revenue per truck-day (jobs completed × average ticket, against capacity).','[{"name":"Intake & Booking","focus":"Turn every inbound call, form, and missed call into a booked job slot","agents":["Intake Bot","Missed-Call Catcher","Slot Booker"]},{"name":"Dispatch & Field Ops","focus":"Route the right tech to the right job and keep the board full","agents":["Dispatch Bot","Route Optimizer","Tech Tracker","On-My-Way Texter"]},{"name":"Quotes & Close","focus":"Build estimates fast, chase pending quotes, win the job","agents":["Estimate Builder","Quote Chaser","Upsell Prompter"]},{"name":"Money & Plans","focus":"Invoice on completion, collect payment, enroll recurring maintenance","agents":["Invoice Closer","Collections Bot","Maintenance Enroller","AR Sentinel"]},{"name":"Reputation & Repeat","focus":"Harvest reviews, win back lapsed customers, defend the local rank","agents":["Review Harvester","Reputation Guard","Win-Back Bot"]}]','["Housecall Pro","Jobber","ServiceTitan","CallRail","Stripe","QuickBooks Online","Google Business Profile","Twilio"]','["Booked revenue per truck-day","Call-to-booked conversion %","Average ticket","Quote win rate","Maintenance plan active members"]','Every agent reads one shared Brain core holding the price book, service-area map, tech skills/certs, warranty terms, and per-customer equipment history, so quotes, dispatch, and follow-ups all speak with the same numbers and the same memory of every property.','The missed-call-to-paid loop: a call that rings out at 7pm gets an instant text-back, books a next-day slot, auto-routes the nearest qualified tech, fires the "on my way" SMS at dawn, drops the invoice the second the job is marked complete, takes the card, and ten minutes later sends the review request — owner never touches the phone.');
INSERT INTO personas VALUES('persona-fractional-exec',9,'Fractional Exec OS','Fractional Operator','One part-time exec, six client orgs, run like one shop.','A fractional CFO/COO running advisory retainers across a portfolio of 6-8 client companies — selling frameworks and judgment, not hours. This OS variant keeps every retainer fed, every deliverable shipped on cadence, and every relationship warm enough to renew or refer, so utilization stays high without the operator living in their inbox.','#ff6259','Monthly recurring retainer revenue (MRR) at >85% portfolio utilization','[{"name":"Pipeline & Retainers","focus":"Inbound leads, scoping calls, proposals, and retainer renewals","agents":["Scope Bot","Proposal Drafter","Renewal Watchdog","Referral Tracer"]},{"name":"Client Delivery","focus":"Per-client workrooms, deliverables, and cadence on every active retainer","agents":["Engagement Lead","Deliverable Forge","Cadence Keeper","Status Reporter"]},{"name":"Thought Leadership","focus":"LinkedIn essays, newsletter, talks — the demand engine that feeds pipeline","agents":["Essay Drafter","Hook Miner","Repurpose Bot"]},{"name":"Practice Finance","focus":"Retainer invoicing, AR aging, utilization, and effective hourly across the book","agents":["Invoice Runner","AR Chaser","Utilization Auditor"]},{"name":"Relationship Ops","focus":"Every client/prospect inbox and DM into one triaged queue with context","agents":["Inbox Triage","Meeting Prep Bot","Follow-up Sentinel"]}]','["Ledger (CRM)","Gmail","Calendly","Notion (client workrooms)","Stripe (retainer billing)","QuickBooks","LinkedIn","Recall (call notes)"]','["Retainer MRR","Portfolio utilization %","Net revenue retention","Pipeline coverage (3x rule)","Effective hourly rate"]','Every client''s frameworks, decisions, board context, and prior deliverables live in one Brain so any agent drafts in that client''s voice and never re-asks what was already decided.','After each client call, the Recall transcript hits Brain, Deliverable Forge drafts the next memo/model in that client''s house style, Status Reporter posts the workroom update, and Cadence Keeper books the follow-up — the engagement advances before the operator''s next call ends.');
INSERT INTO personas VALUES('persona-community-operator',10,'Membership Community Operator','Community Operator','Creator running a paid membership — keep them in, keep them showing up.','A creator who turned an audience into a paid membership/community — recurring members, weekly live calls, a structured curriculum, and an always-on community feed. This OS variant runs the whole retention machine: it watches who''s drifting, fills every event seat, keeps the content cadence on schedule, and saves at-risk subscriptions before the card declines.','#8b7cff','Net MRR retention — members kept and expanded month over month','[{"name":"Membership Growth","focus":"Trials, waitlist, and free-to-paid conversion into the community","agents":["Waitlist Warden","Trial Closer","Referral Engine"]},{"name":"Retention & Churn","focus":"At-risk detection, dunning recovery, and cancel saves","agents":["Churn Sentinel","Dunning Recovery Bot","Win-Back Agent"]},{"name":"Community & Engagement","focus":"Daily feed health, member onboarding, and active-member streaks","agents":["Onboarding Concierge","Engagement Pulse","Ghost Hunter"]},{"name":"Events & Live","focus":"Weekly calls, RSVPs, attendance, and replay distribution","agents":["RSVP Wrangler","Live Producer","Replay Cutter"]},{"name":"Content & Curriculum","focus":"Drip schedule, lesson cadence, and member-only content pipeline","agents":["Drip Scheduler","Curriculum Keeper","Digest Writer"]}]','["Circle","Stripe Billing","ConvertKit","Discord","Zoom Webinars","Memberful","Calendly","Brain"]','["Net MRR retention","Monthly churn rate","Active-member rate (WAU/MAU)","Live-call attendance rate","Trial-to-paid conversion"]','Every agent reads one shared member memory — each person''s join date, tier, engagement history, last login, and save/win-back attempts — so saves, onboarding, and event nudges are personal instead of generic blasts.','Churn Sentinel flags a member who hasn''t logged in for 14 days and skipped two live calls, pulls their history from Brain, then fires a personalized win-back from Win-Back Agent and re-invites them to the next event via RSVP Wrangler — catching the lapse weeks before the renewal date instead of eating the cancel.');
INSERT INTO personas VALUES('persona-law-firm',11,'Boutique Law Firm OS','Managing Attorney','Every matter moved, every hour captured, every deadline safe.','A solo or small-firm managing attorney running a full caseload across intake, matter management, billing, and compliance. This OS variant turns a stack of open files into a watched docket: every deadline calculated, every billable minute captured, every trust dollar accounted for, and every client kept in the loop so nothing slips and no matter goes stale.','#b8860b','Collected realization rate (billed hours that actually get paid) at zero missed deadlines','[{"name":"Intake & Conflicts","focus":"Screen inquiries, clear conflicts, and sign engagements","agents":["Intake Screener","Conflict Checker","Engagement Drafter","Retainer Collector"]},{"name":"Matter Management","focus":"Every open matter''s tasks, deadlines, and court dates on rails","agents":["Matter Pilot","Docket Sentinel","Task Router","Limitations Watch"]},{"name":"Documents & Drafting","focus":"Draft, assemble, and review documents from templates and precedent","agents":["Draft Assembler","Clause Librarian","Redline Reviewer","e-Filing Bot"]},{"name":"Billing & Trust","focus":"Capture time, bill on cadence, and keep trust accounting clean","agents":["Time Capturer","Invoice Runner","Trust Ledger Guard","Collections Chaser"]},{"name":"Compliance & Client Care","focus":"Statutory deadlines, ethics compliance, and client communication","agents":["Deadline Guardian","CLE Tracker","Client Update Bot","Review Harvester"]}]','["Clio (practice management)","LawPay","NetDocuments","Lexis+ / Westlaw","Microsoft 365 (Outlook)","DocuSign","Court e-Filing (PACER / state)","QuickBooks Online"]','["Realization rate (billed vs collected)","Billable hours captured per attorney","Matters opened vs closed","Days to invoice (WIP aging)","Deadline / limitations compliance (missed = 0)"]','One shared Brain holds every matter''s facts, precedent, prior filings, client history, and firm playbooks, so any agent drafts from the firm''s own work product and never re-researches settled ground.','A new inquiry hits the system: Conflict Checker clears it against the client and matter graph in Brain, Engagement Drafter generates the retainer and fee agreement in the firm''s language, Retainer Collector takes the deposit through LawPay into the trust account, and Docket Sentinel opens the matter with every statutory deadline pre-calculated from the jurisdiction''s rules, so the attorney reviews and signs instead of building the file from scratch.');
CREATE TABLE phases (
  id TEXT PRIMARY KEY,
  number INTEGER NOT NULL,
  title TEXT NOT NULL,
  items TEXT NOT NULL DEFAULT '[]'
);
INSERT INTO phases VALUES('phase-1',1,'Real Connections','["4 email inboxes","Slack","Payment processors","Brain"]');
INSERT INTO phases VALUES('phase-2',2,'Real Agents','["Runtime + run log","Honest status board","On-demand runs"]');
INSERT INTO phases VALUES('phase-3',3,'Autonomy','["Scheduled runs","LLM digests","Failure alerts"]');
INSERT INTO phases VALUES('phase-4',4,'Dedicated Host','["Migrate compute","Remote access + auth","24/7 uptime"]');
CREATE TABLE agent_runs (
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT NOT NULL,
  ok INTEGER NOT NULL,
  summary TEXT NOT NULL DEFAULT ''
, model TEXT, tokens_in INTEGER, tokens_out INTEGER, cost_usd REAL);
INSERT INTO agent_runs VALUES('seed-run-conductor-0','conductor','2026-10-02T20:24:01.183Z','2026-10-02T20:24:07.813Z',1,'Conductor completed a run.','claude-sonnet-5',12108,980,0.05102399999999999992);
INSERT INTO agent_runs VALUES('seed-run-conductor-1','conductor','2026-10-01T22:03:31.925Z','2026-10-01T22:03:35.062Z',0,'Conductor run failed and was retried.','claude-sonnet-5',3296,2092,0.04126799999999999913);
INSERT INTO agent_runs VALUES('seed-run-conductor-2','conductor','2026-09-28T04:52:00.089Z','2026-09-28T04:52:06.513Z',1,'Conductor completed a run.','claude-sonnet-5',9280,1331,0.04780500000000000027);
INSERT INTO agent_runs VALUES('seed-run-conductor-3','conductor','2026-10-01T19:40:39.554Z','2026-10-01T19:40:41.852Z',1,'Conductor completed a run.','claude-sonnet-5',9673,2225,0.06239399999999999808);
INSERT INTO agent_runs VALUES('seed-run-conductor-4','conductor','2026-09-22T23:54:10.736Z','2026-09-22T23:54:13.449Z',1,'Conductor completed a run.','claude-sonnet-5',4905,3760,0.07111499999999999766);
INSERT INTO agent_runs VALUES('seed-run-conductor-5','conductor','2026-09-27T01:02:23.347Z','2026-09-27T01:02:26.118Z',1,'Conductor completed a run.','claude-sonnet-5',1512,3551,0.05780099999999999822);
INSERT INTO agent_runs VALUES('seed-run-conductor-6','conductor','2026-10-05T21:40:04.221Z','2026-10-05T21:40:07.585Z',1,'Conductor completed a run.','claude-sonnet-5',12836,1116,0.0552479999999999985);
INSERT INTO agent_runs VALUES('seed-run-conductor-7','conductor','2026-10-01T09:00:30.165Z','2026-10-01T09:00:31.918Z',1,'Conductor completed a run.','claude-sonnet-5',6640,2640,0.05952000000000000345);
INSERT INTO agent_runs VALUES('seed-run-conductor-8','conductor','2026-09-22T17:52:31.390Z','2026-09-22T17:52:32.764Z',0,'Conductor run failed and was retried.','claude-sonnet-5',1927,391,0.0116460000000000001);
INSERT INTO agent_runs VALUES('seed-run-conductor-9','conductor','2026-09-23T15:26:21.090Z','2026-09-23T15:26:23.425Z',1,'Conductor completed a run.','claude-sonnet-5',2041,413,0.01231800000000000082);
INSERT INTO agent_runs VALUES('seed-run-conductor-10','conductor','2026-10-08T03:47:30.027Z','2026-10-08T03:47:35.390Z',1,'Conductor completed a run.','claude-sonnet-5',7352,1889,0.0503909999999999983);
INSERT INTO agent_runs VALUES('seed-run-conductor-11','conductor','2026-09-24T14:46:42.585Z','2026-09-24T14:46:44.227Z',1,'Conductor completed a run.','claude-sonnet-5',14350,4075,0.1041750000000000037);
INSERT INTO agent_runs VALUES('seed-run-conductor-12','conductor','2026-09-29T22:56:03.006Z','2026-09-29T22:56:08.709Z',1,'Conductor completed a run.','claude-sonnet-5',7835,3905,0.08208000000000000018);
INSERT INTO agent_runs VALUES('seed-run-comms-digest-0','comms-digest','2026-10-03T00:24:56.990Z','2026-10-03T00:25:02.236Z',1,'Comms Digest completed a run.','claude-sonnet-5',7715,2778,0.06481499999999999762);
INSERT INTO agent_runs VALUES('seed-run-comms-digest-1','comms-digest','2026-10-07T05:50:51.881Z','2026-10-07T05:50:58.530Z',1,'Comms Digest completed a run.','claude-sonnet-5',13995,2504,0.07954500000000000459);
INSERT INTO agent_runs VALUES('seed-run-comms-digest-2','comms-digest','2026-09-25T18:33:14.762Z','2026-09-25T18:33:20.074Z',1,'Comms Digest completed a run.','claude-sonnet-5',12884,981,0.05336699999999999778);
INSERT INTO agent_runs VALUES('seed-run-comms-digest-3','comms-digest','2026-09-26T17:52:35.961Z','2026-09-26T17:52:43.100Z',1,'Comms Digest completed a run.','claude-sonnet-5',3681,4043,0.07168800000000000172);
INSERT INTO agent_runs VALUES('seed-run-comms-digest-4','comms-digest','2026-09-22T10:16:48.777Z','2026-09-22T10:16:54.219Z',1,'Comms Digest completed a run.','claude-sonnet-5',11662,3951,0.0942510000000000014);
INSERT INTO agent_runs VALUES('seed-run-comms-digest-5','comms-digest','2026-09-29T21:44:49.953Z','2026-09-29T21:44:52.670Z',1,'Comms Digest completed a run.','claude-sonnet-5',11614,649,0.04457699999999999857);
INSERT INTO agent_runs VALUES('seed-run-comms-digest-6','comms-digest','2026-10-09T00:27:23.762Z','2026-10-09T00:27:26.904Z',0,'Comms Digest run failed and was retried.','claude-sonnet-5',6294,2026,0.04927200000000000329);
INSERT INTO agent_runs VALUES('seed-run-comms-digest-7','comms-digest','2026-09-22T20:30:30.622Z','2026-09-22T20:30:32.715Z',1,'Comms Digest completed a run.','claude-sonnet-5',4243,1212,0.03090899999999999898);
INSERT INTO agent_runs VALUES('seed-run-comms-digest-8','comms-digest','2026-09-23T04:52:03.161Z','2026-09-23T04:52:03.611Z',1,'Comms Digest completed a run.','claude-sonnet-5',11957,1117,0.05262599999999999917);
INSERT INTO agent_runs VALUES('seed-run-comms-digest-9','comms-digest','2026-10-05T09:01:25.883Z','2026-10-05T09:01:31.867Z',1,'Comms Digest completed a run.','claude-sonnet-5',5388,2540,0.05426399999999999974);
INSERT INTO agent_runs VALUES('seed-run-comms-digest-10','comms-digest','2026-09-26T16:43:01.636Z','2026-09-26T16:43:02.684Z',1,'Comms Digest completed a run.','claude-sonnet-5',11243,1208,0.05184899999999999926);
INSERT INTO agent_runs VALUES('seed-run-comms-digest-11','comms-digest','2026-10-06T10:48:19.538Z','2026-10-06T10:48:26.778Z',1,'Comms Digest completed a run.','claude-sonnet-5',3745,1422,0.03256499999999999672);
INSERT INTO agent_runs VALUES('seed-run-comms-digest-12','comms-digest','2026-10-02T05:57:17.919Z','2026-10-02T05:57:20.132Z',1,'Comms Digest completed a run.','claude-sonnet-5',12303,3423,0.08825399999999999912);
INSERT INTO agent_runs VALUES('seed-run-comms-digest-13','comms-digest','2026-10-03T14:21:44.875Z','2026-10-03T14:21:48.194Z',1,'Comms Digest completed a run.','claude-sonnet-5',4935,4166,0.07729500000000000259);
INSERT INTO agent_runs VALUES('seed-run-comms-agent-0','comms-agent','2026-10-05T18:58:08.727Z','2026-10-05T18:58:11.999Z',1,'Comms Agent completed a run.','claude-sonnet-5',3043,2072,0.04020900000000000168);
INSERT INTO agent_runs VALUES('seed-run-comms-agent-1','comms-agent','2026-09-28T14:44:59.667Z','2026-09-28T14:45:01.600Z',1,'Comms Agent completed a run.','claude-sonnet-5',1958,2424,0.0422340000000000007);
INSERT INTO agent_runs VALUES('seed-run-comms-agent-2','comms-agent','2026-09-27T03:10:19.972Z','2026-09-27T03:10:20.828Z',1,'Comms Agent completed a run.','claude-sonnet-5',5709,3640,0.0717269999999999991);
INSERT INTO agent_runs VALUES('seed-run-comms-agent-3','comms-agent','2026-09-24T06:23:28.482Z','2026-09-24T06:23:30.078Z',1,'Comms Agent completed a run.','claude-sonnet-5',9817,1180,0.04715099999999999847);
INSERT INTO agent_runs VALUES('seed-run-comms-agent-4','comms-agent','2026-10-02T10:30:38.711Z','2026-10-02T10:30:41.692Z',1,'Comms Agent completed a run.','claude-sonnet-5',10454,229,0.03479700000000000154);
INSERT INTO agent_runs VALUES('seed-run-gmail-worker-0','gmail-worker','2026-10-06T07:48:10.658Z','2026-10-06T07:48:14.781Z',1,'Gmail Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-gmail-worker-1','gmail-worker','2026-09-23T04:43:08.276Z','2026-09-23T04:43:13.415Z',0,'Gmail Worker run failed and was retried.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-gmail-worker-2','gmail-worker','2026-09-23T20:42:54.481Z','2026-09-23T20:43:00.040Z',1,'Gmail Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-gmail-worker-3','gmail-worker','2026-10-05T03:01:23.406Z','2026-10-05T03:01:29.789Z',1,'Gmail Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-gmail-worker-4','gmail-worker','2026-09-20T22:42:26.288Z','2026-09-20T22:42:31.473Z',1,'Gmail Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-gmail-worker-5','gmail-worker','2026-09-26T18:51:17.590Z','2026-09-26T18:51:19.470Z',1,'Gmail Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-gmail-worker-6','gmail-worker','2026-09-29T17:59:27.874Z','2026-09-29T17:59:32.128Z',1,'Gmail Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-gmail-worker-7','gmail-worker','2026-10-01T01:59:15.269Z','2026-10-01T01:59:18.646Z',1,'Gmail Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-gmail-worker-8','gmail-worker','2026-10-05T20:43:45.739Z','2026-10-05T20:43:46.572Z',0,'Gmail Worker run failed and was retried.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-whatsapp-worker-0','whatsapp-worker','2026-10-09T02:03:53.238Z','2026-10-09T02:03:54.721Z',1,'WhatsApp Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-whatsapp-worker-1','whatsapp-worker','2026-09-22T19:18:24.542Z','2026-09-22T19:18:25.786Z',1,'WhatsApp Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-whatsapp-worker-2','whatsapp-worker','2026-10-05T07:36:54.745Z','2026-10-05T07:36:56.113Z',1,'WhatsApp Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-whatsapp-worker-3','whatsapp-worker','2026-09-29T09:13:24.311Z','2026-09-29T09:13:28.780Z',1,'WhatsApp Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-whatsapp-worker-4','whatsapp-worker','2026-09-20T21:33:08.889Z','2026-09-20T21:33:13.348Z',1,'WhatsApp Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-whatsapp-worker-5','whatsapp-worker','2026-09-20T21:22:53.948Z','2026-09-20T21:22:55.895Z',1,'WhatsApp Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-whatsapp-worker-6','whatsapp-worker','2026-10-04T02:09:41.295Z','2026-10-04T02:09:43.574Z',1,'WhatsApp Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-whatsapp-worker-7','whatsapp-worker','2026-09-21T16:50:12.742Z','2026-09-21T16:50:16.844Z',1,'WhatsApp Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-whatsapp-worker-8','whatsapp-worker','2026-10-07T14:35:02.168Z','2026-10-07T14:35:04.867Z',1,'WhatsApp Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-whatsapp-worker-9','whatsapp-worker','2026-10-02T00:38:46.708Z','2026-10-02T00:38:48.962Z',1,'WhatsApp Worker completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-slack-worker-0','slack-worker','2026-09-22T03:09:29.052Z','2026-09-22T03:09:31.879Z',1,'Slack Worker completed a run.','claude-haiku-4.5',13248,2022,0.01868600000000000121);
INSERT INTO agent_runs VALUES('seed-run-slack-worker-1','slack-worker','2026-10-01T19:28:02.507Z','2026-10-01T19:28:09.288Z',1,'Slack Worker completed a run.','claude-haiku-4.5',2047,1956,0.00946199999999999993);
INSERT INTO agent_runs VALUES('seed-run-slack-worker-2','slack-worker','2026-09-29T09:16:24.502Z','2026-09-29T09:16:25.507Z',1,'Slack Worker completed a run.','claude-haiku-4.5',10470,2010,0.01641600000000000003);
INSERT INTO agent_runs VALUES('seed-run-slack-worker-3','slack-worker','2026-10-05T01:02:35.749Z','2026-10-05T01:02:41.568Z',0,'Slack Worker run failed and was retried.','claude-haiku-4.5',7417,1842,0.01330199999999999959);
INSERT INTO agent_runs VALUES('seed-run-slack-worker-4','slack-worker','2026-09-23T13:53:36.395Z','2026-09-23T13:53:42.424Z',1,'Slack Worker completed a run.','claude-haiku-4.5',8674,1442,0.01270699999999999955);
INSERT INTO agent_runs VALUES('seed-run-social-agent-0','social-agent','2026-10-08T22:00:57.365Z','2026-10-08T22:00:58.447Z',1,'Social Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-social-agent-1','social-agent','2026-09-30T14:44:37.646Z','2026-09-30T14:44:42.808Z',1,'Social Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-social-agent-2','social-agent','2026-09-22T19:57:16.965Z','2026-09-22T19:57:18.723Z',0,'Social Agent run failed and was retried.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-social-agent-3','social-agent','2026-09-23T16:59:34.917Z','2026-09-23T16:59:39.895Z',1,'Social Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-social-agent-4','social-agent','2026-09-27T00:18:39.926Z','2026-09-27T00:18:45.759Z',1,'Social Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-social-agent-5','social-agent','2026-10-06T12:23:45.910Z','2026-10-06T12:23:50.025Z',1,'Social Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-social-agent-6','social-agent','2026-10-06T12:38:00.672Z','2026-10-06T12:38:05.859Z',1,'Social Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-brand-deal-agent-0','brand-deal-agent','2026-09-21T19:52:31.880Z','2026-09-21T19:52:34.505Z',1,'Brand Deal Agent completed a run.','claude-haiku-4.5',2299,4094,0.01821499999999999855);
INSERT INTO agent_runs VALUES('seed-run-brand-deal-agent-1','brand-deal-agent','2026-10-01T15:51:10.850Z','2026-10-01T15:51:15.290Z',1,'Brand Deal Agent completed a run.','claude-haiku-4.5',3689,591,0.005315000000000000279);
INSERT INTO agent_runs VALUES('seed-run-brand-deal-agent-2','brand-deal-agent','2026-09-24T11:10:50.136Z','2026-09-24T11:10:55.370Z',1,'Brand Deal Agent completed a run.','claude-haiku-4.5',12840,1879,0.01778800000000000173);
INSERT INTO agent_runs VALUES('seed-run-brand-deal-agent-3','brand-deal-agent','2026-10-02T09:53:32.054Z','2026-10-02T09:53:37.373Z',1,'Brand Deal Agent completed a run.','claude-haiku-4.5',4618,1026,0.007798000000000000243);
INSERT INTO agent_runs VALUES('seed-run-brand-deal-agent-4','brand-deal-agent','2026-09-19T21:24:28.291Z','2026-09-19T21:24:34.618Z',1,'Brand Deal Agent completed a run.','claude-haiku-4.5',13902,1622,0.01761000000000000065);
INSERT INTO agent_runs VALUES('seed-run-brand-deal-agent-5','brand-deal-agent','2026-09-26T23:21:29.225Z','2026-09-26T23:21:36.196Z',1,'Brand Deal Agent completed a run.','claude-haiku-4.5',5419,577,0.0066429999999999996);
INSERT INTO agent_runs VALUES('seed-run-brand-deal-agent-6','brand-deal-agent','2026-09-24T06:42:33.126Z','2026-09-24T06:42:35.418Z',1,'Brand Deal Agent completed a run.','claude-haiku-4.5',11709,1716,0.01623099999999999891);
INSERT INTO agent_runs VALUES('seed-run-brand-deal-agent-7','brand-deal-agent','2026-09-21T22:58:00.748Z','2026-09-21T22:58:02.754Z',1,'Brand Deal Agent completed a run.','claude-haiku-4.5',8191,3199,0.01934900000000000162);
INSERT INTO agent_runs VALUES('seed-run-brand-deal-agent-8','brand-deal-agent','2026-09-26T18:25:12.018Z','2026-09-26T18:25:15.803Z',1,'Brand Deal Agent completed a run.','claude-haiku-4.5',9209,814,0.01062300000000000049);
INSERT INTO agent_runs VALUES('seed-run-brand-deal-agent-9','brand-deal-agent','2026-10-05T06:23:15.304Z','2026-10-05T06:23:21.594Z',1,'Brand Deal Agent completed a run.','claude-haiku-4.5',5734,3022,0.01667499999999999886);
INSERT INTO agent_runs VALUES('seed-run-newsletter-agent-0','newsletter-agent','2026-09-26T19:34:45.343Z','2026-09-26T19:34:52.240Z',1,'Newsletter Agent completed a run.','claude-haiku-4.5',7909,3443,0.02009899999999999882);
INSERT INTO agent_runs VALUES('seed-run-newsletter-agent-1','newsletter-agent','2026-09-20T13:36:03.162Z','2026-09-20T13:36:04.989Z',1,'Newsletter Agent completed a run.','claude-haiku-4.5',1416,1491,0.007097000000000000009);
INSERT INTO agent_runs VALUES('seed-run-newsletter-agent-2','newsletter-agent','2026-09-29T10:18:11.737Z','2026-09-29T10:18:16.767Z',1,'Newsletter Agent completed a run.','claude-haiku-4.5',8774,547,0.009206999999999999907);
INSERT INTO agent_runs VALUES('seed-run-newsletter-agent-3','newsletter-agent','2026-09-21T01:54:49.859Z','2026-09-21T01:54:53.268Z',1,'Newsletter Agent completed a run.','claude-haiku-4.5',11406,1147,0.0137129999999999995);
INSERT INTO agent_runs VALUES('seed-run-newsletter-agent-4','newsletter-agent','2026-09-26T23:03:24.401Z','2026-09-26T23:03:28.976Z',1,'Newsletter Agent completed a run.','claude-haiku-4.5',6034,585,0.00716699999999999976);
INSERT INTO agent_runs VALUES('seed-run-newsletter-agent-5','newsletter-agent','2026-10-01T16:13:12.340Z','2026-10-01T16:13:18.555Z',0,'Newsletter Agent run failed and was retried.','claude-haiku-4.5',3064,2016,0.01051500000000000004);
INSERT INTO agent_runs VALUES('seed-run-newsletter-agent-6','newsletter-agent','2026-10-04T22:58:07.578Z','2026-10-04T22:58:08.032Z',1,'Newsletter Agent completed a run.','claude-haiku-4.5',9141,3405,0.02093300000000000021);
INSERT INTO agent_runs VALUES('seed-run-newsletter-agent-7','newsletter-agent','2026-09-20T04:49:41.625Z','2026-09-20T04:49:45.197Z',1,'Newsletter Agent completed a run.','claude-haiku-4.5',6689,895,0.008930999999999999703);
INSERT INTO agent_runs VALUES('seed-run-newsletter-agent-8','newsletter-agent','2026-09-19T23:05:22.279Z','2026-09-19T23:05:27.848Z',1,'Newsletter Agent completed a run.','claude-haiku-4.5',5929,1421,0.0104270000000000005);
INSERT INTO agent_runs VALUES('seed-run-newsletter-agent-9','newsletter-agent','2026-10-03T00:18:31.779Z','2026-10-03T00:18:32.623Z',1,'Newsletter Agent completed a run.','claude-haiku-4.5',4202,2631,0.01388600000000000077);
INSERT INTO agent_runs VALUES('seed-run-postly-publisher-0','postly-publisher','2026-10-04T06:22:09.837Z','2026-10-04T06:22:10.913Z',1,'Postly Publisher completed a run.','claude-haiku-4.5',11515,3933,0.02494400000000000089);
INSERT INTO agent_runs VALUES('seed-run-postly-publisher-1','postly-publisher','2026-09-21T01:22:42.617Z','2026-09-21T01:22:43.447Z',1,'Postly Publisher completed a run.','claude-haiku-4.5',2627,765,0.005161999999999999922);
INSERT INTO agent_runs VALUES('seed-run-postly-publisher-2','postly-publisher','2026-09-24T13:38:44.970Z','2026-09-24T13:38:48.876Z',1,'Postly Publisher completed a run.','claude-haiku-4.5',9899,1798,0.01511099999999999944);
INSERT INTO agent_runs VALUES('seed-run-postly-publisher-3','postly-publisher','2026-09-28T19:25:35.782Z','2026-09-28T19:25:37.206Z',1,'Postly Publisher completed a run.','claude-haiku-4.5',2971,1288,0.007529000000000000102);
INSERT INTO agent_runs VALUES('seed-run-postly-publisher-4','postly-publisher','2026-10-08T19:43:18.535Z','2026-10-08T19:43:25.592Z',1,'Postly Publisher completed a run.','claude-haiku-4.5',11715,2187,0.01812000000000000068);
INSERT INTO agent_runs VALUES('seed-run-postly-publisher-5','postly-publisher','2026-10-02T07:09:27.533Z','2026-10-02T07:09:29.224Z',1,'Postly Publisher completed a run.','claude-haiku-4.5',8853,1172,0.01176999999999999922);
INSERT INTO agent_runs VALUES('seed-run-adsmith-creative-0','adsmith-creative','2026-10-05T06:09:30.311Z','2026-10-05T06:09:36.339Z',1,'Adsmith Creative completed a run.','claude-haiku-4.5',11830,1373,0.01495600000000000054);
INSERT INTO agent_runs VALUES('seed-run-adsmith-creative-1','adsmith-creative','2026-10-02T03:14:51.985Z','2026-10-02T03:14:54.286Z',1,'Adsmith Creative completed a run.','claude-haiku-4.5',2686,2070,0.01042900000000000077);
INSERT INTO agent_runs VALUES('seed-run-adsmith-creative-2','adsmith-creative','2026-09-25T00:29:17.902Z','2026-09-25T00:29:22.488Z',1,'Adsmith Creative completed a run.','claude-haiku-4.5',2421,3383,0.01546900000000000011);
INSERT INTO agent_runs VALUES('seed-run-adsmith-creative-3','adsmith-creative','2026-10-04T10:08:15.020Z','2026-10-04T10:08:21.747Z',1,'Adsmith Creative completed a run.','claude-haiku-4.5',8918,1439,0.01289000000000000041);
INSERT INTO agent_runs VALUES('seed-run-adsmith-creative-4','adsmith-creative','2026-10-08T04:30:46.568Z','2026-10-08T04:30:49.838Z',1,'Adsmith Creative completed a run.','claude-haiku-4.5',14405,710,0.01436400000000000003);
INSERT INTO agent_runs VALUES('seed-run-adsmith-creative-5','adsmith-creative','2026-09-25T14:56:15.748Z','2026-09-25T14:56:18.024Z',1,'Adsmith Creative completed a run.','claude-haiku-4.5',1699,1866,0.008822999999999999246);
INSERT INTO agent_runs VALUES('seed-run-adsmith-creative-6','adsmith-creative','2026-09-26T15:17:15.878Z','2026-09-26T15:17:21.389Z',1,'Adsmith Creative completed a run.','claude-haiku-4.5',7686,674,0.008845000000000000431);
INSERT INTO agent_runs VALUES('seed-run-adsmith-creative-7','adsmith-creative','2026-09-25T22:52:34.614Z','2026-09-25T22:52:41.027Z',1,'Adsmith Creative completed a run.','claude-haiku-4.5',7561,201,0.006852999999999999717);
INSERT INTO agent_runs VALUES('seed-run-adsmith-creative-8','adsmith-creative','2026-09-27T21:06:14.710Z','2026-09-27T21:06:20.153Z',1,'Adsmith Creative completed a run.','claude-haiku-4.5',12005,3655,0.02422399999999999901);
INSERT INTO agent_runs VALUES('seed-run-adsmith-creative-9','adsmith-creative','2026-10-05T11:02:31.231Z','2026-10-05T11:02:38.031Z',1,'Adsmith Creative completed a run.','claude-haiku-4.5',10755,1951,0.01640799999999999898);
INSERT INTO agent_runs VALUES('seed-run-adsmith-creative-10','adsmith-creative','2026-09-20T21:27:39.595Z','2026-09-20T21:27:41.255Z',1,'Adsmith Creative completed a run.','claude-haiku-4.5',7510,814,0.00926399999999999967);
INSERT INTO agent_runs VALUES('seed-run-adsmith-creative-11','adsmith-creative','2026-09-20T12:51:16.122Z','2026-09-20T12:51:19.116Z',1,'Adsmith Creative completed a run.','claude-haiku-4.5',7507,3520,0.02008599999999999969);
INSERT INTO agent_runs VALUES('seed-run-adsmith-creative-12','adsmith-creative','2026-09-26T05:39:44.133Z','2026-09-26T05:39:48.511Z',1,'Adsmith Creative completed a run.','claude-haiku-4.5',9685,3662,0.02239599999999999925);
INSERT INTO agent_runs VALUES('seed-run-adsmith-creative-13','adsmith-creative','2026-10-07T02:15:35.178Z','2026-10-07T02:15:38.279Z',0,'Adsmith Creative run failed and was retried.','claude-haiku-4.5',6050,288,0.005991999999999999931);
INSERT INTO agent_runs VALUES('seed-run-reelkit-editor-0','reelkit-editor','2026-10-01T01:15:49.617Z','2026-10-01T01:15:56.561Z',1,'Reelkit Editor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-reelkit-editor-1','reelkit-editor','2026-09-20T21:29:40.186Z','2026-09-20T21:29:46.074Z',1,'Reelkit Editor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-reelkit-editor-2','reelkit-editor','2026-10-09T12:10:13.482Z','2026-10-09T12:10:14.590Z',1,'Reelkit Editor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-reelkit-editor-3','reelkit-editor','2026-09-20T19:53:29.160Z','2026-09-20T19:53:33.218Z',1,'Reelkit Editor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-reelkit-editor-4','reelkit-editor','2026-10-07T09:17:11.118Z','2026-10-07T09:17:15.007Z',1,'Reelkit Editor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-reelkit-editor-5','reelkit-editor','2026-10-03T08:04:37.549Z','2026-10-03T08:04:43.174Z',1,'Reelkit Editor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-renderly-creative-0','renderly-creative','2026-09-29T02:41:45.180Z','2026-09-29T02:41:49.385Z',1,'Renderly Creative completed a run.','claude-haiku-4.5',5441,888,0.007905000000000000568);
INSERT INTO agent_runs VALUES('seed-run-renderly-creative-1','renderly-creative','2026-09-30T09:17:27.780Z','2026-09-30T09:17:29.372Z',1,'Renderly Creative completed a run.','claude-haiku-4.5',2190,219,0.002628000000000000127);
INSERT INTO agent_runs VALUES('seed-run-renderly-creative-2','renderly-creative','2026-09-24T18:43:24.881Z','2026-09-24T18:43:27.446Z',1,'Renderly Creative completed a run.','claude-haiku-4.5',4530,2060,0.01186399999999999955);
INSERT INTO agent_runs VALUES('seed-run-renderly-creative-3','renderly-creative','2026-10-07T23:36:15.019Z','2026-10-07T23:36:17.812Z',1,'Renderly Creative completed a run.','claude-haiku-4.5',5076,1842,0.01142899999999999993);
INSERT INTO agent_runs VALUES('seed-run-renderly-creative-4','renderly-creative','2026-09-24T04:43:01.412Z','2026-09-24T04:43:05.289Z',1,'Renderly Creative completed a run.','claude-haiku-4.5',7888,604,0.008725999999999999381);
INSERT INTO agent_runs VALUES('seed-run-renderly-creative-5','renderly-creative','2026-09-28T22:21:40.151Z','2026-09-28T22:21:41.282Z',1,'Renderly Creative completed a run.','claude-haiku-4.5',9780,1192,0.01259200000000000076);
INSERT INTO agent_runs VALUES('seed-run-renderly-creative-6','renderly-creative','2026-10-02T06:42:59.008Z','2026-10-02T06:43:03.886Z',1,'Renderly Creative completed a run.','claude-haiku-4.5',12268,1293,0.01498599999999999933);
INSERT INTO agent_runs VALUES('seed-run-renderly-creative-7','renderly-creative','2026-09-26T02:21:10.693Z','2026-09-26T02:21:17.680Z',1,'Renderly Creative completed a run.','claude-haiku-4.5',8307,630,0.009166000000000000537);
INSERT INTO agent_runs VALUES('seed-run-renderly-creative-8','renderly-creative','2026-10-08T15:59:06.113Z','2026-10-08T15:59:08.034Z',1,'Renderly Creative completed a run.','claude-haiku-4.5',12310,1663,0.01650000000000000077);
INSERT INTO agent_runs VALUES('seed-run-renderly-creative-9','renderly-creative','2026-09-21T19:42:32.394Z','2026-09-21T19:42:32.867Z',1,'Renderly Creative completed a run.','claude-haiku-4.5',6272,3707,0.01984599999999999906);
INSERT INTO agent_runs VALUES('seed-run-renderly-creative-10','renderly-creative','2026-10-03T14:40:51.294Z','2026-10-03T14:40:57.816Z',1,'Renderly Creative completed a run.','claude-haiku-4.5',12726,777,0.01328900000000000046);
INSERT INTO agent_runs VALUES('seed-run-renderly-creative-11','renderly-creative','2026-09-21T04:13:33.207Z','2026-09-21T04:13:38.259Z',1,'Renderly Creative completed a run.','claude-haiku-4.5',8822,851,0.01046200000000000082);
INSERT INTO agent_runs VALUES('seed-run-renderly-creative-12','renderly-creative','2026-09-27T18:00:30.979Z','2026-09-27T18:00:37.959Z',1,'Renderly Creative completed a run.','claude-haiku-4.5',4968,3578,0.01828600000000000017);
INSERT INTO agent_runs VALUES('seed-run-dmflow-mcp-0','dmflow-mcp','2026-10-05T22:02:30.094Z','2026-10-05T22:02:32.266Z',1,'DMFlow MCP completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-dmflow-mcp-1','dmflow-mcp','2026-10-07T16:26:46.167Z','2026-10-07T16:26:53.132Z',1,'DMFlow MCP completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-dmflow-mcp-2','dmflow-mcp','2026-09-27T02:32:02.873Z','2026-09-27T02:32:09.661Z',1,'DMFlow MCP completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-dmflow-mcp-3','dmflow-mcp','2026-10-05T01:54:48.019Z','2026-10-05T01:54:49.747Z',1,'DMFlow MCP completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-dmflow-mcp-4','dmflow-mcp','2026-09-22T07:02:28.753Z','2026-09-22T07:02:30.064Z',1,'DMFlow MCP completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-dmflow-mcp-5','dmflow-mcp','2026-09-30T10:19:00.278Z','2026-09-30T10:19:06.465Z',1,'DMFlow MCP completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-dmflow-mcp-6','dmflow-mcp','2026-10-07T07:33:00.694Z','2026-10-07T07:33:02.482Z',1,'DMFlow MCP completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-agent-0','sales-agent','2026-09-29T07:17:13.802Z','2026-09-29T07:17:16.355Z',1,'Sales Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-agent-1','sales-agent','2026-09-29T11:25:19.502Z','2026-09-29T11:25:26.247Z',0,'Sales Agent run failed and was retried.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-agent-2','sales-agent','2026-09-20T21:02:03.967Z','2026-09-20T21:02:10.720Z',1,'Sales Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-agent-3','sales-agent','2026-10-02T22:28:36.705Z','2026-10-02T22:28:37.306Z',1,'Sales Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-agent-4','sales-agent','2026-10-09T00:06:32.327Z','2026-10-09T00:06:33.440Z',1,'Sales Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-agent-5','sales-agent','2026-09-24T22:12:12.090Z','2026-09-24T22:12:16.149Z',1,'Sales Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-agent-6','sales-agent','2026-10-06T06:10:25.819Z','2026-10-06T06:10:31.218Z',1,'Sales Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-agent-7','sales-agent','2026-09-24T20:56:42.482Z','2026-09-24T20:56:45.326Z',1,'Sales Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-agent-8','sales-agent','2026-10-06T20:42:29.157Z','2026-10-06T20:42:35.507Z',1,'Sales Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-agent-9','sales-agent','2026-09-21T13:39:12.201Z','2026-09-21T13:39:15.668Z',1,'Sales Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-agent-10','sales-agent','2026-09-29T06:28:55.465Z','2026-09-29T06:28:58.923Z',1,'Sales Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-agent-11','sales-agent','2026-10-08T10:59:44.489Z','2026-10-08T10:59:50.473Z',1,'Sales Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-agent-12','sales-agent','2026-09-22T22:46:39.890Z','2026-09-22T22:46:47.222Z',1,'Sales Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-agent-13','sales-agent','2026-10-06T22:48:34.606Z','2026-10-06T22:48:37.013Z',1,'Sales Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-launchpad-cohort-sales-0','launchpad-cohort-sales','2026-09-29T08:10:38.288Z','2026-09-29T08:10:40.184Z',1,'Launchpad Cohort completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-launchpad-cohort-sales-1','launchpad-cohort-sales','2026-10-07T09:59:56.892Z','2026-10-07T09:59:58.527Z',1,'Launchpad Cohort completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-launchpad-cohort-sales-2','launchpad-cohort-sales','2026-09-24T04:02:13.226Z','2026-09-24T04:02:16.006Z',1,'Launchpad Cohort completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-launchpad-cohort-sales-3','launchpad-cohort-sales','2026-10-07T03:34:35.641Z','2026-10-07T03:34:41.507Z',1,'Launchpad Cohort completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-launchpad-cohort-sales-4','launchpad-cohort-sales','2026-10-04T10:03:30.183Z','2026-10-04T10:03:30.646Z',1,'Launchpad Cohort completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-launchpad-cohort-sales-5','launchpad-cohort-sales','2026-09-22T01:19:13.843Z','2026-09-22T01:19:18.221Z',1,'Launchpad Cohort completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-launchpad-cohort-sales-6','launchpad-cohort-sales','2026-10-08T14:14:28.648Z','2026-10-08T14:14:32.029Z',1,'Launchpad Cohort completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-vantage-sales-0','vantage-sales','2026-10-04T11:40:00.477Z','2026-10-04T11:40:07.541Z',1,'Vantage completed a run.','claude-haiku-4.5',6839,806,0.008694999999999999605);
INSERT INTO agent_runs VALUES('seed-run-vantage-sales-1','vantage-sales','2026-10-04T04:23:54.327Z','2026-10-04T04:24:00.865Z',1,'Vantage completed a run.','claude-haiku-4.5',2173,1827,0.009046000000000000221);
INSERT INTO agent_runs VALUES('seed-run-vantage-sales-2','vantage-sales','2026-10-06T06:30:33.545Z','2026-10-06T06:30:35.020Z',1,'Vantage completed a run.','claude-haiku-4.5',1938,1740,0.008510000000000000203);
INSERT INTO agent_runs VALUES('seed-run-vantage-sales-3','vantage-sales','2026-10-06T23:17:46.589Z','2026-10-06T23:17:52.105Z',1,'Vantage completed a run.','claude-haiku-4.5',6202,234,0.005897999999999999598);
INSERT INTO agent_runs VALUES('seed-run-vantage-sales-4','vantage-sales','2026-10-03T14:32:24.096Z','2026-10-03T14:32:25.341Z',0,'Vantage run failed and was retried.','claude-haiku-4.5',11355,2885,0.02062399999999999998);
INSERT INTO agent_runs VALUES('seed-run-vantage-sales-5','vantage-sales','2026-09-24T11:06:30.015Z','2026-09-24T11:06:36.987Z',1,'Vantage completed a run.','claude-haiku-4.5',1468,345,0.002553999999999999847);
INSERT INTO agent_runs VALUES('seed-run-vantage-sales-6','vantage-sales','2026-09-24T11:18:21.752Z','2026-09-24T11:18:22.348Z',1,'Vantage completed a run.','claude-haiku-4.5',4319,2654,0.01407100000000000017);
INSERT INTO agent_runs VALUES('seed-run-vantage-sales-7','vantage-sales','2026-09-27T09:28:47.945Z','2026-09-27T09:28:48.593Z',0,'Vantage run failed and was retried.','claude-haiku-4.5',3068,4092,0.01882199999999999845);
INSERT INTO agent_runs VALUES('seed-run-vantage-sales-8','vantage-sales','2026-10-06T18:24:56.874Z','2026-10-06T18:25:03.320Z',1,'Vantage completed a run.','claude-haiku-4.5',6269,2935,0.01675499999999999907);
INSERT INTO agent_runs VALUES('seed-run-vantage-sales-9','vantage-sales','2026-09-27T22:24:34.550Z','2026-09-27T22:24:41.112Z',1,'Vantage completed a run.','claude-haiku-4.5',7216,3214,0.01862899999999999973);
INSERT INTO agent_runs VALUES('seed-run-vantage-sales-10','vantage-sales','2026-10-05T17:43:23.590Z','2026-10-05T17:43:29.407Z',0,'Vantage run failed and was retried.','claude-haiku-4.5',10720,1225,0.01347600000000000013);
INSERT INTO agent_runs VALUES('seed-run-vantage-sales-11','vantage-sales','2026-09-28T09:53:56.390Z','2026-09-28T09:54:00.947Z',1,'Vantage completed a run.','claude-haiku-4.5',11674,326,0.01064299999999999969);
INSERT INTO agent_runs VALUES('seed-run-vantage-sales-12','vantage-sales','2026-10-07T21:56:57.580Z','2026-10-07T21:56:59.244Z',1,'Vantage completed a run.','claude-haiku-4.5',12500,2874,0.02149600000000000121);
INSERT INTO agent_runs VALUES('seed-run-paykit-sales-0','paykit-sales','2026-10-06T22:56:35.310Z','2026-10-06T22:56:39.142Z',1,'PayKit completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-paykit-sales-1','paykit-sales','2026-10-03T14:00:07.965Z','2026-10-03T14:00:12.181Z',1,'PayKit completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-paykit-sales-2','paykit-sales','2026-10-08T18:42:12.083Z','2026-10-08T18:42:15.366Z',1,'PayKit completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-paykit-sales-3','paykit-sales','2026-09-28T20:39:42.640Z','2026-09-28T20:39:46.929Z',1,'PayKit completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-paykit-sales-4','paykit-sales','2026-09-21T07:40:11.394Z','2026-09-21T07:40:17.287Z',0,'PayKit run failed and was retried.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-vantage-paykit-0','vantage-paykit','2026-10-07T13:11:29.884Z','2026-10-07T13:11:32.571Z',1,'Vantage PayKit completed a run.','claude-haiku-4.5',7790,3339,0.01958800000000000124);
INSERT INTO agent_runs VALUES('seed-run-vantage-paykit-1','vantage-paykit','2026-09-23T23:19:22.598Z','2026-09-23T23:19:24.363Z',1,'Vantage PayKit completed a run.','claude-haiku-4.5',11942,2419,0.01923000000000000056);
INSERT INTO agent_runs VALUES('seed-run-vantage-paykit-2','vantage-paykit','2026-10-04T16:53:20.987Z','2026-10-04T16:53:25.688Z',1,'Vantage PayKit completed a run.','claude-haiku-4.5',14586,2678,0.02238100000000000159);
INSERT INTO agent_runs VALUES('seed-run-vantage-paykit-3','vantage-paykit','2026-09-23T19:26:23.608Z','2026-09-23T19:26:27.002Z',1,'Vantage PayKit completed a run.','claude-haiku-4.5',5752,1767,0.01166999999999999982);
INSERT INTO agent_runs VALUES('seed-run-vantage-paykit-4','vantage-paykit','2026-10-05T18:16:40.519Z','2026-10-05T18:16:45.348Z',0,'Vantage PayKit run failed and was retried.','claude-haiku-4.5',3005,618,0.004876000000000000125);
INSERT INTO agent_runs VALUES('seed-run-vantage-paykit-5','vantage-paykit','2026-09-21T15:22:09.072Z','2026-09-21T15:22:13.671Z',1,'Vantage PayKit completed a run.','claude-haiku-4.5',6799,3061,0.0176830000000000008);
INSERT INTO agent_runs VALUES('seed-run-vantage-paykit-6','vantage-paykit','2026-09-21T06:49:16.126Z','2026-09-21T06:49:22.808Z',1,'Vantage PayKit completed a run.','claude-haiku-4.5',11082,4046,0.02504999999999999936);
INSERT INTO agent_runs VALUES('seed-run-vantage-paykit-7','vantage-paykit','2026-09-22T16:25:09.268Z','2026-09-22T16:25:16.426Z',1,'Vantage PayKit completed a run.','claude-haiku-4.5',5815,2974,0.0165480000000000002);
INSERT INTO agent_runs VALUES('seed-run-vantage-paykit-8','vantage-paykit','2026-09-20T10:02:29.172Z','2026-09-20T10:02:30.677Z',1,'Vantage PayKit completed a run.','claude-haiku-4.5',13043,2433,0.0201659999999999999);
INSERT INTO agent_runs VALUES('seed-run-vantage-paykit-9','vantage-paykit','2026-09-29T06:56:05.544Z','2026-09-29T06:56:12.285Z',1,'Vantage PayKit completed a run.','claude-haiku-4.5',11565,3626,0.02375599999999999934);
INSERT INTO agent_runs VALUES('seed-run-vantage-paykit-10','vantage-paykit','2026-09-19T23:22:15.100Z','2026-09-19T23:22:17.562Z',0,'Vantage PayKit run failed and was retried.','claude-haiku-4.5',12223,3795,0.02495800000000000101);
INSERT INTO agent_runs VALUES('seed-run-vantage-paykit-11','vantage-paykit','2026-09-25T20:02:25.108Z','2026-09-25T20:02:30.171Z',1,'Vantage PayKit completed a run.','claude-haiku-4.5',9358,2873,0.01897799999999999834);
INSERT INTO agent_runs VALUES('seed-run-vantage-paykit-12','vantage-paykit','2026-09-22T00:07:03.529Z','2026-09-22T00:07:09.303Z',1,'Vantage PayKit completed a run.','claude-haiku-4.5',12128,3204,0.02251799999999999983);
INSERT INTO agent_runs VALUES('seed-run-vantage-paykit-13','vantage-paykit','2026-10-02T23:36:29.705Z','2026-10-02T23:36:36.651Z',1,'Vantage PayKit completed a run.','claude-haiku-4.5',13818,3348,0.02444599999999999899);
INSERT INTO agent_runs VALUES('seed-run-stripe-sales-0','stripe-sales','2026-09-21T15:30:27.072Z','2026-09-21T15:30:32.498Z',1,'Stripe completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stripe-sales-1','stripe-sales','2026-09-27T18:33:12.583Z','2026-09-27T18:33:15.807Z',1,'Stripe completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stripe-sales-2','stripe-sales','2026-09-24T07:45:47.599Z','2026-09-24T07:45:49.474Z',1,'Stripe completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stripe-sales-3','stripe-sales','2026-10-05T12:20:50.667Z','2026-10-05T12:20:51.439Z',1,'Stripe completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stripe-sales-4','stripe-sales','2026-09-20T09:46:44.253Z','2026-09-20T09:46:50.776Z',1,'Stripe completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stripe-sales-5','stripe-sales','2026-10-02T11:26:48.416Z','2026-10-02T11:26:51.484Z',1,'Stripe completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stripe-sales-6','stripe-sales','2026-09-27T12:15:34.216Z','2026-09-27T12:15:39.156Z',1,'Stripe completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stripe-sales-7','stripe-sales','2026-09-21T02:09:18.720Z','2026-09-21T02:09:19.587Z',1,'Stripe completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stripe-sales-8','stripe-sales','2026-10-07T11:08:20.006Z','2026-10-07T11:08:23.516Z',1,'Stripe completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stripe-sales-9','stripe-sales','2026-10-01T17:58:41.270Z','2026-10-01T17:58:42.246Z',1,'Stripe completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stripe-sales-10','stripe-sales','2026-09-20T21:08:04.146Z','2026-09-20T21:08:04.633Z',1,'Stripe completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stripe-sales-11','stripe-sales','2026-10-03T09:55:22.698Z','2026-10-03T09:55:24.651Z',1,'Stripe completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stripe-sales-12','stripe-sales','2026-10-01T14:32:58.490Z','2026-10-01T14:33:04.151Z',1,'Stripe completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-processor-confirmation-0','processor-confirmation','2026-10-03T05:40:16.758Z','2026-10-03T05:40:19.769Z',1,'Processor Confirm completed a run.','claude-haiku-4.5',10052,820,0.01132200000000000046);
INSERT INTO agent_runs VALUES('seed-run-processor-confirmation-1','processor-confirmation','2026-10-02T13:08:09.295Z','2026-10-02T13:08:13.799Z',0,'Processor Confirm run failed and was retried.','claude-haiku-4.5',3401,1055,0.006941000000000000121);
INSERT INTO agent_runs VALUES('seed-run-processor-confirmation-2','processor-confirmation','2026-10-06T11:16:10.214Z','2026-10-06T11:16:11.074Z',1,'Processor Confirm completed a run.','claude-haiku-4.5',12165,3304,0.02294799999999999965);
INSERT INTO agent_runs VALUES('seed-run-processor-confirmation-3','processor-confirmation','2026-10-09T06:47:19.287Z','2026-10-09T06:47:23.969Z',1,'Processor Confirm completed a run.','claude-haiku-4.5',6768,3614,0.01986999999999999878);
INSERT INTO agent_runs VALUES('seed-run-processor-confirmation-4','processor-confirmation','2026-10-07T11:09:14.577Z','2026-10-07T11:09:18.350Z',1,'Processor Confirm completed a run.','claude-haiku-4.5',12616,625,0.01259300000000000002);
INSERT INTO agent_runs VALUES('seed-run-processor-confirmation-5','processor-confirmation','2026-10-07T14:32:58.420Z','2026-10-07T14:33:03.216Z',1,'Processor Confirm completed a run.','claude-haiku-4.5',3405,3114,0.01518000000000000078);
INSERT INTO agent_runs VALUES('seed-run-processor-confirmation-6','processor-confirmation','2026-09-25T03:11:30.143Z','2026-09-25T03:11:34.794Z',1,'Processor Confirm completed a run.','claude-haiku-4.5',2161,1727,0.00863700000000000058);
INSERT INTO agent_runs VALUES('seed-run-processor-confirmation-7','processor-confirmation','2026-09-28T23:00:56.341Z','2026-09-28T23:01:01.755Z',1,'Processor Confirm completed a run.','claude-haiku-4.5',12022,982,0.01354600000000000074);
INSERT INTO agent_runs VALUES('seed-run-processor-confirmation-8','processor-confirmation','2026-09-25T01:27:05.119Z','2026-09-25T01:27:06.365Z',1,'Processor Confirm completed a run.','claude-haiku-4.5',1989,3425,0.01529100000000000077);
INSERT INTO agent_runs VALUES('seed-run-processor-confirmation-9','processor-confirmation','2026-10-04T11:52:25.985Z','2026-10-04T11:52:30.595Z',1,'Processor Confirm completed a run.','claude-haiku-4.5',7805,2280,0.01536399999999999919);
INSERT INTO agent_runs VALUES('seed-run-processor-confirmation-10','processor-confirmation','2026-10-08T11:31:55.455Z','2026-10-08T11:32:02.747Z',1,'Processor Confirm completed a run.','claude-haiku-4.5',11327,4128,0.02557399999999999952);
INSERT INTO agent_runs VALUES('seed-run-processor-confirmation-11','processor-confirmation','2026-10-06T17:08:33.933Z','2026-10-06T17:08:36.354Z',1,'Processor Confirm completed a run.','claude-haiku-4.5',2566,3760,0.01709300000000000055);
INSERT INTO agent_runs VALUES('seed-run-flexpay-financing-0','flexpay-financing','2026-10-07T00:42:29.460Z','2026-10-07T00:42:32.603Z',1,'FlexPay Financing completed a run.','claude-haiku-4.5',4707,2499,0.01376199999999999993);
INSERT INTO agent_runs VALUES('seed-run-flexpay-financing-1','flexpay-financing','2026-09-26T12:55:21.805Z','2026-09-26T12:55:26.752Z',1,'FlexPay Financing completed a run.','claude-haiku-4.5',1475,1137,0.005727999999999999585);
INSERT INTO agent_runs VALUES('seed-run-flexpay-financing-2','flexpay-financing','2026-10-02T08:48:25.046Z','2026-10-02T08:48:30.927Z',1,'FlexPay Financing completed a run.','claude-haiku-4.5',11864,3810,0.02473099999999999952);
INSERT INTO agent_runs VALUES('seed-run-flexpay-financing-3','flexpay-financing','2026-10-07T03:29:36.562Z','2026-10-07T03:29:40.717Z',1,'FlexPay Financing completed a run.','claude-haiku-4.5',14326,2297,0.02064900000000000068);
INSERT INTO agent_runs VALUES('seed-run-flexpay-financing-4','flexpay-financing','2026-09-20T21:38:54.053Z','2026-09-20T21:38:54.904Z',1,'FlexPay Financing completed a run.','claude-haiku-4.5',13049,2979,0.02235499999999999988);
INSERT INTO agent_runs VALUES('seed-run-flexpay-financing-5','flexpay-financing','2026-10-03T07:41:45.195Z','2026-10-03T07:41:47.862Z',1,'FlexPay Financing completed a run.','claude-haiku-4.5',13122,365,0.01195799999999999988);
INSERT INTO agent_runs VALUES('seed-run-sales-calls-data-0','sales-calls-data','2026-09-21T08:09:49.908Z','2026-09-21T08:09:50.400Z',1,'Sales Calls Data completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-calls-data-1','sales-calls-data','2026-09-24T03:17:41.597Z','2026-09-24T03:17:45.960Z',1,'Sales Calls Data completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-calls-data-2','sales-calls-data','2026-10-01T19:59:29.564Z','2026-10-01T19:59:30.150Z',0,'Sales Calls Data run failed and was retried.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-calls-data-3','sales-calls-data','2026-10-02T05:50:06.290Z','2026-10-02T05:50:12.129Z',1,'Sales Calls Data completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-calls-data-4','sales-calls-data','2026-10-08T23:27:21.685Z','2026-10-08T23:27:22.315Z',1,'Sales Calls Data completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-calls-data-5','sales-calls-data','2026-09-22T05:52:35.800Z','2026-09-22T05:52:39.669Z',1,'Sales Calls Data completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-calls-data-6','sales-calls-data','2026-10-04T13:10:54.146Z','2026-10-04T13:10:59.798Z',1,'Sales Calls Data completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-calls-data-7','sales-calls-data','2026-10-01T19:27:34.058Z','2026-10-01T19:27:38.514Z',1,'Sales Calls Data completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-calls-data-8','sales-calls-data','2026-10-08T23:03:40.833Z','2026-10-08T23:03:43.362Z',1,'Sales Calls Data completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-calls-data-9','sales-calls-data','2026-09-20T04:05:39.839Z','2026-09-20T04:05:41.506Z',1,'Sales Calls Data completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-calls-data-10','sales-calls-data','2026-09-22T03:13:53.555Z','2026-09-22T03:13:59.203Z',1,'Sales Calls Data completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-calls-data-11','sales-calls-data','2026-09-22T19:04:09.112Z','2026-09-22T19:04:10.647Z',1,'Sales Calls Data completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-sales-calls-data-12','sales-calls-data','2026-09-29T10:45:11.787Z','2026-09-29T10:45:13.186Z',1,'Sales Calls Data completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-data-agent-0','data-agent','2026-09-30T21:42:20.086Z','2026-09-30T21:42:22.732Z',1,'Data Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-data-agent-1','data-agent','2026-09-23T11:05:50.646Z','2026-09-23T11:05:56.127Z',0,'Data Agent run failed and was retried.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-data-agent-2','data-agent','2026-09-25T13:07:05.421Z','2026-09-25T13:07:09.944Z',1,'Data Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-data-agent-3','data-agent','2026-10-07T21:15:17.607Z','2026-10-07T21:15:22.542Z',1,'Data Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-data-agent-4','data-agent','2026-10-04T19:50:49.765Z','2026-10-04T19:50:55.853Z',1,'Data Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-data-agent-5','data-agent','2026-09-21T07:44:42.424Z','2026-09-21T07:44:49.032Z',1,'Data Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-data-agent-6','data-agent','2026-09-24T18:26:46.182Z','2026-09-24T18:26:46.835Z',0,'Data Agent run failed and was retried.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-data-agent-7','data-agent','2026-10-07T05:09:59.779Z','2026-10-07T05:10:01.688Z',0,'Data Agent run failed and was retried.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-data-agent-8','data-agent','2026-09-21T21:27:54.750Z','2026-09-21T21:27:58.778Z',1,'Data Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-data-agent-9','data-agent','2026-09-21T05:59:03.901Z','2026-09-21T05:59:09.007Z',1,'Data Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-data-agent-10','data-agent','2026-09-24T22:33:22.331Z','2026-09-24T22:33:23.783Z',1,'Data Agent completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-markdown-auditor-0','markdown-auditor','2026-10-03T19:42:06.942Z','2026-10-03T19:42:10.605Z',1,'Markdown Auditor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-markdown-auditor-1','markdown-auditor','2026-10-04T15:52:34.660Z','2026-10-04T15:52:38.297Z',1,'Markdown Auditor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-markdown-auditor-2','markdown-auditor','2026-09-27T17:58:44.999Z','2026-09-27T17:58:48.769Z',1,'Markdown Auditor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-markdown-auditor-3','markdown-auditor','2026-09-26T06:41:45.891Z','2026-09-26T06:41:50.337Z',1,'Markdown Auditor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-markdown-auditor-4','markdown-auditor','2026-10-02T13:48:27.401Z','2026-10-02T13:48:30.082Z',1,'Markdown Auditor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-markdown-auditor-5','markdown-auditor','2026-10-07T06:52:30.953Z','2026-10-07T06:52:33.126Z',1,'Markdown Auditor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-markdown-auditor-6','markdown-auditor','2026-10-07T23:11:47.662Z','2026-10-07T23:11:53.936Z',1,'Markdown Auditor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-markdown-auditor-7','markdown-auditor','2026-09-24T20:48:59.991Z','2026-09-24T20:49:02.582Z',0,'Markdown Auditor run failed and was retried.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-markdown-auditor-8','markdown-auditor','2026-10-08T18:22:34.905Z','2026-10-08T18:22:35.934Z',1,'Markdown Auditor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-vector-auditor-0','vector-auditor','2026-09-22T19:34:45.829Z','2026-09-22T19:34:50.509Z',1,'Vector Auditor completed a run.','claude-haiku-4.5',5974,2347,0.01416700000000000077);
INSERT INTO agent_runs VALUES('seed-run-vector-auditor-1','vector-auditor','2026-09-30T20:41:45.272Z','2026-09-30T20:41:49.935Z',1,'Vector Auditor completed a run.','claude-haiku-4.5',8521,369,0.008293000000000000024);
INSERT INTO agent_runs VALUES('seed-run-vector-auditor-2','vector-auditor','2026-09-28T04:07:50.450Z','2026-09-28T04:07:53.127Z',1,'Vector Auditor completed a run.','claude-haiku-4.5',10545,823,0.01172800000000000057);
INSERT INTO agent_runs VALUES('seed-run-vector-auditor-3','vector-auditor','2026-10-07T21:03:53.262Z','2026-10-07T21:03:54.204Z',1,'Vector Auditor completed a run.','claude-haiku-4.5',10916,2974,0.02062900000000000151);
INSERT INTO agent_runs VALUES('seed-run-vector-auditor-4','vector-auditor','2026-10-05T18:29:10.870Z','2026-10-05T18:29:12.021Z',1,'Vector Auditor completed a run.','claude-haiku-4.5',12845,3173,0.02296799999999999884);
INSERT INTO agent_runs VALUES('seed-run-vector-auditor-5','vector-auditor','2026-10-03T00:11:10.424Z','2026-10-03T00:11:12.343Z',1,'Vector Auditor completed a run.','claude-haiku-4.5',12354,1253,0.01489500000000000025);
INSERT INTO agent_runs VALUES('seed-run-vector-auditor-6','vector-auditor','2026-09-24T04:11:17.283Z','2026-09-24T04:11:20.406Z',0,'Vector Auditor run failed and was retried.','claude-haiku-4.5',10591,1202,0.01328099999999999941);
INSERT INTO agent_runs VALUES('seed-run-vector-auditor-7','vector-auditor','2026-09-24T17:20:37.792Z','2026-09-24T17:20:38.371Z',1,'Vector Auditor completed a run.','claude-haiku-4.5',3948,2148,0.01175000000000000002);
INSERT INTO agent_runs VALUES('seed-run-vector-auditor-8','vector-auditor','2026-09-27T22:33:12.017Z','2026-09-27T22:33:18.992Z',1,'Vector Auditor completed a run.','claude-haiku-4.5',13453,2661,0.02140600000000000142);
INSERT INTO agent_runs VALUES('seed-run-payments-pulse-0','payments-pulse','2026-10-05T15:08:11.093Z','2026-10-05T15:08:17.494Z',1,'Payments Pulse completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-payments-pulse-1','payments-pulse','2026-09-24T05:18:34.638Z','2026-09-24T05:18:37.899Z',1,'Payments Pulse completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-payments-pulse-2','payments-pulse','2026-09-26T03:47:31.313Z','2026-09-26T03:47:34.842Z',1,'Payments Pulse completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-payments-pulse-3','payments-pulse','2026-10-04T13:35:47.896Z','2026-10-04T13:35:49.862Z',0,'Payments Pulse run failed and was retried.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-payments-pulse-4','payments-pulse','2026-10-01T15:01:13.799Z','2026-10-01T15:01:20.000Z',1,'Payments Pulse completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-payments-pulse-5','payments-pulse','2026-09-20T00:31:42.602Z','2026-09-20T00:31:44.017Z',1,'Payments Pulse completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-payments-pulse-6','payments-pulse','2026-10-08T06:55:11.910Z','2026-10-08T06:55:12.436Z',1,'Payments Pulse completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-payments-pulse-7','payments-pulse','2026-09-21T20:20:11.402Z','2026-09-21T20:20:18.587Z',1,'Payments Pulse completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-payments-pulse-8','payments-pulse','2026-09-26T03:24:53.387Z','2026-09-26T03:24:59.744Z',1,'Payments Pulse completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-payments-pulse-9','payments-pulse','2026-10-06T15:41:50.718Z','2026-10-06T15:41:54.535Z',1,'Payments Pulse completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-payments-pulse-10','payments-pulse','2026-10-02T08:44:35.236Z','2026-10-02T08:44:41.028Z',1,'Payments Pulse completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-payments-pulse-11','payments-pulse','2026-09-22T09:00:51.848Z','2026-09-22T09:00:56.405Z',1,'Payments Pulse completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-payments-pulse-12','payments-pulse','2026-10-04T05:31:10.492Z','2026-10-04T05:31:17.210Z',1,'Payments Pulse completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-crm-pulse-0','crm-pulse','2026-10-06T04:32:53.020Z','2026-10-06T04:32:59.605Z',1,'Ledger CRM completed a run.','claude-haiku-4.5',3639,3796,0.01809499999999999998);
INSERT INTO agent_runs VALUES('seed-run-crm-pulse-1','crm-pulse','2026-09-20T17:58:32.118Z','2026-09-20T17:58:39.515Z',1,'Ledger CRM completed a run.','claude-haiku-4.5',2645,4097,0.01850399999999999962);
INSERT INTO agent_runs VALUES('seed-run-crm-pulse-2','crm-pulse','2026-09-21T01:47:06.692Z','2026-09-21T01:47:12.918Z',1,'Ledger CRM completed a run.','claude-haiku-4.5',12073,445,0.01143800000000000025);
INSERT INTO agent_runs VALUES('seed-run-crm-pulse-3','crm-pulse','2026-09-26T13:06:16.475Z','2026-09-26T13:06:20.789Z',1,'Ledger CRM completed a run.','claude-haiku-4.5',5766,967,0.008481000000000000691);
INSERT INTO agent_runs VALUES('seed-run-crm-pulse-4','crm-pulse','2026-09-20T01:18:11.827Z','2026-09-20T01:18:16.990Z',1,'Ledger CRM completed a run.','claude-haiku-4.5',6643,2975,0.01721400000000000013);
INSERT INTO agent_runs VALUES('seed-run-crm-pulse-5','crm-pulse','2026-09-25T05:47:27.006Z','2026-09-25T05:47:33.457Z',1,'Ledger CRM completed a run.','claude-haiku-4.5',11927,2776,0.02064600000000000115);
INSERT INTO agent_runs VALUES('seed-run-crm-pulse-6','crm-pulse','2026-10-07T05:38:56.282Z','2026-10-07T05:38:57.220Z',1,'Ledger CRM completed a run.','claude-haiku-4.5',7962,763,0.00942199999999999982);
INSERT INTO agent_runs VALUES('seed-run-crm-pulse-7','crm-pulse','2026-09-23T01:55:08.741Z','2026-09-23T01:55:15.666Z',1,'Ledger CRM completed a run.','claude-haiku-4.5',11660,912,0.01297599999999999969);
INSERT INTO agent_runs VALUES('seed-run-crm-pulse-8','crm-pulse','2026-09-26T09:08:37.995Z','2026-09-26T09:08:42.844Z',1,'Ledger CRM completed a run.','claude-haiku-4.5',14016,3888,0.02676500000000000059);
INSERT INTO agent_runs VALUES('seed-run-stack-monitor-0','stack-monitor','2026-09-21T20:36:49.473Z','2026-09-21T20:36:52.787Z',1,'Stack Monitor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stack-monitor-1','stack-monitor','2026-09-22T22:03:37.510Z','2026-09-22T22:03:38.017Z',1,'Stack Monitor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stack-monitor-2','stack-monitor','2026-09-27T22:32:30.226Z','2026-09-27T22:32:30.966Z',1,'Stack Monitor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stack-monitor-3','stack-monitor','2026-09-30T03:20:35.287Z','2026-09-30T03:20:36.186Z',1,'Stack Monitor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-stack-monitor-4','stack-monitor','2026-09-23T18:29:37.550Z','2026-09-23T18:29:44.793Z',1,'Stack Monitor completed a run.',NULL,NULL,NULL,NULL);
INSERT INTO agent_runs VALUES('seed-run-client-roster-0','client-roster','2026-10-05T04:44:52.601Z','2026-10-05T04:44:56.705Z',1,'Client Roster completed a run.','claude-sonnet-5',7260,4135,0.08380500000000000448);
INSERT INTO agent_runs VALUES('seed-run-client-roster-1','client-roster','2026-10-07T22:28:49.303Z','2026-10-07T22:28:50.508Z',1,'Client Roster completed a run.','claude-sonnet-5',3786,1878,0.03952800000000000063);
INSERT INTO agent_runs VALUES('seed-run-client-roster-2','client-roster','2026-09-29T12:59:18.767Z','2026-09-29T12:59:21.532Z',1,'Client Roster completed a run.','claude-sonnet-5',4791,4159,0.07675800000000000678);
INSERT INTO agent_runs VALUES('seed-run-client-roster-3','client-roster','2026-10-02T21:11:13.899Z','2026-10-02T21:11:19.310Z',1,'Client Roster completed a run.','claude-sonnet-5',11268,3123,0.08064899999999999847);
INSERT INTO agent_runs VALUES('seed-run-client-roster-4','client-roster','2026-09-27T12:22:33.632Z','2026-09-27T12:22:36.246Z',0,'Client Roster run failed and was retried.','claude-sonnet-5',3115,2005,0.0394199999999999967);
INSERT INTO agent_runs VALUES('seed-run-client-roster-5','client-roster','2026-10-03T19:08:03.526Z','2026-10-03T19:08:09.149Z',1,'Client Roster completed a run.','claude-sonnet-5',13503,1692,0.06588900000000000312);
INSERT INTO agent_runs VALUES('seed-run-client-roster-6','client-roster','2026-09-19T20:10:49.048Z','2026-09-19T20:10:52.814Z',1,'Client Roster completed a run.','claude-sonnet-5',4119,1723,0.03820199999999999985);
INSERT INTO agent_runs VALUES('seed-run-client-roster-7','client-roster','2026-10-03T11:39:44.838Z','2026-10-03T11:39:45.570Z',1,'Client Roster completed a run.','claude-sonnet-5',2504,3557,0.0608669999999999975);
INSERT INTO agent_runs VALUES('seed-run-client-roster-8','client-roster','2026-09-29T14:35:41.277Z','2026-09-29T14:35:42.340Z',1,'Client Roster completed a run.','claude-sonnet-5',5296,2309,0.05052299999999999847);
INSERT INTO agent_runs VALUES('seed-run-client-roster-9','client-roster','2026-10-04T15:40:45.744Z','2026-10-04T15:40:46.861Z',1,'Client Roster completed a run.','claude-sonnet-5',11462,3998,0.0943559999999999955);
INSERT INTO agent_runs VALUES('seed-run-client-roster-10','client-roster','2026-09-25T21:26:29.304Z','2026-09-25T21:26:34.101Z',1,'Client Roster completed a run.','claude-sonnet-5',9982,1647,0.0546509999999999982);
INSERT INTO agent_runs VALUES('seed-run-client-roster-11','client-roster','2026-09-24T02:54:16.712Z','2026-09-24T02:54:17.220Z',1,'Client Roster completed a run.','claude-sonnet-5',7590,2484,0.06003000000000000002);
INSERT INTO agent_runs VALUES('seed-run-client-roster-12','client-roster','2026-09-20T05:19:55.177Z','2026-09-20T05:19:59.713Z',1,'Client Roster completed a run.','claude-sonnet-5',10026,3811,0.0872430000000000011);
INSERT INTO agent_runs VALUES('seed-run-client-onboarding-0','client-onboarding','2026-09-24T04:58:50.542Z','2026-09-24T04:58:57.254Z',0,'Onboarding Agent run failed and was retried.','claude-haiku-4.5',13192,3310,0.02379399999999999918);
INSERT INTO agent_runs VALUES('seed-run-client-onboarding-1','client-onboarding','2026-10-03T22:11:06.875Z','2026-10-03T22:11:11.927Z',1,'Onboarding Agent completed a run.','claude-haiku-4.5',7127,3799,0.02089799999999999991);
INSERT INTO agent_runs VALUES('seed-run-client-onboarding-2','client-onboarding','2026-10-08T21:46:01.081Z','2026-10-08T21:46:02.536Z',1,'Onboarding Agent completed a run.','claude-haiku-4.5',3502,1713,0.00965399999999999939);
INSERT INTO agent_runs VALUES('seed-run-client-onboarding-3','client-onboarding','2026-09-26T10:27:13.312Z','2026-09-26T10:27:16.274Z',1,'Onboarding Agent completed a run.','claude-haiku-4.5',10398,2091,0.01668199999999999892);
INSERT INTO agent_runs VALUES('seed-run-client-onboarding-4','client-onboarding','2026-09-24T20:50:13.281Z','2026-09-24T20:50:14.163Z',1,'Onboarding Agent completed a run.','claude-haiku-4.5',8896,908,0.01074899999999999988);
INSERT INTO agent_runs VALUES('seed-run-client-onboarding-5','client-onboarding','2026-10-07T14:52:30.001Z','2026-10-07T14:52:32.831Z',1,'Onboarding Agent completed a run.','claude-haiku-4.5',14145,1856,0.01873999999999999972);
INSERT INTO agent_runs VALUES('seed-run-client-onboarding-6','client-onboarding','2026-10-05T05:30:09.158Z','2026-10-05T05:30:14.930Z',1,'Onboarding Agent completed a run.','claude-haiku-4.5',13848,330,0.0123979999999999993);
INSERT INTO agent_runs VALUES('seed-run-client-onboarding-7','client-onboarding','2026-09-23T05:35:04.611Z','2026-09-23T05:35:06.649Z',1,'Onboarding Agent completed a run.','claude-haiku-4.5',4224,3939,0.01913499999999999924);
INSERT INTO agent_runs VALUES('seed-run-client-onboarding-8','client-onboarding','2026-09-24T22:45:00.476Z','2026-09-24T22:45:03.736Z',1,'Onboarding Agent completed a run.','claude-haiku-4.5',8687,1445,0.01273);
INSERT INTO agent_runs VALUES('seed-run-client-success-0','client-success','2026-10-05T11:16:59.247Z','2026-10-05T11:17:04.660Z',1,'Client Success completed a run.','claude-haiku-4.5',3539,2827,0.01413900000000000052);
INSERT INTO agent_runs VALUES('seed-run-client-success-1','client-success','2026-09-30T09:41:57.986Z','2026-09-30T09:41:59.258Z',1,'Client Success completed a run.','claude-haiku-4.5',2352,2584,0.0122179999999999997);
INSERT INTO agent_runs VALUES('seed-run-client-success-2','client-success','2026-09-28T23:12:47.828Z','2026-09-28T23:12:49.819Z',1,'Client Success completed a run.','claude-haiku-4.5',10942,3251,0.02175799999999999957);
INSERT INTO agent_runs VALUES('seed-run-client-success-3','client-success','2026-09-23T14:57:02.632Z','2026-09-23T14:57:05.216Z',0,'Client Success run failed and was retried.','claude-haiku-4.5',1634,3912,0.01695500000000000133);
INSERT INTO agent_runs VALUES('seed-run-client-success-4','client-success','2026-09-19T23:31:39.629Z','2026-09-19T23:31:43.784Z',1,'Client Success completed a run.','claude-haiku-4.5',8659,1408,0.01255900000000000071);
CREATE TABLE agent_messages (
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL,
  role TEXT NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  tool_calls TEXT NOT NULL DEFAULT '[]',
  created_at TEXT NOT NULL
);
CREATE TABLE trading_snapshots (
  account_id TEXT NOT NULL DEFAULT 'individual',
  account_label TEXT NOT NULL DEFAULT 'Individual',
  captured_at TEXT NOT NULL,
  account_value_usd REAL NOT NULL,
  buying_power_usd REAL NOT NULL,
  cash_usd REAL NOT NULL,
  day_pnl_usd REAL NOT NULL,
  total_pnl_usd REAL NOT NULL,
  source TEXT NOT NULL,
  PRIMARY KEY (account_id, captured_at)
);
INSERT INTO trading_snapshots VALUES('agentic','Agentic','2026-08-13T11:00:00.000Z',1000.0,1000.0,1000.0,0.0,0.0,'seed');
INSERT INTO trading_snapshots VALUES('agentic','Agentic','2026-08-13T12:00:00.000Z',1004.700000000000045,604.7000000000000454,604.7000000000000454,4.700000000000000177,4.700000000000000177,'seed');
INSERT INTO trading_snapshots VALUES('agentic','Agentic','2026-08-13T13:00:00.000Z',1021.899999999999978,418.5,418.5,21.89999999999999857,21.89999999999999857,'seed');
INSERT INTO trading_snapshots VALUES('agentic','Agentic','2026-08-13T14:00:00.000Z',1016.399999999999978,418.5,418.5,16.39999999999999857,16.39999999999999857,'seed');
INSERT INTO trading_snapshots VALUES('individual','Individual','2026-08-13T15:00:00.000Z',10250.0,2960.0,2960.0,130.0,250.0,'seed');
INSERT INTO trading_snapshots VALUES('agentic','Agentic','2026-08-13T15:00:00.000Z',1040.0,420.0,420.0,12.0,40.0,'seed');
CREATE TABLE trading_positions (
  account_id TEXT NOT NULL DEFAULT 'individual',
  captured_at TEXT NOT NULL,
  symbol TEXT NOT NULL,
  quantity REAL NOT NULL,
  avg_cost_usd REAL NOT NULL,
  market_value_usd REAL NOT NULL,
  unrealized_pnl_usd REAL NOT NULL,
  PRIMARY KEY (account_id, captured_at, symbol)
);
INSERT INTO trading_positions VALUES('individual','2026-08-13T15:00:00.000Z','NVDA',4.0,902.1000000000000227,3812.59999999999991,204.1999999999999887);
INSERT INTO trading_positions VALUES('individual','2026-08-13T15:00:00.000Z','AAPL',8.0,214.3499999999999944,1760.40000000000009,45.60000000000000142);
INSERT INTO trading_positions VALUES('individual','2026-08-13T15:00:00.000Z','MSFT',3.0,428.0,1301.700000000000045,17.69999999999999929);
INSERT INTO trading_positions VALUES('individual','2026-08-13T15:00:00.000Z','VOO',5.0,82.90000000000000569,413.7900000000000204,-1.709999999999999965);
INSERT INTO trading_positions VALUES('agentic','2026-08-13T15:00:00.000Z','SPY',1.0,601.3999999999999773,624.7000000000000454,23.30000000000000071);
CREATE TABLE proposals (
  id TEXT PRIMARY KEY,
  client TEXT NOT NULL,
  brand TEXT NOT NULL,
  url TEXT NOT NULL,
  status TEXT NOT NULL,
  amount_usd REAL,
  notes TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  origin TEXT NOT NULL DEFAULT 'seed',
  access_code TEXT NOT NULL DEFAULT ''
);
CREATE TABLE deliverable_decisions (
  id TEXT PRIMARY KEY,
  decision TEXT NOT NULL,
  decided_at TEXT NOT NULL,
  decided_revision TEXT NOT NULL DEFAULT '',
  note TEXT NOT NULL DEFAULT ''
);
CREATE TABLE brand_deals (
  id TEXT PRIMARY KEY,
  brand TEXT NOT NULL,
  status TEXT NOT NULL,
  tier TEXT,
  deal_value_usd REAL,
  budget_usd REAL,
  amount_agreed_usd REAL,
  suggested_rate_usd REAL,
  paid_in_full INTEGER NOT NULL DEFAULT 0,
  deadline TEXT,
  follow_up_date TEXT,
  contact_name TEXT,
  contact_email TEXT,
  main_channel TEXT,
  video_type TEXT,
  source TEXT,
  icp_fit TEXT,
  notion_url TEXT NOT NULL,
  last_edited TEXT NOT NULL,
  seeded INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE trading_orders (
  id TEXT PRIMARY KEY,
  account_id TEXT NOT NULL DEFAULT 'agentic',
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
CREATE TABLE trading_analysis (
  id TEXT PRIMARY KEY,
  at TEXT NOT NULL,
  account_id TEXT NOT NULL DEFAULT 'agentic',
  agent TEXT NOT NULL,
  examined INTEGER NOT NULL DEFAULT 0,
  signals INTEGER NOT NULL DEFAULT 0,
  notes TEXT NOT NULL DEFAULT '',
  rows TEXT NOT NULL DEFAULT '[]'
);
CREATE TABLE trading_activity (
  id TEXT PRIMARY KEY,
  at TEXT NOT NULL,
  account_id TEXT NOT NULL DEFAULT 'individual',
  agent TEXT NOT NULL,
  action TEXT NOT NULL,
  symbol TEXT NOT NULL,
  quantity REAL NOT NULL,
  price_usd REAL NOT NULL,
  rationale TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL
);
INSERT INTO trading_activity VALUES('tr-seed-5','2026-08-13T14:58:00.000Z','agentic','Markets Agent','buy','SPY',1.0,601.3999999999999773,'Parking idle buying power in the index sleeve.','filled');
INSERT INTO trading_activity VALUES('tr-seed-4','2026-08-13T13:20:00.000Z','agentic','Markets Agent','sell','TSLA',2.0,240.8000000000000113,'Trimming into strength; thesis played out, rotating to cash.','filled');
INSERT INTO trading_activity VALUES('tr-seed-3','2026-08-12T18:05:00.000Z','individual','Operator (manual)','buy','AAPL',4.0,213.9000000000000056,'Dollar-cost tranche into the core holding.','filled');
INSERT INTO trading_activity VALUES('tr-seed-2','2026-08-12T15:41:00.000Z','agentic','Markets Agent','buy','VOO',5.0,83.01999999999999603,'Parking idle buying power in the index sleeve.','filled');
INSERT INTO trading_activity VALUES('tr-seed-1','2026-08-11T16:12:00.000Z','individual','Operator (manual)','buy','MSFT',3.0,428.0,'Initiating position per the approved watchlist.','filled');
CREATE TABLE trading_limits (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  max_notional_per_trade_usd REAL NOT NULL,
  max_position_pct_of_sleeve REAL NOT NULL,
  max_risk_pct_per_trade REAL NOT NULL,
  max_concurrent_positions INTEGER NOT NULL,
  max_trades_per_day INTEGER NOT NULL,
  min_sleeve_value_usd REAL NOT NULL,
  max_deployed_capital_usd REAL NOT NULL,
  autopilot INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);
CREATE TABLE usage_snapshots (
  id TEXT PRIMARY KEY,
  captured_at TEXT NOT NULL,
  payload TEXT NOT NULL
);
CREATE TABLE broadcasts (
  id TEXT PRIMARY KEY,
  message TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE agent_tasks (
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL,
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
INSERT INTO agent_tasks VALUES('task-seed-1','comms-agent','Triage overnight inbound across 4 inboxes','open','2026-07-21T12:00:00.000Z','2026-07-21T12:00:00.000Z');
INSERT INTO agent_tasks VALUES('task-seed-2','social-agent','Draft 3 IG hooks for the Vantage launch','open','2026-07-21T12:00:00.000Z','2026-07-21T12:00:00.000Z');
INSERT INTO agent_tasks VALUES('task-seed-3','gmail-worker','Follow up on 6 unreplied warm leads','open','2026-07-21T12:00:00.000Z','2026-07-21T12:00:00.000Z');
INSERT INTO agent_tasks VALUES('task-seed-4','adsmith-creative','Generate 5 UGC variants for the new offer','open','2026-07-21T12:00:00.000Z','2026-07-21T12:00:00.000Z');
INSERT INTO agent_tasks VALUES('task-seed-5','postly-publisher','Schedule this week''s cross-platform posts','doing','2026-07-21T12:00:00.000Z','2026-07-21T12:00:00.000Z');
INSERT INTO agent_tasks VALUES('task-seed-6','comms-agent','Qualify 12 new DMs from the campaign','doing','2026-07-21T12:00:00.000Z','2026-07-21T12:00:00.000Z');
INSERT INTO agent_tasks VALUES('task-seed-7','reelkit-editor','Cut the sales-call highlight reel','doing','2026-07-21T12:00:00.000Z','2026-07-21T12:00:00.000Z');
INSERT INTO agent_tasks VALUES('task-seed-8','gmail-worker','Send the Vantage proposal follow-up','done','2026-07-21T12:00:00.000Z','2026-07-21T12:00:00.000Z');
INSERT INTO agent_tasks VALUES('task-seed-9','slack-worker','Post the Monday standup digest','done','2026-07-21T12:00:00.000Z','2026-07-21T12:00:00.000Z');
INSERT INTO agent_tasks VALUES('task-seed-10','social-agent','Publish the Tuesday carousel','done','2026-07-21T12:00:00.000Z','2026-07-21T12:00:00.000Z');
INSERT INTO agent_tasks VALUES('task-seed-11','postly-publisher','Sync follower counts across 6 platforms','done','2026-07-21T12:00:00.000Z','2026-07-21T12:00:00.000Z');
CREATE TABLE agent_crons (
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL,
  schedule TEXT NOT NULL,
  description TEXT NOT NULL,
  enabled INTEGER NOT NULL,
  created_at TEXT NOT NULL
);
INSERT INTO agent_crons VALUES('cron-comms-digest-0900','comms-digest','0 9 * * *','Morning comms report: 24h of email, WhatsApp and Slack, ranked by who needs a reply',1,'2026-08-18T00:00:00.000Z');
INSERT INTO agent_crons VALUES('cron-plaud-ingest-30m','sales-calls-data','*/30 * * * *','File every newly transcribed Plaud recording into Brain (summary + transcript) and its action items into the claim store; no LLM, pure code',1,'2026-08-26T00:00:00.000Z');
INSERT INTO agent_crons VALUES('cron-stack-monitor-0700','stack-monitor','0 7 * * *','Local stack check: the command center, the worker pool, Brain and the CLIs the OS shells out to',1,'2026-08-18T00:00:00.000Z');
INSERT INTO agent_crons VALUES('cron-payments-pulse-0800','payments-pulse','0 8 * * *','Payment processors reachable, plus Stripe balance and recent charges',1,'2026-08-18T00:00:00.000Z');
INSERT INTO agent_crons VALUES('cron-client-onboarding-0830','client-onboarding','30 8 * * *','Onboarding SOP readiness: the Ledger trigger and the Slack workspace it provisions',1,'2026-08-18T00:00:00.000Z');
INSERT INTO agent_crons VALUES('cron-crm-pulse-0900','crm-pulse','0 9 * * 1-5','Ledger deals pipeline across Vantage and Launchpad Cohort',1,'2026-08-18T00:00:00.000Z');
INSERT INTO agent_crons VALUES('cron-social-agent-1800','social-agent','0 18 * * *','Postly publishing and Adsmith ad generation, checked before the evening',1,'2026-08-18T00:00:00.000Z');
CREATE TABLE cron_runs (
  id TEXT PRIMARY KEY,
  cron_id TEXT NOT NULL,
  agent_id TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT,
  ok INTEGER NOT NULL,
  summary TEXT NOT NULL
);
CREATE TABLE digest_reads (
  key TEXT PRIMARY KEY,
  read_at TEXT NOT NULL
);
CREATE TABLE comms_digests (
  id TEXT PRIMARY KEY,
  generated_at TEXT NOT NULL,
  payload TEXT NOT NULL
);
CREATE TABLE plaud_ingests (
  file_id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  recorded_at TEXT NOT NULL,
  ingested_at TEXT NOT NULL,
  via TEXT NOT NULL,
  slug TEXT NOT NULL,
  claims INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE contact_tags (
  person TEXT NOT NULL,
  channel TEXT NOT NULL,
  tag TEXT NOT NULL,
  tier INTEGER NOT NULL,
  PRIMARY KEY (person, channel)
);
CREATE TABLE social_accounts (
  platform TEXT PRIMARY KEY,
  handle TEXT NOT NULL,
  url TEXT,
  "order" INTEGER NOT NULL
);
INSERT INTO social_accounts VALUES('instagram','@founderos.ai','https://instagram.com/founderos.ai',1);
INSERT INTO social_accounts VALUES('tiktok','@founderos.ai','https://tiktok.com/@founderos.ai',2);
INSERT INTO social_accounts VALUES('twitter','@Founderosai','https://x.com/Founderosai',3);
INSERT INTO social_accounts VALUES('youtube','@founderosai','https://youtube.com/@founderosai',4);
INSERT INTO social_accounts VALUES('linkedin','Alex',NULL,5);
CREATE TABLE social_snapshots (
  platform TEXT NOT NULL,
  captured_at TEXT NOT NULL,
  followers INTEGER NOT NULL,
  source TEXT NOT NULL,
  PRIMARY KEY (platform, captured_at)
);
INSERT INTO social_snapshots VALUES('instagram','2026-03-14',30125,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-15',30224,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-16',30279,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-17',30306,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-18',30331,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-19',30381,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-20',30473,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-21',30604,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-22',30753,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-23',30894,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-24',31006,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-25',31082,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-26',31136,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-27',31194,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-28',31281,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-29',31411,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-30',31578,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-03-31',31761,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-01',31930,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-02',32062,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-03',32152,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-04',32213,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-05',32271,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-06',32351,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-07',32468,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-08',32618,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-09',32779,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-10',32927,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-11',33039,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-12',33113,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-13',33163,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-14',33218,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-15',33303,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-16',33432,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-17',33602,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-18',33790,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-19',33969,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-20',34115,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-21',34225,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-22',34312,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-23',34400,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-24',34516,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-25',34671,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-26',34858,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-27',35055,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-28',35234,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-29',35374,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-04-30',35470,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-01',35540,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-02',35609,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-03',35704,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-04',35840,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-05',36011,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-06',36196,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-07',36369,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-08',36510,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-09',36615,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-10',36702,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-11',36796,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-12',36922,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-13',37093,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-14',37302,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-15',37524,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-16',37731,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-17',37902,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-18',38032,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-19',38137,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-20',38242,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-21',38373,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-22',38541,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-23',38740,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-24',38946,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-25',39132,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-26',39281,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-27',39390,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-28',39477,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-29',39570,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-30',39694,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-05-31',39863,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-06-01',40070,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-06-02',40291,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-06-03',40500,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-06-04',40677,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-06-05',40818,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-06-06',40941,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-06-07',41071,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-06-08',41233,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-06-09',41435,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-06-10',41669,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-06-11',41911,'seed-dummy');
INSERT INTO social_snapshots VALUES('instagram','2026-06-12',42000,'postly-config');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-14',6017,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-15',6039,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-16',6055,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-17',6078,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-18',6120,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-19',6185,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-20',6267,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-21',6352,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-22',6426,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-23',6481,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-24',6518,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-25',6547,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-26',6580,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-27',6628,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-28',6696,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-29',6775,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-30',6855,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-03-31',6921,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-01',6966,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-02',6992,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-03',7010,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-04',7035,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-05',7077,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-06',7141,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-07',7221,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-08',7304,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-09',7377,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-10',7433,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-11',7473,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-12',7508,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-13',7550,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-14',7612,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-15',7694,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-16',7790,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-17',7887,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-18',7970,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-19',8032,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-20',8075,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-21',8109,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-22',8148,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-23',8202,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-24',8276,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-25',8362,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-26',8448,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-27',8521,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-28',8575,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-29',8613,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-04-30',8646,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-01',8687,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-02',8749,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-03',8833,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-04',8933,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-05',9035,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-06',9125,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-07',9198,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-08',9253,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-09',9303,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-10',9359,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-11',9433,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-12',9525,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-13',9629,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-14',9731,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-15',9818,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-16',9883,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-17',9929,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-18',9969,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-19',10015,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-20',10079,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-21',10164,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-22',10262,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-23',10362,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-24',10450,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-25',10520,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-26',10575,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-27',10627,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-28',10689,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-29',10772,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-30',10875,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-05-31',10992,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-06-01',11109,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-06-02',11211,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-06-03',11293,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-06-04',11358,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-06-05',11415,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-06-06',11478,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-06-07',11558,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-06-08',11656,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-06-09',11765,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-06-10',11872,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-06-11',11963,'seed-dummy');
INSERT INTO social_snapshots VALUES('tiktok','2026-06-12',12000,'postly-config');
INSERT INTO social_snapshots VALUES('twitter','2026-03-14',2999,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-15',3009,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-16',3022,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-17',3043,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-18',3070,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-19',3100,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-20',3128,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-21',3150,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-22',3164,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-23',3171,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-24',3178,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-25',3188,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-26',3204,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-27',3228,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-28',3256,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-29',3282,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-30',3302,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-03-31',3316,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-01',3326,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-02',3335,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-03',3350,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-04',3373,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-05',3403,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-06',3437,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-07',3470,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-08',3496,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-09',3516,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-10',3530,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-11',3543,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-12',3560,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-13',3583,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-14',3612,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-15',3644,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-16',3674,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-17',3697,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-18',3712,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-19',3723,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-20',3732,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-21',3747,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-22',3769,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-23',3799,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-24',3833,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-25',3865,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-26',3892,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-27',3913,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-28',3929,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-29',3946,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-04-30',3967,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-01',3996,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-02',4032,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-03',4070,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-04',4106,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-05',4136,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-06',4157,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-07',4173,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-08',4187,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-09',4206,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-10',4232,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-11',4264,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-12',4299,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-13',4331,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-14',4358,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-15',4377,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-16',4393,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-17',4409,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-18',4430,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-19',4460,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-20',4497,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-21',4538,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-22',4577,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-23',4610,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-24',4636,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-25',4658,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-26',4679,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-27',4705,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-28',4737,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-29',4776,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-30',4817,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-05-31',4854,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-06-01',4884,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-06-02',4907,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-06-03',4925,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-06-04',4942,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-06-05',4964,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-06-06',4995,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-06-07',5031,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-06-08',5071,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-06-09',5109,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-06-10',5142,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-06-11',5168,'seed-dummy');
INSERT INTO social_snapshots VALUES('twitter','2026-06-12',5200,'postly-config');
INSERT INTO social_snapshots VALUES('youtube','2026-03-14',300,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-15',303,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-16',308,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-17',315,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-18',322,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-19',328,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-20',332,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-21',335,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-22',336,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-23',338,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-24',342,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-25',348,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-26',356,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-27',364,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-28',372,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-29',378,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-30',382,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-03-31',385,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-01',389,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-02',394,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-03',401,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-04',410,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-05',419,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-06',426,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-07',432,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-08',435,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-09',437,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-10',440,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-11',444,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-12',450,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-13',458,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-14',467,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-15',474,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-16',479,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-17',483,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-18',487,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-19',491,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-20',497,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-21',505,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-22',514,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-23',524,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-24',533,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-25',541,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-26',546,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-27',550,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-28',554,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-29',560,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-04-30',568,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-01',577,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-02',586,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-03',594,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-04',600,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-05',604,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-06',608,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-07',612,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-08',617,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-09',625,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-10',635,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-11',645,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-12',654,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-13',662,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-14',667,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-15',672,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-16',678,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-17',686,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-18',695,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-19',706,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-20',717,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-21',727,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-22',735,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-23',740,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-24',745,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-25',750,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-26',756,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-27',765,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-28',775,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-29',785,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-30',794,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-05-31',801,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-06-01',806,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-06-02',811,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-06-03',817,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-06-04',825,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-06-05',835,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-06-06',847,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-06-07',859,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-06-08',869,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-06-09',878,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-06-10',885,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-06-11',892,'seed-dummy');
INSERT INTO social_snapshots VALUES('youtube','2026-06-12',900,'postly-config');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-14',793,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-15',800,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-16',809,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-17',817,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-18',824,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-19',829,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-20',832,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-21',835,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-22',839,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-23',846,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-24',855,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-25',865,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-26',875,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-27',883,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-28',888,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-29',892,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-30',895,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-03-31',898,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-01',904,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-02',912,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-03',921,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-04',930,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-05',937,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-06',941,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-07',944,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-08',947,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-09',951,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-10',958,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-11',967,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-12',978,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-13',988,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-14',997,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-15',1004,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-16',1009,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-17',1014,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-18',1019,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-19',1027,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-20',1038,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-21',1049,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-22',1059,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-23',1067,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-24',1073,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-25',1077,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-26',1081,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-27',1085,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-28',1092,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-29',1101,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-04-30',1112,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-01',1122,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-02',1131,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-03',1137,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-04',1143,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-05',1148,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-06',1154,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-07',1163,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-08',1175,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-09',1187,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-10',1199,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-11',1210,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-12',1218,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-13',1224,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-14',1229,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-15',1236,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-16',1244,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-17',1255,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-18',1267,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-19',1278,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-20',1287,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-21',1293,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-22',1298,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-23',1303,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-24',1310,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-25',1319,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-26',1330,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-27',1343,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-28',1355,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-29',1366,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-30',1375,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-05-31',1382,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-06-01',1389,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-06-02',1397,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-06-03',1408,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-06-04',1421,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-06-05',1435,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-06-06',1448,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-06-07',1458,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-06-08',1467,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-06-09',1473,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-06-10',1479,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-06-11',1487,'seed-dummy');
INSERT INTO social_snapshots VALUES('linkedin','2026-06-12',1500,'seed-dummy');
CREATE TABLE broadcast_replies (
  id TEXT PRIMARY KEY,
  broadcast_id TEXT NOT NULL REFERENCES broadcasts(id),
  agent_id TEXT NOT NULL,
  ok INTEGER NOT NULL,
  reply TEXT NOT NULL DEFAULT '',
  finished_at TEXT NOT NULL
);
CREATE TABLE email_list_snapshots (
  captured_at TEXT PRIMARY KEY,
  subscribers INTEGER NOT NULL,
  source TEXT NOT NULL
);
INSERT INTO email_list_snapshots VALUES('2026-05-28',1849,'seed-beehiiv');
INSERT INTO email_list_snapshots VALUES('2026-05-29',1849,'seed-beehiiv');
INSERT INTO email_list_snapshots VALUES('2026-05-30',1849,'seed-beehiiv');
INSERT INTO email_list_snapshots VALUES('2026-05-31',1849,'seed-beehiiv');
INSERT INTO email_list_snapshots VALUES('2026-06-01',1849,'seed-beehiiv');
INSERT INTO email_list_snapshots VALUES('2026-06-02',1849,'seed-beehiiv');
INSERT INTO email_list_snapshots VALUES('2026-06-03',1849,'seed-beehiiv');
INSERT INTO email_list_snapshots VALUES('2026-06-04',1849,'seed-beehiiv');
INSERT INTO email_list_snapshots VALUES('2026-06-05',1849,'seed-beehiiv');
INSERT INTO email_list_snapshots VALUES('2026-06-06',1849,'seed-beehiiv');
INSERT INTO email_list_snapshots VALUES('2026-06-07',1849,'seed-beehiiv');
INSERT INTO email_list_snapshots VALUES('2026-06-08',1849,'seed-beehiiv');
INSERT INTO email_list_snapshots VALUES('2026-06-09',1849,'seed-beehiiv');
INSERT INTO email_list_snapshots VALUES('2026-06-10',1849,'seed-beehiiv');
INSERT INTO email_list_snapshots VALUES('2026-06-11',1849,'seed-beehiiv');
INSERT INTO email_list_snapshots VALUES('2026-06-12',1850,'seed-beehiiv');
CREATE TABLE metric_snapshots (
  metric_id TEXT NOT NULL,
  captured_at TEXT NOT NULL,
  value REAL NOT NULL,
  PRIMARY KEY (metric_id, captured_at)
);
CREATE TABLE social_dms (
  platform TEXT PRIMARY KEY,
  count INTEGER NOT NULL,
  updated_at TEXT NOT NULL
);
INSERT INTO social_dms VALUES('instagram',1200,'2026-06-12');
INSERT INTO social_dms VALUES('tiktok',400,'2026-06-12');
INSERT INTO social_dms VALUES('twitter',200,'2026-06-12');
INSERT INTO social_dms VALUES('youtube',60,'2026-06-12');
INSERT INTO social_dms VALUES('linkedin',90,'2026-06-12');
CREATE TABLE social_dm_snapshots (
  platform TEXT NOT NULL,
  captured_at TEXT NOT NULL,
  count INTEGER NOT NULL,
  source TEXT NOT NULL,
  PRIMARY KEY (platform, captured_at)
);
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-14',798,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-15',804,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-16',809,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-17',813,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-18',815,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-19',817,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-20',819,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-21',822,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-22',825,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-23',830,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-24',835,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-25',840,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-26',843,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-27',845,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-28',846,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-29',847,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-30',850,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-03-31',853,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-01',858,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-02',864,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-03',869,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-04',873,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-05',876,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-06',878,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-07',880,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-08',884,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-09',889,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-10',895,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-11',901,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-12',907,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-13',911,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-14',914,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-15',916,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-16',918,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-17',921,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-18',926,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-19',931,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-20',937,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-21',942,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-22',946,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-23',948,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-24',950,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-25',953,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-26',956,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-27',962,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-28',968,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-29',975,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-04-30',981,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-01',986,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-02',990,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-03',993,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-04',996,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-05',1001,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-06',1006,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-07',1013,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-08',1019,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-09',1025,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-10',1030,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-11',1033,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-12',1035,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-13',1038,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-14',1042,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-15',1047,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-16',1053,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-17',1060,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-18',1066,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-19',1071,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-20',1074,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-21',1078,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-22',1082,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-23',1087,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-24',1093,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-25',1101,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-26',1108,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-27',1115,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-28',1121,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-29',1125,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-30',1129,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-05-31',1132,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-06-01',1137,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-06-02',1143,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-06-03',1150,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-06-04',1157,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-06-05',1163,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-06-06',1168,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-06-07',1172,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-06-08',1175,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-06-09',1179,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-06-10',1184,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-06-11',1190,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('instagram','2026-06-12',1200,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-14',202,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-15',204,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-16',205,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-17',206,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-18',206,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-19',207,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-20',208,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-21',210,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-22',213,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-23',215,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-24',217,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-25',218,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-26',219,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-27',220,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-28',221,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-29',223,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-30',225,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-03-31',228,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-01',231,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-02',234,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-03',236,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-04',237,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-05',238,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-06',239,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-07',241,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-08',244,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-09',246,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-10',249,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-11',251,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-12',253,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-13',254,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-14',254,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-15',255,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-16',257,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-17',260,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-18',263,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-19',266,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-20',268,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-21',270,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-22',272,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-23',273,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-24',275,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-25',277,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-26',280,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-27',284,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-28',287,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-29',290,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-04-30',292,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-01',293,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-02',294,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-03',296,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-04',298,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-05',300,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-06',303,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-07',306,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-08',309,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-09',311,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-10',312,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-11',314,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-12',315,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-13',318,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-14',321,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-15',325,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-16',328,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-17',331,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-18',334,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-19',336,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-20',337,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-21',339,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-22',342,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-23',345,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-24',349,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-25',352,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-26',355,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-27',357,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-28',359,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-29',360,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-30',362,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-05-31',365,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-06-01',368,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-06-02',371,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-06-03',375,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-06-04',378,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-06-05',380,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-06-06',382,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-06-07',384,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-06-08',387,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-06-09',390,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-06-10',394,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-06-11',398,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('tiktok','2026-06-12',400,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-14',120,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-15',121,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-16',121,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-17',121,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-18',122,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-19',122,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-20',123,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-21',124,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-22',125,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-23',126,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-24',127,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-25',127,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-26',128,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-27',128,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-28',129,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-29',130,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-30',131,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-03-31',132,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-01',133,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-02',134,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-03',134,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-04',134,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-05',135,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-06',135,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-07',136,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-08',137,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-09',138,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-10',139,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-11',140,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-12',140,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-13',141,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-14',141,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-15',142,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-16',144,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-17',145,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-18',146,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-19',147,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-20',148,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-21',148,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-22',149,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-23',149,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-24',150,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-25',152,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-26',153,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-27',154,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-28',155,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-29',155,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-04-30',156,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-01',156,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-02',157,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-03',158,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-04',159,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-05',160,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-06',161,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-07',163,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-08',163,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-09',164,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-10',165,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-11',166,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-12',167,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-13',168,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-14',170,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-15',171,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-16',172,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-17',173,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-18',173,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-19',174,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-20',175,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-21',176,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-22',177,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-23',178,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-24',179,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-25',180,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-26',181,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-27',182,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-28',183,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-29',183,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-30',185,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-05-31',186,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-06-01',188,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-06-02',189,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-06-03',190,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-06-04',191,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-06-05',192,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-06-06',193,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-06-07',194,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-06-08',195,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-06-09',197,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-06-10',198,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-06-11',200,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('twitter','2026-06-12',200,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-14',25,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-15',25,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-16',25,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-17',26,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-18',26,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-19',26,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-20',27,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-21',27,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-22',28,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-23',28,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-24',28,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-25',28,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-26',28,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-27',29,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-28',29,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-29',30,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-30',30,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-03-31',30,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-01',30,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-02',30,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-03',31,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-04',31,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-05',31,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-06',32,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-07',32,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-08',33,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-09',33,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-10',33,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-11',34,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-12',34,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-13',34,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-14',35,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-15',35,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-16',36,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-17',36,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-18',36,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-19',37,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-20',37,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-21',37,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-22',37,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-23',38,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-24',38,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-25',39,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-26',39,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-27',40,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-28',40,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-29',40,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-04-30',40,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-01',41,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-02',41,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-03',42,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-04',42,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-05',43,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-06',43,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-07',44,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-08',44,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-09',44,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-10',45,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-11',45,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-12',46,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-13',46,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-14',47,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-15',47,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-16',47,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-17',48,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-18',48,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-19',48,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-20',49,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-21',49,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-22',50,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-23',51,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-24',51,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-25',51,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-26',52,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-27',52,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-28',53,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-29',53,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-30',54,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-05-31',54,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-06-01',55,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-06-02',55,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-06-03',56,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-06-04',56,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-06-05',56,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-06-06',57,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-06-07',57,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-06-08',58,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-06-09',59,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-06-10',59,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-06-11',60,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('youtube','2026-06-12',60,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-14',45,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-15',45,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-16',46,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-17',46,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-18',47,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-19',47,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-20',48,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-21',48,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-22',48,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-23',48,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-24',48,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-25',49,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-26',49,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-27',50,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-28',50,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-29',51,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-30',51,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-03-31',51,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-01',52,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-02',52,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-03',52,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-04',53,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-05',54,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-06',54,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-07',55,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-08',55,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-09',55,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-10',56,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-11',56,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-12',56,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-13',57,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-14',58,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-15',58,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-16',59,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-17',59,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-18',59,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-19',59,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-20',60,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-21',60,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-22',61,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-23',62,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-24',62,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-25',63,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-26',63,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-27',64,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-28',64,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-29',64,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-04-30',65,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-01',66,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-02',66,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-03',67,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-04',68,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-05',68,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-06',68,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-07',69,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-08',69,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-09',69,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-10',70,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-11',71,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-12',72,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-13',72,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-14',73,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-15',73,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-16',73,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-17',74,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-18',74,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-19',75,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-20',76,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-21',77,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-22',78,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-23',78,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-24',78,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-25',79,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-26',79,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-27',80,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-28',81,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-29',81,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-30',82,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-05-31',83,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-06-01',83,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-06-02',83,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-06-03',84,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-06-04',84,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-06-05',85,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-06-06',86,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-06-07',87,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-06-08',88,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-06-09',88,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-06-10',89,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-06-11',89,'seed-dummy');
INSERT INTO social_dm_snapshots VALUES('linkedin','2026-06-12',90,'seed-dummy');
CREATE TABLE social_dm_messages (
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
INSERT INTO social_dm_messages VALUES('dm-ig-alex-0','instagram','ig-alex','Alex','alex','saw your reel on the 3-agent setup 🔥 do you actually work with agencies?','in',NULL,'2026-07-18T14:02:00.000Z','seed-dummy');
INSERT INTO social_dm_messages VALUES('dm-ig-alex-1','instagram','ig-alex','Alex','alex','appreciate it! yeah — agencies are exactly who Vantage is built for. what are you running right now?','out',NULL,'2026-07-18T14:09:00.000Z','seed-dummy');
INSERT INTO social_dm_messages VALUES('dm-ig-alex-2','instagram','ig-alex','Alex','alex','SMMA, ~12 clients, drowning in fulfillment tbh 😅','in',NULL,'2026-07-18T14:15:00.000Z','seed-dummy');
INSERT INTO social_dm_messages VALUES('dm-ig-jordan-3','instagram','ig-jordan','Jordan Blake','jordanbuilds','SCALE','in','SCALE','2026-07-18T12:41:00.000Z','seed-dummy');
INSERT INTO social_dm_messages VALUES('dm-ig-jordan-4','instagram','ig-jordan','Jordan Blake','jordanbuilds','boom 💥 here’s the free breakdown → founderos.ai/scale. want me to show how it maps to your funnel?','out','SCALE','2026-07-18T12:41:20.000Z','seed-dummy');
INSERT INTO social_dm_messages VALUES('dm-ig-jordan-5','instagram','ig-jordan','Jordan Blake','jordanbuilds','yes pls','in',NULL,'2026-07-18T13:05:00.000Z','seed-dummy');
INSERT INTO social_dm_messages VALUES('dm-ig-priya-6','instagram','ig-priya','Priya N','priya.builds','replied to your story — I want OUT of retainer hell 😩','in',NULL,'2026-07-17T21:12:00.000Z','seed-dummy');
INSERT INTO social_dm_messages VALUES('dm-ig-priya-7','instagram','ig-priya','Priya N','priya.builds','lol felt. that’s the whole thesis. what’s your current model — retainers or projects?','out',NULL,'2026-07-17T21:30:00.000Z','seed-dummy');
INSERT INTO social_dm_messages VALUES('dm-ig-sam-8','instagram','ig-sam','Sam Ortiz','sam.ortiz.co','what does pricing look like for the done-for-you build?','in',NULL,'2026-07-18T15:48:00.000Z','seed-dummy');
CREATE TABLE social_posts (
  id TEXT PRIMARY KEY,
  caption TEXT NOT NULL,
  media_url TEXT,
  platforms TEXT NOT NULL,
  status TEXT NOT NULL,
  scheduled_for TEXT,
  created_at TEXT NOT NULL
);
INSERT INTO social_posts VALUES('post-seed-1','New Vantage case study — 3x pipeline in 60 days. Full breakdown dropping this week 🚀',NULL,'["instagram","tiktok","twitter"]','queued',NULL,'2026-06-12T18:00:00Z');
CREATE TABLE people (
  id TEXT PRIMARY KEY,
  department_id TEXT NOT NULL REFERENCES departments(id),
  name TEXT NOT NULL,
  role TEXT NOT NULL,
  tools TEXT NOT NULL DEFAULT '[]'
);
INSERT INTO people VALUES('person-marco','dept-sales','Marco','Head of Sales','["recall","ledger"]');
INSERT INTO people VALUES('person-nadia','dept-marketing-growth','Nadia','Head of Growth & Marketing','["postly","dmflow"]');
INSERT INTO people VALUES('person-mia','dept-comms','Mia Torres','Executive Assistant','["imap","slack"]');
INSERT INTO people VALUES('person-dana','dept-finance','Dana Whitfield','Bookkeeper','["stripe","paykit"]');
INSERT INTO people VALUES('person-sasha','dept-clients','Sasha Bell','Account Manager','["ledger","recall"]');
CREATE TABLE sop_tasks (
  id TEXT PRIMARY KEY,
  department_id TEXT NOT NULL REFERENCES departments(id),
  title TEXT NOT NULL,
  summary TEXT NOT NULL DEFAULT '',
  steps TEXT NOT NULL DEFAULT '[]',
  assignee_kind TEXT NOT NULL,
  assignee_id TEXT NOT NULL
);
INSERT INTO sop_tasks VALUES('sop-conductor','dept-tech','Broadcast directives across the fleet','One message in, every agent briefed, replies collected.','["Receive the directive from the operator console","Resolve the target list: the whole fleet, or the pillar the directive names","Poll instance hosts (Clawline, Ollama, tmux) for availability before dispatch","Fan the message out to every target at once and stamp each send","Collect replies as they land and file the run to agent_runs","Report non-responders after sixty seconds so nothing fails silently"]','agent','conductor');
INSERT INTO sop_tasks VALUES('sop-data-agent','dept-tech','Answer questions from Brain','Hybrid search over the second brain, honest fallbacks.','["Parse the incoming question into a optimal-engine query","Run optimal-engine hybrid search (--no-expand) against Supabase","Fall back to local brain-store grep when the database is paused","Rank passages and keep only the ones that actually answer the question","Return cited passages with their source notes, never invented ones","Log unanswerable questions as gaps for the Markdown Auditor to fill"]','agent','data-agent');
INSERT INTO sop_tasks VALUES('sop-markdown-auditor','dept-tech','Audit brain-store markdown health','Keep the knowledge base clean and linkable.','["Walk every markdown file in knowledge/brain-store","Flag broken wiki-links, orphan notes and stale frontmatter","Check generated org docs still match the live agents, SOPs and tools","Write the health report with per-folder scores","Queue fix-ups for the worst offenders and track them to done"]','agent','markdown-auditor');
INSERT INTO sop_tasks VALUES('sop-vector-auditor','dept-tech','Audit the vector index','Embeddings in Supabase must mirror brain-store.','["Ping the Supabase Second Brain project (free tier pauses on idle)","Wake the database and wait until it accepts queries before comparing","Compare pgvector chunk counts against brain-store files","Flag drift and paused-tier warnings on the /brain doctor card","Trigger bge-m3 re-embeds for drifted documents and verify counts after"]','agent','vector-auditor');
INSERT INTO sop_tasks VALUES('sop-stack-monitor','dept-tech','Watch the local stack','Honest status for every port, session and binary.','["Probe the command center :3100 and the worker gateway :8642","Check the brew binaries the agents shell out to (ffmpeg, pdftotext, whisper, gh) and the optimal-engine CLI","Record honest ConnectorStatus, never fake connected","Compare against the last sweep to catch flapping services","Alert the console when something that was up goes down"]','agent','stack-monitor');
INSERT INTO sop_tasks VALUES('sop-comms-digest','dept-comms','Run the 09:00 comms report','Every morning: 24h of email, WhatsApp and Slack, ranked by who needs a reply.','["Pull the trailing 24 hours from all four inboxes, WhatsApp and Slack (one guarded call each — a dead channel degrades the report, it never cancels it)","Load the ranking context: calendar titles for upcoming calls, the Ledger roster for clients, contact tags for community members","Rank every message: calls first, then clients and proposal replies, then community questions, then brand deals, then group chats, companies and software last","Collect the automated senders into an unsubscribe worklist, noisiest first","Store the report so /comms renders it instantly, and log the run against the schedule"]','agent','comms-digest');
INSERT INTO sop_tasks VALUES('sop-comms-agent','dept-comms','Compose the unified comms feed','Three channels, one timeline at /comms.','["Collect fresh output from the Gmail, WhatsApp and Slack workers","Dedupe and merge everything into one ordered timeline","Tag each entry with its contact tier","Bubble urgent and reply-needed items to the top of the feed","Publish the feed and report which channels are live"]','agent','comms-agent');
INSERT INTO sop_tasks VALUES('sop-gmail-worker','dept-comms','Triage the four Gmail inboxes','IMAP slots 1–4 read, classified, escalated.','["Connect the four configured IMAP inboxes on the sync cadence","Pull unread counts and every thread newer than the last sweep","Classify each thread: urgent, reply-needed, waiting-on-us, FYI","Draft suggested replies for reply-needed threads in Alex voice","Hand urgent threads to the escalation queue with a one-line summary","Surface anything from a client domain to the Clients pillar too"]','agent','gmail-worker');
INSERT INTO sop_tasks VALUES('sop-whatsapp-worker','dept-comms','Monitor WhatsApp chats','Local team chats surfaced.','["Read the local ChatStorage.sqlite (read-only, nothing leaves the machine)","Surface new messages from the LC and Vantage team chats","Map senders to their contact tags","Flag messages that mention money, deadlines or blockers","Push tagged messages into the unified feed"]','agent','whatsapp-worker');
INSERT INTO sop_tasks VALUES('sop-slack-worker','dept-comms','Digest Slack channels','Joined channels summarized into the feed.','["List channels the bot has joined","Pull the latest messages per channel since the last sweep","Summarize each channel into a short digest","Call out direct mentions and unanswered questions separately","Push the digest into the unified feed"]','agent','slack-worker');
INSERT INTO sop_tasks VALUES('sop-social-agent','dept-marketing-growth','Run the daily content pipeline','Calendar → briefs → assets → publish queue.','["Pull today’s slots from the content calendar","Brief the creative workers (Adsmith, Renderly, Reelkit) with hooks and formats","Collect finished assets and check them against the brief","Reject anything off-brand with a one-line reason so the fix is fast","Queue approved posts for the Postly publisher with per-platform captions","Log what shipped to the calendar so tomorrow’s brief starts warm"]','agent','social-agent');
INSERT INTO sop_tasks VALUES('sop-newsletter-agent','dept-marketing-growth','Draft the next newsletter issue','Aim the draft at what the list actually opened and clicked.','["Pull the newsletter send history and build the performance brief","Say plainly when the history is too thin to call a pattern","Write three subject lines and the issue against the skill file","Avoid repeating the angle of any recent issue in the brief","Hand the draft over unsent, and never state a metric that was not measured"]','agent','newsletter-agent');
INSERT INTO sop_tasks VALUES('sop-postly-publisher','dept-marketing-growth','Publish to six platforms','One queue out to every @founderos.ai surface.','["Take the next queued post from the pipeline","Adapt the caption per platform (IG, TikTok, X, YouTube, LinkedIn, Facebook)","Publish through the Postly API","Record post ids and verify each went live","Retry failed platforms once, then flag them to the Social Agent"]','agent','postly-publisher');
INSERT INTO sop_tasks VALUES('sop-adsmith-creative','dept-marketing-growth','Generate UGC ad variants','Vantage ad angles rendered as UGC actors.','["Take the ad brief with hook, angle and offer","Generate actor variants across Veo / Sora / Kling","Cull the takes that break the brief before rendering finals","Render finals and name them by angle","Deliver the batch to creative review with a variant sheet"]','agent','adsmith-creative');
INSERT INTO sop_tasks VALUES('sop-reelkit-editor','dept-marketing-growth','Cut short-form edits','Raw footage to platform-ready crops.','["Transcribe the source clip locally with Whisper","Pick the hook and strongest segments from the transcript","Render through the Reelkit pipeline with the right theme (LC / Vantage)","Check captions land on beat before exporting anything","Export platform crops and hand them to the pipeline"]','agent','reelkit-editor');
INSERT INTO sop_tasks VALUES('sop-renderly-creative','dept-marketing-growth','Produce AI visuals','Stills and motion from the creative brief.','["Read the creative brief and pick the matching Renderly model","Generate stills or motion to the spec in the brief","Cull to the strongest takes before spending on upscales","Upscale the picks to delivery resolution","Hand finals to the editor for assembly with the brief attached"]','agent','renderly-creative');
INSERT INTO sop_tasks VALUES('sop-dmflow-mcp','dept-marketing-growth','Automate DM funnels','Keyword triggers to booked conversations.','["Watch configured trigger keywords across platforms","Fire the matching DMFlow flow for each trigger","Tag subscribers by intent as they move through the flow","Hand hot leads to the Sales pillar with their conversation history","Report conversions back to the growth dashboard"]','agent','dmflow-mcp');
INSERT INTO sop_tasks VALUES('sop-nadia','dept-marketing-growth','Set content strategy & approve drops','The human editorial gate on everything published.','["Review last cycle’s performance numbers from the dashboard","Set this week’s angles and slot them on the calendar","Approve or kill every queued asset before it publishes","Spot-check published posts landed exactly as approved","Debrief the crew on what worked and what died"]','person','person-nadia');
INSERT INTO sop_tasks VALUES('sop-sales-agent','dept-sales','Keep the pipeline moving','Deals inspected daily, nothing stalls silently.','["Pull every open deal and its stage from Ledger each morning","Rank deals by value and days-in-stage; anything past 7 days is stalled","Attach a concrete next action and owner to every stalled deal","Prepare payment links across PayKit, Stripe and FlexPay before calls","Brief Marco with the top five deals and their objections before each call","Log stage changes back to Ledger the same day they happen"]','agent','sales-agent');
INSERT INTO sop_tasks VALUES('sop-lc-lane','dept-sales','Run the Launchpad Cohort lane','Webinar registrants to closed LC deals.','["Track LC leads from webinar registration to booked call","Chase no-shows with the rebooking sequence within 24 hours","Sync every stage change back to Ledger","Reconcile LC payments against Stripe","Report lane revenue to the pipeline brief"]','agent','launchpad-cohort-sales');
INSERT INTO sop_tasks VALUES('sop-vantage-lane','dept-sales','Run the Vantage lane','Local-business inbound worked end to end.','["Qualify inbound Vantage leads against the ICP","Book qualified leads onto Marco’s calendar with context attached","Sync stage changes back to Ledger","Reconcile payments across PayKit and Stripe","Report lane revenue to the pipeline brief"]','agent','vantage-sales');
INSERT INTO sop_tasks VALUES('sop-vantage-paykit','dept-sales','Reconcile the Vantage PayKit lane','PayKit customers matched to CRM deals.','["Pull month-to-date customers from PayKit","Match each payment to its Ledger deal","Flag payments with no deal and deals with no payment","Chase every mismatch to a resolution, not just a flag","Post month-to-date totals to Finances"]','agent','vantage-paykit');
INSERT INTO sop_tasks VALUES('sop-sales-calls-data','dept-sales','Mine sales-call recordings','Every Recall call and Plaud recording becomes CRM intelligence.','["Ingest Recall notes after each recorded call","Ingest Plaud transcripts + AI notes after each in-person meeting or site walk","Extract objections, commitments and next steps","Write the extract back to the Ledger record","Tag calls where pricing or competitors came up","Feed recurring patterns into the pipeline brief"]','agent','sales-calls-data');
INSERT INTO sop_tasks VALUES('sop-crm-pulse','dept-sales','Keep Ledger clean','A CRM the numbers can be trusted from.','["Scan records for missing fields and duplicates","Verify deal stages match what actually happened","Merge duplicates and backfill whatever can be backfilled safely","Nudge lane owners on records gone stale","Snapshot pipeline metrics for the dashboard"]','agent','crm-pulse');
INSERT INTO sop_tasks VALUES('sop-brand-deal-agent','dept-sales','Work the brand deal pipeline as Vera','Qualify, quote, chase and bump, without ever sending.','["Read the OS brand deal store and rank what needs answering today","Check the contact governor before touching any thread, and respect a refusal","Draft the reply, counter or bump as Vera, speaking about Alex in third person","Escalate anything below floor, equity shaped, or asking for a call","Leave every draft for Alex to send, and never claim one went out"]','agent','brand-deal-agent');
INSERT INTO sop_tasks VALUES('sop-marco','dept-sales','Run discovery & close calls','The human on the phone from hello to signed.','["Review the pre-call brief and the lead’s last three touches","Run the discovery script and qualify hard on budget and timeline","Handle objections with the objection sheet, never improvise pricing","Present the matching offer and the financing option when it fits","Log the outcome, next step and payment link before the next call"]','person','person-marco');
INSERT INTO sop_tasks VALUES('sop-paykit','dept-finance','Track PayKit income','Month-to-date, split by venture, refunds flagged.','["Pull month-to-date customers from the PayKit API","Split income by venture (LC vs Vantage)","Record the income snapshot for the Finances view","Flag refunds and disputes the day they land","Reconcile the running total against the month-end books"]','agent','paykit-sales');
INSERT INTO sop_tasks VALUES('sop-stripe','dept-finance','Track Stripe income','Balance and charges labeled Launchpad Cohort.','["Pull balance and recent charges from Stripe","Label income to Launchpad Cohort","Record the snapshot for the income chart","Flag anomalies against the trailing average","Note upcoming payouts so cash flow is never a surprise"]','agent','stripe-sales');
INSERT INTO sop_tasks VALUES('sop-processor-confirm','dept-finance','Confirm payments across processors','No deal marked paid without an API receipt.','["Receive the payment claim from a sales lane","Check the claimed processor’s API (Stripe / PayPal / Square / Whop / PayKit)","Confirm the charge or flag the mismatch loudly","Write the confirmation onto the deal record","Keep an audit trail of every confirmation for month-end close"]','agent','processor-confirmation');
INSERT INTO sop_tasks VALUES('sop-flexpay','dept-finance','Quote financing options','Payment plans attached to live offers.','["Take the deal size and buyer profile from the lane","Pull matching plan options from FlexPay","Attach terms to the offer before the call","Track which plans get accepted and which stall deals","Report acceptance rates so pricing keeps getting sharper"]','agent','flexpay-financing');
INSERT INTO sop_tasks VALUES('sop-payments-pulse','dept-finance','Watch processor health','Every processor pinged, status recorded honestly.','["Ping each processor registered in the registry","Record honest ConnectorStatus, never fake connected","Alert Finances when a processor goes down","Re-check failed processors on a tighter cadence until they recover","Keep the uptime history for the analytics view"]','agent','payments-pulse');
INSERT INTO sop_tasks VALUES('sop-client-roster','dept-clients','Keep the client roster live','One list of every client, always current.','["Pull clients and deal states from Ledger and PayKit every morning","Reconcile them against the funnel journeys and payment records","Mark each account active, at risk, or churned with a reason","Flag stale records and missing fields to the owning lane","Publish the roster to the Clients pillar and note the deltas"]','agent','client-roster');
INSERT INTO sop_tasks VALUES('sop-client-onboarding','dept-clients','Onboard new clients','Closed-won to kickoff without a dropped step.','["Trigger when a deal moves to closed-won in Ledger","Verify payment landed with Processor Confirm before anything ships","Send the welcome pack and countersigned agreement within 24 hours","Create their Slack channel, invite the client team, pin the scope doc","Book the kickoff call inside 5 business days and confirm attendance","Collect access and assets (logins, brand kit, tracking) in one request","Hand to Client Success with full context notes and the risk flags"]','agent','client-onboarding');
INSERT INTO sop_tasks VALUES('sop-client-success','dept-clients','Service active clients','Cadence, deliverables and renewals on rails.','["Run the weekly check-in cadence per client, no skipped weeks","Track deliverables against the sold scope and flag slippage early","Log Recall call notes back to the client record the same day","Score account health monthly: green, watch, or at risk with a reason","Raise renewals and upsell openings 30 days out to Sasha and Sales"]','agent','client-success');
INSERT INTO sop_tasks VALUES('sop-mia','dept-comms','Handle escalations & VIP replies','The human hands on the threads that need judgment.','["Review the escalation queue the workers built overnight","Draft replies in Alex’s voice for VIP threads","Send what is cleared, file the rest for Alex’s approval","Chase any thread waiting on us for more than 24 hours","Close the loop in /comms so nothing dangles"]','person','person-mia');
INSERT INTO sop_tasks VALUES('sop-dana','dept-finance','Close the books monthly','The human sign-off on every month’s numbers.','["Import bank and processor statements for the month by the 3rd","Categorize transactions using the statement’s own categories","Reconcile against the income the agents recorded and chase every gap","Confirm refunds and disputes are reflected in the venture totals","Deliver the month-end P&L to Alex with three lines of commentary"]','person','person-dana');
INSERT INTO sop_tasks VALUES('sop-sasha','dept-clients','Own the client relationships','The human accountable for every account.','["Run kickoff and quarterly business review calls","Resolve escalations the same day they land","Approve scope changes before work starts","Review account health scores with Client Success monthly","Sign off renewals and hand pricing changes to Sales"]','person','person-sasha');
CREATE TABLE lead_magnets (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  offer TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL,
  status TEXT NOT NULL,
  captures TEXT NOT NULL,
  destination TEXT NOT NULL DEFAULT '',
  source TEXT NOT NULL DEFAULT '',
  launched_at TEXT NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  origin TEXT NOT NULL DEFAULT 'seed'
);
INSERT INTO lead_magnets VALUES('operator-stack','The Operator Stack','Every layer of the agent stack, and what to use instead of each one','https://stack.example.com','live','email','Newsletter · main list','Carousel · "One person, a company of agents" (comment STACK)','2026-08-12','Ungated. Newsletter signup plus a separate cohort waitlist form.','seed');
INSERT INTO lead_magnets VALUES('automation-teardown','The Automation Teardown','A workflow pulled apart step by step, with the hours each one costs','https://teardown.example.com','live','email','Newsletter · main list','Short · "Where the week actually goes" (comment TEARDOWN)','2026-08-05','Built from the workflows view. Doubles as the cohort lesson one handout.','seed');
INSERT INTO lead_magnets VALUES('cohort-waitlist','Cohort Waitlist','A seat in the next cohort before it opens publicly','https://waitlist.example.com','paused','email','Newsletter · cohort waitlist segment','Bio link + end cards','2026-07-28','Paused between cohorts. Reopen when the next intake is dated.','seed');
CREATE TABLE funnel_contacts (
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
INSERT INTO funnel_contacts VALUES('fc-jake-moreau','Jake Moreau','launchpad-cohort','converted','Launchpad Cohort — mentorship (PIF)',6800.0,'hot',100,NULL,NULL,NULL,NULL,NULL,NULL,'2026-08-11');
INSERT INTO funnel_contacts VALUES('fc-priya-shah','Priya Shah','launchpad-cohort','converted','Launchpad Cohort — mentorship (3-pay)',2600.0,'warm',95,NULL,NULL,NULL,NULL,NULL,NULL,'2026-08-25');
INSERT INTO funnel_contacts VALUES('fc-danny-okafor','Danny Okafor','launchpad-cohort','converted','Launchpad Cohort — mentorship (PIF)',6800.0,'hot',100,NULL,NULL,NULL,NULL,NULL,NULL,'2026-09-01');
INSERT INTO funnel_contacts VALUES('fc-sofia-reyes','Sofia Reyes','launchpad-cohort','converted','Launchpad Cohort — mentorship (3-pay)',2600.0,'warm',95,NULL,NULL,NULL,NULL,NULL,NULL,'2026-09-08');
INSERT INTO funnel_contacts VALUES('fc-liam-carter','Liam Carter','launchpad-cohort','engaged',NULL,NULL,'cold',15,NULL,NULL,NULL,NULL,NULL,NULL,'2026-09-12');
INSERT INTO funnel_contacts VALUES('fc-marcus-webb','Marcus Webb','launchpad-cohort','nurtured',NULL,NULL,'warm',42,NULL,NULL,NULL,NULL,NULL,NULL,'2026-09-15');
INSERT INTO funnel_contacts VALUES('fc-tayla-nguyen','Tayla Nguyen','launchpad-cohort','opted_in',NULL,NULL,'hot',84,'tayla.nguyen@example.com','+15550100841',NULL,NULL,NULL,NULL,'2026-10-05');
INSERT INTO funnel_contacts VALUES('fc-remy-cole','Remy Cole','launchpad-cohort','engaged',NULL,NULL,'cold',25,NULL,NULL,NULL,NULL,NULL,NULL,'2026-07-17');
INSERT INTO funnel_contacts VALUES('fc-jordan-blake','Jordan Blake','launchpad-cohort','engaged',NULL,NULL,'cold',20,NULL,NULL,NULL,NULL,NULL,NULL,'2026-06-13');
INSERT INTO funnel_contacts VALUES('fc-ava-stone','Ava Stone — Northwind Legal','vantage','converted','Vantage — AI intake build (sprint)',12000.0,'hot',100,NULL,NULL,NULL,NULL,NULL,NULL,'2026-08-13');
INSERT INTO funnel_contacts VALUES('fc-omar-haddad','Omar Haddad — Pulse Fitness Group','vantage','converted','Vantage — AI ops retainer (monthly)',4500.0,'warm',95,NULL,NULL,NULL,NULL,NULL,NULL,'2026-08-22');
INSERT INTO funnel_contacts VALUES('fc-elena-brooks','Elena Brooks — Harbor Dental','vantage','converted','Vantage — AI intake build (sprint)',9500.0,'hot',100,NULL,NULL,NULL,NULL,NULL,NULL,'2026-09-08');
INSERT INTO funnel_contacts VALUES('fc-noah-fields','Noah Fields — Fields Roofing','vantage','opted_in',NULL,NULL,'warm',66,NULL,NULL,NULL,NULL,NULL,NULL,'2026-10-01');
INSERT INTO funnel_contacts VALUES('fc-grace-lin','Grace Lin — Lin & Co Accounting','vantage','opted_in',NULL,NULL,'warm',74,'grace@linandco.example.com','+15550100742','Grace Lin','Lin & Co Accounting','Managing Partner','https://linkedin.com/in/gracelin-example','2026-10-03');
CREATE TABLE funnel_touches (
  id TEXT PRIMARY KEY,
  contact_id TEXT NOT NULL REFERENCES funnel_contacts(id),
  seq INTEGER NOT NULL,
  stage TEXT NOT NULL,
  channel TEXT NOT NULL,
  label TEXT NOT NULL,
  source TEXT NOT NULL,
  at TEXT NOT NULL
);
INSERT INTO funnel_touches VALUES('fc-jake-moreau-t1','fc-jake-moreau',1,'first_touch','organic','IG reel: "3 AI offers that close themselves"','trakyo','2026-08-11');
INSERT INTO funnel_touches VALUES('fc-jake-moreau-t2','fc-jake-moreau',2,'engaged','dm','Replied to story CTA — "wants out of retainer hell"','manual','2026-08-13');
INSERT INTO funnel_touches VALUES('fc-jake-moreau-t3','fc-jake-moreau',3,'nurtured','email','Day-3 email: student case study (0→22k/mo)','manual','2026-08-16');
INSERT INTO funnel_touches VALUES('fc-jake-moreau-t4','fc-jake-moreau',4,'opted_in','call','Booked strategy call via Trakyo link','trakyo','2026-08-19');
INSERT INTO funnel_touches VALUES('fc-jake-moreau-t5','fc-jake-moreau',5,'converted','checkout','Paid in full — PayKit checkout','manual','2026-08-21');
INSERT INTO funnel_touches VALUES('fc-priya-shah-t1','fc-priya-shah',1,'first_touch','ads','Meta ad: "Agency owners — install AI in 30 days"','meta-ads','2026-08-25');
INSERT INTO funnel_touches VALUES('fc-priya-shah-t2','fc-priya-shah',2,'engaged','ads','Watched VSL to 80% — retarget pool','meta-ads','2026-08-25');
INSERT INTO funnel_touches VALUES('fc-priya-shah-t3','fc-priya-shah',3,'opted_in','webinar','Registered + attended the live training','manual','2026-08-28');
INSERT INTO funnel_touches VALUES('fc-priya-shah-t4','fc-priya-shah',4,'converted','checkout','First of 3 payments — PayKit','manual','2026-08-30');
INSERT INTO funnel_touches VALUES('fc-danny-okafor-t1','fc-danny-okafor',1,'first_touch','organic','TikTok: "day in the life running an AI agency"','trakyo','2026-09-01');
INSERT INTO funnel_touches VALUES('fc-danny-okafor-t2','fc-danny-okafor',2,'engaged','organic','Binged 6 reels, followed, saved lead magnet post','trakyo','2026-09-03');
INSERT INTO funnel_touches VALUES('fc-danny-okafor-t3','fc-danny-okafor',3,'nurtured','ads','Retargeting ad: student-wins carousel','meta-ads','2026-09-06');
INSERT INTO funnel_touches VALUES('fc-danny-okafor-t4','fc-danny-okafor',4,'opted_in','call','Booked call from link-in-bio (Trakyo attributed)','trakyo','2026-09-09');
INSERT INTO funnel_touches VALUES('fc-danny-okafor-t5','fc-danny-okafor',5,'converted','checkout','Paid in full — PayKit checkout','manual','2026-09-10');
INSERT INTO funnel_touches VALUES('fc-sofia-reyes-t1','fc-sofia-reyes',1,'first_touch','organic','YT long-form: "how I''d start an agency in 2026"','trakyo','2026-09-08');
INSERT INTO funnel_touches VALUES('fc-sofia-reyes-t2','fc-sofia-reyes',2,'engaged','email','Joined newsletter from YT description','manual','2026-09-09');
INSERT INTO funnel_touches VALUES('fc-sofia-reyes-t3','fc-sofia-reyes',3,'nurtured','email','Newsletter: pricing-psychology issue clicked','manual','2026-09-13');
INSERT INTO funnel_touches VALUES('fc-sofia-reyes-t4','fc-sofia-reyes',4,'opted_in','webinar','Attended the live training, stayed for offer','manual','2026-09-16');
INSERT INTO funnel_touches VALUES('fc-sofia-reyes-t5','fc-sofia-reyes',5,'converted','checkout','First of 3 payments — PayKit','manual','2026-09-17');
INSERT INTO funnel_touches VALUES('fc-liam-carter-t1','fc-liam-carter',1,'first_touch','ads','Meta ad: "stop selling hours" (cold traffic)','meta-ads','2026-09-12');
INSERT INTO funnel_touches VALUES('fc-liam-carter-t2','fc-liam-carter',2,'engaged','ads','Clicked through, watched VSL 45%','meta-ads','2026-09-12');
INSERT INTO funnel_touches VALUES('fc-liam-carter-t3','fc-liam-carter',3,'engaged','ads','Retarget click — opened application form, abandoned','meta-ads','2026-09-16');
INSERT INTO funnel_touches VALUES('fc-liam-carter-t4','fc-liam-carter',4,'engaged','email','Abandoned-form email opened, no reply yet','manual','2026-09-18');
INSERT INTO funnel_touches VALUES('fc-marcus-webb-t1','fc-marcus-webb',1,'first_touch','organic','IG carousel: "agency niches that print in 2026"','trakyo','2026-09-15');
INSERT INTO funnel_touches VALUES('fc-marcus-webb-t2','fc-marcus-webb',2,'engaged','dm','DMFlow keyword "SCALE" → DM flow','manual','2026-09-15');
INSERT INTO funnel_touches VALUES('fc-marcus-webb-t3','fc-marcus-webb',3,'nurtured','email','Lead magnet delivered, day-1 email opened','manual','2026-09-27');
INSERT INTO funnel_touches VALUES('fc-marcus-webb-t4','fc-marcus-webb',4,'nurtured','email','Newsletter: student-win breakdown clicked','manual','2026-09-29');
INSERT INTO funnel_touches VALUES('fc-tayla-nguyen-t1','fc-tayla-nguyen',1,'first_touch','organic','TikTok: "AI receptionist demo" went semi-viral','trakyo','2026-10-05');
INSERT INTO funnel_touches VALUES('fc-tayla-nguyen-t2','fc-tayla-nguyen',2,'engaged','organic','Profile visit → followed + commented','trakyo','2026-10-05');
INSERT INTO funnel_touches VALUES('fc-tayla-nguyen-t3','fc-tayla-nguyen',3,'nurtured','dm','DM convo — asked about payment plans','manual','2026-10-06');
INSERT INTO funnel_touches VALUES('fc-tayla-nguyen-t4','fc-tayla-nguyen',4,'opted_in','call','Call booked for next week (Trakyo attributed)','trakyo','2026-10-07');
INSERT INTO funnel_touches VALUES('fc-remy-cole-t1','fc-remy-cole',1,'first_touch','organic','IG reel: "fire your lead-gen agency"','trakyo','2026-07-17');
INSERT INTO funnel_touches VALUES('fc-remy-cole-t2','fc-remy-cole',2,'engaged','dm','Story-reply convo, asked for pricing','manual','2026-07-21');
INSERT INTO funnel_touches VALUES('fc-remy-cole-t3','fc-remy-cole',3,'engaged','email','Pricing breakdown sent, opened twice','manual','2026-07-27');
INSERT INTO funnel_touches VALUES('fc-remy-cole-t4','fc-remy-cole',4,'engaged','email','Follow-up: "circling back" — no reply since','manual','2026-07-31');
INSERT INTO funnel_touches VALUES('fc-jordan-blake-t1','fc-jordan-blake',1,'first_touch','ads','Meta ad: "quit your 9-5 with one client" (old campaign)','meta-ads','2026-06-13');
INSERT INTO funnel_touches VALUES('fc-jordan-blake-t2','fc-jordan-blake',2,'engaged','ads','Clicked through, watched VSL 30%','meta-ads','2026-06-13');
INSERT INTO funnel_touches VALUES('fc-jordan-blake-t3','fc-jordan-blake',3,'engaged','dm','One-word DM reply, then silence','manual','2026-06-19');
INSERT INTO funnel_touches VALUES('fc-jordan-blake-t4','fc-jordan-blake',4,'engaged','email','Re-engagement email bounced-opened, no click','manual','2026-06-27');
INSERT INTO funnel_touches VALUES('fc-ava-stone-t1','fc-ava-stone',1,'first_touch','organic','LinkedIn post: legal-intake automation teardown','trakyo','2026-08-13');
INSERT INTO funnel_touches VALUES('fc-ava-stone-t2','fc-ava-stone',2,'engaged','email','Replied to newsletter — "this is our exact bottleneck"','manual','2026-08-15');
INSERT INTO funnel_touches VALUES('fc-ava-stone-t3','fc-ava-stone',3,'opted_in','call','Discovery call booked via site (Trakyo attributed)','trakyo','2026-08-20');
INSERT INTO funnel_touches VALUES('fc-ava-stone-t4','fc-ava-stone',4,'nurtured','email','Proposal + Loom walkthrough sent, viewed 3×','manual','2026-08-23');
INSERT INTO funnel_touches VALUES('fc-ava-stone-t5','fc-ava-stone',5,'converted','checkout','Signed — 50% deposit via Stripe invoice','manual','2026-08-27');
INSERT INTO funnel_touches VALUES('fc-omar-haddad-t1','fc-omar-haddad',1,'first_touch','ads','Meta ad: "your gym''s front desk, automated"','meta-ads','2026-08-22');
INSERT INTO funnel_touches VALUES('fc-omar-haddad-t2','fc-omar-haddad',2,'engaged','ads','Case-study page dwell 4m — retarget pool','meta-ads','2026-08-23');
INSERT INTO funnel_touches VALUES('fc-omar-haddad-t3','fc-omar-haddad',3,'nurtured','email','ROI one-pager emailed after form fill','manual','2026-08-26');
INSERT INTO funnel_touches VALUES('fc-omar-haddad-t4','fc-omar-haddad',4,'opted_in','call','Demo call — 3 locations scoped','manual','2026-08-29');
INSERT INTO funnel_touches VALUES('fc-omar-haddad-t5','fc-omar-haddad',5,'converted','checkout','Retainer live — Stripe subscription','manual','2026-09-02');
INSERT INTO funnel_touches VALUES('fc-elena-brooks-t1','fc-elena-brooks',1,'first_touch','organic','IG reel: missed-call → booked-patient demo','trakyo','2026-09-08');
INSERT INTO funnel_touches VALUES('fc-elena-brooks-t2','fc-elena-brooks',2,'engaged','dm','DM: "does this work for dental?"','manual','2026-09-09');
INSERT INTO funnel_touches VALUES('fc-elena-brooks-t3','fc-elena-brooks',3,'opted_in','call','Discovery call via link-in-bio (Trakyo attributed)','trakyo','2026-09-12');
INSERT INTO funnel_touches VALUES('fc-elena-brooks-t4','fc-elena-brooks',4,'converted','checkout','Signed — deposit via Stripe invoice','manual','2026-09-16');
INSERT INTO funnel_touches VALUES('fc-noah-fields-t1','fc-noah-fields',1,'first_touch','ads','Meta ad: "book 20 estimates/mo on autopilot"','meta-ads','2026-10-01');
INSERT INTO funnel_touches VALUES('fc-noah-fields-t2','fc-noah-fields',2,'engaged','ads','Lead form opened, 60% VSL','meta-ads','2026-10-01');
INSERT INTO funnel_touches VALUES('fc-noah-fields-t3','fc-noah-fields',3,'nurtured','email','Follow-up sequence day 2 — case study clicked','manual','2026-10-04');
INSERT INTO funnel_touches VALUES('fc-noah-fields-t4','fc-noah-fields',4,'opted_in','call','Discovery call booked for Friday','manual','2026-10-07');
INSERT INTO funnel_touches VALUES('fc-grace-lin-t1','fc-grace-lin',1,'first_touch','organic','X thread: client-onboarding agent breakdown','trakyo','2026-10-03');
INSERT INTO funnel_touches VALUES('fc-grace-lin-t2','fc-grace-lin',2,'engaged','organic','Followed + bookmarked, visited site twice','trakyo','2026-10-04');
INSERT INTO funnel_touches VALUES('fc-grace-lin-t3','fc-grace-lin',3,'nurtured','email','Newsletter signup — welcome sequence started','manual','2026-10-06');
INSERT INTO funnel_touches VALUES('fc-grace-lin-t4','fc-grace-lin',4,'opted_in','call','Call request form submitted (Trakyo attributed)','trakyo','2026-10-08');
CREATE TABLE workflows (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  subtitle TEXT NOT NULL DEFAULT '',
  revenue_usd INTEGER NOT NULL DEFAULT 0,
  ord INTEGER NOT NULL DEFAULT 0,
  steps TEXT NOT NULL DEFAULT '[]'
);
INSERT INTO workflows VALUES('wf-vantage-sales','Vantage sales machine','Cold outbound to closed retainer.',120000,0,'[{"id":"wf-mer-1","title":"Run outbound campaigns","detail":"","branch":null,"ownerKind":"agent","owner":"Postly Publisher","hoursPerWeek":6,"tools":["postly","adsmith"],"edgeLabel":"replies","leakUsd":null,"automation":{"title":"Always-on content + DM outreach","state":"live","recoveredUsd":4200}},{"id":"wf-mer-2","title":"Qualify replies","detail":"","branch":null,"ownerKind":"agent","owner":"Comms Agent","hoursPerWeek":9,"tools":["dmflow","gmail"],"edgeLabel":"qualified","leakUsd":14000,"automation":{"title":"Auto-qualify + book","state":"suggested","recoveredUsd":9000}},{"id":"wf-mer-3","title":"Book demos","detail":"","branch":null,"ownerKind":"human","owner":"Alex · Founder","hoursPerWeek":4,"tools":["calendar","ledger"],"edgeLabel":"demo","leakUsd":null,"automation":null},{"id":"wf-mer-4","title":"Sales call","detail":"","branch":null,"ownerKind":"human","owner":"Alex · Founder","hoursPerWeek":10,"tools":["ledger"],"edgeLabel":"proposal","leakUsd":null,"automation":null},{"id":"wf-mer-5","title":"Proposal & follow-up","detail":"","branch":null,"ownerKind":"human","owner":"Alex · Founder","hoursPerWeek":5,"tools":["proposal-gen","gmail"],"edgeLabel":"won","leakUsd":6000,"automation":{"title":"Proposal follow-up sequence","state":"suggested","recoveredUsd":6000}},{"id":"wf-mer-6","title":"Onboard & deliver","detail":"","branch":null,"ownerKind":"agent","owner":"Onboarding Agent","hoursPerWeek":3,"tools":["ledger","slack"],"edgeLabel":null,"leakUsd":null,"automation":{"title":"Onboarding rails","state":"live","recoveredUsd":3000}}]');
INSERT INTO workflows VALUES('wf-lc-delivery','Launchpad Cohort delivery','Webinar lead to retained program member.',80000,1,'[{"id":"wf-lc-1","title":"Capture webinar leads","detail":"","branch":null,"ownerKind":"agent","owner":"GoHighLevel","hoursPerWeek":2,"tools":["ghl"],"edgeLabel":"registered","leakUsd":null,"automation":{"title":"Webinar to GHL sync","state":"live","recoveredUsd":2500}},{"id":"wf-lc-2","title":"Nurture in GHL","detail":"","branch":null,"ownerKind":"agent","owner":"GoHighLevel","hoursPerWeek":3,"tools":["ghl"],"edgeLabel":"booked","leakUsd":8000,"automation":{"title":"Nurture sequences","state":"live","recoveredUsd":5000}},{"id":"wf-lc-3","title":"Strategy call","detail":"","branch":null,"ownerKind":"human","owner":"Alex · Founder","hoursPerWeek":8,"tools":["ghl","calendar"],"edgeLabel":"closed","leakUsd":null,"automation":null},{"id":"wf-lc-4","title":"Deliver program","detail":"","branch":null,"ownerKind":"human","owner":"LC Team","hoursPerWeek":12,"tools":["skool"],"edgeLabel":"retained","leakUsd":5000,"automation":{"title":"Skool community ops","state":"suggested","recoveredUsd":4000}},{"id":"wf-lc-5","title":"Track attribution","detail":"","branch":null,"ownerKind":"agent","owner":"Trakyo","hoursPerWeek":1,"tools":["trakyo"],"edgeLabel":null,"leakUsd":null,"automation":{"title":"Revenue attribution","state":"suggested","recoveredUsd":0}}]');
CREATE TABLE skills (
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
INSERT INTO skills VALUES('skill-outbound','Cold outbound sequencing','Sales','Multi-touch DM + content cadence that opens conversations at scale.','postly-publisher','live','["postly","dmflow"]',unistr('---\u000aname: cold-outbound-sequencing\u000adescription: Multi-touch DM + content cadence that opens conversations at scale.\u000acategory: Sales\u000astatus: live\u000a---\u000a\u000a# Cold outbound sequencing\u000a\u000aMulti-touch DM + content cadence that opens conversations at scale.\u000a\u000a## When to use\u000aReach for this when the sales flow needs to cold outbound sequencing. It runs on `postly`, `dmflow`.\u000a\u000a## Status\u000aLive in production. The owning agent runs this today.\u000a'),0);
INSERT INTO skills VALUES('skill-qualify','Reply qualification','Sales','Reads inbound replies, scores intent, and books the qualified ones.','comms-agent','live','["dmflow","gmail"]',unistr('---\u000aname: reply-qualification\u000adescription: Reads inbound replies, scores intent, and books the qualified ones.\u000acategory: Sales\u000astatus: live\u000a---\u000a\u000a# Reply qualification\u000a\u000aReads inbound replies, scores intent, and books the qualified ones.\u000a\u000a## When to use\u000aReach for this when the sales flow needs to reply qualification. It runs on `dmflow`, `gmail`.\u000a\u000a## Status\u000aLive in production. The owning agent runs this today.\u000a'),1);
INSERT INTO skills VALUES('skill-proposal','Proposal drafting','Sales','Turns a call transcript into a tailored, on-brand proposal.',NULL,'learning','["proposal-gen","ledger"]',unistr('---\u000aname: proposal-drafting\u000adescription: Turns a call transcript into a tailored, on-brand proposal.\u000acategory: Sales\u000astatus: learning\u000a---\u000a\u000a# Proposal drafting\u000a\u000aTurns a call transcript into a tailored, on-brand proposal.\u000a\u000a## When to use\u000aReach for this when the sales flow needs to proposal drafting. It runs on `proposal-gen`, `ledger`.\u000a\u000a## Status\u000aIn training. Runs with a human in the loop while it calibrates.\u000a'),2);
INSERT INTO skills VALUES('skill-hooks','Hook writing','Content','Short-form hooks and captions tuned to each platform.','social-agent','live','["postly"]',unistr('---\u000aname: hook-writing\u000adescription: Short-form hooks and captions tuned to each platform.\u000acategory: Content\u000astatus: live\u000a---\u000a\u000a# Hook writing\u000a\u000aShort-form hooks and captions tuned to each platform.\u000a\u000a## When to use\u000aReach for this when the content flow needs to hook writing. It runs on `postly`.\u000a\u000a## Status\u000aLive in production. The owning agent runs this today.\u000a'),3);
INSERT INTO skills VALUES('skill-ugc','UGC generation','Content','Generates ad-ready UGC variants (Veo / Sora / Kling).','adsmith-creative','live','["adsmith"]',unistr('---\u000aname: ugc-generation\u000adescription: Generates ad-ready UGC variants (Veo / Sora / Kling).\u000acategory: Content\u000astatus: live\u000a---\u000a\u000a# UGC generation\u000a\u000aGenerates ad-ready UGC variants (Veo / Sora / Kling).\u000a\u000a## When to use\u000aReach for this when the content flow needs to ugc generation. It runs on `adsmith`.\u000a\u000a## Status\u000aLive in production. The owning agent runs this today.\u000a'),4);
INSERT INTO skills VALUES('skill-edit','Video editing','Content','Cuts reels and highlight clips programmatically.','reelkit-editor','live','["reelkit"]',unistr('---\u000aname: video-editing\u000adescription: Cuts reels and highlight clips programmatically.\u000acategory: Content\u000astatus: live\u000a---\u000a\u000a# Video editing\u000a\u000aCuts reels and highlight clips programmatically.\u000a\u000a## When to use\u000aReach for this when the content flow needs to video editing. It runs on `reelkit`.\u000a\u000a## Status\u000aLive in production. The owning agent runs this today.\u000a'),5);
INSERT INTO skills VALUES('skill-schedule','Cross-post scheduling','Content','Queues and publishes across every connected platform.','postly-publisher','live','["postly"]',unistr('---\u000aname: cross-post-scheduling\u000adescription: Queues and publishes across every connected platform.\u000acategory: Content\u000astatus: live\u000a---\u000a\u000a# Cross-post scheduling\u000a\u000aQueues and publishes across every connected platform.\u000a\u000a## When to use\u000aReach for this when the content flow needs to cross-post scheduling. It runs on `postly`.\u000a\u000a## Status\u000aLive in production. The owning agent runs this today.\u000a'),6);
INSERT INTO skills VALUES('skill-triage','Inbox triage','Ops','Sorts the four inboxes into work / personal / misc and flags priority.','gmail-worker','live','["gmail"]',unistr('---\u000aname: inbox-triage\u000adescription: Sorts the four inboxes into work / personal / misc and flags priority.\u000acategory: Ops\u000astatus: live\u000a---\u000a\u000a# Inbox triage\u000a\u000aSorts the four inboxes into work / personal / misc and flags priority.\u000a\u000a## When to use\u000aReach for this when the ops flow needs to inbox triage. It runs on `gmail`.\u000a\u000a## Status\u000aLive in production. The owning agent runs this today.\u000a'),7);
INSERT INTO skills VALUES('skill-dm','DM management','Ops','Handles Instagram and WhatsApp DMs end to end.','comms-agent','live','["dmflow","whatsapp"]',unistr('---\u000aname: dm-management\u000adescription: Handles Instagram and WhatsApp DMs end to end.\u000acategory: Ops\u000astatus: live\u000a---\u000a\u000a# DM management\u000a\u000aHandles Instagram and WhatsApp DMs end to end.\u000a\u000a## When to use\u000aReach for this when the ops flow needs to dm management. It runs on `dmflow`, `whatsapp`.\u000a\u000a## Status\u000aLive in production. The owning agent runs this today.\u000a'),8);
INSERT INTO skills VALUES('skill-retrieval','Knowledge retrieval','Ops','Hybrid search over Brain so every agent shares one memory.','conductor','live','["optimal-engine"]',unistr('---\u000aname: knowledge-retrieval\u000adescription: Hybrid search over Brain so every agent shares one memory.\u000acategory: Ops\u000astatus: live\u000a---\u000a\u000a# Knowledge retrieval\u000a\u000aHybrid search over Brain so every agent shares one memory.\u000a\u000a## When to use\u000aReach for this when the ops flow needs to knowledge retrieval. It runs on `optimal-engine`.\u000a\u000a## Status\u000aLive in production. The owning agent runs this today.\u000a'),9);
INSERT INTO skills VALUES('skill-reconcile','Payment reconciliation','Ops','Matches processor payouts to clients across Stripe and PayKit.',NULL,'planned','["stripe","paykit"]',unistr('---\u000aname: payment-reconciliation\u000adescription: Matches processor payouts to clients across Stripe and PayKit.\u000acategory: Ops\u000astatus: planned\u000a---\u000a\u000a# Payment reconciliation\u000a\u000aMatches processor payouts to clients across Stripe and PayKit.\u000a\u000a## When to use\u000aReach for this when the ops flow needs to payment reconciliation. It runs on `stripe`, `paykit`.\u000a\u000a## Status\u000aPlanned. Scoped and queued, not yet wired.\u000a'),10);
INSERT INTO skills VALUES('skill-attribution','Revenue attribution','Ops','Ties content and calls to closed revenue via Trakyo.',NULL,'planned','["trakyo","ghl"]',unistr('---\u000aname: revenue-attribution\u000adescription: Ties content and calls to closed revenue via Trakyo.\u000acategory: Ops\u000astatus: planned\u000a---\u000a\u000a# Revenue attribution\u000a\u000aTies content and calls to closed revenue via Trakyo.\u000a\u000a## When to use\u000aReach for this when the ops flow needs to revenue attribution. It runs on `trakyo`, `ghl`.\u000a\u000a## Status\u000aPlanned. Scoped and queued, not yet wired.\u000a'),11);
CREATE INDEX idx_cron_runs_cron ON cron_runs (cron_id, started_at DESC);
CREATE INDEX idx_cron_runs_started ON cron_runs (started_at DESC);
CREATE INDEX idx_agent_runs_started ON agent_runs (started_at DESC);
CREATE INDEX idx_social_dm_messages_ts ON social_dm_messages (ts);
COMMIT;
