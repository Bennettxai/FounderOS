defmodule Mix.Tasks.Optimal.IngestWorkspace do
  @shortdoc "Walk a workspace directory and ingest every signal file"

  @moduledoc """
  Walks a workspace directory laid out per the `sample-workspace/`
  convention and ingests every signal into the engine:

      <workspace>/
      ├── nodes/
      │   └── <node-slug>/
      │       ├── context.md
      │       ├── signal.md
      │       └── signals/YYYY-MM-DD-slug.md
      └── .wiki/
          └── <slug>.md

  For each `signals/*.md` file the task:

    1. Parses YAML frontmatter + markdown body.
    2. Derives the node from the file's directory path.
    3. Creates a signal row (`type='signal'`) in `contexts`.
    4. Extracts entities declared in the frontmatter `entities` list
       into the `entities` table.
    5. Writes an `ingest` event to the audit log.

  Wiki pages under `.wiki/` are written via `Wiki.put/1` as projection pages.
  They are useful for browsing, but they are not canonical truth unless their
  contents re-enter the engine as source-backed Claims/Facts.

  ## Usage

      mix optimal.ingest_workspace sample-workspace/
      mix optimal.ingest_workspace ~/Desktop/my-engine --tenant sample
      mix optimal.ingest_workspace sample-workspace/ --workspace sample-q1
      mix optimal.ingest_workspace sample-workspace/ --reset
  """

  use Mix.Task

  alias OptimalEngine.Store
  alias OptimalEngine.Wiki
  alias OptimalEngine.Wiki.Page

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {parsed, rest, _} =
      OptionParser.parse(args,
        strict: [tenant: :string, workspace: :string, reset: :boolean]
      )

    root =
      case rest do
        [r | _] ->
          Path.expand(r)

        [] ->
          Mix.raise(
            "Usage: mix optimal.ingest_workspace <path> [--tenant id] [--workspace id] [--reset]"
          )
      end

    unless File.dir?(root), do: Mix.raise("Not a directory: #{root}")

    tenant = Keyword.get(parsed, :tenant, "default")
    workspace_id = Keyword.get(parsed, :workspace, "default")
    reset? = Keyword.get(parsed, :reset, false)

    if reset?, do: wipe_workspace_rows(tenant, workspace_id)

    stats = %{
      nodes: 0,
      signals: 0,
      wiki_pages: 0,
      entities: 0,
      skipped: 0
    }

    stats =
      stats
      |> ingest_nodes(root, tenant, workspace_id)
      |> ingest_signals(root, tenant, workspace_id)
      |> ingest_wiki(root, tenant, workspace_id)

    IO.puts("""

    Workspace ingested.
      root:        #{root}
      tenant:      #{tenant}
      workspace:   #{workspace_id}
      nodes:       #{stats.nodes}
      signals:     #{stats.signals}
      entities:    #{stats.entities}
      wiki pages:  #{stats.wiki_pages}
      skipped:     #{stats.skipped}

    Try:
      mix optimal.rag "customer portal pricing" --trace
      mix optimal.search "platform"
      mix optimal.wiki list
    """)
  end

  # ─── nodes ──────────────────────────────────────────────────────────────

  defp ingest_nodes(stats, root, tenant, workspace_id) do
    nodes_dir = Path.join(root, "nodes")
    if not File.dir?(nodes_dir), do: Map.update!(stats, :skipped, &(&1 + 1))

    node_dirs =
      if File.dir?(nodes_dir),
        do: nodes_dir |> File.ls!() |> Enum.filter(&File.dir?(Path.join(nodes_dir, &1))),
        else: []

    Enum.reduce(node_dirs, stats, fn slug, acc ->
      ctx_path = Path.join([nodes_dir, slug, "context.md"])

      {name, kind, style, parent_slug} =
        case read_frontmatter(ctx_path) do
          {:ok, fm, body} ->
            {fm["name"] || fm["title"] || markdown_heading(body) || slug,
             fm["kind"] || markdown_identity(body, "Node type") || "context",
             fm["style"] || "internal", fm["parent"] || markdown_identity(body, "Parent")}

          _ ->
            {slug, "node", "internal", nil}
        end

      id = "#{tenant}:#{workspace_id}:#{slug}"
      parent_id = if parent_slug, do: "#{tenant}:#{workspace_id}:#{parent_slug}", else: nil

      Store.raw_query(
        """
        INSERT INTO nodes (id, tenant_id, slug, name, kind, parent_id, path, style, workspace_id, metadata)
        VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?3, ?7, ?8, '{"tag":"workspace-ingest"}')
        ON CONFLICT(id) DO UPDATE SET
          name = excluded.name,
          kind = excluded.kind,
          parent_id = excluded.parent_id,
          style = excluded.style,
          workspace_id = excluded.workspace_id
        """,
        [id, tenant, slug, name, kind, parent_id, style, workspace_id]
      )

      Map.update!(acc, :nodes, &(&1 + 1))
    end)
  end

  # ─── signals ────────────────────────────────────────────────────────────

  defp ingest_signals(stats, root, tenant, workspace_id) do
    signal_files =
      Path.wildcard(Path.join([root, "nodes", "*", "signals", "*.md"]))

    Enum.reduce(signal_files, stats, fn path, acc ->
      case read_frontmatter(path) do
        {:ok, fm, body} ->
          rel = Path.relative_to(path, root)

          node_slug =
            fm["node"] ||
              case Path.split(rel) do
                ["nodes", slug | _] -> slug
                _ -> "inbox"
              end

          path_hash =
            :crypto.hash(:sha256, path)
            |> Base.encode16(case: :lower)
            |> String.slice(0, 12)

          sig_id = "ws:" <> path_hash <> ":" <> Path.basename(path, ".md")

          uri = "optimal://nodes/#{node_slug}/signals/#{Path.basename(path)}"

          {summary, overview} = derive_tiers(body)

          case Store.raw_query(
                 """
                 INSERT INTO contexts (
                   id, tenant_id, uri, type, path, title, l0_abstract, l1_overview,
                   content, genre, mode, signal_type, format, structure, node,
                   sn_ratio, entities, created_at, modified_at, workspace_id, metadata
                 )
                 VALUES (?1, ?2, ?3, 'signal', ?4, ?5, ?6, ?7, ?8, ?9, ?10,
                         'inform', 'markdown', 'prose', ?11, ?12, '[]',
                         COALESCE(?13, datetime('now')), datetime('now'), ?14,
                         '{"tag":"workspace-ingest"}')
                 ON CONFLICT(id) DO UPDATE SET
                   content = excluded.content,
                   title = excluded.title,
                   workspace_id = excluded.workspace_id,
                   modified_at = datetime('now')
                 """,
                 [
                   sig_id,
                   tenant,
                   uri,
                   path,
                   fm["title"] || Path.basename(path, ".md"),
                   summary,
                   overview,
                   body,
                   fm["genre"] || "note",
                   fm["mode"] || "linguistic",
                   node_slug,
                   to_float(fm["sn_ratio"]) || 0.6,
                   fm["authored_at"],
                   workspace_id
                 ]
               ) do
            {:ok, _} ->
              # Entities from frontmatter
              entity_count =
                ingest_entities(sig_id, fm["entities"] || [], workspace_id)

              # Audit event
              Store.raw_query(
                """
                INSERT INTO events (tenant_id, principal, kind, target_uri, workspace_id, metadata)
                VALUES (?1, 'system:workspace-ingest', 'ingest', ?2, ?3,
                        '{"source":"workspace"}')
                """,
                [tenant, uri, workspace_id]
              )

              acc
              |> Map.update!(:signals, &(&1 + 1))
              |> Map.update!(:entities, &(&1 + entity_count))

            _ ->
              Map.update!(acc, :skipped, &(&1 + 1))
          end

        _ ->
          IO.puts("  skip (bad frontmatter): #{path}")
          Map.update!(acc, :skipped, &(&1 + 1))
      end
    end)
  end

  defp ingest_entities(signal_id, entities, workspace_id) when is_list(entities) do
    Enum.reduce(entities, 0, fn e, acc ->
      {name, type} = extract_entity(e)

      if is_binary(name) and name != "" do
        Store.raw_query(
          "INSERT OR IGNORE INTO entities (context_id, name, type, workspace_id) VALUES (?1, ?2, ?3, ?4)",
          [signal_id, name, type || "concept", workspace_id]
        )

        acc + 1
      else
        acc
      end
    end)
  end

  defp ingest_entities(_, _, _), do: 0

  defp extract_entity(%{"name" => n, "type" => t}), do: {n, t}
  defp extract_entity(%{"name" => n}), do: {n, "concept"}
  defp extract_entity(n) when is_binary(n), do: {n, "concept"}
  defp extract_entity(_), do: {nil, nil}

  # ─── wiki ───────────────────────────────────────────────────────────────

  defp ingest_wiki(stats, root, tenant, workspace_id) do
    wiki_files = Path.wildcard(Path.join([root, ".wiki", "*.md"]))

    Enum.reduce(wiki_files, stats, fn path, acc ->
      base = Path.basename(path, ".md")
      # SCHEMA.md is governance, not a wiki page.
      if base == "SCHEMA" do
        acc
      else
        case read_frontmatter(path) do
          {:ok, fm, body} ->
            page = %Page{
              tenant_id: tenant,
              workspace_id: workspace_id,
              slug: fm["slug"] || base,
              audience: fm["audience"] || "default",
              version: to_int(fm["version"]) || 1,
              frontmatter: fm,
              body: body,
              last_curated: fm["last_curated"] || DateTime.utc_now() |> DateTime.to_iso8601(),
              curated_by: fm["curated_by"] || "workspace-ingest"
            }

            case Wiki.put(page) do
              :ok -> Map.update!(acc, :wiki_pages, &(&1 + 1))
              _ -> Map.update!(acc, :skipped, &(&1 + 1))
            end

          _ ->
            Map.update!(acc, :skipped, &(&1 + 1))
        end
      end
    end)
  end

  # ─── helpers ────────────────────────────────────────────────────────────

  # Split `--- yaml --- body` and hand the yaml to :yaml_elixir.
  defp read_frontmatter(path) do
    case File.read(path) do
      {:ok, raw} -> parse_frontmatter(raw)
      _ -> :error
    end
  end

  defp parse_frontmatter("---\n" <> rest) do
    case String.split(rest, "\n---\n", parts: 2) do
      [yaml, body] ->
        case YamlElixir.read_from_string(yaml) do
          {:ok, fm} when is_map(fm) -> {:ok, fm, String.trim_leading(body)}
          _ -> {:ok, %{}, String.trim_leading(body)}
        end

      _ ->
        {:ok, %{}, rest}
    end
  end

  defp parse_frontmatter(body), do: {:ok, %{}, body}

  # The decomposer eventually produces proper L0 / L1 from the parsed
  # markdown; for ingest we just grab the first "## Summary" block for
  # L0 and the first N paragraphs for L1. Stand-in until the pipeline
  # runs over the ingested rows.
  defp derive_tiers(body) do
    summary =
      case Regex.run(~r/##\s+Summary\s*\n+(.+?)(?=\n##|\z)/s, body) do
        [_, s] -> String.trim(s)
        _ -> body |> String.split("\n\n") |> List.first() |> to_string() |> String.trim()
      end
      |> String.slice(0, 400)

    overview = String.slice(body, 0, 2000)

    {summary, overview}
  end

  defp markdown_heading(body) do
    case Regex.run(~r/^#\s+(.+)$/m, body) do
      [_, heading] -> String.trim(heading)
      _ -> nil
    end
  end

  defp markdown_identity(body, label) do
    pattern = ~r/^-\s+#{Regex.escape(label)}:\s+`?([^`\n]+)`?\s*$/mi

    case Regex.run(pattern, body) do
      [_, value] -> String.trim(value)
      _ -> nil
    end
  end

  defp to_float(nil), do: nil
  defp to_float(n) when is_number(n), do: n / 1

  defp to_float(s) when is_binary(s) do
    case Float.parse(s) do
      {f, _} -> f
      _ -> nil
    end
  end

  defp to_int(nil), do: nil
  defp to_int(n) when is_integer(n), do: n

  defp to_int(s) when is_binary(s) do
    case Integer.parse(s) do
      {n, _} -> n
      _ -> nil
    end
  end

  defp wipe_workspace_rows(tenant, workspace_id) do
    Store.raw_query(
      "DELETE FROM contexts WHERE tenant_id = ?1 AND workspace_id = ?2 AND id LIKE 'ws:%'",
      [tenant, workspace_id]
    )

    Store.raw_query(
      "DELETE FROM entities WHERE context_id LIKE 'ws:%'",
      []
    )

    Store.raw_query(
      "DELETE FROM nodes WHERE tenant_id = ?1 AND workspace_id = ?2 AND json_extract(metadata, '$.tag') = 'workspace-ingest'",
      [tenant, workspace_id]
    )

    Store.raw_query(
      "DELETE FROM events WHERE tenant_id = ?1 AND workspace_id = ?2 AND principal = 'system:workspace-ingest'",
      [tenant, workspace_id]
    )
  end
end
