defmodule OptimalEngine.Architecture.Apply do
  @moduledoc """
  Runtime dispatcher: given a data point + architecture, walk every
  field and hand it to the declared processor. Records each processor
  invocation in the `processor_runs` table for provenance.

  ## Flow

      data_point = %{title: "…", body: "…", image: "…"}
      {:ok, arch} = Registry.fetch("text_signal")
      {:ok, outputs} = Apply.run(arch, data_point, context_id: ctx_id)

  `outputs` is a map `%{field_name => processor_output}`. The engine
  persists each one into its canonical table (embeddings, entities,
  classifications, …) — that persistence is separate so processors
  stay pure.

  ## Validation

  `validate/2` checks a data point against an architecture without
  running any processors. Returns `:ok` or `{:error, [problems]}`.

  ## Model-agnostic by construction

  This module doesn't import any model library — it only dispatches
  atoms to modules registered as processors. Swap a text embedder
  for a vision model by changing the `processor:` hint in the
  architecture; no code path here changes.
  """

  alias OptimalEngine.Architecture.{Architecture, Field, ProcessorRegistry}
  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant

  require Logger

  @type data_point :: map()
  @type field_output :: OptimalEngine.Architecture.Processor.output()
  @type run_opts :: [
          context_id: String.t(),
          tenant_id: String.t(),
          architecture_id: String.t(),
          skip_persist: boolean()
        ]

  @doc """
  Validate a data point against an architecture spec. Returns `:ok`
  or `{:error, [{field, reason}]}` listing every mismatch.
  """
  @spec validate(Architecture.t(), data_point()) :: :ok | {:error, [{atom(), atom()}]}
  def validate(%Architecture{} = arch, data) when is_map(data) do
    data = normalize_keys(data)

    problems =
      Enum.flat_map(arch.fields, fn field ->
        value = Map.get(data, field.name)

        cond do
          field.required and is_nil(value) -> [{field.name, :missing_required}]
          Field.compatible?(field, value) -> []
          true -> [{field.name, :incompatible}]
        end
      end)

    if problems == [], do: :ok, else: {:error, problems}
  end

  @doc """
  Dispatch each field to its processor. Returns
  `{:ok, %{field => output}}` or `{:error, reason}`.
  """
  @spec run(Architecture.t(), data_point(), run_opts()) ::
          {:ok, %{atom() => field_output()}} | {:error, term()}
  def run(%Architecture{} = arch, data, opts \\ []) do
    data = normalize_keys(data)
    context_id = Keyword.get(opts, :context_id)
    tenant_id = Keyword.get(opts, :tenant_id, Tenant.default_id())
    skip_persist? = Keyword.get(opts, :skip_persist, false)

    with :ok <- validate(arch, data) do
      outputs =
        arch.fields
        |> Enum.filter(&has_processor?/1)
        |> Enum.reduce(%{}, fn field, acc ->
          case run_field(
                 field,
                 Map.get(data, field.name),
                 arch,
                 context_id,
                 tenant_id,
                 skip_persist?
               ) do
            {:ok, output} ->
              Map.put(acc, field.name, output)

            {:error, reason} ->
              Logger.warning("[Apply] field=#{field.name} failed: #{inspect(reason)}")
              acc
          end
        end)

      {:ok, outputs}
    end
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp has_processor?(%Field{processor: nil}), do: false
  defp has_processor?(%Field{}), do: true

  defp run_field(field, value, arch, context_id, tenant_id, skip_persist?) do
    with {:ok, mod} <- ProcessorRegistry.fetch(field.processor),
         {:ok, state} <- init_processor(mod),
         {:ok, output} <- mod.process(field, value, state) do
      unless skip_persist? do
        record_run(context_id, tenant_id, arch.id, field.name, field.processor, output)
      end

      {:ok, output}
    end
  end

  defp init_processor(mod) do
    if function_exported?(mod, :init, 1) do
      mod.init(%{})
    else
      {:ok, %{}}
    end
  end

  defp record_run(nil, _tenant_id, _arch_id, _field, _processor, _output), do: :ok

  defp record_run(context_id, tenant_id, architecture_id, field, processor, output) do
    Store.raw_query(
      """
      INSERT INTO processor_runs
        (tenant_id, context_id, architecture_id, processor, field, status, completed_at, metadata)
      VALUES (?1, ?2, ?3, ?4, ?5, 'success', datetime('now'), ?6)
      """,
      [
        tenant_id,
        context_id,
        architecture_id,
        Atom.to_string(processor),
        Atom.to_string(field),
        Jason.encode!(%{kind: output.kind, metadata: output[:metadata] || %{}})
      ]
    )
  end

  defp normalize_keys(map) when is_map(map) do
    Map.new(map, fn
      {k, v} when is_binary(k) -> {String.to_atom(k), v}
      {k, v} when is_atom(k) -> {k, v}
    end)
  rescue
    ArgumentError -> map
  end
end
