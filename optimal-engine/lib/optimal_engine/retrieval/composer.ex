defmodule OptimalEngine.Retrieval.Composer do
  @moduledoc """
  Re-encodes signals for specific receivers.

  The Composer is stateless — it takes a Signal and a receiver_id, looks up that
  receiver's genre competence from the topology, and reformats the signal content
  into the receiver's preferred genre.

  Per Signal Theory: "Path of least resistance = minimum decoding effort at the receiver."

  ## Genre reformatting rules
  - `brief`    — Objective + Key Messages (3-5 bullets) + Call to Action
  - `spec`     — Goal + Requirements + Constraints + Acceptance Criteria
  - `runbook`  — Procedure + Steps (numbered)
  - `note`     — Raw signal content with header
  - default    — Pass-through with genre header

  The topology is passed explicitly (not fetched internally) to keep this module
  testable without starting the full supervision tree.
  """

  alias OptimalEngine.{Signal, Routing}
  alias OptimalEngine.Signal.GenreRegistry

  @doc """
  Renders a Signal for a specific receiver.

  Returns `{:ok, rendered_string}` in the receiver's preferred genre,
  or `{:error, :unknown_receiver}` if the receiver is not in the topology.
  """
  @spec render_for(Signal.t(), String.t(), Routing.t()) ::
          {:ok, String.t()} | {:error, term()}
  def render_for(%Signal{} = signal, receiver_id, topology) when is_binary(receiver_id) do
    target_genre =
      case Routing.primary_genre_for(topology, receiver_id) do
        nil -> "note"
        genre -> genre
      end

    rendered = reformat(signal, target_genre)
    {:ok, rendered}
  end

  @doc """
  Returns the optimal genre for a receiver, given the signal's original genre.

  If the signal is already in a genre the receiver can decode, returns the original.
  Otherwise returns the receiver's primary genre competence.
  """
  @spec optimal_genre(Signal.t(), String.t(), Routing.t()) :: String.t()
  def optimal_genre(%Signal{genre: signal_genre}, receiver_id, topology) do
    case Routing.endpoint_for(topology, receiver_id) do
      %{genre_competence: competencies} ->
        if signal_genre in competencies do
          signal_genre
        else
          List.first(competencies, "note")
        end

      nil ->
        signal_genre
    end
  end

  # --- Private: Genre Skeletons ---

  defp reformat(signal, "brief") do
    """
    # Brief: #{signal.title}

    **Node:** #{signal.node} | **Date:** #{format_date(signal.modified_at)}

    ## Objective
    #{extract_objective(signal)}

    ## Key Messages
    #{extract_key_messages(signal)}

    ## Call to Action
    #{extract_cta(signal)}
    """
    |> String.trim()
  end

  defp reformat(signal, "spec") do
    """
    # Spec: #{signal.title}

    **Node:** #{signal.node} | **Genre:** #{signal.genre} | **Date:** #{format_date(signal.modified_at)}

    ## Goal
    #{extract_goal(signal)}

    ## Requirements
    #{extract_requirements(signal)}

    ## Constraints
    - Source node: #{signal.node}
    - Format: #{signal.format}

    ## Acceptance Criteria
    Signal has been classified and routed. S/N ratio: #{signal.sn_ratio}
    """
    |> String.trim()
  end

  defp reformat(signal, "runbook") do
    """
    # Runbook: #{signal.title}

    **Node:** #{signal.node} | **Date:** #{format_date(signal.modified_at)}

    ## Procedure

    #{to_numbered_steps(signal.content)}
    """
    |> String.trim()
  end

  defp reformat(signal, "note") do
    """
    # Note: #{signal.title}

    **Node:** #{signal.node} | **Genre:** #{signal.genre} | **S/N:** #{signal.sn_ratio}

    #{signal.l1_description}
    """
    |> String.trim()
  end

  defp reformat(signal, "pitch") do
    """
    # #{signal.title}

    ## Problem
    #{extract_objective(signal)}

    ## Solution
    #{extract_key_messages(signal)}

    ## Next Step
    #{extract_cta(signal)}
    """
    |> String.trim()
  end

  defp reformat(signal, genre) do
    # Any of the 142 registry genres renders its canonical section skeleton.
    # Unknown genres fall through to a metadata-headed pass-through.
    case GenreRegistry.skeleton(genre) do
      [] -> reformat_passthrough(signal, genre)
      sections -> reformat_from_skeleton(signal, genre, sections)
    end
  end

  defp reformat_from_skeleton(signal, genre, sections) do
    category = GenreRegistry.category(genre) || "general"

    section_blocks =
      sections
      |> Enum.map_join("\n\n", fn %{name: name, required: required} ->
        marker = if required, do: "[required]", else: "[optional]"
        "## #{name}\n_#{marker}_"
      end)

    """
    # #{title_case(genre)}: #{signal.title}

    > Category: #{category} | Node: #{signal.node} | S/N: #{signal.sn_ratio}

    #{section_blocks}

    ---
    _Source content:_

    #{signal.l1_description || signal.content}
    """
    |> String.trim()
  end

  defp reformat_passthrough(signal, genre) do
    """
    # #{signal.title}

    > Genre: #{genre} | Node: #{signal.node} | S/N: #{signal.sn_ratio}

    #{signal.content}
    """
    |> String.trim()
  end

  defp title_case(genre) do
    genre
    |> to_string()
    |> String.split("-", trim: true)
    |> Enum.map_join(" ", &String.capitalize/1)
  end

  # --- Content extraction helpers ---

  defp extract_objective(signal) do
    # Try to find an ## Objective section, or fall back to l1_description
    case Regex.run(~r/##\s+Objective\s*\n(.*?)(?=\n##|\z)/s, signal.content || "") do
      [_, text] -> String.trim(text)
      _ -> signal.l1_description || signal.l0_summary
    end
  end

  defp extract_key_messages(signal) do
    # Extract bullet points from content, max 5
    bullets =
      Regex.scan(~r/^[-*]\s+(.+)$/m, signal.content || "")
      |> Enum.map(fn [_, bullet] -> "- #{String.trim(bullet)}" end)
      |> Enum.take(5)

    if bullets == [] do
      "- #{signal.l0_summary}"
    else
      Enum.join(bullets, "\n")
    end
  end

  defp extract_cta(signal) do
    case Regex.run(
           ~r/##\s+(?:Call to Action|Next Step|Action Items?)\s*\n(.*?)(?=\n##|\z)/is,
           signal.content || ""
         ) do
      [_, text] -> String.trim(text)
      _ -> "Review and respond to: #{signal.title}"
    end
  end

  defp extract_goal(signal) do
    case Regex.run(
           ~r/##\s+(?:Goal|Objective|Purpose)\s*\n(.*?)(?=\n##|\z)/is,
           signal.content || ""
         ) do
      [_, text] -> String.trim(text)
      _ -> signal.l1_description || signal.l0_summary
    end
  end

  defp extract_requirements(signal) do
    case Regex.run(~r/##\s+Requirements\s*\n(.*?)(?=\n##|\z)/is, signal.content || "") do
      [_, text] ->
        String.trim(text)

      _ ->
        signal.content
        |> extract_numbered_list()
        |> case do
          [] -> "- No explicit requirements found in source signal."
          items -> Enum.join(items, "\n")
        end
    end
  end

  defp extract_numbered_list(nil), do: []

  defp extract_numbered_list(content) do
    Regex.scan(~r/^\d+\.\s+(.+)$/m, content)
    |> Enum.map(fn [_, item] -> "1. #{String.trim(item)}" end)
    |> Enum.take(10)
  end

  defp to_numbered_steps(nil), do: "No steps found."

  defp to_numbered_steps(content) do
    # Convert bullet points to numbered steps
    bullets =
      Regex.scan(~r/^[-*]\s+(.+)$/m, content)
      |> Enum.map(fn [_, step] -> step end)

    if bullets == [] do
      content |> String.trim() |> truncate(300)
    else
      bullets
      |> Enum.with_index(1)
      |> Enum.map(fn {step, n} -> "#{n}. #{String.trim(step)}" end)
      |> Enum.join("\n")
    end
  end

  defp format_date(nil), do: "unknown"
  defp format_date(%DateTime{} = dt), do: Calendar.strftime(dt, "%Y-%m-%d")

  defp truncate(str, max) when byte_size(str) > max do
    String.slice(str, 0, max) <> "..."
  end

  defp truncate(str, _), do: str
end
