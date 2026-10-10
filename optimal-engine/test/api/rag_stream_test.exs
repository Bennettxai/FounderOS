defmodule OptimalEngine.API.RagStreamTest do
  @moduledoc """
  Integration tests for GET /api/rag/stream.

  Strategy: we call the router through Plug.Test (same pattern as router_test.exs).
  Because SSE sends a chunked response, we assert on the accumulated `resp_body`
  which Plug.Test collects automatically.

  All SSE integration tests pass `skip_intent=true` so they bypass Ollama and
  complete in milliseconds regardless of local model availability. The skip_intent
  parameter is a test-only shortcut honoured only in Mix.env() == :test.

  The :intent event is still emitted even with skip_intent=true — the fallback
  analyzer populates the intent map and the event hook fires unconditionally.
  """

  use ExUnit.Case, async: false

  import Plug.Test
  import Plug.Conn

  alias OptimalEngine.API.RateLimiter
  alias OptimalEngine.API.Router
  alias OptimalEngine.Wiki.{Page, Store}

  @opts Router.init([])

  # Reset the rate-limiter bucket before each test so the full suite does
  # not trigger 429s when many /api/rag/stream requests share one IP bucket.
  setup do
    RateLimiter.reset()
    :ok
  end

  # ── helpers ──────────────────────────────────────────────────────────────

  defp stream_get(path) do
    conn(:get, path)
    |> Router.call(@opts)
  end

  # Append skip_intent=true to any /api/rag/stream URL. This bypasses Ollama
  # so tests are fast and not environment-dependent.
  defp with_skip_intent(path) do
    joiner = if String.contains?(path, "?"), do: "&", else: "?"
    path <> joiner <> "skip_intent=true"
  end

  # Parse an SSE body into a list of `%{event: string, data: map}` entries.
  # Ignores comment lines (": keepalive\n\n") and blank separators.
  defp parse_sse(body) when is_binary(body) do
    body
    |> String.split("\n\n")
    |> Enum.flat_map(fn block ->
      lines = String.split(block, "\n", trim: true)

      event_line = Enum.find(lines, &String.starts_with?(&1, "event: "))
      data_line = Enum.find(lines, &String.starts_with?(&1, "data: "))

      if event_line && data_line do
        event = event_line |> String.trim_leading("event: ") |> String.trim()
        raw = data_line |> String.trim_leading("data: ") |> String.trim()

        case Jason.decode(raw) do
          {:ok, data} -> [%{event: event, data: data}]
          _ -> []
        end
      else
        []
      end
    end)
  end

  defp event_names(events), do: Enum.map(events, & &1.event)

  # ── tests: 400 guard ─────────────────────────────────────────────────────

  describe "GET /api/rag/stream — empty query guard" do
    test "returns 400 JSON when query param is absent" do
      conn = stream_get("/api/rag/stream")
      assert conn.status == 400
      assert {:ok, body} = Jason.decode(conn.resp_body)
      assert body["error"] =~ "query"
    end

    test "returns 400 JSON when query param is empty string" do
      conn = stream_get("/api/rag/stream?query=")
      assert conn.status == 400
      assert {:ok, body} = Jason.decode(conn.resp_body)
      assert body["error"] =~ "query"
    end
  end

  # ── tests: SSE headers ───────────────────────────────────────────────────

  describe "GET /api/rag/stream — SSE headers" do
    test "responds with text/event-stream content type" do
      path =
        "/api/rag/stream?query=pricing-probe-#{System.unique_integer([:positive])}"
        |> with_skip_intent()

      conn = stream_get(path)
      assert conn.status == 200

      [ct] = get_resp_header(conn, "content-type")
      assert ct =~ "text/event-stream"
    end

    test "responds with cache-control: no-cache" do
      path =
        "/api/rag/stream?query=probe-#{System.unique_integer([:positive])}"
        |> with_skip_intent()

      conn = stream_get(path)
      [cc] = get_resp_header(conn, "cache-control")
      assert cc =~ "no-cache"
    end
  end

  # ── tests: event ordering — chunks path ──────────────────────────────────

  describe "GET /api/rag/stream — chunks path (no wiki hit)" do
    test "emits intent event before envelope and done" do
      query = "stream-chunks-probe-#{System.unique_integer([:positive])}"

      path =
        "/api/rag/stream?query=#{URI.encode(query)}"
        |> with_skip_intent()

      conn = stream_get(path)
      assert conn.status == 200

      events = parse_sse(conn.resp_body)
      names = event_names(events)

      assert "intent" in names, "expected :intent event, got: #{inspect(names)}"

      intent_idx = Enum.find_index(names, &(&1 == "intent"))
      envelope_idx = Enum.find_index(names, &(&1 == "envelope"))
      done_idx = Enum.find_index(names, &(&1 == "done"))

      assert is_integer(envelope_idx), "expected :envelope event, got: #{inspect(names)}"
      assert is_integer(done_idx), "expected :done event, got: #{inspect(names)}"
      assert intent_idx < envelope_idx, "intent must precede envelope"
      assert envelope_idx < done_idx, "envelope must precede done"
    end

    test "intent event payload has expected keys" do
      query = "stream-intent-keys-#{System.unique_integer([:positive])}"

      path =
        "/api/rag/stream?query=#{URI.encode(query)}&format=markdown"
        |> with_skip_intent()

      conn = stream_get(path)
      assert conn.status == 200

      events = parse_sse(conn.resp_body)
      intent_event = Enum.find(events, &(&1.event == "intent"))

      assert intent_event, "expected intent event in: #{inspect(event_names(events))}"

      assert Map.has_key?(intent_event.data, "expanded_query") or
               Map.has_key?(intent_event.data, "intent_type"),
             "intent payload missing expected keys: #{inspect(intent_event.data)}"
    end

    test "envelope event payload has body, format, sources" do
      query = "stream-envelope-keys-#{System.unique_integer([:positive])}"

      path =
        "/api/rag/stream?query=#{URI.encode(query)}"
        |> with_skip_intent()

      conn = stream_get(path)
      assert conn.status == 200

      events = parse_sse(conn.resp_body)
      env_event = Enum.find(events, &(&1.event == "envelope"))

      assert env_event, "expected envelope event in: #{inspect(event_names(events))}"
      assert Map.has_key?(env_event.data, "body"), "envelope missing body"
      assert Map.has_key?(env_event.data, "format"), "envelope missing format"
      assert Map.has_key?(env_event.data, "sources"), "envelope missing sources"
    end

    test "done event payload has elapsed_ms and source" do
      query = "stream-done-keys-#{System.unique_integer([:positive])}"

      path =
        "/api/rag/stream?query=#{URI.encode(query)}"
        |> with_skip_intent()

      conn = stream_get(path)
      assert conn.status == 200

      events = parse_sse(conn.resp_body)
      done_event = Enum.find(events, &(&1.event == "done"))

      assert done_event, "expected done event in: #{inspect(event_names(events))}"
      assert Map.has_key?(done_event.data, "elapsed_ms"), "done missing elapsed_ms"
      assert Map.has_key?(done_event.data, "source"), "done missing source"
    end
  end

  # ── tests: wiki path ─────────────────────────────────────────────────────

  describe "GET /api/rag/stream — wiki hit path" do
    test "emits wiki_hit event before envelope when wiki page exists" do
      suffix = System.unique_integer([:positive])
      slug = "rag-stream-wiki-#{suffix}"

      :ok =
        Store.put(%Page{
          tenant_id: "default",
          slug: slug,
          audience: "default",
          version: 1,
          frontmatter: %{"slug" => slug},
          body: "## Pricing\n\nSee {{cite: optimal://test}}."
        })

      path =
        "/api/rag/stream?query=#{URI.encode(slug)}"
        |> with_skip_intent()

      conn = stream_get(path)
      assert conn.status == 200

      events = parse_sse(conn.resp_body)
      names = event_names(events)

      wiki_idx = Enum.find_index(names, &(&1 == "wiki_hit"))
      envelope_idx = Enum.find_index(names, &(&1 == "envelope"))

      assert is_integer(wiki_idx), "expected wiki_hit event, got: #{inspect(names)}"
      assert is_integer(envelope_idx)
      assert wiki_idx < envelope_idx, "wiki_hit must precede envelope"
    end

    test "wiki_hit event payload has slug, audience, version" do
      suffix = System.unique_integer([:positive])
      slug = "rag-stream-wiki-payload-#{suffix}"

      :ok =
        Store.put(%Page{
          tenant_id: "default",
          slug: slug,
          audience: "default",
          version: 1,
          frontmatter: %{"slug" => slug},
          body: "# Wiki content"
        })

      path =
        "/api/rag/stream?query=#{URI.encode(slug)}"
        |> with_skip_intent()

      conn = stream_get(path)
      assert conn.status == 200

      events = parse_sse(conn.resp_body)
      wiki_event = Enum.find(events, &(&1.event == "wiki_hit"))

      assert wiki_event, "expected wiki_hit event"
      assert Map.has_key?(wiki_event.data, "slug")
      assert Map.has_key?(wiki_event.data, "audience")
      assert Map.has_key?(wiki_event.data, "version")
    end
  end

  # ── tests: workspace / audience param forwarding ──────────────────────────

  describe "GET /api/rag/stream — param forwarding" do
    test "accepts workspace param without error" do
      query = "stream-workspace-probe-#{System.unique_integer([:positive])}"

      path =
        "/api/rag/stream?query=#{URI.encode(query)}&workspace=default"
        |> with_skip_intent()

      conn = stream_get(path)
      assert conn.status == 200

      events = parse_sse(conn.resp_body)
      assert "done" in event_names(events)
    end

    test "accepts audience param without error" do
      query = "stream-audience-probe-#{System.unique_integer([:positive])}"

      path =
        "/api/rag/stream?query=#{URI.encode(query)}&audience=default"
        |> with_skip_intent()

      conn = stream_get(path)
      assert conn.status == 200

      events = parse_sse(conn.resp_body)
      assert "done" in event_names(events)
    end
  end

  # ── tests: RagStream unit ────────────────────────────────────────────────

  describe "RagStream.start_link/4" do
    alias OptimalEngine.Retrieval.{RagStream, Receiver}

    test "sends rag_stream_done to listener after a successful ask" do
      query = "rag-stream-unit-#{System.unique_integer([:positive])}"
      receiver = Receiver.anonymous()

      {:ok, task} =
        RagStream.start_link(query, receiver, self(),
          workspace_id: "default",
          skip_intent: true,
          skip_wiki: true
        )

      # skip_intent: true makes this sub-millisecond; 15s is ample headroom
      assert_receive {:rag_stream_done, result}, 15_000
      assert Map.has_key?(result, :source)
      assert Map.has_key?(result, :envelope)
      assert Map.has_key?(result, :trace)

      Task.await(task, 15_000)
    end

    test "sends rag_stream_event for :intent stage" do
      # skip_intent: true still fires the :intent event — the event hook runs
      # unconditionally after the intent block regardless of which path ran.
      query = "rag-stream-events-#{System.unique_integer([:positive])}"
      receiver = Receiver.anonymous()

      {:ok, task} =
        RagStream.start_link(query, receiver, self(),
          workspace_id: "default",
          skip_intent: true,
          skip_wiki: true
        )

      messages = collect_stream_messages([], 10_000)

      events =
        Enum.filter(messages, fn
          {:rag_stream_event, _, _} -> true
          _ -> false
        end)

      event_atoms = Enum.map(events, fn {:rag_stream_event, e, _} -> e end)
      assert :intent in event_atoms, "expected :intent in #{inspect(event_atoms)}"

      Task.await(task, 15_000)
    end

    test "intent event fires before done message" do
      # skip_intent: true makes the intent step deterministic and instant.
      query = "rag-stream-order-#{System.unique_integer([:positive])}"
      receiver = Receiver.anonymous()

      {:ok, task} =
        RagStream.start_link(query, receiver, self(),
          workspace_id: "default",
          skip_intent: true,
          skip_wiki: true
        )

      messages = collect_stream_messages([], 10_000)

      intent_pos =
        Enum.find_index(messages, fn
          {:rag_stream_event, :intent, _} -> true
          _ -> false
        end)

      done_pos =
        Enum.find_index(messages, fn
          {:rag_stream_done, _} -> true
          _ -> false
        end)

      assert is_integer(intent_pos), "no :intent event found in #{inspect(messages)}"
      assert is_integer(done_pos), "no :rag_stream_done found in #{inspect(messages)}"
      assert intent_pos < done_pos, "intent must arrive before done"

      Task.await(task, 15_000)
    end
  end

  # ── private helpers ───────────────────────────────────────────────────────

  # Collects {:rag_stream_event, _, _} and {:rag_stream_done, _} messages
  # until :rag_stream_done or :rag_stream_error is received, or timeout fires.
  defp collect_stream_messages(acc, timeout) do
    receive do
      {:rag_stream_done, _} = msg ->
        Enum.reverse([msg | acc])

      {:rag_stream_error, _} = msg ->
        Enum.reverse([msg | acc])

      {:rag_stream_event, _, _} = msg ->
        collect_stream_messages([msg | acc], timeout)

      _other ->
        collect_stream_messages(acc, timeout)
    after
      timeout ->
        Enum.reverse(acc)
    end
  end
end
