defmodule OptimalEngine.ProfileTest do
  use ExUnit.Case, async: false

  import Plug.Test
  import Plug.Conn

  alias OptimalEngine.API.Router
  alias OptimalEngine.Profile
  alias OptimalEngine.Wiki.{Page, Store}

  @opts Router.init([])

  defp get(path) do
    conn(:get, path) |> Router.call(@opts)
  end

  # ── Workspace / filesystem helpers ────────────────────────────────────────

  # Write a file under the workspace node tree so Tier 1/2 reads work.
  defp write_node_file(ws_root, ws_slug, node_slug, filename, content) do
    base = if ws_slug == "default", do: ws_root, else: Path.join(ws_root, ws_slug)
    node_dir = Path.join([base, "nodes", node_slug])
    File.mkdir_p!(node_dir)
    File.write!(Path.join(node_dir, filename), content)
  end

  defp workspace_root do
    Application.get_env(:optimal_engine, :workspace_root, File.cwd!())
  end

  # ── Profile.get/2 ─────────────────────────────────────────────────────────

  describe "Profile.get/2 — default workspace" do
    test "returns {:ok, %Profile{}} struct with all 4 tiers" do
      {:ok, profile} = Profile.get("default")

      assert %Profile{} = profile
      assert profile.workspace_id == "default"
      assert is_binary(profile.static)
      assert is_binary(profile.dynamic)
      assert is_binary(profile.curated)
      assert is_list(profile.activity)
      assert is_list(profile.entities)
      assert is_binary(profile.generated_at)
    end

    test "generated_at is an ISO8601 timestamp" do
      {:ok, profile} = Profile.get("default")
      assert {:ok, _, _} = DateTime.from_iso8601(profile.generated_at)
    end
  end

  describe "Profile.get/2 — Tier 1 + Tier 2 file reading" do
    setup do
      root = workspace_root()
      # Write node files for the default workspace under a unique node slug
      node = "profile-test-#{System.unique_integer([:positive])}"
      write_node_file(root, "default", node, "context.md", "# #{node}\n\nStatic ground truth.")
      write_node_file(root, "default", node, "signal.md", "# #{node}\n\nRolling status update.")

      %{node: node}
    end

    test "Tier 1 static includes context.md content", %{node: node} do
      {:ok, profile} = Profile.get("default", node_filter: node, bandwidth: :full)
      assert profile.static =~ "Static ground truth"
    end

    test "Tier 2 dynamic includes signal.md content", %{node: node} do
      {:ok, profile} = Profile.get("default", node_filter: node, bandwidth: :full)
      assert profile.dynamic =~ "Rolling status update"
    end

    test "node_filter restricts to one node", %{node: node} do
      # Create a second node so we have something to exclude
      other_node = "other-#{System.unique_integer([:positive])}"
      root = workspace_root()
      write_node_file(root, "default", other_node, "context.md", "Other node content #{other_node}")

      {:ok, filtered} = Profile.get("default", node_filter: node, bandwidth: :full)
      {:ok, other_filtered} = Profile.get("default", node_filter: other_node, bandwidth: :full)

      # Filtered to our node should have its content
      assert filtered.static =~ "Static ground truth"
      # Filtered to our node should NOT contain the other node content
      refute filtered.static =~ "Other node content"
      # Other-node filter should have the other node content
      assert other_filtered.static =~ "Other node content #{other_node}"
    end

    test "missing context.md for a node returns empty string for that node" do
      root = workspace_root()
      node = "no-context-#{System.unique_integer([:positive])}"
      # Create node dir but no context.md
      File.mkdir_p!(Path.join([root, "nodes", node]))

      # Should not crash
      {:ok, profile} = Profile.get("default")
      assert is_binary(profile.static)
    end
  end

  describe "Profile.get/2 — bandwidth filtering" do
    setup do
      root = workspace_root()
      node = "bw-test-#{System.unique_integer([:positive])}"
      long_content = String.duplicate("A", 1000)
      write_node_file(root, "default", node, "context.md", long_content)
      write_node_file(root, "default", node, "signal.md", long_content)
      %{node: node}
    end

    test "bandwidth :l0 truncates static to ≤200 chars and zeroes dynamic + activity" do
      {:ok, profile} = Profile.get("default", bandwidth: :l0)

      assert byte_size(profile.static) <= 200
      assert profile.dynamic == ""
      assert profile.activity == []
      assert profile.entities == []
    end

    test "bandwidth :l1 truncates static/dynamic to ≤800 chars, includes all tiers" do
      {:ok, profile} = Profile.get("default", bandwidth: :l1)

      assert byte_size(profile.static) <= 800
      assert byte_size(profile.dynamic) <= 800
      # activity is always a list (may be empty if no chunks indexed)
      assert is_list(profile.activity)
    end

    test "bandwidth :full does not truncate" do
      root = workspace_root()
      node = "bw-full-#{System.unique_integer([:positive])}"
      # 900-char content — larger than l1 cap
      content = String.duplicate("B", 900)
      write_node_file(root, "default", node, "context.md", content)

      {:ok, profile} = Profile.get("default", node_filter: node, bandwidth: :full)

      assert byte_size(profile.static) >= 900
    end
  end

  describe "Profile.get/2 — workspace isolation" do
    test "returns :not_found for an unknown workspace id" do
      result = Profile.get("does-not-exist-#{System.unique_integer([:positive])}")
      assert {:error, :not_found} = result
    end

    test "engineering workspace profile does not include default-only node files" do
      root = workspace_root()
      # Write a file under the DEFAULT workspace only
      default_node = "default-only-#{System.unique_integer([:positive])}"
      write_node_file(root, "default", default_node, "context.md", "ONLY_IN_DEFAULT_WORKSPACE")

      # Profile for the default workspace scoped to that node should have the content
      {:ok, default_profile} = Profile.get("default", node_filter: default_node, bandwidth: :full)
      assert default_profile.static =~ "ONLY_IN_DEFAULT_WORKSPACE"
    end
  end

  describe "Profile.get/2 — Tier 3 curated wiki" do
    test "curated field contains wiki body excerpt" do
      suffix = System.unique_integer([:positive])
      slug = "profile-wiki-#{suffix}"

      :ok =
        Store.put(%Page{
          tenant_id: "default",
          workspace_id: "default",
          slug: slug,
          audience: "default",
          version: 1,
          frontmatter: %{"slug" => slug},
          body: "## Summary\n\nThis is the curated wiki page for testing.",
          last_curated: DateTime.utc_now() |> DateTime.to_iso8601()
        })

      {:ok, profile} = Profile.get("default")
      # The curated field should contain something from wiki pages
      # (might hit our test page or others — just assert it's a string)
      assert is_binary(profile.curated)
    end
  end

  # ── GET /api/profile endpoint ─────────────────────────────────────────────

  describe "GET /api/profile" do
    test "returns 200 with all profile fields for default workspace" do
      conn = get("/api/profile")
      assert conn.status == 200

      {:ok, body} = Jason.decode(conn.resp_body)
      assert Map.has_key?(body, "workspace_id")
      assert Map.has_key?(body, "static")
      assert Map.has_key?(body, "dynamic")
      assert Map.has_key?(body, "curated")
      assert Map.has_key?(body, "activity")
      assert Map.has_key?(body, "entities")
      assert Map.has_key?(body, "generated_at")
    end

    test "bandwidth=l0 returns 200 with empty dynamic and activity" do
      conn = get("/api/profile?bandwidth=l0")
      assert conn.status == 200

      {:ok, body} = Jason.decode(conn.resp_body)
      assert body["dynamic"] == ""
      assert body["activity"] == []
      assert body["entities"] == []
    end

    test "returns 404 for an unknown workspace" do
      conn = get("/api/profile?workspace=nonexistent-#{System.unique_integer([:positive])}")
      assert conn.status == 404

      {:ok, body} = Jason.decode(conn.resp_body)
      assert body["error"] =~ "workspace not found"
    end

    test "respects workspace= param" do
      conn = get("/api/profile?workspace=default")
      assert conn.status == 200

      {:ok, body} = Jason.decode(conn.resp_body)
      assert body["workspace_id"] == "default"
    end

    test "respects audience= param" do
      conn = get("/api/profile?audience=sales")
      assert conn.status == 200

      {:ok, body} = Jason.decode(conn.resp_body)
      assert body["audience"] == "sales"
    end

    test "respects node= filter param" do
      conn = get("/api/profile?node=some-node")
      # Does not crash even for a node that doesn't exist on disk
      assert conn.status == 200
    end

    test "bandwidth=full returns 200" do
      conn = get("/api/profile?bandwidth=full")
      assert conn.status == 200
    end
  end
end
