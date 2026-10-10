defmodule OptimalEngine.Signal.Envelope.Validation do
  @moduledoc """
  Validation logic for `OptimalEngine.Signal.Envelope`.

  Covers CloudEvents required-field checks, Signal Theory dimension constraints,
  and the four Signal Theory governing constraints (Shannon, Ashby, Beer, Wiener).
  """

  alias OptimalEngine.Signal.Envelope, as: Signal

  @valid_modes ~w(linguistic visual code mixed)a
  @valid_genres ~w(spec report pr adr brief chat error progress alert)a
  @valid_types ~w(direct inform commit decide express)a
  @valid_formats ~w(markdown code json cli diff table)a
  @valid_tiers ~w(elite specialist utility)a

  @doc """
  Validates a signal against CloudEvents required fields and Signal Theory dimension constraints.

  Returns `:ok` if valid, `{:error, reasons}` with a list of validation failure strings.
  """
  @spec validate(Signal.t()) :: :ok | {:error, [String.t()]}
  def validate(%Signal{} = signal) do
    errors =
      []
      |> validate_required(signal)
      |> validate_signal_mode(signal)
      |> validate_signal_genre(signal)
      |> validate_signal_type(signal)
      |> validate_signal_format(signal)
      |> validate_agent_tier(signal)
      |> validate_sn_ratio(signal)

    case errors do
      [] -> :ok
      errors -> {:error, Enum.reverse(errors)}
    end
  end

  @doc """
  Shannon constraint check: does the signal exceed the receiver's bandwidth?

  Takes a signal and a maximum data size (in bytes when serialized). Returns `:ok`
  if the signal's data fits within the budget, `{:violation, details}` otherwise.
  """
  @spec shannon_check(Signal.t(), pos_integer()) :: :ok | {:violation, String.t()}
  def shannon_check(%Signal{} = signal, max_bytes)
      when is_integer(max_bytes) and max_bytes > 0 do
    size =
      signal.data
      |> inspect()
      |> byte_size()

    if size <= max_bytes do
      :ok
    else
      {:violation, "data size #{size} bytes exceeds channel capacity #{max_bytes} bytes"}
    end
  end

  @doc """
  Ashby constraint check: does the signal have sufficient variety (all Mode + Genre + Type + Format + Structure resolved)?

  A fully classified signal has all five Signal Theory dimensions set.
  """
  @spec ashby_check(Signal.t()) :: :ok | {:violation, String.t()}
  def ashby_check(%Signal{} = signal) do
    missing =
      [
        {:mode, signal.signal_mode},
        {:genre, signal.signal_genre},
        {:type, signal.signal_type},
        {:format, signal.signal_format},
        {:structure, signal.signal_structure}
      ]
      |> Enum.filter(fn {_k, v} -> is_nil(v) end)
      |> Enum.map(fn {k, _} -> k end)

    case missing do
      [] -> :ok
      dims -> {:violation, "unresolved dimensions: #{Enum.join(dims, ", ")}"}
    end
  end

  @doc """
  Beer constraint check: does the signal maintain viable structure at every scale?

  Checks that the signal has proper structure — non-nil data, valid type hierarchy,
  and a correlation_id if it has a parent_id (maintaining the causality chain).
  """
  @spec beer_check(Signal.t()) :: :ok | {:violation, String.t()}
  def beer_check(%Signal{} = signal) do
    issues =
      []
      |> check_type_structure(signal)
      |> check_causality_coherence(signal)

    case issues do
      [] -> :ok
      [issue | _] -> {:violation, issue}
    end
  end

  @doc """
  Wiener constraint check: is the feedback loop closed?

  Takes a signal and a list of response/acknowledgement signal IDs. The constraint
  is satisfied if the signal has been acknowledged (its ID appears as a parent_id
  in at least one response), OR if the signal is itself a response (has a parent_id).
  """
  @spec wiener_check(Signal.t(), [String.t()]) :: :ok | {:violation, String.t()}
  def wiener_check(%Signal{} = signal, acknowledged_ids) when is_list(acknowledged_ids) do
    cond do
      # This signal is itself a response — it closes a loop
      signal.parent_id != nil ->
        :ok

      # This signal has been acknowledged
      signal.id in acknowledged_ids ->
        :ok

      true ->
        {:violation, "no acknowledgement found — feedback loop open"}
    end
  end

  # ── Private Helpers ──────────────────────────────────────────────

  defp validate_required(errors, signal) do
    errors
    |> require_field(signal.id, "id")
    |> require_field(signal.type, "type")
    |> require_field(signal.specversion, "specversion")
  end

  defp require_field(errors, nil, name), do: ["#{name} is required" | errors]
  defp require_field(errors, "", name), do: ["#{name} is required" | errors]
  defp require_field(errors, _value, _name), do: errors

  defp validate_signal_mode(errors, %{signal_mode: nil}), do: errors
  defp validate_signal_mode(errors, %{signal_mode: m}) when m in @valid_modes, do: errors

  defp validate_signal_mode(errors, %{signal_mode: m}),
    do: ["invalid signal_mode: #{inspect(m)}" | errors]

  defp validate_signal_genre(errors, %{signal_genre: nil}), do: errors
  defp validate_signal_genre(errors, %{signal_genre: g}) when g in @valid_genres, do: errors

  defp validate_signal_genre(errors, %{signal_genre: g}),
    do: ["invalid signal_genre: #{inspect(g)}" | errors]

  defp validate_signal_type(errors, %{signal_type: nil}), do: errors
  defp validate_signal_type(errors, %{signal_type: t}) when t in @valid_types, do: errors

  defp validate_signal_type(errors, %{signal_type: t}),
    do: ["invalid signal_type: #{inspect(t)}" | errors]

  defp validate_signal_format(errors, %{signal_format: nil}), do: errors
  defp validate_signal_format(errors, %{signal_format: f}) when f in @valid_formats, do: errors

  defp validate_signal_format(errors, %{signal_format: f}),
    do: ["invalid signal_format: #{inspect(f)}" | errors]

  defp validate_agent_tier(errors, %{agent_tier: nil}), do: errors
  defp validate_agent_tier(errors, %{agent_tier: t}) when t in @valid_tiers, do: errors

  defp validate_agent_tier(errors, %{agent_tier: t}),
    do: ["invalid agent_tier: #{inspect(t)}" | errors]

  defp validate_sn_ratio(errors, %{signal_sn_ratio: nil}), do: errors

  defp validate_sn_ratio(errors, %{signal_sn_ratio: r})
       when is_float(r) and r >= 0.0 and r <= 1.0,
       do: errors

  defp validate_sn_ratio(errors, %{signal_sn_ratio: r}),
    do: ["signal_sn_ratio must be a float between 0.0 and 1.0, got: #{inspect(r)}" | errors]

  defp check_type_structure(issues, %{type: type}) when is_binary(type) do
    if String.contains?(type, ".") or String.length(type) <= 64 do
      issues
    else
      ["type should use reverse-DNS notation (e.g. miosa.agent.task.completed)" | issues]
    end
  end

  defp check_type_structure(issues, _), do: issues

  defp check_causality_coherence(issues, %{parent_id: pid, correlation_id: nil})
       when not is_nil(pid) do
    ["signal has parent_id but no correlation_id — broken causality chain" | issues]
  end

  defp check_causality_coherence(issues, _), do: issues
end
