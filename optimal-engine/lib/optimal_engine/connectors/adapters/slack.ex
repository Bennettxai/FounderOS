defmodule OptimalEngine.Connectors.Adapters.Slack do
  @moduledoc """
  Slack connector -- channels, threads, DMs, files.

  ## Required config keys
    * `:workspace_id`, `:channels` (list; empty = all)

  ## Credentials
    * `:bot_token` -- starts with `xoxb-...`

  ## Cursor shape
  Opaque: `"<channel_id>:<oldest_ts>"`. Initial sync: `nil`.
  """

  use OptimalEngine.Connectors.Adapters.Base,
    kind: :slack,
    display_name: "Slack",
    auth_scheme: :oauth2,
    required_keys: [:workspace_id, :channels],
    credential_keys: [:bot_token]

  alias OptimalEngine.Connectors.HTTP

  @base_url "https://slack.com/api"
  @page_size 200

  @impl true
  def sync(state, cursor) do
    token = pick(state, :bot_token)
    headers = [{"authorization", "Bearer #{token}"}]

    channels_cfg = pick(state, :channels, [])

    channel_ids =
      case channels_cfg do
        [] -> fetch_all_channels(headers)
        ids when is_list(ids) -> ids
        _ -> []
      end

    {channel_id, oldest} = parse_cursor(cursor)

    # If cursor has a specific channel, resume only that channel
    channels_to_sync =
      if channel_id && channel_id != "" do
        [channel_id]
      else
        channel_ids
      end

    {signals, next_cursor} =
      Enum.reduce(channels_to_sync, {[], nil}, fn ch_id, {acc, _last_cursor} ->
        ch_oldest = if ch_id == channel_id, do: oldest, else: nil

        case fetch_messages(ch_id, ch_oldest, headers) do
          {:ok, messages, next_ts} ->
            new_signals =
              Enum.flat_map(messages, fn msg ->
                case transform(msg) do
                  {:ok, s} -> [s]
                  _ -> []
                end
              end)

            new_cursor = if next_ts, do: "#{ch_id}:#{next_ts}", else: "#{ch_id}:"
            {acc ++ new_signals, new_cursor}

          {:error, _} ->
            {acc, "#{ch_id}:"}
        end
      end)

    {:ok, %{signals: signals, cursor: next_cursor}}
  end

  @impl true
  def transform(raw) when is_map(raw) do
    ext_id = raw["client_msg_id"] || raw["ts"] || ""
    text = raw["text"] || ""
    user = raw["user"] || "slack-user"

    {:ok,
     Transform.new_signal(%{
       id: Transform.signal_id(:slack, ext_id),
       title: String.slice(text, 0, 80),
       content: text,
       path: Transform.source_uri(:slack, ext_id),
       genre: "message",
       entities: [user],
       modified_at: parse_ts(raw["ts"])
     })}
  end

  # ---- private ----------------------------------------------------------------

  defp fetch_all_channels(headers) do
    url = "#{@base_url}/conversations.list?types=public_channel,private_channel&limit=#{@page_size}"

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: %{"ok" => true, "channels" => chs}}} ->
        Enum.map(chs, & &1["id"])

      _ ->
        []
    end
  end

  defp fetch_messages(channel_id, oldest, headers) do
    url =
      "#{@base_url}/conversations.history?channel=#{channel_id}&limit=#{@page_size}" <>
        if(oldest, do: "&oldest=#{oldest}", else: "")

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: %{"ok" => true, "messages" => messages} = body}} ->
        next_ts =
          if body["has_more"] do
            messages |> Enum.map(& &1["ts"]) |> Enum.min(fn -> nil end)
          else
            nil
          end

        {:ok, messages, next_ts}

      {:ok, %{status: 200, body: %{"ok" => false, "error" => "ratelimited"}}} ->
        {:error, :rate_limited}

      {:ok, %{status: 200, body: %{"ok" => false, "error" => "token_expired"}}} ->
        {:error, :auth_expired}

      {:ok, %{status: 429}} ->
        {:error, :rate_limited}

      {:ok, %{status: 401}} ->
        {:error, :auth_expired}

      {:error, reason} ->
        {:error, reason}

      _ ->
        {:ok, [], nil}
    end
  end

  defp parse_cursor(nil), do: {"", nil}

  defp parse_cursor(cursor) when is_binary(cursor) do
    case String.split(cursor, ":", parts: 2) do
      [ch_id, oldest] -> {ch_id, if(oldest == "", do: nil, else: oldest)}
      _ -> {"", nil}
    end
  end

  defp parse_ts(nil), do: DateTime.utc_now()

  defp parse_ts(ts) when is_binary(ts) do
    case Float.parse(ts) do
      {f, _} -> f |> round() |> DateTime.from_unix!()
      _ -> DateTime.utc_now()
    end
  end
end
