defmodule OptimalEngine.Connectors.Adapters.SourcesFolder do
  @moduledoc """
  Local filesystem connector — walks a configured root directory, finds
  Markdown evidence files, and feeds each not-yet-ingested file through
  the engine intake pipeline.

  Unlike the SaaS connectors (Slack, Gmail, …) this adapter reads no
  external API: its "external system" is a folder tree on disk. It is the
  bridge that makes a local raw-evidence backlog flow into the engine.

  ## Required config keys

  None. The adapter falls back to sane defaults so it works with an empty
  config map.

  ## Config

    * `:root` — directory to walk.
      Defaults to `OPTIMAL_ENGINE_SOURCES_ROOT`, then `./sources`.
    * `:glob` — file extension to match (default `".md"`).
    * `:default_genre` — genre stamped on files that match no genre rule
      (default `"note"`).
    * `:genre_map` — list of `%{"prefix" => relative_prefix, "genre" => g}`
      rules. The first rule whose `prefix` matches the file's path
      (relative to `root`) wins. Default maps `meetings/` → `"transcript"`,
      `highlight/` → `"transcript"`.

  ## Cursor shape

  JSON: `{"ingested": ["meetings/2026-04-01-foo.md", ...]}`. The cursor is
  the dedupe ledger — files already listed are skipped on re-run, so the
  connector is idempotent. The Runner persists the returned cursor only on
  a successful sync.

  ## Transform

  `transform/1` maps a payload `%{"path" => abs, "rel" => rel, "content"
  => text, "genre" => g}` to a `%Signal{}` whose `content` carries the
  file body and `genre` carries the routed genre. The PullScheduler sink
  re-drives each signal through `Pipeline.Intake.process/2`, so the file's
  text flows through classify → route → store like any other ingest.
  """

  use OptimalEngine.Connectors.Adapters.Base,
    kind: :sources_folder,
    display_name: "Sources Folder",
    auth_scheme: :token,
    required_keys: [],
    credential_keys: []

  @default_root System.get_env("OPTIMAL_ENGINE_SOURCES_ROOT", Path.expand("sources", File.cwd!()))
  @default_glob ".md"
  @default_genre "note"
  @default_genre_map [
    %{"prefix" => "meetings/", "genre" => "transcript"},
    %{"prefix" => "highlight/", "genre" => "transcript"}
  ]
  # Bound the batch so a cold first run over ~1264 files doesn't block the
  # Runner indefinitely; the cursor lets the next tick pick up the rest.
  @default_batch_limit 50

  def hydrate_state(config) do
    %{
      root: pick(config, :root, @default_root),
      glob: pick(config, :glob, @default_glob),
      default_genre: pick(config, :default_genre, @default_genre),
      genre_map: normalize_genre_map(pick(config, :genre_map, @default_genre_map)),
      batch_limit: pick(config, :batch_limit, @default_batch_limit)
    }
  end

  @impl true
  def sync(state, cursor) do
    ingested = decode_cursor(cursor)

    case list_markdown(state.root, state.glob) do
      {:error, reason} ->
        {:error, reason}

      {:ok, abs_paths} ->
        pending =
          abs_paths
          |> Enum.map(fn abs -> {abs, relativize(abs, state.root)} end)
          |> Enum.reject(fn {_abs, rel} -> MapSet.member?(ingested, rel) end)
          |> Enum.sort_by(fn {_abs, rel} -> rel end)
          |> Enum.take(state.batch_limit)

        {payloads, newly_ingested} = build_payloads(pending, state)

        signals =
          payloads
          |> Enum.map(&transform/1)
          |> Enum.flat_map(fn
            {:ok, signal} -> [signal]
            _ -> []
          end)

        next_cursor =
          ingested
          |> MapSet.union(MapSet.new(newly_ingested))
          |> encode_cursor()

        # Map form: the Runner handles `{:ok, %{signals:, cursor:, payloads:}}`
        # natively (the bare 4-tuple is declared in the behaviour but not yet
        # destructured by the Runner — use the supported shape).
        {:ok, %{signals: signals, cursor: next_cursor, payloads: payloads}}
    end
  end

  @impl true
  def transform(%{"rel" => rel, "content" => content} = raw) do
    genre = raw["genre"] || @default_genre
    title = derive_title(rel, content)

    {:ok,
     Transform.new_signal(%{
       id: Transform.signal_id(:sources_folder, rel),
       title: title,
       content: content,
       path: Transform.source_uri(:sources_folder, rel),
       genre: genre,
       modified_at: raw["modified_at"] || DateTime.utc_now()
     })}
  end

  def transform(_), do: {:error, :invalid_payload}

  # ── helpers ──────────────────────────────────────────────────────────

  defp build_payloads(pending, state) do
    Enum.reduce(pending, {[], []}, fn {abs, rel}, {payloads, ingested} ->
      case File.read(abs) do
        {:ok, content} ->
          payload = %{
            "path" => abs,
            "rel" => rel,
            "content" => content,
            "genre" => genre_for(rel, state),
            "modified_at" => mtime(abs)
          }

          {[payload | payloads], [rel | ingested]}

        {:error, _reason} ->
          # Unreadable file: leave it OUT of the ingested set so a later
          # run retries it rather than silently dropping it forever.
          {payloads, ingested}
      end
    end)
  end

  defp list_markdown(root, glob) do
    if File.dir?(root) do
      paths =
        Path.join([root, "**", "*" <> glob])
        |> Path.wildcard()
        |> Enum.filter(&File.regular?/1)

      {:ok, paths}
    else
      {:error, {:root_not_found, root}}
    end
  end

  defp genre_for(rel, state) do
    Enum.find_value(state.genre_map, state.default_genre, fn %{"prefix" => prefix, "genre" => g} ->
      if String.starts_with?(rel, prefix), do: g
    end)
  end

  defp normalize_genre_map(map) when is_list(map) do
    Enum.flat_map(map, fn
      %{"prefix" => p, "genre" => g} -> [%{"prefix" => p, "genre" => g}]
      %{prefix: p, genre: g} -> [%{"prefix" => p, "genre" => g}]
      _ -> []
    end)
  end

  defp normalize_genre_map(_), do: @default_genre_map

  defp relativize(abs, root) do
    abs
    |> Path.relative_to(root)
    |> String.trim_leading("/")
  end

  defp derive_title(rel, content) do
    case Regex.run(~r/^\s*#\s+(.+)$/m, content) do
      [_, heading] -> String.slice(String.trim(heading), 0, 120)
      _ -> rel |> Path.basename(Path.extname(rel)) |> String.slice(0, 120)
    end
  end

  defp mtime(abs) do
    case File.stat(abs, time: :posix) do
      {:ok, %File.Stat{mtime: secs}} -> DateTime.from_unix!(secs)
      _ -> DateTime.utc_now()
    end
  end

  defp decode_cursor(nil), do: MapSet.new()
  defp decode_cursor(""), do: MapSet.new()

  defp decode_cursor(cursor) when is_binary(cursor) do
    case Jason.decode(cursor) do
      {:ok, %{"ingested" => list}} when is_list(list) -> MapSet.new(list)
      _ -> MapSet.new()
    end
  end

  defp encode_cursor(set) do
    Jason.encode!(%{"ingested" => Enum.sort(MapSet.to_list(set))})
  end
end
