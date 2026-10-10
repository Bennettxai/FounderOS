defmodule OptimalEngine.Connectors.Adapters.Salesforce do
  @moduledoc """
  Salesforce connector - accounts, opportunities, contacts, cases, notes.

  ## Required config keys
    * `:instance_url`, `:objects` (list of SObject API names)

  ## Credentials
    * `:client_id`, `:client_secret`, `:refresh_token`

  ## Cursor shape
  ISO-8601 timestamp (`LastModifiedDate >` filter). Initial sync: `nil` (last 30 days).
  """

  use OptimalEngine.Connectors.Adapters.Base,
    kind: :salesforce,
    display_name: "Salesforce",
    auth_scheme: :oauth2,
    required_keys: [:instance_url, :objects],
    credential_keys: [:client_id, :client_secret, :refresh_token]

  alias OptimalEngine.Connectors.HTTP

  @page_size 200
  @token_url "https://login.salesforce.com/services/oauth2/token"

  @impl true
  def sync(state, cursor) do
    instance_url = pick(state, :instance_url)
    objects = pick(state, :objects, ["Account"])

    with {:ok, access_token} <- get_access_token(state) do
      headers = [{"authorization", "Bearer #{access_token}"}]
      since = cursor || default_since()

      result =
        Enum.reduce_while(objects, {:ok, [], since}, fn sobject, {:ok, acc, last_ts} ->
          case fetch_sobject(instance_url, sobject, since, headers) do
            {:ok, records, new_ts} ->
              new_signals =
                Enum.flat_map(records, fn r ->
                  case transform(r) do
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
    ext_id = raw["Id"] || ""
    sobject = Map.get(raw, "attributes", %{}) |> Map.get("type", "record")

    {:ok,
     Transform.new_signal(%{
       id: Transform.signal_id(:salesforce, ext_id),
       title: raw["Name"] || raw["Subject"] || ext_id,
       content: raw["Description"] || raw["Body"] || "",
       path: Transform.source_uri(:salesforce, ext_id),
       genre: String.downcase(sobject),
       modified_at: Transform.parse_iso8601(raw["LastModifiedDate"])
     })}
  end

  # ---- private ----------------------------------------------------------------

  defp get_access_token(state) do
    client_id = pick(state, :client_id)
    client_secret = pick(state, :client_secret)
    refresh_token = pick(state, :refresh_token)

    body =
      URI.encode_query(%{
        grant_type: "refresh_token",
        client_id: client_id,
        client_secret: client_secret,
        refresh_token: refresh_token
      })

    case HTTP.request(@token_url,
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

  defp fetch_sobject(instance_url, sobject, since, headers) do
    fields = sobject_fields(sobject)

    soql =
      "SELECT #{fields} FROM #{sobject} WHERE LastModifiedDate > #{since} ORDER BY LastModifiedDate DESC LIMIT #{@page_size}"

    encoded = URI.encode_query(%{"q" => soql})
    url = "#{instance_url}/services/data/v59.0/query?#{encoded}"

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: %{"records" => records}}} ->
        new_ts =
          records
          |> Enum.map(& &1["LastModifiedDate"])
          |> Enum.reject(&is_nil/1)
          |> Enum.max(fn -> since end)

        {:ok, records, new_ts}

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

  defp sobject_fields("Account"), do: "Id,Name,Description,LastModifiedDate"
  defp sobject_fields("Opportunity"), do: "Id,Name,Description,LastModifiedDate"
  defp sobject_fields("Contact"), do: "Id,Name,Description,LastModifiedDate"
  defp sobject_fields("Case"), do: "Id,Subject,Description,Body,LastModifiedDate"
  defp sobject_fields("Note"), do: "Id,Title,Body,LastModifiedDate"
  defp sobject_fields(_), do: "Id,Name,LastModifiedDate"

  defp default_since do
    DateTime.utc_now()
    |> DateTime.add(-30 * 86_400, :second)
    |> DateTime.to_iso8601()
  end
end
