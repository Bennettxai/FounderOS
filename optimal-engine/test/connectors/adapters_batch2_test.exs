defmodule OptimalEngine.Connectors.AdaptersBatch2Test do
  @moduledoc """
  Unit tests for batch-2 adapters (hubspot, salesforce, jira, linear, confluence, teams, zoom)
  with a mocked HTTP layer.

  Each adapter group verifies:
    * request shape (URL, method, auth header) via the mock fn
    * transform/1 field mapping: id, title, content, genre, modified_at
    * sync/2 cursor propagation
    * sync/2 returns {:error, :auth_expired} on HTTP 401
    * sync/2 returns {:error, :rate_limited} on HTTP 429
  """

  use ExUnit.Case, async: false

  alias OptimalEngine.Signal

  # ── mock helpers ─────────────────────────────────────────────────────────────

  defp with_mock(fun, test_fn) do
    Application.put_env(:optimal_engine, :http_mock, fun)

    try do
      test_fn.()
    after
      Application.delete_env(:optimal_engine, :http_mock)
    end
  end

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

  defp base_state(extra \\ %{}) do
    Map.merge(%{"_state_version" => 1}, extra)
  end

  # ═══════════════════════════════════════════════════════════════════
  # HubSpot
  # ═══════════════════════════════════════════════════════════════════

  describe "HubSpot adapter" do
    alias OptimalEngine.Connectors.Adapters.HubSpot

    defp hubspot_state do
      base_state(%{
        "portal_id" => "123",
        "objects" => ["contacts"],
        "access_token" => "hs_tok"
      })
    end

    test "kind/0, display_name/0, auth_scheme/0 are correct" do
      assert HubSpot.kind() == :hubspot
      assert HubSpot.display_name() == "HubSpot"
      assert HubSpot.auth_scheme() == :oauth2
    end

    test "sync/2 sends Bearer token in Authorization header" do
      me = self()

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})
          httpc_ok(%{"results" => [], "paging" => %{}})
        end,
        fn ->
          HubSpot.sync(hubspot_state(), nil)
          assert_received {:request, url, opts}
          headers = Keyword.get(opts, :headers, [])

          assert Enum.any?(headers, fn {k, v} ->
                   k == "authorization" and String.starts_with?(v, "Bearer ")
                 end)

          assert String.contains?(url, "api.hubapi.com")
        end
      )
    end

    test "sync/2 uses cursor as after parameter" do
      me = self()

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})
          httpc_ok(%{"results" => [], "paging" => %{}})
        end,
        fn ->
          HubSpot.sync(hubspot_state(), "contacts:cursor123")
          assert_received {:request, url, _opts}
          assert String.contains?(url, "after=cursor123")
        end
      )
    end

    test "sync/2 returns {:error, :auth_expired} on 401" do
      with_mock(
        fn _url, _opts -> httpc_status(401) end,
        fn ->
          assert {:error, :auth_expired} = HubSpot.sync(hubspot_state(), nil)
        end
      )
    end

    test "sync/2 returns {:error, :rate_limited} on 429" do
      with_mock(
        fn _url, _opts -> httpc_status(429) end,
        fn ->
          assert {:error, :rate_limited} = HubSpot.sync(hubspot_state(), nil)
        end
      )
    end

    test "sync/2 returns cursor from paging.next.after when present" do
      with_mock(
        fn _url, _opts ->
          httpc_ok(%{
            "results" => [
              %{
                "id" => "1",
                "properties" => %{
                  "dealname" => "Deal A",
                  "hs_lastmodifieddate" => "2024-01-01T00:00:00Z"
                }
              }
            ],
            "paging" => %{"next" => %{"after" => "page2cursor"}}
          })
        end,
        fn ->
          {:ok, %{cursor: cursor}} = HubSpot.sync(hubspot_state(), nil)
          assert cursor == "contacts:page2cursor"
        end
      )
    end

    test "sync/2 returns nil cursor when no next page" do
      with_mock(
        fn _url, _opts ->
          httpc_ok(%{"results" => [], "paging" => %{}})
        end,
        fn ->
          {:ok, %{cursor: cursor}} = HubSpot.sync(hubspot_state(), nil)
          assert is_nil(cursor)
        end
      )
    end

    test "transform/1 maps contact fields correctly" do
      raw = %{
        "id" => "42",
        "properties" => %{
          "firstname" => "Alice",
          "description" => "A contact",
          "hs_lastmodifieddate" => "2024-03-15T10:00:00Z"
        }
      }

      {:ok, signal} = HubSpot.transform(raw)
      assert %Signal{} = signal
      assert signal.title == "Alice"
      assert signal.content == "A contact"
      assert signal.genre == "crm"
      assert signal.id =~ "hubspot"
    end

    test "transform/1 falls back to ext_id when no title property" do
      raw = %{"id" => "99", "properties" => %{}}
      {:ok, signal} = HubSpot.transform(raw)
      assert signal.title == "99"
    end
  end

  # ═══════════════════════════════════════════════════════════════════
  # Salesforce
  # ═══════════════════════════════════════════════════════════════════

  describe "Salesforce adapter" do
    alias OptimalEngine.Connectors.Adapters.Salesforce

    defp sf_state do
      base_state(%{
        "instance_url" => "https://myorg.salesforce.com",
        "objects" => ["Account"],
        "client_id" => "sf_client",
        "client_secret" => "sf_secret",
        "refresh_token" => "sf_refresh"
      })
    end

    test "kind/0, display_name/0, auth_scheme/0 are correct" do
      assert Salesforce.kind() == :salesforce
      assert Salesforce.display_name() == "Salesforce"
      assert Salesforce.auth_scheme() == :oauth2
    end

    test "sync/2 exchanges refresh token then queries SOQL" do
      me = self()
      call_count = :counters.new(1, [])

      with_mock(
        fn url, opts ->
          :counters.add(call_count, 1, 1)
          send(me, {:request, url, opts})

          cond do
            String.contains?(url, "oauth2/token") ->
              httpc_ok(%{"access_token" => "sf_access_tok"})

            String.contains?(url, "query") ->
              httpc_ok(%{"records" => [], "totalSize" => 0})

            true ->
              httpc_ok(%{})
          end
        end,
        fn ->
          Salesforce.sync(sf_state(), nil)
          # Should have made at least 2 requests: token + query
          assert :counters.get(call_count, 1) >= 2

          messages = collect_messages(:request)
          urls = Enum.map(messages, fn {:request, url, _} -> url end)
          assert Enum.any?(urls, &String.contains?(&1, "oauth2/token"))
          assert Enum.any?(urls, &String.contains?(&1, "query"))
        end
      )
    end

    test "sync/2 sends Bearer token in Authorization header for SOQL" do
      me = self()

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})

          if String.contains?(url, "oauth2/token") do
            httpc_ok(%{"access_token" => "sf_tok"})
          else
            httpc_ok(%{"records" => []})
          end
        end,
        fn ->
          Salesforce.sync(sf_state(), nil)

          messages = collect_messages(:request)

          query_req =
            Enum.find(messages, fn {:request, url, _} -> String.contains?(url, "query") end)

          assert query_req != nil
          {:request, _url, opts} = query_req
          headers = Keyword.get(opts, :headers, [])
          assert Enum.any?(headers, fn {k, v} -> k == "authorization" and v == "Bearer sf_tok" end)
        end
      )
    end

    test "sync/2 returns {:error, :auth_expired} when token exchange returns 401" do
      with_mock(
        fn _url, _opts -> httpc_status(401) end,
        fn ->
          assert {:error, :auth_expired} = Salesforce.sync(sf_state(), nil)
        end
      )
    end

    test "sync/2 returns {:error, :rate_limited} on 429 from SOQL endpoint" do
      with_mock(
        fn url, _opts ->
          if String.contains?(url, "oauth2/token") do
            httpc_ok(%{"access_token" => "sf_tok"})
          else
            httpc_status(429)
          end
        end,
        fn ->
          assert {:error, :rate_limited} = Salesforce.sync(sf_state(), nil)
        end
      )
    end

    test "sync/2 uses cursor as timestamp filter" do
      me = self()

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})

          if String.contains?(url, "oauth2/token") do
            httpc_ok(%{"access_token" => "sf_tok"})
          else
            httpc_ok(%{"records" => []})
          end
        end,
        fn ->
          cursor_ts = "2024-01-15T00:00:00Z"
          Salesforce.sync(sf_state(), cursor_ts)

          messages = collect_messages(:request)

          query_req =
            Enum.find(messages, fn {:request, url, _} -> String.contains?(url, "query") end)

          assert query_req != nil
          {:request, url, _} = query_req

          assert String.contains?(url, URI.encode_www_form(cursor_ts) |> String.replace("+", "%20")) or
                   String.contains?(URI.decode(url), cursor_ts)
        end
      )
    end

    test "transform/1 maps Account fields correctly" do
      raw = %{
        "Id" => "001xx",
        "Name" => "Acme Corp",
        "Description" => "Big company",
        "LastModifiedDate" => "2024-04-01T08:00:00Z",
        "attributes" => %{"type" => "Account"}
      }

      {:ok, signal} = Salesforce.transform(raw)
      assert signal.title == "Acme Corp"
      assert signal.content == "Big company"
      assert signal.genre == "account"
      assert signal.id =~ "salesforce"
    end

    test "transform/1 maps Case with Subject correctly" do
      raw = %{
        "Id" => "500xx",
        "Subject" => "Login broken",
        "Description" => "Can't log in",
        "LastModifiedDate" => "2024-04-02T08:00:00Z",
        "attributes" => %{"type" => "Case"}
      }

      {:ok, signal} = Salesforce.transform(raw)
      assert signal.title == "Login broken"
      assert signal.genre == "case"
    end
  end

  # ═══════════════════════════════════════════════════════════════════
  # Jira
  # ═══════════════════════════════════════════════════════════════════

  describe "Jira adapter" do
    alias OptimalEngine.Connectors.Adapters.Jira

    defp jira_state do
      base_state(%{
        "site_url" => "https://myorg.atlassian.net",
        "projects" => ["PROJ"],
        "email" => "user@example.com",
        "api_token" => "jira_tok"
      })
    end

    test "kind/0, display_name/0, auth_scheme/0 are correct" do
      assert Jira.kind() == :jira
      assert Jira.display_name() == "Jira"
      assert Jira.auth_scheme() == :basic
    end

    test "sync/2 sends Basic auth header" do
      me = self()

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})
          httpc_ok(%{"issues" => [], "total" => 0})
        end,
        fn ->
          Jira.sync(jira_state(), nil)
          assert_received {:request, url, opts}
          headers = Keyword.get(opts, :headers, [])

          assert Enum.any?(headers, fn {k, v} ->
                   k == "authorization" and String.starts_with?(v, "Basic ")
                 end)

          assert String.contains?(url, "atlassian.net")
        end
      )
    end

    test "sync/2 encodes JQL with project and since timestamp" do
      me = self()

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})
          httpc_ok(%{"issues" => []})
        end,
        fn ->
          Jira.sync(jira_state(), nil)
          assert_received {:request, url, _opts}
          assert String.contains?(url, "jql")
          assert String.contains?(URI.decode(url), "PROJ")
        end
      )
    end

    test "sync/2 returns {:error, :auth_expired} on 401" do
      with_mock(
        fn _url, _opts -> httpc_status(401) end,
        fn ->
          assert {:error, :auth_expired} = Jira.sync(jira_state(), nil)
        end
      )
    end

    test "sync/2 returns {:error, :rate_limited} on 429" do
      with_mock(
        fn _url, _opts -> httpc_status(429) end,
        fn ->
          assert {:error, :rate_limited} = Jira.sync(jira_state(), nil)
        end
      )
    end

    test "sync/2 advances cursor to latest updated_at" do
      with_mock(
        fn _url, _opts ->
          httpc_ok(%{
            "issues" => [
              %{
                "key" => "PROJ-1",
                "fields" => %{
                  "summary" => "Fix bug",
                  "description" => "",
                  "updated" => "2024-05-10T12:00:00.000Z",
                  "assignee" => nil,
                  "reporter" => nil
                }
              },
              %{
                "key" => "PROJ-2",
                "fields" => %{
                  "summary" => "Add feat",
                  "description" => "",
                  "updated" => "2024-05-12T08:00:00.000Z",
                  "assignee" => nil,
                  "reporter" => nil
                }
              }
            ]
          })
        end,
        fn ->
          {:ok, %{cursor: cursor, signals: signals}} = Jira.sync(jira_state(), nil)
          assert length(signals) == 2
          assert cursor == "2024-05-12T08:00:00.000Z"
        end
      )
    end

    test "transform/1 maps issue fields correctly" do
      raw = %{
        "key" => "PROJ-42",
        "id" => "10042",
        "fields" => %{
          "summary" => "Fix the bug",
          "description" => "<p>Details here</p>",
          "updated" => "2024-05-01T09:00:00.000Z",
          "assignee" => %{"displayName" => "Alice"},
          "reporter" => %{"displayName" => "Bob"}
        }
      }

      {:ok, signal} = Jira.transform(raw)
      assert signal.title =~ "PROJ-42"
      assert signal.title =~ "Fix the bug"
      assert signal.genre == "ticket"
      assert signal.id =~ "jira"
      assert "Alice" in signal.entities
      assert "Bob" in signal.entities
    end

    test "transform/1 handles nil assignee/reporter gracefully" do
      raw = %{
        "key" => "PROJ-1",
        "fields" => %{
          "summary" => "Simple issue",
          "description" => "",
          "updated" => "2024-01-01T00:00:00.000Z",
          "assignee" => nil,
          "reporter" => nil
        }
      }

      {:ok, signal} = Jira.transform(raw)
      assert signal.entities == []
    end
  end

  # ═══════════════════════════════════════════════════════════════════
  # Linear
  # ═══════════════════════════════════════════════════════════════════

  describe "Linear adapter" do
    alias OptimalEngine.Connectors.Adapters.Linear

    defp linear_state do
      base_state(%{
        "team_ids" => ["team-uuid-1"],
        "api_key" => "lin_api_abc123"
      })
    end

    test "kind/0, display_name/0, auth_scheme/0 are correct" do
      assert Linear.kind() == :linear
      assert Linear.display_name() == "Linear"
      assert Linear.auth_scheme() == :token
    end

    test "sync/2 sends api_key in Authorization header" do
      me = self()

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})

          httpc_ok(%{
            "data" => %{
              "issues" => %{
                "nodes" => [],
                "pageInfo" => %{"hasNextPage" => false, "endCursor" => nil}
              }
            }
          })
        end,
        fn ->
          Linear.sync(linear_state(), nil)
          assert_received {:request, url, opts}
          headers = Keyword.get(opts, :headers, [])
          assert Enum.any?(headers, fn {k, v} -> k == "authorization" and v == "lin_api_abc123" end)
          assert String.contains?(url, "linear.app")
        end
      )
    end

    test "sync/2 passes cursor as after variable in GraphQL" do
      me = self()

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})

          httpc_ok(%{
            "data" => %{
              "issues" => %{
                "nodes" => [],
                "pageInfo" => %{"hasNextPage" => false, "endCursor" => nil}
              }
            }
          })
        end,
        fn ->
          Linear.sync(linear_state(), "gql_cursor_xyz")
          assert_received {:request, _url, opts}
          body = Keyword.get(opts, :body, %{})
          vars = Map.get(body, "variables", %{})
          assert vars["after"] == "gql_cursor_xyz"
        end
      )
    end

    test "sync/2 returns next cursor when hasNextPage is true" do
      with_mock(
        fn _url, _opts ->
          httpc_ok(%{
            "data" => %{
              "issues" => %{
                "nodes" => [
                  %{
                    "id" => "issue-1",
                    "title" => "T1",
                    "description" => "",
                    "updatedAt" => "2024-01-01T00:00:00Z",
                    "assignee" => nil,
                    "creator" => nil,
                    "state" => nil
                  }
                ],
                "pageInfo" => %{"hasNextPage" => true, "endCursor" => "next_page_cursor"}
              }
            }
          })
        end,
        fn ->
          {:ok, %{cursor: cursor}} = Linear.sync(linear_state(), nil)
          assert cursor == "next_page_cursor"
        end
      )
    end

    test "sync/2 returns nil cursor when no next page" do
      with_mock(
        fn _url, _opts ->
          httpc_ok(%{
            "data" => %{
              "issues" => %{
                "nodes" => [],
                "pageInfo" => %{"hasNextPage" => false, "endCursor" => nil}
              }
            }
          })
        end,
        fn ->
          {:ok, %{cursor: cursor}} = Linear.sync(linear_state(), nil)
          assert is_nil(cursor)
        end
      )
    end

    test "sync/2 returns {:error, :auth_expired} on HTTP 401" do
      with_mock(
        fn _url, _opts -> httpc_status(401) end,
        fn ->
          assert {:error, :auth_expired} = Linear.sync(linear_state(), nil)
        end
      )
    end

    test "sync/2 returns {:error, :rate_limited} on HTTP 429" do
      with_mock(
        fn _url, _opts -> httpc_status(429) end,
        fn ->
          assert {:error, :rate_limited} = Linear.sync(linear_state(), nil)
        end
      )
    end

    test "transform/1 maps issue fields correctly" do
      raw = %{
        "id" => "uuid-issue-1",
        "title" => "Fix login flow",
        "description" => "Users can't log in on mobile",
        "updatedAt" => "2024-06-01T10:00:00Z",
        "assignee" => %{"name" => "Alice"},
        "creator" => %{"name" => "Bob"},
        "state" => %{"name" => "In Progress"}
      }

      {:ok, signal} = Linear.transform(raw)
      assert signal.title == "Fix login flow"
      assert signal.content == "Users can't log in on mobile"
      assert signal.genre == "ticket"
      assert signal.id =~ "linear"
      assert "Alice" in signal.entities
      assert "Bob" in signal.entities
    end

    test "transform/1 handles nil assignee/creator gracefully" do
      raw = %{
        "id" => "uuid-2",
        "title" => "Issue",
        "description" => nil,
        "updatedAt" => "2024-01-01T00:00:00Z",
        "assignee" => nil,
        "creator" => nil
      }

      {:ok, signal} = Linear.transform(raw)
      assert signal.entities == []
    end
  end

  # ═══════════════════════════════════════════════════════════════════
  # Confluence
  # ═══════════════════════════════════════════════════════════════════

  describe "Confluence adapter" do
    alias OptimalEngine.Connectors.Adapters.Confluence

    defp confluence_state do
      base_state(%{
        "site_url" => "https://myorg.atlassian.net",
        "spaces" => ["TEAM"],
        "email" => "user@example.com",
        "api_token" => "conf_tok"
      })
    end

    test "kind/0, display_name/0, auth_scheme/0 are correct" do
      assert Confluence.kind() == :confluence
      assert Confluence.display_name() == "Confluence"
      assert Confluence.auth_scheme() == :basic
    end

    test "sync/2 sends Basic auth header" do
      me = self()

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})
          httpc_ok(%{"results" => [], "size" => 0})
        end,
        fn ->
          Confluence.sync(confluence_state(), nil)
          assert_received {:request, url, opts}
          headers = Keyword.get(opts, :headers, [])

          assert Enum.any?(headers, fn {k, v} ->
                   k == "authorization" and String.starts_with?(v, "Basic ")
                 end)

          assert String.contains?(url, "atlassian.net")
        end
      )
    end

    test "sync/2 fetches all spaces when spaces is empty" do
      me = self()
      call_count = :counters.new(1, [])

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})
          :counters.add(call_count, 1, 1)

          if String.contains?(url, "/space") and not String.contains?(url, "content") do
            httpc_ok(%{"results" => [%{"key" => "AUTO"}], "size" => 1})
          else
            httpc_ok(%{"results" => [], "size" => 0})
          end
        end,
        fn ->
          no_spaces_state = Map.put(confluence_state(), "spaces", [])
          Confluence.sync(no_spaces_state, nil)
          assert :counters.get(call_count, 1) >= 2
        end
      )
    end

    test "sync/2 returns {:error, :auth_expired} on 401" do
      with_mock(
        fn _url, _opts -> httpc_status(401) end,
        fn ->
          assert {:error, :auth_expired} = Confluence.sync(confluence_state(), nil)
        end
      )
    end

    test "sync/2 returns {:error, :rate_limited} on 429" do
      with_mock(
        fn _url, _opts -> httpc_status(429) end,
        fn ->
          assert {:error, :rate_limited} = Confluence.sync(confluence_state(), nil)
        end
      )
    end

    test "sync/2 propagates cursor from latest page version" do
      with_mock(
        fn _url, _opts ->
          httpc_ok(%{
            "results" => [
              %{
                "id" => "p1",
                "title" => "Page 1",
                "body" => %{"storage" => %{"value" => ""}},
                "version" => %{"when" => "2024-05-01T10:00:00Z"}
              },
              %{
                "id" => "p2",
                "title" => "Page 2",
                "body" => %{"storage" => %{"value" => ""}},
                "version" => %{"when" => "2024-05-03T08:00:00Z"}
              }
            ]
          })
        end,
        fn ->
          {:ok, %{cursor: cursor, signals: signals}} = Confluence.sync(confluence_state(), nil)
          assert length(signals) == 2
          assert cursor == "2024-05-03T08:00:00Z"
        end
      )
    end

    test "transform/1 maps page fields correctly" do
      raw = %{
        "id" => "page-123",
        "title" => "Architecture Decision",
        "body" => %{"storage" => %{"value" => "<p>We decided to use Elixir.</p>"}},
        "version" => %{"when" => "2024-03-10T14:00:00Z"}
      }

      {:ok, signal} = Confluence.transform(raw)
      assert signal.title == "Architecture Decision"
      assert signal.content =~ "Elixir"
      assert signal.genre == "document"
      assert signal.id =~ "confluence"
    end

    test "transform/1 strips HTML from body content" do
      raw = %{
        "id" => "page-999",
        "title" => "Notes",
        "body" => %{
          "storage" => %{"value" => "<h1>Header</h1><p>Paragraph <strong>text</strong>.</p>"}
        },
        "version" => %{"when" => nil}
      }

      {:ok, signal} = Confluence.transform(raw)
      refute String.contains?(signal.content, "<h1>")

      assert String.contains?(signal.content, "Header") or
               String.contains?(signal.content, "Paragraph")
    end
  end

  # ═══════════════════════════════════════════════════════════════════
  # Microsoft Teams
  # ═══════════════════════════════════════════════════════════════════

  describe "Teams adapter" do
    alias OptimalEngine.Connectors.Adapters.Teams

    defp teams_state do
      base_state(%{
        "tenant_id_ms" => "tenant-abc",
        "team_ids" => ["team-id-1"],
        "client_id" => "ms_client",
        "client_secret" => "ms_secret"
      })
    end

    test "kind/0, display_name/0, auth_scheme/0 are correct" do
      assert Teams.kind() == :teams
      assert Teams.display_name() == "Microsoft Teams"
      assert Teams.auth_scheme() == :oauth2
    end

    test "sync/2 acquires token from tenant-specific endpoint then fetches channels" do
      me = self()

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})

          cond do
            String.contains?(url, "oauth2/v2.0/token") ->
              httpc_ok(%{"access_token" => "ms_tok"})

            String.contains?(url, "/channels") and not String.contains?(url, "messages") ->
              httpc_ok(%{"value" => [%{"id" => "channel-1"}]})

            String.contains?(url, "messages") ->
              httpc_ok(%{"value" => []})

            true ->
              httpc_ok(%{"value" => []})
          end
        end,
        fn ->
          Teams.sync(teams_state(), nil)

          messages = collect_messages(:request)
          urls = Enum.map(messages, fn {:request, url, _} -> url end)
          assert Enum.any?(urls, &String.contains?(&1, "tenant-abc"))
          assert Enum.any?(urls, &String.contains?(&1, "oauth2"))
        end
      )
    end

    test "sync/2 sends Bearer token in Authorization header for Graph API calls" do
      me = self()

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})

          if String.contains?(url, "oauth2/v2.0/token") do
            httpc_ok(%{"access_token" => "ms_bearer_tok"})
          else
            httpc_ok(%{"value" => []})
          end
        end,
        fn ->
          Teams.sync(teams_state(), nil)

          messages = collect_messages(:request)

          graph_req =
            Enum.find(messages, fn {:request, url, _} ->
              String.contains?(url, "graph.microsoft.com")
            end)

          assert graph_req != nil
          {:request, _url, opts} = graph_req
          headers = Keyword.get(opts, :headers, [])

          assert Enum.any?(headers, fn {k, v} ->
                   k == "authorization" and v == "Bearer ms_bearer_tok"
                 end)
        end
      )
    end

    test "sync/2 returns {:error, :auth_expired} when token exchange returns 401" do
      with_mock(
        fn _url, _opts -> httpc_status(401) end,
        fn ->
          assert {:error, :auth_expired} = Teams.sync(teams_state(), nil)
        end
      )
    end

    test "sync/2 uses cursor as since timestamp filter in messages URL" do
      me = self()

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})

          cond do
            String.contains?(url, "oauth2") ->
              httpc_ok(%{"access_token" => "ms_tok"})

            String.contains?(url, "/channels") and not String.contains?(url, "messages") ->
              httpc_ok(%{"value" => [%{"id" => "ch-1"}]})

            String.contains?(url, "messages") ->
              httpc_ok(%{"value" => []})

            true ->
              httpc_ok(%{"value" => []})
          end
        end,
        fn ->
          cursor_ts = "2024-06-01T00:00:00Z"
          Teams.sync(teams_state(), cursor_ts)

          messages = collect_messages(:request)

          msg_req =
            Enum.find(messages, fn {:request, url, _} -> String.contains?(url, "messages") end)

          assert msg_req != nil
          {:request, url, _} = msg_req
          assert String.contains?(url, "2024-06-01")
        end
      )
    end

    test "transform/1 maps message fields correctly" do
      raw = %{
        "id" => "msg-abc",
        "body" => %{"content" => "<p>Hello team!</p>"},
        "lastModifiedDateTime" => "2024-04-15T09:30:00Z",
        "from" => %{"user" => %{"displayName" => "Roberto"}}
      }

      {:ok, signal} = Teams.transform(raw)
      assert signal.genre == "message"
      assert signal.id =~ "teams"
      assert "Roberto" in signal.entities
    end

    test "transform/1 strips HTML from body content" do
      raw = %{
        "id" => "msg-2",
        "body" => %{"content" => "<b>Important</b> update"},
        "lastModifiedDateTime" => "2024-01-01T00:00:00Z",
        "from" => %{"user" => %{"displayName" => "Alice"}}
      }

      {:ok, signal} = Teams.transform(raw)
      refute String.contains?(signal.content, "<b>")
    end
  end

  # ═══════════════════════════════════════════════════════════════════
  # Zoom
  # ═══════════════════════════════════════════════════════════════════

  describe "Zoom adapter" do
    alias OptimalEngine.Connectors.Adapters.Zoom

    defp zoom_state do
      base_state(%{
        "account_id" => "zoom_acct",
        "client_id" => "zoom_client",
        "client_secret" => "zoom_secret"
      })
    end

    test "kind/0, display_name/0, auth_scheme/0 are correct" do
      assert Zoom.kind() == :zoom
      assert Zoom.display_name() == "Zoom"
      assert Zoom.auth_scheme() == :oauth2
    end

    test "sync/2 acquires token via server-to-server OAuth" do
      me = self()

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})

          if String.contains?(url, "zoom.us/oauth/token") do
            httpc_ok(%{"access_token" => "zoom_tok"})
          else
            httpc_ok(%{"meetings" => []})
          end
        end,
        fn ->
          Zoom.sync(zoom_state(), nil)

          messages = collect_messages(:request)

          token_req =
            Enum.find(messages, fn {:request, url, _} -> String.contains?(url, "oauth/token") end)

          assert token_req != nil
          {:request, url, opts} = token_req
          assert String.contains?(url, "zoom_acct")
          headers = Keyword.get(opts, :headers, [])

          assert Enum.any?(headers, fn {k, v} ->
                   k == "authorization" and String.starts_with?(v, "Basic ")
                 end)
        end
      )
    end

    test "sync/2 sends Bearer token in Authorization header for recordings API" do
      me = self()

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})

          if String.contains?(url, "oauth/token") do
            httpc_ok(%{"access_token" => "zoom_bearer"})
          else
            httpc_ok(%{"meetings" => []})
          end
        end,
        fn ->
          Zoom.sync(zoom_state(), nil)

          messages = collect_messages(:request)

          rec_req =
            Enum.find(messages, fn {:request, url, _} -> String.contains?(url, "recordings") end)

          assert rec_req != nil
          {:request, _url, opts} = rec_req
          headers = Keyword.get(opts, :headers, [])

          assert Enum.any?(headers, fn {k, v} ->
                   k == "authorization" and v == "Bearer zoom_bearer"
                 end)
        end
      )
    end

    test "sync/2 uses cursor as from date" do
      me = self()

      with_mock(
        fn url, opts ->
          send(me, {:request, url, opts})

          if String.contains?(url, "oauth/token") do
            httpc_ok(%{"access_token" => "zoom_tok"})
          else
            httpc_ok(%{"meetings" => []})
          end
        end,
        fn ->
          Zoom.sync(zoom_state(), "2024-03-01")

          messages = collect_messages(:request)

          rec_req =
            Enum.find(messages, fn {:request, url, _} -> String.contains?(url, "recordings") end)

          assert rec_req != nil
          {:request, url, _} = rec_req
          assert String.contains?(url, "from=2024-03-01")
        end
      )
    end

    test "sync/2 returns {:error, :auth_expired} when token exchange returns 401" do
      with_mock(
        fn _url, _opts -> httpc_status(401) end,
        fn ->
          assert {:error, :auth_expired} = Zoom.sync(zoom_state(), nil)
        end
      )
    end

    test "sync/2 returns {:error, :rate_limited} on 429 from recordings endpoint" do
      with_mock(
        fn url, _opts ->
          if String.contains?(url, "oauth/token") do
            httpc_ok(%{"access_token" => "zoom_tok"})
          else
            httpc_status(429)
          end
        end,
        fn ->
          assert {:error, :rate_limited} = Zoom.sync(zoom_state(), nil)
        end
      )
    end

    test "sync/2 advances cursor to to_date when no next_page_token" do
      with_mock(
        fn url, _opts ->
          if String.contains?(url, "oauth/token") do
            httpc_ok(%{"access_token" => "zoom_tok"})
          else
            httpc_ok(%{"meetings" => [], "next_page_token" => ""})
          end
        end,
        fn ->
          {:ok, %{cursor: cursor}} = Zoom.sync(zoom_state(), "2024-03-01")
          # Cursor should advance (to_date is from + 30 days)
          assert cursor != nil
          assert cursor > "2024-03-01"
        end
      )
    end

    test "transform/1 maps meeting fields correctly" do
      raw = %{
        "uuid" => "meet-uuid-1",
        "topic" => "Weekly sync",
        "transcript" => "We discussed Q2 goals.",
        "start_time" => "2024-04-10T15:00:00Z"
      }

      {:ok, signal} = Zoom.transform(raw)
      assert signal.title == "Weekly sync"
      assert signal.content == "We discussed Q2 goals."
      assert signal.genre == "transcript"
      assert signal.id =~ "zoom"
    end

    test "transform/1 falls back to 'Zoom meeting' when topic absent" do
      raw = %{
        "id" => "12345",
        "start_time" => "2024-01-01T00:00:00Z"
      }

      {:ok, signal} = Zoom.transform(raw)
      assert signal.title == "Zoom meeting"
    end
  end

  # ── helpers ──────────────────────────────────────────────────────────────────

  defp collect_messages(tag) do
    collect_messages(tag, [])
  end

  defp collect_messages(tag, acc) do
    receive do
      {^tag, _, _} = msg -> collect_messages(tag, [msg | acc])
    after
      0 -> Enum.reverse(acc)
    end
  end
end
