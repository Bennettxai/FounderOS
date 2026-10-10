defmodule OptimalEngine.Connectors.Adapters.Notion do
  @moduledoc """
  Notion connector -- pages, databases, blocks.

  ## Required config keys
    * `:workspace_name`, `:database_ids` (list; empty = all accessible)

  ## Credentials
    * `:integration_token` -- `secret_...`

  ## Cursor shape
  Opaque: page `last_edited_time` ISO timestamp. Initial sync: `nil`.
  """

  use OptimalEngine.Connectors.Adapters.Base,
    kind: :notion,
    display_name: "Notion",
    auth_scheme: :token,
    required_keys: [:workspace_name, :database_ids],
    credential_keys: [:integration_token]

  alias OptimalEngine.Connectors.HTTP

  @base_url "https://api.notion.com/v1"
  @notion_version "2022-06-28"
  @page_size 100

  @impl true
  def sync(state, cursor) do
    token = pick(state, :integration_token)

    headers = [
      {"authorization", "Bearer #{token}"},
      {"notion-version", @notion_version}
    ]

    db_ids = pick(state, :database_ids, [])

    databases =
      case db_ids do
        [] -> fetch_all_databases(headers)
        ids -> ids
      end

    since = cursor

    {signals, next_cursor} =
      Enum.reduce(databases, {[], since}, fn db_id, {acc, _last_ts} ->
        case query_database(db_id, since, nil, headers) do
          {:ok, pages, last_edited} ->
            new_signals =
              Enum.flat_map(pages, fn page ->
                case transform(page) do
                  {:ok, s} -> [s]
                  _ -> []
                end
              end)

            {acc ++ new_signals, last_edited}

          {:error, _} ->
            {acc, since}
        end
      end)

    {:ok, %{signals: signals, cursor: next_cursor}}
  end

  @impl true
  def transform(raw) when is_map(raw) do
    ext_id = raw["id"] || ""
    title = extract_title(raw) || "Untitled"

    {:ok,
     Transform.new_signal(%{
       id: Transform.signal_id(:notion, ext_id),
       title: title,
       content: flatten_blocks(raw["blocks"] || []),
       path: Transform.source_uri(:notion, ext_id),
       genre: "document",
       mode: :linguistic,
       format: :markdown,
       modified_at: Transform.parse_iso8601(raw["last_edited_time"])
     })}
  end

  # ---- private ----------------------------------------------------------------

  defp fetch_all_databases(headers) do
    body = %{"filter" => %{"value" => "database", "property" => "object"}}

    case HTTP.post_json("#{@base_url}/search", body, headers: headers) do
      {:ok, %{status: 200, body: %{"results" => results}}} ->
        Enum.map(results, & &1["id"])

      _ ->
        []
    end
  end

  defp query_database(db_id, since, start_cursor, headers) do
    filter =
      if since do
        %{
          "filter" => %{
            "timestamp" => "last_edited_time",
            "last_edited_time" => %{"after" => since}
          },
          "sorts" => [%{"timestamp" => "last_edited_time", "direction" => "ascending"}],
          "page_size" => @page_size
        }
      else
        %{
          "sorts" => [%{"timestamp" => "last_edited_time", "direction" => "ascending"}],
          "page_size" => @page_size
        }
      end

    body = if start_cursor, do: Map.put(filter, "start_cursor", start_cursor), else: filter

    case HTTP.post_json("#{@base_url}/databases/#{db_id}/query", body, headers: headers) do
      {:ok, %{status: 200, body: %{"results" => results} = resp}} ->
        last_edited =
          results
          |> Enum.map(& &1["last_edited_time"])
          |> Enum.reject(&is_nil/1)
          |> Enum.max(fn -> since end)

        if resp["has_more"] do
          next = resp["next_cursor"]

          case query_database(db_id, since, next, headers) do
            {:ok, more_pages, more_last} ->
              all_last = Enum.max([last_edited, more_last], fn -> last_edited end)
              {:ok, results ++ more_pages, all_last}

            {:error, _} ->
              {:ok, results, last_edited}
          end
        else
          {:ok, results, last_edited}
        end

      {:ok, %{status: 401}} ->
        {:error, :auth_expired}

      {:ok, %{status: 429}} ->
        {:error, :rate_limited}

      {:ok, %{status: status}} ->
        {:error, {:http, status}}

      {:error, reason} ->
        {:error, reason}
    end
  end

  defp extract_title(%{"properties" => %{"Name" => %{"title" => [%{"plain_text" => t} | _]}}}),
    do: t

  defp extract_title(%{"properties" => %{"title" => %{"title" => [%{"plain_text" => t} | _]}}}),
    do: t

  defp extract_title(_), do: nil

  defp flatten_blocks(blocks) when is_list(blocks) do
    blocks
    |> Enum.map_join("\n", fn
      %{"type" => type, "paragraph" => %{"rich_text" => rt}} when type == "paragraph" ->
        Enum.map_join(rt, "", fn %{"plain_text" => t} -> t end)

      %{"type" => "heading_1", "heading_1" => %{"rich_text" => rt}} ->
        "# " <> Enum.map_join(rt, "", fn %{"plain_text" => t} -> t end)

      %{"type" => "heading_2", "heading_2" => %{"rich_text" => rt}} ->
        "## " <> Enum.map_join(rt, "", fn %{"plain_text" => t} -> t end)

      _ ->
        ""
    end)
  end
end
