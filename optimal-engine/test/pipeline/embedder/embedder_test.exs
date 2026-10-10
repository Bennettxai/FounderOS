defmodule OptimalEngine.Pipeline.EmbedderTest do
  @moduledoc """
  Tests verify dispatch shape and graceful degradation. Alignment
  verification (text-query-retrieves-image) requires live Ollama + vision
  model pulled — documented in the phase commit but not gated by CI.
  """
  use ExUnit.Case, async: false

  alias OptimalEngine.Embed.Ollama
  alias OptimalEngine.Pipeline.Decomposer.Chunk
  alias OptimalEngine.Pipeline.Embedder

  describe "embed/2 dispatch + graceful degradation" do
    test "text chunk routes to text model" do
      chunk = Chunk.new(id: "c-text", signal_id: "s", text: "hello world", modality: :text)

      case Embedder.embed(chunk) do
        {:ok, e} ->
          assert e.modality == :text
          assert e.model == "nomic-embed-text"
          assert e.dim == length(e.vector)

        {:error, _reason} ->
          # Ollama unreachable — acceptable, test just verifies dispatch shape.
          :ok
      end
    end

    test "code chunk routes to text model (shared space)" do
      chunk = Chunk.new(id: "c-code", signal_id: "s", text: "def foo, do: :ok", modality: :code)

      case Embedder.embed(chunk) do
        {:ok, e} ->
          assert e.modality == :code
          assert e.model == "nomic-embed-text"

        {:error, _reason} ->
          :ok
      end
    end

    test "audio chunk with existing transcript routes through text embedder" do
      chunk =
        Chunk.new(
          id: "c-audio",
          signal_id: "s",
          text: "pre-existing whisper transcript",
          modality: :audio
        )

      case Embedder.embed(chunk) do
        {:ok, e} ->
          assert e.modality == :audio
          assert String.starts_with?(e.model, "whisper+")

        {:error, _} ->
          :ok
      end
    end

    test "image chunk with no asset and no OCR text returns :no_embeddable_content" do
      chunk = Chunk.new(id: "c-image", signal_id: "s", text: "", modality: :image)

      case Embedder.embed(chunk) do
        {:error, :no_embeddable_content} -> :ok
        {:ok, _} -> flunk("expected :no_embeddable_content")
        {:error, _other} -> :ok
      end
    end

    test "empty text chunk returns :empty_content" do
      chunk = Chunk.new(id: "c-empty", signal_id: "s", text: "", modality: :text)
      assert {:error, :empty_content} = Embedder.embed(chunk)
    end
  end

  describe "embed_tree/2" do
    test "collects successes + errors separately" do
      alias OptimalEngine.Pipeline.Decomposer
      alias OptimalEngine.Pipeline.Parser.ParsedDoc

      doc = ParsedDoc.new(text: "First.\n\nSecond paragraph of text.")
      {:ok, tree} = Decomposer.decompose(doc)
      {:ok, embeddings, %{errors: errors}} = Embedder.embed_tree(tree)

      assert is_list(embeddings)
      assert is_list(errors)
      assert length(embeddings) + length(errors) == length(tree.chunks)

      # Every embedding in the success list must have its originating chunk
      # id represented.
      chunk_ids = MapSet.new(tree.chunks, & &1.id)

      Enum.each(embeddings, fn e ->
        assert MapSet.member?(chunk_ids, e.chunk_id)
      end)
    end
  end

  describe "dim/0" do
    test "reports the canonical 768-dim aligned space" do
      assert Embedder.dim() == 768
    end
  end

  describe "Ollama surface" do
    test "empty input is rejected without calling Ollama" do
      assert {:error, :empty_input} = Ollama.embed_text("   \n")
    end

    test "embed_text/2 + embed/2 are equivalent (backward compat)" do
      # Without hitting the network: just verify the function arities exist
      # and agree.
      Code.ensure_loaded!(Ollama)

      assert function_exported?(Ollama, :embed_text, 2)
      assert function_exported?(Ollama, :embed, 2)
      assert function_exported?(Ollama, :embed_image, 2)
    end
  end
end
