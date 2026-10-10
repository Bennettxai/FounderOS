defmodule OptimalEngine.Connectors.Adapters.Linear do
  @moduledoc """
  Linear connector - issues, comments, cycles, projects.

  ## Required config keys
    * `:team_ids` (list of Linear team UUIDs)

  ## Credentials
    * `:api_key` - `lin_api_...`

  ## Cursor shape
  Opaque pagination cursor from Linear's GraphQL `pageInfo.endCursor`. Initial sync: `nil`.
  """

  use OptimalEngine.Connectors.Adapters.Base,
    kind: :linear,
    display_name: "Linear",
    auth_scheme: :token,
    required_keys: [:team_ids],
    credential_keys: [:api_key]

  alias OptimalEngine.Connectors.HTTP

  @graphql_url "https://api.linear.app/graphql"
  @page_size 50

  @impl true
  def sync(state, cursor) do
    api_key = pick(state, :api_key)
    team_ids = pick(state, :team_ids, [])
    headers = [{"authorization", api_key}, {"content-type", "application/json"}]

    team_filter = Enum.map_join(team_ids, ", ", &"\"#{&1}\"")

    query = """
    query($after: String) {
      issues(
        filter: { team: { id: { in: [#{team_filter}] } } }
        first: #{@page_size}
        after: $after
        orderBy: updatedAt
      ) {
        nodes {
          id
          title
          description
          updatedAt
          assignee { name }
          creator { name }
          state { name }
        }
        pageInfo { hasNextPage endCursor }
      }
    }
    """

    body = %{"query" => query, "variables" => %{"after" => cursor}}

    case HTTP.post_json(@graphql_url, body, headers: headers) do
      {:ok,
       %{
         status: 200,
         body: %{"data" => %{"issues" => %{"nodes" => nodes, "pageInfo" => page_info}}}
       }} ->
        signals =
          Enum.flat_map(nodes, fn node ->
            case transform(node) do
              {:ok, s} -> [s]
              _ -> []
            end
          end)

        next_cursor =
          if page_info["hasNextPage"],
            do: page_info["endCursor"],
            else: nil

        {:ok, %{signals: signals, cursor: next_cursor}}

      {:ok, %{status: 401}} ->
        {:error, :auth_expired}

      {:ok, %{status: 429}} ->
        {:error, :rate_limited}

      {:ok,
       %{status: 200, body: %{"errors" => [%{"extensions" => %{"type" => "AUTHENTICATION"}} | _]}}} ->
        {:error, :auth_expired}

      {:ok, %{status: status}} ->
        {:error, {:http, status}}

      {:error, reason} ->
        {:error, reason}
    end
  end

  @impl true
  def transform(raw) when is_map(raw) do
    ext_id = raw["id"] || ""

    {:ok,
     Transform.new_signal(%{
       id: Transform.signal_id(:linear, ext_id),
       title: raw["title"] || "Untitled",
       content: raw["description"] || "",
       path: Transform.source_uri(:linear, ext_id),
       genre: "ticket",
       modified_at: Transform.parse_iso8601(raw["updatedAt"]),
       entities:
         Enum.filter([get_in(raw, ["assignee", "name"]), get_in(raw, ["creator", "name"])], & &1)
     })}
  end
end
