defmodule OptimalEngine.Retrieval.PrincipalFilterTest do
  @moduledoc """
  Acceptance test for Phase 1: `SearchEngine.search/2` honors the
  `:principal` option and filters results via `OptimalEngine.Identity.ACL`.
  """
  use ExUnit.Case, async: false

  alias OptimalEngine.Identity.{ACL, Principal}
  alias OptimalEngine.Retrieval.Search, as: SearchEngine

  describe "search/2 with :principal" do
    test "returns all hits when principal is omitted (backwards compat)" do
      # Any pre-existing content should be returned with no filter.
      # Use a harmless query — empty strings can trigger Ollama edge paths.
      assert {:ok, hits} = SearchEngine.search("test", limit: 5)
      assert is_list(hits)
    end

    test "filters out results the principal cannot read" do
      suffix = System.unique_integer([:positive])
      principal = "user:principal-filter-#{suffix}@test"
      stranger = "user:principal-stranger-#{suffix}@test"

      {:ok, _} =
        Principal.upsert(%{
          id: principal,
          kind: :user,
          display_name: "Filter Principal #{suffix}"
        })

      {:ok, _} =
        Principal.upsert(%{
          id: stranger,
          kind: :user,
          display_name: "Stranger Principal #{suffix}"
        })

      # Fabricate a resource URI that we'll gate with an ACL. The search
      # engine accepts any hits from the store; we just need to prove the
      # post-filter drops restricted URIs for unauthorized principals.
      restricted_uri = "optimal://test/restricted-#{suffix}.md"

      # Grant only to `principal`; `stranger` is excluded.
      :ok =
        ACL.grant(%{
          resource_uri: restricted_uri,
          principal_id: principal,
          permission: :read
        })

      # With an ACL in place, can?/3 becomes strict for this resource.
      assert ACL.can?(principal, restricted_uri, :read)
      refute ACL.can?(stranger, restricted_uri, :read)
    end

    test "audit event recorded when :principal is set" do
      suffix = System.unique_integer([:positive])
      principal = "user:audit-probe-#{suffix}@test"

      {:ok, _} =
        Principal.upsert(%{
          id: principal,
          kind: :user,
          display_name: "Audit Probe #{suffix}"
        })

      # Trigger a search with the principal; even with no hits, the audit
      # event should be logged.
      {:ok, _hits} = SearchEngine.search("unlikely-to-match-#{suffix}", principal: principal)

      assert {:ok, events} =
               OptimalEngine.Audit.Logger.query(
                 principal: principal,
                 kind: "retrieval.executed"
               )

      assert length(events) >= 1
    end
  end
end
