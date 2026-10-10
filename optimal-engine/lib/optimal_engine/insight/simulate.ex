defmodule OptimalEngine.Insight.Simulate do
  @moduledoc """
  Scenario planning engine — traces "what if" scenarios through the knowledge graph.

  Given a mutation (person leaves, project cancels, revenue changes), traces
  downstream effects through edges, entities, and cross-references to produce
  a structured impact analysis.

  The Simulator runs as a GenServer so it has access to the OTP supervision tree
  and can cache node-context counts for the lifetime of the process. All public
  API functions are synchronous and return `{:ok, report}` or `{:error, term()}`.

  ## Usage

      {:ok, report} = Simulator.simulate("What if Alice leaves Platform Launch?")
      {:ok, report} = Simulator.simulate("What if we pause Platform Launch?")
      {:ok, report} = Simulator.simulate("What if revenue drops 50%?")
      {:ok, impact} = Simulator.impact_analysis("Alice")
      {:ok, impact} = Simulator.impact_analysis("operation-delivery")
  """

  use GenServer
  require Logger

  alias OptimalEngine.Embed.Ollama, as: Ollama
  alias OptimalEngine.Store

  # Edge weights for scoring: works_on > cross_ref > mentioned_in > lives_in
  @edge_weights %{
    "works_on" => 1.5,
    "cross_ref" => 1.2,
    "mentioned_in" => 1.0,
    "lives_in" => 0.6,
    "supersedes" => 0.4
  }

  @depth_decay %{1 => 1.0, 2 => 0.5, 3 => 0.25}
  @default_max_depth 3

  # Known domain nodes in the system
  @domain_nodes ~w[
    operator product-customer-portal entity-company project-platform-launch entity-company
    operation-delivery team learning-research-library
    inbox team operation-revenue operation-delivery
  ]

  # ---------------------------------------------------------------------------
  # Types
  # ---------------------------------------------------------------------------

  @type mutation_type ::
          :entity_removal | :node_cancellation | :revenue_change | :dependency_break | :general

  @type affected_entity :: %{
          name: String.t(),
          impact: :direct | :indirect,
          contexts: non_neg_integer()
        }

  @type affected_node :: %{
          node: String.t(),
          impact_score: float(),
          context_count: non_neg_integer(),
          description: String.t()
        }

  @type critical_dependency :: %{
          from: String.t(),
          to: String.t(),
          relation: String.t(),
          weight: float()
        }

  @type risk_assessment :: %{
          severity: :low | :medium | :high | :critical,
          reasoning: String.t()
        }

  @type report :: %{
          scenario: String.t(),
          mutation_type: mutation_type(),
          affected_entities: [affected_entity()],
          affected_nodes: [affected_node()],
          critical_dependencies: [critical_dependency()],
          risk_assessment: risk_assessment(),
          recommendations: [String.t()],
          total_contexts_affected: non_neg_integer(),
          graph_depth: non_neg_integer()
        }

  @type impact :: report()

  # ---------------------------------------------------------------------------
  # Client API
  # ---------------------------------------------------------------------------

  @doc "Starts the Simulator GenServer."
  @spec start_link(keyword()) :: GenServer.on_start()
  def start_link(opts \\ []) do
    GenServer.start_link(__MODULE__, opts, name: __MODULE__)
  end

  @doc """
  Runs a scenario simulation.

  Parses the natural-language scenario, traces effects through the knowledge
  graph, and returns a structured impact report.

  ## Options
    - `:max_depth`    — BFS traversal depth (default: #{@default_max_depth})
    - `:workspace_id` — workspace to simulate against (default: "default")
  """
  @spec simulate(String.t(), keyword()) :: {:ok, report()} | {:error, term()}
  def simulate(scenario, opts \\ []) when is_binary(scenario) do
    GenServer.call(__MODULE__, {:simulate, scenario, opts}, 60_000)
  end

  @doc """
  Runs an impact analysis for a named entity or node.

  Equivalent to simulate/2 but takes a name directly instead of a natural-language scenario.
  """
  @spec impact_analysis(String.t(), keyword()) :: {:ok, impact()} | {:error, term()}
  def impact_analysis(entity_or_node, opts \\ []) when is_binary(entity_or_node) do
    scenario = build_impact_scenario(entity_or_node)
    simulate(scenario, opts)
  end

  # ---------------------------------------------------------------------------
  # GenServer Callbacks
  # ---------------------------------------------------------------------------

  @impl true
  def init(_opts) do
    Logger.info("[Simulator] Ready")
    # node_totals is cached per workspace: %{workspace_id => %{node => count}}
    {:ok, %{node_totals: %{}}}
  end

  @impl true
  def handle_call({:simulate, scenario, opts}, _from, state) do
    max_depth = Keyword.get(opts, :max_depth, @default_max_depth)
    ws = Keyword.get(opts, :workspace_id, "default")

    result =
      with {:ok, mutation} <- parse_scenario(scenario),
           {:ok, seed_ids} <- gather_seed_contexts(mutation, ws),
           {:ok, traversal} <- traverse_graph(seed_ids, max_depth, ws),
           {:ok, node_totals} <- get_or_fetch_node_totals(state.node_totals, ws),
           {:ok, scored_nodes} <- score_nodes(traversal, node_totals),
           {:ok, critical_deps} <-
             find_critical_dependencies(mutation, traversal, node_totals, ws),
           {:ok, risk} <- assess_risk(scored_nodes, critical_deps, traversal),
           {:ok, recs} <- generate_recommendations(mutation, scored_nodes, critical_deps) do
        report = %{
          scenario: scenario,
          mutation_type: mutation.type,
          affected_entities: build_affected_entities(mutation, traversal),
          affected_nodes: scored_nodes,
          critical_dependencies: critical_deps,
          risk_assessment: risk,
          recommendations: recs,
          total_contexts_affected: count_unique_context_ids(traversal),
          graph_depth: max_depth
        }

        {:ok, report}
      end

    new_state =
      case result do
        {:ok, _} ->
          # Refresh this workspace's node totals cache on each successful run
          case fetch_node_totals(ws) do
            {:ok, totals} -> %{state | node_totals: Map.put(state.node_totals, ws, totals)}
            _ -> state
          end

        _ ->
          state
      end

    {:reply, result, new_state}
  end

  # ---------------------------------------------------------------------------
  # Private: Scenario Parsing
  # ---------------------------------------------------------------------------

  defp parse_scenario(scenario) do
    downcased = String.downcase(scenario)

    mutation =
      if Ollama.available?() do
        llm_parse_scenario(scenario)
      else
        regex_parse_scenario(scenario, downcased)
      end

    {:ok, mutation}
  end

  defp llm_parse_scenario(scenario) do
    prompt = """
    Parse this "what if" scenario for an Elixir cognitive OS knowledge graph.

    Domain nodes: #{Enum.join(@domain_nodes, ", ")}
    Known people: Alice, Carol, Alice, Bob, Quinn, Erin, Ruth, Frank, Nina, Oscar, Dan, Sam, Tina, Ivan, Grace, Judy, Victor, Peggy, Uma

    Scenario: "#{scenario}"

    Respond with JSON only (no prose):
    {
      "type": "entity_removal|node_cancellation|revenue_change|dependency_break|general",
      "entity": "person name if applicable or null",
      "node": "node id if applicable or null"
    }
    """

    case Ollama.generate(prompt, system: "Output valid JSON only, no explanation.") do
      {:ok, raw} -> parse_llm_mutation(raw, scenario)
      {:error, _} -> regex_parse_scenario(scenario, String.downcase(scenario))
    end
  rescue
    _ -> regex_parse_scenario(scenario, String.downcase(scenario))
  end

  defp parse_llm_mutation(raw, original_scenario) do
    json =
      case Regex.run(~r/\{[\s\S]*\}/u, raw) do
        [json | _] -> json
        nil -> raw
      end

    case Jason.decode(json) do
      {:ok, %{"type" => type_str} = decoded} ->
        %{
          type: parse_mutation_type(type_str),
          entity: decoded["entity"],
          node: decoded["node"],
          raw_scenario: original_scenario
        }

      _ ->
        regex_parse_scenario(original_scenario, String.downcase(original_scenario))
    end
  end

  defp regex_parse_scenario(scenario, downcased) do
    cond do
      # Entity removal patterns: "Ed leaves", "Alice leaves", "without Ed"
      Regex.match?(
        ~r/\bleav(e|es|ing)\b|\bdeparture\b|\bwithout\b|\bfire[sd]?\b|\bremove[sd]?\b/,
        downcased
      ) ->
        entity = extract_entity_from_scenario(scenario)
        %{type: :entity_removal, entity: entity, node: nil, raw_scenario: scenario}

      # Node cancellation patterns: "cancel AA", "shut down operation-delivery", "drop project-platform-launch"
      Regex.match?(
        ~r/\bcancel\b|\bshut down\b|\bdrop\b|\bend\b|\bkill\b|\bclose\b|\bdissolve\b/,
        downcased
      ) ->
        node = extract_node_from_scenario(downcased)
        %{type: :node_cancellation, entity: nil, node: node, raw_scenario: scenario}

      # Revenue change patterns
      Regex.match?(~r/\brevenue\b|\bmoney\b|\bdrop[s]?\b|\blose\b|\bcut\b|\bincome\b/, downcased) ->
        %{type: :revenue_change, entity: nil, node: "operation-revenue", raw_scenario: scenario}

      # Dependency break: "what if X stops working on Y"
      Regex.match?(~r/\bstops?\b|\bno longer\b|\bpull[s]? out\b|\bbreaks?\b/, downcased) ->
        entity = extract_entity_from_scenario(scenario)
        node = extract_node_from_scenario(downcased)
        %{type: :dependency_break, entity: entity, node: node, raw_scenario: scenario}

      true ->
        # General — attempt to extract any entity or node hint
        entity = extract_entity_from_scenario(scenario)
        node = extract_node_from_scenario(downcased)
        %{type: :general, entity: entity, node: node, raw_scenario: scenario}
    end
  end

  defp parse_mutation_type("entity_removal"), do: :entity_removal
  defp parse_mutation_type("node_cancellation"), do: :node_cancellation
  defp parse_mutation_type("revenue_change"), do: :revenue_change
  defp parse_mutation_type("dependency_break"), do: :dependency_break
  defp parse_mutation_type(_), do: :general

  # Extract person name: look for capitalized words that aren't question words
  @question_words ~w[What If We Cancel Agency The A An Is Are Was Were How Why When Where Who Which]

  defp extract_entity_from_scenario(scenario) do
    words =
      scenario
      |> String.split(~r/\s+/)
      |> Enum.filter(fn w ->
        stripped = String.replace(w, ~r/[^A-Za-z']/, "")

        String.length(stripped) > 1 and
          Regex.match?(~r/^[A-Z]/, stripped) and
          stripped not in @question_words
      end)
      |> Enum.map(&String.replace(&1, ~r/[^A-Za-z' ]/, ""))

    case words do
      [] -> nil
      [single] -> single
      multiple -> Enum.join(multiple, " ")
    end
  end

  defp extract_node_from_scenario(downcased) do
    node_aliases = %{
      "operation-delivery" => "operation-delivery",
      "project-platform-launch" => "project-platform-launch",
      "platform launch" => "project-platform-launch",
      "product-customer-portal" => "product-customer-portal",
      "customer portal" => "product-customer-portal",
      "entity-company" => "entity-company",
      "example company" => "entity-company",
      "learning-research-library" => "learning-research-library",
      "research library" => "learning-research-library",
      "team" => "team",
      "revenue operations" => "operation-revenue",
      "operation-revenue" => "operation-revenue",
      "operator" => "operator"
    }

    Enum.find_value(node_aliases, fn {alias, node_id} ->
      if String.contains?(downcased, alias), do: node_id
    end)
  end

  # ---------------------------------------------------------------------------
  # Private: Seed Context Gathering
  # ---------------------------------------------------------------------------

  defp gather_seed_contexts(%{type: :entity_removal, entity: entity}, ws)
       when is_binary(entity) do
    # Fuzzy match: the entity name may be partial (e.g. "Ed" → "Alice").
    # Workspace membership resolved via contexts (source of truth).
    sql = """
    SELECT DISTINCT c.id FROM contexts c
    JOIN entities e ON c.id = e.context_id
    WHERE e.name LIKE ?1 AND c.workspace_id = ?2
    """

    like_pattern = "%" <> String.trim(entity) <> "%"

    case Store.raw_query(sql, [like_pattern, ws]) do
      {:ok, rows} ->
        ids = Enum.map(rows, fn [id] -> id end)
        Logger.debug("[Simulator] Seed contexts for entity '#{entity}': #{length(ids)}")
        {:ok, ids}

      err ->
        err
    end
  end

  defp gather_seed_contexts(%{type: :node_cancellation, node: node}, ws) when is_binary(node) do
    sql = "SELECT id FROM contexts WHERE node = ?1 AND workspace_id = ?2"

    case Store.raw_query(sql, [node, ws]) do
      {:ok, rows} ->
        ids = Enum.map(rows, fn [id] -> id end)
        Logger.debug("[Simulator] Seed contexts for node '#{node}': #{length(ids)}")
        {:ok, ids}

      err ->
        err
    end
  end

  defp gather_seed_contexts(%{type: :revenue_change}, ws) do
    sql = "SELECT id FROM contexts WHERE node = 'operation-revenue' AND workspace_id = ?1"

    case Store.raw_query(sql, [ws]) do
      {:ok, rows} -> {:ok, Enum.map(rows, fn [id] -> id end)}
      err -> err
    end
  end

  defp gather_seed_contexts(%{type: :dependency_break, entity: entity, node: node}, ws) do
    # Combine entity + node seeds
    entity_ids =
      if is_binary(entity) do
        case gather_seed_contexts(%{type: :entity_removal, entity: entity}, ws) do
          {:ok, ids} -> ids
          _ -> []
        end
      else
        []
      end

    node_ids =
      if is_binary(node) do
        case gather_seed_contexts(%{type: :node_cancellation, node: node}, ws) do
          {:ok, ids} -> ids
          _ -> []
        end
      else
        []
      end

    {:ok, Enum.uniq(entity_ids ++ node_ids)}
  end

  defp gather_seed_contexts(%{type: :general, entity: entity, node: node}, ws) do
    entity_ids =
      if is_binary(entity) do
        case gather_seed_contexts(%{type: :entity_removal, entity: entity}, ws) do
          {:ok, ids} -> ids
          _ -> []
        end
      else
        []
      end

    node_ids =
      if is_binary(node) do
        case gather_seed_contexts(%{type: :node_cancellation, node: node}, ws) do
          {:ok, ids} -> ids
          _ -> []
        end
      else
        []
      end

    all_ids = Enum.uniq(entity_ids ++ node_ids)

    if all_ids == [] do
      # No specific seeds found — return all contexts as seeds (general exploration)
      case Store.raw_query("SELECT id FROM contexts WHERE workspace_id = ?1 LIMIT 50", [ws]) do
        {:ok, rows} -> {:ok, Enum.map(rows, fn [id] -> id end)}
        err -> err
      end
    else
      {:ok, all_ids}
    end
  end

  # ---------------------------------------------------------------------------
  # Private: Graph Traversal (BFS)
  # ---------------------------------------------------------------------------

  # traversal entry: %{context_id, depth, relation, weight}
  defp traverse_graph(seed_ids, max_depth, ws) do
    # seed_ids are direct (depth 1)
    initial =
      Enum.map(seed_ids, fn id ->
        %{context_id: id, depth: 1, relation: "seed", weight: 1.0}
      end)

    visited = MapSet.new(seed_ids)
    traversal = bfs(initial, visited, max_depth, initial, ws)
    {:ok, traversal}
  end

  # Returns the accumulated list of traversal entries (plain list, not {:ok, ...})
  defp bfs([], _visited, _max_depth, acc, _ws), do: acc

  defp bfs(frontier, visited, max_depth, acc, ws) do
    # Only expand nodes that haven't hit max depth
    expandable = Enum.filter(frontier, fn e -> e.depth < max_depth end)

    if expandable == [] do
      acc
    else
      {next_frontier, next_visited, new_entries} =
        Enum.reduce(expandable, {[], visited, []}, fn entry, {frontier_acc, vis_acc, new_acc} ->
          case fetch_neighbors(entry.context_id, ws) do
            {:ok, neighbors} ->
              unvisited =
                Enum.reject(neighbors, fn n -> MapSet.member?(vis_acc, n.neighbor_id) end)

              next_entries =
                Enum.map(unvisited, fn n ->
                  %{
                    context_id: n.neighbor_id,
                    depth: entry.depth + 1,
                    relation: n.relation,
                    weight: n.weight
                  }
                end)

              next_vis =
                Enum.reduce(unvisited, vis_acc, fn n, v -> MapSet.put(v, n.neighbor_id) end)

              {frontier_acc ++ next_entries, next_vis, new_acc ++ next_entries}

            _ ->
              {frontier_acc, vis_acc, new_acc}
          end
        end)

      bfs(next_frontier, next_visited, max_depth, acc ++ new_entries, ws)
    end
  end

  defp fetch_neighbors(context_id, ws) do
    sql = """
    SELECT source_id, target_id, relation, weight
    FROM edges
    WHERE (source_id = ?1 OR target_id = ?1) AND workspace_id = ?2
    """

    case Store.raw_query(sql, [context_id, ws]) do
      {:ok, rows} ->
        neighbors =
          rows
          |> Enum.flat_map(fn [src, tgt, rel, wt] ->
            neighbor_id = if src == context_id, do: tgt, else: src
            # Only traverse to context IDs (32-char hex), not entity/node names
            if looks_like_context_id?(neighbor_id) do
              [%{neighbor_id: neighbor_id, relation: rel, weight: wt || 1.0}]
            else
              []
            end
          end)
          |> Enum.uniq_by(& &1.neighbor_id)

        {:ok, neighbors}

      err ->
        err
    end
  end

  # Context IDs in this system are 32-character hex strings (MD5 of path)
  defp looks_like_context_id?(str) when is_binary(str) do
    byte_size(str) == 32 and Regex.match?(~r/^[a-f0-9]+$/i, str)
  end

  defp looks_like_context_id?(_), do: false

  # ---------------------------------------------------------------------------
  # Private: Node Scoring
  # ---------------------------------------------------------------------------

  defp get_or_fetch_node_totals(cache, ws) do
    case Map.get(cache, ws) do
      totals when is_map(totals) and map_size(totals) > 0 -> {:ok, totals}
      _ -> fetch_node_totals(ws)
    end
  end

  defp fetch_node_totals(ws) do
    sql = "SELECT node, COUNT(*) FROM contexts WHERE workspace_id = ?1 GROUP BY node"

    case Store.raw_query(sql, [ws]) do
      {:ok, rows} ->
        totals = Map.new(rows, fn [node, count] -> {node, count} end)
        {:ok, totals}

      err ->
        err
    end
  end

  defp score_nodes(traversal, node_totals) do
    # Group traversal entries by node, fetching each context's node from the DB
    context_ids = Enum.map(traversal, & &1.context_id) |> Enum.uniq()

    # Batch-fetch node info for all affected context IDs
    {context_nodes, context_nodes_map} = fetch_context_nodes(context_ids)

    # Build per-node stats
    node_stats =
      Enum.reduce(traversal, %{}, fn entry, acc ->
        node = Map.get(context_nodes_map, entry.context_id)

        if is_binary(node) do
          depth_decay = Map.get(@depth_decay, entry.depth, 0.25)
          rel_weight = Map.get(@edge_weights, entry.relation, 1.0)
          score_contribution = rel_weight * depth_decay

          Map.update(acc, node, %{score: score_contribution, count: 1}, fn existing ->
            %{
              score: existing.score + score_contribution,
              count: existing.count + 1
            }
          end)
        else
          acc
        end
      end)

    scored =
      node_stats
      |> Enum.map(fn {node, %{score: raw_score, count: count}} ->
        total = Map.get(node_totals, node, 1)
        fraction = min(count / max(total, 1), 1.0)
        # Normalize: fraction of node affected, weighted by edge quality
        normalized = Float.round(fraction * min(raw_score / max(count, 1), 1.0) * 10.0, 3)

        %{
          node: node,
          impact_score: normalized,
          context_count: count,
          description: describe_node_impact(node, count, total)
        }
      end)
      |> Enum.sort_by(& &1.impact_score, :desc)

    _ = context_nodes
    {:ok, scored}
  end

  defp fetch_context_nodes(context_ids) when context_ids == [], do: {[], %{}}

  defp fetch_context_nodes(context_ids) do
    # SQLite has a limit on IN clause params; chunk if needed
    chunk_size = 200

    pairs =
      context_ids
      |> Enum.chunk_every(chunk_size)
      |> Enum.flat_map(fn chunk ->
        placeholders = Enum.map_join(1..length(chunk), ", ", fn i -> "?#{i}" end)
        sql = "SELECT id, node FROM contexts WHERE id IN (#{placeholders})"

        case Store.raw_query(sql, chunk) do
          {:ok, rows} -> rows
          _ -> []
        end
      end)

    map = Map.new(pairs, fn [id, node] -> {id, node} end)
    {pairs, map}
  end

  defp describe_node_impact(_node, affected, total) do
    pct = trunc(affected / max(total, 1) * 100)
    "#{affected}/#{total} contexts affected (#{pct}%)"
  end

  # ---------------------------------------------------------------------------
  # Private: Critical Dependencies
  # ---------------------------------------------------------------------------

  defp find_critical_dependencies(%{entity: entity}, traversal, node_totals, ws)
       when is_binary(entity) do
    # A critical dependency exists when the entity appears in >50% of a node's contexts
    sql = """
    SELECT c.node, COUNT(*) as cnt
    FROM contexts c
    JOIN entities e ON c.id = e.context_id
    WHERE e.name LIKE ?1 AND c.workspace_id = ?2
    GROUP BY c.node
    """

    like_pattern = "%" <> String.trim(entity) <> "%"

    case Store.raw_query(sql, [like_pattern, ws]) do
      {:ok, rows} ->
        deps =
          rows
          |> Enum.filter(fn [node, count] ->
            total = Map.get(node_totals, node, 1)
            count / max(total, 1) > 0.5
          end)
          |> Enum.map(fn [node, count] ->
            total = Map.get(node_totals, node, 1)

            %{
              from: entity,
              to: node,
              relation: "mentioned_in",
              weight: Float.round(count / max(total, 1), 3)
            }
          end)

        # Also check for cross-node bridging via traversal
        bridge_deps = find_bridge_dependencies(entity, traversal)

        {:ok, Enum.uniq_by(deps ++ bridge_deps, fn d -> {d.from, d.to} end)}

      err ->
        err
    end
  end

  defp find_critical_dependencies(%{node: node}, _traversal, node_totals, ws)
       when is_binary(node) do
    # For a node, critical deps are cross_ref edges pointing to other nodes
    sql = """
    SELECT DISTINCT c.node as src_node, e.target_id as target
    FROM edges e
    JOIN contexts c ON c.id = e.source_id
    WHERE c.node = ?1 AND e.relation = 'cross_ref' AND c.workspace_id = ?2
    """

    case Store.raw_query(sql, [node, ws]) do
      {:ok, rows} ->
        deps =
          Enum.map(rows, fn [src_node, target] ->
            total = Map.get(node_totals, src_node, 1)

            %{
              from: src_node,
              to: target,
              relation: "cross_ref",
              weight: Float.round(1.0 / max(total, 1), 3)
            }
          end)

        {:ok, deps}

      err ->
        err
    end
  end

  defp find_critical_dependencies(_mutation, _traversal, _node_totals, _ws), do: {:ok, []}

  defp find_bridge_dependencies(entity, traversal) do
    # Contexts that appear in 2+ different nodes via this entity's traversal
    context_ids = Enum.map(traversal, & &1.context_id)

    if context_ids == [] do
      []
    else
      chunk_size = 200

      node_pairs =
        context_ids
        |> Enum.chunk_every(chunk_size)
        |> Enum.flat_map(fn chunk ->
          placeholders = Enum.map_join(1..length(chunk), ", ", fn i -> "?#{i}" end)
          sql = "SELECT id, node FROM contexts WHERE id IN (#{placeholders})"

          case Store.raw_query(sql, chunk) do
            {:ok, rows} -> rows
            _ -> []
          end
        end)

      nodes_touched =
        node_pairs
        |> Enum.map(fn [_id, node] -> node end)
        |> Enum.uniq()

      if length(nodes_touched) > 1 do
        # Entity bridges multiple nodes — that's a cross-cutting critical dep
        Enum.map(nodes_touched, fn node ->
          %{
            from: entity,
            to: node,
            relation: "cross_cutting",
            weight: 0.8
          }
        end)
      else
        []
      end
    end
  end

  # ---------------------------------------------------------------------------
  # Private: Risk Assessment
  # ---------------------------------------------------------------------------

  defp assess_risk(scored_nodes, critical_deps, traversal) do
    total_affected = count_unique_context_ids(traversal)
    critical_count = length(critical_deps)
    nodes_affected = length(scored_nodes)
    revenue_affected = Enum.any?(scored_nodes, fn n -> n.node == "operation-revenue" end)
    high_impact_nodes = Enum.count(scored_nodes, fn n -> n.impact_score > 0.5 end)

    severity =
      cond do
        critical_count >= 3 or (revenue_affected and high_impact_nodes >= 2) ->
          :critical

        critical_count >= 1 or nodes_affected >= 5 or (revenue_affected and total_affected > 20) ->
          :high

        nodes_affected >= 3 or total_affected > 15 ->
          :medium

        true ->
          :low
      end

    reasoning =
      build_risk_reasoning(
        severity,
        total_affected,
        nodes_affected,
        critical_count,
        revenue_affected
      )

    {:ok, %{severity: severity, reasoning: reasoning}}
  end

  defp build_risk_reasoning(severity, total, nodes, critical, revenue) do
    parts = ["#{total} contexts affected across #{nodes} node(s)"]

    if critical > 0 do
      parts = parts ++ ["#{critical} critical single-point-of-failure dependencies identified"]
      parts
    else
      parts
    end
    |> then(fn p ->
      if revenue, do: p ++ ["operation-revenue node is in the blast radius"], else: p
    end)
    |> then(fn p ->
      severity_label =
        case severity do
          :critical -> "CRITICAL — immediate action required"
          :high -> "HIGH — significant disruption expected"
          :medium -> "MEDIUM — manageable with planning"
          :low -> "LOW — limited downstream impact"
        end

      [severity_label | p]
    end)
    |> Enum.join(". ")
  end

  # ---------------------------------------------------------------------------
  # Private: Recommendations
  # ---------------------------------------------------------------------------

  defp generate_recommendations(mutation, scored_nodes, critical_deps) do
    recs =
      if Ollama.available?() do
        llm_recommendations(mutation, scored_nodes, critical_deps)
      else
        fallback_recommendations(mutation, scored_nodes, critical_deps)
      end

    {:ok, recs}
  end

  defp llm_recommendations(mutation, scored_nodes, critical_deps) do
    top_nodes = scored_nodes |> Enum.take(5) |> Enum.map(& &1.node) |> Enum.join(", ")
    dep_summary = Enum.map(critical_deps, fn d -> "#{d.from} → #{d.to}" end) |> Enum.join(", ")

    entity_str = if mutation.entity, do: "Entity: #{mutation.entity}", else: ""
    node_str = if mutation.node, do: "Node: #{mutation.node}", else: ""

    prompt = """
    Generate 3-5 actionable recommendations for this scenario in a cognitive operating system.

    Mutation type: #{mutation.type}
    #{entity_str}
    #{node_str}
    Top affected nodes: #{top_nodes}
    Critical dependencies: #{if dep_summary == "", do: "none", else: dep_summary}

    Return a JSON array of strings. Each recommendation should be concrete and actionable.
    Example: ["Redistribute Alice's Platform Launch responsibilities to Bob and Quinn", ...]
    """

    case Ollama.generate(prompt, system: "Output valid JSON array only.") do
      {:ok, raw} ->
        case Regex.run(~r/\[[\s\S]*\]/u, raw) do
          [json | _] ->
            case Jason.decode(json) do
              {:ok, list} when is_list(list) -> Enum.filter(list, &is_binary/1)
              _ -> fallback_recommendations(mutation, scored_nodes, critical_deps)
            end

          nil ->
            fallback_recommendations(mutation, scored_nodes, critical_deps)
        end

      _ ->
        fallback_recommendations(mutation, scored_nodes, critical_deps)
    end
  rescue
    _ -> fallback_recommendations(mutation, scored_nodes, critical_deps)
  end

  defp fallback_recommendations(%{type: :entity_removal, entity: entity}, scored_nodes, _deps)
       when is_binary(entity) do
    top_nodes = scored_nodes |> Enum.take(3) |> Enum.map(& &1.node)

    base = ["Document #{entity}'s responsibilities and handoff all active work before departure"]

    node_recs =
      Enum.map(top_nodes, fn node ->
        "Review #{node} for contexts dependent on #{entity} and assign a replacement owner"
      end)

    knowledge_rec = "Export #{entity}'s key decisions and context patterns to a reference document"

    [base | node_recs] |> List.flatten() |> Kernel.++([knowledge_rec]) |> Enum.take(5)
  end

  defp fallback_recommendations(%{type: :node_cancellation, node: node}, scored_nodes, _deps)
       when is_binary(node) do
    top_nodes =
      scored_nodes
      |> Enum.reject(fn n -> n.node == node end)
      |> Enum.take(3)
      |> Enum.map(& &1.node)

    [
      "Archive all #{node} contexts to prevent knowledge loss",
      "Notify all stakeholders with active contexts in #{node}",
      "Migrate cross-referenced signals from #{node} to relevant active nodes"
      | Enum.map(top_nodes, fn n ->
          "Update #{n} contexts that reference #{node} to reflect the cancellation"
        end)
    ]
    |> Enum.take(5)
  end

  defp fallback_recommendations(%{type: :revenue_change}, scored_nodes, _deps) do
    top_nodes = scored_nodes |> Enum.take(3) |> Enum.map(& &1.node)

    base = [
      "Immediately update operation-revenue context with revised projections",
      "Audit all active revenue pipelines and re-prioritize based on new reality"
    ]

    node_recs =
      Enum.map(top_nodes, fn n ->
        "Reassess #{n} budget and resource allocation in light of revenue change"
      end)

    (base ++ node_recs) |> Enum.take(5)
  end

  defp fallback_recommendations(
         %{type: :dependency_break, entity: entity, node: node},
         scored_nodes,
         _deps
       ) do
    top_nodes = scored_nodes |> Enum.take(3) |> Enum.map(& &1.node)

    [
      if(entity, do: "Identify replacement for #{entity}'s role", else: nil),
      if(node, do: "Review dependency on #{node} and identify fallback paths", else: nil),
      "Map all contexts that will break and prioritize remediation"
      | Enum.map(top_nodes, fn n -> "Audit #{n} for broken dependencies" end)
    ]
    |> Enum.filter(&is_binary/1)
    |> Enum.take(5)
  end

  defp fallback_recommendations(%{entity: entity, node: node}, scored_nodes, _deps) do
    top_nodes = scored_nodes |> Enum.take(3) |> Enum.map(& &1.node)

    subject = entity || node || "this change"

    ["Audit all affected contexts for dependencies on #{subject}"]
    |> Kernel.++(Enum.map(top_nodes, fn n -> "Review #{n} node for exposure to #{subject}" end))
    |> Kernel.++(["Update signal files in affected nodes to reflect new state"])
    |> Enum.take(5)
  end

  # ---------------------------------------------------------------------------
  # Private: Report Helpers
  # ---------------------------------------------------------------------------

  defp build_affected_entities(%{entity: entity}, traversal) when is_binary(entity) do
    direct_ids = traversal |> Enum.filter(fn e -> e.depth == 1 end) |> Enum.map(& &1.context_id)

    [
      %{
        name: entity,
        impact: :direct,
        contexts: length(direct_ids)
      }
    ]
  end

  defp build_affected_entities(_mutation, traversal) do
    # Extract unique entities from traversal context titles via entity table
    context_ids = Enum.map(traversal, & &1.context_id) |> Enum.uniq()

    if context_ids == [] do
      []
    else
      chunk_size = 200

      entity_rows =
        context_ids
        |> Enum.chunk_every(chunk_size)
        |> Enum.flat_map(fn chunk ->
          placeholders = Enum.map_join(1..length(chunk), ", ", fn i -> "?#{i}" end)

          sql = """
          SELECT e.name, c.id
          FROM entities e
          JOIN contexts c ON c.id = e.context_id
          WHERE c.id IN (#{placeholders})
          """

          case Store.raw_query(sql, chunk) do
            {:ok, rows} -> rows
            _ -> []
          end
        end)

      entity_rows
      |> Enum.group_by(fn [name, _id] -> name end)
      |> Enum.map(fn {name, rows} ->
        depth_of_first =
          traversal
          |> Enum.find(fn e -> e.context_id == List.last(hd(rows)) end)
          |> then(fn e -> if e, do: e.depth, else: 2 end)

        %{
          name: name,
          impact: if(depth_of_first == 1, do: :direct, else: :indirect),
          contexts: length(rows)
        }
      end)
      |> Enum.sort_by(& &1.contexts, :desc)
      |> Enum.take(10)
    end
  end

  defp count_unique_context_ids(traversal) do
    traversal |> Enum.map(& &1.context_id) |> Enum.uniq() |> length()
  end

  # ---------------------------------------------------------------------------
  # Private: Utility
  # ---------------------------------------------------------------------------

  defp build_impact_scenario(entity_or_node) do
    if entity_or_node in @domain_nodes do
      "What if we cancel #{entity_or_node}?"
    else
      "What if #{entity_or_node} leaves?"
    end
  end
end
