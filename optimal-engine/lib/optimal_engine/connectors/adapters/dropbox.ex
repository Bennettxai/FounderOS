defmodule OptimalEngine.Connectors.Adapters.Dropbox do
  @moduledoc """
  Dropbox connector -- files + folders + shared content.

  ## Required config keys
    * `:namespace` -- `:personal | :team`

  ## Credentials
    * `:refresh_token`, `:app_key`, `:app_secret`

  ## Cursor shape
  Opaque: Dropbox list_folder cursor string. Initial sync: `nil`.
  """

  use OptimalEngine.Connectors.Adapters.Base,
    kind: :dropbox,
    display_name: "Dropbox",
    auth_scheme: :oauth2,
    required_keys: [:namespace],
    credential_keys: [:refresh_token, :app_key, :app_secret]

  alias OptimalEngine.Connectors.HTTP

  @token_url "https://api.dropboxapi.com/oauth2/token"
  @content_url "https://api.dropboxapi.com/2"

  @impl true
  def sync(state, cursor) do
    with {:ok, access_token} <- refresh_access_token(state) do
      headers = [{"authorization", "Bearer #{access_token}"}]

      case cursor do
        nil -> full_sync(state, headers)
        dbx_cursor -> incremental_sync(dbx_cursor, headers)
      end
    end
  end

  @impl true
  def transform(raw) when is_map(raw) do
    ext_id = raw["id"] || raw["path_lower"] || ""

    {:ok,
     Transform.new_signal(%{
       id: Transform.signal_id(:dropbox, ext_id),
       title: raw["name"] || Path.basename(raw["path_display"] || ""),
       content: raw["content"] || "",
       path: Transform.source_uri(:dropbox, ext_id),
       genre: "file",
       modified_at: Transform.parse_iso8601(raw["server_modified"])
     })}
  end

  # ---- private ----------------------------------------------------------------

  defp refresh_access_token(state) do
    body =
      "grant_type=refresh_token" <>
        "&refresh_token=#{pick(state, :refresh_token)}" <>
        "&client_id=#{pick(state, :app_key)}" <>
        "&client_secret=#{pick(state, :app_secret)}"

    case HTTP.request(@token_url,
           method: :post,
           headers: [{"content-type", "application/x-www-form-urlencoded"}],
           body: body,
           parse_json: true
         ) do
      {:ok, %{status: 200, body: %{"access_token" => token}}} ->
        {:ok, token}

      {:ok, %{status: 401}} ->
        {:error, :auth_expired}

      {:ok, %{body: body}} ->
        {:error, {:token_error, body}}

      {:error, reason} ->
        {:error, reason}
    end
  end

  defp full_sync(state, headers) do
    path =
      case pick(state, :namespace) do
        "team" -> ""
        _ -> ""
      end

    body = %{"path" => path, "recursive" => true, "limit" => 500}

    case HTTP.post_json("#{@content_url}/files/list_folder", body, headers: headers) do
      {:ok, %{status: 200, body: resp}} ->
        process_list_folder_response(resp, headers)

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

  defp incremental_sync(cursor, headers) do
    body = %{"cursor" => cursor}

    case HTTP.post_json("#{@content_url}/files/list_folder/continue", body, headers: headers) do
      {:ok, %{status: 200, body: resp}} ->
        process_list_folder_response(resp, headers)

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

  defp process_list_folder_response(resp, headers) do
    entries = resp["entries"] || []

    files =
      Enum.filter(entries, fn e ->
        e[".tag"] == "file"
      end)

    signals =
      Enum.flat_map(files, fn f ->
        case transform(f) do
          {:ok, s} -> [s]
          _ -> []
        end
      end)

    if resp["has_more"] do
      next_cursor = resp["cursor"]

      case incremental_sync(next_cursor, headers) do
        {:ok, %{signals: more, cursor: final_cursor}} ->
          {:ok, %{signals: signals ++ more, cursor: final_cursor}}

        {:error, _} ->
          {:ok, %{signals: signals, cursor: next_cursor}}
      end
    else
      {:ok, %{signals: signals, cursor: resp["cursor"]}}
    end
  end
end
