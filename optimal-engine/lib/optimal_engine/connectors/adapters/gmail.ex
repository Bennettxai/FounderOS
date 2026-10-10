defmodule OptimalEngine.Connectors.Adapters.Gmail do
  @moduledoc """
  Gmail connector -- messages, threads, labels, attachments.

  ## Required config keys
    * `:user_email`, `:labels` (list of label names to sync; default all)

  ## Credentials
    * `:oauth_refresh_token` -- long-lived OAuth2 refresh token
    * `:client_id`, `:client_secret` -- Google app credentials

  ## Cursor shape
  Opaque: a Gmail `historyId`. Initial sync: `nil` -> full fetch bounded
  by `:initial_lookback_days` (default 30).
  """

  use OptimalEngine.Connectors.Adapters.Base,
    kind: :gmail,
    display_name: "Gmail",
    auth_scheme: :oauth2,
    required_keys: [:user_email],
    credential_keys: [:oauth_refresh_token, :client_id, :client_secret]

  alias OptimalEngine.Connectors.HTTP

  @token_url "https://oauth2.googleapis.com/token"
  @base_url "https://gmail.googleapis.com/gmail/v1/users/me"
  @page_size 100

  @impl true
  def sync(state, cursor) do
    with {:ok, access_token} <- refresh_access_token(state) do
      headers = [{"authorization", "Bearer #{access_token}"}]

      case cursor do
        nil -> full_sync(state, headers)
        history_id -> incremental_sync(history_id, headers)
      end
    end
  end

  @impl true
  def transform(raw) when is_map(raw) do
    ext_id = raw["id"] || ""
    subject = get_header(raw, "Subject") || "(no subject)"
    from = get_header(raw, "From") || ""

    {:ok,
     Transform.new_signal(%{
       id: Transform.signal_id(:gmail, ext_id),
       title: String.slice(subject, 0, 120),
       content: Transform.strip_html(raw["snippet"] || raw["body"] || ""),
       path: Transform.source_uri(:gmail, ext_id),
       genre: "email",
       mode: :linguistic,
       entities: extract_addresses(from),
       modified_at: Transform.parse_iso8601(raw["internalDate"])
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

  defp full_sync(state, headers) do
    lookback = pick(state, :initial_lookback_days, 30)

    after_ts =
      DateTime.utc_now()
      |> DateTime.add(-lookback * 86_400, :second)
      |> DateTime.to_unix()

    query = "after:#{after_ts}"
    labels = pick(state, :labels, [])

    query =
      if labels != [] do
        label_q = labels |> Enum.map(&"label:#{&1}") |> Enum.join(" OR ")
        "#{query} (#{label_q})"
      else
        query
      end

    fetch_messages(query, nil, headers)
  end

  defp incremental_sync(history_id, headers) do
    url = "#{@base_url}/history?startHistoryId=#{history_id}&historyTypes=messageAdded"

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: body}} ->
        messages =
          (body["history"] || [])
          |> Enum.flat_map(fn h -> h["messagesAdded"] || [] end)
          |> Enum.map(fn %{"message" => m} -> m end)

        next_cursor = body["historyId"] || history_id
        enrich_and_return(messages, next_cursor, headers)

      {:ok, %{status: 404}} ->
        # historyId too old -- fall back to full sync
        full_sync(%{}, headers)

      {:ok, %{status: 401}} ->
        {:error, :auth_expired}

      {:ok, %{status: 429}} ->
        {:error, :rate_limited}

      {:error, reason} ->
        {:error, reason}
    end
  end

  defp fetch_messages(query, page_token, headers) do
    url =
      "#{@base_url}/messages?maxResults=#{@page_size}&q=#{URI.encode_www_form(query)}" <>
        if(page_token, do: "&pageToken=#{page_token}", else: "")

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: body}} ->
        messages = body["messages"] || []
        next_page = body["nextPageToken"]

        enrich_and_return(messages, next_page, headers)

      {:ok, %{status: 401}} ->
        {:error, :auth_expired}

      {:ok, %{status: 429}} ->
        {:error, :rate_limited}

      {:error, reason} ->
        {:error, reason}
    end
  end

  defp enrich_and_return(message_stubs, next_cursor, headers) do
    signals =
      message_stubs
      |> Enum.flat_map(fn %{"id" => id} ->
        case fetch_message(id, headers) do
          {:ok, msg} ->
            case transform(msg) do
              {:ok, s} -> [s]
              _ -> []
            end

          _ ->
            []
        end
      end)

    {:ok, %{signals: signals, cursor: next_cursor}}
  end

  defp fetch_message(id, headers) do
    url = "#{@base_url}/messages/#{id}?format=metadata&metadataHeaders=Subject,From,Date"

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: msg}} -> {:ok, msg}
      {:ok, %{status: status}} -> {:error, {:http, status}}
      {:error, reason} -> {:error, reason}
    end
  end

  defp get_header(%{"payload" => %{"headers" => headers}}, name) when is_list(headers) do
    Enum.find_value(headers, fn
      %{"name" => ^name, "value" => v} -> v
      _ -> nil
    end)
  end

  defp get_header(_, _), do: nil

  defp extract_addresses(header) when is_binary(header) do
    Regex.scan(~r/[\w.+-]+@[\w.-]+\.[A-Za-z]{2,}/, header)
    |> Enum.flat_map(& &1)
  end

  defp extract_addresses(_), do: []
end
