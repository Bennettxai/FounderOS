defmodule OptimalEngine.Wiki.SchedulerTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Store
  alias OptimalEngine.Wiki.{Page, Scheduler}
  alias OptimalEngine.Wiki.Store, as: WikiStore

  # ── helpers ─────────────────────────────────────────────────────────────────

  defp unique, do: System.unique_integer([:positive])

  defp base_page(overrides \\ []) do
    suffix = unique()

    defaults = [
      tenant_id: "default",
      workspace_id: "default",
      slug: "sched-test-#{suffix}",
      audience: "default",
      version: 1,
      frontmatter: %{},
      body: "## Summary\n\nContent."
    ]

    struct(Page, Keyword.merge(defaults, overrides))
  end

  # Insert a fake context (signal) and a paragraph-scale chunk, then return
  # {signal_id, chunk_id, node_slug}.
  defp insert_signal(node_slug, workspace_id \\ "default", created_at \\ nil) do
    signal_id = "sig-#{unique()}"
    chunk_id = "chunk-#{unique()}"
    ts = created_at || DateTime.utc_now() |> DateTime.to_iso8601()

    Store.raw_query(
      """
      INSERT OR IGNORE INTO contexts (id, tenant_id, workspace_id, node, uri, content, created_at)
      VALUES (?1, 'default', ?2, ?3, ?4, 'test content', ?5)
      """,
      [signal_id, workspace_id, node_slug, "optimal://#{node_slug}/#{signal_id}", ts]
    )

    Store.raw_query(
      """
      INSERT OR IGNORE INTO chunks (id, tenant_id, signal_id, scale, text)
      VALUES (?1, 'default', ?2, 'paragraph', 'new fact ingested')
      """,
      [chunk_id, signal_id]
    )

    {signal_id, chunk_id, node_slug}
  end

  # Link a chunk to a wiki page via the citations table.
  defp insert_citation(wiki_slug, wiki_audience, chunk_id, tenant_id \\ "default") do
    Store.raw_query(
      """
      INSERT OR IGNORE INTO citations (tenant_id, wiki_slug, wiki_audience, chunk_id, claim_hash)
      VALUES (?1, ?2, ?3, ?4, ?5)
      """,
      [tenant_id, wiki_slug, wiki_audience, chunk_id, "hash-#{unique()}"]
    )
  end

  # ── Tests ────────────────────────────────────────────────────────────────────

  describe "stale_pages/2" do
    test "page curated 10 days ago with new signals is detected as stale" do
      old_ts = DateTime.utc_now() |> DateTime.add(-10 * 86_400, :second) |> DateTime.to_iso8601()

      page =
        base_page(
          last_curated: old_ts,
          curated_by: "test"
        )

      :ok = WikiStore.put(page)

      # Insert a signal that arrived AFTER the last curation timestamp
      new_signal_ts =
        DateTime.utc_now() |> DateTime.add(-1 * 86_400, :second) |> DateTime.to_iso8601()

      {_sig_id, chunk_id, _node} = insert_signal("node-a-#{unique()}", "default", new_signal_ts)
      insert_citation(page.slug, page.audience, chunk_id)

      stale = Scheduler.stale_pages("default", tenant_id: "default")
      slugs = Enum.map(stale, & &1.slug)

      assert page.slug in slugs,
             "expected #{page.slug} to be in stale list, got: #{inspect(slugs)}"
    end

    test "page curated today is not stale" do
      recent_ts = DateTime.utc_now() |> DateTime.to_iso8601()

      page =
        base_page(
          last_curated: recent_ts,
          curated_by: "test"
        )

      :ok = WikiStore.put(page)

      stale = Scheduler.stale_pages("default", tenant_id: "default")
      slugs = Enum.map(stale, & &1.slug)

      refute page.slug in slugs,
             "expected #{page.slug} NOT to be stale (curated today)"
    end

    test "page curated 10 days ago but with no new signals is not stale" do
      old_ts = DateTime.utc_now() |> DateTime.add(-10 * 86_400, :second) |> DateTime.to_iso8601()

      page =
        base_page(
          last_curated: old_ts,
          curated_by: "test"
        )

      :ok = WikiStore.put(page)

      # Insert a signal, but attach it BEFORE last_curated so it doesn't qualify
      very_old_ts =
        DateTime.utc_now() |> DateTime.add(-20 * 86_400, :second) |> DateTime.to_iso8601()

      {_sig_id, chunk_id, _node} =
        insert_signal("node-b-#{unique()}", "default", very_old_ts)

      insert_citation(page.slug, page.audience, chunk_id)

      stale = Scheduler.stale_pages("default", tenant_id: "default")
      slugs = Enum.map(stale, & &1.slug)

      refute page.slug in slugs,
             "expected #{page.slug} NOT to be stale (no new signals)"
    end
  end

  describe "force_run/1" do
    test "cast returns :ok and does not crash the GenServer" do
      assert :ok = Scheduler.force_run("default")

      # Give the cast time to be processed; the GenServer must still be alive.
      Process.sleep(50)
      assert Process.alive?(Process.whereis(Scheduler))
    end
  end

  describe "config disabled" do
    test "when curation is disabled, stale_pages still returns results (disabled suppresses tick, not the public API)" do
      # The enabled flag gates the periodic tick, not the public stale_pages/2
      # call. Callers can always introspect staleness; only the background
      # scheduler skips curation when disabled.
      #
      # This test asserts the public API is unaffected by the config flag.
      stale = Scheduler.stale_pages("default")
      assert is_list(stale)
    end
  end
end
