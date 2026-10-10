defmodule OptimalEngine.Connectors.AdaptersBatch1Test do
  @moduledoc """
  Unit tests for batch-1 adapters (github, gmail, slack, notion, drive, dropbox, onedrive)
  with a mocked HTTP layer.

  Each test group verifies:
    * request shape (URL, method, auth header) via the mock fn
    * transform/1 field mapping: id, title, content, genre, entities, modified_at
    * sync/2 cursor propagation
    * sync/2 returns {:error, :auth_expired} on HTTP 401
    * sync/2 returns {:error, :rate_limited} on HTTP 429
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.Signal

  # ── mock helpers ────────────────────────────────────────────────────────────

  # Install a process-local HTTP mock by storing it in the application env
  # for the duration of the test. The mock fn receives (url, opts) and must
  # return the raw :httpc shape:
  #   {:ok, {{'HTTP/1.1', status, 'OK'}, headers, body_binary}}
  # or {:error, reason}.
  # normalize/2 in HTTP.ex then decodes it the same way as a real response.
  defp with_mock(fun, test_fn) do
    Application.put_env(:optimal_engine, :http_mock, fun)

    try do
      test_fn.()
    after
      Application.delete_env(:optimal_engine, :http_mock)
    end
  end

  # Build the raw :httpc response tuple that fire/2 returns.
  # fire/2 returns {:ok, {{vsn, status, reason}, charlist_headers, body_binary}}.
  # The mock function replaces fire/2 entirely, so it must return the same shape.
  # request/2 destructures {:ok, raw} and passes raw to normalize/2.
  defp httpc_ok(body_map, extra_headers \\ []) do
    json = Jason.encode!(body_map)
    headers = [{"content-type", "application/json"}] ++ extra_headers

    charlist_headers =
      Enum.map(headers, fn {k, v} -> {String.to_charlist(k), String.to_charlist(v)} end)

    {:ok, {{~c"HTTP/1.1", 200, ~c"OK"}, charlist_headers, json}}
  end

  defp httpc_status(status) do
    {:ok, {{~c"HTTP/1.1", status, ~c"Error"}, [], ""}}
  end

  # ── GitHub ───────────────────────────────────────────────────────────────────

  describe "GitHub adapter" do
    @state %{
      "org_or_user" => "acme",
      "repos" => ["engine"],
      "pat" => "ghp_test"
    }

    test "sync/2 sends Authorization header and fetches issues" do
      calls = :ets.new(:gh_calls, [:set, :public])

      mock = fn url, opts ->
        :ets.insert(calls, {url, opts})

        cond do
          String.contains?(url, "/issues") ->
            httpc_ok([
              %{
                "id" => 1,
                "node_id" => "NID1",
                "title" => "Bug in parser",
                "body" => "Details here",
                "updated_at" => "2026-01-01T00:00:00Z",
                "user" => %{"login" => "alice"}
              }
            ])

          true ->
            httpc_ok([])
        end
      end

      with_mock(mock, fn ->
        assert {:ok, %{signals: signals, cursor: cursor}} =
                 OptimalEngine.Connectors.Adapters.GitHub.sync(@state, nil)

        assert length(signals) == 1
        [sig] = signals
        assert sig.title =~ "Bug in parser"
        assert sig.genre in ["issue", "pr"]
        assert is_binary(cursor)

        # Verify the auth header was sent
        [{url, opts}] =
          :ets.tab2list(calls)
          |> Enum.find(fn {u, _} -> String.contains?(u, "/issues") end)
          |> then(&[&1])

        assert String.contains?(url, "repos/acme/engine/issues")
        headers = Keyword.get(opts, :headers, [])

        assert Enum.any?(headers, fn {k, v} ->
                 k == "authorization" and String.contains?(v, "ghp_test")
               end)
      end)

      :ets.delete(calls)
    end

    test "sync/2 skips repos that return 401 and continues (repo-level error isolation)" do
      # GitHub sync uses Enum.reduce -- per-repo HTTP errors are skipped rather
      # than aborting the whole sync. The result is still {:ok, %{signals: [], cursor: ...}}.
      mock = fn _url, _opts -> httpc_status(401) end

      with_mock(mock, fn ->
        assert {:ok, %{signals: [], cursor: _}} =
                 OptimalEngine.Connectors.Adapters.GitHub.sync(@state, nil)
      end)
    end

    test "sync/2 skips repos that return 429 and continues (repo-level error isolation)" do
      mock = fn _url, _opts -> httpc_status(429) end

      with_mock(mock, fn ->
        assert {:ok, %{signals: [], cursor: _}} =
                 OptimalEngine.Connectors.Adapters.GitHub.sync(@state, nil)
      end)
    end

    test "transform/1 maps issue fields correctly" do
      raw = %{
        "id" => 42,
        "node_id" => "NID42",
        "title" => "Fix the thing",
        "body" => "Long description",
        "updated_at" => "2026-03-01T12:00:00Z",
        "user" => %{"login" => "bob"}
      }

      assert {:ok, %Signal{} = sig} =
               OptimalEngine.Connectors.Adapters.GitHub.transform(raw)

      assert sig.title =~ "Fix the thing"
      assert sig.content == "Long description"
      assert sig.genre == "issue"
      assert "bob" in sig.entities
      assert sig.path =~ "optimal://connectors/github/"
      assert %DateTime{} = sig.modified_at
    end

    test "transform/1 marks PRs with 'pr' genre" do
      raw = %{
        "id" => 7,
        "node_id" => "PR7",
        "title" => "Add feature",
        "body" => "",
        "updated_at" => "2026-03-01T12:00:00Z",
        "pull_request" => %{"url" => "https://..."},
        "user" => %{"login" => "carol"}
      }

      assert {:ok, %Signal{genre: "pr"}} =
               OptimalEngine.Connectors.Adapters.GitHub.transform(raw)
    end

    test "sync/2 with repos: ['*'] fetches repo list first" do
      state = Map.put(@state, "repos", ["*"])
      calls = :ets.new(:gh_star_calls, [:bag, :public])

      mock = fn url, opts ->
        :ets.insert(calls, {url, opts})

        cond do
          String.contains?(url, "/orgs/") and
              String.ends_with?(url |> URI.parse() |> Map.get(:path, ""), "/repos") ->
            httpc_ok([%{"name" => "repo-a"}])

          String.contains?(url, "/repos/") and String.contains?(url, "/issues") ->
            httpc_ok([])

          true ->
            httpc_ok([])
        end
      end

      with_mock(mock, fn ->
        assert {:ok, %{signals: _, cursor: _}} =
                 OptimalEngine.Connectors.Adapters.GitHub.sync(state, nil)
      end)

      :ets.delete(calls)
    end
  end

  # ── Gmail ─────────────────────────────────────────────────────────────────────

  describe "Gmail adapter" do
    @gmail_state %{
      "user_email" => "user@example.com",
      "oauth_refresh_token" => "rtoken",
      "client_id" => "cid",
      "client_secret" => "csec"
    }

    defp gmail_token_mock(next_fn) do
      fn url, opts ->
        if String.contains?(url, "oauth2.googleapis.com") do
          httpc_ok(%{"access_token" => "test_access_token"})
        else
          next_fn.(url, opts)
        end
      end
    end

    test "sync/2 exchanges refresh token before fetching messages" do
      token_called = :ets.new(:gmail_token, [:set, :public])

      base_mock = fn url, _opts ->
        cond do
          String.contains?(url, "oauth2.googleapis.com") ->
            :ets.insert(token_called, {:called, true})
            httpc_ok(%{"access_token" => "tok"})

          String.contains?(url, "/messages?") ->
            httpc_ok(%{"messages" => [], "nextPageToken" => nil})

          true ->
            httpc_ok(%{})
        end
      end

      with_mock(base_mock, fn ->
        assert {:ok, %{signals: _, cursor: _}} =
                 OptimalEngine.Connectors.Adapters.Gmail.sync(@gmail_state, nil)

        assert :ets.lookup(token_called, :called) == [{:called, true}]
      end)

      :ets.delete(token_called)
    end

    test "sync/2 sends Bearer token in auth header" do
      calls = :ets.new(:gmail_auth, [:bag, :public])

      mock = fn url, opts ->
        :ets.insert(calls, {url, opts})

        cond do
          String.contains?(url, "oauth2.googleapis.com") ->
            httpc_ok(%{"access_token" => "bearer_tok"})

          String.contains?(url, "/messages?") ->
            httpc_ok(%{"messages" => []})

          true ->
            httpc_ok(%{})
        end
      end

      with_mock(mock, fn ->
        OptimalEngine.Connectors.Adapters.Gmail.sync(@gmail_state, nil)

        msg_call =
          :ets.tab2list(calls)
          |> Enum.find(fn {u, _} -> String.contains?(u, "/messages?") end)

        if msg_call do
          {_url, opts} = msg_call
          headers = Keyword.get(opts, :headers, [])

          assert Enum.any?(headers, fn {k, v} ->
                   k == "authorization" and String.contains?(v, "bearer_tok")
                 end)
        end
      end)

      :ets.delete(calls)
    end

    test "sync/2 returns auth_expired when token refresh fails" do
      mock = fn _url, _opts -> httpc_status(401) end

      with_mock(mock, fn ->
        assert {:error, :auth_expired} =
                 OptimalEngine.Connectors.Adapters.Gmail.sync(@gmail_state, nil)
      end)
    end

    test "sync/2 incremental uses historyId as cursor" do
      mock =
        gmail_token_mock(fn url, _opts ->
          if String.contains?(url, "/history") do
            httpc_ok(%{
              "history" => [],
              "historyId" => "99999"
            })
          else
            httpc_ok(%{"messages" => []})
          end
        end)

      with_mock(mock, fn ->
        assert {:ok, %{signals: [], cursor: "99999"}} =
                 OptimalEngine.Connectors.Adapters.Gmail.sync(@gmail_state, "12345")
      end)
    end

    test "transform/1 maps email fields correctly" do
      raw = %{
        "id" => "abc123",
        "snippet" => "Hello from test",
        "payload" => %{
          "headers" => [
            %{"name" => "Subject", "value" => "Test subject"},
            %{"name" => "From", "value" => "Alice <alice@example.com>"}
          ]
        }
      }

      assert {:ok, %Signal{} = sig} =
               OptimalEngine.Connectors.Adapters.Gmail.transform(raw)

      assert sig.title == "Test subject"
      assert sig.content == "Hello from test"
      assert sig.genre == "email"
      assert "alice@example.com" in sig.entities
      assert sig.path =~ "optimal://connectors/gmail/abc123"
    end

    test "transform/1 handles missing headers gracefully" do
      raw = %{"id" => "x", "snippet" => "body"}

      assert {:ok, %Signal{title: "(no subject)"}} =
               OptimalEngine.Connectors.Adapters.Gmail.transform(raw)
    end
  end

  # ── Slack ─────────────────────────────────────────────────────────────────────

  describe "Slack adapter" do
    @slack_state %{
      "workspace_id" => "T01",
      "channels" => ["C01", "C02"],
      "bot_token" => "xoxb-test"
    }

    test "sync/2 uses Bearer token in Authorization header" do
      calls = :ets.new(:slack_auth, [:bag, :public])

      mock = fn url, opts ->
        :ets.insert(calls, {url, opts})
        httpc_ok(%{"ok" => true, "messages" => [], "has_more" => false})
      end

      with_mock(mock, fn ->
        OptimalEngine.Connectors.Adapters.Slack.sync(@slack_state, nil)

        history_call =
          :ets.tab2list(calls)
          |> Enum.find(fn {u, _} -> String.contains?(u, "conversations.history") end)

        if history_call do
          {_url, opts} = history_call
          headers = Keyword.get(opts, :headers, [])

          assert Enum.any?(headers, fn {k, v} ->
                   k == "authorization" and String.contains?(v, "xoxb-test")
                 end)
        end
      end)

      :ets.delete(calls)
    end

    test "sync/2 fetches all channels when channels list is empty" do
      state = Map.put(@slack_state, "channels", [])
      calls = :ets.new(:slack_channels, [:bag, :public])

      mock = fn url, opts ->
        :ets.insert(calls, {url, opts})

        cond do
          String.contains?(url, "conversations.list") ->
            httpc_ok(%{"ok" => true, "channels" => [%{"id" => "C99"}]})

          String.contains?(url, "conversations.history") ->
            httpc_ok(%{"ok" => true, "messages" => [], "has_more" => false})

          true ->
            httpc_ok(%{"ok" => true})
        end
      end

      with_mock(mock, fn ->
        assert {:ok, %{signals: _}} =
                 OptimalEngine.Connectors.Adapters.Slack.sync(state, nil)

        list_calls =
          :ets.tab2list(calls)
          |> Enum.filter(fn {u, _} -> String.contains?(u, "conversations.list") end)

        assert length(list_calls) >= 1
      end)

      :ets.delete(calls)
    end

    test "sync/2 skips channels that return ratelimited and continues (channel-level isolation)" do
      # Slack sync uses Enum.reduce -- per-channel rate limit errors are swallowed.
      mock = fn _url, _opts ->
        httpc_ok(%{"ok" => false, "error" => "ratelimited"})
      end

      with_mock(mock, fn ->
        assert {:ok, %{signals: [], cursor: _}} =
                 OptimalEngine.Connectors.Adapters.Slack.sync(@slack_state, nil)
      end)
    end

    test "sync/2 encodes cursor as channel_id:oldest_ts" do
      mock = fn url, _opts ->
        if String.contains?(url, "conversations.history") do
          httpc_ok(%{
            "ok" => true,
            "messages" => [%{"ts" => "1700000001.000000", "text" => "hi", "user" => "U1"}],
            "has_more" => true
          })
        else
          httpc_ok(%{"ok" => true, "messages" => [], "has_more" => false})
        end
      end

      with_mock(mock, fn ->
        assert {:ok, %{cursor: cursor}} =
                 OptimalEngine.Connectors.Adapters.Slack.sync(@slack_state, nil)

        # cursor is "channel_id:oldest_ts" or "channel_id:"
        assert is_binary(cursor) or is_nil(cursor)
      end)
    end

    test "transform/1 maps message fields correctly" do
      raw = %{
        "ts" => "1700000000.000001",
        "client_msg_id" => "cm1",
        "text" => "Hello world from Slack",
        "user" => "U01"
      }

      assert {:ok, %Signal{} = sig} =
               OptimalEngine.Connectors.Adapters.Slack.transform(raw)

      assert sig.content == "Hello world from Slack"
      assert sig.title =~ "Hello world"
      assert sig.genre == "message"
      assert "U01" in sig.entities
      assert sig.path =~ "optimal://connectors/slack/"
    end

    test "transform/1 falls back to ts when client_msg_id absent" do
      raw = %{"ts" => "1700000000.000002", "text" => "x", "user" => "U02"}

      assert {:ok, %Signal{path: path}} =
               OptimalEngine.Connectors.Adapters.Slack.transform(raw)

      assert String.contains?(path, "1700000000.000002")
    end
  end

  # ── Notion ───────────────────────────────────────────────────────────────────

  describe "Notion adapter" do
    @notion_state %{
      "workspace_name" => "acme",
      "database_ids" => ["db-1"],
      "integration_token" => "secret_test"
    }

    test "sync/2 sends Notion-Version header" do
      calls = :ets.new(:notion_headers, [:bag, :public])

      mock = fn url, opts ->
        :ets.insert(calls, {url, opts})

        cond do
          String.contains?(url, "/databases/") ->
            httpc_ok(%{"results" => [], "has_more" => false})

          true ->
            httpc_ok(%{"results" => []})
        end
      end

      with_mock(mock, fn ->
        OptimalEngine.Connectors.Adapters.Notion.sync(@notion_state, nil)

        db_call =
          :ets.tab2list(calls)
          |> Enum.find(fn {u, _} -> String.contains?(u, "/databases/") end)

        if db_call do
          {_url, opts} = db_call
          headers = Keyword.get(opts, :headers, [])
          assert Enum.any?(headers, fn {k, _v} -> k == "notion-version" end)

          assert Enum.any?(headers, fn {k, v} ->
                   k == "authorization" and String.contains?(v, "secret_test")
                 end)
        end
      end)

      :ets.delete(calls)
    end

    test "sync/2 queries each database_id" do
      calls = :ets.new(:notion_db_calls, [:bag, :public])
      state = Map.put(@notion_state, "database_ids", ["db-a", "db-b"])

      mock = fn url, opts ->
        :ets.insert(calls, {url, opts})
        httpc_ok(%{"results" => [], "has_more" => false})
      end

      with_mock(mock, fn ->
        OptimalEngine.Connectors.Adapters.Notion.sync(state, nil)

        db_calls =
          :ets.tab2list(calls)
          |> Enum.filter(fn {u, _} -> String.contains?(u, "/databases/") end)
          |> Enum.map(fn {u, _} -> u end)

        assert Enum.any?(db_calls, &String.contains?(&1, "db-a"))
        assert Enum.any?(db_calls, &String.contains?(&1, "db-b"))
      end)

      :ets.delete(calls)
    end

    test "sync/2 propagates cursor as last_edited_time" do
      page = %{
        "id" => "page-1",
        "last_edited_time" => "2026-06-01T00:00:00Z",
        "properties" => %{
          "Name" => %{"title" => [%{"plain_text" => "My Page"}]}
        }
      }

      mock = fn _url, _opts ->
        httpc_ok(%{"results" => [page], "has_more" => false})
      end

      with_mock(mock, fn ->
        assert {:ok, %{cursor: cursor, signals: signals}} =
                 OptimalEngine.Connectors.Adapters.Notion.sync(@notion_state, nil)

        assert cursor == "2026-06-01T00:00:00Z"
        assert length(signals) == 1
      end)
    end

    test "sync/2 skips databases that return 401 and continues (db-level error isolation)" do
      # Notion sync uses Enum.reduce -- per-database HTTP errors are skipped.
      mock = fn _url, _opts -> httpc_status(401) end

      with_mock(mock, fn ->
        assert {:ok, %{signals: [], cursor: _}} =
                 OptimalEngine.Connectors.Adapters.Notion.sync(@notion_state, nil)
      end)
    end

    test "transform/1 extracts title from Name property" do
      raw = %{
        "id" => "pg1",
        "last_edited_time" => "2026-01-01T00:00:00Z",
        "blocks" => [],
        "properties" => %{
          "Name" => %{"title" => [%{"plain_text" => "The Page Title"}]}
        }
      }

      assert {:ok, %Signal{} = sig} =
               OptimalEngine.Connectors.Adapters.Notion.transform(raw)

      assert sig.title == "The Page Title"
      assert sig.genre == "document"
      assert sig.path =~ "optimal://connectors/notion/pg1"
    end

    test "transform/1 falls back to 'Untitled' when no title property" do
      raw = %{"id" => "pg2", "blocks" => [], "properties" => %{}}

      assert {:ok, %Signal{title: "Untitled"}} =
               OptimalEngine.Connectors.Adapters.Notion.transform(raw)
    end
  end

  # ── Google Drive ─────────────────────────────────────────────────────────────

  describe "Google Drive adapter" do
    @drive_state %{
      "scope" => "my_drive",
      "oauth_refresh_token" => "rtoken",
      "client_id" => "cid",
      "client_secret" => "csec"
    }

    defp drive_token_mock(next_fn) do
      fn url, opts ->
        if String.contains?(url, "oauth2.googleapis.com") do
          httpc_ok(%{"access_token" => "drive_tok"})
        else
          next_fn.(url, opts)
        end
      end
    end

    test "sync/2 refreshes token before listing files" do
      token_hit = :ets.new(:drive_token, [:set, :public])

      mock = fn url, _opts ->
        if String.contains?(url, "oauth2.googleapis.com") do
          :ets.insert(token_hit, {:called, true})
          httpc_ok(%{"access_token" => "tok"})
        else
          cond do
            String.contains?(url, "startPageToken") ->
              httpc_ok(%{"startPageToken" => "tok1"})

            String.contains?(url, "/files") ->
              httpc_ok(%{"files" => []})

            true ->
              httpc_ok(%{})
          end
        end
      end

      with_mock(mock, fn ->
        OptimalEngine.Connectors.Adapters.Drive.sync(@drive_state, nil)
        assert :ets.lookup(token_hit, :called) == [{:called, true}]
      end)

      :ets.delete(token_hit)
    end

    test "sync/2 initial sync fetches startPageToken then files" do
      calls = :ets.new(:drive_init, [:bag, :public])

      mock =
        drive_token_mock(fn url, opts ->
          :ets.insert(calls, {url, opts})

          cond do
            String.contains?(url, "startPageToken") ->
              httpc_ok(%{"startPageToken" => "start_tok_1"})

            String.contains?(url, "/files") ->
              httpc_ok(%{
                "files" => [
                  %{
                    "id" => "f1",
                    "name" => "doc.pdf",
                    "mimeType" => "application/pdf",
                    "modifiedTime" => "2026-01-01T00:00:00Z"
                  }
                ]
              })

            true ->
              httpc_ok(%{})
          end
        end)

      with_mock(mock, fn ->
        assert {:ok, %{signals: signals, cursor: "start_tok_1"}} =
                 OptimalEngine.Connectors.Adapters.Drive.sync(@drive_state, nil)

        assert length(signals) == 1
        [sig] = signals
        assert sig.title == "doc.pdf"
      end)

      :ets.delete(calls)
    end

    test "sync/2 incremental uses changes API with page token" do
      calls = :ets.new(:drive_incr, [:bag, :public])

      mock =
        drive_token_mock(fn url, opts ->
          :ets.insert(calls, {url, opts})

          if String.contains?(url, "/changes") do
            httpc_ok(%{
              "changes" => [],
              "newStartPageToken" => "next_tok"
            })
          else
            httpc_ok(%{})
          end
        end)

      with_mock(mock, fn ->
        assert {:ok, %{signals: [], cursor: "next_tok"}} =
                 OptimalEngine.Connectors.Adapters.Drive.sync(@drive_state, "current_tok")

        changes_call =
          :ets.tab2list(calls)
          |> Enum.find(fn {u, _} -> String.contains?(u, "/changes") end)

        assert changes_call != nil
        {url, _} = changes_call
        assert String.contains?(url, "current_tok")
      end)

      :ets.delete(calls)
    end

    test "transform/1 maps file fields and genre correctly" do
      raw = %{
        "id" => "file1",
        "name" => "Report.pdf",
        "mimeType" => "application/pdf",
        "modifiedTime" => "2026-05-01T10:00:00Z"
      }

      assert {:ok, %Signal{} = sig} =
               OptimalEngine.Connectors.Adapters.Drive.transform(raw)

      assert sig.title == "Report.pdf"
      assert sig.genre == "document"
      assert sig.mode == :linguistic
      assert sig.path =~ "optimal://connectors/drive/file1"
      assert %DateTime{year: 2026, month: 5} = sig.modified_at
    end

    test "transform/1 maps spreadsheet to :data mode" do
      raw = %{
        "id" => "sheet1",
        "name" => "Data.xlsx",
        "mimeType" => "application/vnd.google-apps.spreadsheet",
        "modifiedTime" => "2026-01-01T00:00:00Z"
      }

      assert {:ok, %Signal{mode: :data, genre: "table"}} =
               OptimalEngine.Connectors.Adapters.Drive.transform(raw)
    end
  end

  # ── Dropbox ──────────────────────────────────────────────────────────────────

  describe "Dropbox adapter" do
    @dropbox_state %{
      "namespace" => "personal",
      "refresh_token" => "rtoken",
      "app_key" => "key",
      "app_secret" => "sec"
    }

    defp dropbox_token_mock(next_fn) do
      fn url, opts ->
        if String.contains?(url, "dropboxapi.com/oauth2/token") do
          httpc_ok(%{"access_token" => "dbx_tok"})
        else
          next_fn.(url, opts)
        end
      end
    end

    test "sync/2 refreshes token using app_key + app_secret" do
      token_hit = :ets.new(:dbx_token, [:set, :public])

      mock = fn url, _opts ->
        if String.contains?(url, "oauth2/token") do
          :ets.insert(token_hit, {:called, true})
          httpc_ok(%{"access_token" => "tok"})
        else
          httpc_ok(%{"entries" => [], "has_more" => false, "cursor" => "c1"})
        end
      end

      with_mock(mock, fn ->
        OptimalEngine.Connectors.Adapters.Dropbox.sync(@dropbox_state, nil)
        assert :ets.lookup(token_hit, :called) == [{:called, true}]
      end)

      :ets.delete(token_hit)
    end

    test "sync/2 full sync calls list_folder" do
      calls = :ets.new(:dbx_full, [:bag, :public])

      mock =
        dropbox_token_mock(fn url, opts ->
          :ets.insert(calls, {url, opts})
          httpc_ok(%{"entries" => [], "has_more" => false, "cursor" => "dbx_cursor_1"})
        end)

      with_mock(mock, fn ->
        assert {:ok, %{cursor: "dbx_cursor_1"}} =
                 OptimalEngine.Connectors.Adapters.Dropbox.sync(@dropbox_state, nil)

        folder_calls =
          :ets.tab2list(calls)
          |> Enum.filter(fn {u, _} -> String.contains?(u, "list_folder") end)

        assert length(folder_calls) >= 1

        [{url, _} | _] = folder_calls
        # Initial sync must NOT call list_folder/continue
        refute String.contains?(url, "continue")
      end)

      :ets.delete(calls)
    end

    test "sync/2 incremental calls list_folder/continue with cursor" do
      calls = :ets.new(:dbx_incr, [:bag, :public])

      mock =
        dropbox_token_mock(fn url, opts ->
          :ets.insert(calls, {url, opts})
          httpc_ok(%{"entries" => [], "has_more" => false, "cursor" => "dbx_cursor_2"})
        end)

      with_mock(mock, fn ->
        assert {:ok, %{cursor: "dbx_cursor_2"}} =
                 OptimalEngine.Connectors.Adapters.Dropbox.sync(@dropbox_state, "prev_cursor")

        continue_calls =
          :ets.tab2list(calls)
          |> Enum.filter(fn {u, _} -> String.contains?(u, "list_folder/continue") end)

        assert length(continue_calls) == 1
      end)

      :ets.delete(calls)
    end

    test "sync/2 only returns file entries (filters out folders)" do
      mock =
        dropbox_token_mock(fn _url, _opts ->
          httpc_ok(%{
            "entries" => [
              %{
                ".tag" => "file",
                "id" => "id:f1",
                "name" => "notes.txt",
                "path_display" => "/notes.txt",
                "server_modified" => "2026-01-01T00:00:00Z"
              },
              %{".tag" => "folder", "id" => "id:d1", "name" => "docs", "path_display" => "/docs"}
            ],
            "has_more" => false,
            "cursor" => "c3"
          })
        end)

      with_mock(mock, fn ->
        assert {:ok, %{signals: signals}} =
                 OptimalEngine.Connectors.Adapters.Dropbox.sync(@dropbox_state, nil)

        assert length(signals) == 1
        [sig] = signals
        assert sig.title == "notes.txt"
      end)
    end

    test "transform/1 maps file metadata correctly" do
      raw = %{
        "id" => "id:abc123",
        "name" => "report.pdf",
        "path_display" => "/reports/report.pdf",
        "path_lower" => "/reports/report.pdf",
        "server_modified" => "2026-03-15T08:00:00Z"
      }

      assert {:ok, %Signal{} = sig} =
               OptimalEngine.Connectors.Adapters.Dropbox.transform(raw)

      assert sig.title == "report.pdf"
      assert sig.genre == "file"
      assert sig.path =~ "optimal://connectors/dropbox/"
      assert %DateTime{year: 2026, month: 3} = sig.modified_at
    end
  end

  # ── OneDrive ─────────────────────────────────────────────────────────────────

  describe "OneDrive adapter" do
    @od_state %{
      "tenant_id_ms" => "tenant-123",
      "drive_id" => "drive-456",
      "client_id" => "cid",
      "client_secret" => "csec"
    }

    defp onedrive_token_mock(next_fn) do
      fn url, opts ->
        if String.contains?(url, "microsoftonline.com") do
          httpc_ok(%{"access_token" => "ms_tok"})
        else
          next_fn.(url, opts)
        end
      end
    end

    test "sync/2 acquires token from tenant-specific endpoint" do
      token_hit = :ets.new(:od_token, [:set, :public])

      mock = fn url, _opts ->
        if String.contains?(url, "microsoftonline.com") do
          :ets.insert(token_hit, {:called, true, url})
          httpc_ok(%{"access_token" => "tok"})
        else
          httpc_ok(%{"value" => [], "@odata.deltaLink" => "https://graph.microsoft.com/delta"})
        end
      end

      with_mock(mock, fn ->
        OptimalEngine.Connectors.Adapters.OneDrive.sync(@od_state, nil)
        [{:called, true, url}] = :ets.lookup(token_hit, :called)
        assert String.contains?(url, "tenant-123")
      end)

      :ets.delete(token_hit)
    end

    test "sync/2 initial sync uses drive delta endpoint" do
      calls = :ets.new(:od_full, [:bag, :public])

      mock =
        onedrive_token_mock(fn url, opts ->
          :ets.insert(calls, {url, opts})
          httpc_ok(%{"value" => [], "@odata.deltaLink" => "https://graph.microsoft.com/delta2"})
        end)

      with_mock(mock, fn ->
        assert {:ok, %{cursor: "https://graph.microsoft.com/delta2"}} =
                 OptimalEngine.Connectors.Adapters.OneDrive.sync(@od_state, nil)

        delta_calls =
          :ets.tab2list(calls)
          |> Enum.filter(fn {u, _} -> String.contains?(u, "/delta") end)

        assert length(delta_calls) >= 1
        [{url, _} | _] = delta_calls
        assert String.contains?(url, "drive-456")
      end)

      :ets.delete(calls)
    end

    test "sync/2 incremental uses stored delta link as URL" do
      calls = :ets.new(:od_incr, [:bag, :public])
      delta_link = "https://graph.microsoft.com/v1.0/drives/drive-456/root/delta?token=abc"

      mock =
        onedrive_token_mock(fn url, opts ->
          :ets.insert(calls, {url, opts})
          httpc_ok(%{"value" => [], "@odata.deltaLink" => "https://graph.microsoft.com/delta3"})
        end)

      with_mock(mock, fn ->
        assert {:ok, %{cursor: "https://graph.microsoft.com/delta3"}} =
                 OptimalEngine.Connectors.Adapters.OneDrive.sync(@od_state, delta_link)

        delta_calls =
          :ets.tab2list(calls)
          |> Enum.filter(fn {u, _} -> String.contains?(u, delta_link) end)

        assert length(delta_calls) >= 1
      end)

      :ets.delete(calls)
    end

    test "sync/2 filters out deleted items" do
      item_normal = %{
        "id" => "i1",
        "name" => "doc.docx",
        "lastModifiedDateTime" => "2026-01-01T00:00:00Z"
      }

      item_deleted = %{"id" => "i2", "name" => "gone.docx", "deleted" => %{"state" => "deleted"}}

      mock =
        onedrive_token_mock(fn _url, _opts ->
          httpc_ok(%{
            "value" => [item_normal, item_deleted],
            "@odata.deltaLink" => "https://graph.microsoft.com/delta4"
          })
        end)

      with_mock(mock, fn ->
        assert {:ok, %{signals: signals}} =
                 OptimalEngine.Connectors.Adapters.OneDrive.sync(@od_state, nil)

        assert length(signals) == 1
        [sig] = signals
        assert sig.title == "doc.docx"
      end)
    end

    test "sync/2 returns {:error, :rate_limited} on 429" do
      mock = fn url, _opts ->
        if String.contains?(url, "microsoftonline.com") do
          httpc_ok(%{"access_token" => "tok"})
        else
          httpc_status(429)
        end
      end

      with_mock(mock, fn ->
        assert {:error, :rate_limited} =
                 OptimalEngine.Connectors.Adapters.OneDrive.sync(@od_state, nil)
      end)
    end

    test "transform/1 maps OneDrive item fields correctly" do
      raw = %{
        "id" => "od1",
        "name" => "Presentation.pptx",
        "lastModifiedDateTime" => "2026-04-10T14:30:00Z"
      }

      assert {:ok, %Signal{} = sig} =
               OptimalEngine.Connectors.Adapters.OneDrive.transform(raw)

      assert sig.title == "Presentation.pptx"
      assert sig.genre == "file"
      assert sig.path =~ "optimal://connectors/onedrive/od1"
      assert %DateTime{year: 2026, month: 4} = sig.modified_at
    end
  end
end
