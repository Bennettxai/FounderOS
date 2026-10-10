defmodule OptimalEngine.Architecture.Architectures.StructuredRecord do
  @moduledoc """
  A structured business record — invoice, ticket, contact, CRM deal,
  clinical encounter. Key-value data with typed fields and references
  to other records.

  The `payload` field is the raw JSON; separate fields carry the
  indexable projections (`title`, `status`, `amount`, `parties`).
  This lets classical search match on `status='paid'` while semantic
  search matches on the free-text `notes`.
  """

  alias OptimalEngine.Architecture.{Architecture, Field}

  def definition do
    Architecture.new(
      name: "structured_record",
      version: 1,
      description: "Structured business record (invoice, ticket, deal, …)",
      modality_primary: :structured,
      granularity: [:record, :field],
      fields: [
        %Field{
          name: :kind,
          modality: :structured,
          required: true,
          description: "Record type: :invoice | :ticket | :deal | :contact | …"
        },
        %Field{
          name: :external_id,
          modality: :structured,
          required: true,
          description: "Stable id from the source system"
        },
        %Field{
          name: :payload,
          modality: :structured,
          required: true,
          description: "Raw record JSON"
        },
        %Field{
          name: :title,
          modality: :text,
          required: false,
          processor: :text_embedder,
          description: "Human-readable label for retrieval"
        },
        %Field{
          name: :notes,
          modality: :text,
          required: false,
          processor: :text_embedder,
          description: "Free-text notes attached to the record"
        },
        %Field{
          name: :status,
          modality: :structured,
          required: false,
          description: "Lifecycle state (open/closed/paid/…)"
        },
        %Field{
          name: :parties,
          modality: :structured,
          required: false,
          description: "References to related principals / orgs"
        }
      ]
    )
  end
end
