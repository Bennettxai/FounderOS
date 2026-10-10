defmodule OptimalEngine.Memory.Webhook do
  @moduledoc """
  Delivers surfacing events to webhook endpoints.

  Sends a JSON POST to the configured URL when a surfacing subscription has a
  `metadata.webhook_url` set. Signing is optional: when `metadata.webhook_secret`
  is present, an HMAC-SHA256 signature is added as the `X-Optimal-Signature` header.

  Retry strategy: 3 attempts with exponential backoff — 1s, 4s, 16s.
  Each attempt times out after 10 seconds. Failures are logged at warning level
  but never propagate exceptions — the Surfacer must never crash due to a webhook
  failure.

  Usage:

      # Fire-and-forget from the Surfacer:
      Task.start(fn -> Webhook.deliver(payload, opts) end)

      # opts keys (string keys, from metadata JSON):
      #   "webhook_url"     — target URL (required)
      #   "webhook_secret"  — HMAC signing key (optional)
      #   "webhook_headers" — map of extra headers (optional)
  """

  require Logger

  @timeout_ms 10_000
  # Backoff delays in milliseconds: 1s, 4s, 16s
  @backoffs [1_000, 4_000, 16_000]

  @doc """
  POST the surface payload to the webhook URL.

  `payload` is the surface map (already structured by the Surfacer).
  `opts` is the subscription metadata map (string-keyed, from JSON).

  Returns `{:ok, status}` on a 2xx response, `{:error, term()}` after all retries
  are exhausted. Always returns — never raises.
  """
  @spec deliver(map(), map()) :: {:ok, integer()} | {:error, term()}
  def deliver(payload, opts) when is_map(opts) do
    url = Map.get(opts, "webhook_url")

    if is_nil(url) or url == "" do
      {:error, :no_webhook_url}
    else
      body = Jason.encode!(payload)
      secret = Map.get(opts, "webhook_secret")
      extra_headers = Map.get(opts, "webhook_headers", %{})
      headers = build_headers(body, secret, extra_headers)
      attempt_with_retries(url, headers, body, @backoffs)
    end
  end

  @doc """
  Compute an HMAC-SHA256 signature over `body` using `secret`.

  Returns the lowercase hex-encoded digest. Used to populate the
  `X-Optimal-Signature: sha256=<hex>` header.
  """
  @spec sign(binary(), binary()) :: binary()
  def sign(body, secret) when is_binary(body) and is_binary(secret) do
    :crypto.mac(:hmac, :sha256, secret, body)
    |> Base.encode16(case: :lower)
  end

  # ---------------------------------------------------------------------------
  # Private
  # ---------------------------------------------------------------------------

  # Attempt delivery, retrying with the next backoff on failure.
  # `remaining_backoffs` is the list of delays still available; an empty list
  # means no more retries — return the last error.
  defp attempt_with_retries(url, headers, body, remaining_backoffs) do
    case attempt_post(url, headers, body) do
      {:ok, status} when status in 200..299 ->
        {:ok, status}

      {:ok, status} ->
        Logger.warning("[Webhook] Non-2xx response #{status} from #{url}")

        retry_after_failure(
          url,
          headers,
          body,
          remaining_backoffs,
          {:error, {:http_status, status}}
        )

      {:error, reason} ->
        Logger.warning("[Webhook] Delivery failed for #{url}: #{inspect(reason)}")
        retry_after_failure(url, headers, body, remaining_backoffs, {:error, reason})
    end
  end

  defp retry_after_failure(_url, _headers, _body, [], last_error) do
    Logger.warning("[Webhook] All retries exhausted")
    last_error
  end

  defp retry_after_failure(url, headers, body, [delay | rest], _last_error) do
    Process.sleep(delay)
    attempt_with_retries(url, headers, body, rest)
  end

  defp build_headers(body, secret, extra_headers) do
    base = [
      {"content-type", "application/json"},
      {"user-agent", "OptimalEngine-Webhook/1.0"}
    ]

    base
    |> maybe_add_signature(body, secret)
    |> add_extra_headers(extra_headers)
  end

  defp maybe_add_signature(headers, _body, nil), do: headers
  defp maybe_add_signature(headers, _body, ""), do: headers

  defp maybe_add_signature(headers, body, secret) do
    sig = sign(body, secret)
    [{"x-optimal-signature", "sha256=#{sig}"} | headers]
  end

  defp add_extra_headers(headers, extra) when is_map(extra) do
    Enum.reduce(extra, headers, fn {k, v}, acc ->
      if is_binary(k) and is_binary(v) do
        [{String.downcase(k), v} | acc]
      else
        acc
      end
    end)
  end

  defp add_extra_headers(headers, _), do: headers

  defp attempt_post(url, headers, body) do
    uri = String.to_charlist(url)
    body_chars = :binary.bin_to_list(body)

    # :httpc expects erlang charlist headers; content-type is passed separately
    erl_headers =
      headers
      |> Enum.reject(fn {k, _} -> k == "content-type" end)
      |> Enum.map(fn {k, v} -> {String.to_charlist(k), String.to_charlist(v)} end)

    request = {uri, erl_headers, ~c"application/json", body_chars}
    http_opts = [{:timeout, @timeout_ms}, {:connect_timeout, @timeout_ms}]
    opts = [{:body_format, :binary}]

    case :httpc.request(:post, request, http_opts, opts) do
      {:ok, {{_version, status, _reason}, _resp_headers, _resp_body}} ->
        {:ok, status}

      {:error, reason} ->
        {:error, reason}
    end
  rescue
    e -> {:error, {:exception, Exception.message(e)}}
  end
end
