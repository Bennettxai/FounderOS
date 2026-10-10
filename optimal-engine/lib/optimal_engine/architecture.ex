defmodule OptimalEngine.Architecture do
  @moduledoc """
  Top-level facade for the **data-architecture** layer — the engine's
  universal schema system.

  An architecture is how a data point declares its shape: which fields
  it carries, what modality each field is (text / image / audio /
  time-series / tabular / code / tensor / …), and which processor
  owns each field.

  The engine is **model-agnostic**: architectures bind fields to
  processors (via atom id), and the processor registry resolves those
  to actual code — whether that code is an LLM call, a vision model,
  a classical algorithm, a rule engine, or an agent workflow. None
  of them are privileged.

      Architecture.list()              — every built-in + custom
      Architecture.fetch("text_signal")
      Architecture.register(arch)      — persist a tenant's custom schema
      Architecture.processors()        — every registered processor
      Architecture.apply(arch, data, opts)
                                       — validate + dispatch processors

  See `docs/architecture/DATA_ARCHITECTURE.md` for the design
  philosophy and how we're applying SOTA retrieval / multi-modal
  alignment / memory-consolidation work from the wider research
  field.
  """

  alias OptimalEngine.Architecture.{Apply, ProcessorRegistry, Registry}

  defdelegate list(opts \\ []), to: Registry
  defdelegate built_ins, to: Registry
  defdelegate summary, to: Registry
  defdelegate fetch(id_or_name, opts \\ []), to: Registry
  defdelegate fetch!(id_or_name, opts \\ []), to: Registry
  defdelegate register(arch, opts \\ []), to: Registry

  defdelegate processors, to: ProcessorRegistry, as: :all
  defdelegate processor_summary, to: ProcessorRegistry, as: :summary
  defdelegate fetch_processor(id), to: ProcessorRegistry, as: :fetch

  @doc "Validate + dispatch every field-processor in the architecture."
  def apply(arch, data, opts \\ []), do: Apply.run(arch, data, opts)

  @doc "Validate a data point against an architecture without running anything."
  defdelegate validate(arch, data), to: Apply
end
