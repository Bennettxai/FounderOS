defmodule OptimalEngine.Storage.Planner do
  @moduledoc """
  Converts workspace use cases into storage capabilities and providers.

  Plans are advisory. They never enable a provider or move data. Activation is
  an explicit, reviewed deployment decision because canonical-store changes and
  cross-workspace routing have a large blast radius.
  """

  alias OptimalEngine.Storage.Providers

  @use_cases %{
    "desktop_local" => %{
      label: "Private desktop or single-node engine",
      providers: ~w(sqlite fts5 embedded_vectors rocksdb filesystem ets),
      reason: "Durable local memory with no external service dependency"
    },
    "cloud_team" => %{
      label: "Multi-user cloud workspace",
      providers: ~w(postgres s3 openbao nats_jetstream valkey),
      reason: "Concurrent writes, durable artifacts, managed secrets, and distributed coordination"
    },
    "multi_device" => %{
      label: "Offline-capable multi-device workspace",
      providers: ~w(postgres s3 nats_jetstream openbao),
      reason: "Governed mutation replay, shared durable state, and device-safe credentials"
    },
    "high_volume_connectors" => %{
      label: "High-volume connector ingestion",
      providers: ~w(nats_jetstream s3 postgres),
      reason: "Replayable ingestion, raw payload retention, and concurrent canonical writes"
    },
    "analytics" => %{
      label: "Large reports and cross-workspace analytics",
      providers: ~w(duckdb),
      reason: "Columnar analytical projection without loading the transactional store"
    },
    "high_scale_semantic" => %{
      label: "High-scale semantic retrieval",
      providers: ~w(qdrant),
      reason: "Indexed vector filtering and horizontal vector capacity"
    },
    "media_archive" => %{
      label: "Audio, video, document, or image evidence",
      providers: ~w(s3),
      reason: "Durable object storage for source evidence and extraction artifacts"
    },
    "regulated" => %{
      label: "Regulated or high-assurance deployment",
      providers: ~w(postgres s3 openbao nats_jetstream),
      reason: "Auditable secrets, durable evidence, replay, retention, and recovery boundaries"
    },
    "enterprise_ai" => %{
      label: "Enterprise AI over protected systems of record",
      providers: ~w(postgres s3 openbao nats_jetstream fractal),
      reason:
        "Governed canonical state plus a Fractal synchronized digital twin for isolated enterprise AI processing"
    },
    "geospatial" => %{
      label: "Location, polygon, or routing workloads",
      providers: ~w(postgres),
      reason: "PostGIS should be enabled on the production Postgres provider"
    },
    "time_series" => %{
      label: "Operational or domain time-series workloads",
      providers: ~w(postgres duckdb),
      reason: "Transactional recent state plus a columnar historical projection"
    }
  }

  @spec use_cases() :: [map()]
  def use_cases do
    @use_cases
    |> Enum.map(fn {id, spec} -> Map.put(spec, :id, id) end)
    |> Enum.sort_by(& &1.id)
  end

  @spec plan([String.t()], keyword()) :: {:ok, map()} | {:error, {:unknown_use_cases, [String.t()]}}
  def plan(use_case_ids, opts \\ []) do
    ids = use_case_ids |> Enum.map(&to_string/1) |> Enum.uniq()
    unknown = Enum.reject(ids, &Map.has_key?(@use_cases, &1))

    if unknown == [] do
      provider_ids =
        ids
        |> Enum.flat_map(&@use_cases[&1].providers)
        |> Enum.uniq()

      providers =
        Providers.list(probe: Keyword.get(opts, :probe, false))
        |> Map.new(&{&1.id, &1})

      selected = Enum.map(provider_ids, &Map.fetch!(providers, &1))

      {:ok,
       %{
         use_cases: Enum.map(ids, &Map.put(@use_cases[&1], :id, &1)),
         providers: selected,
         ready: Enum.all?(selected, &ready?/1),
         activation_required: Enum.filter(selected, &(not ready?(&1))) |> Enum.map(& &1.id),
         invariant:
           "Canonical data remains governed by its domain owner; projections and transports never become truth."
       }}
    else
      {:error, {:unknown_use_cases, unknown}}
    end
  end

  defp ready?(%{lifecycle_state: state}), do: state in ["active", "available"]
end
