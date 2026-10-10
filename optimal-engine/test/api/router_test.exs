defmodule OptimalEngine.API.RouterTest do
  use ExUnit.Case, async: false

  import Plug.Test
  import Plug.Conn

  alias OptimalEngine.API.Router
  alias OptimalEngine.Wiki.{Page, Store}

  @opts Router.init([])

  defp request(method, path, body \\ nil) do
    conn =
      case body do
        nil ->
          conn(method, path)

        b ->
          conn(method, path, Jason.encode!(b)) |> put_req_header("content-type", "application/json")
      end

    Router.call(conn, @opts)
  end

  defp extracted_policy_claim(workspace_id, opts) do
    source =
      OptimalEngine.MemoryCore.source_package_from_text(Keyword.fetch!(opts, :claim_text),
        workspace_id: workspace_id,
        source_type: "api_test_source",
        security_labels: ["internal"],
        partition_ids: ["api-test"]
      )

    OptimalEngine.MemoryCore.extract_claim(source, opts)
  end

  describe "GET /api/status" do
    test "returns a JSON status payload" do
      conn = request(:get, "/api/status")
      assert conn.status == 200
      assert {:ok, body} = Jason.decode(conn.resp_body)
      assert Map.has_key?(body, "status")
      assert Map.has_key?(body, "ok?")
      assert Map.has_key?(body, "checks")
    end
  end

  describe "GET /api/metrics" do
    test "returns counters + histograms + uptime" do
      conn = request(:get, "/api/metrics")
      assert conn.status == 200
      assert {:ok, body} = Jason.decode(conn.resp_body)
      assert Map.has_key?(body, "counters")
      assert Map.has_key?(body, "histograms")
      assert Map.has_key?(body, "uptime_ms")
    end
  end

  describe "storage catalog" do
    test "lists every agent-readable logical store" do
      conn = request(:get, "/api/stores")
      assert conn.status == 200
      assert {:ok, body} = Jason.decode(conn.resp_body)

      ids = Enum.map(body["stores"], & &1["id"])

      assert body["count"] == 13
      assert "relational" in ids
      assert "full_text" in ids
      assert "vector" in ids
      assert "graph" in ids
      assert "jobs" in ids
      assert "metrics" in ids
      assert "backups" in ids
      assert "decomposition" in ids
      assert "models" in ids
      assert "secrets" in ids
    end

    test "returns one store without exposing record content" do
      conn = request(:get, "/api/stores/vector")
      assert conn.status == 200
      assert {:ok, body} = Jason.decode(conn.resp_body)
      assert body["id"] == "vector"
      assert body["status"] == "available"
      assert is_integer(body["row_count"])
      refute Map.has_key?(body, "records")
    end

    test "requires workspace scope and refuses secret records" do
      assert request(:get, "/api/stores/jobs/records").status == 400

      conn = request(:get, "/api/stores/jobs/records?workspace_id=default:knowledge-intake")
      assert conn.status == 200
      assert {:ok, %{"records" => records}} = Jason.decode(conn.resp_body)
      assert is_list(records)

      assert request(:get, "/api/stores/secrets/records?workspace_id=x").status == 403
    end

    test "exposes the deep storage audit" do
      conn = request(:get, "/api/stores/audit")
      assert {:ok, body} = Jason.decode(conn.resp_body)
      assert conn.status == if(body["ok"], do: 200, else: 503)
      assert is_boolean(body["ok"])
      assert is_list(body["checks"])
      assert Enum.any?(body["checks"], &(&1["name"] == "sqlite_integrity"))
      assert Enum.any?(body["checks"], &(&1["name"] == "vector_integrity"))
    end
  end

  describe "physical storage providers" do
    test "lists providers without exposing connection values" do
      System.put_env("OPTIMAL_POSTGRES_URL", "postgres://private-user:private-pass@localhost/db")
      on_exit(fn -> System.delete_env("OPTIMAL_POSTGRES_URL") end)

      conn = request(:get, "/api/storage/providers")
      assert conn.status == 200
      assert {:ok, body} = Jason.decode(conn.resp_body)
      assert body["count"] >= 13
      assert Enum.any?(body["providers"], &(&1["id"] == "postgres"))
      refute conn.resp_body =~ "private-user"
      refute conn.resp_body =~ "private-pass"
    end

    test "returns use cases and builds a deduplicated plan" do
      cases = request(:get, "/api/storage/use-cases")
      assert cases.status == 200
      assert {:ok, %{"use_cases" => use_cases}} = Jason.decode(cases.resp_body)
      assert Enum.any?(use_cases, &(&1["id"] == "multi_device"))

      conn = request(:post, "/api/storage/plan", %{"use_cases" => ["cloud_team", "media_archive"]})
      assert conn.status == 200
      assert {:ok, body} = Jason.decode(conn.resp_body)
      ids = Enum.map(body["providers"], & &1["id"])
      assert "postgres" in ids
      assert "s3" in ids
      assert length(ids) == length(Enum.uniq(ids))
    end

    test "rejects unknown use cases" do
      conn = request(:post, "/api/storage/plan", %{"use_cases" => ["unknown"]})
      assert conn.status == 422
    end

    test "new workspaces receive local-first policy and scoped mutation ledger" do
      suffix = System.unique_integer([:positive])

      created =
        request(:post, "/api/workspaces", %{
          "slug" => "storage-api-#{suffix}",
          "name" => "Storage API #{suffix}"
        })

      assert created.status == 201
      assert {:ok, %{"id" => workspace_id}} = Jason.decode(created.resp_body)

      policy = request(:get, "/api/storage/workspaces/#{workspace_id}/policy")
      assert policy.status == 200
      assert {:ok, %{"use_cases" => ["desktop_local"]}} = Jason.decode(policy.resp_body)

      mutation =
        request(:post, "/api/storage/workspaces/#{workspace_id}/mutations", %{
          "device_id" => "api-device",
          "entity_type" => "claim",
          "entity_id" => "claim-1",
          "operation" => "create",
          "payload" => %{"text" => "Scoped"},
          "idempotency_key" => "api-mutation-#{suffix}"
        })

      assert mutation.status == 201

      listed = request(:get, "/api/storage/workspaces/#{workspace_id}/mutations?after=0")
      assert listed.status == 200
      assert {:ok, %{"count" => 1}} = Jason.decode(listed.resp_body)
    end
  end

  describe "POST /api/workspaces" do
    test "creates workspaces through topology lifecycle and seeds node types" do
      suffix = System.unique_integer([:positive])
      tmp_dir = Path.join(System.tmp_dir!(), "api_workspace_topology_#{suffix}")
      original_root = Application.get_env(:optimal_engine, :root_path)
      Application.put_env(:optimal_engine, :root_path, tmp_dir)

      on_exit(fn ->
        if original_root do
          Application.put_env(:optimal_engine, :root_path, original_root)
        else
          Application.delete_env(:optimal_engine, :root_path)
        end

        File.rm_rf(tmp_dir)
      end)

      conn =
        request(:post, "/api/workspaces", %{
          "slug" => "api-topology-#{suffix}",
          "name" => "API Topology #{suffix}",
          "description" => "API workspace should use topology lifecycle"
        })

      assert conn.status == 201
      assert {:ok, body} = Jason.decode(conn.resp_body)

      assert {:ok, node_types} =
               OptimalEngine.WorkspaceTopology.list_node_types(workspace_id: body["id"])

      slugs = Enum.map(node_types, & &1.slug)
      assert "project" in slugs
      assert "person" in slugs
      assert "operation" in slugs
      assert "context" in slugs
    end
  end

  describe "organizations" do
    test "creates organizations and scopes workspaces to them" do
      suffix = System.unique_integer([:positive])

      create_conn =
        request(:post, "/api/organizations", %{
          "slug" => "api-org-#{suffix}",
          "name" => "API Organization #{suffix}"
        })

      assert create_conn.status == 201
      assert {:ok, organization} = Jason.decode(create_conn.resp_body)

      list_conn = request(:get, "/api/organizations")
      assert list_conn.status == 200
      assert {:ok, body} = Jason.decode(list_conn.resp_body)
      assert Enum.any?(body["organizations"], &(&1["id"] == organization["id"]))
    end
  end

  describe "POST /api/rag" do
    test "requires a query in the body" do
      conn = request(:post, "/api/rag", %{})
      assert conn.status == 400
    end

    test "returns a RAG envelope for a valid query" do
      conn =
        request(:post, "/api/rag", %{
          "query" => "nonexistent-rag-api-probe-#{System.unique_integer([:positive])}",
          "format" => "markdown",
          "audience" => "default",
          "skip_intent" => true,
          "skip_wiki" => true
        })

      assert conn.status == 200
      {:ok, body} = Jason.decode(conn.resp_body)
      assert Map.has_key?(body, "source")
      assert Map.has_key?(body, "envelope")
      assert Map.has_key?(body, "trace")
    end

    test "forwards benchmark controls to the memory fallback path" do
      workspace_id = "api-rag-memory-fallback-#{System.unique_integer([:positive])}"

      {:ok, _memory} =
        OptimalEngine.Memory.create(%{
          workspace_id: workspace_id,
          audience: "default",
          content:
            "Conversation bench: Project Atlas handled customer portal pricing. Decision: standardize on annual price protection. Owner: Revenue Lead."
        })

      conn =
        request(:post, "/api/rag", %{
          "query" => "What decision did Project Atlas make for customer portal pricing?",
          "workspace" => workspace_id,
          "format" => "markdown",
          "skip_intent" => true,
          "skip_wiki" => true,
          "memory_limit" => 2
        })

      assert conn.status == 200
      {:ok, body} = Jason.decode(conn.resp_body)
      assert body["source"] == "memory"
      assert body["trace"]["n_candidates"] >= 1
      assert body["envelope"]["body"] =~ "standardize on annual price protection"
    end
  end

  describe "Memory Core claim governance API" do
    test "gated memory intake adds salient memory and skips duplicates" do
      workspace_id = "api-memory-remember-#{System.unique_integer([:positive])}"

      first_conn =
        request(:post, "/api/memory/remember", %{
          "workspace" => workspace_id,
          "content" => "Decision: Revenue lead owns renewal pricing for Q4 at $2000.",
          "metadata" => %{"source" => "api-remember-test"}
        })

      assert first_conn.status == 200
      assert {:ok, first_body} = Jason.decode(first_conn.resp_body)
      assert first_body["action"] == "add"
      assert first_body["memory"]["content"] =~ "Revenue lead"
      assert first_body["gate"]["should_encode"] == true
      assert first_body["dedup"]["action"] == "add"

      list_conn = request(:get, "/api/memory-core/claims?workspace=#{workspace_id}")
      assert {:ok, list_body} = Jason.decode(list_conn.resp_body)
      assert list_body["count"] == 1

      duplicate_conn =
        request(:post, "/api/memory/remember", %{
          "workspace" => workspace_id,
          "content" => "Decision: Revenue lead owns renewal pricing for Q4 at $2000."
        })

      assert duplicate_conn.status == 200
      assert {:ok, duplicate_body} = Jason.decode(duplicate_conn.resp_body)
      assert duplicate_body["action"] == "skip"
      assert duplicate_body["memory"]["id"] == first_body["memory"]["id"]
      assert duplicate_body["memory"]["was_existing"] == true
      assert duplicate_body["dedup"]["action"] == "skip"

      after_conn = request(:get, "/api/memory-core/claims?workspace=#{workspace_id}")
      assert {:ok, after_body} = Jason.decode(after_conn.resp_body)
      assert after_body["count"] == 1
    end

    test "lists pending claims and promotes one into fact and memory object" do
      workspace_id = "api-claim-review-#{System.unique_integer([:positive])}"
      content = "API claim promotion #{System.unique_integer([:positive])}"

      create_conn =
        request(:post, "/api/memory", %{
          "workspace" => workspace_id,
          "content" => content,
          "metadata" => %{"source" => "api-test"}
        })

      assert create_conn.status == 201

      list_conn = request(:get, "/api/memory-core/claims?workspace=#{workspace_id}")
      assert list_conn.status == 200
      assert {:ok, list_body} = Jason.decode(list_conn.resp_body)
      assert list_body["count"] == 1

      [claim] = list_body["claims"]
      assert claim["claim_text"] == content
      assert claim["review_status"] == "unreviewed"

      get_conn = request(:get, "/api/memory-core/claims/#{claim["id"]}?workspace=#{workspace_id}")
      assert get_conn.status == 200
      assert {:ok, get_body} = Jason.decode(get_conn.resp_body)
      assert get_body["id"] == claim["id"]

      promote_conn =
        request(:post, "/api/memory-core/claims/#{claim["id"]}/promote", %{
          "workspace" => workspace_id,
          "actor_id" => "user:api-reviewer",
          "fact_text" => "Accepted #{content}",
          "summary" => "Remember #{content}",
          "memory_type" => "reviewed_api_note"
        })

      assert promote_conn.status == 200
      assert {:ok, promote_body} = Jason.decode(promote_conn.resp_body)
      assert promote_body["claim"]["review_status"] == "accepted"
      assert promote_body["fact"]["fact_text"] == "Accepted #{content}"
      assert promote_body["memory_object"]["summary"] == "Remember #{content}"

      after_conn = request(:get, "/api/memory-core/claims?workspace=#{workspace_id}")
      assert after_conn.status == 200
      assert {:ok, after_body} = Jason.decode(after_conn.resp_body)
      assert after_body["count"] == 0
    end

    test "rejects claims and prevents later promotion" do
      workspace_id = "api-claim-reject-#{System.unique_integer([:positive])}"
      content = "API claim rejection #{System.unique_integer([:positive])}"

      assert request(:post, "/api/memory", %{
               "workspace" => workspace_id,
               "content" => content
             }).status == 201

      list_conn = request(:get, "/api/memory-core/claims?workspace=#{workspace_id}")
      assert {:ok, %{"claims" => [claim]}} = Jason.decode(list_conn.resp_body)

      reject_conn =
        request(:post, "/api/memory-core/claims/#{claim["id"]}/reject", %{
          "workspace" => workspace_id,
          "actor_id" => "user:api-reviewer"
        })

      assert reject_conn.status == 200
      assert {:ok, reject_body} = Jason.decode(reject_conn.resp_body)
      assert reject_body["claim"]["review_status"] == "rejected"

      promote_conn =
        request(:post, "/api/memory-core/claims/#{claim["id"]}/promote", %{
          "workspace" => workspace_id,
          "actor_id" => "user:api-reviewer"
        })

      assert promote_conn.status == 409
      assert {:ok, %{"error" => "claim rejected"}} = Jason.decode(promote_conn.resp_body)
    end

    test "returns claim review queue summary with filters" do
      workspace_id = "api-claim-queue-#{System.unique_integer([:positive])}"

      assert request(:post, "/api/memory", %{
               "workspace" => workspace_id,
               "content" => "API queue pending #{System.unique_integer([:positive])}"
             }).status == 201

      assert request(:post, "/api/memory", %{
               "workspace" => workspace_id,
               "content" => "API queue rejected #{System.unique_integer([:positive])}"
             }).status == 201

      list_conn = request(:get, "/api/memory-core/claims?workspace=#{workspace_id}")
      assert {:ok, %{"claims" => [claim_a, claim_b]}} = Jason.decode(list_conn.resp_body)

      reject_conn =
        request(:post, "/api/memory-core/claims/#{claim_b["id"]}/reject", %{
          "workspace" => workspace_id,
          "actor_id" => "user:api-reviewer"
        })

      assert reject_conn.status == 200

      queue_conn = request(:get, "/api/memory-core/claim-review?workspace=#{workspace_id}")
      assert queue_conn.status == 200
      assert {:ok, queue_body} = Jason.decode(queue_conn.resp_body)

      assert queue_body["count"] == 2
      assert queue_body["review_counts"]["unreviewed"] == 1
      assert queue_body["review_counts"]["rejected"] == 1
      assert queue_body["lifecycle_counts"]["pending"] == 1
      assert queue_body["lifecycle_counts"]["rejected"] == 1

      filtered_conn =
        request(
          :get,
          "/api/memory-core/claim-review?workspace=#{workspace_id}&review_status=unreviewed&lifecycle_state=pending"
        )

      assert filtered_conn.status == 200
      assert {:ok, filtered_body} = Jason.decode(filtered_conn.resp_body)
      assert filtered_body["count"] == 1
      assert [filtered_claim] = filtered_body["claims"]
      assert filtered_claim["id"] == claim_a["id"]
    end

    test "promotion API reports conflicts and accepts explicit supersession" do
      workspace_id = "api-claim-policy-#{System.unique_integer([:positive])}"

      assert {:ok, original_claim} =
               extracted_policy_claim(workspace_id,
                 claim_text: "The customer onboarding date is June 10.",
                 subject_anchor: "customer_onboarding",
                 action_class: "date",
                 object_anchor: "launch"
               )

      original_conn =
        request(:post, "/api/memory-core/claims/#{original_claim.id}/promote", %{
          "workspace" => workspace_id,
          "actor_id" => "user:api-reviewer",
          "fact_text" => "The customer onboarding date is June 10."
        })

      assert original_conn.status == 200
      assert {:ok, %{"fact" => %{"id" => original_fact_id}}} = Jason.decode(original_conn.resp_body)

      assert {:ok, replacement_claim} =
               extracted_policy_claim(workspace_id,
                 claim_text: "The customer onboarding date is June 17.",
                 subject_anchor: "customer_onboarding",
                 action_class: "date",
                 object_anchor: "launch"
               )

      conflict_conn =
        request(:post, "/api/memory-core/claims/#{replacement_claim.id}/promote", %{
          "workspace" => workspace_id,
          "actor_id" => "user:api-reviewer",
          "fact_text" => "The customer onboarding date is June 17."
        })

      assert conflict_conn.status == 409

      assert {:ok,
              %{"error" => "claim contradicts current facts", "fact_ids" => [^original_fact_id]}} =
               Jason.decode(conflict_conn.resp_body)

      supersede_conn =
        request(:post, "/api/memory-core/claims/#{replacement_claim.id}/promote", %{
          "workspace" => workspace_id,
          "actor_id" => "user:api-reviewer",
          "fact_text" => "The customer onboarding date is June 17.",
          "supersedes_fact_id" => original_fact_id,
          "supersession_reason" => "new source evidence"
        })

      assert supersede_conn.status == 200

      assert {:ok, [["superseded"]]} =
               OptimalEngine.Store.raw_query(
                 "SELECT lifecycle_state FROM facts WHERE workspace_id = ?1 AND id = ?2",
                 [workspace_id, original_fact_id]
               )

      assert {:ok, [["superseded", "superseded"]]} =
               OptimalEngine.Store.raw_query(
                 "SELECT lifecycle_state, supersession_status FROM memory_objects WHERE workspace_id = ?1 AND fact_links LIKE ?2",
                 [workspace_id, "%#{original_fact_id}%"]
               )
    end

    test "refreshes stale context packages through API" do
      workspace_id = "api-context-refresh-#{System.unique_integer([:positive])}"

      assert {:ok, claim} =
               extracted_policy_claim(workspace_id,
                 claim_text: "The API launch plan is approved.",
                 subject_anchor: "api_launch",
                 action_class: "approved",
                 object_anchor: "plan"
               )

      assert {:ok, accepted} =
               OptimalEngine.MemoryCore.promote_claim(claim.id,
                 workspace_id: workspace_id,
                 actor_id: "user:api-reviewer",
                 fact_text: "The API launch plan is approved.",
                 summary: "API launch plan approval is source-backed."
               )

      assert {:ok, package} =
               OptimalEngine.MemoryCore.retrieve("launch",
                 workspace_id: workspace_id,
                 actor_id: "agent:api",
                 allowed_partitions: ["api-test"],
                 allowed_security_labels: ["internal"]
               )

      assert :ok =
               OptimalEngine.MemoryCore.Store.invalidate_context_packages_for_object(
                 "fact",
                 accepted.fact.id,
                 workspace_id: workspace_id,
                 reason: "api_refresh_test"
               )

      refresh_conn =
        request(:post, "/api/memory-core/context-packages/refresh-stale", %{
          "workspace" => workspace_id,
          "actor_id" => "agent:api",
          "batch_limit" => 10
        })

      assert refresh_conn.status == 200
      assert {:ok, body} = Jason.decode(refresh_conn.resp_body)
      assert body["stale_context_package_ids"] == [package.id]
      assert body["refreshed_count"] == 1
      assert body["error_count"] == 0
      assert [%{"refresh_state" => "fresh"}] = body["refreshed_context_packages"]

      assert {:ok, [["refreshed"]]} =
               OptimalEngine.Store.raw_query(
                 "SELECT refresh_state FROM context_packages WHERE workspace_id = ?1 AND id = ?2",
                 [workspace_id, package.id]
               )
    end

    test "active pool API opens task memory, loads context, records observations, and closes" do
      workspace_id = "api-active-pool-#{System.unique_integer([:positive])}"

      assert {:ok, claim} =
               extracted_policy_claim(workspace_id,
                 claim_text: "The active pool launch plan is approved.",
                 subject_anchor: "active_pool_launch",
                 action_class: "approved",
                 object_anchor: "plan"
               )

      assert {:ok, _accepted} =
               OptimalEngine.MemoryCore.promote_claim(claim.id,
                 workspace_id: workspace_id,
                 actor_id: "user:api-reviewer",
                 fact_text: "The active pool launch plan is approved.",
                 summary: "Active pool launch approval is source-backed."
               )

      open_conn =
        request(:post, "/api/memory-core/active-pools", %{
          "workspace" => workspace_id,
          "task_type" => "launch_review",
          "subject_anchor" => "active_pool_launch",
          "member_links" => [%{"type" => "human", "id" => "human:reviewer"}],
          "agent_links" => [%{"type" => "agent", "id" => "agent:api"}],
          "security_labels" => ["internal"],
          "partition_ids" => ["api-test"]
        })

      assert open_conn.status == 200
      assert {:ok, %{"active_memory_pool" => pool}} = Jason.decode(open_conn.resp_body)
      assert pool["lifecycle_state"] == "open"
      assert pool["workspace_id"] == workspace_id

      get_conn = request(:get, "/api/memory-core/active-pools/#{pool["id"]}")
      assert get_conn.status == 200

      retrieve_conn =
        request(:post, "/api/memory-core/active-pools/#{pool["id"]}/retrieve", %{
          "workspace" => workspace_id,
          "query" => "launch",
          "actor_id" => "agent:api",
          "allowed_partitions" => ["api-test"],
          "allowed_security_labels" => ["internal"]
        })

      assert retrieve_conn.status == 200
      assert {:ok, retrieve_body} = Jason.decode(retrieve_conn.resp_body)
      package = retrieve_body["context_package"]
      loaded_pool = retrieve_body["active_memory_pool"]
      assert package["filtered_object_summary"]["returned_facts"] == 1

      assert %{"type" => "context_package", "id" => package["id"]} in loaded_pool[
               "context_package_links"
             ]

      refresh_conn =
        request(:post, "/api/memory-core/active-pools/#{pool["id"]}/refresh-context", %{
          "allowed_partitions" => ["api-test"],
          "allowed_security_labels" => ["internal"]
        })

      assert refresh_conn.status == 200
      assert {:ok, refresh_body} = Jason.decode(refresh_conn.resp_body)
      assert refresh_body["refreshed_count"] == 0
      assert refresh_body["skipped_context_package_ids"] == [package["id"]]

      observation_conn =
        request(:post, "/api/memory-core/active-pools/#{pool["id"]}/observations", %{
          "observation" => "Agent observed that launch approval still needs finance follow-up.",
          "claim_text" => "Launch approval still needs finance follow-up.",
          "actor_id" => "agent:api",
          "subject_anchor" => "active_pool_launch",
          "action_class" => "needs_follow_up",
          "object_anchor" => "finance",
          "aggregate_confidence" => 0.66,
          "aggregate_precision" => 0.6
        })

      assert observation_conn.status == 200
      assert {:ok, observation_body} = Jason.decode(observation_conn.resp_body)
      assert observation_body["pending_claim"]["review_status"] == "unreviewed"
      assert observation_body["active_memory_pool"]["refresh_state"] == "dirty"

      close_conn =
        request(:post, "/api/memory-core/active-pools/#{pool["id"]}/close", %{
          "reason" => "api test complete"
        })

      assert close_conn.status == 200
      assert {:ok, %{"active_memory_pool" => closed_pool}} = Jason.decode(close_conn.resp_body)
      assert closed_pool["lifecycle_state"] == "closed"
      assert closed_pool["archive_state"] == "archived"
    end
  end

  describe "POST /api/assets" do
    test "preserves uploaded bytes through Memory Core and optionally runs an adapter" do
      suffix = System.unique_integer([:positive])
      tmp_dir = Path.join(System.tmp_dir!(), "api_asset_upload_#{suffix}")
      original_root = Application.get_env(:optimal_engine, :root_path)
      Application.put_env(:optimal_engine, :root_path, tmp_dir)

      on_exit(fn ->
        if original_root do
          Application.put_env(:optimal_engine, :root_path, original_root)
        else
          Application.delete_env(:optimal_engine, :root_path)
        end

        File.rm_rf(tmp_dir)
      end)

      workspace_id = "api-asset-upload-#{suffix}"

      transcript_json =
        Jason.encode!(%{segments: [%{start: 0.0, end: 1.0, text: "API upload segment"}]})

      conn =
        request(:post, "/api/assets", %{
          "workspace" => workspace_id,
          "filename" => "meeting.wav",
          "content_base64" => Base.encode64("RIFF....WAVE api upload"),
          "security_labels" => ["internal"],
          "partition_ids" => ["project-api"],
          "adapter_id" => "openai_whisper",
          "adapter_role" => "audio_transcription",
          "command" => "printf",
          "args" => [transcript_json],
          "confidence" => 0.83,
          "precision" => 0.74
        })

      assert conn.status == 201
      assert {:ok, body} = Jason.decode(conn.resp_body)

      assert body["asset"]["workspace_id"] == workspace_id
      assert body["asset"]["modality"] == "audio"
      assert body["asset"]["source_package_id"] == body["source_package"]["id"]
      assert body["source_package"]["source_type"] == "file"
      assert body["adapter_run"]["status"] == "completed"
      assert body["adapter_run"]["adapter_id"] == "openai_whisper"

      assert [%{"extraction_type" => "transcript", "content_text" => "API upload segment"}] =
               body["asset_extractions"]

      assert {:ok, stored_asset} =
               OptimalEngine.MemoryCore.get_asset(body["asset"]["id"], workspace_id: workspace_id)

      assert File.exists?(stored_asset.storage_path)
    end

    test "rejects uploads without a file source" do
      conn = request(:post, "/api/assets", %{"workspace" => "api-missing-upload"})
      assert conn.status == 400

      assert {:ok, %{"error" => "path or content_base64 is required"}} =
               Jason.decode(conn.resp_body)
    end
  end

  describe "GET /api/grep" do
    test "returns 400 when q is missing" do
      conn = request(:get, "/api/grep")
      assert conn.status == 400
      {:ok, body} = Jason.decode(conn.resp_body)
      assert body["error"] =~ "q is required"
    end

    test "returns JSON with query, workspace_id, and results array" do
      conn = request(:get, "/api/grep?q=test&workspace=default&limit=5")
      assert conn.status == 200
      {:ok, body} = Jason.decode(conn.resp_body)
      assert Map.has_key?(body, "query")
      assert Map.has_key?(body, "workspace_id")
      assert Map.has_key?(body, "results")
      assert is_list(body["results"])
    end

    test "each result has the full signal trace keys" do
      conn = request(:get, "/api/grep?q=test&limit=3")
      assert conn.status == 200
      {:ok, body} = Jason.decode(conn.resp_body)

      Enum.each(body["results"], fn r ->
        assert Map.has_key?(r, "slug")
        assert Map.has_key?(r, "scale")
        assert Map.has_key?(r, "intent")
        assert Map.has_key?(r, "sn_ratio")
        assert Map.has_key?(r, "modality")
        assert Map.has_key?(r, "snippet")
        assert Map.has_key?(r, "score")
      end)
    end

    test "literal=true still returns well-formed response" do
      conn = request(:get, "/api/grep?q=test&literal=true&limit=3")
      assert conn.status == 200
      {:ok, body} = Jason.decode(conn.resp_body)
      assert is_list(body["results"])
    end

    test "intent and scale params are accepted without error" do
      conn = request(:get, "/api/grep?q=pricing&intent=record_fact&scale=section&limit=5")
      assert conn.status == 200
    end

    test "unknown intent is gracefully tolerated" do
      conn = request(:get, "/api/grep?q=test&intent=totally_bogus&limit=3")
      # Should not crash — engine ignores invalid intent
      assert conn.status == 200
    end
  end

  describe "GET /api/wiki" do
    test "returns a list of wiki pages" do
      conn = request(:get, "/api/wiki?tenant=default")
      assert conn.status == 200
      {:ok, body} = Jason.decode(conn.resp_body)
      assert is_list(body["pages"])
    end
  end

  describe "GET /api/wiki/:slug" do
    test "returns 404 for an unknown slug" do
      conn = request(:get, "/api/wiki/does-not-exist-#{System.unique_integer([:positive])}")
      assert conn.status == 404
    end

    test "returns rendered body for an existing page" do
      suffix = System.unique_integer([:positive])
      slug = "api-wiki-render-#{suffix}"

      :ok =
        Store.put(%Page{
          tenant_id: "default",
          slug: slug,
          audience: "default",
          version: 1,
          frontmatter: %{"slug" => slug},
          body: "## Summary\n\nHello."
        })

      conn = request(:get, "/api/wiki/#{slug}?format=plain")
      assert conn.status == 200
      {:ok, body} = Jason.decode(conn.resp_body)
      assert body["slug"] == slug
      assert body["body"] =~ "Hello"
    end
  end

  # ---------------------------------------------------------------------------
  # API versioning — /v1/ prefix rewrite + response headers + status field
  # ---------------------------------------------------------------------------

  describe "GET /v1/status (version rewrite)" do
    test "returns 200 — same as /api/status" do
      conn = request(:get, "/v1/status")
      assert conn.status == 200
    end

    test "response body has same keys as /api/status" do
      v1_conn = request(:get, "/v1/status")
      api_conn = request(:get, "/api/status")
      {:ok, v1_body} = Jason.decode(v1_conn.resp_body)
      {:ok, api_body} = Jason.decode(api_conn.resp_body)
      assert Enum.sort(Map.keys(v1_body)) == Enum.sort(Map.keys(api_body))
    end

    test "status response includes api_version field set to v1" do
      conn = request(:get, "/v1/status")
      {:ok, body} = Jason.decode(conn.resp_body)
      assert body["api_version"] == "v1"
    end
  end

  describe "GET /api/status (backward compat)" do
    test "still returns 200" do
      conn = request(:get, "/api/status")
      assert conn.status == 200
    end

    test "includes api_version field" do
      conn = request(:get, "/api/status")
      {:ok, body} = Jason.decode(conn.resp_body)
      assert body["api_version"] == "v1"
    end
  end

  describe "POST /v1/rag (version rewrite)" do
    test "returns 400 when query is missing — same as /api/rag" do
      conn = request(:post, "/v1/rag", %{})
      assert conn.status == 400
    end

    test "returns a RAG envelope for a valid query" do
      conn =
        request(:post, "/v1/rag", %{
          "query" => "nonexistent-v1-rag-probe-#{System.unique_integer([:positive])}",
          "format" => "markdown",
          "audience" => "default"
        })

      assert conn.status == 200
      {:ok, body} = Jason.decode(conn.resp_body)
      assert Map.has_key?(body, "source")
      assert Map.has_key?(body, "envelope")
    end
  end

  describe "X-API-Version response header" do
    test "is present on GET /api/status" do
      conn = request(:get, "/api/status")
      assert get_resp_header(conn, "x-api-version") == ["v1"]
    end

    test "is present on GET /v1/status" do
      conn = request(:get, "/v1/status")
      assert get_resp_header(conn, "x-api-version") == ["v1"]
    end
  end
end
