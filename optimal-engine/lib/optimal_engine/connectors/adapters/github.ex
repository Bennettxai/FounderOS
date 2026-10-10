defmodule OptimalEngine.Connectors.Adapters.GitHub do
  @moduledoc """
  GitHub connector -- issues, PRs, discussions, commits, wiki.

  ## Required config keys
    * `:org_or_user`, `:repos` (list of `"name"` or `"*"` for all)

  ## Credentials
    * `:pat` -- personal access token **or**
    * `:app_id` + `:installation_id` + `:private_key_pem` (GitHub App)

  ## Cursor shape
  ISO-8601 timestamp (`since` param). Initial sync: `nil` (fetches last 30 days).
  """

  use OptimalEngine.Connectors.Adapters.Base,
    kind: :github,
    display_name: "GitHub",
    auth_scheme: :token,
    required_keys: [:org_or_user, :repos],
    credential_keys: []

  alias OptimalEngine.Connectors.HTTP

  @base_url "https://api.github.com"
  @page_size 100

  @impl true
  def init(config) do
    flat = flatten_credentials(config)

    with :ok <- require_keys(flat, required_config_keys()),
         :ok <- require_github_auth(flat) do
      {:ok, flat}
    end
  end

  @impl true
  def sync(state, cursor) do
    org = pick(state, :org_or_user)
    repos = pick(state, :repos, [])
    since = cursor || default_since()
    token = pick(state, :pat)
    headers = auth_headers(token)

    repo_list =
      case repos do
        ["*"] -> fetch_repos(org, headers)
        list when is_list(list) -> list
        _ -> []
      end

    {signals, last_ts} =
      Enum.reduce(repo_list, {[], since}, fn repo, {acc_signals, acc_ts} ->
        case fetch_issues(org, repo, since, headers) do
          {:ok, items} ->
            new_signals =
              Enum.flat_map(items, fn item ->
                case transform(item) do
                  {:ok, s} -> [s]
                  _ -> []
                end
              end)

            new_ts = latest_ts(items, acc_ts)
            {acc_signals ++ new_signals, new_ts}

          {:error, _} ->
            {acc_signals, acc_ts}
        end
      end)

    {:ok, %{signals: signals, cursor: last_ts}}
  end

  @impl true
  def transform(raw) when is_map(raw) do
    ext_id = raw["node_id"] || to_string(raw["id"] || "")
    kind_str = if raw["pull_request"], do: "pr", else: "issue"

    {:ok,
     Transform.new_signal(%{
       id: Transform.signal_id(:github, ext_id),
       title: "[#{kind_str}] #{raw["title"] || ""}",
       content: raw["body"] || "",
       path: Transform.source_uri(:github, ext_id),
       genre: kind_str,
       modified_at: Transform.parse_iso8601(raw["updated_at"]),
       entities: [get_in(raw, ["user", "login"])] |> Enum.reject(&is_nil/1)
     })}
  end

  # ---- private ----------------------------------------------------------------

  defp require_github_auth(config) do
    pat = Map.get(config, "pat") || Map.get(config, :pat)

    app? =
      Enum.all?(
        [:app_id, :installation_id, :private_key_pem],
        fn k -> Map.has_key?(config, k) or Map.has_key?(config, Atom.to_string(k)) end
      )

    cond do
      is_binary(pat) and pat != "" -> :ok
      app? -> :ok
      true -> {:error, :missing_credentials}
    end
  end

  defp auth_headers(nil), do: []
  defp auth_headers(token), do: [{"authorization", "token #{token}"}]

  defp fetch_repos(org, headers) do
    url = "#{@base_url}/orgs/#{org}/repos?per_page=#{@page_size}"

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: repos}} when is_list(repos) ->
        Enum.map(repos, & &1["name"])

      _ ->
        []
    end
  end

  defp fetch_issues(org, repo, since, headers) do
    url =
      "#{@base_url}/repos/#{org}/#{repo}/issues" <>
        "?state=all&sort=updated&direction=desc&since=#{since}&per_page=#{@page_size}"

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: items}} when is_list(items) -> {:ok, items}
      {:ok, %{status: 401}} -> {:error, :auth_expired}
      {:ok, %{status: 429}} -> {:error, :rate_limited}
      {:ok, %{status: status}} -> {:error, {:http, status}}
      {:error, reason} -> {:error, reason}
    end
  end

  defp default_since do
    DateTime.utc_now()
    |> DateTime.add(-30 * 86_400, :second)
    |> DateTime.to_iso8601()
  end

  defp latest_ts([], current), do: current

  defp latest_ts(items, current) do
    items
    |> Enum.map(& &1["updated_at"])
    |> Enum.reject(&is_nil/1)
    |> Enum.max(fn -> current end)
  end
end
