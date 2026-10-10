defmodule OptimalEngine.Signal.Envelope.Builder do
  @moduledoc """
  CloudEvents serialization and deserialization for `OptimalEngine.Signal.Envelope`.

  Handles conversion between the Signal struct and the CloudEvents v1.0.2
  JSON-compatible map format. Signal Theory and MIOSA extension attributes
  are mapped to/from the `miosa_` prefix per CloudEvents extension naming rules.
  """

  alias OptimalEngine.Signal.Envelope, as: Signal
  alias OptimalEngine.Signal.Envelope.Validation

  @valid_modes ~w(linguistic visual code mixed)a
  @valid_genres ~w(spec report pr adr brief chat error progress alert)a
  @valid_types ~w(direct inform commit decide express)a
  @valid_formats ~w(markdown code json cli diff table)a
  @valid_tiers ~w(elite specialist utility)a

  @doc """
  Serializes a signal to CloudEvents v1.0.2 JSON-compatible map.

  Signal Theory and MIOSA extensions are placed under the `miosa_` prefix
  as CloudEvents extension attributes.

  ## Examples

      iex> signal = OptimalEngine.Signal.Envelope.new!("miosa.test", source: "/test", data: %{x: 1})
      iex> ce = OptimalEngine.Signal.Envelope.Builder.to_cloud_event(signal)
      iex> ce["type"]
      "miosa.test"
      iex> ce["specversion"]
      "1.0.2"
  """
  @spec to_cloud_event(Signal.t()) :: map()
  def to_cloud_event(%Signal{} = signal) do
    base =
      %{
        "specversion" => signal.specversion,
        "id" => signal.id,
        "source" => signal.source,
        "type" => signal.type
      }
      |> put_if("time", format_time(signal.time))
      |> put_if("subject", signal.subject)
      |> put_if("data", signal.data)
      |> put_if("datacontenttype", signal.datacontenttype)
      |> put_if("dataschema", signal.dataschema)

    # MIOSA extensions use miosa_ prefix per CloudEvents extension naming
    extensions =
      %{}
      |> put_if("miosa_signal_mode", safe_to_string(signal.signal_mode))
      |> put_if("miosa_signal_genre", safe_to_string(signal.signal_genre))
      |> put_if("miosa_signal_type", safe_to_string(signal.signal_type))
      |> put_if("miosa_signal_format", safe_to_string(signal.signal_format))
      |> put_if("miosa_signal_structure", safe_to_string(signal.signal_structure))
      |> put_if("miosa_signal_sn_ratio", signal.signal_sn_ratio)
      |> put_if("miosa_agent_id", signal.agent_id)
      |> put_if("miosa_agent_tier", safe_to_string(signal.agent_tier))
      |> put_if("miosa_session_id", signal.session_id)
      |> put_if("miosa_parent_id", signal.parent_id)
      |> put_if("miosa_correlation_id", signal.correlation_id)

    custom_ext =
      Map.new(signal.extensions || %{}, fn {k, v} ->
        {"miosa_ext_#{k}", v}
      end)

    Map.merge(base, Map.merge(extensions, custom_ext))
  end

  @doc """
  Deserializes a CloudEvents JSON map into a Signal struct.

  Extracts `miosa_*` prefixed attributes as Signal Theory and MIOSA extensions.

  ## Examples

      iex> signal = OptimalEngine.Signal.Envelope.new!("miosa.test", source: "/test")
      iex> ce = OptimalEngine.Signal.Envelope.Builder.to_cloud_event(signal)
      iex> {:ok, restored} = OptimalEngine.Signal.Envelope.Builder.from_cloud_event(ce)
      iex> restored.type
      "miosa.test"
  """
  @spec from_cloud_event(map()) :: {:ok, Signal.t()} | {:error, term()}
  def from_cloud_event(map) when is_map(map) do
    time = parse_time(map["time"])

    custom_extensions =
      map
      |> Enum.filter(fn {k, _v} -> String.starts_with?(k, "miosa_ext_") end)
      |> Map.new(fn {"miosa_ext_" <> key, v} -> {key, v} end)

    signal = %Signal{
      id: map["id"],
      source: map["source"],
      type: map["type"],
      specversion: map["specversion"] || "1.0.2",
      time: time,
      subject: map["subject"],
      data: map["data"],
      datacontenttype: map["datacontenttype"] || "application/json",
      dataschema: map["dataschema"],
      signal_mode: safe_to_atom(map["miosa_signal_mode"], @valid_modes),
      signal_genre: safe_to_atom(map["miosa_signal_genre"], @valid_genres),
      signal_type: safe_to_atom(map["miosa_signal_type"], @valid_types),
      signal_format: safe_to_atom(map["miosa_signal_format"], @valid_formats),
      signal_structure: safe_to_existing_atom(map["miosa_signal_structure"]),
      signal_sn_ratio: map["miosa_signal_sn_ratio"],
      agent_id: map["miosa_agent_id"],
      agent_tier: safe_to_atom(map["miosa_agent_tier"], @valid_tiers),
      session_id: map["miosa_session_id"],
      parent_id: map["miosa_parent_id"],
      correlation_id: map["miosa_correlation_id"],
      extensions: custom_extensions
    }

    case Validation.validate(signal) do
      :ok -> {:ok, signal}
      {:error, _} = err -> err
    end
  end

  # ── Private Helpers ──────────────────────────────────────────────

  defp format_time(nil), do: nil
  defp format_time(%DateTime{} = dt), do: DateTime.to_iso8601(dt)

  defp parse_time(nil), do: nil

  defp parse_time(str) when is_binary(str) do
    case DateTime.from_iso8601(str) do
      {:ok, dt, _offset} -> dt
      _ -> nil
    end
  end

  defp parse_time(%DateTime{} = dt), do: dt

  defp put_if(map, _key, nil), do: map
  defp put_if(map, key, value), do: Map.put(map, key, value)

  defp safe_to_string(nil), do: nil
  defp safe_to_string(atom) when is_atom(atom), do: Atom.to_string(atom)
  defp safe_to_string(val), do: to_string(val)

  defp safe_to_atom(nil, _valid), do: nil

  defp safe_to_atom(str, valid) when is_binary(str) do
    atom = String.to_existing_atom(str)
    if atom in valid, do: atom, else: nil
  rescue
    ArgumentError -> nil
  end

  defp safe_to_existing_atom(nil), do: nil

  defp safe_to_existing_atom(str) when is_binary(str) do
    String.to_existing_atom(str)
  rescue
    ArgumentError -> nil
  end
end
