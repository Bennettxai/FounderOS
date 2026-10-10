defmodule OptimalEngine.Compliance.RetentionTest do
  use ExUnit.Case, async: false

  alias OptimalEngine.Compliance.Retention
  alias OptimalEngine.Store

  setup do
    # Clean the retention_policies table scope we care about
    suffix = System.unique_integer([:positive])
    {:ok, %{suffix: suffix}}
  end

  test "sweep/1 runs without error when no policies exist", %{suffix: suffix} do
    # Use a unique tenant so no prior rows interfere
    tenant = "tenant-rsweep-#{suffix}"

    assert {:ok, result} = Retention.sweep(tenant_id: tenant)
    assert result.policies_evaluated >= 0
    assert result.actions_taken == 0
  end

  test "dry_run reports evaluations but takes no action", %{suffix: suffix} do
    tenant = "tenant-rsweep-dry-#{suffix}"

    # Seed a tenant-wide TTL=1 policy then an old row
    Store.raw_query(
      """
      INSERT INTO retention_policies (tenant_id, scope_type, scope_value, ttl_days, action)
      VALUES (?1, 'tenant', NULL, 1, 'archive')
      """,
      [tenant]
    )

    ctx_id = "rsweep-#{suffix}"

    Store.raw_query(
      """
      INSERT INTO contexts (id, tenant_id, uri, title, content, genre, node, created_at)
      VALUES (?1, ?2, ?3, 't', 'body', 'note', 'person-operator', '2020-01-01T00:00:00Z')
      """,
      [ctx_id, tenant, "optimal://rsweep/#{ctx_id}"]
    )

    {:ok, result} = Retention.sweep(tenant_id: tenant, dry_run: true)
    assert result.policies_evaluated >= 1
    assert result.actions_taken == 0

    # Row still there
    {:ok, [[count]]} = Store.raw_query("SELECT COUNT(*) FROM contexts WHERE id = ?1", [ctx_id])
    assert count == 1
  end
end
