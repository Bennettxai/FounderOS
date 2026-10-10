defmodule OptimalEngine.Audit.Event do
  @moduledoc """
  Append-only audit event. Every read + write in the engine emits one of
  these so SOC 2 / GDPR audit queries can answer questions like
  "who accessed X between T1 and T2."

  Events are written to the `events` table and can be streamed to SIEM
  backends via `OptimalEngine.Audit.SIEM` (Phase 11).

  ## Canonical event kinds

  - `"ingest.started" | "ingest.completed" | "ingest.failed"`
  - `"retrieval.executed"`
  - `"wiki.page.curated" | "wiki.page.read"`
  - `"permission.denied" | "permission.granted"`
  - `"connector.synced" | "connector.failed"`
  - `"tenant.created" | "principal.created" | "principal.logged_in"`
  - `"legal_hold.placed" | "legal_hold.released"`
  - `"retention.applied"`
  """

  alias OptimalEngine.Tenancy.Tenant

  @type t :: %__MODULE__{
          tenant_id: String.t(),
          ts: String.t() | nil,
          principal: String.t(),
          kind: String.t(),
          target_uri: String.t() | nil,
          latency_ms: integer() | nil,
          metadata: map()
        }

  defstruct tenant_id: Tenant.default_id(),
            ts: nil,
            principal: "system",
            kind: nil,
            target_uri: nil,
            latency_ms: nil,
            metadata: %{}

  @doc "Build an Event struct with default tenant + timestamp."
  @spec new(String.t(), keyword()) :: t()
  def new(kind, opts \\ []) when is_binary(kind) do
    %__MODULE__{
      tenant_id: Keyword.get(opts, :tenant_id, Tenant.default_id()),
      ts: Keyword.get(opts, :ts, DateTime.utc_now() |> DateTime.to_iso8601()),
      principal: Keyword.get(opts, :principal, "system"),
      kind: kind,
      target_uri: Keyword.get(opts, :target_uri),
      latency_ms: Keyword.get(opts, :latency_ms),
      metadata: Keyword.get(opts, :metadata, %{})
    }
  end
end
