defmodule OptimalEngine.Connectors.Adapters.Teams do
  @moduledoc """
  Microsoft Teams connector - channel messages, chats, meetings.

  ## Required config keys
    * `:tenant_id_ms` (Azure AD tenant), `:team_ids` (list of team IDs)

  ## Credentials
    * `:client_id`, `:client_secret` (Azure AD app; uses client_credentials grant)

  ## Cursor shape
  ISO-8601 timestamp (`lastModifiedDateTime > cursor` filter). Initial sync: `nil`.
  """

  use OptimalEngine.Connectors.Adapters.Base,
    kind: :teams,
    display_name: "Microsoft Teams",
    auth_scheme: :oauth2,
    required_keys: [:tenant_id_ms, :team_ids],
    credential_keys: [:client_id, :client_secret]

  alias OptimalEngine.Connectors.HTTP

  @graph_url "https://graph.microsoft.com/v1.0"
  @page_size 50

  @impl true
  def sync(state, cursor) do
    tenant_id = pick(state, :tenant_id_ms)
    team_ids = pick(state, :team_ids, [])

    with {:ok, access_token} <- get_access_token(state, tenant_id) do
      headers = [{"authorization", "Bearer #{access_token}"}]
      since = cursor || default_since()

      result =
        Enum.reduce_while(team_ids, {:ok, [], since}, fn team_id, {:ok, acc, last_ts} ->
          case fetch_channel_messages(team_id, since, headers) do
            {:ok, messages, new_ts} ->
              new_signals =
                Enum.flat_map(messages, fn msg ->
                  case transform(msg) do
                    {:ok, s} -> [s]
                    _ -> []
                  end
                end)

              {:cont, {:ok, acc ++ new_signals, new_ts || last_ts}}

            {:error, :auth_expired} ->
              {:halt, {:error, :auth_expired}}

            {:error, :rate_limited} ->
              {:halt, {:error, :rate_limited}}

            {:error, _} ->
              {:cont, {:ok, acc, last_ts}}
          end
        end)

      case result do
        {:ok, signals, next_cursor} -> {:ok, %{signals: signals, cursor: next_cursor}}
        {:error, _} = err -> err
      end
    end
  end

  @impl true
  def transform(raw) when is_map(raw) do
    ext_id = raw["id"] || ""
    body_text = get_in(raw, ["body", "content"]) || ""

    {:ok,
     Transform.new_signal(%{
       id: Transform.signal_id(:teams, ext_id),
       title: String.slice(Transform.strip_html(body_text), 0, 80),
       content: Transform.strip_html(body_text),
       path: Transform.source_uri(:teams, ext_id),
       genre: "message",
       modified_at: Transform.parse_iso8601(raw["lastModifiedDateTime"]),
       entities: [get_in(raw, ["from", "user", "displayName"])] |> Enum.reject(&is_nil/1)
     })}
  end

  # ---- private ----------------------------------------------------------------

  defp get_access_token(state, tenant_id) do
    client_id = pick(state, :client_id)
    client_secret = pick(state, :client_secret)
    token_url = "https://login.microsoftonline.com/#{tenant_id}/oauth2/v2.0/token"

    body =
      URI.encode_query(%{
        grant_type: "client_credentials",
        client_id: client_id,
        client_secret: client_secret,
        scope: "https://graph.microsoft.com/.default"
      })

    case HTTP.request(token_url,
           method: :post,
           body: body,
           headers: [{"content-type", "application/x-www-form-urlencoded"}]
         ) do
      {:ok, %{status: 200, body: %{"access_token" => token}}} ->
        {:ok, token}

      {:ok, %{status: 401}} ->
        {:error, :auth_expired}

      {:ok, %{status: status}} ->
        {:error, {:http, status}}

      {:error, reason} ->
        {:error, reason}
    end
  end

  defp fetch_channel_messages(team_id, since, headers) do
    # Fetch all channels for the team, then messages per channel
    with {:ok, channel_ids} <- list_channels(team_id, headers) do
      {all_messages, new_ts} =
        Enum.reduce(channel_ids, {[], nil}, fn channel_id, {msgs, last_ts} ->
          url =
            "#{@graph_url}/teams/#{team_id}/channels/#{channel_id}/messages" <>
              "?$top=#{@page_size}&$filter=lastModifiedDateTime gt #{since}"

          case HTTP.get_json(url, headers: headers) do
            {:ok, %{status: 200, body: %{"value" => messages}}} ->
              message_ts =
                messages
                |> Enum.map(& &1["lastModifiedDateTime"])
                |> Enum.reject(&is_nil/1)
                |> Enum.max(fn -> nil end)

              new_last =
                if message_ts && (is_nil(last_ts) || message_ts > last_ts),
                  do: message_ts,
                  else: last_ts

              {msgs ++ messages, new_last}

            _ ->
              {msgs, last_ts}
          end
        end)

      {:ok, all_messages, new_ts}
    end
  end

  defp list_channels(team_id, headers) do
    url = "#{@graph_url}/teams/#{team_id}/channels"

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: %{"value" => channels}}} ->
        {:ok, Enum.map(channels, & &1["id"])}

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

  defp default_since do
    DateTime.utc_now()
    |> DateTime.add(-30 * 86_400, :second)
    |> DateTime.to_iso8601()
  end
end
