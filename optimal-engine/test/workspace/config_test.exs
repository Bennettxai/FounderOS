defmodule OptimalEngine.Workspace.ConfigTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Workspace.Config

  # Each test gets its own isolated temp directory so tests can run concurrently.
  setup do
    root = Path.join(System.tmp_dir!(), "config_test_#{System.unique_integer([:positive])}")
    File.mkdir_p!(root)
    on_exit(fn -> File.rm_rf!(root) end)
    %{root: root}
  end

  # ── defaults/0 ──────────────────────────────────────────────────────────

  describe "defaults/0" do
    test "returns a map with all required top-level sections" do
      d = Config.defaults()
      assert is_map(d)

      for section <- [:visualizations, :profile, :grep, :contradictions, :retention] do
        assert Map.has_key?(d, section), "missing section: #{section}"
      end
    end

    test "visualizations.enabled contains expected chart types" do
      %{visualizations: %{enabled: enabled}} = Config.defaults()
      assert "timeline" in enabled
      assert "heatmap" in enabled
      assert "graph" in enabled
      assert "contradictions" in enabled
    end

    test "retention.default_ttl_days is nil by default" do
      assert Config.defaults().retention.default_ttl_days == nil
    end
  end

  # ── get/2 — no file ─────────────────────────────────────────────────────

  describe "get/2 when no config.yaml exists" do
    test "returns {:ok, defaults} without error", %{root: root} do
      assert {:ok, cfg} = Config.get("default", root)
      assert cfg == Config.defaults()
    end

    test "works for a named (non-default) workspace slug", %{root: root} do
      ws_dir = Path.join(root, "engineering")
      File.mkdir_p!(ws_dir)
      assert {:ok, cfg} = Config.get("engineering", root)
      assert cfg.grep.max_results == 25
    end
  end

  # ── put/3 + get/2 round-trip ────────────────────────────────────────────

  describe "put/3 followed by get/2" do
    test "persists a top-level section override", %{root: root} do
      :ok = Config.put("default", %{grep: %{max_results: 99}}, root)
      {:ok, cfg} = Config.get("default", root)

      assert cfg.grep.max_results == 99
    end

    test "deep-merges so untouched keys survive", %{root: root} do
      :ok = Config.put("default", %{grep: %{max_results: 50}}, root)
      {:ok, cfg} = Config.get("default", root)

      # Untouched grep sub-keys must still have default values.
      assert cfg.grep.default_scale == "paragraph"
      assert cfg.grep.literal_threshold == 0.8

      # Other top-level sections are unaffected.
      assert cfg.profile.recent_chunks_limit == 20
    end

    test "successive puts accumulate via deep-merge", %{root: root} do
      :ok = Config.put("default", %{grep: %{max_results: 10}}, root)
      :ok = Config.put("default", %{profile: %{default_audience: "exec"}}, root)

      {:ok, cfg} = Config.get("default", root)
      assert cfg.grep.max_results == 10
      assert cfg.profile.default_audience == "exec"
    end

    test "boolean false round-trips correctly", %{root: root} do
      :ok = Config.put("default", %{profile: %{include_archived: false}}, root)
      {:ok, cfg} = Config.get("default", root)
      assert cfg.profile.include_archived == false
    end

    test "nil value round-trips as null", %{root: root} do
      :ok = Config.put("default", %{retention: %{default_ttl_days: nil}}, root)
      {:ok, cfg} = Config.get("default", root)
      assert cfg.retention.default_ttl_days == nil
    end

    test "creates .optimal/ directory when absent", %{root: root} do
      # "default" workspace lives at root itself, so config is at <root>/.optimal/config.yaml
      refute File.exists?(Path.join([root, ".optimal", "config.yaml"]))
      :ok = Config.put("default", %{}, root)
      assert File.exists?(Path.join([root, ".optimal", "config.yaml"]))
    end
  end

  # ── get_section/4 ────────────────────────────────────────────────────────

  describe "get_section/4" do
    test "returns the requested section from defaults when no config file", %{root: root} do
      profile = Config.get_section("default", :profile, %{}, root)
      assert profile.default_audience == "default"
      assert profile.recent_chunks_limit == 20
    end

    test "returns custom default when section key is absent", %{root: root} do
      result = Config.get_section("default", :nonexistent_section, :my_default, root)
      assert result == :my_default
    end

    test "reflects on-disk value after a put", %{root: root} do
      :ok = Config.put("default", %{contradictions: %{auto_dismiss_days: 7}}, root)

      section = Config.get_section("default", :contradictions, %{}, root)
      assert section.auto_dismiss_days == 7
    end
  end

  # ── default workspace path (root itself) ─────────────────────────────────

  describe "default workspace (slug == 'default')" do
    test "config file lives at <root>/.optimal/config.yaml", %{root: root} do
      :ok = Config.put("default", %{}, root)
      assert File.exists?(Path.join([root, ".optimal", "config.yaml"]))
    end
  end

  # ── named workspace path ─────────────────────────────────────────────────

  describe "named workspace (slug != 'default')" do
    test "config file lives at <root>/<slug>/.optimal/config.yaml", %{root: root} do
      ws_dir = Path.join(root, "sales")
      File.mkdir_p!(ws_dir)
      :ok = Config.put("sales", %{}, root)
      assert File.exists?(Path.join([root, "sales", ".optimal", "config.yaml"]))
    end
  end

  # ── to_yaml/1 — YAML serialiser sanity ───────────────────────────────────

  describe "to_yaml/1 (internal serialiser)" do
    test "produces valid YAML that round-trips through YamlElixir" do
      data = %{
        foo: %{bar: "baz", num: 42, flag: true, nothing: nil},
        list: ["a", "b", "c"]
      }

      yaml = Config.to_yaml(data)
      assert is_binary(yaml)

      {:ok, parsed} = YamlElixir.read_from_string(yaml)
      assert parsed["foo"]["bar"] == "baz"
      assert parsed["foo"]["num"] == 42
      assert parsed["foo"]["flag"] == true
      assert parsed["foo"]["nothing"] == nil
      assert parsed["list"] == ["a", "b", "c"]
    end
  end
end
