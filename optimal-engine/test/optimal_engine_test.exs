defmodule OptimalEngineTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Pipeline.Classifier, as: Classifier
  alias OptimalEngine.Retrieval.Composer, as: Composer
  alias OptimalEngine.Context
  alias OptimalEngine.Signal
  alias OptimalEngine.Routing
  alias OptimalEngine.URI

  # ─────────────────────────────────────────────────────────────
  # Signal struct — round trip (backward compat)
  # ─────────────────────────────────────────────────────────────

  describe "Signal.to_row/1 and from_row/1" do
    test "round-trips a fully populated signal" do
      now = DateTime.utc_now() |> DateTime.truncate(:second)

      signal = %Signal{
        id: "abc123",
        path: "/test/path.md",
        title: "Test Signal",
        mode: :linguistic,
        genre: "spec",
        type: :inform,
        format: :markdown,
        structure: "test-structure",
        created_at: now,
        modified_at: now,
        valid_from: now,
        valid_until: nil,
        supersedes: nil,
        node: "operator",
        sn_ratio: 0.85,
        entities: ["Alice", "Carol"],
        l0_summary: "SPEC | operator | Test Signal [S/N: 0.9]",
        l1_description: "A test signal description",
        content: "Test content",
        routed_to: ["person-operator"],
        score: nil
      }

      row = Signal.to_row(signal)
      assert row.id == "abc123"
      assert row.mode == "linguistic"
      assert row.genre == "spec"
      assert row.entities == Jason.encode!(["Alice", "Carol"])

      row_list = [
        row.id,
        row.path,
        row.title,
        row.mode,
        row.genre,
        row.type,
        row.format,
        row.structure,
        row.created_at,
        row.modified_at,
        row.valid_from,
        row.valid_until,
        row.supersedes,
        row.node,
        row.sn_ratio,
        row.entities,
        row.l0_summary,
        row.l1_description,
        row.content,
        row.routed_to
      ]

      reconstructed = Signal.from_row(row_list)
      assert reconstructed.id == "abc123"
      assert reconstructed.mode == :linguistic
      assert reconstructed.genre == "spec"
      assert reconstructed.entities == ["Alice", "Carol"]
    end
  end

  # ─────────────────────────────────────────────────────────────
  # Context struct
  # ─────────────────────────────────────────────────────────────

  describe "Context.to_row/1 and from_row/1" do
    test "round-trips a resource context" do
      now = DateTime.utc_now() |> DateTime.truncate(:second)

      ctx = %Context{
        id: "res001",
        uri: "optimal://resources/api-docs.md",
        type: :resource,
        path: "/test/docs/api-docs.md",
        title: "API Docs",
        content: "# API\nGet /users",
        l0_abstract: "RESOURCE | document | API Docs",
        l1_overview: "API documentation for the platform",
        signal: nil,
        node: "resources",
        sn_ratio: 0.5,
        entities: [],
        created_at: now,
        modified_at: now,
        routed_to: [],
        metadata: %{"extension" => ".md"}
      }

      row = Context.to_row(ctx)
      assert row.id == "res001"
      assert row.type == "resource"
      assert row.uri == "optimal://resources/api-docs.md"
      assert row.mode == nil
      assert row.genre == nil

      # Reconstruct from row list (24 columns matching context_columns/0)
      row_list = [
        row.id,
        row.uri,
        row.type,
        row.path,
        row.title,
        row.l0_abstract,
        row.l1_overview,
        row.content,
        row.mode,
        row.genre,
        row.signal_type,
        row.format,
        row.structure,
        row.node,
        row.sn_ratio,
        row.entities,
        row.created_at,
        row.modified_at,
        row.valid_from,
        row.valid_until,
        row.supersedes,
        row.routed_to,
        row.metadata,
        row.workspace_id
      ]

      reconstructed = Context.from_row(row_list)
      assert reconstructed.id == "res001"
      assert reconstructed.type == :resource
      assert reconstructed.signal == nil
    end

    test "round-trips a signal context" do
      now = DateTime.utc_now() |> DateTime.truncate(:second)

      signal = %Signal{
        id: "sig001",
        path: "/test/person-operator/signal.md",
        title: "Architecture Decision",
        mode: :linguistic,
        genre: "adr",
        type: :decide,
        format: :markdown,
        structure: "",
        created_at: now,
        modified_at: now,
        valid_from: nil,
        valid_until: nil,
        supersedes: nil,
        node: "operator",
        sn_ratio: 0.9,
        entities: ["Alice"],
        l0_summary: "ADR | operator | Architecture Decision [S/N: 0.9]",
        l1_description: "Decided to use SQLite",
        content: "We decided to use SQLite.",
        routed_to: ["person-operator"],
        score: nil
      }

      ctx = Context.from_signal(signal)
      assert ctx.type == :signal
      assert ctx.signal == signal
      assert String.contains?(ctx.uri, "operator")

      row = Context.to_row(ctx)
      assert row.type == "signal"
      assert row.genre == "adr"
      assert row.mode == "linguistic"
    end

    test "to_signal/1 returns embedded signal for :signal type" do
      sig = %Signal{
        id: "x",
        path: "/test.md",
        title: "Test",
        mode: :linguistic,
        genre: "note",
        type: :inform,
        format: :markdown,
        structure: "",
        created_at: DateTime.utc_now(),
        modified_at: DateTime.utc_now(),
        node: "operator",
        sn_ratio: 0.5,
        entities: [],
        l0_summary: "",
        l1_description: "",
        content: "",
        routed_to: [],
        score: nil
      }

      ctx = Context.from_signal(sig)
      assert Context.to_signal(ctx) == sig
    end

    test "to_signal/1 builds a minimal signal for :resource type" do
      ctx = %Context{
        id: "r1",
        uri: "optimal://resources/doc.md",
        type: :resource,
        title: "Doc",
        content: "content",
        l0_abstract: "RESOURCE | document | Doc",
        l1_overview: "A document",
        node: "resources",
        sn_ratio: 0.5,
        entities: [],
        created_at: DateTime.utc_now(),
        modified_at: DateTime.utc_now(),
        routed_to: [],
        metadata: %{}
      }

      sig = Context.to_signal(ctx)
      assert sig.title == "Doc"
      assert sig.genre == "note"
    end
  end

  # ─────────────────────────────────────────────────────────────
  # URI module
  # ─────────────────────────────────────────────────────────────

  describe "URI.parse/1" do
    test "parses nodes URI" do
      {:ok, parsed} = URI.parse("optimal://nodes/project-platform-launch/context.md")
      assert parsed.namespace == :nodes
      assert parsed.segments == ["project-platform-launch", "context.md"]
    end

    test "parses resources URI" do
      {:ok, parsed} = URI.parse("optimal://resources/api-docs.md")
      assert parsed.namespace == :resources
      assert parsed.segments == ["api-docs.md"]
    end

    test "parses inbox URI" do
      {:ok, parsed} = URI.parse("optimal://inbox/")
      assert parsed.namespace == :inbox
      assert parsed.segments == []
    end

    test "returns error for non-optimal scheme" do
      assert {:error, {:invalid_scheme, _}} = URI.parse("https://example.com")
    end

    test "returns error for empty URI" do
      assert {:error, _} = URI.parse("optimal://")
    end
  end

  describe "URI.context_type/1" do
    test "returns :resource for resources namespace" do
      assert URI.context_type("optimal://resources/doc.md") == :resource
    end

    test "returns :signal for nodes namespace" do
      assert URI.context_type("optimal://nodes/operator/signal.md") == :signal
    end

    test "returns :memory for user/memories namespace" do
      assert URI.context_type("optimal://user/memories/note.md") == :memory
    end

    test "returns :skill for agent/skills namespace" do
      assert URI.context_type("optimal://agent/skills/search.md") == :skill
    end
  end

  describe "URI.node_id/1" do
    test "returns node id for nodes URI" do
      assert URI.node_id("optimal://nodes/project-platform-launch/context.md") ==
               "project-platform-launch"
    end

    test "returns inbox for inbox URI" do
      assert URI.node_id("optimal://inbox/test.md") == "inbox"
    end

    test "returns nil for non-node URI" do
      assert URI.node_id("optimal://resources/doc.md") == nil
    end
  end

  describe "URI.from_path/1" do
    setup do
      original_root = Application.get_env(:optimal_engine, :root_path)
      Application.put_env(:optimal_engine, :root_path, "/test/root")

      on_exit(fn -> Application.put_env(:optimal_engine, :root_path, original_root) end)
    end

    test "builds node URI from org folder path" do
      uri = URI.from_path("/test/root/project-platform-launch/context.md")
      assert uri == "optimal://nodes/project-platform-launch/context.md"
    end

    test "builds resources URI from docs folder path" do
      uri = URI.from_path("/test/root/docs/api.md")
      assert uri == "optimal://resources/api.md"
    end

    test "builds inbox URI for unknown folder" do
      uri = URI.from_path("/test/root/unknown-folder/file.md")
      assert String.starts_with?(uri, "optimal://")
    end
  end

  # ─────────────────────────────────────────────────────────────
  # Classifier — context type detection
  # ─────────────────────────────────────────────────────────────

  describe "Classifier.detect_type/2" do
    test "detects signal from YAML frontmatter with node key" do
      content = """
      ---
      node: operator
      signal:
        genre: spec
      ---
      # A Signal
      """

      assert Classifier.detect_type(content, path: "/person-operator/signal.md") == :signal
    end

    test "detects signal for org folder markdown" do
      content = "# Some note\n\nJust some content."
      assert Classifier.detect_type(content, path: "/person-operator/note.md") == :signal
    end

    test "detects memory from path" do
      assert Classifier.detect_type("content", path: "/_memories/user/fact.md") == :memory
    end

    test "detects skill from path" do
      assert Classifier.detect_type("content", path: "/_skills/search.md") == :skill
    end

    test "defaults to resource for docs content" do
      content = "# API Reference\n\nSome docs here."
      assert Classifier.detect_type(content, path: "/docs/api.md") == :resource
    end

    test "defaults to resource for code files" do
      content = "defmodule Foo do\n  def bar, do: :ok\nend"
      assert Classifier.detect_type(content, path: "/engine/lib/foo.ex") == :resource
    end
  end

  describe "Classifier.classify_context/2" do
    test "classifies a signal markdown with full dimensions" do
      content = """
      ---
      node: operator
      signal:
        genre: spec
        mode: linguistic
        type: inform
        sn_ratio: 0.9
      ---
      # Platform Spec
      ## Requirements
      1. Multi-tenancy
      """

      ctx =
        Classifier.classify_context(content,
          path: "/person-operator/spec.md",
          known_entities: []
        )

      assert ctx.type == :signal
      assert ctx.signal != nil
      assert ctx.signal.genre == "spec"
      assert ctx.title == "Platform Spec"
      assert ctx.node == "operator"
      assert String.length(ctx.l0_abstract) > 0
    end

    test "classifies a resource code file" do
      content = "defmodule Foo do\n  def bar, do: :ok\nend"

      ctx =
        Classifier.classify_context(content,
          path: "/engine/lib/foo.ex",
          type: :resource
        )

      assert ctx.type == :resource
      assert ctx.signal == nil
      assert ctx.title != ""
    end

    test "classifies a memory file" do
      content = "Alice prefers specs over briefs for technical decisions."

      ctx =
        Classifier.classify_context(content,
          path: "/_memories/user/preference.md",
          known_entities: []
        )

      assert ctx.type == :memory
      assert ctx.signal == nil
    end

    test "generates l0_abstract for all types" do
      content = "# API Docs\n\nSome content."
      ctx = Classifier.classify_context(content, path: "/docs/api.md", type: :resource)
      assert String.length(ctx.l0_abstract) > 0
    end
  end

  # ─────────────────────────────────────────────────────────────
  # Classifier — signal classification (backward compat)
  # ─────────────────────────────────────────────────────────────

  describe "Classifier.parse_frontmatter/1" do
    test "parses valid YAML frontmatter" do
      content = """
      ---
      signal:
        mode: linguistic
        genre: spec
        type: inform
        format: markdown
        sn_ratio: 0.9
      node: operator
      ---
      # My Document

      Body content here.
      """

      {fm, body} = Classifier.parse_frontmatter(content)
      assert get_in(fm, ["signal", "genre"]) == "spec"
      assert Map.get(fm, "node") == "operator"
      assert String.contains?(body, "Body content here")
    end

    test "returns empty map when no frontmatter present" do
      content = "# Just a heading\n\nSome content."
      {fm, body} = Classifier.parse_frontmatter(content)
      assert fm == %{}
      assert String.contains?(body, "Just a heading")
    end
  end

  describe "Classifier.classify/2" do
    test "extracts genre from frontmatter" do
      content = """
      ---
      signal:
        mode: linguistic
        genre: decision-log
        type: decide
        sn_ratio: 0.9
      node: operator
      ---
      # Architecture Decision
      We decided to use SQLite.
      """

      signal = Classifier.classify(content)
      assert signal.genre == "decision-log"
      assert signal.type == :decide
      assert signal.node == "operator"
    end

    test "auto-detects genre from content patterns" do
      content = """
      # Spec

      ## Requirements
      1. Must index markdown files
      2. Must classify signals

      ## Acceptance Criteria
      All tests pass.
      """

      signal = Classifier.classify(content)
      assert signal.genre == "spec"
    end

    test "auto-detects spec from Requirements heading" do
      content = "# Spec\n## Requirements\n- item 1\n## Acceptance Criteria\n- done"
      signal = Classifier.classify(content)
      assert signal.genre == "spec"
    end

    test "auto-detects decision-log type from content" do
      content = "# Choice\n\nWe decided to use Elixir for the backend."
      signal = Classifier.classify(content)
      assert signal.type == :decide
    end

    test "extracts known entities from content" do
      content = "Alice and Carol met to discuss the platform."
      signal = Classifier.classify(content, known_entities: ["Alice", "Carol", "Ed"])
      assert "Alice" in signal.entities
      assert "Carol" in signal.entities
      refute "Ed" in signal.entities
    end

    test "generates l0_summary" do
      content = "# My Signal\n\nSome content"
      signal = Classifier.classify(content)
      assert String.contains?(signal.l0_summary, "My Signal")
    end

    test "uses Untitled when no title found" do
      content = "Just some content without a heading."
      signal = Classifier.classify(content)
      assert signal.title == "Untitled"
    end

    test "detects code mode" do
      content = "# Code File\n\n```elixir\ndefmodule Foo do\n  def bar, do: :baz\nend\n```"
      signal = Classifier.classify(content)
      assert signal.mode in [:code, :mixed]
    end

    test "clamps sn_ratio to 0.0..1.0" do
      content = """
      ---
      signal:
        sn_ratio: 1.5
      ---
      # Signal
      """

      signal = Classifier.classify(content)
      assert signal.sn_ratio <= 1.0
    end
  end

  # ─────────────────────────────────────────────────────────────
  # Composer
  # ─────────────────────────────────────────────────────────────

  describe "Composer.render_for/3" do
    setup do
      topology = %{
        endpoints: %{
          "sales-lead" => %{
            id: "sales-lead",
            name: "Bob",
            role: "Sales",
            genre_competence: ["brief", "pitch", "email"],
            channels: ["slack"],
            notes: "Salesperson. Brief genre only."
          },
          "pedro" => %{
            id: "pedro",
            name: "Erin Afonso",
            role: "Frontend Dev",
            genre_competence: ["spec", "readme", "changelog"],
            channels: ["slack", "github"],
            notes: nil
          }
        },
        half_lives: %{"default" => 720}
      }

      signal = %Signal{
        id: "test-id",
        path: "/test/spec.md",
        title: "Platform Architecture Spec",
        mode: :linguistic,
        genre: "spec",
        type: :inform,
        format: :markdown,
        structure: "",
        created_at: DateTime.utc_now(),
        modified_at: DateTime.utc_now(),
        valid_from: nil,
        valid_until: nil,
        supersedes: nil,
        node: "operator",
        sn_ratio: 0.85,
        entities: [],
        l0_summary: "SPEC | operator | Platform Architecture Spec [S/N: 0.9]",
        l1_description: "The platform architecture for the example company.",
        content: """
        ## Requirements
        1. Must support multi-tenancy
        2. Must handle 10K RPS

        ## Acceptance Criteria
        All load tests pass at 10K RPS.
        """,
        routed_to: [],
        score: nil
      }

      %{topology: topology, signal: signal}
    end

    test "renders brief for sales-lead", %{topology: topology, signal: signal} do
      {:ok, rendered} = Composer.render_for(signal, "sales-lead", topology)
      assert String.contains?(rendered, "# Brief:")
      assert String.contains?(rendered, "Platform Architecture Spec")
    end

    test "renders spec for pedro", %{topology: topology, signal: signal} do
      {:ok, rendered} = Composer.render_for(signal, "pedro", topology)
      assert String.contains?(rendered, "# Spec:")
      assert String.contains?(rendered, "Goal")
    end

    test "returns note genre for unknown receiver", %{topology: topology, signal: signal} do
      {:ok, rendered} = Composer.render_for(signal, "unknown-person", topology)
      assert is_binary(rendered)
    end
  end

  # ─────────────────────────────────────────────────────────────
  # Topology
  # ─────────────────────────────────────────────────────────────

  describe "Routing.half_life_for/2" do
    test "returns genre-specific half-life" do
      topology = %{half_lives: %{"spec" => 4320, "message" => 168, "default" => 720}}
      assert Routing.half_life_for(topology, "spec") == 4320
      assert Routing.half_life_for(topology, "message") == 168
    end

    test "returns default when genre not found" do
      topology = %{half_lives: %{"default" => 720}}
      assert Routing.half_life_for(topology, "unknown-genre") == 720
    end
  end

  describe "Routing.primary_genre_for/2" do
    test "returns first genre competence for known receiver" do
      topology = %{
        endpoints: %{
          "pedro" => %{
            genre_competence: ["spec", "readme"],
            name: "Erin",
            role: "",
            channels: [],
            notes: nil
          }
        }
      }

      assert Routing.primary_genre_for(topology, "pedro") == "spec"
    end

    test "returns note for unknown receiver" do
      topology = %{endpoints: %{}}
      assert Routing.primary_genre_for(topology, "nobody") == "note"
    end
  end

  # ─────────────────────────────────────────────────────────────
  # SearchEngine temporal decay (works with Context now)
  # ─────────────────────────────────────────────────────────────

  describe "SearchEngine.temporal_factor/3" do
    test "returns ~1.0 for context modified right now" do
      ctx = %Context{
        modified_at: DateTime.utc_now(),
        created_at: nil,
        signal: %Signal{genre: "spec", modified_at: DateTime.utc_now(), created_at: nil}
      }

      topology = %{half_lives: %{"spec" => 4320, "default" => 720}}
      factor = OptimalEngine.Retrieval.Search.temporal_factor(ctx, DateTime.utc_now(), topology)
      assert_in_delta factor, 1.0, 0.01
    end

    test "returns < 0.5 for context older than one half-life" do
      half_life_hours = 4320
      old_dt = DateTime.add(DateTime.utc_now(), -half_life_hours * 3600 - 1, :second)

      ctx = %Context{
        modified_at: old_dt,
        created_at: nil,
        signal: %Signal{genre: "spec", modified_at: old_dt, created_at: nil}
      }

      topology = %{half_lives: %{"spec" => half_life_hours, "default" => 720}}
      factor = OptimalEngine.Retrieval.Search.temporal_factor(ctx, DateTime.utc_now(), topology)
      assert factor < 0.5
    end

    test "returns 0.5 for nil modified_at" do
      ctx = %Context{modified_at: nil, created_at: nil, signal: nil}
      topology = %{half_lives: %{"default" => 720}}
      factor = OptimalEngine.Retrieval.Search.temporal_factor(ctx, DateTime.utc_now(), topology)
      assert factor == 0.5
    end

    # Backward compat: also works with Signal structs directly
    test "works with a Signal struct" do
      signal = %Signal{modified_at: DateTime.utc_now(), genre: "spec", created_at: nil}
      topology = %{half_lives: %{"spec" => 4320, "default" => 720}}
      factor = OptimalEngine.Retrieval.Search.temporal_factor(signal, DateTime.utc_now(), topology)
      assert_in_delta factor, 1.0, 0.01
    end
  end
end
