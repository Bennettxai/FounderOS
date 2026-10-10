defmodule Mix.Tasks.Optimal.Cluster do
  @shortdoc "Inspect or rebuild the tenant's clusters"

  @moduledoc """
  Clusters are the wide-pass grouping of chunks by theme — Phase 6 of the
  pipeline. This task lists existing clusters and their members, or
  triggers a full rebuild from scratch.

  ## Usage

      mix optimal.cluster                          list clusters in the default tenant
      mix optimal.cluster --tenant sample-corp       list clusters in a specific tenant
      mix optimal.cluster rebuild                  full rebuild (clears + re-assigns)
      mix optimal.cluster rebuild --threshold 0.7  custom similarity threshold
  """

  use Mix.Task

  alias OptimalEngine.Pipeline.Clusterer
  alias OptimalEngine.Store
  alias OptimalEngine.Tenancy.Tenant

  @impl Mix.Task
  def run(args) do
    Mix.Task.run("app.start")

    {opts, positional, _} =
      OptionParser.parse(args, strict: [tenant: :string, threshold: :float])

    tenant_id = Keyword.get(opts, :tenant, Tenant.default_id())

    case positional do
      ["rebuild" | _] ->
        threshold = Keyword.get(opts, :threshold, 0.65)
        rebuild(tenant_id, threshold)

      _ ->
        list(tenant_id)
    end
  end

  defp list(tenant_id) do
    case Store.raw_query(
           """
           SELECT id, theme, intent_dominant, member_count, updated_at
           FROM clusters
           WHERE tenant_id = ?1
           ORDER BY member_count DESC, updated_at DESC
           """,
           [tenant_id]
         ) do
      {:ok, []} ->
        IO.puts("\n  No clusters for tenant #{tenant_id}. Run `mix optimal.cluster rebuild`.\n")

      {:ok, rows} ->
        IO.puts("")
        IO.puts("  Clusters — tenant: #{tenant_id}  (#{length(rows)} total)")
        IO.puts("  " <> String.duplicate("─", 70))

        Enum.each(rows, fn [id, theme, intent_dom, count, updated] ->
          intent_str = if is_nil(intent_dom), do: "?", else: to_string(intent_dom)

          IO.puts(
            "    #{String.pad_trailing(id, 30)} [#{String.pad_trailing(intent_str, 18)}] n=#{count}"
          )

          IO.puts("      #{theme}  (updated #{updated})")
        end)

        IO.puts("")

      _ ->
        IO.puts("  (clusters unavailable)")
    end
  end

  defp rebuild(tenant_id, threshold) do
    IO.puts("")
    IO.puts("  Rebuilding clusters for tenant #{tenant_id} (threshold #{threshold})…")

    case Clusterer.rebuild(tenant_id, threshold: threshold) do
      {:ok, %{clusters: c, members: m}} ->
        IO.puts("  ✓ #{c} clusters, #{m} memberships")

      {:error, reason} ->
        IO.puts("  ✗ #{inspect(reason)}")
        System.halt(1)
    end

    IO.puts("")
  end
end
