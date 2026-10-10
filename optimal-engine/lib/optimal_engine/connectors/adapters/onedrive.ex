defmodule OptimalEngine.Connectors.Adapters.OneDrive do
  @moduledoc """
  OneDrive / SharePoint connector -- files, documents, sites.

  ## Required config keys
    * `:tenant_id_ms`, `:drive_id`

  ## Credentials
    * `:client_id`, `:client_secret` (Azure AD app)

  ## Cursor shape
  Microsoft Graph delta link (URL string). Initial sync: `nil`.
  """

  use OptimalEngine.Connectors.Adapters.Base,
    kind: :onedrive,
    display_name: "OneDrive",
    auth_scheme: :oauth2,
    required_keys: [:tenant_id_ms, :drive_id],
    credential_keys: [:client_id, :client_secret]

  alias OptimalEngine.Connectors.HTTP

  @graph_base "https://graph.microsoft.com/v1.0"
  @page_size 100

  @impl true
  def sync(state, cursor) do
    with {:ok, access_token} <- acquire_token(state) do
      headers = [{"authorization", "Bearer #{access_token}"}]

      case cursor do
        nil -> full_sync(state, headers)
        delta_link -> delta_sync(delta_link, headers)
      end
    end
  end

  @impl true
  def transform(raw) when is_map(raw) do
    ext_id = raw["id"] || ""

    {:ok,
     Transform.new_signal(%{
       id: Transform.signal_id(:onedrive, ext_id),
       title: raw["name"] || "Untitled",
       content: raw["content"] || "",
       path: Transform.source_uri(:onedrive, ext_id),
       genre: "file",
       modified_at: Transform.parse_iso8601(raw["lastModifiedDateTime"])
     })}
  end

  # ---- private ----------------------------------------------------------------

  defp acquire_token(state) do
    tenant_id = pick(state, :tenant_id_ms)
    url = "https://login.microsoftonline.com/#{tenant_id}/oauth2/v2.0/token"

    body =
      "grant_type=client_credentials" <>
        "&client_id=#{pick(state, :client_id)}" <>
        "&client_secret=#{pick(state, :client_secret)}" <>
        "&scope=https%3A%2F%2Fgraph.microsoft.com%2F.default"

    case HTTP.request(url,
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
    drive_id = pick(state, :drive_id)
    url = "#{@graph_base}/drives/#{drive_id}/root/delta?$top=#{@page_size}"
    fetch_delta_page(url, [], headers)
  end

  defp delta_sync(delta_link, headers) do
    fetch_delta_page(delta_link, [], headers)
  end

  defp fetch_delta_page(url, acc, headers) do
    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: body}} ->
        items =
          (body["value"] || [])
          |> Enum.reject(fn item -> Map.has_key?(item, "deleted") end)

        signals =
          Enum.flat_map(items, fn item ->
            case transform(item) do
              {:ok, s} -> [s]
              _ -> []
            end
          end)

        all_signals = acc ++ signals

        cond do
          body["@odata.nextLink"] ->
            fetch_delta_page(body["@odata.nextLink"], all_signals, headers)

          body["@odata.deltaLink"] ->
            {:ok, %{signals: all_signals, cursor: body["@odata.deltaLink"]}}

          true ->
            {:ok, %{signals: all_signals, cursor: url}}
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
end
