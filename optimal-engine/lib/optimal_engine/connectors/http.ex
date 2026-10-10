defmodule OptimalEngine.Connectors.HTTP do
  @moduledoc """
  Thin wrapper around `:httpc` for connector adapters.

  We deliberately avoid pulling in `Req` / `Finch` / `HTTPoison`: the
  engine is an Elixir app consumed as a library, and each extra
  dependency widens the attack surface for downstream users.

  What this module gives you beyond `:httpc`:

    * JSON request + response handling
    * Uniform `{:ok, %{status, headers, body}}` / `{:error, reason}` shape
    * Automatic `Retry-After` extraction on 429
    * Adapter-owned rate-limit bucket via `RateLimit.wait/4`
    * A single seam for `Mox` in tests — see `request/2`

  Swap to Finch in Phase 10 when pooling becomes critical. The public
  API is designed so callers won't change.
  """

  alias OptimalEngine.Connectors.RateLimit

  @default_timeout 10_000

  @type method :: :get | :post | :put | :delete | :patch
  @type headers :: [{String.t(), String.t()}]
  @type body :: map() | binary() | nil
  @type request_opts :: keyword()
  @type response :: %{status: non_neg_integer(), headers: headers(), body: any()}

  @doc """
  Make an HTTP request. Options:

    * `:method`         — `:get | :post | :put | :delete | :patch` (default `:get`)
    * `:headers`        — list of `{name, value}`
    * `:body`           — map (will be JSON-encoded) or raw binary
    * `:timeout`        — ms (default 10_000)
    * `:ratelimit`      — `{bucket_key, rate, burst}` to wait on before firing
    * `:parse_json`     — try to decode response body (default `true`)
  """
  @spec request(String.t(), request_opts()) :: {:ok, response()} | {:error, term()}
  def request(url, opts \\ []) when is_binary(url) do
    with :ok <- maybe_wait_for_quota(opts),
         {:ok, raw} <- fire(url, opts) do
      {:ok, normalize(raw, opts)}
    end
  end

  @doc "Convenience GET with JSON-decoded body."
  @spec get_json(String.t(), request_opts()) :: {:ok, response()} | {:error, term()}
  def get_json(url, opts \\ []) do
    request(url, Keyword.put(opts, :method, :get))
  end

  @doc "Convenience POST with JSON-encoded body + JSON-decoded response."
  @spec post_json(String.t(), map(), request_opts()) :: {:ok, response()} | {:error, term()}
  def post_json(url, body, opts \\ []) when is_map(body) do
    opts = opts |> Keyword.put(:method, :post) |> Keyword.put(:body, body)
    request(url, opts)
  end

  # ─── private ─────────────────────────────────────────────────────────────

  defp maybe_wait_for_quota(opts) do
    case Keyword.get(opts, :ratelimit) do
      nil -> :ok
      {bucket, rate, burst} -> RateLimit.wait(bucket, rate, burst, timeout: 5_000)
    end
  end

  defp fire(url, opts) do
    case Application.get_env(:optimal_engine, :http_mock) do
      nil -> fire_httpc(url, opts)
      mock_fn when is_function(mock_fn, 2) -> mock_fn.(url, opts)
    end
  end

  defp fire_httpc(url, opts) do
    method = Keyword.get(opts, :method, :get)
    headers = build_headers(opts)
    timeout = Keyword.get(opts, :timeout, @default_timeout)

    {content_type, body_payload} = encode_body(Keyword.get(opts, :body), method)

    http_request =
      case method do
        :get -> {String.to_charlist(url), to_charlist_headers(headers)}
        :delete -> {String.to_charlist(url), to_charlist_headers(headers)}
        _ -> {String.to_charlist(url), to_charlist_headers(headers), content_type, body_payload}
      end

    http_opts = [{:timeout, timeout}, {:connect_timeout, timeout}]

    :httpc.request(method, http_request, http_opts, body_format: :binary)
  end

  defp build_headers(opts) do
    [{"accept", "application/json"} | Keyword.get(opts, :headers, [])]
    |> Enum.uniq_by(fn {k, _} -> String.downcase(k) end)
  end

  defp encode_body(nil, _), do: {~c"application/json", ""}
  defp encode_body(bin, _) when is_binary(bin), do: {~c"application/octet-stream", bin}

  defp encode_body(map, _) when is_map(map) do
    {~c"application/json", Jason.encode!(map)}
  end

  defp to_charlist_headers(headers) do
    Enum.map(headers, fn {k, v} -> {String.to_charlist(k), String.to_charlist(v)} end)
  end

  # :httpc returns {:ok, {{vsn, status, reason}, headers, body}}.
  # fire/2 returns that tuple directly; request/2 unwraps the {:ok, raw} via
  # a with-clause, so normalize receives the inner raw value: {{vsn...}, headers, body}.
  defp normalize({{_vsn, status, _reason}, resp_headers, raw_body}, opts) do
    headers = Enum.map(resp_headers, fn {k, v} -> {to_string(k), to_string(v)} end)
    body = maybe_decode(raw_body, headers, opts)
    %{status: status, headers: headers, body: body}
  end

  defp normalize({:error, reason}, _), do: {:error, reason}

  defp maybe_decode(body, headers, opts) do
    parse? = Keyword.get(opts, :parse_json, true)

    content_type =
      headers
      |> Enum.find(fn {k, _} -> String.downcase(k) == "content-type" end)
      |> case do
        {_, v} -> String.downcase(v)
        _ -> ""
      end

    if parse? and String.contains?(content_type, "json") do
      case Jason.decode(body) do
        {:ok, decoded} -> decoded
        _ -> body
      end
    else
      body
    end
  end

  @doc "Extract `Retry-After` seconds from response headers, if present."
  @spec retry_after(headers()) :: non_neg_integer() | nil
  def retry_after(headers) do
    headers
    |> Enum.find(fn {k, _} -> String.downcase(k) == "retry-after" end)
    |> case do
      {_, v} ->
        case Integer.parse(v) do
          {n, _} when n > 0 -> n * 1000
          _ -> nil
        end

      _ ->
        nil
    end
  end
end
