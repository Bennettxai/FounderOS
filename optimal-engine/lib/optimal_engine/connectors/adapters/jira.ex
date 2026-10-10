defmodule OptimalEngine.Connectors.Adapters.Jira do
  @moduledoc """
  Jira connector - issues, comments, sprints, releases.

  ## Required config keys
    * `:site_url` (e.g. `https://sample.atlassian.net`), `:projects` (list of keys)

  ## Credentials
    * `:email`, `:api_token`

  ## Cursor shape
  ISO-8601 timestamp (`updated > "cursor"` JQL filter). Initial sync: `nil`.
  """

  use OptimalEngine.Connectors.Adapters.Base,
    kind: :jira,
    display_name: "Jira",
    auth_scheme: :basic,
    required_keys: [:site_url, :projects],
    credential_keys: [:email, :api_token]

  alias OptimalEngine.Connectors.HTTP

  @page_size 100

  @impl true
  def sync(state, cursor) do
    site_url = pick(state, :site_url)
    projects = pick(state, :projects, [])
    email = pick(state, :email)
    api_token = pick(state, :api_token)

    credentials = Base.encode64("#{email}:#{api_token}")
    headers = [{"authorization", "Basic #{credentials}"}]

    since = cursor || default_since()

    project_jql = Enum.map_join(projects, ", ", &"\"#{&1}\"")
    jql = "project in (#{project_jql}) AND updated > \"#{since}\" ORDER BY updated DESC"

    case fetch_issues(site_url, jql, 0, headers) do
      {:ok, issues, new_ts} ->
        signals =
          Enum.flat_map(issues, fn issue ->
            case transform(issue) do
              {:ok, s} -> [s]
              _ -> []
            end
          end)

        {:ok, %{signals: signals, cursor: new_ts || since}}

      {:error, :auth_expired} ->
        {:error, :auth_expired}

      {:error, :rate_limited} ->
        {:error, :rate_limited}

      {:error, reason} ->
        {:error, reason}
    end
  end

  @impl true
  def transform(raw) when is_map(raw) do
    ext_id = raw["key"] || raw["id"] || ""
    fields = raw["fields"] || %{}

    {:ok,
     Transform.new_signal(%{
       id: Transform.signal_id(:jira, ext_id),
       title: "[#{ext_id}] #{fields["summary"] || ""}",
       content: Transform.strip_html(fields["description"] || ""),
       path: Transform.source_uri(:jira, ext_id),
       genre: "ticket",
       modified_at: Transform.parse_iso8601(fields["updated"]),
       entities:
         Enum.filter([fields["assignee"]["displayName"], fields["reporter"]["displayName"]], & &1)
     })}
  rescue
    _ ->
      {:error, :bad_payload}
  end

  # ---- private ----------------------------------------------------------------

  defp fetch_issues(site_url, jql, start_at, headers) do
    encoded =
      URI.encode_query(%{
        "jql" => jql,
        "startAt" => start_at,
        "maxResults" => @page_size,
        "fields" => "summary,description,updated,assignee,reporter"
      })

    url = "#{site_url}/rest/api/3/search?#{encoded}"

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: %{"issues" => issues}}} ->
        new_ts =
          issues
          |> Enum.map(fn i -> get_in(i, ["fields", "updated"]) end)
          |> Enum.reject(&is_nil/1)
          |> Enum.max(fn -> nil end)

        {:ok, issues, new_ts}

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
    |> Calendar.strftime("%Y-%m-%d %H:%M")
  end
end
