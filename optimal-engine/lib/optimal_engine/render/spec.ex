defmodule OptimalEngine.Render.Spec do
  @moduledoc """
  A render specification — describes how to produce one audience-specific
  HTML artifact from a source document.

  Each spec maps to a downstream `Receiver` (audience + genre + bandwidth)
  and carries additional constraints and hints that guide the LLM when
  generating the final HTML.

  ## Fields

  - `:name`        — unique key within a workspace (`"exec"`, `"engineering"`)
  - `:title`       — human-readable label
  - `:audience`    — maps to `Receiver.audience` for context assembly
  - `:genre`       — maps to `Receiver.genre` for composition skeleton
  - `:bandwidth`   — `:small | :medium | :large` — controls context depth
  - `:constraints` — freeform map of generation constraints (tone, length, etc.)
  - `:html_hints`  — freeform map of HTML output hints (style, interactivity, etc.)
  - `:modalities`  — list of content types this spec can consume
                     (`[:text, :image, :video, :audio]`)
  """

  @type t :: %__MODULE__{
          name: String.t(),
          title: String.t(),
          audience: String.t(),
          genre: String.t(),
          bandwidth: :small | :medium | :large,
          constraints: map(),
          html_hints: map(),
          modalities: [atom()]
        }

  @enforce_keys [:name]
  defstruct name: nil,
            title: "",
            audience: "default",
            genre: "brief",
            bandwidth: :large,
            constraints: %{},
            html_hints: %{},
            modalities: [:text]

  @doc "Build a spec from a decoded YAML/JSON map (string keys)."
  @spec from_map(map()) :: {:ok, t()} | {:error, :missing_name}
  def from_map(%{"name" => name} = m) when is_binary(name) and name != "" do
    {:ok,
     %__MODULE__{
       name: name,
       title: m["title"] || name,
       audience: m["audience"] || "default",
       genre: m["genre"] || "brief",
       bandwidth: parse_bandwidth(m["bandwidth"]),
       constraints: m["constraints"] || %{},
       html_hints: m["html_hints"] || %{},
       modalities: parse_modalities(m["modalities"])
     }}
  end

  def from_map(_), do: {:error, :missing_name}

  @doc "Serialize to a plain map (for JSON responses)."
  @spec to_map(t()) :: map()
  def to_map(%__MODULE__{} = s) do
    %{
      name: s.name,
      title: s.title,
      audience: s.audience,
      genre: s.genre,
      bandwidth: to_string(s.bandwidth),
      constraints: s.constraints,
      html_hints: s.html_hints,
      modalities: Enum.map(s.modalities, &to_string/1)
    }
  end

  @doc """
  Build prompt guidance from a spec's constraints and html_hints.

  Returns a string the agent can prepend to its generation prompt so the
  LLM knows exactly what kind of HTML to produce.
  """
  @spec prompt_guidance(t()) :: String.t()
  def prompt_guidance(%__MODULE__{} = s) do
    constraint_lines =
      s.constraints
      |> Enum.map(fn {k, v} -> "- #{k}: #{v}" end)
      |> Enum.join("\n")

    hint_lines =
      s.html_hints
      |> Enum.map(fn {k, v} -> "- #{k}: #{v}" end)
      |> Enum.join("\n")

    modality_note =
      case s.modalities do
        [:text] ->
          ""

        mods ->
          "\n\nContent modalities available: #{Enum.join(Enum.map(mods, &to_string/1), ", ")}. Include appropriate elements (img, video, audio) when source assets are provided."
      end

    """
    You are rendering: #{s.title} (audience: #{s.audience}).

    Constraints:
    #{constraint_lines}

    HTML output hints:
    #{hint_lines}#{modality_note}

    Output: a complete, self-contained HTML document with inline CSS. No external dependencies. The HTML must be viewable by opening the file directly in a browser.
    """
    |> String.trim()
  end

  defp parse_bandwidth("small"), do: :small
  defp parse_bandwidth("medium"), do: :medium
  defp parse_bandwidth("large"), do: :large
  defp parse_bandwidth(_), do: :large

  defp parse_modalities(nil), do: [:text]

  defp parse_modalities(list) when is_list(list) do
    list
    |> Enum.map(fn
      m when is_binary(m) -> String.to_atom(m)
      m when is_atom(m) -> m
    end)
    |> Enum.filter(&(&1 in [:text, :image, :video, :audio]))
    |> case do
      [] -> [:text]
      mods -> mods
    end
  end

  defp parse_modalities(_), do: [:text]
end
