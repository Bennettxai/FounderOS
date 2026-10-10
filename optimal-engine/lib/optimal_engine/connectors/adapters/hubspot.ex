defmodule OptimalEngine.Connectors.Adapters.HubSpot do
  @moduledoc """
  HubSpot connector - CRM contacts, companies, deals, engagements, notes.

  ## Required config keys
    * `:portal_id`, `:objects` (list of HubSpot object API names)

  ## Credentials
    * `:access_token` (private app) **or**
    * `:client_id` + `:client_secret` + `:refresh_token` (OAuth app)

  ## Cursor shape
  Opaque string: `"<object_type>:<after_cursor>"`. Initial sync: `nil`.
  """

  use OptimalEngine.Connectors.Adapters.Base,
    kind: :hubspot,
    display_name: "HubSpot",
    auth_scheme: :oauth2,
    required_keys: [:portal_id, :objects],
    credential_keys: []

  alias OptimalEngine.Connectors.HTTP

  @base_url "https://api.hubapi.com"
  @page_size 100

  @impl true
  def init(config) do
    flat = flatten_credentials(config)

    with :ok <- require_keys(flat, required_config_keys()),
         :ok <- require_hubspot_auth(flat) do
      {:ok, flat}
    end
  end

  @impl true
  def sync(state, cursor) do
    token = pick(state, :access_token)
    objects = pick(state, :objects, ["contacts"])
    headers = [{"authorization", "Bearer #{token}"}]

    {resume_object, after_cursor} = parse_cursor(cursor)

    objects_to_sync =
      if resume_object && resume_object != "" do
        # Resume from the specific object in the cursor
        idx = Enum.find_index(objects, &(&1 == resume_object)) || 0
        Enum.drop(objects, idx)
      else
        objects
      end

    result =
      Enum.reduce_while(objects_to_sync, {:ok, [], nil}, fn object_type, {:ok, acc, _} ->
        obj_after = if object_type == resume_object, do: after_cursor, else: nil

        case fetch_objects(object_type, obj_after, headers) do
          {:ok, items, next_after} ->
            new_signals =
              Enum.flat_map(items, fn item ->
                case transform(item) do
                  {:ok, s} -> [s]
                  _ -> []
                end
              end)

            new_cursor =
              if next_after,
                do: "#{object_type}:#{next_after}",
                else: nil

            {:cont, {:ok, acc ++ new_signals, new_cursor}}

          {:error, :auth_expired} ->
            {:halt, {:error, :auth_expired}}

          {:error, :rate_limited} ->
            {:halt, {:error, :rate_limited}}

          {:error, _} ->
            {:cont, {:ok, acc, nil}}
        end
      end)

    case result do
      {:ok, signals, next_cursor} -> {:ok, %{signals: signals, cursor: next_cursor}}
      {:error, _} = err -> err
    end
  end

  @impl true
  def transform(raw) when is_map(raw) do
    ext_id = to_string(raw["id"] || "")
    props = raw["properties"] || %{}

    {:ok,
     Transform.new_signal(%{
       id: Transform.signal_id(:hubspot, ext_id),
       title:
         props["dealname"] || props["firstname"] || props["company"] || props["subject"] ||
           ext_id,
       content: props["description"] || props["notes_last_contacted"] || "",
       path: Transform.source_uri(:hubspot, ext_id),
       genre: "crm",
       modified_at: Transform.parse_iso8601(props["hs_lastmodifieddate"] || raw["updatedAt"])
     })}
  end

  # ---- private ----------------------------------------------------------------

  defp fetch_objects(object_type, after_cursor, headers) do
    url =
      "#{@base_url}/crm/v3/objects/#{object_type}?limit=#{@page_size}" <>
        "&properties=dealname,firstname,lastname,company,subject,description,notes_last_contacted,hs_lastmodifieddate" <>
        if(after_cursor, do: "&after=#{after_cursor}", else: "")

    case HTTP.get_json(url, headers: headers) do
      {:ok, %{status: 200, body: %{"results" => items} = body}} ->
        next_after = get_in(body, ["paging", "next", "after"])
        {:ok, items, next_after}

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

  defp parse_cursor(nil), do: {"", nil}

  defp parse_cursor(cursor) when is_binary(cursor) do
    case String.split(cursor, ":", parts: 2) do
      [obj, after_c] -> {obj, if(after_c == "", do: nil, else: after_c)}
      _ -> {"", nil}
    end
  end

  defp require_hubspot_auth(config) do
    token = Map.get(config, "access_token") || Map.get(config, :access_token)

    oauth? =
      Enum.all?([:client_id, :client_secret, :refresh_token], fn k ->
        Map.has_key?(config, k) or Map.has_key?(config, Atom.to_string(k))
      end)

    cond do
      is_binary(token) and token != "" -> :ok
      oauth? -> :ok
      true -> {:error, :missing_credentials}
    end
  end
end
