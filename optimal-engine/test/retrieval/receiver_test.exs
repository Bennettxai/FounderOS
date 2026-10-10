defmodule OptimalEngine.Retrieval.ReceiverTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Identity.Principal
  alias OptimalEngine.Retrieval.Receiver

  describe "new/1" do
    test "builds a receiver with defaults" do
      r = Receiver.new(%{})
      assert r.audience == "default"
      assert r.bandwidth == :medium
      assert r.token_budget == 6_000
      assert r.format == :markdown
    end

    test "derives token_budget from bandwidth when not supplied" do
      assert Receiver.new(%{bandwidth: :small}).token_budget == 1_500
      assert Receiver.new(%{bandwidth: :large}).token_budget == 24_000
    end

    test "explicit token_budget overrides the bandwidth derivation" do
      r = Receiver.new(%{bandwidth: :small, token_budget: 9_999})
      assert r.token_budget == 9_999
    end

    test "accepts keyword list too" do
      r = Receiver.new(audience: "sales", bandwidth: :large)
      assert r.audience == "sales"
      assert r.bandwidth == :large
    end
  end

  describe "from_principal_struct/2" do
    test "uses metadata keys when present" do
      p = %Principal{
        id: "user:x@y",
        tenant_id: "default",
        kind: :user,
        display_name: "X",
        metadata: %{
          "audience" => "legal",
          "bandwidth" => "small",
          "format" => "claude",
          "genre" => "spec",
          "locale" => "fr-FR"
        }
      }

      r = Receiver.from_principal_struct(p)
      assert r.id == "user:x@y"
      assert r.audience == "legal"
      assert r.bandwidth == :small
      assert r.format == :claude
      assert r.genre == "spec"
      assert r.locale == "fr-FR"
      assert r.token_budget == 1_500
    end

    test "falls back to kind-specific defaults when metadata missing" do
      p = %Principal{
        id: "agent:bot",
        tenant_id: "default",
        kind: :agent,
        display_name: "Bot",
        metadata: %{}
      }

      r = Receiver.from_principal_struct(p)
      assert r.kind == :agent
      assert r.bandwidth == :large
      assert r.format == :claude
      assert r.genre == "spec"
    end

    test "overrides win over metadata" do
      p = %Principal{
        id: "user:x@y",
        tenant_id: "default",
        kind: :user,
        display_name: "X",
        metadata: %{"audience" => "legal"}
      }

      r = Receiver.from_principal_struct(p, %{audience: "sales"})
      assert r.audience == "sales"
    end
  end

  describe "anonymous/1" do
    test "returns a receiver with unknown kind + default audience" do
      r = Receiver.anonymous()
      assert r.kind == :unknown
      assert r.audience == "default"
    end
  end

  describe "budget_for/1" do
    test "maps bandwidth labels to token counts" do
      assert Receiver.budget_for(:small) == 1_500
      assert Receiver.budget_for(:medium) == 6_000
      assert Receiver.budget_for(:large) == 24_000
    end
  end
end
