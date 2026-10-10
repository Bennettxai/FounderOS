defmodule OptimalEngine.Retrieval.RAG do
  @moduledoc """
  End-to-end Retrieval-Augmented Generation flow.

      query + receiver
         │
         ▼
      [1] IntentAnalyzer   — expand query, extract entities
         │
         ▼
      [2] WikiFirst        — Tier 3 lookup (curated page?)
         │
         ├──hit──▶ [3a] Deliver.render_wiki   ──▶ envelope
         │
         └─miss─▶ [3b] Search (hybrid, principal-filtered)
                   │
                   ▼
                 [4] BandwidthPlanner.plan
                   │
                   ▼
                 [5] Deliver.render_chunks    ──▶ envelope

  One function — `ask/2` — is the public entry point. Everything
  else is plumbing.

  With `context_package: true`, `ask/2` skips this pipeline and answers
  through `OptimalEngine.MemoryCore.RetrievalCoordinator`, returning a
  governed Context Package alongside the envelope.

  ## Return shape

      {:ok, %{
        source: :wiki | :chunks | :empty | :context_package,
        envelope: %{body, format, sources, warnings},
        trace: %{
          intent: map(),
          wiki_hit?: boolean(),
          n_candidates: non_neg_integer(),
          n_delivered: non_neg_integer(),
          truncated?: boolean(),
          elapsed_ms: non_neg_integer()
        }
      }}

  The `trace` field is the Wiener feedback loop — it tells the caller
  exactly what happened so the receiver (human or agent) can decide
  whether to dig deeper.
  """

  alias OptimalEngine.Retrieval.{
    BandwidthPlanner,
    Deliver,
    IntentAnalyzer,
    Receiver,
    Search,
    WikiFirst
  }

  alias OptimalEngine.MemoryCore.{ContextPackage, RetrievalCoordinator, ScopeEnvelope}
  alias OptimalEngine.{Memory, Store}
  alias OptimalEngine.Wiki.Directives

  require Logger

  @type ask_result :: %{
          optional(:context_package) => ContextPackage.t(),
          source: :wiki | :chunks | :memory | :empty | :context_package,
          envelope: Deliver.envelope(),
          trace: trace()
        }

  @type trace :: %{
          intent: map(),
          wiki_hit?: boolean(),
          n_candidates: non_neg_integer(),
          n_delivered: non_neg_integer(),
          truncated?: boolean(),
          elapsed_ms: non_neg_integer()
        }

  @default_hybrid_limit 20

  @doc """
  Ask the engine a question on behalf of `receiver`.

  Return shape (cleanup rule 7): by default this is the legacy compatibility
  path — the envelope is rendered from a curated wiki page or from **raw
  search hits** over compatibility `contexts` rows, not from governed memory.
  With `context_package: true` the question routes through
  `OptimalEngine.MemoryCore.RetrievalCoordinator` instead and the result
  carries a governed `%ContextPackage{}` under `:context_package`
  (`source: :context_package`); the envelope body is rendered from the
  package's assembled sections.

  Options:
    * `:receiver` — `%Receiver{}`; defaults to `Receiver.anonymous/0`
    * `:hybrid_limit` — max chunks pulled from hybrid search (default 20)
    * `:skip_wiki` — bypass Tier 3 entirely (testing / debugging)
    * `:skip_intent` — bypass intent expansion (testing)
    * `:tenant_id` — override tenant scope (defaults to `receiver.tenant_id`)
    * `:workspace_id` — scope retrieval to a workspace. Defaults to the
      tenant's own default workspace (`"default"` for the default tenant,
      `"<tenant_id>:default"` otherwise) — never to a workspace shared
      across tenants
    * `:context_package` — when `true`, answer with a governed Context Package
      from the Memory Core Retrieval Coordinator (see above)
    * `:actor_id`, `:allowed_partitions`, `:allowed_security_labels` —
      authorization scope forwarded to the coordinator on the
      `context_package: true` path (actor defaults to `receiver.id`)
    * `:on_event` — `(event_atom, payload_map -> any)` — called at each pipeline
      stage boundary for streaming consumers. Existing callers that omit this
      opt see no behaviour change. The function is called synchronously in the
      ask process; keep it non-blocking (e.g. `send/2` to a listener PID).
      Events fired (in order): `:intent`, `:wiki_hit`, `:chunks`, `:composing`
  """
  @spec ask(String.t(), keyword()) :: {:ok, ask_result()} | {:error, term()}
  def ask(query, opts \\ []) when is_binary(query) do
    if Keyword.get(opts, :context_package, false) do
      ask_context_package(query, opts)
    else
      ask_legacy(query, opts)
    end
  end

  defp ask_legacy(query, opts) do
    started = System.monotonic_time(:millisecond)
    on_event = Keyword.get(opts, :on_event, fn _event, _payload -> :ok end)

    receiver =
      case Keyword.get(opts, :receiver) do
        %Receiver{} = r -> r
        nil -> Receiver.anonymous()
      end

    intent =
      if Keyword.get(opts, :skip_intent, false) do
        %{expanded_query: query, intent_type: :lookup, key_entities: [], temporal_hint: :any}
      else
        case IntentAnalyzer.analyze(query) do
          {:ok, i} -> i
          _ -> %{expanded_query: query, intent_type: :lookup, key_entities: [], temporal_hint: :any}
        end
      end

    on_event.(:intent, %{
      intent_type: intent[:intent_type],
      expanded_query: intent[:expanded_query],
      key_entities: intent[:key_entities] || [],
      node_hints: intent[:node_hints] || []
    })

    workspace_id = resolve_workspace_id(opts, receiver)

    wiki_result =
      if Keyword.get(opts, :skip_wiki, false) do
        :miss
      else
        WikiFirst.lookup(query, receiver.audience,
          tenant_id: receiver.tenant_id,
          workspace_id: workspace_id
        )
      end

    {source, envelope, counts} =
      case wiki_result do
        {:hit, page, _reason} ->
          on_event.(:wiki_hit, %{
            slug: page.slug,
            audience: page.audience,
            version: page.version
          })

          on_event.(:composing, %{format: receiver.format, bandwidth: receiver.bandwidth})
          {:wiki, deliver_wiki(page, receiver), %{candidates: 1, delivered: 1, truncated?: false}}

        :miss ->
          fallback_deliver(
            query,
            intent,
            receiver,
            Keyword.merge(Keyword.put(opts, :workspace_id, workspace_id),
              on_event: on_event
            )
          )
      end

    elapsed = System.monotonic_time(:millisecond) - started

    {:ok,
     %{
       source: source,
       envelope: envelope,
       trace: %{
         intent: intent,
         wiki_hit?: source == :wiki,
         n_candidates: counts.candidates,
         n_delivered: counts.delivered,
         truncated?: counts.truncated?,
         elapsed_ms: elapsed
       }
     }}
  end

  # ─── private ─────────────────────────────────────────────────────────────

  # Governed recall path. Returns the persisted ContextPackage (never raw
  # chunks); the envelope body is rendered from the package's assembled
  # sections so existing envelope consumers keep working.
  defp ask_context_package(query, opts) do
    started = System.monotonic_time(:millisecond)

    receiver =
      case Keyword.get(opts, :receiver) do
        %Receiver{} = r -> r
        nil -> Receiver.anonymous()
      end

    coordinator_opts =
      [
        workspace_id: resolve_workspace_id(opts, receiver),
        tenant_id: Keyword.get(opts, :tenant_id) || receiver.tenant_id,
        actor_id: Keyword.get(opts, :actor_id) || receiver.id,
        allowed_partitions: Keyword.get(opts, :allowed_partitions, []),
        allowed_security_labels: Keyword.get(opts, :allowed_security_labels, []),
        limit: Keyword.get(opts, :hybrid_limit, @default_hybrid_limit),
        token_budget: Keyword.get(opts, :token_budget, receiver.token_budget)
      ]

    case RetrievalCoordinator.retrieve(query, coordinator_opts) do
      {:ok, package} ->
        elapsed = System.monotonic_time(:millisecond) - started
        delivered = length(package.fact_links) + length(package.memory_links)
        summary = package.filtered_object_summary

        candidates =
          Map.get(summary, :candidate_facts, 0) + Map.get(summary, :candidate_memory_objects, 0)

        {:ok,
         %{
           source: :context_package,
           envelope: %{
             body: package_body(package),
             format: receiver.format,
             sources: package_sources(package),
             warnings: []
           },
           context_package: package,
           trace: %{
             intent: %{
               expanded_query: query,
               intent_type: :recall,
               key_entities: [],
               temporal_hint: :any
             },
             wiki_hit?: false,
             n_candidates: candidates,
             n_delivered: delivered,
             truncated?: Map.get(package.sections, :truncated?, false),
             elapsed_ms: elapsed
           }
         }}

      {:error, reason} ->
        {:error, reason}
    end
  end

  # Workspace scope derives from the tenant when the caller does not name a
  # workspace. ScopeEnvelope owns the derivation: the default tenant keeps the
  # bare "default" workspace (single-workspace behavior preserved); every
  # other tenant falls back to its own "<tenant_id>:default" workspace, so
  # retrieval never reads the shared literal "default" workspace across
  # tenants.
  defp resolve_workspace_id(opts, receiver) do
    ScopeEnvelope.resolve(
      tenant_id: Keyword.get(opts, :tenant_id) || receiver.tenant_id,
      workspace_id: Keyword.get(opts, :workspace_id)
    ).workspace_id
  end

  defp package_body(package) do
    [package.sections[:summary], package.sections[:facts], package.sections[:memory]]
    |> Enum.reject(&(&1 in [nil, ""]))
    |> Enum.join("\n")
  end

  defp package_sources(package) do
    package.source_package_links
    |> Enum.map(fn
      %{id: id} -> "optimal://source-packages/#{id}"
      %{"id" => id} -> "optimal://source-packages/#{id}"
      _ -> nil
    end)
    |> Enum.reject(&is_nil/1)
    |> Enum.uniq()
  end

  defp deliver_wiki(page, receiver) do
    Deliver.render_wiki(page, receiver, &wiki_resolver/2)
  end

  defp fallback_deliver(query, intent, receiver, opts) do
    case Keyword.get(opts, :memory_limit) do
      nil ->
        case hybrid_deliver(query, intent, receiver, opts) do
          {:empty, _envelope, %{candidates: 0, delivered: 0}} ->
            memory_deliver(query, intent, receiver, opts)

          other ->
            other
        end

      memory_limit when is_integer(memory_limit) and memory_limit > 0 ->
        case memory_deliver(query, intent, receiver, opts) do
          {:empty, _envelope, %{candidates: 0, delivered: 0}} ->
            hybrid_deliver(query, intent, receiver, opts)

          other ->
            other
        end

      _ ->
        case hybrid_deliver(query, intent, receiver, opts) do
          {:empty, _envelope, %{candidates: 0, delivered: 0}} ->
            memory_deliver(query, intent, receiver, opts)

          other ->
            other
        end
    end
  end

  defp hybrid_deliver(query, intent, receiver, opts) do
    on_event = Keyword.get(opts, :on_event, fn _event, _payload -> :ok end)
    limit = Keyword.get(opts, :hybrid_limit, @default_hybrid_limit)
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    expanded = Map.get(intent, :expanded_query, query)

    search_opts =
      [limit: limit, workspace_id: workspace_id]
      |> maybe_put(:principal, receiver.id)

    candidates = safe_search(expanded, search_opts)

    items =
      Enum.map(candidates, fn c ->
        %{
          score: Map.get(c, :score, 0.0),
          content: Map.get(c, :content) || Map.get(c, :l1_description) || "",
          title: Map.get(c, :title) || "Untitled",
          uri: Map.get(c, :uri)
        }
      end)

    on_event.(
      :chunks,
      Enum.map(items, fn item ->
        %{slug: item.uri, score: item.score, scale: "chunk"}
      end)
    )

    plan = BandwidthPlanner.plan(items, receiver.token_budget)

    on_event.(:composing, %{format: receiver.format, bandwidth: receiver.bandwidth})

    case plan.kept do
      [] ->
        {:empty, Deliver.empty(receiver, "No results for `#{query}`."),
         %{
           candidates: length(candidates),
           delivered: 0,
           truncated?: plan.truncated?
         }}

      kept ->
        {:chunks, Deliver.render_chunks(kept, receiver),
         %{
           candidates: length(candidates),
           delivered: length(kept),
           truncated?: plan.truncated?
         }}
    end
  end

  defp memory_deliver(query, intent, receiver, opts) do
    on_event = Keyword.get(opts, :on_event, fn _event, _payload -> :ok end)
    workspace_id = Keyword.get(opts, :workspace_id, "default")
    limit = Keyword.get(opts, :memory_limit, 5)

    candidates =
      query
      |> memory_query_candidates(intent)
      |> Enum.reduce_while([], fn memory_query, [] ->
        case Memory.list(
               workspace_id: workspace_id,
               audience: receiver.audience,
               q: memory_query,
               limit: limit
             ) do
          [] -> {:cont, []}
          memories -> {:halt, memories}
        end
      end)

    items =
      Enum.map(candidates, fn mem ->
        %{
          score: 1.0,
          content: mem.content || "",
          title: "Memory #{mem.id}",
          uri: "optimal://memory/#{mem.id}"
        }
      end)

    on_event.(
      :chunks,
      Enum.map(items, fn item ->
        %{slug: item.uri, score: item.score, scale: "memory"}
      end)
    )

    plan = BandwidthPlanner.plan(items, receiver.token_budget)
    on_event.(:composing, %{format: receiver.format, bandwidth: receiver.bandwidth})

    case plan.kept do
      [] ->
        {:empty, Deliver.empty(receiver, "No results for `#{query}`."),
         %{candidates: 0, delivered: 0, truncated?: false}}

      kept ->
        {:memory, Deliver.render_chunks(kept, receiver),
         %{
           candidates: length(candidates),
           delivered: length(kept),
           truncated?: plan.truncated?
         }}
    end
  end

  @memory_stopwords MapSet.new(~w[
    a an and are as at by did do does for from give handled has have how i in
    is it make made name of on or our recall record recorded remains should
    still that the this to was were what when which who why with work
    decision question unresolved issue responsible rationale explain attached
    owner owners owns owned ownership
  ])

  defp memory_query_candidates(query, intent) do
    intent_entities =
      intent
      |> Map.get(:key_entities, [])
      |> Enum.join(" ")

    [focused_memory_query(query), focused_memory_query(intent_entities), query]
    |> Enum.reject(&(&1 == ""))
    |> Enum.uniq()
  end

  defp focused_memory_query(query) when is_binary(query) do
    Regex.scan(~r/[a-z0-9][a-z0-9_-]*/i, String.downcase(query))
    |> List.flatten()
    |> Enum.reject(&MapSet.member?(@memory_stopwords, &1))
    |> Enum.reject(&(byte_size(&1) < 2))
    |> Enum.join(" ")
  end

  defp focused_memory_query(_), do: ""

  # Resolver used when rendering a wiki page. Today we only resolve
  # `cite` (URIs pass straight through to the format wrapper) and
  # `include` (fetches the referenced context body from the store).
  # Other verbs degrade gracefully to the raw token via `render_fallback`.
  defp wiki_resolver(%Directives.Directive{verb: :cite, argument: uri}, _opts) do
    {:ok, "", %{uri: uri}}
  end

  defp wiki_resolver(%Directives.Directive{verb: :include, argument: uri}, _opts) do
    case fetch_context_body(uri) do
      {:ok, body} -> {:ok, body, %{uri: uri}}
      _ -> {:error, :not_found}
    end
  end

  defp wiki_resolver(%Directives.Directive{verb: :wikilink, argument: slug}, _opts) do
    {:ok, slug, %{}}
  end

  defp wiki_resolver(_, _opts), do: {:error, :unresolved}

  defp fetch_context_body(uri) when is_binary(uri) do
    case Store.raw_query(
           "SELECT content FROM contexts WHERE uri = ?1 LIMIT 1",
           [uri]
         ) do
      {:ok, [[body]]} when is_binary(body) -> {:ok, body}
      _ -> {:error, :not_found}
    end
  end

  defp maybe_put(kw, _k, nil), do: kw
  defp maybe_put(kw, k, v), do: Keyword.put(kw, k, v)

  # Hybrid search can stall (Ollama timeout, vector-store cold cache). We
  # catch the GenServer :timeout exit so a flaky downstream dependency
  # returns an empty result set rather than blowing up the whole ask.
  defp safe_search(query, opts) do
    try do
      case Search.search(query, opts) do
        {:ok, results} -> results
        _ -> []
      end
    catch
      :exit, {:timeout, _} ->
        Logger.warning("RAG: hybrid search timed out for query=#{inspect(query)}")
        []

      :exit, reason ->
        Logger.warning("RAG: hybrid search crashed: #{inspect(reason)}")
        []
    end
  end
end
