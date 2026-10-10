defmodule OptimalEngine.Connectors.AdaptersTest do
  @moduledoc """
  Contract tests that exercise every adapter's Behaviour without
  hitting real APIs. We check:

    * `init/1` rejects missing config keys
    * `init/1` rejects missing credentials
    * `init/1` succeeds with a minimal valid payload
    * `sync/2` returns `{:error, :not_implemented}` (Phase 10 wiring gap)
    * `transform/1` returns a `%Signal{}` for a minimal payload
  """

  use ExUnit.Case, async: true

  alias OptimalEngine.Connectors.Registry
  alias OptimalEngine.Signal

  # Minimal valid config per adapter. Matches the `required_keys` +
  # `credential_keys` declared by each module.
  @fixtures %{
    slack: %{
      "workspace_id" => "T01",
      "channels" => ["C01"],
      "credentials" => %{"bot_token" => "xoxb-test"}
    },
    gmail: %{
      "user_email" => "x@y.com",
      "credentials" => %{
        "oauth_refresh_token" => "r",
        "client_id" => "c",
        "client_secret" => "s"
      }
    },
    drive: %{
      "scope" => "my_drive",
      "credentials" => %{
        "oauth_refresh_token" => "r",
        "client_id" => "c",
        "client_secret" => "s"
      }
    },
    notion: %{
      "workspace_name" => "sample",
      "database_ids" => [],
      "credentials" => %{"integration_token" => "secret_x"}
    },
    jira: %{
      "site_url" => "https://sample.atlassian.net",
      "projects" => ["PROJ"],
      "credentials" => %{"email" => "x@y.com", "api_token" => "t"}
    },
    linear: %{
      "team_ids" => ["team-1"],
      "credentials" => %{"api_key" => "lin_api_x"}
    },
    github: %{
      "org_or_user" => "sample",
      "repos" => ["engine"],
      "credentials" => %{"pat" => "ghp_x"}
    },
    zoom: %{
      "account_id" => "acct-1",
      "credentials" => %{"client_id" => "c", "client_secret" => "s"}
    },
    confluence: %{
      "site_url" => "https://sample.atlassian.net",
      "spaces" => ["DOCS"],
      "credentials" => %{"email" => "x@y.com", "api_token" => "t"}
    },
    teams: %{
      "tenant_id_ms" => "ms-t",
      "team_ids" => ["t-1"],
      "credentials" => %{"client_id" => "c", "client_secret" => "s"}
    },
    dropbox: %{
      "namespace" => "personal",
      "credentials" => %{
        "refresh_token" => "r",
        "app_key" => "k",
        "app_secret" => "s"
      }
    },
    onedrive: %{
      "tenant_id_ms" => "ms-t",
      "drive_id" => "d-1",
      "credentials" => %{"client_id" => "c", "client_secret" => "s"}
    },
    salesforce: %{
      "instance_url" => "https://sample.my.salesforce.com",
      "objects" => ["Account"],
      "credentials" => %{
        "client_id" => "c",
        "client_secret" => "s",
        "refresh_token" => "r"
      }
    },
    hubspot: %{
      "portal_id" => "12345",
      "objects" => ["contacts"],
      "credentials" => %{"access_token" => "pat-x"}
    }
  }

  for {kind, config} <- Map.to_list(@fixtures) do
    @kind kind
    @config config

    describe "adapter #{kind}" do
      test "init/1 succeeds with a minimal valid payload" do
        {:ok, mod} = Registry.fetch(@kind)
        assert {:ok, _state} = mod.init(@config)
      end

      test "init/1 rejects missing config keys" do
        {:ok, mod} = Registry.fetch(@kind)
        # Strip the required keys (keep credentials) → expect error
        creds = Map.take(@config, ["credentials"])
        assert {:error, _} = mod.init(creds)
      end

      test "sync/2 attempts real HTTP -- no adapter returns :not_implemented" do
        {:ok, mod} = Registry.fetch(@kind)
        {:ok, state} = mod.init(@config)
        result = mod.sync(state, nil)
        # All adapters now have real sync/2 implementations. Without valid
        # credentials each returns either {:ok, %{signals, cursor}} (when its
        # reduce loop silently skips per-item auth errors) or {:error, reason}
        # where reason is a real HTTP/auth failure -- never :not_implemented.
        case result do
          {:error, :not_implemented} ->
            flunk("expected real code path, got {:error, :not_implemented}")

          {:error, _reason} ->
            :ok

          {:ok, %{signals: _, cursor: _}} ->
            :ok
        end
      end

      test "transform/1 builds a %Signal{} from a minimal payload" do
        {:ok, mod} = Registry.fetch(@kind)
        payload = minimal_payload(@kind)

        assert {:ok, %Signal{} = signal} = mod.transform(payload)
        assert is_binary(signal.id)
        assert byte_size(signal.id) > 0
      end

      test "kind/0 matches the registry lookup" do
        {:ok, mod} = Registry.fetch(@kind)
        assert mod.kind() == @kind
      end
    end
  end

  # GitHub and HubSpot declare `credential_keys: []` and gate credentials in
  # their own `init/1` (either-or auth schemes), so the generic loop above
  # cannot exercise the rejection path. Pin it explicitly here.
  describe "adapters with alternate credential schemes" do
    test "github init/1 rejects config without a pat or GitHub App credentials" do
      {:ok, mod} = Registry.fetch(:github)

      assert {:error, :missing_credentials} =
               mod.init(%{"org_or_user" => "sample", "repos" => ["engine"]})
    end

    test "github init/1 accepts GitHub App credentials in place of a pat" do
      {:ok, mod} = Registry.fetch(:github)

      config = %{
        "org_or_user" => "sample",
        "repos" => ["engine"],
        "credentials" => %{
          "app_id" => "1",
          "installation_id" => "2",
          "private_key_pem" => "fake-rsa-private-key"
        }
      }

      assert {:ok, _state} = mod.init(config)
    end

    test "hubspot init/1 rejects config without an access token or OAuth credentials" do
      {:ok, mod} = Registry.fetch(:hubspot)

      assert {:error, :missing_credentials} =
               mod.init(%{"portal_id" => "12345", "objects" => ["contacts"]})
    end

    test "hubspot init/1 accepts OAuth credentials in place of an access token" do
      {:ok, mod} = Registry.fetch(:hubspot)

      config = %{
        "portal_id" => "12345",
        "objects" => ["contacts"],
        "credentials" => %{
          "client_id" => "c",
          "client_secret" => "s",
          "refresh_token" => "r"
        }
      }

      assert {:ok, _state} = mod.init(config)
    end
  end

  # Per-adapter minimal inbound payload that satisfies transform/1.
  defp minimal_payload(:slack), do: %{"ts" => "1700000000.000000", "text" => "hi", "user" => "U01"}

  defp minimal_payload(:gmail),
    do: %{"id" => "abc", "snippet" => "hi", "payload" => %{"headers" => []}}

  defp minimal_payload(:drive),
    do: %{"id" => "abc", "name" => "doc.pdf", "mimeType" => "application/pdf"}

  defp minimal_payload(:notion), do: %{"id" => "abc", "blocks" => []}

  defp minimal_payload(:jira),
    do: %{
      "key" => "PROJ-1",
      "fields" => %{
        "summary" => "s",
        "description" => "",
        "assignee" => %{"displayName" => "A"},
        "reporter" => %{"displayName" => "R"},
        "updated" => "2026-01-01T00:00:00Z"
      }
    }

  defp minimal_payload(:linear),
    do: %{
      "id" => "uuid",
      "title" => "t",
      "description" => "",
      "updatedAt" => "2026-01-01T00:00:00Z"
    }

  defp minimal_payload(:github),
    do: %{
      "id" => 1,
      "node_id" => "n1",
      "title" => "t",
      "body" => "b",
      "updated_at" => "2026-01-01T00:00:00Z"
    }

  defp minimal_payload(:zoom),
    do: %{"uuid" => "z1", "topic" => "Sync", "start_time" => "2026-01-01T00:00:00Z"}

  defp minimal_payload(:confluence),
    do: %{
      "id" => "c1",
      "title" => "T",
      "body" => %{"storage" => %{"value" => "<p>hi</p>"}},
      "version" => %{"when" => "2026-01-01T00:00:00Z"}
    }

  defp minimal_payload(:teams),
    do: %{
      "id" => "m1",
      "body" => %{"content" => "<p>hi</p>"},
      "lastModifiedDateTime" => "2026-01-01T00:00:00Z"
    }

  defp minimal_payload(:dropbox),
    do: %{
      "id" => "d1",
      "name" => "f.txt",
      "path_display" => "/f.txt",
      "server_modified" => "2026-01-01T00:00:00Z"
    }

  defp minimal_payload(:onedrive),
    do: %{"id" => "o1", "name" => "f.docx", "lastModifiedDateTime" => "2026-01-01T00:00:00Z"}

  defp minimal_payload(:salesforce),
    do: %{
      "Id" => "sf1",
      "Name" => "Example",
      "LastModifiedDate" => "2026-01-01T00:00:00.000+0000",
      "attributes" => %{"type" => "Account"}
    }

  defp minimal_payload(:hubspot),
    do: %{
      "id" => 1,
      "properties" => %{"company" => "Example", "hs_lastmodifieddate" => "2026-01-01T00:00:00Z"}
    }
end
