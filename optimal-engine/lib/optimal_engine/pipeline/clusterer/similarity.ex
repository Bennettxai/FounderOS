defmodule OptimalEngine.Pipeline.Clusterer.Similarity do
  @moduledoc """
  The Phase 6 similarity function:

      sim(a, b) = 0.6 · cosine(a.embedding, b.embedding)
                + 0.2 · entity_overlap(a, b)    # Jaccard over entity sets
                + 0.15 · intent_match(a, b)     # 1.0 if same intent, else 0.0
                + 0.05 · node_affinity(a, b)    # 1.0 same node, 0.5 same tree, else 0.0

  All components return values in [0.0, 1.0], so the composite also lands
  in [0.0, 1.0].

  This module is pure — no I/O, no side effects, trivially testable. The
  Clusterer assembles feature maps from chunks + embeddings + classifications
  + intents and calls the functions here.
  """

  @cosine_weight 0.60
  @entity_weight 0.20
  @intent_weight 0.15
  @node_weight 0.05

  @type feature :: %{
          embedding: [float()],
          entities: [String.t()],
          intent: atom() | nil,
          node_id: String.t() | nil,
          node_ancestors: [String.t()]
        }

  @doc """
  Weighted composite similarity in [0.0, 1.0].

  Normalizes by the weight of available components — a pure-embedding
  match (no entities, no intent, no node) still produces a full 1.0
  score when the cosine is 1.0, rather than being capped at 0.6.
  """
  @spec sim(feature(), feature()) :: float()
  def sim(%{} = a, %{} = b) do
    components = [
      {@cosine_weight, cosine(a.embedding, b.embedding), has_embedding?(a) and has_embedding?(b)},
      {@entity_weight, entity_overlap(a.entities, b.entities),
       has_entities?(a) and has_entities?(b)},
      {@intent_weight, intent_match(a.intent, b.intent), has_intent?(a) and has_intent?(b)},
      {@node_weight, node_affinity(a, b), has_node?(a) and has_node?(b)}
    ]

    {total_weight, total_score} =
      Enum.reduce(components, {0.0, 0.0}, fn {w, s, present?}, {tw, ts} ->
        if present?, do: {tw + w, ts + w * s}, else: {tw, ts}
      end)

    if total_weight == 0.0, do: 0.0, else: total_score / total_weight
  end

  defp has_embedding?(%{embedding: v}) when is_list(v) and v != [], do: true
  defp has_embedding?(_), do: false

  defp has_entities?(%{entities: e}) when is_list(e) and e != [], do: true
  defp has_entities?(_), do: false

  defp has_intent?(%{intent: i}) when is_atom(i) and not is_nil(i), do: true
  defp has_intent?(_), do: false

  defp has_node?(%{node_id: nid}) when is_binary(nid), do: true
  defp has_node?(_), do: false

  @doc "Cosine similarity of two equal-length vectors. Returns 0.0 on mismatch or empty."
  @spec cosine([float()], [float()]) :: float()
  def cosine([], _), do: 0.0
  def cosine(_, []), do: 0.0

  def cosine(a, b) when is_list(a) and is_list(b) do
    if length(a) != length(b) do
      0.0
    else
      dot = dot_product(a, b)
      mag_a = :math.sqrt(dot_product(a, a))
      mag_b = :math.sqrt(dot_product(b, b))

      if mag_a == 0.0 or mag_b == 0.0 do
        0.0
      else
        # Cosine in [-1, 1]; rescale to [0, 1]. For embedding models
        # that produce normalized vectors in the positive orthant
        # (nomic-embed-* does) this mostly lands in [0, 1] already.
        raw = dot / (mag_a * mag_b)
        (raw + 1.0) / 2.0
      end
    end
  end

  defp dot_product(a, b) do
    a
    |> Enum.zip(b)
    |> Enum.reduce(0.0, fn {x, y}, acc -> acc + x * y end)
  end

  @doc "Jaccard overlap over two entity name lists. 0.0..1.0."
  @spec entity_overlap([String.t()], [String.t()]) :: float()
  def entity_overlap([], []), do: 0.0
  def entity_overlap([], _), do: 0.0
  def entity_overlap(_, []), do: 0.0

  def entity_overlap(a, b) when is_list(a) and is_list(b) do
    set_a = MapSet.new(a)
    set_b = MapSet.new(b)
    intersection = MapSet.intersection(set_a, set_b) |> MapSet.size()
    union = MapSet.union(set_a, set_b) |> MapSet.size()

    if union == 0, do: 0.0, else: intersection / union
  end

  @doc "1.0 if same intent (non-nil), else 0.0."
  @spec intent_match(atom() | nil, atom() | nil) :: float()
  def intent_match(nil, _), do: 0.0
  def intent_match(_, nil), do: 0.0
  def intent_match(same, same), do: 1.0
  def intent_match(_a, _b), do: 0.0

  @doc """
  Node affinity between two features:

    * 1.0 — same `node_id`
    * 0.5 — share any ancestor in their `node_ancestors` trees
    * 0.0 — otherwise
  """
  @spec node_affinity(feature(), feature()) :: float()
  def node_affinity(%{node_id: nil}, _), do: 0.0
  def node_affinity(_, %{node_id: nil}), do: 0.0
  def node_affinity(%{node_id: same}, %{node_id: same}), do: 1.0

  def node_affinity(%{node_ancestors: a}, %{node_ancestors: b})
      when is_list(a) and is_list(b) do
    if Enum.any?(a, fn anc -> anc in b end), do: 0.5, else: 0.0
  end

  def node_affinity(_, _), do: 0.0

  @doc "Element-wise vector mean of a non-empty list of equal-length vectors."
  @spec mean_vector([[float()]]) :: [float()]
  def mean_vector([]), do: []

  def mean_vector([first | _] = vectors) do
    n = length(vectors)
    dim = length(first)
    zeros = List.duplicate(0.0, dim)

    sums =
      Enum.reduce(vectors, zeros, fn vec, acc ->
        Enum.zip_with([acc, vec], fn [x, y] -> x + y end)
      end)

    Enum.map(sums, &(&1 / n))
  end
end
