defmodule OptimalEngine.Memory.WebhookTest do
  @moduledoc """
  Tests for OptimalEngine.Memory.Webhook.

  HTTP server tests spin up a raw :gen_tcp listener on a random port for each
  test that needs to capture outbound POST requests. This avoids external deps
  and proves the full `:httpc` delivery path.
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.Memory.{Subscription, Surfacer, Webhook}

  # ── Helpers ──────────────────────────────────────────────────────────────────

  # Unique workspace per test to avoid cross-test pollution in the shared DB.
  defp ws, do: "wh-test-ws-#{:erlang.unique_integer([:positive])}"

  # Spin up a minimal :gen_tcp HTTP server on a random port.
  # Captures the first POST it receives and sends it to `parent`.
  # Returns {"http://127.0.0.1:<port>", ref}.
  defp start_test_server(parent \\ self()) do
    port = Enum.random(30_000..39_999)
    ref = make_ref()

    {:ok, listen_sock} =
      :gen_tcp.listen(port, [
        :binary,
        packet: :raw,
        active: false,
        reuseaddr: true
      ])

    Task.start(fn ->
      case :gen_tcp.accept(listen_sock, 15_000) do
        {:ok, client} ->
          {:ok, raw} = recv_http_request(client)
          parsed = parse_http_request(raw)
          send(parent, {:webhook_received, ref, parsed})

          response =
            "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: 10\r\n\r\n{\"ok\":true}"

          :gen_tcp.send(client, response)
          :gen_tcp.close(client)

        _ ->
          :ok
      end

      :gen_tcp.close(listen_sock)
    end)

    {"http://127.0.0.1:#{port}", ref}
  end

  # Read from a TCP socket until the HTTP body is complete.
  defp recv_http_request(sock) do
    recv_all(sock, "")
  end

  defp recv_all(sock, acc) do
    case :gen_tcp.recv(sock, 0, 5_000) do
      {:ok, chunk} ->
        data = acc <> chunk

        if has_full_body?(data) do
          {:ok, data}
        else
          recv_all(sock, data)
        end

      {:error, :closed} ->
        {:ok, acc}

      {:error, :timeout} ->
        {:ok, acc}
    end
  end

  defp has_full_body?(data) do
    case String.split(data, "\r\n\r\n", parts: 2) do
      [headers, body] ->
        case Regex.run(~r/content-length:\s*(\d+)/i, headers) do
          [_, len_str] -> byte_size(body) >= String.to_integer(len_str)
          nil -> true
        end

      _ ->
        false
    end
  end

  defp parse_http_request(raw) do
    [header_section | body_parts] = String.split(raw, "\r\n\r\n", parts: 2)
    body = Enum.join(body_parts)
    [_request_line | header_lines] = String.split(header_section, "\r\n")

    headers =
      Enum.reduce(header_lines, %{}, fn line, acc ->
        case String.split(line, ":", parts: 2) do
          [k, v] -> Map.put(acc, String.downcase(String.trim(k)), String.trim(v))
          _ -> acc
        end
      end)

    %{body: body, headers: headers}
  end

  # ── sign/2 ───────────────────────────────────────────────────────────────────

  describe "sign/2" do
    test "produces a lowercase hex-encoded 64-char HMAC-SHA256 digest" do
      sig = Webhook.sign("hello world", "my-secret")
      assert String.match?(sig, ~r/^[0-9a-f]{64}$/)
    end

    test "is deterministic for same inputs" do
      assert Webhook.sign("body", "key") == Webhook.sign("body", "key")
    end

    test "differs for different body" do
      refute Webhook.sign("body-a", "key") == Webhook.sign("body-b", "key")
    end

    test "differs for different secret" do
      refute Webhook.sign("body", "key-a") == Webhook.sign("body", "key-b")
    end

    test "matches stdlib :crypto.mac output" do
      expected =
        :crypto.mac(:hmac, :sha256, "secret", "payload")
        |> Base.encode16(case: :lower)

      assert Webhook.sign("payload", "secret") == expected
    end
  end

  # ── deliver/2 — missing webhook_url ──────────────────────────────────────────

  describe "deliver/2 when no webhook_url" do
    test "returns error immediately for missing key" do
      assert {:error, :no_webhook_url} = Webhook.deliver(%{event: "test"}, %{})
    end

    test "returns error immediately for blank webhook_url" do
      assert {:error, :no_webhook_url} =
               Webhook.deliver(%{event: "test"}, %{"webhook_url" => ""})
    end
  end

  # ── deliver/2 — successful POST ──────────────────────────────────────────────

  describe "deliver/2 with a live HTTP server" do
    test "sends POST with Content-Type application/json" do
      {url, ref} = start_test_server()
      opts = %{"webhook_url" => url}

      assert {:ok, 200} = Webhook.deliver(%{subscription_id: "sub:test"}, opts)

      assert_receive {:webhook_received, ^ref, captured}, 3_000
      assert captured.headers["content-type"] =~ "application/json"
    end

    test "sends JSON-encoded payload as body" do
      {url, ref} = start_test_server()
      payload = %{subscription_id: "sub:test", trigger: "wiki_updated", score: 0.85}

      assert {:ok, 200} = Webhook.deliver(payload, %{"webhook_url" => url})

      assert_receive {:webhook_received, ^ref, captured}, 3_000
      decoded = Jason.decode!(captured.body)
      assert decoded["subscription_id"] == "sub:test"
      assert decoded["score"] == 0.85
    end

    test "adds X-Optimal-Signature header when secret is provided" do
      {url, ref} = start_test_server()
      secret = "my-webhook-secret"
      payload = %{event: "test"}
      opts = %{"webhook_url" => url, "webhook_secret" => secret}

      assert {:ok, 200} = Webhook.deliver(payload, opts)

      assert_receive {:webhook_received, ^ref, captured}, 3_000
      sig_header = captured.headers["x-optimal-signature"]
      assert is_binary(sig_header)
      assert String.starts_with?(sig_header, "sha256=")

      expected_sig = "sha256=" <> Webhook.sign(Jason.encode!(payload), secret)
      assert sig_header == expected_sig
    end

    test "does not add X-Optimal-Signature when no secret" do
      {url, ref} = start_test_server()

      assert {:ok, 200} = Webhook.deliver(%{event: "test"}, %{"webhook_url" => url})

      assert_receive {:webhook_received, ^ref, captured}, 3_000
      refute Map.has_key?(captured.headers, "x-optimal-signature")
    end

    test "includes extra webhook_headers in the request" do
      {url, ref} = start_test_server()

      opts = %{
        "webhook_url" => url,
        "webhook_headers" => %{"x-tenant-id" => "sample", "x-custom" => "value"}
      }

      assert {:ok, 200} = Webhook.deliver(%{event: "test"}, opts)

      assert_receive {:webhook_received, ^ref, captured}, 3_000
      assert captured.headers["x-tenant-id"] == "sample"
      assert captured.headers["x-custom"] == "value"
    end
  end

  # ── deliver/2 — failed delivery ──────────────────────────────────────────────

  describe "deliver/2 when endpoint is unreachable" do
    # Validates the retry-exhaustion contract. The full backoff schedule is
    # 1s + 4s + 16s = 21s, so we allow 30s. Production behavior is correct;
    # the test is intentionally slow to prove the contract.
    @tag timeout: 30_000
    test "returns error tuple without raising after all retries" do
      opts = %{"webhook_url" => "http://this-host-does-not-exist.invalid/hook"}

      result = Webhook.deliver(%{event: "test"}, opts)
      assert {:error, _reason} = result
    end
  end

  # ── Surfacer integration ──────────────────────────────────────────────────────

  describe "Surfacer webhook delivery on surface event" do
    test "fires webhook when subscription has webhook_url" do
      workspace_id = ws()
      {url, ref} = start_test_server()

      {:ok, sub} =
        Subscription.create(%{
          workspace_id: workspace_id,
          scope: :workspace,
          metadata: %{"webhook_url" => url}
        })

      Surfacer.notify_wiki_updated(workspace_id, "pricing-decision")

      # The Surfacer uses Task.start (async) — wait for the Task to deliver
      assert_receive {:webhook_received, ^ref, captured}, 3_000
      decoded = Jason.decode!(captured.body)
      assert decoded["workspace_id"] == workspace_id
      assert decoded["trigger"] == "wiki_updated"

      Subscription.delete(sub.id)
    end

    test "does not attempt webhook delivery when subscription has no webhook_url" do
      workspace_id = ws()

      {:ok, sub} =
        Subscription.create(%{
          workspace_id: workspace_id,
          scope: :workspace,
          metadata: %{}
        })

      Surfacer.notify_wiki_updated(workspace_id, "no-webhook-slug")

      # No webhook task fires — nothing arrives in mailbox
      refute_receive {:webhook_received, _, _}, 500

      Subscription.delete(sub.id)
    end

    test "webhook failure does not crash the Surfacer" do
      workspace_id = ws()

      # Invalid URL — DNS fails instantly, no long TCP timeouts
      {:ok, sub} =
        Subscription.create(%{
          workspace_id: workspace_id,
          scope: :workspace,
          metadata: %{"webhook_url" => "http://this-host-does-not-exist.invalid/hook"}
        })

      surfacer_pid = Process.whereis(Surfacer)
      assert is_pid(surfacer_pid)

      Surfacer.notify_wiki_updated(workspace_id, "crash-test-slug")

      # Give the Task time to run (it will fail, log a warning, and exit cleanly)
      Process.sleep(300)

      # Surfacer must still be alive
      assert Process.alive?(surfacer_pid)
      assert Process.whereis(Surfacer) == surfacer_pid

      Subscription.delete(sub.id)
    end
  end
end
