defmodule OptimalEngine.Connectors.Adapters.Confluence do
  @moduledoc """
  Confluence connector - spaces + pages + blog posts.

  ## Required config keys
    * `:site_url` (e.g. `https://sample.atlassian.net`), `:spaces` (list of space keys; empty = all)

  ## Credentials
    * `:email`, `:api_token`

  ## Cursor shape
  ISO-8601 timestamp string (pages last modified after this). Initial sync: `nil`.
  """

  use OptimalEngine.Connectors.Adapters.Base,
    kind: :confluence,
    display_name: "Confluence",
    auth_scheme: :basic,
    required_keys: [:site_url, :spaces],
    credential_keys: [:email, :api_token]

  alias OptimalEngine.Connectors.HTTP

  @page_size 50

  @impl true
  def sync(state, cursor) do
    site_url = pick(state, :site_url)
    spaces = pick(state, :spaces, [])
    email = pick(state, :email)
    api_token = pick(state, :api_token)

    credentials = Base.encode64("#{email}:#{api_token}")
    headers = [{"authorization", "Basic #{credentials}"}]

    spaces_to_sync =
      case spaces do
        [] -> fetch_all_spaces(site_url, headers)
        list -> list
      end

    result =
      Enum.reduce_while(spaces_to_sync, {:ok, [], cursor}, fn space_key, {:ok, acc, _last_ts} ->
        case fetch_pages(site_url, space_key, cursor, headers) do
          {:ok, pages, new_ts} ->
            new_signals =
              Enum.flat_map(pages, fn page ->
                case transform(page) do
                  {:ok, s} -> [s]
                  _ -> []
                end
              end)

            {:cont, {:ok, acc ++ new_signals, new_ts || cursor}}

          {:error, :auth_expired} ->
            {:halt, {:error, :auth_expired}}

          {:error, :rate_limited} ->
            {:halt, {:error, :rate_limited}}

          {:error, _} ->
            {:cont, {:ok, acc, cursor}}
        end
      end)

    case result do
      {:ok, signals, next_cursor} -> {:ok, %{signals: signals, cursor: next_cursor}}
      {:error, _} = err -> err
    end
  end

  @impl true
  def transform(raw) when is_map(raw) do
    ext_id = raw["id"] || ""

    {:ok,
     Transform.new_signal(%{
       id: Transform.signal_id(:confluence, ext_id),
       title: raw["title"] || "Untitled",
       content: Transform.strip_html(get_in(raw, ["body", "storage", "value"]) || ""),
       path: Transform.source_uri(:confluence, ext_id),
       genre: "document",
       mode: :linguistic,
       modified_at: Transform.parse_iso8601(get_in(raw, ["version", "when"]))
     })}
  end

  # ---- private ----------------------------------------------------------------

  defp fetch_all_spaces(site_url, headers) do
    url = "#{site_url}/wiki/rest/api/space?limit=50&type=global"

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: %{"results" => spaces}}} ->
        Enum.map(spaces, & &1["key"])

      _ ->
        []
    end
  end

  defp fetch_pages(site_url, space_key, cursor, headers) do
    params =
      %{
        "spaceKey" => space_key,
        "limit" => @page_size,
        "expand" => "body.storage,version",
        "status" => "current"
      }
      |> maybe_add_cursor(cursor)
      |> URI.encode_query()

    url = "#{site_url}/wiki/rest/api/content?#{params}"

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: %{"results" => pages}}} ->
        new_ts =
          pages
          |> Enum.map(fn p -> get_in(p, ["version", "when"]) end)
          |> Enum.reject(&is_nil/1)
          |> Enum.max(fn -> nil end)

        {:ok, pages, new_ts}

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

  defp maybe_add_cursor(params, nil), do: params

  defp maybe_add_cursor(params, cursor) when is_binary(cursor) do
    # Use lastModified filter via CQL if supported; fall back to simple start
    Map.put(params, "start", 0)
    |> Map.put(
      "cql",
      "space = \"#{params["spaceKey"]}\" AND lastModified > \"#{cursor}\" ORDER BY lastModified DESC"
    )
    |> Map.delete("spaceKey")
  end
end
