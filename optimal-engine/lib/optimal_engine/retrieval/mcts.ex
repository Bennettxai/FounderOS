defmodule OptimalEngine.Retrieval.MCTS do
  @moduledoc """
  Budget-aware context-package assembly via Monte Carlo Tree Search.

  Classic FTS/RRF retrieval hands you a *ranked* candidate list. Greedy
  packing of that list (take highest score until the budget fills) is
  myopic: it ignores that two top-ranked items may be near-duplicates,
  burning budget on redundant coverage. MCTS searches the combinatorial
  space of *subsets* that fit the token budget and maximizes **coverage**
  (distinct concepts) rather than raw summed relevance.

  ## Algorithm

  Each tree node is a partial selection (a set of chosen candidate ids).
  Children add one not-yet-chosen candidate that still fits the budget.

  - **Selection**  — descend by UCT until a node with unexpanded children.
  - **Expansion**  — add one random affordable child.
  - **Rollout**    — greedily fill the remaining budget by marginal reward
                     (relevance * novelty), simulating to a terminal set.
  - **Backprop**   — propagate the terminal reward up the visited path.

  After `iterations`, the best selection seen (highest terminal reward) is
  returned — not just the most-visited root child, so a single lucky
  high-coverage rollout is never lost.

  ## Reward

      reward(set) = Σ relevance_i  +  λ * coverage(set)

  where `coverage` counts distinct concept tokens contributed across the
  set (novelty), and `relevance_i` is the candidate's fused score. This
  rewards covering *new* ground over piling on redundant high scorers.

  ## Candidates

  Each candidate is a map carrying at least:

      %{id: term, score: float, content: String.t() | nil, tokens: integer | nil}

  `:tokens` is optional; falls back to the 4-chars-per-token heuristic on
  `:content`. `:concepts` (a list/MapSet of tokens) is optional; falls back
  to tokenizing `:content`/`:l0_abstract`.

  ## Configuration

      config :optimal_engine, :retrieval,
        mcts_enabled: true,
        mcts_iterations: 200,
        mcts_exploration: 1.41

  When `mcts_enabled` is false the caller should fall back to greedy
  packing; this module also exposes `greedy/2` for that path and so tests
  can compare the two.
  """

  @chars_per_token 4
  @default_iterations 200
  @default_exploration 1.41
  # weight on coverage (novelty) relative to summed relevance
  @coverage_lambda 0.5

  @type candidate :: %{
          required(:id) => term(),
          optional(:score) => number(),
          optional(:content) => String.t() | nil,
          optional(:tokens) => non_neg_integer(),
          optional(:concepts) => Enum.t()
        }

  @type result :: %{
          selected: [candidate()],
          used_tokens: non_neg_integer(),
          budget: non_neg_integer(),
          reward: float(),
          strategy: :mcts | :greedy
        }

  # ---------------------------------------------------------------------------
  # Public API
  # ---------------------------------------------------------------------------

  @doc """
  Select a subset of `candidates` maximizing coverage within `budget`
  tokens using MCTS. Returns a `result` map.

  Options:
    * `:iterations`  — MCTS rollouts (default from config / #{@default_iterations})
    * `:exploration` — UCT constant (default from config / #{@default_exploration})
    * `:coverage_lambda` — coverage weight (default #{@coverage_lambda})
  """
  @spec select([candidate()], non_neg_integer(), keyword()) :: result()
  def select(candidates, budget, opts \\ [])

  def select([], budget, _opts) do
    %{selected: [], used_tokens: 0, budget: budget, reward: 0.0, strategy: :mcts}
  end

  def select(candidates, budget, opts) when is_list(candidates) and budget >= 0 do
    cfg = Application.get_env(:optimal_engine, :retrieval, [])

    iterations =
      Keyword.get(opts, :iterations, Keyword.get(cfg, :mcts_iterations, @default_iterations))

    c = Keyword.get(opts, :exploration, Keyword.get(cfg, :mcts_exploration, @default_exploration))
    lambda = Keyword.get(opts, :coverage_lambda, @coverage_lambda)

    cands = normalize(candidates)
    # Affordable singletons only — anything bigger than budget can never be chosen.
    affordable = Enum.reject(cands, fn cand -> cand.tokens > budget end)

    if affordable == [] do
      %{selected: [], used_tokens: 0, budget: budget, reward: 0.0, strategy: :mcts}
    else
      run_mcts(affordable, budget, iterations, c, lambda)
    end
  end

  @doc """
  Greedy baseline: pack by descending score until the budget is exhausted.
  Used as the fallback when `mcts_enabled` is false and as a comparison
  baseline in tests.
  """
  @spec greedy([candidate()], non_neg_integer(), keyword()) :: result()
  def greedy(candidates, budget, opts \\ []) when is_list(candidates) and budget >= 0 do
    lambda = Keyword.get(opts, :coverage_lambda, @coverage_lambda)
    cands = normalize(candidates)

    {selected_rev, used} =
      cands
      |> Enum.sort_by(& &1.score, :desc)
      |> Enum.reduce({[], 0}, fn cand, {sel, tokens} ->
        if tokens + cand.tokens <= budget do
          {[cand | sel], tokens + cand.tokens}
        else
          {sel, tokens}
        end
      end)

    selected = Enum.reverse(selected_rev)

    %{
      selected: selected,
      used_tokens: used,
      budget: budget,
      reward: reward(selected, lambda),
      strategy: :greedy
    }
  end

  @doc "Whether MCTS selection is enabled via config (default true)."
  @spec enabled?() :: boolean()
  def enabled? do
    Application.get_env(:optimal_engine, :retrieval, [])
    |> Keyword.get(:mcts_enabled, true)
  end

  @doc """
  Reward of a selected set: summed relevance + λ * distinct-concept coverage.
  Exposed for tests and for the assembler to compare strategies.
  """
  @spec reward([candidate()], float()) :: float()
  def reward(selected, lambda \\ @coverage_lambda) do
    relevance = selected |> Enum.map(& &1.score) |> Enum.sum()
    coverage = selected |> covered_concepts() |> MapSet.size()
    relevance + lambda * coverage
  end

  # ---------------------------------------------------------------------------
  # Private: MCTS core
  # ---------------------------------------------------------------------------

  # Tree node: %{key, selected (ids set), used, children (key=>node), visits, total, untried (ids)}
  defp run_mcts(cands, budget, iterations, c, lambda) do
    by_id = Map.new(cands, &{&1.id, &1})
    all_ids = Enum.map(cands, & &1.id)

    root = new_node([], 0, all_ids)

    {_tree, best_set, best_reward} =
      Enum.reduce(1..iterations, {root, [], -1.0}, fn _i, {tree, best, best_r} ->
        {tree, terminal_ids, r} = iterate(tree, by_id, budget, c, lambda)

        if r > best_r do
          {tree, terminal_ids, r}
        else
          {tree, best, best_r}
        end
      end)

    selected = Enum.map(best_set, &Map.fetch!(by_id, &1))
    used = selected |> Enum.map(& &1.tokens) |> Enum.sum()

    %{
      selected: selected,
      used_tokens: used,
      budget: budget,
      reward: best_reward,
      strategy: :mcts
    }
  end

  defp new_node(selected_ids, used, untried) do
    %{
      selected: selected_ids,
      used: used,
      children: %{},
      visits: 0,
      total: 0.0,
      untried: untried
    }
  end

  # One MCTS iteration: returns updated tree + the terminal selection + reward.
  defp iterate(node, by_id, budget, c, lambda) do
    {node, terminal_ids, reward} = tree_policy(node, by_id, budget, c, lambda)
    {node, terminal_ids, reward}
  end

  # Recursive selection + expansion + rollout + backprop.
  defp tree_policy(node, by_id, budget, c, lambda) do
    affordable_untried =
      Enum.filter(node.untried, fn id ->
        node.used + by_id[id].tokens <= budget
      end)

    cond do
      # Expandable: add one untried affordable child, roll out from it.
      affordable_untried != [] ->
        id = Enum.random(affordable_untried)
        cand = by_id[id]
        child_selected = [id | node.selected]
        child_used = node.used + cand.tokens
        remaining_untried = List.delete(node.untried, id)
        child = new_node(child_selected, child_used, remaining_untried)

        {terminal_ids, reward} =
          rollout(child_selected, child_used, remaining_untried, by_id, budget, lambda)

        child = backprop_local(child, reward)

        node =
          node
          |> Map.update!(:untried, &List.delete(&1, id))
          |> put_in([:children, id], child)
          |> backprop_local(reward)

        {node, terminal_ids, reward}

      # Fully expanded with children: descend by UCT.
      map_size(node.children) > 0 ->
        {best_id, best_child} = best_uct_child(node, c)
        {updated_child, terminal_ids, reward} = tree_policy(best_child, by_id, budget, c, lambda)

        node =
          node
          |> put_in([:children, best_id], updated_child)
          |> backprop_local(reward)

        {node, terminal_ids, reward}

      # Terminal leaf (nothing affordable to add).
      true ->
        reward = reward(Enum.map(node.selected, &by_id[&1]), lambda)
        {backprop_local(node, reward), node.selected, reward}
    end
  end

  defp backprop_local(node, reward) do
    %{node | visits: node.visits + 1, total: node.total + reward}
  end

  defp best_uct_child(node, c) do
    parent_visits = max(node.visits, 1)

    node.children
    |> Enum.max_by(fn {_id, child} ->
      exploit = child.total / max(child.visits, 1)
      explore = c * :math.sqrt(:math.log(parent_visits + 1) / max(child.visits, 1))
      exploit + explore
    end)
  end

  # Greedy rollout: from a partial selection, fill remaining budget by
  # marginal reward (relevance + novelty of new concepts), to a terminal set.
  defp rollout(selected_ids, used, untried, by_id, budget, lambda) do
    covered = covered_concepts(Enum.map(selected_ids, &by_id[&1]))

    {final_ids, _used, _covered} =
      do_rollout(selected_ids, used, covered, untried, by_id, budget, lambda)

    reward = reward(Enum.map(final_ids, &by_id[&1]), lambda)
    {final_ids, reward}
  end

  defp do_rollout(selected_ids, used, covered, untried, by_id, budget, lambda) do
    affordable =
      Enum.filter(untried, fn id -> used + by_id[id].tokens <= budget end)

    case affordable do
      [] ->
        {selected_ids, used, covered}

      _ ->
        # pick candidate with best marginal reward per token
        {best_id, _gain} =
          affordable
          |> Enum.map(fn id ->
            cand = by_id[id]
            novel = MapSet.difference(cand.concepts, covered) |> MapSet.size()
            marginal = cand.score + lambda * novel
            {id, marginal / max(cand.tokens, 1)}
          end)
          |> Enum.max_by(fn {_id, g} -> g end)

        cand = by_id[best_id]

        do_rollout(
          [best_id | selected_ids],
          used + cand.tokens,
          MapSet.union(covered, cand.concepts),
          List.delete(untried, best_id),
          by_id,
          budget,
          lambda
        )
    end
  end

  # ---------------------------------------------------------------------------
  # Private: helpers
  # ---------------------------------------------------------------------------

  defp covered_concepts(cands) do
    Enum.reduce(cands, MapSet.new(), fn cand, acc -> MapSet.union(acc, cand.concepts) end)
  end

  # Normalize raw candidate maps into the internal shape with :tokens + :concepts.
  defp normalize(candidates) do
    Enum.map(candidates, fn cand ->
      tokens = candidate_tokens(cand)

      %{
        id: Map.get(cand, :id) || Map.get(cand, :uri) || make_ref(),
        score: (Map.get(cand, :score) || 0.0) * 1.0,
        tokens: tokens,
        concepts: candidate_concepts(cand)
      }
    end)
  end

  defp candidate_tokens(%{tokens: t}) when is_integer(t) and t >= 0, do: t

  defp candidate_tokens(cand) do
    content =
      Map.get(cand, :content) || Map.get(cand, :l1_overview) || Map.get(cand, :l0_abstract) || ""

    max(div(String.length(content), @chars_per_token), 1)
  end

  defp candidate_concepts(%{concepts: cs}) when not is_nil(cs) do
    cond do
      is_struct(cs, MapSet) -> cs
      is_list(cs) -> MapSet.new(cs)
      true -> MapSet.new()
    end
  end

  defp candidate_concepts(cand) do
    text =
      [Map.get(cand, :l0_abstract), Map.get(cand, :title), Map.get(cand, :content)]
      |> Enum.reject(&is_nil/1)
      |> Enum.join(" ")

    text
    |> String.downcase()
    |> String.split(~r/[^\p{L}\p{N}]+/u, trim: true)
    |> Enum.reject(&(String.length(&1) < 3))
    |> MapSet.new()
  end
end
