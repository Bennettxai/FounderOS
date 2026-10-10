defmodule OptimalEngine.Retrieval.BandwidthPlanner do
  @moduledoc """
  Plans which retrieved items survive a receiver's token budget.

  RAG almost always produces more relevant chunks than the receiver
  can swallow. The planner is the place we enforce Shannon's ceiling:
  pack the highest-scoring items until the budget is exhausted, then
  stop — with a loud `:truncated` count so the caller knows what was
  dropped.

  ## Inputs

      items :: [%{score: float, content: String.t(), uri: String.t() | nil}]

  Any map that carries `:content` and `:score` works; surplus keys are
  preserved so the caller can thread metadata through.

  ## Output

      %{
        kept:        [item],
        dropped:     [item],
        used_tokens: integer,
        budget:      integer,
        truncated?:  boolean
      }

  ## Token estimation

  We use the 4-chars-per-token heuristic — same as the
  `ContextAssembler`. This is intentionally conservative; swap in a
  real tokenizer before shipping to paid LLMs.
  """

  @chars_per_token 4
  @reserve_overhead 64

  # Tier ladder, richest → leanest. L3 is verbatim passthrough (full
  # `:content`), L2 full content, L1 the `:l1_overview`, L0 the
  # `:l0_abstract`. Downgrading an item swaps which field renders so a
  # high-value item survives the budget at a coarser tier instead of being
  # dropped outright.
  @tier_order [:l3, :l2, :l1, :l0]

  @type item :: %{required(:content) => String.t(), optional(any()) => any()}
  @type plan :: %{
          kept: [item()],
          dropped: [item()],
          used_tokens: non_neg_integer(),
          budget: non_neg_integer(),
          truncated?: boolean()
        }

  @doc """
  Pack `items` greedily by descending score, honoring `budget`.

  Items without an explicit `:score` are treated as 0 and ordered
  last. A per-item overhead of #{@reserve_overhead} tokens is reserved
  to cover the citation/source wrapper the Deliver layer adds.
  """
  @spec plan([item()], non_neg_integer()) :: plan()
  def plan(items, budget) when is_list(items) and is_integer(budget) and budget >= 0 do
    sorted = Enum.sort_by(items, &Map.get(&1, :score, 0.0), :desc)

    {kept_rev, dropped_rev, used} =
      Enum.reduce(sorted, {[], [], 0}, fn item, {kept, dropped, tokens} ->
        cost = estimate_tokens(item) + @reserve_overhead

        if tokens + cost <= budget do
          {[item | kept], dropped, tokens + cost}
        else
          {kept, [item | dropped], tokens}
        end
      end)

    %{
      kept: Enum.reverse(kept_rev),
      dropped: Enum.reverse(dropped_rev),
      used_tokens: used,
      budget: budget,
      truncated?: dropped_rev != []
    }
  end

  @doc """
  Tier-aware packing: instead of binary keep/drop, **downgrade** items to
  leaner tiers (L3 verbatim → L2 full → L1 overview → L0 abstract) so the
  highest-scoring items survive the budget at a coarser fidelity rather than
  being cut entirely.

  Each item may carry tier fields; missing fields fall back to `:content`:

      %{
        score: float,
        content:     String.t() | nil,   # L3 verbatim / L2 full
        l1_overview: String.t() | nil,   # L1
        l0_abstract: String.t() | nil    # L0
      }

  Algorithm: sort by score desc; for each item try the richest tier that
  still fits the remaining budget, walking down `#{inspect(@tier_order)}`.
  Only when even L0 will not fit is the item dropped.

  Returns the same shape as `plan/2` plus a `:tier` on each kept item and a
  `:downgrades` count.

      %{kept: [item_with_tier], dropped: [item], used_tokens: int,
        budget: int, truncated?: bool, downgrades: int}
  """
  @spec plan_tiered([item()], non_neg_integer()) :: map()
  def plan_tiered(items, budget) when is_list(items) and is_integer(budget) and budget >= 0 do
    sorted = Enum.sort_by(items, &Map.get(&1, :score, 0.0), :desc)

    {kept_rev, dropped_rev, used, downgrades} =
      Enum.reduce(sorted, {[], [], 0, 0}, fn item, {kept, dropped, tokens, downs} ->
        remaining = budget - tokens

        case fit_tier(item, remaining) do
          {:ok, tier, cost} ->
            kept_item = Map.put(item, :tier, tier)
            downs = downs + if tier != :l3, do: 1, else: 0
            {[kept_item | kept], dropped, tokens + cost, downs}

          :drop ->
            {kept, [item | dropped], tokens, downs}
        end
      end)

    %{
      kept: Enum.reverse(kept_rev),
      dropped: Enum.reverse(dropped_rev),
      used_tokens: used,
      budget: budget,
      truncated?: dropped_rev != [],
      downgrades: downgrades
    }
  end

  # Find the richest tier whose cost fits `remaining`. Returns {:ok, tier, cost}
  # or :drop when even the leanest tier overflows.
  defp fit_tier(item, remaining) do
    Enum.reduce_while(@tier_order, :drop, fn tier, _acc ->
      cost = tier_tokens(item, tier) + @reserve_overhead

      if cost <= remaining do
        {:halt, {:ok, tier, cost}}
      else
        {:cont, :drop}
      end
    end)
  end

  @doc """
  Render the text for an item at a given tier. L3/L2 = full `:content`,
  L1 = `:l1_overview` (falls back to content), L0 = `:l0_abstract`
  (falls back to overview, then content).
  """
  @spec render_tier(item(), :l0 | :l1 | :l2 | :l3) :: String.t()
  def render_tier(item, :l3), do: tier_text(item, :l3)
  def render_tier(item, :l2), do: tier_text(item, :l2)
  def render_tier(item, :l1), do: tier_text(item, :l1)
  def render_tier(item, :l0), do: tier_text(item, :l0)

  defp tier_text(item, tier) when tier in [:l3, :l2], do: Map.get(item, :content) || ""

  defp tier_text(item, :l1) do
    Map.get(item, :l1_overview) || Map.get(item, :content) || ""
  end

  defp tier_text(item, :l0) do
    Map.get(item, :l0_abstract) || Map.get(item, :l1_overview) || Map.get(item, :content) || ""
  end

  defp tier_tokens(item, tier), do: div(String.length(tier_text(item, tier)), @chars_per_token)

  @doc """
  Estimate the token cost of an item. Uses the 4-char heuristic on the
  `:content` field. Nil/empty content = 0.
  """
  @spec estimate_tokens(item() | String.t() | nil) :: non_neg_integer()
  def estimate_tokens(nil), do: 0
  def estimate_tokens(%{content: content}), do: estimate_tokens(content)
  def estimate_tokens(text) when is_binary(text), do: div(String.length(text), @chars_per_token)
  def estimate_tokens(_), do: 0

  @doc "Hard-truncate a single string to fit a token budget."
  @spec truncate(String.t(), non_neg_integer()) :: String.t()
  def truncate(text, budget) when is_binary(text) and is_integer(budget) and budget >= 0 do
    max_chars = budget * @chars_per_token

    cond do
      budget == 0 -> ""
      String.length(text) <= max_chars -> text
      true -> String.slice(text, 0, max_chars) <> "…"
    end
  end
end
