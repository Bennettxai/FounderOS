defmodule OptimalEngine.Retrieval.Receiver do
  @moduledoc """
  Receiver profile — the downstream consumer of a retrieval result.

  Every retrieval call is *for someone*: a human user, an AI agent, a
  connector worker piping text into an external system. The Receiver
  struct normalizes "who/what is asking" into a single description the
  Composer + Deliver layer can plan against.

  ## Fields

  - `:id`             — principal id (`"user:ada@sample.com"` / `"agent:bot"` / nil)
  - `:kind`           — `:user | :agent | :service | :unknown`
  - `:audience`       — wiki audience tag (`"default" | "sales" | "legal" | …`)
  - `:tenant_id`      — tenant scope
  - `:bandwidth`      — `:small | :medium | :large` (target output size)
  - `:token_budget`   — hard cap, derived from bandwidth when absent
  - `:format`         — `:plain | :markdown | :claude | :openai`
  - `:genre`          — preferred genre skeleton (brief, spec, pitch, …)
  - `:mode`           — `:linguistic | :code | :visual`
  - `:locale`         — BCP-47 locale string (defaults to `"en-US"`)

  ## Bandwidth → token budget

      :small    → 1_500 tokens   (chat UI, mobile, voice)
      :medium   → 6_000 tokens   (desktop assistant, RAG-with-reasoning)
      :large    → 24_000 tokens  (long-context LLM, batch analysis)

  Call `Receiver.from_principal/2` to hydrate a receiver from a stored
  principal, or `Receiver.new/1` for an ad-hoc caller (CLI, connector,
  test).

  This module is pure — it never hits the DB unless you explicitly ask
  `from_principal/2` to look up metadata.
  """

  alias OptimalEngine.Identity.Principal
  alias OptimalEngine.Tenancy.Tenant

  @type bandwidth :: :small | :medium | :large
  @type format :: :plain | :markdown | :claude | :openai
  @type mode :: :linguistic | :code | :visual

  @type t :: %__MODULE__{
          id: String.t() | nil,
          kind: :user | :agent | :service | :unknown,
          audience: String.t(),
          tenant_id: String.t(),
          bandwidth: bandwidth(),
          token_budget: non_neg_integer(),
          format: format(),
          genre: String.t(),
          mode: mode(),
          locale: String.t()
        }

  defstruct id: nil,
            kind: :unknown,
            audience: "default",
            tenant_id: nil,
            bandwidth: :medium,
            token_budget: 6_000,
            format: :markdown,
            genre: "brief",
            mode: :linguistic,
            locale: "en-US"

  @budgets %{small: 1_500, medium: 6_000, large: 24_000}

  @doc """
  Build a receiver from a map of overrides. Any omitted key falls back
  to the struct default. The `token_budget` is auto-derived from
  `bandwidth` when not supplied.
  """
  @spec new(map() | keyword()) :: t()
  def new(attrs) when is_map(attrs) or is_list(attrs) do
    attrs = Map.new(attrs)

    bandwidth = Map.get(attrs, :bandwidth, :medium)

    base = %__MODULE__{
      tenant_id: Map.get(attrs, :tenant_id, Tenant.default_id()),
      bandwidth: bandwidth,
      token_budget: Map.get(attrs, :token_budget, Map.fetch!(@budgets, bandwidth))
    }

    attrs
    |> Enum.reduce(base, fn
      {:token_budget, v}, acc -> %{acc | token_budget: v}
      {:bandwidth, _}, acc -> acc
      {:tenant_id, _}, acc -> acc
      {k, v}, acc -> Map.put(acc, k, v)
    end)
  end

  @doc """
  Hydrate a receiver from a stored `Principal`. Audience, bandwidth,
  genre, and format come from the principal's `metadata` map when
  present; otherwise sane defaults are picked per `kind`.

  Returns `{:ok, receiver}` or `{:error, :not_found}`.
  """
  @spec from_principal(String.t(), keyword()) :: {:ok, t()} | {:error, :not_found}
  def from_principal(principal_id, opts \\ []) when is_binary(principal_id) do
    case Principal.get(principal_id) do
      {:ok, %Principal{} = p} ->
        overrides = Enum.into(opts, %{})
        {:ok, from_principal_struct(p, overrides)}

      _ ->
        {:error, :not_found}
    end
  end

  @doc """
  Build a receiver directly from a `Principal` struct (skipping the DB
  lookup). Handy in tests and hot paths.
  """
  @spec from_principal_struct(Principal.t(), map()) :: t()
  def from_principal_struct(%Principal{} = p, overrides \\ %{}) do
    meta = p.metadata || %{}

    bandwidth = atom_value(overrides[:bandwidth] || meta["bandwidth"] || default_bandwidth(p.kind))
    format = atom_value(overrides[:format] || meta["format"] || default_format(p.kind))
    mode = atom_value(overrides[:mode] || meta["mode"] || :linguistic)

    %__MODULE__{
      id: p.id,
      kind: p.kind,
      tenant_id: p.tenant_id,
      audience: overrides[:audience] || meta["audience"] || "default",
      bandwidth: bandwidth,
      token_budget: overrides[:token_budget] || Map.fetch!(@budgets, bandwidth),
      format: format,
      genre: overrides[:genre] || meta["genre"] || default_genre(p.kind),
      mode: mode,
      locale: overrides[:locale] || meta["locale"] || "en-US"
    }
  end

  @doc "Default receiver for anonymous/system callers (CLI, batch jobs)."
  @spec anonymous(keyword()) :: t()
  def anonymous(opts \\ []) do
    new([audience: "default", kind: :unknown] ++ opts)
  end

  @doc "Returns the token budget for a given bandwidth label."
  @spec budget_for(bandwidth()) :: non_neg_integer()
  def budget_for(bw) when bw in [:small, :medium, :large], do: Map.fetch!(@budgets, bw)

  # ─── private ─────────────────────────────────────────────────────────────

  defp default_bandwidth(:service), do: :large
  defp default_bandwidth(:agent), do: :large
  defp default_bandwidth(_), do: :medium

  defp default_format(:agent), do: :claude
  defp default_format(:service), do: :openai
  defp default_format(_), do: :markdown

  defp default_genre(:user), do: "brief"
  defp default_genre(:agent), do: "spec"
  defp default_genre(:service), do: "note"
  defp default_genre(_), do: "note"

  defp atom_value(v) when is_atom(v), do: v
  defp atom_value(v) when is_binary(v), do: String.to_existing_atom(v)
end
