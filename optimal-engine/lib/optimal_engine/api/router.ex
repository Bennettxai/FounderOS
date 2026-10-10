defmodule OptimalEngine.API.Router do
  @moduledoc """
  Plug-based HTTP JSON API for Optimal Engine graph, workspace, retrieval, and
  governed memory data.

  Endpoints:
    GET /api/graph           — Full graph (all edges + entity summary + node summary)
    GET /api/graph/hubs      — Hub entities (degree > 2σ)
    GET /api/graph/triangles — Open triangles (synthesis opportunities)
    GET /api/graph/clusters  — Connected components
    GET /api/graph/reflect   — Co-occurrence gaps (?min=2)
    GET /api/node/:node_id   — Subgraph for one node
    GET /api/search          — Full-text search (?q=query&limit=10)
    GET /api/grep            — Hybrid semantic+literal grep with full signal trace (?q=query&workspace=W&intent=I&scale=S&limit=N)
    GET /api/l0              — L0 context cache
    GET /api/health          — Health diagnostics
    GET /api/profile         — 4-tier workspace profile snapshot

  All responses are JSON with `Content-Type: application/json`.
  CORS headers are set on every response (GET + OPTIONS allowed).
  """

  use Plug.Router

  alias OptimalEngine.API.Pagination
  alias OptimalEngine.Graph.Analyzer, as: GraphAnalyzer
  alias OptimalEngine.Health
  alias OptimalEngine.Insight.Health, as: HealthDiagnostics
  alias OptimalEngine.Graph.Reflector, as: Reflector
  alias OptimalEngine.MemoryCore.ContextPackage
  alias OptimalEngine.Profile
  alias OptimalEngine.MemoryCore
  alias OptimalEngine.Retrieval
  alias OptimalEngine.Retrieval.Grep
  alias OptimalEngine.Retrieval.RagStream
  alias OptimalEngine.Retrieval.Receiver
  alias OptimalEngine.Auth.ApiKey
  alias OptimalEngine.Store
  alias OptimalEngine.Telemetry
  alias OptimalEngine.Wiki

  plug(OptimalEngine.API.VersionPlug)
  plug(:cors)
  plug(:match)
  plug(Plug.Parsers, parsers: [:json], json_decoder: Jason, pass: ["application/json"])

  plug(OptimalEngine.API.RateLimitPlug,
    exempt_paths: ["/api/status", "/api/health", "/api/metrics/prometheus"]
  )

  plug(OptimalEngine.API.AuthPlug)

  # Workspace authorization — must run after AuthPlug (needs :current_tenant /
  # :current_api_key) and before :dispatch. Resolves the requested workspace
  # (path/body/query), sets conn.assigns[:workspace_id], and fails closed:
  # 403 on API-key workspace_scope mismatch, 404 on another tenant's workspace.
  plug(OptimalEngine.API.WorkspaceAuthPlug)
  plug(:dispatch)

  # ---------------------------------------------------------------------------
  # CORS preflight
  # ---------------------------------------------------------------------------

  options _ do
    send_resp(conn, 204, "")
  end

  # ---------------------------------------------------------------------------
  # Routes
  # ---------------------------------------------------------------------------

  # Full graph payload — every context as a renderable node + visual edges + entities.
  # Workspace-scoped: ?workspace=<id> (default "default") — never leaks other
  # workspaces' contexts, edges, or entities.
  get "/api/graph" do
    workspace = query_param(conn, "workspace", "default")
    edges = fetch_visual_edges(workspace)
    entities = fetch_entity_summary(workspace)
    nodes = fetch_node_summary(workspace)
    contexts = fetch_all_contexts(workspace)

    json(conn, %{
      workspace_id: workspace,
      contexts: contexts,
      edges: edges,
      entities: format_entities(entities),
      nodes: nodes,
      stats: %{
        context_count: length(contexts),
        edge_count: length(edges),
        entity_count: length(entities)
      }
    })
  end

  get "/api/graph/hubs" do
    workspace = query_param(conn, "workspace", "default")
    {:ok, hubs} = GraphAnalyzer.hubs(workspace_id: workspace)
    json(conn, %{hubs: format_hubs(hubs)})
  end

  get "/api/graph/triangles" do
    workspace = query_param(conn, "workspace", "default")
    limit = conn |> query_param("limit", "20") |> parse_int(20)
    # LLM classification is intentionally opt-in; default is low-latency
    # candidate-only triangles. Use `?classify=true` to enable enrichment.
    skip_llm = not (conn |> query_param("classify", "false") |> parse_bool(false))

    {:ok, triangles} =
      GraphAnalyzer.triangles(limit: limit, workspace_id: workspace, skip_llm: skip_llm)

    json(conn, %{triangles: format_triangles(triangles)})
  end

  get "/api/graph/clusters" do
    workspace = query_param(conn, "workspace", "default")
    {:ok, clusters} = GraphAnalyzer.clusters(workspace_id: workspace)
    json(conn, %{clusters: format_clusters(clusters)})
  end

  get "/api/graph/reflect" do
    workspace = query_param(conn, "workspace", "default")
    min = conn |> query_param("min", "2") |> parse_int(2)
    # LLM classification is intentionally opt-in; default is workspace-safe,
    # low-latency candidate gaps only. Use `?classify=true` to enable.
    skip_llm = not (conn |> query_param("classify", "false") |> parse_bool(false))

    {:ok, gaps} =
      Reflector.reflect(
        min_cooccurrences: min,
        workspace_id: workspace,
        skip_llm: skip_llm
      )

    json(conn, %{gaps: format_gaps(gaps)})
  end

  get "/api/node/:node_id" do
    workspace = query_param(conn, "workspace", "default")
    contexts = fetch_node_contexts(node_id, workspace)
    edges = fetch_node_edges(node_id, workspace)

    json(conn, %{
      node: node_id,
      workspace_id: workspace,
      contexts: contexts,
      edges: format_edges(edges)
    })
  end

  get "/api/search" do
    q = query_param(conn, "q", "")
    workspace = query_param(conn, "workspace", "default")
    {offset, limit} = Pagination.parse(conn, 10)

    {results, total} =
      if q == "" do
        {[], 0}
      else
        # Fetch up to max_limit to get the real total, then slice.
        fetch_limit = min(limit + offset + 200, 200)

        case OptimalEngine.search(q, limit: fetch_limit, workspace_id: workspace) do
          {:ok, contexts} ->
            all = format_search_results(contexts)
            total = length(all)
            page = all |> Enum.drop(offset) |> Enum.take(limit)
            {page, total}

          _ ->
            {[], 0}
        end
      end

    pagination = Pagination.wrap(results, total, offset, limit)

    json(conn, %{
      query: q,
      results: results,
      pagination: pagination.pagination
    })
  end

  # GET /api/grep — hybrid semantic + literal grep over a workspace.
  #
  # Unlike /api/search (which returns context-level metadata), /api/grep
  # returns chunk-level matches with the full signal trace per match:
  # slug, scale, intent, sn_ratio, modality, and a 200-char snippet.
  #
  # Params:
  #   q=<query>       search terms (required)
  #   workspace=<id>  workspace id (default: "default")
  #   intent=<val>    filter by intent (one of 10 canonical values)
  #   scale=<val>     filter by scale: document | section | paragraph | chunk
  #   modality=<val>  filter by modality (e.g. text | image | audio)
  #   limit=<n>       max results (default 25)
  #   literal=true    force literal FTS match — skip semantic/vector search
  #   path=<prefix>   restrict to node slug or slug prefix
  get "/api/grep" do
    q = query_param(conn, "q", "")
    workspace_id = query_param(conn, "workspace", "default")
    intent = query_param(conn, "intent", "")
    scale = query_param(conn, "scale", "")
    modality = query_param(conn, "modality", "")
    {offset, limit} = Pagination.parse(conn, 25)
    literal? = conn |> query_param("literal", "false") |> parse_bool(false)
    path_prefix = query_param(conn, "path", "")

    if q == "" do
      send_resp(conn, 400, Jason.encode!(%{error: "q is required"}))
    else
      grep_opts =
        [
          workspace_id: workspace_id,
          limit: limit + offset,
          literal: literal?
        ]
        |> api_maybe_put(:intent, parse_atom(intent, nil))
        |> api_maybe_put(:scale, parse_atom(scale, nil))
        |> api_maybe_put(:modality, parse_atom(modality, nil))
        |> api_maybe_put(:path_prefix, if(path_prefix == "", do: nil, else: path_prefix))

      case Grep.grep(q, grep_opts) do
        {:ok, all_matches} ->
          total = length(all_matches)
          matches = all_matches |> Enum.drop(offset) |> Enum.take(limit)

          results =
            Enum.map(matches, fn m ->
              %{
                slug: m.slug,
                scale: to_string(m.scale),
                intent: if(m.intent, do: to_string(m.intent), else: nil),
                sn_ratio: m.sn_ratio,
                modality: if(m.modality, do: to_string(m.modality), else: nil),
                snippet: m.snippet,
                score: m.score
              }
            end)

          pagination = Pagination.wrap(results, total, offset, limit)

          json(conn, %{
            query: q,
            workspace_id: workspace_id,
            results: results,
            pagination: pagination.pagination
          })

        {:error, reason} ->
          send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
      end
    end
  end

  get "/api/l0" do
    json(conn, %{l0: OptimalEngine.l0()})
  end

  # GET /api/profile — 4-tier workspace snapshot.
  # Params:
  #   workspace=<id>        workspace id or slug (default: "default")
  #   audience=<tag>        audience tag for wiki variant (default: "default")
  #   bandwidth=l0|l1|full  response density (default: "l1")
  #   node=<slug>           restrict Tier 1/2 to one node (optional)
  get "/api/profile" do
    workspace = query_param(conn, "workspace", "default")
    audience = query_param(conn, "audience", "default")
    bandwidth_str = query_param(conn, "bandwidth", "l1")
    node = query_param(conn, "node", "")
    tenant = query_param(conn, "tenant", "default")

    bandwidth =
      case bandwidth_str do
        "l0" -> :l0
        "full" -> :full
        _ -> :l1
      end

    opts =
      [audience: audience, bandwidth: bandwidth, tenant_id: tenant]
      |> then(fn o -> if node == "", do: o, else: Keyword.put(o, :node_filter, node) end)

    case Profile.get(workspace, opts) do
      {:ok, profile} ->
        json(conn, profile_to_map(profile))

      {:error, :not_found} ->
        send_resp(conn, 404, Jason.encode!(%{error: "workspace not found"}))

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  get "/api/health" do
    # Lightweight by default: liveness + readiness (store/migrations/creds),
    # skipping the slow embedder probe so this endpoint never blocks on Ollama.
    # Pass ?full=true for the heavy 10-check knowledge-base diagnostics.
    case query_param(conn, "full", "false") do
      "true" ->
        workspace = query_param(conn, "workspace", "default")
        {:ok, checks} = HealthDiagnostics.run(workspace_id: workspace)
        json(conn, %{health: format_health(checks)})

      _ ->
        report = Health.ready(skip: [:embedder])

        json(conn, %{
          status: Health.status(),
          live: Health.live?(),
          ok?: report.ok?,
          degraded: report.degraded,
          checks: Map.new(report.checks, fn {k, v} -> {k, inspect(v)} end)
        })
    end
  end

  # ── Phase 12: runtime + retrieval + wiki endpoints ──────────────────────

  # Engine liveness + readiness (Phase 10 Health module).
  get "/api/status" do
    report = Health.ready(skip: [:embedder])

    json(conn, %{
      status: Health.status(),
      ok?: report.ok?,
      checks: Map.new(report.checks, fn {k, v} -> {k, inspect(v)} end),
      degraded: report.degraded,
      api_version: "v1"
    })
  end

  # Prometheus text exposition format — not rate-limited, no auth required.
  # MUST appear before the JSON /api/metrics route (first-match routing).
  get "/api/metrics/prometheus" do
    alias OptimalEngine.Telemetry.Prometheus

    body = Prometheus.render()

    conn
    |> put_resp_content_type("text/plain; version=0.0.4; charset=utf-8")
    |> send_resp(200, body)
  end

  # Runtime metrics snapshot (Phase 10 Telemetry module).
  get "/api/metrics" do
    # `try/rescue` doesn't catch GenServer exits (they aren't exceptions).
    # `try/catch :exit` does. Without this, a downed Telemetry GenServer
    # would crash this request handler instead of returning the fallback.
    snap =
      try do
        Telemetry.snapshot()
      rescue
        _ -> %{counters: %{}, histograms: %{}, uptime_ms: 0}
      catch
        :exit, _ -> %{counters: %{}, histograms: %{}, uptime_ms: 0}
      end

    json(conn, snap)
  end

  # Agent-facing inventory of every logical data store and index.
  get "/api/stores" do
    stores = OptimalEngine.StorageCatalog.list()
    json(conn, %{stores: stores, count: length(stores)})
  end

  get "/api/stores/audit" do
    audit = OptimalEngine.StorageAudit.run()
    conn |> put_status(if(audit.ok, do: 200, else: 503)) |> json(audit)
  end

  # Physical provider inventory. Logical stores above describe ownership;
  # providers describe replaceable deployment technology.
  get "/api/storage/providers" do
    conn = Plug.Conn.fetch_query_params(conn)
    probe? = conn.query_params["probe"] in ["1", "true", "yes"]
    providers = OptimalEngine.Storage.Providers.list(probe: probe?)
    json(conn, %{providers: providers, count: length(providers)})
  end

  get "/api/storage/use-cases" do
    use_cases = OptimalEngine.Storage.Planner.use_cases()
    json(conn, %{use_cases: use_cases, count: length(use_cases)})
  end

  post "/api/storage/plan" do
    body = conn.body_params || %{}

    use_cases =
      case Map.get(body, "use_cases", []) do
        value when is_list(value) -> value
        value when is_binary(value) -> String.split(value, ",", trim: true)
        _ -> []
      end

    case OptimalEngine.Storage.Planner.plan(use_cases,
           probe: Map.get(body, "probe", false) == true
         ) do
      {:ok, plan} ->
        json(conn, plan)

      {:error, {:unknown_use_cases, unknown}} ->
        send_resp(conn, 422, Jason.encode!(%{error: "unknown use cases", unknown: unknown}))
    end
  end

  get "/api/storage/providers/:id" do
    conn = Plug.Conn.fetch_query_params(conn)
    probe? = conn.query_params["probe"] in ["1", "true", "yes"]

    case OptimalEngine.Storage.Providers.get(id, probe: probe?) do
      {:ok, provider} -> json(conn, provider)
      {:error, :not_found} -> send_resp(conn, 404, Jason.encode!(%{error: "provider not found"}))
    end
  end

  get "/api/storage/workspaces/:workspace_id/policy" do
    case OptimalEngine.Storage.PolicyStore.get_policy(workspace_id) do
      {:ok, policy} ->
        json(conn, policy)

      {:error, :not_found} ->
        send_resp(conn, 404, Jason.encode!(%{error: "storage policy not found"}))

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  put "/api/storage/workspaces/:workspace_id/policy" do
    body = conn.body_params || %{}

    use_cases =
      case Map.get(body, "use_cases", []) do
        value when is_list(value) -> value
        value when is_binary(value) -> String.split(value, ",", trim: true)
        _ -> []
      end

    case OptimalEngine.Storage.PolicyStore.put_policy(workspace_id, use_cases,
           provider_overrides: Map.get(body, "provider_overrides", %{}),
           actor_id: conn.assigns[:current_principal_id]
         ) do
      {:ok, policy} -> json(conn, policy)
      {:error, reason} -> send_resp(conn, 422, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  get "/api/storage/workspaces/:workspace_id/mutations" do
    conn = Plug.Conn.fetch_query_params(conn)
    after_sequence = parse_int(conn.query_params["after"], 0)
    limit = parse_int(conn.query_params["limit"], 100)

    case OptimalEngine.Storage.PolicyStore.mutations_after(workspace_id, after_sequence, limit) do
      {:ok, mutations} -> json(conn, %{mutations: mutations, count: length(mutations)})
      {:error, reason} -> send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  post "/api/storage/workspaces/:workspace_id/mutations" do
    body = conn.body_params || %{}

    attrs = %{
      workspace_id: workspace_id,
      device_id: Map.get(body, "device_id"),
      actor_id: conn.assigns[:current_principal_id],
      entity_type: Map.get(body, "entity_type"),
      entity_id: Map.get(body, "entity_id"),
      operation: Map.get(body, "operation"),
      payload: Map.get(body, "payload"),
      idempotency_key: Map.get(body, "idempotency_key"),
      occurred_at: Map.get(body, "occurred_at")
    }

    attrs = attrs |> Enum.reject(fn {_key, value} -> is_nil(value) end) |> Map.new()

    case OptimalEngine.Storage.PolicyStore.append_mutation(attrs) do
      {:ok, mutation} -> conn |> put_status(201) |> json(mutation)
      {:error, reason} -> send_resp(conn, 422, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  post "/api/storage/workspaces/:workspace_id/cursors/:replica_id" do
    sequence = Map.get(conn.body_params || %{}, "last_sequence", 0)

    case OptimalEngine.Storage.PolicyStore.advance_cursor(workspace_id, replica_id, sequence) do
      :ok -> json(conn, %{ok: true, workspace_id: workspace_id, replica_id: replica_id})
      {:error, reason} -> send_resp(conn, 422, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  get "/api/stores/:id/records" do
    conn = Plug.Conn.fetch_query_params(conn)
    workspace_id = conn.query_params["workspace_id"]
    limit = parse_int(conn.query_params["limit"], 25)

    case OptimalEngine.StorageCatalog.records(id, workspace_id, limit) do
      {:ok, records} ->
        json(conn, %{store: id, records: records, count: length(records)})

      {:error, :not_found} ->
        send_resp(conn, 404, Jason.encode!(%{error: "store not found"}))

      {:error, :not_inspectable} ->
        send_resp(conn, 403, Jason.encode!(%{error: "store does not expose records"}))

      {:error, :workspace_required} ->
        send_resp(conn, 400, Jason.encode!(%{error: "workspace_id is required"}))

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: "store inspection failed: #{inspect(reason)}"}))
    end
  end

  get "/api/stores/:id" do
    case OptimalEngine.StorageCatalog.get(id) do
      {:ok, store} -> json(conn, store)
      {:error, :not_found} -> send_resp(conn, 404, Jason.encode!(%{error: "store not found"}))
    end
  end

  # POST /api/rag — end-to-end retrieval for LLM consumption.
  # Body: {"query": "…", "format": "markdown", "audience": "default", "bandwidth": "medium", "workspace": "default"}
  # Optional benchmark/debug controls: skip_intent, skip_wiki, hybrid_limit, memory_limit.
  #
  # Governed recall (additive opt-in): pass "context_package": true to answer
  # through the Memory Core Retrieval Coordinator. The response then carries
  # "source": "context_package" and a "context_package" object (the persisted
  # ContextPackage) alongside the usual envelope.
  #
  # The authorization envelope (actor + allowed_partitions +
  # allowed_security_labels) is derived SERVER-SIDE from the authenticated
  # principal — body-supplied "actor"/"partitions"/"security_labels" are
  # ignored. See rag_context_package_opts/3.
  post "/api/rag" do
    body = conn.body_params || %{}

    case Map.get(body, "query") do
      q when is_binary(q) and q != "" ->
        receiver =
          Receiver.new(%{
            format: parse_atom(body["format"], :markdown),
            bandwidth: parse_atom(body["bandwidth"], :medium),
            audience: body["audience"] || "default"
          })

        workspace_id = body["workspace"] || "default"

        rag_opts =
          [
            receiver: receiver,
            workspace_id: workspace_id,
            skip_intent: parse_bool(body["skip_intent"], false),
            skip_wiki: parse_bool(body["skip_wiki"], false)
          ]
          |> api_maybe_put(:hybrid_limit, parse_int(body["hybrid_limit"], nil))
          |> api_maybe_put(:memory_limit, parse_int(body["memory_limit"], nil))
          |> rag_context_package_opts(body, conn)

        case Retrieval.ask(q, rag_opts) do
          {:ok, result} ->
            json(conn, render_rag_result(result))

          {:error, reason} ->
            send_resp(conn, 502, Jason.encode!(%{error: inspect(reason)}))
        end

      _ ->
        send_resp(conn, 400, Jason.encode!(%{error: "query is required"}))
    end
  end

  # GET /api/rag/stream — SSE streaming variant of POST /api/rag.
  # Emits pipeline-stage events as the retrieval composes the envelope:
  #   intent → wiki_hit (if wiki) → chunks (if hybrid) → composing → envelope → done
  #
  # Params:
  #   query=<text>       required — the question
  #   workspace=<id>     workspace id (default: "default")
  #   audience=<tag>     wiki audience (default: "default")
  #   format=<atom>      response format: markdown | plain | claude | openai (default: markdown)
  #   bandwidth=<atom>   small | medium | large (default: medium)
  get "/api/rag/stream" do
    query = query_param(conn, "query", "")

    if query == "" do
      send_resp(conn, 400, Jason.encode!(%{error: "query is required"}))
    else
      receiver =
        Receiver.new(%{
          format: parse_atom(query_param(conn, "format", "markdown"), :markdown),
          bandwidth: parse_atom(query_param(conn, "bandwidth", "medium"), :medium),
          audience: query_param(conn, "audience", "default")
        })

      workspace_id = query_param(conn, "workspace", "default")

      # skip_intent is a test-only shortcut that bypasses Ollama analysis.
      # It is ignored in production (non-test) builds.
      rag_opts =
        [workspace_id: workspace_id]
        |> then(fn opts ->
          if Mix.env() == :test and query_param(conn, "skip_intent", "") == "true" do
            Keyword.put(opts, :skip_intent, true)
          else
            opts
          end
        end)

      conn = stream_init(conn)
      {:ok, _task} = RagStream.start_link(query, receiver, self(), rag_opts)

      rag_stream_loop(conn)
    end
  end

  # ── Render pipeline ─────────────────────────────────────────────────────
  #
  # 1-source → N-renders: the engine assembles audience-matched context and
  # prompt guidance; the calling agent generates the final HTML via its LLM.

  # GET /api/render/specs — list all render specs for a workspace.
  get "/api/render/specs" do
    workspace = query_param(conn, "workspace", "default")
    specs = OptimalEngine.Render.Registry.list(workspace)
    json(conn, %{specs: Enum.map(specs, &OptimalEngine.Render.Spec.to_map/1)})
  end

  # POST /api/render/context — assemble context for a specific render spec.
  # Body: {"source": "architecture", "spec": "exec", "workspace": "default"}
  # Returns: {spec, context (RAG envelope), prompt_guidance}
  post "/api/render/context" do
    body = conn.body_params || %{}
    source = body["source"]
    spec_name = body["spec"]
    workspace_id = body["workspace"] || "default"

    with source when is_binary(source) and source != "" <- source,
         spec_name when is_binary(spec_name) and spec_name != "" <- spec_name,
         {:ok, spec} <- OptimalEngine.Render.Registry.get(workspace_id, spec_name) do
      receiver =
        Receiver.new(%{
          format: :markdown,
          bandwidth: spec.bandwidth,
          audience: spec.audience,
          genre: spec.genre
        })

      {:ok, rag_result} = Retrieval.ask(source, receiver: receiver, workspace_id: workspace_id)

      json(conn, %{
        spec: OptimalEngine.Render.Spec.to_map(spec),
        context: rag_result.envelope,
        prompt_guidance: OptimalEngine.Render.Spec.prompt_guidance(spec),
        trace: rag_result.trace
      })
    else
      nil ->
        send_resp(conn, 400, Jason.encode!(%{error: "source and spec are required"}))

      {:error, :not_found} ->
        send_resp(conn, 404, Jason.encode!(%{error: "render spec not found"}))

      _ ->
        send_resp(conn, 400, Jason.encode!(%{error: "source and spec are required"}))
    end
  end

  # Wiki listing — GET /api/wiki?tenant=default&workspace=default
  get "/api/wiki" do
    tenant = query_param(conn, "tenant", "default")
    workspace = query_param(conn, "workspace", "default")
    {offset, limit} = Pagination.parse(conn)

    total =
      case Wiki.count(tenant, workspace),
        do: (
          {:ok, n} -> n
          _ -> 0
        )

    case Wiki.list(tenant, workspace, limit: limit, offset: offset) do
      {:ok, pages} ->
        formatted =
          Enum.map(pages, fn p ->
            %{
              slug: p.slug,
              audience: p.audience,
              version: p.version,
              last_curated: p.last_curated,
              curated_by: p.curated_by,
              size_bytes: byte_size(p.body || ""),
              workspace_id: p.workspace_id
            }
          end)

        pagination = Pagination.wrap(formatted, total, offset, limit)

        json(conn, %{
          tenant_id: tenant,
          workspace_id: workspace,
          pages: formatted,
          pagination: pagination.pagination
        })

      _ ->
        json(conn, %{pages: [], pagination: Pagination.wrap([], 0, offset, limit).pagination})
    end
  end

  # GET /api/wiki/stale?workspace=X&tenant=Y
  # Returns stale pages (last_curated too old AND new signals in cited nodes).
  # MUST sit before /:slug — "stale" would otherwise match as a slug.
  get "/api/wiki/stale" do
    workspace = query_param(conn, "workspace", "default")
    tenant = query_param(conn, "tenant", "default")

    pages = OptimalEngine.Wiki.Scheduler.stale_pages(workspace, tenant_id: tenant)

    now = DateTime.utc_now()

    items =
      Enum.map(pages, fn p ->
        days_stale =
          case DateTime.from_iso8601(p.last_curated || "") do
            {:ok, curated_at, _} ->
              diff = DateTime.diff(now, curated_at, :second)
              Float.round(diff / 86_400, 1)

            _ ->
              nil
          end

        %{
          slug: p.slug,
          audience: p.audience,
          workspace_id: p.workspace_id,
          last_curated: p.last_curated,
          days_stale: days_stale
        }
      end)

    json(conn, %{workspace_id: workspace, stale_count: length(items), pages: items})
  end

  # POST /api/wiki/curate?workspace=X&slug=Y&audience=Z
  # Force-curate one page immediately (async). Returns 202 Accepted.
  post "/api/wiki/curate" do
    workspace = query_param(conn, "workspace", "default")
    slug = query_param(conn, "slug", "")
    audience = query_param(conn, "audience", "default")
    tenant = query_param(conn, "tenant", "default")

    cond do
      slug == "" ->
        send_resp(conn, 400, Jason.encode!(%{error: "slug is required"}))

      true ->
        case Wiki.latest(tenant, slug, audience, workspace) do
          {:ok, page} ->
            OptimalEngine.Wiki.Scheduler.force_run(workspace)

            conn
            |> put_resp_content_type("application/json")
            |> send_resp(
              202,
              Jason.encode!(%{
                accepted: true,
                slug: page.slug,
                audience: page.audience,
                workspace_id: workspace,
                message: "re-curation queued"
              })
            )

          {:error, :not_found} ->
            send_resp(conn, 404, Jason.encode!(%{error: "page not found"}))

          {:error, reason} ->
            send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
        end
    end
  end

  # GET /api/wiki/contradictions?workspace=X
  # Active contradiction surfacing events from the last 90 days. MUST sit
  # before the /:slug route below — Plug routers match first-defined-wins,
  # and "contradictions" would otherwise be matched as a slug.
  get "/api/wiki/contradictions" do
    workspace = query_param(conn, "workspace", "default")
    {offset, limit} = Pagination.parse(conn)

    count_sql = """
    SELECT COUNT(*)
    FROM surfacing_events
    WHERE workspace_id = ?1
      AND category = 'contradictions'
      AND datetime(pushed_at) >= datetime('now', '-90 days')
    """

    total =
      case Store.raw_query(count_sql, [workspace]) do
        {:ok, [[n]]} -> n
        _ -> 0
      end

    sql = """
    SELECT envelope_slug, metadata, score, pushed_at
    FROM surfacing_events
    WHERE workspace_id = ?1
      AND category = 'contradictions'
      AND datetime(pushed_at) >= datetime('now', '-90 days')
    ORDER BY pushed_at DESC
    LIMIT ?2 OFFSET ?3
    """

    rows =
      case Store.raw_query(sql, [workspace, limit, offset]) do
        {:ok, r} -> r
        _ -> []
      end

    items =
      Enum.map(rows, fn [slug, metadata_json, score, pushed_at] ->
        meta =
          case Jason.decode(metadata_json || "{}") do
            {:ok, m} -> m
            _ -> %{}
          end

        %{
          page_slug: slug,
          contradictions: Map.get(meta, "contradictions", []),
          entities: Map.get(meta, "entities", []),
          score: score,
          detected_at: pushed_at
        }
      end)

    pagination = Pagination.wrap(items, total, offset, limit)

    json(conn, %{
      workspace_id: workspace,
      contradictions: items,
      count: length(items),
      pagination: pagination.pagination
    })
  end

  # Wiki page render — GET /api/wiki/:slug?workspace=default&audience=default&format=markdown
  get "/api/wiki/:slug" do
    tenant = query_param(conn, "tenant", "default")
    workspace = query_param(conn, "workspace", "default")
    audience = query_param(conn, "audience", "default")
    format = query_param(conn, "format", "markdown") |> parse_atom(:markdown)

    case Wiki.latest(tenant, slug, audience, workspace) do
      {:ok, page} ->
        resolver = fn _d, _opts -> {:ok, "", %{}} end
        {rendered, warnings} = Wiki.render(page, resolver, format: format)

        json(conn, %{
          slug: page.slug,
          audience: page.audience,
          version: page.version,
          workspace_id: page.workspace_id,
          body: rendered,
          warnings: warnings
        })

      _ ->
        send_resp(conn, 404, Jason.encode!(%{error: "page not found"}))
    end
  end

  # ── Phase 12.5: endpoints the ported BusinessOS knowledge components call ─
  #
  # The Svelte components in desktop/src/lib/components/knowledge/ (copied from
  # BusinessOS-5) hit these paths directly. We keep the shape stable so the
  # component code stays unmodified between the two projects.

  # GET /api/optimal/graph — shape expected by OptimalGraphView.svelte:
  #   %{entities: [%{name, type, connections}], edges: [%{source, target, relation, weight}], stats: %{...}}
  # Workspace-scoped via ?workspace=<id> (default "default").
  get "/api/optimal/graph" do
    workspace = query_param(conn, "workspace", "default")
    entities = optimal_entity_summary(workspace)
    edges = optimal_relation_edges(workspace)
    edge_types = Enum.frequencies_by(edges, & &1.relation)

    json(conn, %{
      entities: entities,
      edges: edges,
      stats: %{
        entity_count: length(entities),
        edge_count: length(edges),
        edge_types: edge_types
      }
    })
  end

  # GET /api/optimal/nodes — shape expected by NodeDrillDown level-0 card grid:
  #   %{nodes: [%{slug, name, type, signal_count}]}
  get "/api/optimal/nodes" do
    workspace = query_param(conn, "workspace", "default")
    json(conn, %{nodes: optimal_node_summary(workspace)})
  end

  # GET /api/optimal/nodes/:slug/files — drill-down file tree. Component
  # expects `{files: [...]}`; each entry `%{name, path, is_dir, size, children?}`.
  get "/api/optimal/nodes/:slug/files" do
    workspace = query_param(conn, "workspace", "default")
    json(conn, %{files: optimal_node_files(slug, workspace)})
  end

  # ── Phase 14: workspace explorer endpoints ──────────────────────────────

  # GET /api/workspace — full node forest with parent/child + signal counts.
  # Returned as a flat list; the UI builds the tree client-side using parent_id.
  # Accepts optional ?workspace=<id> to scope to a specific workspace (default: "default").
  get "/api/workspace" do
    workspace = query_param(conn, "workspace", "default")
    json(conn, %{nodes: workspace_nodes(workspace)})
  end

  # GET /api/signals/:id — full signal granularity: chunks (4 scales),
  # entities (by type), classification, intent, clusters, wiki citations.
  # This is the "see everything the engine knows about this one data point"
  # endpoint that the workspace drill-down mounts.
  #
  # Workspace-scoped via ?workspace=<id> (default "default"): a signal id
  # belonging to another workspace returns 404, never its content (IDOR fix).
  get "/api/signals/:id" do
    workspace = query_param(conn, "workspace", "default")

    case signal_detail(id, workspace) do
      nil -> send_resp(conn, 404, Jason.encode!(%{error: "signal not found"}))
      detail -> json(conn, detail)
    end
  end

  # GET /api/activity — reverse-chron events (audit log). Params:
  #   limit=N   (default 50, max 200 via pagination; legacy max 1000 honoured when no offset given)
  #   offset=N  (default 0)
  #   kind=ingest|erasure|retention_action|…  (optional filter)
  #   workspace=<id>  (default "default" — only that workspace's events are returned)
  get "/api/activity" do
    {offset, limit} = Pagination.parse(conn, 100)
    kind = query_param(conn, "kind", "")
    workspace = query_param(conn, "workspace", "default")
    total = count_events(kind, workspace)
    events = recent_events(limit, offset, kind, workspace)
    pagination = Pagination.wrap(events, total, offset, limit)
    json(conn, %{workspace_id: workspace, events: events, pagination: pagination.pagination})
  end

  # GET /api/architectures — every data architecture the engine recognises,
  # built-ins first then any tenant-registered schemas.
  get "/api/architectures" do
    archs =
      Enum.map(OptimalEngine.Architecture.list(), fn a ->
        %{
          id: a.id,
          name: a.name,
          version: a.version,
          description: a.description,
          modality_primary: a.modality_primary,
          granularity: a.granularity,
          field_count: length(a.fields)
        }
      end)

    processors =
      Enum.map(OptimalEngine.Architecture.processor_summary(), fn {id, modality, emits} ->
        %{id: id, modality: modality, emits: emits}
      end)

    json(conn, %{architectures: archs, processors: processors})
  end

  # GET /api/architectures/:id — field-level detail for one architecture.
  get "/api/architectures/:id" do
    case OptimalEngine.Architecture.fetch(id) do
      {:ok, arch} ->
        json(conn, architecture_detail(arch))

      {:error, _} ->
        send_resp(conn, 404, Jason.encode!(%{error: "architecture not found"}))
    end
  end

  # ── Organizations + Workspaces (Phase 1.5) ─────────────────────────────

  # GET /api/organizations — organizations inside the caller's tenant.
  get "/api/organizations" do
    tenant_id = query_param(conn, "tenant", OptimalEngine.Tenancy.Tenant.default_id())
    status = if query_param(conn, "status", "active") == "all", do: :all, else: :active

    case OptimalEngine.Organization.list(tenant_id: tenant_id, status: status) do
      {:ok, organizations} ->
        json(conn, %{
          tenant_id: tenant_id,
          organizations: Enum.map(organizations, &organization_to_map/1)
        })

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # POST /api/organizations — create an organization inside a tenant.
  post "/api/organizations" do
    body = conn.body_params || %{}

    attrs = %{
      slug: Map.get(body, "slug"),
      name: Map.get(body, "name"),
      tenant_id: Map.get(body, "tenant", OptimalEngine.Tenancy.Tenant.default_id()),
      description: Map.get(body, "description"),
      status: parse_organization_status(Map.get(body, "status", "active")),
      metadata: Map.get(body, "metadata", %{})
    }

    case OptimalEngine.Organization.create(attrs) do
      {:ok, organization} ->
        conn
        |> put_resp_content_type("application/json")
        |> send_resp(201, Jason.encode!(organization_to_map(organization)))

      {:error, :missing_required_fields} ->
        send_resp(conn, 400, Jason.encode!(%{error: "slug and name required"}))

      {:error, reason} ->
        send_resp(conn, 422, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # GET /api/workspaces?tenant=default — list workspaces in an org.
  # Status filter via ?status=active|archived|all (default: active).
  get "/api/workspaces" do
    tenant_id = query_param(conn, "tenant", OptimalEngine.Tenancy.Tenant.default_id())
    organization_id = query_param(conn, "organization", nil)
    {offset, limit} = Pagination.parse(conn)

    status =
      conn
      |> query_param("status", "active")
      |> case do
        "all" -> :all
        "archived" -> :archived
        _ -> :active
      end

    total =
      case OptimalEngine.Workspace.count(
             tenant_id: tenant_id,
             organization_id: organization_id,
             status: status
           ) do
        {:ok, n} -> n
        _ -> 0
      end

    case OptimalEngine.Workspace.list(
           tenant_id: tenant_id,
           organization_id: organization_id,
           status: status,
           limit: limit,
           offset: offset
         ) do
      {:ok, list} ->
        ws = Enum.map(list, &workspace_to_map/1)
        pagination = Pagination.wrap(ws, total, offset, limit)

        json(conn, %{
          tenant_id: tenant_id,
          workspaces: ws,
          pagination: pagination.pagination
        })

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # POST /api/workspaces — create a workspace under a tenant.
  # Body: {"slug":"engineering","name":"Engineering Brain","description":"…","tenant":"default"}
  post "/api/workspaces" do
    body = conn.body_params || %{}
    slug = Map.get(body, "slug")
    name = Map.get(body, "name")

    cond do
      not is_binary(slug) or slug == "" ->
        send_resp(conn, 400, Jason.encode!(%{error: "slug required"}))

      not is_binary(name) or name == "" ->
        send_resp(conn, 400, Jason.encode!(%{error: "name required"}))

      true ->
        tenant_id = Map.get(body, "tenant", OptimalEngine.Tenancy.Tenant.default_id())
        organization_id = Map.get(body, "organization", OptimalEngine.Organization.default_id())
        description = Map.get(body, "description")

        case OptimalEngine.WorkspaceTopology.create_workspace(%{
               slug: slug,
               name: name,
               description: description,
               tenant_id: tenant_id,
               organization_id: organization_id
             }) do
          {:ok, ws} ->
            conn
            |> put_resp_content_type("application/json")
            |> send_resp(201, Jason.encode!(workspace_to_map(ws)))

          {:error, reason} ->
            send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
        end
    end
  end

  # GET /api/workspaces/:id — fetch one.
  get "/api/workspaces/:id" do
    case OptimalEngine.Workspace.get(id) do
      {:ok, ws} -> json(conn, workspace_to_map(ws))
      {:error, :not_found} -> send_resp(conn, 404, Jason.encode!(%{error: "workspace not found"}))
      {:error, reason} -> send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # PATCH /api/workspaces/:id — rename / re-describe.
  patch "/api/workspaces/:id" do
    body = conn.body_params || %{}

    attrs =
      %{}
      |> maybe_put(:name, Map.get(body, "name"))
      |> maybe_put(:description, Map.get(body, "description"))

    result =
      with {:ok, workspace} <- OptimalEngine.Workspace.update(id, attrs) do
        case Map.get(body, "organization") do
          organization_id when is_binary(organization_id) ->
            OptimalEngine.Workspace.assign_organization(workspace.id, organization_id)

          _ ->
            {:ok, workspace}
        end
      end

    case result do
      {:ok, ws} -> json(conn, workspace_to_map(ws))
      {:error, :not_found} -> send_resp(conn, 404, Jason.encode!(%{error: "workspace not found"}))
      {:error, reason} -> send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # POST /api/workspaces/:id/archive — soft delete.
  post "/api/workspaces/:id/archive" do
    case OptimalEngine.Workspace.archive(id) do
      :ok -> send_resp(conn, 204, "")
      {:error, reason} -> send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # GET /api/workspaces/:id/config — merged (defaults + on-disk) config.
  # :id is the workspace id (e.g. "default" or "default:engineering").
  # Resolved to slug via Workspace.get/1 before reading the filesystem.
  get "/api/workspaces/:id/config" do
    with {:ok, ws} <- resolve_workspace(id),
         {:ok, cfg} <- OptimalEngine.Workspace.Config.get(ws.slug) do
      json(conn, %{workspace_id: id, config: stringify_keys(cfg)})
    else
      {:error, :not_found} ->
        send_resp(conn, 404, Jason.encode!(%{error: "workspace not found"}))

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # PATCH /api/workspaces/:id/config — deep-merge request body into the
  # on-disk YAML and return the full merged config. Body keys are top-level
  # section names (string keys); nested values replace on a key-by-key basis.
  patch "/api/workspaces/:id/config" do
    body = conn.body_params || %{}

    with {:ok, ws} <- resolve_workspace(id),
         :ok <- OptimalEngine.Workspace.Config.put(ws.slug, body),
         {:ok, cfg} <- OptimalEngine.Workspace.Config.get(ws.slug) do
      json(conn, %{workspace_id: id, config: stringify_keys(cfg)})
    else
      {:error, :not_found} ->
        send_resp(conn, 404, Jason.encode!(%{error: "workspace not found"}))

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # ── Cued recall (Engramme-flavored memory-failure flows) ────────────────
  #
  # Five typed endpoints, one per memory-failure pattern from Kreiman's
  # 2026 "Questions in the Wild" study. Each builds a shaped query string,
  # passes it to the same Retrieval.ask flow as /api/rag, and returns the
  # same envelope. Difference vs /api/rag: the question shape is unambiguous,
  # so IntentAnalyzer decodes intent with maximum confidence and the
  # retrieval boost picks chunks that match the intent type directly.

  # GET /api/recall/actions?actor=X&topic=Y&since=Z&workspace=default
  # Past actions / decisions / commits.
  get "/api/recall/actions" do
    actor = query_param(conn, "actor", "")
    topic = query_param(conn, "topic", "")
    since = query_param(conn, "since", "")
    workspace = query_param(conn, "workspace", "default")

    q =
      ["what actions, decisions, or commitments"]
      |> append_if(actor != "", "by #{actor}")
      |> append_if(topic != "", "about #{topic}")
      |> append_if(since != "", "since #{since}")
      |> Enum.join(" ")

    do_recall(conn, q, body_audience(conn), workspace)
  end

  # GET /api/recall/who?topic=X&role=Y&workspace=default
  # Contact / ownership lookup — "who owns this".
  get "/api/recall/who" do
    topic = query_param(conn, "topic", "")
    role = query_param(conn, "role", "owner")
    workspace = query_param(conn, "workspace", "default")
    q = "who is the #{role} of #{topic}, and how do I reach them"
    do_recall(conn, q, body_audience(conn), workspace)
  end

  # GET /api/recall/when?event=X&workspace=default
  # Schedule / temporal lookup.
  get "/api/recall/when" do
    event = query_param(conn, "event", "")
    workspace = query_param(conn, "workspace", "default")
    q = "when does #{event} happen, and what is the current schedule"
    do_recall(conn, q, body_audience(conn), workspace)
  end

  # GET /api/recall/where?thing=X&workspace=default
  # Object-location lookup — which node/file/path.
  get "/api/recall/where" do
    thing = query_param(conn, "thing", "")
    workspace = query_param(conn, "workspace", "default")
    q = "where is #{thing} kept, owned, or discussed in the workspace"
    do_recall(conn, q, body_audience(conn), workspace)
  end

  # GET /api/recall/owns?actor=X&workspace=default
  # Open-task / current-commitment lookup for an actor.
  get "/api/recall/owns" do
    actor = query_param(conn, "actor", "")
    workspace = query_param(conn, "workspace", "default")
    q = "what is #{actor} currently committed to, and what tasks are open"
    do_recall(conn, q, body_audience(conn), workspace)
  end

  # ── Subscriptions + proactive surfacing (Phase 15) ──────────────────────

  # GET /api/subscriptions?workspace=default
  get "/api/subscriptions" do
    workspace_id = query_param(conn, "workspace", OptimalEngine.Workspace.default_id())
    {offset, limit} = Pagination.parse(conn)

    total =
      case OptimalEngine.Memory.Subscription.count(workspace_id: workspace_id) do
        {:ok, n} -> n
        _ -> 0
      end

    case OptimalEngine.Memory.Subscription.list(
           workspace_id: workspace_id,
           limit: limit,
           offset: offset
         ) do
      {:ok, subs} ->
        formatted = Enum.map(subs, &subscription_to_map/1)
        pagination = Pagination.wrap(formatted, total, offset, limit)

        json(conn, %{
          workspace_id: workspace_id,
          subscriptions: formatted,
          pagination: pagination.pagination
        })

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # POST /api/subscriptions
  # Body: {"workspace": "default", "scope": "topic", "scope_value": "pricing",
  #        "categories": ["recent_actions","ownership"], "principal_id": "alice",
  #        "webhook_url": "https://…", "webhook_secret": "s3cr3t",
  #        "webhook_headers": {"x-custom": "value"}}
  post "/api/subscriptions" do
    body = conn.body_params || %{}

    # Collect webhook fields into metadata (stored as JSON in the metadata column)
    webhook_meta =
      %{}
      |> maybe_put("webhook_url", Map.get(body, "webhook_url"))
      |> maybe_put("webhook_secret", Map.get(body, "webhook_secret"))
      |> maybe_put("webhook_headers", Map.get(body, "webhook_headers"))

    base_metadata = coerce_metadata(Map.get(body, "metadata"))
    merged_metadata = Map.merge(base_metadata, webhook_meta)

    attrs =
      %{
        workspace_id: Map.get(body, "workspace", OptimalEngine.Workspace.default_id()),
        principal_id: Map.get(body, "principal_id"),
        scope: parse_atom(Map.get(body, "scope", "workspace"), :workspace),
        scope_value: Map.get(body, "scope_value"),
        activity: Map.get(body, "activity"),
        metadata: merged_metadata
      }
      |> maybe_put(:categories, parse_categories(Map.get(body, "categories")))

    case OptimalEngine.Memory.Subscription.create(attrs) do
      {:ok, sub} ->
        conn
        |> put_resp_content_type("application/json")
        |> send_resp(201, Jason.encode!(subscription_to_map(sub)))

      {:error, reason} ->
        send_resp(conn, 400, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # POST /api/subscriptions/:id/pause
  post "/api/subscriptions/:id/pause" do
    case OptimalEngine.Memory.Subscription.pause(id) do
      :ok -> send_resp(conn, 204, "")
      other -> send_resp(conn, 500, Jason.encode!(%{error: inspect(other)}))
    end
  end

  # POST /api/subscriptions/:id/resume
  post "/api/subscriptions/:id/resume" do
    case OptimalEngine.Memory.Subscription.resume(id) do
      :ok -> send_resp(conn, 204, "")
      other -> send_resp(conn, 500, Jason.encode!(%{error: inspect(other)}))
    end
  end

  # POST /api/subscriptions/:id/test-webhook
  # Triggers a test delivery to the subscription's configured webhook URL.
  # Returns: 200 + {delivered: true, status: <http_status>}
  #       or 200 + {delivered: false, error: "..."}
  # Returns 404 if the subscription is not found.
  # Returns 400 if the subscription has no webhook_url configured.
  post "/api/subscriptions/:id/test-webhook" do
    alias OptimalEngine.Memory.{Subscription, Webhook}

    subs =
      case Subscription.list(status: :all) do
        {:ok, list} -> list
        _ -> []
      end

    case Enum.find(subs, &(&1.id == id)) do
      nil ->
        send_resp(conn, 404, Jason.encode!(%{error: "subscription not found"}))

      sub ->
        webhook_url = get_in(sub.metadata, ["webhook_url"])

        if is_nil(webhook_url) or webhook_url == "" do
          send_resp(
            conn,
            400,
            Jason.encode!(%{error: "subscription has no webhook_url configured"})
          )
        else
          test_payload = %{
            event: "test",
            subscription_id: sub.id,
            workspace_id: sub.workspace_id,
            message: "OptimalEngine webhook test delivery",
            sent_at: DateTime.utc_now() |> DateTime.to_iso8601()
          }

          response =
            case Webhook.deliver(test_payload, sub.metadata) do
              {:ok, status} -> %{delivered: true, status: status}
              {:error, reason} -> %{delivered: false, error: inspect(reason)}
            end

          json(conn, response)
        end
    end
  end

  # DELETE /api/subscriptions/:id
  delete "/api/subscriptions/:id" do
    case OptimalEngine.Memory.Subscription.delete(id) do
      :ok -> send_resp(conn, 204, "")
      other -> send_resp(conn, 500, Jason.encode!(%{error: inspect(other)}))
    end
  end

  # POST /api/surface/test  body: {"subscription":"sub:...","slug":"customer-portal-pricing-decision"}
  # Convenience: trigger a synthetic push to all listeners of a subscription.
  post "/api/surface/test" do
    body = conn.body_params || %{}
    sub_id = Map.get(body, "subscription")
    slug = Map.get(body, "slug")

    cond do
      not is_binary(sub_id) ->
        send_resp(conn, 400, Jason.encode!(%{error: "subscription required"}))

      not is_binary(slug) ->
        send_resp(conn, 400, Jason.encode!(%{error: "slug required"}))

      true ->
        OptimalEngine.Memory.Surfacer.test_push(sub_id, slug)
        send_resp(conn, 204, "")
    end
  end

  # GET /api/surface/stream?subscription=<id>
  # Server-Sent Events stream. Client sends EventSource and receives
  # newline-delimited JSON envelopes whenever the Surfacer pushes.
  get "/api/surface/stream" do
    sub_id = query_param(conn, "subscription", "")

    if sub_id == "" do
      send_resp(conn, 400, Jason.encode!(%{error: "subscription query param required"}))
    else
      conn = stream_init(conn)
      OptimalEngine.Memory.Surfacer.subscribe(sub_id, self())
      send_sse(conn, "ready", %{subscription: sub_id})
      stream_loop(conn, sub_id)
    end
  end

  # ── Memory primitive (Phase 17) ─────────────────────────────────────────

  # POST /api/memory — create a new memory entry.
  # Body: {content, workspace?, is_static?, audience?, citation_uri?, source_chunk_id?, metadata?}
  # Returns: 201 + memory struct.
  post "/api/memory" do
    body = conn.body_params || %{}
    content = Map.get(body, "content")

    if not (is_binary(content) and content != "") do
      send_resp(conn, 400, Jason.encode!(%{error: "content is required"}))
    else
      workspace_id = Map.get(body, "workspace", "default")

      attrs =
        %{content: content, workspace_id: workspace_id}
        |> maybe_put(:is_static, Map.get(body, "is_static"))
        |> maybe_put(:audience, Map.get(body, "audience"))
        |> maybe_put(:citation_uri, Map.get(body, "citation_uri"))
        |> maybe_put(:source_chunk_id, Map.get(body, "source_chunk_id"))
        |> maybe_put(:metadata, Map.get(body, "metadata"))

      case OptimalEngine.Memory.create(attrs) do
        {:ok, mem} ->
          conn
          |> put_resp_content_type("application/json")
          |> send_resp(201, Jason.encode!(memory_to_map(mem)))

        {:error, reason} ->
          send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
      end
    end
  end

  # POST /api/memory/remember — gated memory intake for agent/app writes.
  # Body: {content, workspace?, audience?, metadata?, force?, gate_threshold?,
  #        salience_floor?, skip_threshold?, update_threshold?}
  # Returns: 200 + {action, memory?, gate, dedup?}.
  post "/api/memory/remember" do
    body = conn.body_params || %{}
    content = Map.get(body, "content")

    if not (is_binary(content) and content != "") do
      send_resp(conn, 400, Jason.encode!(%{error: "content is required"}))
    else
      attrs =
        %{
          content: content,
          workspace_id: Map.get(body, "workspace", Map.get(body, "workspace_id", "default"))
        }
        |> maybe_put(:tenant_id, Map.get(body, "tenant", Map.get(body, "tenant_id")))
        |> maybe_put(:is_static, Map.get(body, "is_static"))
        |> maybe_put(:audience, Map.get(body, "audience"))
        |> maybe_put(:citation_uri, Map.get(body, "citation_uri"))
        |> maybe_put(:source_chunk_id, Map.get(body, "source_chunk_id"))
        |> maybe_put(:metadata, Map.get(body, "metadata"))
        |> maybe_put(:actor_id, Map.get(body, "actor_id"))
        |> maybe_put(:access_policy_id, Map.get(body, "access_policy_id"))
        |> maybe_put(:security_labels, string_list_param(body, "security_labels"))
        |> maybe_put(:partition_ids, string_list_param(body, "partition_ids"))

      opts =
        [
          force: parse_bool(Map.get(body, "force"), false),
          gate_threshold: parse_float(Map.get(body, "gate_threshold")),
          salience_floor: parse_float(Map.get(body, "salience_floor")),
          skip_threshold: parse_float(Map.get(body, "skip_threshold")),
          update_threshold: parse_float(Map.get(body, "update_threshold")),
          versioned_projection: true
        ]
        |> reject_nil_keyword()

      case OptimalEngine.Memory.remember(attrs, opts) do
        {:ok, result} ->
          json(conn, memory_intake_result_to_map(result))

        {:error, reason} ->
          send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
      end
    end
  end

  # POST /api/assets — preserve a multimodal file through the governed asset path.
  #
  # JSON body supports either:
  #   {"path": "/local/file.pdf", "workspace": "default"}
  #   {"filename": "call.wav", "content_base64": "...", "workspace": "default"}
  #
  # Optional adapter execution:
  #   {"adapter_id": "openai_whisper", "command": "printf", "args": ["..."]}
  post "/api/assets" do
    body = conn.body_params || %{}

    case asset_upload_source_path(body) do
      {:ok, source_path, cleanup} ->
        try do
          handle_asset_upload(conn, body, source_path)
        after
          cleanup.()
        end

      {:error, reason} ->
        send_resp(conn, 400, Jason.encode!(%{error: asset_upload_error(reason)}))
    end
  end

  # GET /api/memory/:id/versions — version chain for a memory (BEFORE bare /:id).
  # Returns: {memory_id, root_id, versions: [...]} in chronological order.
  get "/api/memory/:id/versions" do
    case OptimalEngine.Memory.versions(id) do
      {:ok, vs} ->
        root_id = vs |> List.first() |> then(fn v -> v && v.root_memory_id end)

        json(conn, %{
          memory_id: id,
          root_id: root_id,
          versions: Enum.map(vs, &memory_to_map/1)
        })

      {:error, :not_found} ->
        send_resp(conn, 404, Jason.encode!(%{error: "memory not found"}))

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # GET /api/memory/:id/relations — inbound + outbound relation graph (BEFORE bare /:id).
  # Returns: {memory_id, inbound: [...], outbound: [...]}
  get "/api/memory/:id/relations" do
    case OptimalEngine.Memory.relations(id) do
      {:ok, rels} ->
        {inbound, outbound} =
          Enum.split_with(rels, fn r -> to_string(r.target_id) == id end)

        json(conn, %{memory_id: id, inbound: inbound, outbound: outbound})

      {:error, :not_found} ->
        send_resp(conn, 404, Jason.encode!(%{error: "memory not found"}))

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # GET /api/memory/:id — fetch one memory by id.
  get "/api/memory/:id" do
    case OptimalEngine.Memory.get(id) do
      {:ok, mem} -> json(conn, memory_to_map(mem))
      {:error, :not_found} -> send_resp(conn, 404, Jason.encode!(%{error: "memory not found"}))
      {:error, reason} -> send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # GET /api/memory — list memories for a workspace.
  # Params:
  #   workspace            — workspace_id (default "default")
  #   audience             — filter by audience string
  #   include_forgotten    — bool, default false
  #   include_old_versions — bool, default false
  #   limit                — int, default 50
  #   q                    — full-text search (FTS5 MATCH on content, BM25-ranked)
  #                          Empty string treated as absent (no FTS filter applied).
  # GET /api/memory — list memories for a workspace.
  # Params: workspace, audience, include_forgotten (bool), include_old_versions (bool),
  #         limit (int, default 50), offset (int, default 0), q (FTS query)
  get "/api/memory" do
    workspace_id = query_param(conn, "workspace", "default")
    audience = query_param(conn, "audience", "")
    include_forgotten = conn |> query_param("include_forgotten", "false") |> parse_bool(false)
    include_old_versions = conn |> query_param("include_old_versions", "false") |> parse_bool(false)
    {offset, limit} = Pagination.parse(conn)
    q = query_param(conn, "q", "")

    filter_opts =
      [
        workspace_id: workspace_id,
        include_forgotten: include_forgotten,
        include_old_versions: include_old_versions
      ]
      |> api_maybe_put(:audience, if(audience == "", do: nil, else: audience))

    total =
      case OptimalEngine.Memory.count(filter_opts) do
        {:ok, n} -> n
        _ -> 0
      end

    list_opts =
      filter_opts
      |> Keyword.put(:limit, limit)
      |> Keyword.put(:offset, offset)
      |> api_maybe_put(:q, if(q == "", do: nil, else: q))

    memories = OptimalEngine.Memory.list(list_opts)
    formatted = Enum.map(memories, &memory_to_map/1)
    pagination = Pagination.wrap(formatted, total, offset, limit)

    response = %{
      workspace_id: workspace_id,
      count: length(memories),
      memories: formatted,
      pagination: pagination.pagination
    }

    response = if q != "", do: Map.put(response, :query, q), else: response
    json(conn, response)
  end

  # POST /api/memory/:id/forget — mark a memory as forgotten.
  # Body: {reason?, forget_after?}  Returns: 204.
  post "/api/memory/:id/forget" do
    body = conn.body_params || %{}

    opts =
      []
      |> api_maybe_put(:reason, Map.get(body, "reason"))
      |> api_maybe_put(:forget_after, Map.get(body, "forget_after"))

    case OptimalEngine.Memory.forget(id, opts) do
      {:ok, _} -> send_resp(conn, 204, "")
      :ok -> send_resp(conn, 204, "")
      {:error, :not_found} -> send_resp(conn, 404, Jason.encode!(%{error: "memory not found"}))
      {:error, reason} -> send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # POST /api/memory/:id/update — create a new version of a memory.
  # Body: {content, audience?, citation_uri?, metadata?}  Returns: 201 + new memory struct.
  post "/api/memory/:id/update" do
    conn |> memory_mutation_endpoint(id, &OptimalEngine.Memory.update/2)
  end

  # POST /api/memory/:id/extend — create a child memory with :extends relation.
  # Body: {content, audience?, citation_uri?, metadata?}  Returns: 201 + new memory struct.
  post "/api/memory/:id/extend" do
    conn |> memory_mutation_endpoint(id, &OptimalEngine.Memory.extend/2)
  end

  # POST /api/memory/:id/derive — create a derived memory with :derives relation.
  # Body: {content, audience?, citation_uri?, metadata?}  Returns: 201 + new memory struct.
  post "/api/memory/:id/derive" do
    conn |> memory_mutation_endpoint(id, &OptimalEngine.Memory.derive/2)
  end

  # DELETE /api/memory/:id — hard delete. Returns: 204.
  delete "/api/memory/:id" do
    case OptimalEngine.Memory.delete(id) do
      :ok -> send_resp(conn, 204, "")
      {:ok, _} -> send_resp(conn, 204, "")
      {:error, :not_found} -> send_resp(conn, 404, Jason.encode!(%{error: "memory not found"}))
      {:error, reason} -> send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # POST /api/memory/:id/promote — manually promote a memory to a wiki page section.
  # Body: {"slug": "my-page", "audience"?: "default", "workspace"?: "default"}
  # Returns: 200 + updated page, 404 if memory not found.
  post "/api/memory/:id/promote" do
    body = conn.body_params || %{}
    slug = Map.get(body, "slug")

    if not (is_binary(slug) and slug != "") do
      send_resp(conn, 400, Jason.encode!(%{error: "slug is required"}))
    else
      workspace_id = Map.get(body, "workspace", "default")
      audience = Map.get(body, "audience", "default")
      tenant_id = Map.get(body, "tenant", "default")

      opts = [workspace_id: workspace_id, tenant_id: tenant_id, audience: audience]

      case OptimalEngine.Memory.WikiBridge.promote_memory_to_wiki(id, slug, opts) do
        {:ok, page} ->
          json(conn, %{
            slug: page.slug,
            audience: page.audience,
            version: page.version,
            workspace_id: page.workspace_id,
            body: page.body,
            last_curated: page.last_curated,
            curated_by: page.curated_by
          })

        {:error, :not_found} ->
          send_resp(conn, 404, Jason.encode!(%{error: "memory not found"}))

        {:error, reason} ->
          send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
      end
    end
  end

  # ── Memory Core claim governance ──────────────────────────────────────────

  # POST /api/memory-core/context-packages/refresh-stale — batch refresh stale
  # Context Packages for scheduler, app, or agent callers.
  # Body: {workspace?, tenant?, actor_id?, batch_limit?, limit?,
  #        continue_on_error?, allowed_partitions?, allowed_security_labels?}
  post "/api/memory-core/context-packages/refresh-stale" do
    body = conn.body_params || %{}
    workspace_id = Map.get(body, "workspace", Map.get(body, "workspace_id", "default"))

    tenant_id =
      Map.get(
        body,
        "tenant",
        Map.get(body, "tenant_id", conn.assigns[:current_tenant] || "default")
      )

    actor_id = Map.get(body, "actor_id", conn.assigns[:current_principal])

    opts =
      [
        workspace_id: workspace_id,
        tenant_id: tenant_id,
        actor_id: actor_id,
        batch_limit: parse_optional_positive_int(Map.get(body, "batch_limit")),
        limit: parse_optional_positive_int(Map.get(body, "limit")),
        continue_on_error: parse_bool(Map.get(body, "continue_on_error"), true)
      ]
      |> maybe_put_string_list(body, "allowed_partitions", :allowed_partitions)
      |> maybe_put_string_list(body, "allowed_security_labels", :allowed_security_labels)
      |> Enum.reject(fn {_key, value} -> is_nil(value) end)

    case MemoryCore.refresh_stale_context_packages(opts) do
      {:ok, result} ->
        json(conn, %{
          tenant_id: tenant_id,
          workspace_id: workspace_id,
          stale_context_package_ids: result.stale_context_package_ids,
          refreshed_count: length(result.refreshed_context_packages),
          error_count: length(result.errors),
          refreshed_context_packages:
            Enum.map(result.refreshed_context_packages, &context_package_to_map/1),
          errors: Enum.map(result.errors, &stringify_keys/1)
        })

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # POST /api/memory-core/active-pools — open task-scoped working memory.
  # Body: {workspace?, tenant?, task_type?, subject_anchor?, member_links?,
  #        agent_links?, tool_links?, security_labels?, partition_ids?, metadata?}
  post "/api/memory-core/active-pools" do
    body = conn.body_params || %{}

    opts =
      [
        workspace_id: Map.get(body, "workspace", Map.get(body, "workspace_id", "default")),
        tenant_id: Map.get(body, "tenant", Map.get(body, "tenant_id", "default")),
        task_type: Map.get(body, "task_type"),
        subject_anchor: Map.get(body, "subject_anchor"),
        time_mode: Map.get(body, "time_mode", "current_valid"),
        pool_scope: coerce_metadata(Map.get(body, "pool_scope")),
        member_links: Map.get(body, "member_links", []),
        agent_links: Map.get(body, "agent_links", []),
        tool_links: Map.get(body, "tool_links", []),
        membership_policy: coerce_metadata(Map.get(body, "membership_policy")),
        delegation_chain_links: Map.get(body, "delegation_chain_links", []),
        access_policy_id: Map.get(body, "access_policy_id"),
        security_labels: string_list_param(body, "security_labels"),
        partition_ids: string_list_param(body, "partition_ids"),
        policy_version: Map.get(body, "policy_version"),
        metadata: coerce_metadata(Map.get(body, "metadata"))
      ]
      |> reject_nil_keyword()

    case MemoryCore.open_active_pool(opts) do
      {:ok, pool} -> json(conn, %{active_memory_pool: active_pool_to_map(pool)})
      {:error, reason} -> send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # GET /api/memory-core/active-pools/:id — fetch one Active Memory Pool.
  get "/api/memory-core/active-pools/:id" do
    case MemoryCore.get_active_pool(id) do
      {:ok, pool} -> json(conn, %{active_memory_pool: active_pool_to_map(pool)})
      {:error, :not_found} -> send_resp(conn, 404, Jason.encode!(%{error: "active pool not found"}))
      {:error, reason} -> send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # POST /api/memory-core/active-pools/:id/retrieve — retrieve governed context
  # for a pool and load the resulting Context Package into that pool.
  post "/api/memory-core/active-pools/:id/retrieve" do
    body = conn.body_params || %{}
    query = Map.get(body, "query", "")

    if not (is_binary(query) and String.trim(query) != "") do
      send_resp(conn, 400, Jason.encode!(%{error: "query is required"}))
    else
      opts =
        [
          tenant_id: Map.get(body, "tenant", Map.get(body, "tenant_id", "default")),
          workspace_id: Map.get(body, "workspace", Map.get(body, "workspace_id", "default")),
          actor_id: Map.get(body, "actor_id", conn.assigns[:current_principal]),
          active_memory_pool_id: id,
          request_intent: Map.get(body, "request_intent", "recall"),
          time_mode: Map.get(body, "time_mode", "current_valid"),
          detail_depth: Map.get(body, "detail_depth", "summary"),
          limit: parse_optional_positive_int(Map.get(body, "limit"))
        ]
        |> maybe_put_string_list(body, "allowed_partitions", :allowed_partitions)
        |> maybe_put_string_list(body, "allowed_security_labels", :allowed_security_labels)
        |> reject_nil_keyword()

      with {:ok, package} <- MemoryCore.retrieve(query, opts),
           {:ok, pool} <- MemoryCore.load_context_package(id, package) do
        json(conn, %{
          active_memory_pool: active_pool_to_map(pool),
          context_package: context_package_to_map(package)
        })
      else
        {:error, :not_found} ->
          send_resp(conn, 404, Jason.encode!(%{error: "active pool not found"}))

        {:error, reason} ->
          send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
      end
    end
  end

  # POST /api/memory-core/active-pools/:id/refresh-context — refresh stale
  # Context Packages already loaded into the pool.
  post "/api/memory-core/active-pools/:id/refresh-context" do
    body = conn.body_params || %{}

    opts =
      [
        actor_id: Map.get(body, "actor_id", conn.assigns[:current_principal]),
        continue_on_error: parse_bool(Map.get(body, "continue_on_error"), true)
      ]
      |> maybe_put_string_list(body, "allowed_partitions", :allowed_partitions)
      |> maybe_put_string_list(body, "allowed_security_labels", :allowed_security_labels)
      |> reject_nil_keyword()

    case MemoryCore.refresh_pool_context_packages(id, opts) do
      {:ok, result} ->
        json(conn, %{
          active_memory_pool: active_pool_to_map(result.pool),
          refreshed_count: length(result.refreshed_context_packages),
          skipped_context_package_ids: result.skipped_context_package_ids,
          errors: Enum.map(result.errors, &stringify_keys/1),
          refreshed_context_packages:
            Enum.map(result.refreshed_context_packages, &context_package_to_map/1)
        })

      {:error, :not_found} ->
        send_resp(conn, 404, Jason.encode!(%{error: "active pool not found"}))

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # POST /api/memory-core/active-pools/:id/observations — publish a pool-local
  # observation as Source Package + pending Claim.
  post "/api/memory-core/active-pools/:id/observations" do
    body = conn.body_params || %{}
    observation = Map.get(body, "observation", Map.get(body, "text", ""))

    if not (is_binary(observation) and String.trim(observation) != "") do
      send_resp(conn, 400, Jason.encode!(%{error: "observation is required"}))
    else
      opts =
        [
          actor_id: Map.get(body, "actor_id", conn.assigns[:current_principal]),
          source_type: Map.get(body, "source_type", "pool_observation"),
          observation_kind: Map.get(body, "observation_kind", "observation"),
          claim_text: Map.get(body, "claim_text", observation),
          subject_anchor: Map.get(body, "subject_anchor"),
          action_class: Map.get(body, "action_class"),
          object_anchor: Map.get(body, "object_anchor"),
          aggregate_confidence: parse_float(Map.get(body, "aggregate_confidence")),
          aggregate_precision: parse_float(Map.get(body, "aggregate_precision"))
        ]
        |> reject_nil_keyword()

      case MemoryCore.publish_pool_observation(id, observation, opts) do
        {:ok, result} ->
          json(conn, %{
            active_memory_pool: active_pool_to_map(result.pool),
            source_package: source_package_to_map(result.source_package),
            pending_claim: claim_to_map(result.pending_claim)
          })

        {:error, :not_found} ->
          send_resp(conn, 404, Jason.encode!(%{error: "active pool not found"}))

        {:error, reason} ->
          send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
      end
    end
  end

  # POST /api/memory-core/active-pools/:id/close — close/archive a pool.
  post "/api/memory-core/active-pools/:id/close" do
    body = conn.body_params || %{}

    opts =
      [
        reason: Map.get(body, "reason"),
        lifecycle_state: Map.get(body, "lifecycle_state", "closed"),
        archive_state: Map.get(body, "archive_state", "archived")
      ]
      |> reject_nil_keyword()

    case MemoryCore.close_active_pool(id, opts) do
      {:ok, pool} -> json(conn, %{active_memory_pool: active_pool_to_map(pool)})
      {:error, :not_found} -> send_resp(conn, 404, Jason.encode!(%{error: "active pool not found"}))
      {:error, reason} -> send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # GET /api/memory-core/claim-review — review queue summary and claims.
  # Params: workspace, tenant, review_status?, lifecycle_state?, limit?
  get "/api/memory-core/claim-review" do
    workspace_id = query_param(conn, "workspace", "default")
    tenant_id = query_param(conn, "tenant", conn.assigns[:current_tenant] || "default")

    opts =
      [
        workspace_id: workspace_id,
        tenant_id: tenant_id,
        review_status: query_param(conn, "review_status", nil),
        lifecycle_state: query_param(conn, "lifecycle_state", nil),
        limit: parse_optional_positive_int(query_param(conn, "limit", nil))
      ]
      |> Enum.reject(fn {_key, value} -> is_nil(value) end)

    case MemoryCore.claim_review_queue(opts) do
      {:ok, queue} ->
        json(conn, %{
          tenant_id: queue.tenant_id,
          workspace_id: queue.workspace_id,
          count: queue.count,
          review_counts: queue.review_counts,
          lifecycle_counts: queue.lifecycle_counts,
          claims: Enum.map(queue.claims, &claim_to_map/1)
        })

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # GET /api/memory-core/claims — list pending claims for review.
  # Params: workspace, tenant
  get "/api/memory-core/claims" do
    workspace_id = query_param(conn, "workspace", "default")
    tenant_id = query_param(conn, "tenant", conn.assigns[:current_tenant] || "default")

    case MemoryCore.pending_claims(workspace_id: workspace_id, tenant_id: tenant_id) do
      {:ok, claims} ->
        json(conn, %{
          tenant_id: tenant_id,
          workspace_id: workspace_id,
          count: length(claims),
          claims: Enum.map(claims, &claim_to_map/1)
        })

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # GET /api/memory-core/claims/:id — fetch one claim by id.
  # Params: workspace, tenant
  get "/api/memory-core/claims/:id" do
    workspace_id = query_param(conn, "workspace", nil)
    tenant_id = query_param(conn, "tenant", conn.assigns[:current_tenant] || "default")

    case MemoryCore.get_claim(id, workspace_id: workspace_id, tenant_id: tenant_id) do
      {:ok, claim} -> json(conn, claim_to_map(claim))
      {:error, :not_found} -> send_resp(conn, 404, Jason.encode!(%{error: "claim not found"}))
      {:error, reason} -> send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # POST /api/memory-core/claims/:id/reject — reject a pending claim.
  # Body: {workspace?, tenant?, actor_id?}
  post "/api/memory-core/claims/:id/reject" do
    body = conn.body_params || %{}
    workspace_id = Map.get(body, "workspace", Map.get(body, "workspace_id"))
    tenant_id = Map.get(body, "tenant", conn.assigns[:current_tenant] || "default")
    actor_id = Map.get(body, "actor_id", conn.assigns[:current_principal])

    case MemoryCore.reject_claim(id,
           workspace_id: workspace_id,
           tenant_id: tenant_id,
           actor_id: actor_id
         ) do
      {:ok, claim} -> json(conn, %{claim: claim_to_map(claim)})
      {:error, :not_found} -> send_resp(conn, 404, Jason.encode!(%{error: "claim not found"}))
      {:error, reason} -> send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # POST /api/memory-core/claims/:id/promote — accept a claim as a Fact and
  # build a Memory Object.
  # Body: {workspace?, tenant?, actor_id?, fact_text?, summary?, memory_type?,
  #        verification_status?, aggregate_confidence?, aggregate_precision?,
  #        valid_time_start?, valid_time_end?, stale_after?, fact_metadata?,
  #        memory_metadata?, supersedes_fact_id?, allow_stale?,
  #        supersession_reason?}
  post "/api/memory-core/claims/:id/promote" do
    body = conn.body_params || %{}
    workspace_id = Map.get(body, "workspace", Map.get(body, "workspace_id"))
    tenant_id = Map.get(body, "tenant", conn.assigns[:current_tenant] || "default")
    actor_id = Map.get(body, "actor_id", conn.assigns[:current_principal])

    opts =
      [
        workspace_id: workspace_id,
        tenant_id: tenant_id,
        actor_id: actor_id,
        verifier_id: Map.get(body, "verifier_id"),
        fact_text: Map.get(body, "fact_text"),
        fact_type: Map.get(body, "fact_type"),
        verification_status: Map.get(body, "verification_status"),
        aggregate_confidence: Map.get(body, "aggregate_confidence"),
        aggregate_precision: Map.get(body, "aggregate_precision"),
        valid_time_start: Map.get(body, "valid_time_start"),
        valid_time_end: Map.get(body, "valid_time_end"),
        stale_after: Map.get(body, "stale_after"),
        supersedes_fact_id: Map.get(body, "supersedes_fact_id"),
        superseded_valid_time_end: Map.get(body, "superseded_valid_time_end"),
        supersession_reason: Map.get(body, "supersession_reason"),
        allow_stale: truthy?(Map.get(body, "allow_stale")),
        summary: Map.get(body, "summary"),
        memory_type: Map.get(body, "memory_type"),
        salience: Map.get(body, "salience"),
        fact_metadata: coerce_metadata(Map.get(body, "fact_metadata")),
        memory_metadata: coerce_metadata(Map.get(body, "memory_metadata"))
      ]
      |> Enum.reject(fn {_key, value} -> is_nil(value) end)

    case MemoryCore.promote_claim(id, opts) do
      {:ok, result} ->
        json(conn, %{
          claim: claim_to_map(result.claim),
          fact: stringify_keys(result.fact),
          memory_object: stringify_keys(result.memory_object)
        })

      {:error, :not_found} ->
        send_resp(conn, 404, Jason.encode!(%{error: "claim not found"}))

      {:error, :claim_rejected} ->
        send_resp(conn, 409, Jason.encode!(%{error: "claim rejected"}))

      {:error, {:claim_stale, stale_after}} ->
        send_resp(conn, 409, Jason.encode!(%{error: "claim stale", stale_after: stale_after}))

      {:error, {:contradicts_current_facts, fact_ids}} ->
        send_resp(
          conn,
          409,
          Jason.encode!(%{error: "claim contradicts current facts", fact_ids: fact_ids})
        )

      {:error, {:superseded_fact_not_found, fact_id}} ->
        send_resp(
          conn,
          409,
          Jason.encode!(%{error: "superseded fact not found", fact_id: fact_id})
        )

      {:error, {:superseded_fact_not_current, fact_id}} ->
        send_resp(
          conn,
          409,
          Jason.encode!(%{error: "superseded fact is not current", fact_id: fact_id})
        )

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # ── API key management (Phase 18) ──────────────────────────────────────────
  #
  # These endpoints manage API keys for the current tenant. When auth is on
  # (auth_required: true) they require the `admin` scope. In dev (auth_required:
  # false) they are open — same behaviour as every other route.

  # POST /api/auth/keys — mint a new key. Returns id, key, prefix.
  # Body: {name, scopes?, workspace_scope?, expires_at?}
  post "/api/auth/keys" do
    body = conn.body_params || %{}
    name = Map.get(body, "name")
    tenant_id = conn.assigns[:current_tenant] || "default"

    if not (is_binary(name) and name != "") do
      send_resp(conn, 400, Jason.encode!(%{error: "name is required"}))
    else
      attrs =
        %{tenant_id: tenant_id, name: name}
        |> maybe_put(:scopes, Map.get(body, "scopes"))
        |> maybe_put(:workspace_scope, Map.get(body, "workspace_scope"))
        |> maybe_put(:expires_at, Map.get(body, "expires_at"))
        |> maybe_put(:principal_id, Map.get(body, "principal_id"))

      case ApiKey.mint(attrs) do
        {:ok, %{id: id, key: key, secret: _secret}} ->
          conn
          |> put_resp_content_type("application/json")
          |> send_resp(201, Jason.encode!(%{id: id, key: key, prefix: String.slice(key, 0, 8)}))

        {:error, reason} ->
          send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
      end
    end
  end

  # GET /api/auth/keys — list non-revoked keys for current tenant (no secrets).
  get "/api/auth/keys" do
    tenant_id = conn.assigns[:current_tenant] || "default"
    {offset, limit} = Pagination.parse(conn)

    total =
      case ApiKey.count(tenant_id) do
        {:ok, n} -> n
        _ -> 0
      end

    case ApiKey.list(tenant_id, limit: limit, offset: offset) do
      {:ok, keys} ->
        formatted = Enum.map(keys, &api_key_to_map/1)
        pagination = Pagination.wrap(formatted, total, offset, limit)

        json(conn, %{
          tenant_id: tenant_id,
          keys: formatted,
          pagination: pagination.pagination
        })

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # POST /api/auth/keys/:id/revoke — soft revoke a key.
  post "/api/auth/keys/:id/revoke" do
    case ApiKey.revoke(id) do
      :ok -> send_resp(conn, 204, "")
      {:error, reason} -> send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # DELETE /api/auth/keys/:id — hard delete a key.
  delete "/api/auth/keys/:id" do
    case ApiKey.delete(id) do
      :ok -> send_resp(conn, 204, "")
      {:error, reason} -> send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # ── Batch import / export (Phase 19) ───────────────────────────────────────
  #
  # Designed for workspace migration and API-level backup. All import endpoints
  # accumulate errors per-item — a single bad item never aborts the batch.
  #
  # POST /api/batch/import/signals
  # Body: {workspace: "default", signals: [{content, title?, genre?, node?, entities?}]}
  # Returns: {imported: N, skipped: M, errors: K}
  post "/api/batch/import/signals" do
    body = conn.body_params || %{}
    workspace_id = Map.get(body, "workspace", "default")
    items = Map.get(body, "signals", [])

    if not is_list(items) do
      send_resp(conn, 400, Jason.encode!(%{error: "signals must be a list"}))
    else
      case OptimalEngine.Batch.import_signals(items, workspace_id: workspace_id) do
        {:ok, summary} -> json(conn, summary)
      end
    end
  end

  # POST /api/batch/import/memories
  # Body: {workspace: "default", memories: [{content, is_static?, audience?, ...}]}
  # Returns: {imported: N, skipped: M, errors: K}
  post "/api/batch/import/memories" do
    body = conn.body_params || %{}
    workspace_id = Map.get(body, "workspace", "default")
    items = Map.get(body, "memories", [])

    if not is_list(items) do
      send_resp(conn, 400, Jason.encode!(%{error: "memories must be a list"}))
    else
      case OptimalEngine.Batch.import_memories(items, workspace_id: workspace_id) do
        {:ok, summary} -> json(conn, summary)
      end
    end
  end

  # GET /api/batch/export/signals?workspace=default
  # Returns: {workspace_id, signals: [...]}
  get "/api/batch/export/signals" do
    workspace_id = query_param(conn, "workspace", "default")

    case OptimalEngine.Batch.export_signals(workspace_id: workspace_id) do
      {:ok, signals} -> json(conn, %{workspace_id: workspace_id, signals: signals})
    end
  end

  # GET /api/batch/export/memories?workspace=default
  # Returns: {workspace_id, memories: [...]}
  get "/api/batch/export/memories" do
    workspace_id = query_param(conn, "workspace", "default")

    case OptimalEngine.Batch.export_memories(workspace_id: workspace_id) do
      {:ok, memories} -> json(conn, %{workspace_id: workspace_id, memories: memories})
    end
  end

  # GET /api/batch/export/workspace?workspace=default
  # Returns full snapshot: {workspace_id, signals, memories, wiki, config}
  get "/api/batch/export/workspace" do
    workspace_id = query_param(conn, "workspace", "default")
    tenant_id = query_param(conn, "tenant", OptimalEngine.Tenancy.Tenant.default_id())

    case OptimalEngine.Batch.export_workspace(workspace_id, tenant_id: tenant_id) do
      {:ok, snapshot} -> json(conn, snapshot)
    end
  end

  # POST /api/backup — run an online SQLite backup and return the file info.
  # Body: {path?: "/tmp/backup.db"} — path defaults to a timestamped temp file.
  # Returns: {path, size_bytes, rows_backed_up, duration_ms}
  post "/api/backup" do
    body = conn.body_params || %{}

    target_path =
      Map.get(body, "path") ||
        Path.join(System.tmp_dir!(), "optimal-backup-#{System.os_time(:second)}.db")

    case OptimalEngine.Backup.create(target_path) do
      {:ok, info} ->
        json(conn, %{
          id: info.id,
          path: info.target,
          size_bytes: info.size_bytes,
          rows_backed_up: info.rows_backed_up,
          duration_ms: info.duration_ms,
          checksum_sha256: info.checksum_sha256,
          integrity_status: info.integrity_status
        })

      {:error, {:target_exists, path}} ->
        send_resp(conn, 409, Jason.encode!(%{error: "backup target already exists: #{path}"}))

      {:error, {:parent_missing, dir}} ->
        send_resp(conn, 400, Jason.encode!(%{error: "parent directory missing: #{dir}"}))

      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  # POST /api/assemble — tiered context assembly with MCTS budget-aware selection.
  #
  # Builds L0-L3 tiered context for a query via ContextAssembler.assemble/2,
  # which drives the MCTS selection loop for L2 candidate ranking. Returns the
  # assembled tiers plus MCTS selection metadata (selected candidate ids, scores,
  # token budget used) so callers can inspect how the budget was spent.
  #
  # Body: {"query": "...", "workspace"?: "default", "tier_budgets"?: {l0, l1, l2}}
  # Returns: {l0, l1, l2, l3, total_tokens, sources, mcts_metadata, workspace_id}
  post "/api/nodes" do
    body = conn.body_params || %{}
    name = Map.get(body, "name", "")
    kind = Map.get(body, "kind", "")

    cond do
      not (is_binary(name) and String.trim(name) != "") ->
        send_resp(conn, 400, Jason.encode!(%{error: "name is required"}))

      not (is_binary(kind) and kind != "") ->
        send_resp(conn, 400, Jason.encode!(%{error: "kind is required"}))

      true ->
        slug =
          Map.get(body, "slug") ||
            name
            |> String.downcase()
            |> String.replace(~r/[^a-z0-9]+/, "-")
            |> String.trim("-")

        attrs = %{
          slug: slug,
          name: name,
          kind: String.to_atom(kind),
          workspace_id: Map.get(body, "workspace", "default"),
          description: Map.get(body, "description")
        }

        case OptimalEngine.Topology.Node.upsert(attrs) do
          {:ok, node} ->
            send_resp(
              conn,
              200,
              Jason.encode!(%{ok: true, id: node.id, slug: node.slug, kind: kind})
            )

          {:error, reason} ->
            send_resp(conn, 422, Jason.encode!(%{ok: false, error: inspect(reason)}))
        end
    end
  end

  post "/api/ingest" do
    body = conn.body_params || %{}
    text = Map.get(body, "text", "")

    if not (is_binary(text) and String.trim(text) != "") do
      send_resp(conn, 400, Jason.encode!(%{error: "text is required"}))
    else
      intake_opts =
        []
        |> then(fn acc -> if(v = body["genre"], do: Keyword.put(acc, :genre, v), else: acc) end)
        |> then(fn acc -> if(v = body["title"], do: Keyword.put(acc, :title, v), else: acc) end)
        |> then(fn acc -> if(v = body["node"], do: Keyword.put(acc, :node, v), else: acc) end)
        |> then(fn acc ->
          if(v = body["workspace"], do: Keyword.put(acc, :workspace_id, v), else: acc)
        end)
        |> then(fn acc ->
          if(body["extract_claims"] == true, do: Keyword.put(acc, :extract_claims, true), else: acc)
        end)

      case OptimalEngine.Pipeline.Intake.process(text, intake_opts) do
        {:ok, result} ->
          sig = result.signal

          send_resp(
            conn,
            200,
            Jason.encode!(%{
              ok: true,
              signal_id: Map.get(result, :context_id) || Map.get(sig, :id),
              genre: sig.genre,
              type: sig.type,
              entities: sig.entities,
              source_package_id: result[:source_package] && result.source_package.id
            })
          )

        {:error, reason} ->
          send_resp(conn, 422, Jason.encode!(%{ok: false, error: inspect(reason)}))
      end
    end
  end

  post "/api/assemble" do
    body = conn.body_params || %{}
    query = Map.get(body, "query", "")

    if not (is_binary(query) and String.trim(query) != "") do
      send_resp(conn, 400, Jason.encode!(%{error: "query is required"}))
    else
      workspace_id = Map.get(body, "workspace", "default")

      tier_budgets =
        case Map.get(body, "tier_budgets") do
          %{"l0" => l0, "l1" => l1, "l2" => l2} ->
            [
              tier_budgets: %{
                l0: parse_int(l0, 3_000),
                l1: parse_int(l1, 10_000),
                l2: parse_int(l2, 50_000)
              }
            ]

          _ ->
            []
        end

      assemble_opts =
        tier_budgets ++
          [
            workspace_id: workspace_id
          ]

      {:ok, assembled} =
        OptimalEngine.Retrieval.ContextAssembler.assemble(query, assemble_opts)

      mcts_meta =
        assembled.search_scores
        |> Enum.map(fn s ->
          %{
            id: s[:id],
            title: s[:title],
            score: s[:score],
            uri: s[:uri],
            node: s[:node]
          }
        end)

      json(conn, %{
        query: query,
        workspace_id: workspace_id,
        l0: assembled.l0,
        l1: assembled.l1,
        l2: assembled.l2,
        l3: "",
        total_tokens: assembled.total_tokens,
        sources: assembled.sources,
        mcts_metadata: %{
          candidate_count: length(assembled.search_scores),
          selected_sources: mcts_meta,
          mcts_enabled: OptimalEngine.Retrieval.MCTS.enabled?()
        }
      })
    end
  end

  match _ do
    send_resp(conn, 404, Jason.encode!(%{error: "not found"}))
  end

  # ---------------------------------------------------------------------------
  # Helpers: response
  # ---------------------------------------------------------------------------

  defp json(conn, data) do
    body = Jason.encode!(data)

    conn
    |> put_resp_content_type("application/json")
    |> send_resp(conn.status || 200, body)
  end

  defp cors(conn, _opts) do
    conn
    |> put_resp_header("access-control-allow-origin", "*")
    |> put_resp_header("access-control-allow-methods", "GET, POST, PATCH, DELETE, OPTIONS")
    # Allow the auth + workspace headers BusinessOS (and any app) sends so a
    # browser/desktop client can connect this engine as its second brain.
    # Without `authorization` here the browser blocks every authenticated
    # cross-origin request to the engine.
    |> put_resp_header(
      "access-control-allow-headers",
      "content-type, authorization, x-api-key, api-key, x-workspace-id"
    )
    |> put_resp_header("access-control-max-age", "86400")
    |> put_resp_header("x-api-version", "v1")
  end

  defp workspace_to_map(ws) do
    %{
      id: ws.id,
      tenant_id: ws.tenant_id,
      organization_id: ws.organization_id,
      slug: ws.slug,
      name: ws.name,
      description: ws.description,
      status: Atom.to_string(ws.status),
      created_at: ws.created_at,
      archived_at: ws.archived_at,
      metadata: ws.metadata
    }
  end

  defp organization_to_map(organization) do
    %{
      id: organization.id,
      tenant_id: organization.tenant_id,
      slug: organization.slug,
      name: organization.name,
      description: organization.description,
      status: Atom.to_string(organization.status),
      created_at: organization.created_at,
      archived_at: organization.archived_at,
      metadata: organization.metadata
    }
  end

  defp parse_organization_status("dormant"), do: :dormant
  defp parse_organization_status("archived"), do: :archived
  defp parse_organization_status(_), do: :active

  defp profile_to_map(%Profile{} = p) do
    %{
      workspace_id: p.workspace_id,
      tenant_id: p.tenant_id,
      audience: p.audience,
      static: p.static,
      dynamic: p.dynamic,
      curated: p.curated,
      activity: p.activity,
      entities: p.entities,
      generated_at: p.generated_at
    }
  end

  defp memory_to_map(mem) do
    %{
      id: mem.id,
      tenant_id: mem.tenant_id,
      workspace_id: mem.workspace_id,
      content: mem.content,
      is_static: mem.is_static,
      is_forgotten: mem.is_forgotten,
      forget_after: mem.forget_after,
      forget_reason: mem.forget_reason,
      version: mem.version,
      parent_memory_id: mem.parent_memory_id,
      root_memory_id: mem.root_memory_id,
      is_latest: mem.is_latest,
      citation_uri: mem.citation_uri,
      source_chunk_id: mem.source_chunk_id,
      audience: mem.audience,
      metadata: mem.metadata,
      created_at: mem.created_at,
      updated_at: mem.updated_at,
      was_existing: Map.get(mem, :was_existing, false)
    }
  end

  defp memory_intake_result_to_map(result) do
    %{
      action: Atom.to_string(result.action),
      memory: result.memory && memory_to_map(result.memory),
      gate: memory_intake_gate_to_map(result.gate),
      dedup: memory_intake_dedup_to_map(Map.get(result, :dedup))
    }
  end

  defp memory_intake_gate_to_map(nil), do: nil

  defp memory_intake_gate_to_map(gate) do
    %{
      should_encode: gate.should_encode,
      score: gate.score,
      novelty: gate.novelty,
      salience: gate.salience,
      prediction_error: gate.prediction_error,
      reason: gate.reason,
      similar_memory_id: gate.similar_memory_id
    }
  end

  defp memory_intake_dedup_to_map(nil), do: nil

  defp memory_intake_dedup_to_map(dedup) do
    %{
      action: Atom.to_string(dedup.action),
      reason: dedup.reason,
      similarity: dedup.similarity,
      existing_memory_id: dedup.existing && dedup.existing.id
    }
  end

  defp claim_to_map(claim) when is_map(claim) do
    if Map.has_key?(claim, :__struct__) do
      claim
      |> Map.from_struct()
      |> stringify_keys()
    else
      stringify_keys(claim)
    end
  end

  defp parse_optional_positive_int(nil), do: nil
  defp parse_optional_positive_int(""), do: nil

  defp parse_optional_positive_int(value) when is_binary(value) do
    case Integer.parse(value) do
      {int, ""} when int > 0 -> int
      _ -> nil
    end
  end

  defp parse_optional_positive_int(value) when is_integer(value) and value > 0, do: value
  defp parse_optional_positive_int(_value), do: nil

  defp truthy?(true), do: true
  defp truthy?("true"), do: true
  defp truthy?("1"), do: true
  defp truthy?(1), do: true
  defp truthy?(_value), do: nil

  defp asset_upload_source_path(body) do
    path = Map.get(body, "path")
    content_base64 = Map.get(body, "content_base64")

    cond do
      is_binary(path) and String.trim(path) != "" ->
        if File.exists?(path) do
          {:ok, path, fn -> :ok end}
        else
          {:error, :path_not_found}
        end

      is_binary(content_base64) and String.trim(content_base64) != "" ->
        case Base.decode64(content_base64) do
          {:ok, bytes} ->
            with {:ok, path} <-
                   write_upload_tempfile(bytes, Map.get(body, "filename", "upload.bin")) do
              {:ok, path, fn -> File.rm(path) end}
            end

          :error ->
            {:error, :invalid_base64}
        end

      true ->
        {:error, :asset_source_required}
    end
  end

  defp write_upload_tempfile(bytes, filename) when is_binary(bytes) do
    safe_name =
      filename
      |> to_string()
      |> Path.basename()
      |> String.replace(~r/[^A-Za-z0-9._-]/, "_")
      |> case do
        "" -> "upload.bin"
        value -> value
      end

    path =
      Path.join(
        System.tmp_dir!(),
        "optimal-api-asset-#{System.unique_integer([:positive])}-#{safe_name}"
      )

    case File.write(path, bytes) do
      :ok -> {:ok, path}
      {:error, reason} -> {:error, {:write_failed, reason}}
    end
  end

  defp handle_asset_upload(conn, body, source_path) do
    workspace_id = Map.get(body, "workspace", Map.get(body, "workspace_id", "default"))
    tenant_id = Map.get(body, "tenant", Map.get(body, "tenant_id", "default"))
    actor_id = Map.get(body, "actor_id", "api:asset-upload")

    store_opts =
      [
        tenant_id: tenant_id,
        workspace_id: workspace_id,
        actor_id: actor_id,
        content_type: Map.get(body, "content_type"),
        modality: parse_atom(Map.get(body, "modality"), nil),
        trust_label: Map.get(body, "trust_label", "unreviewed"),
        retention_class: Map.get(body, "retention_class", "standard"),
        access_policy_id: Map.get(body, "access_policy_id"),
        security_labels: string_list_param(body, "security_labels"),
        partition_ids: string_list_param(body, "partition_ids"),
        metadata:
          Map.merge(coerce_metadata(Map.get(body, "metadata")), %{
            "api_upload" => true,
            "filename" => Map.get(body, "filename") || Path.basename(source_path)
          })
      ]
      |> reject_nil_keyword()

    with {:ok, %{asset: asset, source_package: source_package}} <-
           MemoryCore.store_asset_file(source_path, store_opts),
         {:ok, adapter_result} <- maybe_run_upload_adapter(asset, body, store_opts) do
      response =
        %{
          asset: asset_to_map(asset),
          source_package: source_package_to_map(source_package),
          adapter_run: Map.get(adapter_result, :adapter_run),
          asset_extractions: Map.get(adapter_result, :asset_extractions, [])
        }
        |> stringify_keys()

      conn
      |> put_resp_content_type("application/json")
      |> send_resp(201, Jason.encode!(response))
    else
      {:error, reason} ->
        send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
    end
  end

  defp maybe_run_upload_adapter(asset, body, store_opts) do
    case Map.get(body, "adapter_id") do
      adapter_id when is_binary(adapter_id) and adapter_id != "" ->
        adapter_opts =
          store_opts
          |> Keyword.merge(
            command: Map.get(body, "command"),
            args: Map.get(body, "args", []),
            adapter_role: Map.get(body, "adapter_role"),
            auto_extract: parse_bool(Map.get(body, "auto_extract"), true),
            confidence: Map.get(body, "confidence"),
            precision: Map.get(body, "precision"),
            model_id: Map.get(body, "model_id"),
            model_version: Map.get(body, "model_version"),
            content_text: Map.get(body, "content_text"),
            content_ref: Map.get(body, "content_ref"),
            embedding_ref: Map.get(body, "embedding_ref"),
            embedding_model_id: Map.get(body, "embedding_model_id"),
            embedding_model_version: Map.get(body, "embedding_model_version"),
            embedding_dim: Map.get(body, "embedding_dim"),
            embedding_space: Map.get(body, "embedding_space"),
            target_ref: Map.get(body, "target_ref"),
            extraction_type: Map.get(body, "extraction_type"),
            extraction_metadata: coerce_metadata(Map.get(body, "extraction_metadata"))
          )
          |> reject_nil_keyword()

        with {:ok, run} <- MemoryCore.run_asset_adapter(asset.id, adapter_id, adapter_opts),
             {:ok, extractions} <-
               MemoryCore.list_asset_extractions(asset.id,
                 tenant_id: asset.tenant_id,
                 workspace_id: asset.workspace_id
               ) do
          {:ok, %{adapter_run: run, asset_extractions: extractions}}
        end

      _ ->
        {:ok, %{adapter_run: nil, asset_extractions: []}}
    end
  end

  defp asset_to_map(asset), do: clean_map(asset)
  defp source_package_to_map(source_package), do: clean_map(source_package)
  defp active_pool_to_map(pool), do: pool |> clean_map() |> stringify_keys()

  defp clean_map(value) when is_map(value) do
    value
    |> maybe_from_struct()
    |> Map.drop([:__struct__])
  end

  defp maybe_from_struct(%{__struct__: _} = value), do: Map.from_struct(value)
  defp maybe_from_struct(value), do: value

  defp string_list_param(body, key) do
    case Map.get(body, key, []) do
      value when is_list(value) -> Enum.map(value, &to_string/1)
      value when is_binary(value) and value != "" -> [value]
      _ -> []
    end
  end

  defp maybe_put_string_list(keyword, body, body_key, opt_key) do
    if Map.has_key?(body, body_key) do
      Keyword.put(keyword, opt_key, string_list_param(body, body_key))
    else
      keyword
    end
  end

  defp reject_nil_keyword(keyword) do
    Enum.reject(keyword, fn {_key, value} -> is_nil(value) end)
  end

  defp asset_upload_error(:asset_source_required), do: "path or content_base64 is required"
  defp asset_upload_error(:path_not_found), do: "path not found"
  defp asset_upload_error(:invalid_base64), do: "content_base64 is invalid"

  defp asset_upload_error({:write_failed, reason}),
    do: "could not write upload temp file: #{reason}"

  defp asset_upload_error(reason), do: inspect(reason)

  # Shared handler for update/extend/derive — each takes (id, attrs) and returns {:ok, mem}.
  defp memory_mutation_endpoint(conn, id, fun) do
    body = conn.body_params || %{}
    content = Map.get(body, "content")

    if not (is_binary(content) and content != "") do
      send_resp(conn, 400, Jason.encode!(%{error: "content is required"}))
    else
      attrs =
        %{content: content}
        |> maybe_put(:audience, Map.get(body, "audience"))
        |> maybe_put(:citation_uri, Map.get(body, "citation_uri"))
        |> maybe_put(:metadata, Map.get(body, "metadata"))

      case fun.(id, attrs) do
        {:ok, mem} ->
          conn
          |> put_resp_content_type("application/json")
          |> send_resp(201, Jason.encode!(memory_to_map(mem)))

        {:error, :not_found} ->
          send_resp(conn, 404, Jason.encode!(%{error: "memory not found"}))

        {:error, reason} ->
          send_resp(conn, 500, Jason.encode!(%{error: inspect(reason)}))
      end
    end
  end

  defp api_key_to_map(key) do
    %{
      id: key.id,
      tenant_id: key.tenant_id,
      principal_id: key.principal_id,
      prefix: key.prefix,
      name: key.name,
      scopes: key.scopes,
      workspace_scope: key.workspace_scope,
      expires_at: key.expires_at,
      created_at: key.created_at,
      last_used_at: key.last_used_at,
      revoked_at: key.revoked_at
    }
  end

  defp maybe_put(map, _key, nil), do: map
  defp maybe_put(map, key, val), do: Map.put(map, key, val)

  # Coerce an arbitrary metadata value from the request body into a string-keyed map.
  defp coerce_metadata(nil), do: %{}
  defp coerce_metadata(m) when is_map(m), do: m
  defp coerce_metadata(_), do: %{}
  # Keyword-list variant for the /api/grep opts builder.
  defp api_maybe_put(kw, _key, nil), do: kw
  defp api_maybe_put(kw, key, val), do: Keyword.put(kw, key, val)

  # Governed-recall opt-in for POST /api/rag: forwards context_package opts
  # to Retrieval.ask/2. No-op unless "context_package": true.
  #
  # SECURITY: the authorization envelope (actor + allowed_partitions +
  # allowed_security_labels) is derived SERVER-SIDE from the authenticated
  # principal — body-supplied "actor"/"partitions"/"security_labels" are
  # ignored, so a caller can never widen recall by asserting an envelope.
  # Without an API key (anonymous dev mode) no grants are forwarded and the
  # Retrieval Coordinator's fail-closed empty-envelope behavior applies: only
  # unlabeled, unpartitioned objects are admitted.
  defp rag_context_package_opts(rag_opts, body, conn) do
    if parse_bool(body["context_package"], false) do
      rag_opts
      |> Keyword.put(:context_package, true)
      |> api_maybe_put(:actor_id, authenticated_actor(conn))
      |> api_maybe_put(:allowed_partitions, principal_grants(conn, "allowed_partitions"))
      |> api_maybe_put(
        :allowed_security_labels,
        principal_grants(conn, "allowed_security_labels")
      )
    else
      rag_opts
    end
  end

  # The actor recorded in the governed scope envelope is the authenticated
  # principal set by AuthPlug — never a body-asserted name.
  defp authenticated_actor(conn) do
    case conn.assigns[:current_principal] do
      principal when is_binary(principal) and principal != "" -> principal
      _ -> nil
    end
  end

  # Server-side ACL grants for governed recall, read from the verified API
  # key's metadata ("allowed_partitions" / "allowed_security_labels" string
  # lists, set at mint time). Returns nil (opt omitted) when the caller has no
  # key or the key carries no such grant — the coordinator then fails closed.
  defp principal_grants(conn, grant) do
    case conn.assigns[:current_api_key] do
      %ApiKey{metadata: %{} = metadata} ->
        case Map.get(metadata, grant) do
          grants when is_list(grants) -> Enum.map(grants, &to_string/1)
          _ -> nil
        end

      _ ->
        nil
    end
  end

  # ContextPackage structs are projected to plain maps before JSON encoding;
  # legacy RAG results pass through unchanged.
  defp render_rag_result(%{context_package: %ContextPackage{} = package} = result) do
    Map.put(result, :context_package, ContextPackage.to_map(package))
  end

  defp render_rag_result(result), do: result

  defp parse_bool(true, _default), do: true
  defp parse_bool(false, _default), do: false
  defp parse_bool("true", _default), do: true
  defp parse_bool("1", _default), do: true
  defp parse_bool("false", _default), do: false
  defp parse_bool("0", _default), do: false
  defp parse_bool(_, default), do: default

  defp parse_float(nil), do: nil
  defp parse_float(""), do: nil
  defp parse_float(value) when is_float(value), do: value
  defp parse_float(value) when is_integer(value), do: value / 1

  defp parse_float(value) when is_binary(value) do
    case Float.parse(value) do
      {float, ""} -> float
      _ -> nil
    end
  end

  defp parse_float(_value), do: nil

  # Resolve a workspace id (e.g. "default" or "default:engineering") to a
  # Workspace struct. Returns `{:error, :not_found}` for unknown ids.
  defp resolve_workspace(id) do
    OptimalEngine.Workspace.get(id)
  end

  # Recursively convert atom keys to strings for JSON serialisation.
  # Config maps use atom keys internally; JSON must emit strings.
  defp context_package_to_map(%ContextPackage{} = package) do
    package
    |> ContextPackage.to_map()
    |> stringify_keys()
  end

  defp context_package_to_map(package), do: stringify_keys(package)

  defp stringify_keys(%DateTime{} = value), do: DateTime.to_iso8601(value)
  defp stringify_keys(%NaiveDateTime{} = value), do: NaiveDateTime.to_iso8601(value)
  defp stringify_keys(%Date{} = value), do: Date.to_iso8601(value)
  defp stringify_keys(%Time{} = value), do: Time.to_iso8601(value)

  defp stringify_keys(%_struct{} = value) do
    value
    |> Map.from_struct()
    |> stringify_keys()
  end

  defp stringify_keys(map) when is_map(map) do
    Map.new(map, fn
      {k, v} when is_atom(k) -> {Atom.to_string(k), stringify_keys(v)}
      {k, v} -> {k, stringify_keys(v)}
    end)
  end

  defp stringify_keys(list) when is_list(list), do: Enum.map(list, &stringify_keys/1)
  defp stringify_keys(other), do: other

  defp append_if(list, true, item), do: list ++ [item]
  defp append_if(list, false, _item), do: list

  defp body_audience(conn), do: query_param(conn, "audience", "default")

  defp do_recall(conn, q, audience, workspace) do
    if String.length(String.trim(q)) < 8 do
      send_resp(
        conn,
        400,
        Jason.encode!(%{error: "missing required parameter(s) — recall query was too short"})
      )
    else
      receiver = Receiver.new(%{format: :markdown, bandwidth: :medium, audience: audience})
      {:ok, result} = Retrieval.ask(q, receiver: receiver, workspace_id: workspace)
      json(conn, Map.put(result, :recall_query, q))
    end
  end

  # ── SSE plumbing for /api/surface/stream ────────────────────────────────

  defp subscription_to_map(s) do
    # Strip webhook_secret from the API response — never expose signing keys
    safe_metadata = Map.drop(s.metadata || %{}, ["webhook_secret"])

    %{
      id: s.id,
      tenant_id: s.tenant_id,
      workspace_id: s.workspace_id,
      principal_id: s.principal_id,
      scope: Atom.to_string(s.scope),
      scope_value: s.scope_value,
      categories: Enum.map(s.categories, &Atom.to_string/1),
      activity: s.activity,
      status: Atom.to_string(s.status),
      created_at: s.created_at,
      metadata: safe_metadata
    }
  end

  defp parse_categories(nil), do: nil

  defp parse_categories(list) when is_list(list) do
    Enum.map(list, fn c ->
      try do
        String.to_existing_atom(c)
      rescue
        _ -> :unassigned
      end
    end)
  end

  defp parse_categories(_), do: nil

  defp stream_init(conn) do
    conn
    |> put_resp_header("content-type", "text/event-stream")
    |> put_resp_header("cache-control", "no-cache")
    |> put_resp_header("connection", "keep-alive")
    |> send_chunked(200)
  end

  defp send_sse(conn, event, data) do
    payload = Jason.encode!(data)
    chunk = "event: #{event}\ndata: #{payload}\n\n"

    case Plug.Conn.chunk(conn, chunk) do
      {:ok, conn} -> conn
      {:error, _} -> conn
    end
  end

  # Long-running receive loop. Listens for surface pushes, heartbeats every
  # 25s to keep proxies from cutting the connection, exits when the client
  # disconnects (chunk write fails).
  defp stream_loop(conn, sub_id) do
    receive do
      {:surface, payload} ->
        case Plug.Conn.chunk(conn, "event: surface\ndata: #{Jason.encode!(payload)}\n\n") do
          {:ok, conn} -> stream_loop(conn, sub_id)
          {:error, _} -> stream_close(conn, sub_id)
        end

      {:plug_conn, :sent} ->
        stream_loop(conn, sub_id)

      _other ->
        stream_loop(conn, sub_id)
    after
      25_000 ->
        case Plug.Conn.chunk(conn, ": keepalive\n\n") do
          {:ok, conn} -> stream_loop(conn, sub_id)
          {:error, _} -> stream_close(conn, sub_id)
        end
    end
  end

  # Receive loop for /api/rag/stream — drains RagStream messages and forwards
  # each as an SSE event. Keepalive every 25s; hard timeout after 30s of silence
  # from the pipeline. Returns the final conn when done or on error.
  defp rag_stream_loop(conn) do
    receive do
      {:rag_stream_event, event, payload} ->
        conn = send_sse(conn, to_string(event), payload)
        rag_stream_loop(conn)

      {:rag_stream_done, final} ->
        conn = send_sse(conn, "envelope", final.envelope)

        send_sse(conn, "done", %{
          elapsed_ms: final.trace.elapsed_ms,
          source: final.source
        })

      {:rag_stream_error, reason} ->
        send_sse(conn, "error", %{error: inspect(reason)})

      {:plug_conn, :sent} ->
        rag_stream_loop(conn)

      # Task links emit a {:DOWN, ...} or normal exit message — ignore safely.
      {_ref, _result} ->
        rag_stream_loop(conn)

      _other ->
        rag_stream_loop(conn)
    after
      30_000 ->
        send_sse(conn, "error", %{error: "timeout"})
    end
  end

  defp stream_close(conn, sub_id) do
    OptimalEngine.Memory.Surfacer.unsubscribe(sub_id, self())
    conn
  end

  defp parse_atom(v, default) when is_atom(default) do
    case v do
      nil -> default
      "" -> default
      s when is_binary(s) -> String.to_existing_atom(s)
      a when is_atom(a) -> a
    end
  rescue
    _ -> default
  end

  defp query_param(conn, key, default) do
    conn = Plug.Conn.fetch_query_params(conn)
    Map.get(conn.query_params, key, default)
  end

  defp parse_int(nil, default), do: default
  defp parse_int(n, _default) when is_integer(n) and n > 0, do: n

  defp parse_int(str, default) do
    case Integer.parse(to_string(str)) do
      {n, _} when n > 0 -> n
      _ -> default
    end
  end

  # ---------------------------------------------------------------------------
  # Helpers: data fetching (raw SQL via Store.raw_query/2)
  # ---------------------------------------------------------------------------

  # Every context in the requested workspace — each becomes a renderable node
  # in the graph. Always workspace-filtered: no cross-workspace leakage.
  defp fetch_all_contexts(workspace_id) do
    case Store.raw_query(
           """
           SELECT id, title, node, type, genre, sn_ratio, modified_at, l0_abstract, uri
           FROM contexts
           WHERE workspace_id = ?1
           ORDER BY node, modified_at DESC
           """,
           [workspace_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [id, title, node, type, genre, sn, mod, abstract, uri] ->
          %{
            id: id,
            title: title,
            node: node,
            type: type,
            genre: genre,
            sn_ratio: sn,
            modified_at: mod,
            l0_abstract: abstract,
            uri: uri
          }
        end)

      _ ->
        []
    end
  end

  # Build visual edges between context IDs (not entity/node names).
  #
  # Two strategies:
  #   1. shared_entity — contexts that both mention the same entity are linked.
  #      Weight = number of shared entities, normalised to 1.0 max.
  #   2. cross_ref     — cross_ref edges in the edges table go from a context_id
  #      to a node name; we resolve the node name to actual contexts in that node.
  #
  # All edges are deduplicated: only the (min_id, max_id) pair is kept so
  # A→B and B→A are not both emitted.
  defp fetch_visual_edges(workspace_id) do
    entity_edges = fetch_shared_entity_edges(workspace_id)
    cross_ref_edges = fetch_cross_ref_edges(workspace_id)

    (entity_edges ++ cross_ref_edges)
    |> deduplicate_edges()
  end

  # Both endpoints are resolved through the contexts table (the source of
  # truth for workspace membership) so legacy entity rows backfilled to
  # 'default' can never bridge two workspaces.
  defp fetch_shared_entity_edges(workspace_id) do
    sql = """
    SELECT e1.context_id AS source, e2.context_id AS target, e1.name AS entity
    FROM entities e1
    JOIN entities e2 ON e1.name = e2.name AND e1.context_id < e2.context_id
    JOIN contexts c1 ON c1.id = e1.context_id AND c1.workspace_id = ?1
    JOIN contexts c2 ON c2.id = e2.context_id AND c2.workspace_id = ?1
    LIMIT 2000
    """

    case Store.raw_query(sql, [workspace_id]) do
      {:ok, rows} ->
        # Group by (source, target) pair and count shared entities
        rows
        |> Enum.group_by(fn [source, target, _entity] -> {source, target} end)
        |> Enum.map(fn {{source, target}, matches} ->
          shared_count = length(matches)
          # Pull distinct entity names for metadata (first 3 for brevity)
          entity_names = matches |> Enum.map(fn [_, _, e] -> e end) |> Enum.uniq() |> Enum.take(3)
          weight = min(shared_count / 5.0, 1.0)

          %{
            source: source,
            target: target,
            relation: "shared_entity",
            weight: Float.round(weight, 2),
            entities: entity_names,
            shared_count: shared_count
          }
        end)

      _ ->
        []
    end
  end

  defp fetch_cross_ref_edges(workspace_id) do
    # cross_ref edges: source_id is a context_id, target_id is a node name.
    # Resolve target node name → all context IDs in that node, requiring both
    # endpoints to live in the requested workspace.
    sql = """
    SELECT e.source_id AS source, c.id AS target
    FROM edges e
    JOIN contexts c ON c.node = e.target_id AND c.workspace_id = ?1
    WHERE e.relation = 'cross_ref'
      AND EXISTS (SELECT 1 FROM contexts WHERE id = e.source_id AND workspace_id = ?1)
    LIMIT 500
    """

    case Store.raw_query(sql, [workspace_id]) do
      {:ok, rows} ->
        rows
        |> Enum.map(fn [source, target] ->
          %{source: source, target: target, relation: "cross_ref", weight: 1.0}
        end)

      _ ->
        []
    end
  end

  # Remove duplicate pairs: keep only one direction per (source, target) pair.
  # For shared_entity edges this is already handled by the `e1.context_id < e2.context_id`
  # constraint, but cross_ref edges may produce duplicates across both strategies.
  defp deduplicate_edges(edges) do
    edges
    |> Enum.uniq_by(fn %{source: s, target: t} ->
      if s < t, do: {s, t}, else: {t, s}
    end)
  end

  defp fetch_entity_summary(workspace_id) do
    case Store.raw_query(
           """
           SELECT e.name, e.type, COUNT(*) as count
           FROM entities e
           JOIN contexts c ON c.id = e.context_id AND c.workspace_id = ?1
           GROUP BY e.name, e.type
           ORDER BY count DESC
           LIMIT 200
           """,
           [workspace_id]
         ) do
      {:ok, rows} -> rows
      _ -> []
    end
  end

  defp fetch_node_summary(workspace_id) do
    case Store.raw_query(
           """
           SELECT node, COUNT(*) as count, MAX(modified_at) as last_modified
           FROM contexts
           WHERE workspace_id = ?1
           GROUP BY node
           ORDER BY node
           """,
           [workspace_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [node, count, last_mod] ->
          %{node: node, context_count: count, last_modified: last_mod}
        end)

      _ ->
        []
    end
  end

  # Node slugs are reused across workspaces — both queries must filter by
  # workspace_id in addition to the slug (REALITY-AUDIT: natural key is
  # workspace_id + slug).
  defp fetch_node_contexts(node_id, workspace_id) do
    case Store.raw_query(
           """
           SELECT id, title, type, genre, sn_ratio, modified_at
           FROM contexts
           WHERE node = ?1 AND workspace_id = ?2
           ORDER BY modified_at DESC
           LIMIT 50
           """,
           [node_id, workspace_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [id, title, type, genre, sn, mod] ->
          %{id: id, title: title, type: type, genre: genre, sn_ratio: sn, modified_at: mod}
        end)

      _ ->
        []
    end
  end

  defp fetch_node_edges(node_id, workspace_id) do
    # Return edges where at least one endpoint lives in this node.
    # We join via the contexts table to resolve node membership.
    case Store.raw_query(
           """
           SELECT DISTINCT e.source_id, e.target_id, e.relation, e.weight
           FROM edges e
           JOIN contexts c ON (c.id = e.source_id OR c.id = e.target_id)
           WHERE c.node = ?1 AND c.workspace_id = ?2
           LIMIT 200
           """,
           [node_id, workspace_id]
         ) do
      {:ok, rows} -> rows
      _ -> []
    end
  end

  # ---------------------------------------------------------------------------
  # Helpers: data fetching for the ported knowledge components
  # ---------------------------------------------------------------------------

  # Entities for OptimalGraphView — one row per unique (name, type), with
  # `connections` = how many distinct contexts reference that entity.
  # Workspace membership resolved via the contexts table.
  defp optimal_entity_summary(workspace_id) do
    case Store.raw_query(
           """
           SELECT e.name, e.type, COUNT(DISTINCT e.context_id) AS connections
           FROM entities e
           JOIN contexts c ON c.id = e.context_id AND c.workspace_id = ?1
           WHERE e.name IS NOT NULL AND e.name <> ''
           GROUP BY e.name, e.type
           ORDER BY connections DESC
           LIMIT 300
           """,
           [workspace_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [name, type, connections] ->
          %{name: name, type: type || "concept", connections: connections}
        end)

      _ ->
        []
    end
  end

  # Edges between entities by co-occurrence in the same context. Two entities
  # sharing a context produce one `related_to` edge; weight is the number of
  # shared contexts (capped at 5.0 to keep visuals stable).
  defp optimal_relation_edges(workspace_id) do
    sql = """
    SELECT e1.name AS source, e2.name AS target, COUNT(*) AS shared
    FROM entities e1
    JOIN entities e2
      ON e1.context_id = e2.context_id
     AND e1.name < e2.name
    JOIN contexts c ON c.id = e1.context_id AND c.workspace_id = ?1
    WHERE e1.name IS NOT NULL AND e2.name IS NOT NULL
    GROUP BY e1.name, e2.name
    HAVING shared >= 1
    ORDER BY shared DESC
    LIMIT 800
    """

    case Store.raw_query(sql, [workspace_id]) do
      {:ok, rows} ->
        Enum.map(rows, fn [s, t, shared] ->
          %{
            source: s,
            target: t,
            relation: "related_to",
            weight: min(shared * 1.0, 5.0)
          }
        end)

      _ ->
        []
    end
  end

  # Node list for the drill-down level-0 card grid. Pulls from the workspace
  # `nodes` table (Phase 3.5) joined against context counts, so operators see
  # real signal volumes per node rather than just names.
  defp optimal_node_summary(workspace_id) do
    sql = """
    SELECT n.slug, n.name, COALESCE(n.kind, 'node') AS type,
           COALESCE((SELECT COUNT(*) FROM contexts c WHERE c.node = n.slug AND c.workspace_id = ?1), 0) AS signal_count
    FROM nodes n
    WHERE n.workspace_id = ?1
    ORDER BY n.slug
    """

    case Store.raw_query(sql, [workspace_id]) do
      {:ok, rows} ->
        Enum.map(rows, fn [slug, name, type, count] ->
          %{slug: slug, name: name || slug, type: type, signal_count: count}
        end)

      _ ->
        # Fallback for tenants that haven't populated the nodes table yet —
        # derive distinct nodes straight from contexts.
        case Store.raw_query(
               "SELECT node, COUNT(*) FROM contexts WHERE node IS NOT NULL AND workspace_id = ?1 GROUP BY node",
               [workspace_id]
             ) do
          {:ok, rows} ->
            Enum.map(rows, fn [slug, count] ->
              %{slug: slug, name: slug, type: "node", signal_count: count}
            end)

          _ ->
            []
        end
    end
  end

  # File tree for NodeDrillDown — one entry per signal in a given node,
  # flattened with `is_dir: false`. The component also accepts nested
  # children but the engine stores a flat list per node, so we return that.
  defp optimal_node_files(slug, workspace_id) do
    case Store.raw_query(
           """
           SELECT id, title, uri, genre, modified_at, LENGTH(content) AS size
           FROM contexts
           WHERE node = ?1 AND workspace_id = ?2
           ORDER BY modified_at DESC
           LIMIT 200
           """,
           [slug, workspace_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [id, title, uri, genre, modified_at, size] ->
          %{
            name: title || id,
            path: uri || id,
            is_dir: false,
            size: size || 0,
            genre: genre,
            modified_at: modified_at
          }
        end)

      _ ->
        []
    end
  end

  # ---------------------------------------------------------------------------
  # Helpers: workspace / signal / activity / architecture (Phase 14)
  # ---------------------------------------------------------------------------

  # Flat node list with enough structure for a client-side tree build.
  # Scoped to the given workspace_id (default: "default").
  defp workspace_nodes(workspace_id) do
    sql = """
    SELECT n.id, n.slug, n.name, n.kind, n.parent_id, n.style, n.status,
           COALESCE((SELECT COUNT(*) FROM contexts c WHERE c.node = n.slug AND c.workspace_id = ?1), 0) AS signal_count
    FROM nodes n
    WHERE n.workspace_id = ?1
    ORDER BY COALESCE(n.parent_id, ''), n.slug
    """

    case Store.raw_query(sql, [workspace_id]) do
      {:ok, rows} ->
        Enum.map(rows, fn [id, slug, name, kind, parent_id, style, status, count] ->
          %{
            id: id,
            slug: slug,
            name: name || slug,
            kind: kind || "node",
            parent_id: parent_id,
            style: style || "internal",
            status: status || "active",
            signal_count: count
          }
        end)

      _ ->
        []
    end
  end

  # Full granularity for one signal — enough for a drill-down to visualize
  # every layer the engine tracks. The parent row is fetched with a
  # workspace_id guard so an id from another workspace yields nil (→ 404);
  # the child fetchers below are only reachable once that guard passed.
  defp signal_detail(id, workspace_id) do
    with {:ok, [row]} <- signal_row(id, workspace_id) do
      [
        ctx_id,
        uri,
        title,
        genre,
        mode,
        type,
        format,
        structure,
        node,
        sn_ratio,
        content,
        l0,
        l1,
        modified_at,
        architecture_id
      ] = row

      %{
        id: ctx_id,
        uri: uri,
        title: title,
        node: node,
        genre: genre,
        modified_at: modified_at,
        architecture_id: architecture_id,
        signal_dimensions: %{
          mode: mode,
          genre: genre,
          type: type,
          format: format,
          structure: structure
        },
        sn_ratio: sn_ratio,
        content: content,
        l0_abstract: l0,
        l1_overview: l1,
        chunks: signal_chunks(ctx_id),
        entities: signal_entities(ctx_id),
        classification: signal_classification(ctx_id),
        intent: signal_intent(ctx_id),
        clusters: signal_clusters(ctx_id),
        citations: signal_wiki_citations(ctx_id)
      }
    else
      _ -> nil
    end
  end

  defp signal_row(id, workspace_id) do
    Store.raw_query(
      """
      SELECT id, uri, title, genre, mode, signal_type, format, structure, node,
             sn_ratio, content, l0_abstract, l1_overview, modified_at,
             architecture_id
      FROM contexts
      WHERE id = ?1 AND workspace_id = ?2 LIMIT 1
      """,
      [id, workspace_id]
    )
  end

  defp signal_chunks(signal_id) do
    case Store.raw_query(
           """
           SELECT id, parent_id, scale, modality, length_bytes
           FROM chunks WHERE signal_id = ?1
           ORDER BY
             CASE scale
               WHEN 'document'  THEN 0
               WHEN 'section'   THEN 1
               WHEN 'paragraph' THEN 2
               WHEN 'sentence'  THEN 3
               ELSE 4
             END,
             id
           """,
           [signal_id]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [cid, parent, scale, modality, len] ->
          %{id: cid, parent_id: parent, scale: scale, modality: modality, length_bytes: len}
        end)

      _ ->
        []
    end
  end

  defp signal_entities(signal_id) do
    case Store.raw_query(
           "SELECT name, type FROM entities WHERE context_id = ?1 ORDER BY type, name",
           [signal_id]
         ) do
      {:ok, rows} -> Enum.map(rows, fn [n, t] -> %{name: n, type: t} end)
      _ -> []
    end
  end

  defp signal_classification(signal_id) do
    case Store.raw_query(
           """
           SELECT mode, genre, signal_type, format, structure, sn_ratio, confidence
           FROM classifications WHERE chunk_id = ?1 LIMIT 1
           """,
           ["#{signal_id}:doc"]
         ) do
      {:ok, [[mode, genre, type, format, structure, sn, conf]]} ->
        %{
          mode: mode,
          genre: genre,
          type: type,
          format: format,
          structure: structure,
          sn_ratio: sn,
          confidence: conf
        }

      _ ->
        nil
    end
  end

  defp signal_intent(signal_id) do
    case Store.raw_query(
           "SELECT intent, confidence FROM intents WHERE chunk_id = ?1 LIMIT 1",
           ["#{signal_id}:doc"]
         ) do
      {:ok, [[intent, confidence]]} -> %{intent: intent, confidence: confidence}
      _ -> nil
    end
  end

  defp signal_clusters(signal_id) do
    sql = """
    SELECT c.id, c.theme, c.intent_dominant, cm.weight
    FROM cluster_members cm
    JOIN clusters c ON c.id = cm.cluster_id
    WHERE cm.chunk_id = ?1
    """

    case Store.raw_query(sql, ["#{signal_id}:doc"]) do
      {:ok, rows} ->
        Enum.map(rows, fn [cid, theme, dom, w] ->
          %{id: cid, theme: theme, intent_dominant: dom, weight: w}
        end)

      _ ->
        []
    end
  end

  defp signal_wiki_citations(signal_id) do
    case Store.raw_query(
           """
           SELECT wiki_slug, wiki_audience, claim_hash, last_verified
           FROM citations WHERE chunk_id = ?1
           """,
           ["#{signal_id}:doc"]
         ) do
      {:ok, rows} ->
        Enum.map(rows, fn [slug, audience, claim, verified] ->
          %{wiki_slug: slug, audience: audience, claim_hash: claim, last_verified: verified}
        end)

      _ ->
        []
    end
  end

  # Activity feed — append-only events table. Most-recent first.
  # Always workspace-filtered (uses idx_events_ws_ts) so callers never see
  # other workspaces' audit entries.
  defp recent_events(limit, offset, kind, workspace_id) do
    {where, params} =
      if kind == "" do
        {"WHERE workspace_id = ?3", [limit, offset, workspace_id]}
      else
        {"WHERE workspace_id = ?3 AND kind = ?4", [limit, offset, workspace_id, kind]}
      end

    sql = """
    SELECT id, tenant_id, ts, principal, kind, target_uri, latency_ms, metadata
    FROM events #{where}
    ORDER BY ts DESC, id DESC
    LIMIT ?1 OFFSET ?2
    """

    case Store.raw_query(sql, params) do
      {:ok, rows} ->
        Enum.map(rows, fn [id, tenant, ts, principal, kind, uri, latency, metadata] ->
          %{
            id: id,
            tenant_id: tenant,
            ts: ts,
            principal: principal,
            kind: kind,
            target_uri: uri,
            latency_ms: latency,
            metadata: decode_json_meta(metadata)
          }
        end)

      _ ->
        []
    end
  end

  defp count_events(kind, workspace_id) do
    {where, params} =
      if kind == "" do
        {"WHERE workspace_id = ?1", [workspace_id]}
      else
        {"WHERE workspace_id = ?1 AND kind = ?2", [workspace_id, kind]}
      end

    sql = "SELECT COUNT(*) FROM events #{where}"

    case Store.raw_query(sql, params) do
      {:ok, [[n]]} -> n
      _ -> 0
    end
  end

  defp decode_json_meta(nil), do: %{}
  defp decode_json_meta(""), do: %{}

  defp decode_json_meta(str) do
    case Jason.decode(str) do
      {:ok, m} when is_map(m) -> m
      _ -> %{}
    end
  end

  # Full architecture detail for the /api/architectures/:id endpoint.
  defp architecture_detail(arch) do
    %{
      id: arch.id,
      name: arch.name,
      version: arch.version,
      description: arch.description,
      modality_primary: arch.modality_primary,
      granularity: arch.granularity,
      retention: arch.retention,
      fields:
        Enum.map(arch.fields, fn f ->
          %{
            name: f.name,
            modality: f.modality,
            dims: f.dims,
            required: f.required,
            processor: f.processor,
            description: f.description
          }
        end)
    }
  end

  # ---------------------------------------------------------------------------
  # Helpers: formatting
  # ---------------------------------------------------------------------------

  # Raw edge rows come from raw_query as [source_id, target_id, relation, weight]
  defp format_edges(rows) when is_list(rows) do
    Enum.map(rows, fn
      [s, t, r, w] ->
        %{source: s, target: t, relation: r, weight: w}

      %{source_id: s, target_id: t, relation: r, weight: w} ->
        %{source: s, target: t, relation: r, weight: w}

      other ->
        %{raw: inspect(other)}
    end)
  end

  defp format_edges(_), do: []

  # Entity rows: [name, type, count]
  defp format_entities(rows) when is_list(rows) do
    Enum.map(rows, fn
      [name, type, count] -> %{name: name, type: type, count: count}
      other -> %{raw: inspect(other)}
    end)
  end

  defp format_entities(_), do: []

  # GraphAnalyzer.hubs/0 returns [%{id:, degree:, sigma:}]
  defp format_hubs(hubs) when is_list(hubs) do
    Enum.map(hubs, fn
      %{id: id, degree: degree, sigma: sigma} ->
        %{entity: id, degree: degree, sigma: Float.round(sigma, 2)}

      other ->
        %{raw: inspect(other)}
    end)
  end

  defp format_hubs(_), do: []

  # GraphAnalyzer.triangles/1 returns [%{a:, b:, c:, suggestion:}]
  defp format_triangles(triangles) when is_list(triangles) do
    Enum.map(triangles, fn
      %{a: a, b: b, c: c, suggestion: suggestion} ->
        %{a: a, b: b, missing_link: c, suggestion: suggestion}

      other ->
        %{raw: inspect(other)}
    end)
  end

  defp format_triangles(_), do: []

  # GraphAnalyzer.clusters/0 returns [MapSet.t()]
  defp format_clusters(clusters) when is_list(clusters) do
    clusters
    |> Enum.with_index()
    |> Enum.map(fn {members, idx} ->
      member_list = MapSet.to_list(members)
      %{id: idx, size: length(member_list), members: member_list}
    end)
  end

  defp format_clusters(_), do: []

  # Reflector.reflect/1 returns [%{source:, target:, cooccurrences:, confidence:, suggested_relation:, reason:}]
  defp format_gaps(gaps) when is_list(gaps) do
    Enum.map(gaps, fn
      %{source: s, target: t, cooccurrences: c, confidence: conf} = gap ->
        %{
          source: s,
          target: t,
          cooccurrences: c,
          confidence: Float.round(conf, 2),
          suggested_relation: Map.get(gap, :suggested_relation, "related"),
          reason: Map.get(gap, :reason, nil)
        }

      other ->
        %{raw: inspect(other)}
    end)
  end

  defp format_gaps(_), do: []

  # OptimalEngine.search/2 returns [%Context{}]
  defp format_search_results(contexts) when is_list(contexts) do
    Enum.map(contexts, fn ctx ->
      %{
        id: Map.get(ctx, :id),
        title: Map.get(ctx, :title),
        node: Map.get(ctx, :node),
        genre: Map.get(ctx, :genre),
        uri: Map.get(ctx, :uri),
        l0_abstract: Map.get(ctx, :l0_abstract),
        sn_ratio: Map.get(ctx, :sn_ratio)
      }
    end)
  end

  defp format_search_results(_), do: []

  # HealthDiagnostics.run/0 returns [%{name:, severity:, message:, details:, fix:}]
  defp format_health(checks) when is_list(checks) do
    Enum.map(checks, fn
      %{name: name, severity: severity, message: message} = check ->
        %{
          check: to_string(name),
          status: to_string(severity),
          message: message,
          details: Map.get(check, :details, []),
          fix: Map.get(check, :fix, nil)
        }

      other ->
        %{raw: inspect(other)}
    end)
  end

  defp format_health(_), do: []
end
