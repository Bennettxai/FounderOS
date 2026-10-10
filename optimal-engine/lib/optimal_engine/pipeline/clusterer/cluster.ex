defmodule OptimalEngine.Pipeline.Clusterer.Cluster do
  @moduledoc """
  One cluster — a theme-group of chunks that co-locate under the Phase 6
  similarity function.

  Mirrors the `clusters` table (Phase 1 migration 007):

      %Cluster{id, tenant_id, theme, intent_dominant, member_count,
               centroid, updated_at}

  Clusters are incrementally grown: new chunks join the nearest existing
  cluster if similarity exceeds a threshold, otherwise seed a new cluster.
  The centroid is the running mean of member embeddings.
  """

  @type t :: %__MODULE__{
          id: String.t(),
          tenant_id: String.t(),
          theme: String.t(),
          intent_dominant: atom() | nil,
          member_count: non_neg_integer(),
          centroid: [float()],
          updated_at: String.t() | nil
        }

  defstruct id: nil,
            tenant_id: "default",
            theme: "",
            intent_dominant: nil,
            member_count: 0,
            centroid: [],
            updated_at: nil

  @doc "Build a cluster with a deterministic id `{tenant}:cluster-{uuid-ish}`."
  @spec new(keyword()) :: t()
  def new(fields) when is_list(fields) do
    fields =
      Keyword.put_new_lazy(fields, :id, fn ->
        generate_id(Keyword.get(fields, :tenant_id, "default"))
      end)

    struct(__MODULE__, fields)
  end

  defp generate_id(tenant_id) do
    random = :crypto.strong_rand_bytes(6) |> Base.encode16(case: :lower)
    "#{tenant_id}:cluster-#{random}"
  end
end
