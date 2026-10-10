defmodule OptimalEngine.Retrieval.RAGTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Retrieval
  alias OptimalEngine.Retrieval.{Receiver, RAG}
  alias OptimalEngine.Wiki.{Page, Store}

  describe "ask/2 — wiki-first path" do
    test "returns source: :wiki when a curated page exists" do
      suffix = System.unique_integer([:positive])
      slug = "rag-wiki-hit-#{suffix}"

      :ok =
        Store.put(%Page{
          tenant_id: "default",
          slug: slug,
          audience: "default",
          version: 1,
          frontmatter: %{"slug" => slug},
          body: "## Summary\n\nThe answer {{cite: optimal://a}}."
        })

      receiver = Receiver.new(%{format: :markdown, audience: "default"})
      {:ok, result} = RAG.ask(slug, receiver: receiver, skip_intent: true)

      assert result.source == :wiki
      assert result.trace.wiki_hit? == true
      assert result.envelope.body =~ "The answer"
      assert "optimal://a" in result.envelope.sources
    end
  end

  describe "ask/2 — chunks fallback path" do
    test "falls through to hybrid search when no wiki hit" do
      receiver = Receiver.new(%{format: :plain, token_budget: 200})

      {:ok, result} =
        RAG.ask("some_nonexistent_wiki_query_#{System.unique_integer([:positive])}",
          receiver: receiver,
          skip_intent: true
        )

      refute result.trace.wiki_hit?
      assert result.source in [:chunks, :empty]
    end

    test "empty receiver returns a format-appropriate empty envelope" do
      receiver = Receiver.new(%{format: :markdown})

      {:ok, result} =
        RAG.ask("this-query-matches-nothing-#{System.unique_integer([:positive])}",
          receiver: receiver,
          skip_intent: true,
          skip_wiki: true
        )

      assert result.envelope.format == :markdown
      assert is_binary(result.envelope.body)
    end
  end

  describe "ask/2 — versioned memory fallback path" do
    test "falls back to versioned memories when wiki and chunk search miss" do
      workspace_id = "rag-memory-fallback-#{System.unique_integer([:positive])}"

      {:ok, _memory} =
        OptimalEngine.Memory.create(%{
          workspace_id: workspace_id,
          audience: "default",
          content:
            "Conversation bench: Project Atlas handled customer portal pricing. Decision: standardize on annual price protection. Owner: Revenue Lead."
        })

      receiver = Receiver.new(%{format: :markdown, audience: "default", token_budget: 500})

      {:ok, result} =
        RAG.ask("What decision did Project Atlas make for customer portal pricing?",
          receiver: receiver,
          workspace_id: workspace_id,
          hybrid_limit: 0,
          skip_intent: true,
          skip_wiki: true
        )

      assert result.source == :memory
      assert result.trace.n_candidates >= 1
      assert result.trace.n_delivered >= 1
      assert result.envelope.body =~ "standardize on annual price protection"
      assert Enum.any?(result.envelope.sources, &String.starts_with?(&1, "optimal://memory/"))
    end

    test "normalizes ownership phrasing for memory fallback" do
      workspace_id = "rag-memory-owner-fallback-#{System.unique_integer([:positive])}"

      {:ok, _memory} =
        OptimalEngine.Memory.create(%{
          workspace_id: workspace_id,
          audience: "default",
          content:
            "Conversation bench: Project Atlas handled customer portal pricing. Decision: standardize on annual price protection. Owner: Revenue Lead. Backup owner: Operations Lead."
        })

      receiver = Receiver.new(%{format: :markdown, audience: "default", token_budget: 500})

      {:ok, result} =
        RAG.ask("Who owns Project Atlas's customer portal pricing work?",
          receiver: receiver,
          workspace_id: workspace_id,
          hybrid_limit: 0,
          skip_intent: true,
          skip_wiki: true
        )

      assert result.source == :memory
      assert result.envelope.body =~ "Owner: Revenue Lead"
    end
  end

  describe "ask/2 — governed context package path" do
    test "context_package: true answers through the Retrieval Coordinator" do
      workspace_id = "rag-context-package-#{System.unique_integer([:positive])}"

      source =
        OptimalEngine.MemoryCore.SourcePackage.from_text(
          "The project launch was approved in the planning meeting.",
          workspace_id: workspace_id,
          source_type: "meeting_note",
          security_labels: ["internal"],
          partition_ids: ["project-launch"]
        )

      {:ok, claim} =
        OptimalEngine.MemoryCore.ClaimExtractor.extract_from_source(source,
          claim_text: "The source states that the project launch was approved.",
          subject_anchor: "project_launch",
          action_class: "approved",
          actor_id: "agent:rag-test"
        )

      {:ok, fact} =
        OptimalEngine.MemoryCore.FactPromoter.promote(claim,
          fact_text: "The project launch was approved in the planning meeting.",
          verifier_id: "human:reviewer"
        )

      receiver = Receiver.new(%{format: :markdown, audience: "default"})

      {:ok, result} =
        RAG.ask("launch",
          receiver: receiver,
          workspace_id: workspace_id,
          context_package: true,
          actor_id: "agent:rag-test",
          allowed_partitions: ["project-launch"],
          allowed_security_labels: ["internal"]
        )

      assert result.source == :context_package
      assert %OptimalEngine.MemoryCore.ContextPackage{} = result.context_package
      assert result.context_package.fact_links == [%{type: "fact", id: fact.id}]
      assert result.context_package.requesting_actor_id == "agent:rag-test"
      assert result.envelope.format == :markdown
      assert result.envelope.body =~ "The project launch was approved"
      assert result.envelope.body =~ "Context Package"
      assert Enum.any?(result.envelope.sources, &(&1 =~ source.id))
      assert result.trace.n_delivered >= 1
      refute result.trace.wiki_hit?
    end
  end

  describe "ask/2 — tenant-derived workspace scope" do
    test "derives the tenant's default workspace when workspace_id is omitted" do
      tenant_id = "rag-tenant-#{System.unique_integer([:positive])}"
      derived_workspace_id = "#{tenant_id}:default"

      {:ok, _memory} =
        OptimalEngine.Memory.create(%{
          workspace_id: derived_workspace_id,
          audience: "default",
          content:
            "Conversation bench: Project Borealis handled billing. Decision: adopt usage-based billing."
        })

      receiver = Receiver.new(%{format: :markdown, audience: "default", token_budget: 500})

      {:ok, result} =
        RAG.ask("What billing decision did Project Borealis adopt?",
          receiver: receiver,
          tenant_id: tenant_id,
          hybrid_limit: 0,
          skip_intent: true,
          skip_wiki: true
        )

      assert result.source == :memory
      assert result.envelope.body =~ "usage-based billing"

      # Another tenant's unscoped ask resolves to its own default workspace
      # and never reads this tenant's rows.
      {:ok, miss} =
        RAG.ask("What billing decision did Project Borealis adopt?",
          receiver: receiver,
          tenant_id: "other-#{tenant_id}",
          hybrid_limit: 0,
          skip_intent: true,
          skip_wiki: true
        )

      assert miss.source == :empty
    end

    test "context_package path derives the workspace from the tenant" do
      tenant_id = "rag-cp-tenant-#{System.unique_integer([:positive])}"

      {:ok, result} =
        RAG.ask("launch",
          context_package: true,
          tenant_id: tenant_id,
          actor_id: "agent:rag-test"
        )

      assert result.source == :context_package
      assert result.context_package.tenant_id == tenant_id
      assert result.context_package.workspace_id == "#{tenant_id}:default"
    end
  end

  describe "ask/2 — trace fields" do
    test "includes timing and counts" do
      {:ok, result} =
        RAG.ask("anything", skip_intent: true, skip_wiki: true)

      assert is_integer(result.trace.elapsed_ms)
      assert result.trace.elapsed_ms >= 0
      assert is_integer(result.trace.n_candidates)
      assert is_integer(result.trace.n_delivered)
      assert is_boolean(result.trace.truncated?)
    end
  end

  describe "Retrieval facade" do
    test "Retrieval.ask/2 delegates to RAG.ask/2" do
      {:ok, r1} = Retrieval.ask("x", skip_intent: true, skip_wiki: true)
      assert Map.has_key?(r1, :source)
      assert Map.has_key?(r1, :envelope)
      assert Map.has_key?(r1, :trace)
    end

    test "Retrieval.plan/2 delegates to BandwidthPlanner" do
      plan = Retrieval.plan([%{content: "hi", score: 1.0}], 10_000)
      assert length(plan.kept) == 1
    end

    test "Retrieval.anonymous_receiver/1 returns a receiver" do
      r = Retrieval.anonymous_receiver()
      assert r.kind == :unknown
    end
  end
end
