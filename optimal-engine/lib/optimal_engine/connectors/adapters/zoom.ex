defmodule OptimalEngine.Connectors.Adapters.Zoom do
  @moduledoc """
  Zoom connector - meeting recordings + transcripts.

  ## Required config keys
    * `:account_id`

  ## Credentials
    * `:client_id`, `:client_secret` (server-to-server OAuth)

  ## Cursor shape
  ISO-8601 date string used as `from` query parameter. Initial sync: `nil` (last 30 days).
  """

  use OptimalEngine.Connectors.Adapters.Base,
    kind: :zoom,
    display_name: "Zoom",
    auth_scheme: :oauth2,
    required_keys: [:account_id],
    credential_keys: [:client_id, :client_secret]

  alias OptimalEngine.Connectors.HTTP

  @base_url "https://api.zoom.us/v2"
  @page_size 30

  @impl true
  def sync(state, cursor) do
    account_id = pick(state, :account_id)

    with {:ok, access_token} <- get_access_token(state, account_id) do
      headers = [{"authorization", "Bearer #{access_token}"}]
      from_date = cursor || default_from()

      case fetch_recordings(from_date, headers) do
        {:ok, meetings, next_cursor} ->
          signals =
            Enum.flat_map(meetings, fn meeting ->
              case transform(meeting) do
                {:ok, s} -> [s]
                _ -> []
              end
            end)

          {:ok, %{signals: signals, cursor: next_cursor || from_date}}

        {:error, :auth_expired} ->
          {:error, :auth_expired}

        {:error, :rate_limited} ->
          {:error, :rate_limited}

        {:error, reason} ->
          {:error, reason}
      end
    end
  end

  @impl true
  def transform(raw) when is_map(raw) do
    ext_id = raw["uuid"] || raw["id"] || ""

    {:ok,
     Transform.new_signal(%{
       id: Transform.signal_id(:zoom, to_string(ext_id)),
       title: raw["topic"] || "Zoom meeting",
       content: raw["transcript"] || "",
       path: Transform.source_uri(:zoom, to_string(ext_id)),
       genre: "transcript",
       mode: :linguistic,
       modified_at: Transform.parse_iso8601(raw["start_time"])
     })}
  end

  # ---- private ----------------------------------------------------------------

  defp get_access_token(state, account_id) do
    client_id = pick(state, :client_id)
    client_secret = pick(state, :client_secret)

    credentials = Base.encode64("#{client_id}:#{client_secret}")

    token_url =
      "https://zoom.us/oauth/token?grant_type=account_credentials&account_id=#{account_id}"

    case HTTP.request(token_url,
           method: :post,
           body: "",
           headers: [
             {"authorization", "Basic #{credentials}"},
             {"content-type", "application/x-www-form-urlencoded"}
           ]
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

  defp fetch_recordings(from_date, headers) do
    # Zoom's /users/me/recordings uses `from` + `to` date range (max 1 month)
    to_date =
      from_date
      |> Date.from_iso8601!()
      |> Date.add(30)
      |> Date.to_iso8601()

    url =
      "#{@base_url}/users/me/recordings?from=#{from_date}&to=#{to_date}&page_size=#{@page_size}"

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: %{"meetings" => meetings} = body}} ->
        # next_page_token for further pagination within the window
        next_token = body["next_page_token"]

        next_cursor =
          if next_token && next_token != "" do
            # Encode as from_date so incremental re-entry keeps the same window
            from_date
          else
            # Advance window to to_date for next sync
            to_date
          end

        {:ok, meetings, next_cursor}

      {:ok, %{status: 401}} ->
        {:error, :auth_expired}

      {:ok, %{status: 429}} ->
        {:error, :rate_limited}

      {:ok, %{status: status}} ->
        {:error, {:http, status}}

      {:error, reason} ->
        {:error, reason}
    end
  rescue
    _ -> {:error, :bad_date}
  end

  defp default_from do
    Date.utc_today()
    |> Date.add(-30)
    |> Date.to_iso8601()
  end
end
