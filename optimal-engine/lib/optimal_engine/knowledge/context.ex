defmodule OptimalEngine.Knowledge.Context do
  @moduledoc """
  Agent context injection — queries the knowledge graph and returns
  structured context that can be injected into an agent's prompt or
  decision loop.

  This is the primary integration point between miosa_knowledge and
  the agent system. Instead of agents querying raw triples, they call
  `for_agent/2` to get a pre-structured context map.
  """

  alias OptimalEngine.Knowledge.Store

  @type context_map :: %{
          agent_id: String.t(),
          facts: [OptimalEngine.Knowledge.triple()],
          relationships: %{String.t() => [String.t()]},
          properties: %{String.t() => String.t()},
          fact_count: non_neg_integer()
        }

  @doc """
  Build a context snapshot for a specific agent.

  Queries the knowledge store for all facts relevant to the given agent
  and returns a structured map ready for prompt injection.

  ## Options

  - `:agent_id` — Agent identifier to scope the query (required)
  - `:scope` — Additional subject prefixes to include (default: [agent_id])
  - `:max_facts` — Maximum facts to return (default: 100)

  ## Examples

      ctx = OptimalEngine.Knowledge.Context.for_agent(store, agent_id: "agent_1")
      # => %{agent_id: "agent_1", facts: [...], relationships: %{...}, ...}
  """
  @spec for_agent(GenServer.server(), keyword()) :: context_map()
  def for_agent(store, opts) do
    agent_id = Keyword.fetch!(opts, :agent_id)
    scopes = Keyword.get(opts, :scope, [agent_id])
    max_facts = Keyword.get(opts, :max_facts, 100)

    facts =
      scopes
      |> Enum.flat_map(fn scope ->
        case Store.query(store, subject: scope) do
          {:ok, results} -> results
          _ -> []
        end
      end)
      |> Enum.take(max_facts)

    relationships = build_relationships(facts)
    properties = build_properties(facts)

    %{
      agent_id: agent_id,
      facts: facts,
      relationships: relationships,
      properties: properties,
      fact_count: length(facts)
    }
  end

  @doc """
  Render context as a text block suitable for LLM prompt injection.

  Converts the structured context map into a formatted string that can
  be appended to an agent's system prompt or inserted into a context window.
  """
  @spec to_prompt(context_map()) :: String.t()
  def to_prompt(%{facts: [], agent_id: agent_id}) do
    "# Knowledge Context (#{agent_id})\nNo facts in knowledge graph."
  end

  def to_prompt(%{} = ctx) do
    sections = [
      "# Knowledge Context (#{ctx.agent_id})",
      "Facts: #{ctx.fact_count}",
      "",
      "## Properties",
      format_properties(ctx.properties),
      "",
      "## Relationships",
      format_relationships(ctx.relationships)
    ]

    Enum.join(sections, "\n")
  end

  # --- Private ---

  defp build_relationships(facts) do
    facts
    |> Enum.filter(fn {_s, _p, o} -> String.contains?(o, ":") end)
    |> Enum.group_by(fn {_s, p, _o} -> p end, fn {_s, _p, o} -> o end)
  end

  defp build_properties(facts) do
    facts
    |> Enum.reject(fn {_s, _p, o} -> String.contains?(o, ":") end)
    |> Map.new(fn {_s, p, o} -> {p, o} end)
  end

  defp format_properties(props) when map_size(props) == 0, do: "  (none)"

  defp format_properties(props) do
    props
    |> Enum.map(fn {k, v} -> "  - #{k}: #{v}" end)
    |> Enum.join("\n")
  end

  defp format_relationships(rels) when map_size(rels) == 0, do: "  (none)"

  defp format_relationships(rels) do
    rels
    |> Enum.map(fn {pred, objects} ->
      "  - #{pred}: #{Enum.join(objects, ", ")}"
    end)
    |> Enum.join("\n")
  end
end
