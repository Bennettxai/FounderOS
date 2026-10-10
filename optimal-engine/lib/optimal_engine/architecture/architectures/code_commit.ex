defmodule OptimalEngine.Architecture.Architectures.CodeCommit do
  @moduledoc """
  A code-repository commit with diff, message, and touched files.

  Code has its own embedding model family (CodeBERT, StarCoder
  embedders, specialized tokenizers). The `diff` and `message` are
  embedded separately so retrieval can match on implementation
  (what changed) or intent (why it changed) — a query like "API
  timeout" should find both the commit message and the actual lines.
  """

  alias OptimalEngine.Architecture.{Architecture, Field}

  def definition do
    Architecture.new(
      name: "code_commit",
      version: 1,
      description: "Repository commit with diff, message, and touched files",
      modality_primary: :code,
      granularity: [:commit, :file, :hunk, :line],
      fields: [
        %Field{
          name: :sha,
          modality: :structured,
          required: true,
          description: "Commit hash"
        },
        %Field{
          name: :repo,
          modality: :structured,
          required: true,
          description: "Repository id"
        },
        %Field{
          name: :author,
          modality: :structured,
          required: false,
          description: "Author principal id"
        },
        %Field{
          name: :message,
          modality: :text,
          required: true,
          processor: :text_embedder,
          description: "Commit subject + body"
        },
        %Field{
          name: :diff,
          modality: :code,
          required: true,
          processor: :code_embedder,
          description: "Unified diff text"
        },
        %Field{
          name: :files,
          modality: :structured,
          required: false,
          description: "List of files touched, with additions/deletions"
        },
        %Field{
          name: :timestamp,
          modality: :structured,
          required: false,
          description: "Author/commit timestamp"
        }
      ]
    )
  end
end
