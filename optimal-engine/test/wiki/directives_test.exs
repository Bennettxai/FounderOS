defmodule OptimalEngine.Wiki.DirectivesTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Wiki.Directives

  describe "parse/1" do
    test "parses all whitelisted verbs" do
      body = """
      Here is a cite {{cite: optimal://nodes/a/1.md}}.
      Here is an include {{include: optimal://nodes/b/2.md}}.
      Here is an expand {{expand: topic-slug}}.
      Here is a search {{search: "pricing"}}.
      Here is a table {{table: optimal://data.csv#col=val}}.
      Here is a trace {{trace: Alice}}.
      Here is a recent {{recent: node=project-platform-launch limit=5}}.
      Here is a [[wikilink]].
      """

      {:ok, directives} = Directives.parse(body)

      verbs = Enum.map(directives, & &1.verb) |> Enum.sort()
      assert :cite in verbs
      assert :include in verbs
      assert :expand in verbs
      assert :search in verbs
      assert :table in verbs
      assert :trace in verbs
      assert :recent in verbs
      assert :wikilink in verbs
    end

    test "preserves position (offset + length)" do
      body = "Prefix {{cite: uri}} suffix."
      {:ok, [d]} = Directives.parse(body)
      assert d.offset == 7
      # {{cite: uri}} is 13 bytes
      assert d.length == byte_size("{{cite: uri}}")
      assert d.raw == "{{cite: uri}}"
    end

    test "parses key=value options on recent" do
      body = "{{recent: node=project-platform-launch limit=5}}"
      {:ok, [d]} = Directives.parse(body)
      assert d.verb == :recent
      assert d.options["node"] == "project-platform-launch"
      assert d.options["limit"] == "5"
    end

    test "returns empty list for body with no directives" do
      assert {:ok, []} = Directives.parse("just plain text")
    end
  end

  describe "all_verbs_whitelisted?/1" do
    test "true when body uses only whitelisted verbs + wikilinks" do
      body = "{{cite: x}} and [[y]]"
      assert Directives.all_verbs_whitelisted?(body)
    end

    test "false when body contains an unknown verb" do
      body = "{{delete: all}} is suspicious"
      refute Directives.all_verbs_whitelisted?(body)
    end
  end

  describe "render/3" do
    defp resolver_ok(%{verb: :cite, argument: uri}, _opts), do: {:ok, "", %{uri: uri}}
    defp resolver_ok(%{verb: :include, argument: uri}, _opts), do: {:ok, "(included #{uri})", %{}}
    defp resolver_ok(%{verb: :wikilink, argument: slug}, _opts), do: {:ok, "→#{slug}", %{}}
    defp resolver_ok(_, _opts), do: {:ok, "(resolved)", %{}}

    test "markdown format produces footnote citations" do
      body = "Claim {{cite: optimal://a}}. Another {{cite: optimal://b}}."

      {rendered, warnings} = Directives.render(body, &resolver_ok/2, format: :markdown)

      assert warnings == []
      assert rendered =~ "[^1]"
      assert rendered =~ "[^2]"
      assert rendered =~ "[^1]: optimal://a"
      assert rendered =~ "[^2]: optimal://b"
    end

    test "plain format uses numeric [n] citations + Sources footer" do
      body = "Claim {{cite: optimal://a}}."
      {rendered, _} = Directives.render(body, &resolver_ok/2, format: :plain)
      assert rendered =~ "[1]"
      assert rendered =~ "Sources:"
      assert rendered =~ "[1] optimal://a"
    end

    test "claude format wraps in XML context tags" do
      body = "Claim {{cite: optimal://a}}."
      {rendered, _} = Directives.render(body, &resolver_ok/2, format: :claude)
      assert String.starts_with?(rendered, "<context>")
      assert String.ends_with?(rendered, "</context>")
      assert rendered =~ ~s(source="optimal://a")
    end

    test "include directive expands inline" do
      body = "Before {{include: optimal://doc.md}} after."
      {rendered, _} = Directives.render(body, &resolver_ok/2, format: :plain)
      assert rendered =~ "(included optimal://doc.md)"
    end

    test "resolver errors produce fallback rendering + warning list" do
      resolver = fn
        %{verb: :cite}, _ -> {:error, :not_found}
        _, _ -> {:ok, "", %{}}
      end

      body = "Broken cite {{cite: optimal://missing}}."
      {rendered, warnings} = Directives.render(body, resolver, format: :plain)

      assert warnings != []
      assert rendered =~ "⟨{{cite: optimal://missing}}⟩"
    end
  end
end
