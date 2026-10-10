defmodule OptimalEngine.Connectors.Adapters.Drive do
  @moduledoc """
  Google Drive connector -- docs, sheets, slides, PDFs, arbitrary files.

  ## Required config keys
    * `:scope` -- `:my_drive | :shared_drive | :starred`
    * `:drive_id` -- required when `scope: :shared_drive`

  ## Credentials
    * `:oauth_refresh_token`, `:client_id`, `:client_secret`

  ## Cursor shape
  Google Drive `changes.startPageToken`. Initial sync: `nil`.
  """

  use OptimalEngine.Connectors.Adapters.Base,
    kind: :drive,
    display_name: "Google Drive",
    auth_scheme: :oauth2,
    required_keys: [:scope],
    credential_keys: [:oauth_refresh_token, :client_id, :client_secret]

  alias OptimalEngine.Connectors.HTTP

  @token_url "https://oauth2.googleapis.com/token"
  @base_url "https://www.googleapis.com/drive/v3"
  @page_size 100

  @impl true
  def sync(state, cursor) do
    with {:ok, access_token} <- refresh_access_token(state) do
      headers = [{"authorization", "Bearer #{access_token}"}]

      case cursor do
        nil -> bootstrap_sync(state, headers)
        page_token -> incremental_sync(page_token, state, headers)
      end
    end
  end

  @impl true
  def transform(raw) when is_map(raw) do
    ext_id = raw["id"] || ""
    name = raw["name"] || "Untitled"
    mime = raw["mimeType"] || ""

    {:ok,
     Transform.new_signal(%{
       id: Transform.signal_id(:drive, ext_id),
       title: name,
       content: raw["exportedText"] || raw["content"] || "",
       path: Transform.source_uri(:drive, ext_id),
       genre: drive_genre(mime),
       mode: drive_mode(mime),
       format: drive_format(mime),
       modified_at: Transform.parse_iso8601(raw["modifiedTime"])
     })}
  end

  # ---- private ----------------------------------------------------------------

  defp refresh_access_token(state) do
    body = %{
      "grant_type" => "refresh_token",
      "refresh_token" => pick(state, :oauth_refresh_token),
      "client_id" => pick(state, :client_id),
      "client_secret" => pick(state, :client_secret)
    }

    case HTTP.post_json(@token_url, body) do
      {:ok, %{status: 200, body: %{"access_token" => token}}} -> {:ok, token}
      {:ok, %{status: 401}} -> {:error, :auth_expired}
      {:ok, %{body: body}} -> {:error, {:token_error, body}}
      {:error, reason} -> {:error, reason}
    end
  end

  # Initial sync: get a start page token, then list files using it as the
  # baseline so subsequent runs use the changes API.
  defp bootstrap_sync(state, headers) do
    case get_start_page_token(headers) do
      {:ok, start_token} ->
        case list_files(state, headers) do
          {:ok, files} ->
            signals =
              Enum.flat_map(files, fn f ->
                case transform(f) do
                  {:ok, s} -> [s]
                  _ -> []
                end
              end)

            {:ok, %{signals: signals, cursor: start_token}}

          {:error, reason} ->
            {:error, reason}
        end

      {:error, reason} ->
        {:error, reason}
    end
  end

  defp incremental_sync(page_token, _state, headers) do
    url =
      "#{@base_url}/changes" <>
        "?pageToken=#{page_token}&fields=nextPageToken,newStartPageToken,changes(file)&pageSize=#{@page_size}"

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: body}} ->
        files =
          (body["changes"] || [])
          |> Enum.map(& &1["file"])
          |> Enum.reject(&is_nil/1)

        signals =
          Enum.flat_map(files, fn f ->
            case transform(f) do
              {:ok, s} -> [s]
              _ -> []
            end
          end)

        next_cursor = body["newStartPageToken"] || body["nextPageToken"] || page_token
        {:ok, %{signals: signals, cursor: next_cursor}}

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

  defp get_start_page_token(headers) do
    url = "#{@base_url}/changes/startPageToken"

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: %{"startPageToken" => token}}} -> {:ok, token}
      {:ok, %{status: status}} -> {:error, {:http, status}}
      {:error, reason} -> {:error, reason}
    end
  end

  defp list_files(_state, headers) do
    url =
      "#{@base_url}/files" <>
        "?fields=files(id,name,mimeType,modifiedTime)&pageSize=#{@page_size}"

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: %{"files" => files}}} -> {:ok, files}
      {:ok, %{status: 401}} -> {:error, :auth_expired}
      {:ok, %{status: status}} -> {:error, {:http, status}}
      {:error, reason} -> {:error, reason}
    end
  end

  defp drive_genre("application/vnd.google-apps.document"), do: "document"
  defp drive_genre("application/vnd.google-apps.spreadsheet"), do: "table"
  defp drive_genre("application/vnd.google-apps.presentation"), do: "slides"
  defp drive_genre("application/pdf"), do: "document"
  defp drive_genre(_), do: "file"

  defp drive_mode("application/vnd.google-apps.spreadsheet"), do: :data
  defp drive_mode("image/" <> _), do: :visual
  defp drive_mode(_), do: :linguistic

  defp drive_format("application/vnd.google-apps.document"), do: :markdown
  defp drive_format("application/vnd.google-apps.spreadsheet"), do: :json
  defp drive_format(_), do: :unknown
end
