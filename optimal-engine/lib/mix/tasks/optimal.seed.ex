defmodule Mix.Tasks.Optimal.Seed do
  @shortdoc "Load a small demo dataset so the graph + RAG + wiki have something to show"

  @moduledoc """
  Seeds the engine with a realistic public demo workspace dataset: one
  organization, product, project, operations, research, package, and signal flow.

  Use this after a fresh `mix compile` to have something to look at when
  you run graph, RAG, wiki, or API commands against an empty engine.

  ## Usage

      mix optimal.seed                    — default ~20 signals
      mix optimal.seed --reset            — wipe demo rows first, re-seed
      mix optimal.seed --tenant sample      — target a specific tenant

  ## What gets created

    * a workspace-like node tree using current semantic node IDs
    * ~20 signals spanning decisions, calls, plans, transcripts
    * ~30 extracted entities with type stamps (person, org, product,
      concept, operation) so the `/api/optimal/graph` co-occurrence
      query produces a navigable graph
    * 1 curated wiki page so `/api/wiki` has at least one example

  Idempotent — re-running inserts fresh rows with unique ids unless
  `--reset` is passed.
  """

  use Mix.Task

  alias OptimalEngine.Store
  alias OptimalEngine.Wiki
  alias OptimalEngine.Wiki.Page

  @demo_tag "demo-seed"

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, _, _} =
      OptionParser.parse(args, strict: [reset: :boolean, tenant: :string])

    tenant = Keyword.get(parsed, :tenant, "default")
    reset? = Keyword.get(parsed, :reset, false)

    if reset?, do: wipe_demo_rows(tenant)

    seed_nodes(tenant)
    signals = seed_signals(tenant)
    seed_entities(signals)
    seed_chunks(signals, tenant)
    seed_classifications(signals, tenant)
    seed_cluster(signals, tenant)
    seed_wiki_page(tenant)
    seed_events(signals, tenant)

    stats = count_rows(tenant)

    IO.puts("""

    Demo seed complete.
      tenant:          #{tenant}
      nodes:           #{stats.nodes}
      signals:         #{stats.contexts}
      entities:        #{stats.entities}
      chunks:          #{stats.chunks}
      classifications: #{stats.classifications}
      intents:         #{stats.intents}
      clusters:        #{stats.clusters}
      events:          #{stats.events}
      wiki pages:      #{stats.wiki}

    Try:
      mix optimal.rag "customer portal pricing"
      mix optimal.wiki list
      (enable the HTTP API and open the desktop at /graph, /workspace, /activity)
    """)
  end

  # ─── schema seeding ─────────────────────────────────────────────────────

  # Nodes with a parent/child hierarchy so the workspace explorer shows
  # real depth. `parent` is a slug; the function resolves it to the parent's
  # id on insert.
  defp seed_nodes(tenant) do
    nodes = [
      {"entity-company", "Example Company", "entity", nil, "internal"},
      {"team", "Team", "team", "entity-company", "internal"},
      {"product-customer-portal", "Customer Portal", "product", "entity-company", "internal"},
      {"product-customer-portal-core", "Customer Portal Core", "project", "product-customer-portal",
       "internal"},
      {"product-customer-portal-services", "Customer Portal Services", "operation",
       "product-customer-portal", "internal"},
      {"product-customer-portal-investors", "Investor Materials", "context",
       "product-customer-portal", "internal"},
      {"project-platform-launch", "Platform Launch", "project", "product-customer-portal",
       "internal"},
      {"project-platform-launch-customer-portal", "Customer Portal Delivery", "project",
       "project-platform-launch", "external"},
      {"operation-delivery", "Delivery Operations", "operation", "entity-company", "internal"},
      {"operation-revenue", "Revenue Operations", "operation", "entity-company", "internal"},
      {"operation-weekly-review", "Weekly Review", "operation", "entity-company", "internal"},
      {"operation-weekly-review-beginner", "Beginner Review Track", "project",
       "operation-weekly-review", "internal"},
      {"operation-weekly-review-advanced", "Advanced Review Track", "project",
       "operation-weekly-review", "internal"},
      {"learning-research-library", "Research Library", "learning", "entity-company", "internal"},
      {"inbox", "Inbox", "context", nil, "internal"}
    ]

    Enum.each(nodes, fn {slug, name, kind, parent_slug, style} ->
      id = "#{tenant}:node:#{slug}"
      parent_id = if parent_slug, do: "#{tenant}:node:#{parent_slug}", else: nil

      Store.raw_query(
        """
        INSERT OR IGNORE INTO nodes (id, tenant_id, slug, name, kind, parent_id, path, style, metadata)
        VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?3, ?7, ?8)
        """,
        [id, tenant, slug, name, kind, parent_id, style, Jason.encode!(%{tag: @demo_tag})]
      )
    end)
  end

  defp seed_signals(tenant) do
    signals = signal_fixtures()

    Enum.map(signals, fn sig ->
      id = "seed-#{sig.slug}-#{System.unique_integer([:positive])}"
      uri = "optimal://nodes/#{sig.node}/signals/#{sig.slug}.md"

      Store.raw_query(
        """
        INSERT INTO contexts (
          id, tenant_id, uri, type, path, title, l0_abstract, l1_overview,
          content, genre, mode, signal_type, format, structure, node,
          sn_ratio, entities, created_at, modified_at, metadata
        )
        VALUES (?1, ?2, ?3, 'signal', ?4, ?5, ?6, ?7, ?8, ?9, 'linguistic',
                'inform', 'markdown', 'prose', ?10, 0.75, '[]',
                datetime('now'), datetime('now'), ?11)
        """,
        [
          id,
          tenant,
          uri,
          uri,
          sig.title,
          sig.abstract,
          sig.abstract,
          sig.content,
          sig.genre,
          sig.node,
          Jason.encode!(%{tag: @demo_tag})
        ]
      )

      Map.put(sig, :id, id)
    end)
  end

  defp seed_entities(signals) do
    Enum.each(signals, fn sig ->
      Enum.each(sig.entities, fn {name, type} ->
        Store.raw_query(
          "INSERT OR IGNORE INTO entities (context_id, name, type) VALUES (?1, ?2, ?3)",
          [sig.id, name, type]
        )
      end)
    end)
  end

  # Hierarchical decomposition at 4 scales — document → section → paragraph
  # → sentence. Viewers can see how the engine breaks a signal down for
  # embedding + retrieval at each granularity.
  defp seed_chunks(signals, tenant) do
    Enum.each(signals, fn sig ->
      sentences =
        sig.content
        |> String.split(~r/\.\s+/, trim: true)
        |> Enum.reject(&(&1 == ""))

      doc_id = "#{sig.id}:doc"
      sec_id = "#{sig.id}:sec:0"

      # Scale 0: document
      insert_chunk(doc_id, tenant, sig.id, nil, "document", sig.content)

      # Scale 1: sections (one section for the demo; parent = document)
      insert_chunk(sec_id, tenant, sig.id, doc_id, "section", sig.content)

      # Scale 2: paragraphs under the section
      Enum.with_index(sentences)
      |> Enum.each(fn {s, i} ->
        par_id = "#{sig.id}:par:#{i}"
        insert_chunk(par_id, tenant, sig.id, sec_id, "paragraph", s <> ".")

        # Scale 3: sentence under each paragraph
        sen_id = "#{sig.id}:sen:#{i}"
        insert_chunk(sen_id, tenant, sig.id, par_id, "sentence", s <> ".")
      end)
    end)
  end

  defp insert_chunk(chunk_id, tenant, signal_id, parent_id, scale, text) do
    Store.raw_query(
      """
      INSERT OR IGNORE INTO chunks (id, tenant_id, signal_id, parent_id, scale, text, length_bytes)
      VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7)
      """,
      [chunk_id, tenant, signal_id, parent_id, scale, text, byte_size(text)]
    )
  end

  # Per-chunk classification + intent rows (schema keys on chunk_id, not
  # context_id). We stamp only the document-scale chunk so the viewer has
  # a signal-level classification to display.
  defp seed_classifications(signals, tenant) do
    Enum.each(signals, fn sig ->
      doc_id = "#{sig.id}:doc"
      intent = intent_for_genre(sig.genre)

      Store.raw_query(
        """
        INSERT OR IGNORE INTO classifications (tenant_id, chunk_id, mode, genre, signal_type, format, structure, sn_ratio, confidence)
        VALUES (?1, ?2, 'linguistic', ?3, 'inform', 'markdown', 'prose', 0.75, 0.85)
        """,
        [tenant, doc_id, sig.genre]
      )

      Store.raw_query(
        """
        INSERT OR IGNORE INTO intents (tenant_id, chunk_id, intent, confidence)
        VALUES (?1, ?2, ?3, 0.80)
        """,
        [tenant, doc_id, Atom.to_string(intent)]
      )
    end)
  end

  defp intent_for_genre("transcript"), do: :capture
  defp intent_for_genre("decision_log"), do: :decide
  defp intent_for_genre("plan"), do: :commit
  defp intent_for_genre("spec"), do: :commit
  defp intent_for_genre("note"), do: :inform
  defp intent_for_genre(_), do: :inform

  # A single theme cluster spanning the customer portal-related signals so the
  # workspace explorer shows cross-signal grouping at the wide-pass layer.
  defp seed_cluster(signals, tenant) do
    theme_signals =
      Enum.filter(signals, fn s -> String.contains?(s.content, "customer portal") end)

    if theme_signals != [] do
      cluster_id = "seed-cluster-customer-portal-#{System.unique_integer([:positive])}"

      Store.raw_query(
        """
        INSERT OR IGNORE INTO clusters (id, tenant_id, theme, intent_dominant, member_count)
        VALUES (?1, ?2, 'Customer Portal delivery + pricing', 'inform', ?3)
        """,
        [cluster_id, tenant, length(theme_signals)]
      )

      Enum.each(theme_signals, fn s ->
        Store.raw_query(
          """
          INSERT OR IGNORE INTO cluster_members (tenant_id, cluster_id, chunk_id, weight)
          VALUES (?1, ?2, ?3, 0.85)
          """,
          [tenant, cluster_id, "#{s.id}:doc"]
        )
      end)
    end
  end

  # Ingest events so the /activity log has real entries to display.
  defp seed_events(signals, tenant) do
    Enum.each(signals, fn sig ->
      Store.raw_query(
        """
        INSERT INTO events (tenant_id, principal, kind, target_uri, metadata)
        VALUES (?1, 'system:demo-seed', 'ingest', ?2, ?3)
        """,
        [
          tenant,
          "optimal://contexts/#{sig.id}",
          Jason.encode!(%{genre: sig.genre, node: sig.node, tag: @demo_tag})
        ]
      )
    end)
  end

  defp seed_wiki_page(tenant) do
    slug = "customer-portal-pricing-decision"

    page = %Page{
      tenant_id: tenant,
      slug: slug,
      audience: "default",
      version: 1,
      frontmatter: %{
        "slug" => slug,
        "title" => "Customer Portal pricing decision",
        "tag" => @demo_tag
      },
      body: """
      ## Summary

      Customer Portal partner pricing set at $2K per seat for Q4 based on the
      customer call {{cite: optimal://nodes/operation-weekly-review/signals/customer-pricing-call.md}}.

      ## Related

      Partner closed the first customer portal deal {{cite: optimal://nodes/project-platform-launch/signals/first-customer-portal-close.md}}.
      """,
      last_curated: DateTime.utc_now() |> DateTime.to_iso8601(),
      curated_by: "deterministic:demo-seed"
    }

    Wiki.put(page)
  end

  # ─── fixtures ───────────────────────────────────────────────────────────
  #
  # Generic Example-Corp demo dataset. Names are placeholders ("Alice",
  # "Bob", …) chosen so the seeder can ship with the public repo without
  # exposing any real person's correspondence. Swap them for your own
  # fixtures when you wire the engine into a real workspace.

  defp signal_fixtures do
    [
      %{
        slug: "customer-pricing-call",
        node: "operation-weekly-review",
        title: "Customer pricing call — Q4",
        genre: "transcript",
        abstract: "Customer wants $2K per seat pricing for the customer portal product.",
        content:
          "Customer lead called to discuss customer portal pricing for Q4. They want $2K per seat. Alice agreed to run the numbers with Bob before replying.",
        entities: [
          {"Alice", "person"},
          {"Bob", "person"},
          {"Customer Lead", "person"},
          {"Customer Portal Product", "product"},
          {"Customer Academy", "operation"}
        ]
      },
      %{
        slug: "first-customer-portal-close",
        node: "project-platform-launch",
        title: "First customer portal deal closed",
        genre: "note",
        abstract: "Partner closed first customer portal deal.",
        content:
          "Partner Dan closed the first customer portal deal with a Sacramento clinic network. Onboarding handoff to Eve; Alice CC'd for context.",
        entities: [
          {"Dan", "person"},
          {"Eve", "person"},
          {"Alice", "person"},
          {"Customer Portal Product", "product"},
          {"Partner Network", "org"}
        ]
      },
      %{
        slug: "platform-microvm-spec",
        node: "product-customer-portal",
        title: "Core platform — microVM isolation",
        genre: "spec",
        abstract: "Per-tenant microVM isolation on the core platform.",
        content:
          "Bob and Carol scoped per-tenant microVM isolation on the core compute plane. Each tenant gets an isolated VM; an in-VM daemon runs inside. Dan owns the proxy layer.",
        entities: [
          {"Bob", "person"},
          {"Carol", "person"},
          {"Dan", "person"},
          {"Core Platform", "product"},
          {"microVM", "concept"},
          {"in-VM daemon", "concept"}
        ]
      },
      %{
        slug: "services-sales-package",
        node: "product-customer-portal",
        title: "Managed services — sales enablement package",
        genre: "plan",
        abstract: "Send Bob the managed-services sales package by Thursday.",
        content:
          "Assemble the managed-services sales package for the partner network — ad scripts, walk-through videos, the offer stack — and hand off by Thursday. Dan provides conversion data.",
        entities: [
          {"Bob", "person"},
          {"Dan", "person"},
          {"Managed Services", "product"},
          {"Partner Network", "org"},
          {"Ad Scripts", "concept"}
        ]
      },
      %{
        slug: "media-stack-handoff",
        node: "learning-research-library",
        title: "Media stack build handoff",
        genre: "decision_log",
        abstract: "Media stack handed off after spec review.",
        content:
          "Decided: Carol takes the media-stack build now that Bob finished the spec review. Eve owns the launch series.",
        entities: [
          {"Carol", "person"},
          {"Bob", "person"},
          {"Eve", "person"},
          {"Media Stack", "product"}
        ]
      },
      %{
        slug: "academy-course-structure",
        node: "operation-weekly-review",
        title: "Academy course structure — beginner + advanced",
        genre: "plan",
        abstract: "Academy splits into beginner + advanced tracks with a premium tier on top.",
        content:
          "Academy structure confirmed: beginner track, advanced track, and a $10K premium tier on top. Alice co-teaches beginner; Bob handles advanced. Dan runs sales.",
        entities: [
          {"Alice", "person"},
          {"Bob", "person"},
          {"Dan", "person"},
          {"Customer Academy", "operation"}
        ]
      },
      %{
        slug: "services-invoice-1234",
        node: "operation-delivery",
        title: "Services division — invoice 1234",
        genre: "note",
        abstract: "Services invoice 1234 to the customer portal partner for October.",
        content:
          "Services Division invoiced the customer portal partner $8,500 for October agency services (invoice 1234). 30-day terms.",
        entities: [
          {"Services Division", "org"},
          {"Customer Portal Product", "product"}
        ]
      },
      %{
        slug: "platform-youtube-pilot",
        node: "product-customer-portal",
        title: "Platform YouTube channel — first pilot",
        genre: "plan",
        abstract: "Frank kicks off the platform YouTube channel with pilot episode.",
        content:
          "Frank is kicking off the platform YouTube channel; pilot episode on compute-layer design. Alice appears as guest.",
        entities: [
          {"Frank", "person"},
          {"Alice", "person"},
          {"Platform YouTube", "concept"}
        ]
      },
      %{
        slug: "technical-partnership",
        node: "product-customer-portal",
        title: "Bob — 50/50 technical partnership",
        genre: "decision_log",
        abstract: "Bob confirmed as 50/50 technical partner on the core platform.",
        content:
          "Decision: Bob is now 50/50 technical partner on the core platform. Licensing, infrastructure, middleware all report to him.",
        entities: [
          {"Bob", "person"},
          {"Alice", "person"},
          {"Core Platform", "product"}
        ]
      },
      %{
        slug: "advisor-consortium-call",
        node: "entity-company",
        title: "Advisor consortium call — strategy framing",
        genre: "transcript",
        abstract: "Advisor walked through the consortium pitch framing.",
        content:
          "Advisor Gina walked through the consortium pitch — strategy framing. Henry joined on recruitment. Alice to follow up with a brief.",
        entities: [
          {"Gina", "person"},
          {"Henry", "person"},
          {"Alice", "person"}
        ]
      },
      %{
        slug: "beginner-track-pilot",
        node: "operation-weekly-review-beginner",
        title: "Beginner track pilot — 12-week curriculum",
        genre: "plan",
        abstract: "Beginner track curriculum pilot ready; first cohort in two weeks.",
        content:
          "Beginner track curriculum locked: 12 weeks, 3 modules, weekly live session. Alice co-teaching with Ivan. Judy runs office hours. First cohort launches in two weeks; Dan will drive enrollment.",
        entities: [
          {"Alice", "person"},
          {"Ivan", "person"},
          {"Judy", "person"},
          {"Dan", "person"},
          {"Customer Academy", "operation"}
        ]
      },
      %{
        slug: "advanced-track-labs",
        node: "operation-weekly-review-advanced",
        title: "Advanced track — hands-on labs specification",
        genre: "spec",
        abstract: "Advanced track labs spec — 8 hands-on exercises covering the full pipeline.",
        content:
          "Advanced track labs: 8 exercises covering ingest → classify → embed → retrieve → curate. Bob authors the retrieval labs. Carol authors the embedding labs. Ivan covers the pipeline glue. Peer review by Alice before ship.",
        entities: [
          {"Bob", "person"},
          {"Carol", "person"},
          {"Ivan", "person"},
          {"Alice", "person"},
          {"Customer Academy", "operation"}
        ]
      },
      %{
        slug: "investor-deck-v2",
        node: "product-customer-portal-investors",
        title: "Investor deck v2 — revenue model update",
        genre: "plan",
        abstract: "Investor deck v2 updates the revenue model and adds partner traction slide.",
        content:
          "Investor deck v2: added partner traction slide (customer portal, 3 signed), revised revenue model (SaaS + services), tightened the ask to $8M Series A. Grace reviewed; Alice to present to Karl next week.",
        entities: [
          {"Grace", "person"},
          {"Alice", "person"},
          {"Karl", "person"},
          {"Investor Deck", "product"}
        ]
      },
      %{
        slug: "customer-portal-partner-onboarding",
        node: "project-platform-launch-customer-portal",
        title: "Customer Portal partner onboarding — runbook",
        genre: "spec",
        abstract: "Runbook for onboarding a new customer portal partner in under 10 days.",
        content:
          "Customer Portal onboarding runbook: day 1 kickoff, day 2-4 data mapping, day 5 staging deploy, day 7 pilot users, day 10 go-live. Dan is the accountable owner. Eve runs the technical handoff. Heidi handles legal.",
        entities: [
          {"Dan", "person"},
          {"Eve", "person"},
          {"Heidi", "person"},
          {"Customer Portal Product", "product"},
          {"Partner Network", "org"}
        ]
      },
      %{
        slug: "media-content-calendar",
        node: "learning-research-library",
        title: "Q1 media content calendar",
        genre: "plan",
        abstract: "Q1 media calendar — 12 episodes, 3 content pillars, weekly cadence.",
        content:
          "Q1 media calendar confirmed: 12 episodes across 3 pillars (platform deep-dives, customer stories, industry analysis). Carol produces, Eve edits, Frank hosts. Target is 2x subscriber growth by end of Q1.",
        entities: [
          {"Carol", "person"},
          {"Eve", "person"},
          {"Frank", "person"},
          {"Media Stack", "product"}
        ]
      },
      %{
        slug: "security-audit-followup",
        node: "product-customer-portal-core",
        title: "Security audit — remediation plan",
        genre: "decision_log",
        abstract: "Security audit closed with 3 mediums + 1 high; remediation plan shipped.",
        content:
          "Security audit results: 3 medium findings (session handling, input validation, rate limiting), 1 high finding (ACL propagation bug). Carol fixes the high this week. Bob owns the mediums across the sprint. Nina verifies before close-out.",
        entities: [
          {"Carol", "person"},
          {"Bob", "person"},
          {"Nina", "person"},
          {"Core Platform", "product"}
        ]
      },
      %{
        slug: "customer-success-q4-review",
        node: "operation-weekly-review",
        title: "Customer success Q4 review",
        genre: "note",
        abstract: "Q4 customer-success metrics — NPS 54, retention 91%, expansion 38%.",
        content:
          "Q4 customer success: NPS 54 (up from 47), retention 91% (flat), expansion revenue 38% (up from 29%). Judy drove the expansion number via quarterly health checks. Alice wants to double expansion next quarter.",
        entities: [
          {"Judy", "person"},
          {"Alice", "person"},
          {"Customer Academy", "operation"}
        ]
      },
      %{
        slug: "data-architecture-review",
        node: "product-customer-portal-core",
        title: "Data-architecture review — new signal types",
        genre: "spec",
        abstract:
          "Review of adding time-series + geo signal types to the data-architecture registry.",
        content:
          "Data-architecture review: adding time-series (IoT telemetry) and geo (polygon / trajectory) as first-class modalities. Carol designs the processor bindings. Bob extends the SQL schema. Ivan writes the tests. Target: ship in 2 sprints.",
        entities: [
          {"Carol", "person"},
          {"Bob", "person"},
          {"Ivan", "person"},
          {"DataArchitecture", "product"},
          {"time-series", "concept"},
          {"geo", "concept"}
        ]
      },
      %{
        slug: "partner-renewal-pipeline",
        node: "project-platform-launch",
        title: "Partner renewal pipeline — Q1",
        genre: "plan",
        abstract: "6 partner renewals in Q1 pipeline — 4 likely, 2 at risk.",
        content:
          "Q1 partner renewals: 6 in flight — 4 likely (Example Clinics, BetaHealth, CareLink, DentAI) and 2 at risk (EpicMed, FastScan). Dan runs point. Eve handles the technical uplift clauses. Heidi redlines contracts.",
        entities: [
          {"Dan", "person"},
          {"Eve", "person"},
          {"Heidi", "person"},
          {"Partner Network", "org"},
          {"Customer Portal Product", "product"}
        ]
      },
      %{
        slug: "office-hours-learnings",
        node: "operation-weekly-review-beginner",
        title: "Office hours — week 4 learnings",
        genre: "note",
        abstract: "Week 4 office hours feedback — two concept gaps identified.",
        content:
          "Office hours week 4: two consistent concept gaps — students struggle with chunk-level ACLs and the S/N ratio. Judy recommends an extra lecture on each. Alice agrees; update coming next week.",
        entities: [
          {"Judy", "person"},
          {"Alice", "person"},
          {"Customer Academy", "operation"},
          {"S/N ratio", "concept"}
        ]
      },
      %{
        slug: "vendor-selection-embeddings",
        node: "product-customer-portal-core",
        title: "Embedding vendor selection — decision",
        genre: "decision_log",
        abstract:
          "Decision: default to local nomic-embed-text + nomic-embed-vision, with optional cloud fallback.",
        content:
          "After the vendor review (local vs. cloud embedders, cost, latency, alignment), decided to default to local nomic-embed-text + nomic-embed-vision. Cloud fallback for enterprise tenants who opt in. Ivan implements the config switch. Carol verifies alignment across the two spaces.",
        entities: [
          {"Ivan", "person"},
          {"Carol", "person"},
          {"Core Platform", "product"},
          {"embeddings", "concept"}
        ]
      },
      %{
        slug: "compliance-soc2-scoping",
        node: "product-customer-portal",
        title: "SOC 2 scoping — Type II path",
        genre: "plan",
        abstract: "SOC 2 Type II scope defined; target audit window starts Q2.",
        content:
          "SOC 2 Type II scope: all tenant data processing + credential storage + audit log. Observation window starts Q2. Heidi owns policy drafts. Carol runs technical controls. Bob fixes the audit trail gaps.",
        entities: [
          {"Heidi", "person"},
          {"Carol", "person"},
          {"Bob", "person"},
          {"SOC 2", "concept"}
        ]
      },
      %{
        slug: "founder-weekly-notes",
        node: "entity-company",
        title: "Founder weekly — focus notes",
        genre: "note",
        abstract: "Weekly focus: close Series A, ship 3D graph, hire head of growth.",
        content:
          "Weekly focus: close Series A (Karl lead), ship 3D graph viewer, hire head of growth. Unblock Carol on the data-architecture review; unblock Dan on the two at-risk renewals. Skip the vendor call — delegate to Ivan.",
        entities: [
          {"Karl", "person"},
          {"Carol", "person"},
          {"Dan", "person"},
          {"Ivan", "person"}
        ]
      }
    ]
  end

  # ─── housekeeping ───────────────────────────────────────────────────────

  defp wipe_demo_rows(tenant) do
    # Downstream tables (chunks, classifications, intents, cluster_members,
    # entities, processor_runs) cascade off the seed context ids. We collect
    # the demo context ids first, then delete from the dependent tables, then
    # finally delete the contexts + nodes + wiki + events themselves.
    {:ok, ids} =
      Store.raw_query(
        "SELECT id FROM contexts WHERE tenant_id = ?1 AND json_extract(metadata, '$.tag') = ?2",
        [tenant, @demo_tag]
      )

    ids = Enum.map(ids || [], fn [id] -> id end)

    for id <- ids do
      Store.raw_query("DELETE FROM entities WHERE context_id = ?1", [id])
      Store.raw_query("DELETE FROM chunks WHERE signal_id = ?1", [id])
      Store.raw_query("DELETE FROM processor_runs WHERE context_id = ?1", [id])
    end

    # Classifications / intents / cluster_members key on chunk_id; once
    # chunks are gone their cascade triggers handle it in schema-declared
    # FKs, but SQLite may not be running with foreign_keys=ON, so be explicit.
    Store.raw_query(
      """
      DELETE FROM classifications WHERE chunk_id LIKE 'seed-%:doc'
      """,
      []
    )

    Store.raw_query("DELETE FROM intents WHERE chunk_id LIKE 'seed-%:doc'", [])
    Store.raw_query("DELETE FROM cluster_members WHERE chunk_id LIKE 'seed-%:doc'", [])
    Store.raw_query("DELETE FROM clusters WHERE id LIKE 'seed-cluster-%'", [])

    Store.raw_query(
      "DELETE FROM contexts WHERE tenant_id = ?1 AND json_extract(metadata, '$.tag') = ?2",
      [tenant, @demo_tag]
    )

    Store.raw_query(
      "DELETE FROM nodes WHERE tenant_id = ?1 AND json_extract(metadata, '$.tag') = ?2",
      [tenant, @demo_tag]
    )

    Store.raw_query(
      "DELETE FROM wiki_pages WHERE tenant_id = ?1 AND json_extract(frontmatter, '$.tag') = ?2",
      [tenant, @demo_tag]
    )

    Store.raw_query(
      "DELETE FROM events WHERE tenant_id = ?1 AND json_extract(metadata, '$.tag') = ?2",
      [tenant, @demo_tag]
    )
  end

  defp count_rows(tenant) do
    %{
      nodes: count(tenant, "SELECT COUNT(*) FROM nodes WHERE tenant_id = ?1"),
      contexts: count(tenant, "SELECT COUNT(*) FROM contexts WHERE tenant_id = ?1"),
      entities:
        count(tenant, """
        SELECT COUNT(*) FROM entities e
        JOIN contexts c ON c.id = e.context_id
        WHERE c.tenant_id = ?1
        """),
      chunks: count(tenant, "SELECT COUNT(*) FROM chunks WHERE tenant_id = ?1"),
      classifications: count(tenant, "SELECT COUNT(*) FROM classifications WHERE tenant_id = ?1"),
      intents: count(tenant, "SELECT COUNT(*) FROM intents WHERE tenant_id = ?1"),
      clusters: count(tenant, "SELECT COUNT(*) FROM clusters WHERE tenant_id = ?1"),
      events: count(tenant, "SELECT COUNT(*) FROM events WHERE tenant_id = ?1"),
      wiki: count(tenant, "SELECT COUNT(*) FROM wiki_pages WHERE tenant_id = ?1")
    }
  end

  defp count(tenant, sql) do
    case Store.raw_query(sql, [tenant]) do
      {:ok, [[n]]} when is_integer(n) -> n
      _ -> 0
    end
  end
end
