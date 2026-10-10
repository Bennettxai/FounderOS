defmodule OptimalEngine.Retrieval.DeliverTest do
  use ExUnit.Case, async: true

  alias OptimalEngine.Retrieval.{Deliver, Receiver}
  alias OptimalEngine.Wiki.Page

  defp plain_receiver, do: Receiver.new(%{format: :plain})
  defp md_receiver, do: Receiver.new(%{format: :markdown})
  defp claude_receiver, do: Receiver.new(%{format: :claude})
  defp openai_receiver, do: Receiver.new(%{format: :openai})

  describe "render_chunks/2" do
    test "plain format cites each chunk by numeric [n]" do
      chunks = [
        %{title: "A", content: "first fact.", uri: "optimal://a"},
        %{title: "B", content: "second fact.", uri: "optimal://b"}
      ]

      env = Deliver.render_chunks(chunks, plain_receiver())
      assert env.format == :plain
      assert env.body =~ "[1]"
      assert env.body =~ "[2]"
      assert "optimal://a" in env.sources
      assert "optimal://b" in env.sources
    end

    test "markdown format uses footnote-style citations" do
      chunks = [%{title: "A", content: "fact.", uri: "optimal://a"}]
      env = Deliver.render_chunks(chunks, md_receiver())

      assert env.body =~ "[^1]"
      assert env.body =~ "[^1]: optimal://a"
    end

    test "claude format wraps in <context> with <document> entries" do
      chunks = [%{title: "A", content: "fact.", uri: "optimal://a"}]
      env = Deliver.render_chunks(chunks, claude_receiver())

      assert String.starts_with?(env.body, "<context>")
      assert String.ends_with?(env.body, "</context>")
      assert env.body =~ ~s(source="optimal://a")
    end

    test "openai format emits a JSON message array" do
      chunks = [%{title: "A", content: "fact.", uri: "optimal://a"}]
      env = Deliver.render_chunks(chunks, openai_receiver())

      assert {:ok, messages} = Jason.decode(env.body)
      assert is_list(messages)
      assert Enum.any?(messages, fn m -> m["role"] == "system" end)
    end

    test "chunks without a uri are emitted without citations" do
      chunks = [%{title: "A", content: "no source"}]
      env = Deliver.render_chunks(chunks, plain_receiver())
      assert env.body =~ "no source"
      assert env.sources == []
    end
  end

  describe "render_wiki/3" do
    test "renders a wiki page with resolver-provided content" do
      page = %Page{
        tenant_id: "default",
        slug: "t",
        audience: "default",
        version: 1,
        frontmatter: %{},
        body: "Claim {{cite: optimal://a}}. Include {{include: optimal://b}}."
      }

      resolver = fn
        %{verb: :cite, argument: uri}, _opts -> {:ok, "", %{uri: uri}}
        %{verb: :include, argument: _}, _opts -> {:ok, "(included content)", %{}}
        _, _ -> {:error, :unresolved}
      end

      env = Deliver.render_wiki(page, md_receiver(), resolver)
      assert env.body =~ "(included content)"
      assert env.body =~ "[^1]"
      assert "optimal://a" in env.sources
    end
  end

  describe "empty/2" do
    test "returns a receiver-formatted empty envelope" do
      env = Deliver.empty(md_receiver(), "Nothing.")
      assert env.body =~ "Nothing."
      assert env.format == :markdown
      assert env.sources == []
    end

    test "openai empty is still valid JSON" do
      env = Deliver.empty(openai_receiver())
      assert {:ok, _} = Jason.decode(env.body)
    end
  end
end
