defmodule OptimalEngine.Workspace.Config do
  @moduledoc """
  Per-workspace configuration persisted at `<workspace_path>/.optimal/config.yaml`.

  ## Usage

      # Read full config (merged defaults + on-disk)
      {:ok, cfg} = Config.get("engineering")

      # Read one section
      viz = Config.get_section("engineering", :visualizations)

      # Write / update
      :ok = Config.put("engineering", %{grep: %{max_results: 50}})

  ## Schema

  The canonical schema is returned by `defaults/0`. On-disk values deep-merge
  over the defaults so partial config files are always valid.

  ## YAML round-trip

  `yaml_elixir` (our YAML dep) is read-only. Writes are serialized by the
  minimal `to_yaml/1` encoder implemented here, which handles the exact data
  types that appear in this config schema (nested atom-key maps, string lists,
  integers, booleans, nil). Atom keys are written as their string equivalents
  so the file is clean YAML; on read `YamlElixir` returns string keys, which
  `get/2` atomises one level deep (section keys only) before returning.
  """

  alias OptimalEngine.Workspace.Filesystem

  @config_rel_path Path.join([".optimal", "config.yaml"])

  # ── Public API ──────────────────────────────────────────────────────────────

  @doc """
  Returns the default config map applied to every workspace that has no
  (or a partial) `config.yaml` on disk.
  """
  @spec defaults() :: map()
  def defaults do
    %{
      visualizations: %{
        enabled: ["timeline", "heatmap", "graph", "contradictions"],
        timeline: %{group_by: "intent", default_window_days: 30},
        heatmap: %{group_by: "node", granularity: "week"}
      },
      profile: %{
        default_audience: "default",
        include_archived: false,
        recent_chunks_limit: 20
      },
      grep: %{
        default_scale: "paragraph",
        literal_threshold: 0.8,
        max_results: 25
      },
      contradictions: %{
        policy: "flag_for_review",
        auto_dismiss_days: 30
      },
      retention: %{
        default_ttl_days: nil,
        archive_after_days: 365
      },
      memory: %{
        extract_from_wiki: false,
        auto_promote_to_wiki: false,
        dedup_window_days: 30,
        dedup_policy: "return_existing"
      },
      rate_limit: %{
        requests_per_minute: 100,
        burst_capacity: 200,
        exempt_paths: ["/api/status", "/api/health"]
      },
      curation: %{
        enabled: true,
        interval_minutes: 60,
        max_stale_days: 7,
        max_pages_per_cycle: 5
      }
    }
  end

  @doc """
  Reads the workspace config, merging on-disk values over the defaults.

  Returns `{:ok, merged_map}` — even when no `config.yaml` exists (defaults
  are returned). Returns `{:error, reason}` only on unexpected filesystem or
  parse errors (e.g. unreadable file, malformed YAML).
  """
  @spec get(String.t(), String.t()) :: {:ok, map()} | {:error, term()}
  def get(workspace_slug, root \\ File.cwd!()) do
    config_path = config_path(workspace_slug, root)

    case read_yaml(config_path) do
      {:ok, nil} ->
        # File absent — return defaults unmodified
        {:ok, defaults()}

      {:ok, disk_map} when is_map(disk_map) ->
        merged = deep_merge(defaults(), atomise_keys(disk_map))
        {:ok, merged}

      {:ok, _unexpected} ->
        {:ok, defaults()}

      {:error, :enoent} ->
        {:ok, defaults()}

      {:error, _} = err ->
        err
    end
  end

  @doc """
  Writes `config_map` to `<workspace_path>/.optimal/config.yaml`.

  The map is deep-merged with the existing on-disk file first (so a caller
  that PATCHes one section doesn't wipe the others), then the full merged
  map is serialised. Atom or string keys are both accepted.

  Returns `:ok` or `{:error, reason}`.
  """
  @spec put(String.t(), map(), String.t()) :: :ok | {:error, term()}
  def put(workspace_slug, config_map, root \\ File.cwd!()) do
    config_path = config_path(workspace_slug, root)

    with {:ok, current} <- get(workspace_slug, root),
         merged = deep_merge(current, atomise_keys(config_map)),
         :ok <- File.mkdir_p(Path.dirname(config_path)),
         :ok <- File.write(config_path, to_yaml(merged)) do
      :ok
    end
  end

  @doc """
  Convenience wrapper: returns one top-level section of the config as a map.

  Falls back to `default` (default `%{}`) when:
  - `get/2` succeeds but the section key is absent, or
  - `get/2` returns an error (non-fatal; returns the default silently).
  """
  @spec get_section(String.t(), atom(), term(), String.t()) :: term()
  def get_section(workspace_slug, section_atom, default \\ %{}, root \\ File.cwd!()) do
    case get(workspace_slug, root) do
      {:ok, cfg} -> Map.get(cfg, section_atom, default)
      {:error, _} -> default
    end
  end

  # ── Helpers ─────────────────────────────────────────────────────────────────

  defp config_path(slug, root) do
    Path.join([Filesystem.path(root, slug), @config_rel_path])
  end

  # Read YAML from `path`. Returns `{:ok, nil}` when file is empty, `{:ok,
  # parsed}` on success, `{:error, :enoent}` when absent, `{:error, reason}`
  # otherwise.
  defp read_yaml(path) do
    case File.read(path) do
      {:ok, ""} ->
        {:ok, nil}

      {:ok, content} ->
        case YamlElixir.read_from_string(content) do
          {:ok, parsed} -> {:ok, parsed}
          {:error, _} = err -> err
        end

      {:error, :enoent} ->
        {:error, :enoent}

      {:error, reason} ->
        {:error, reason}
    end
  end

  # Deep-merge `right` into `left` (right wins on scalar conflicts).
  # Both maps must have atom keys (call `atomise_keys/1` first when needed).
  defp deep_merge(left, right) when is_map(left) and is_map(right) do
    Map.merge(left, right, fn _key, l, r ->
      if is_map(l) and is_map(r), do: deep_merge(l, r), else: r
    end)
  end

  # Convert all string keys in a (possibly nested) map to atoms. Unknown atom
  # keys coming from user-supplied YAML are safe here: they're section names
  # from a controlled schema. We use `String.to_atom/1` (not
  # `to_existing_atom`) intentionally — workspace config keys are not the same
  # threat surface as arbitrary user-controlled atom creation in a server
  # request path.
  defp atomise_keys(map) when is_map(map) do
    Map.new(map, fn
      {k, v} when is_binary(k) -> {String.to_atom(k), atomise_keys(v)}
      {k, v} when is_atom(k) -> {k, atomise_keys(v)}
    end)
  end

  defp atomise_keys(other), do: other

  # ── YAML Serialiser ─────────────────────────────────────────────────────────
  #
  # `yaml_elixir` is read-only; there is no Hex YAML writer in this project.
  # This minimal serialiser handles exactly the types that appear in the config
  # schema: nested maps (atom or string keys), string lists, integers, floats,
  # booleans, and nil. It is NOT a general YAML encoder.

  @doc false
  def to_yaml(map) when is_map(map) do
    encode_map(map, 0) <> "\n"
  end

  defp encode_map(map, indent) do
    pad = String.duplicate("  ", indent)

    map
    |> Enum.sort_by(fn {k, _} -> to_string(k) end)
    |> Enum.map_join("\n", fn {k, v} ->
      key = to_string(k)

      case v do
        v when is_map(v) and map_size(v) > 0 ->
          "#{pad}#{key}:\n#{encode_map(v, indent + 1)}"

        v when is_list(v) ->
          "#{pad}#{key}:\n#{encode_list(v, indent + 1)}"

        nil ->
          "#{pad}#{key}: null"

        true ->
          "#{pad}#{key}: true"

        false ->
          "#{pad}#{key}: false"

        v when is_integer(v) ->
          "#{pad}#{key}: #{v}"

        v when is_float(v) ->
          "#{pad}#{key}: #{v}"

        v when is_binary(v) ->
          "#{pad}#{key}: #{yaml_quote(v)}"

        v when is_atom(v) ->
          "#{pad}#{key}: #{to_string(v)}"

        _ ->
          "#{pad}#{key}: #{inspect(v)}"
      end
    end)
  end

  defp encode_list([], _indent), do: "  []"

  defp encode_list(list, indent) do
    pad = String.duplicate("  ", indent)

    Enum.map_join(list, "\n", fn
      item when is_map(item) ->
        # Block-sequence map: first key gets the `- ` prefix
        lines = encode_map(item, indent + 1) |> String.split("\n")
        first = "#{pad}- #{String.trim_leading(hd(lines))}"
        rest = Enum.drop(lines, 1)
        Enum.join([first | rest], "\n")

      item ->
        "#{pad}- #{encode_scalar(item)}"
    end)
  end

  defp encode_scalar(nil), do: "null"
  defp encode_scalar(true), do: "true"
  defp encode_scalar(false), do: "false"
  defp encode_scalar(v) when is_integer(v), do: to_string(v)
  defp encode_scalar(v) when is_float(v), do: to_string(v)
  defp encode_scalar(v) when is_binary(v), do: yaml_quote(v)
  defp encode_scalar(v) when is_atom(v), do: to_string(v)
  defp encode_scalar(v), do: inspect(v)

  # Quote a string value when it would be misinterpreted by a YAML parser
  # (contains special chars, looks like a keyword, etc.). We always quote
  # strings for safety — single quotes, with inner single-quotes doubled.
  defp yaml_quote(str) do
    escaped = String.replace(str, "'", "''")
    "'#{escaped}'"
  end
end
