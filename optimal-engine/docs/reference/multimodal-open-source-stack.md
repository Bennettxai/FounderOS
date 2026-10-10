# Multimodal Adapter Stack

Optimal Engine is local-first and self-hostable. Multimodal support should
therefore be built around open-source tools that can run locally, while still
allowing hosted adapters when a deployment chooses them.

This page defines the public adapter targets. It does not mean every runtime is
installed by default. The engine should preserve raw evidence first, then route
assets through the best configured local adapter for the modality.

## Core Rule

```text
Raw asset
  -> Source Package
  -> governed Asset row
  -> parser / model adapter
  -> extracted text, structure, transcript, visual facts, embeddings
  -> derived Source Package + pending Claim
  -> Facts / Memory Objects only when reviewed
```

The raw asset remains the evidence. Extracted text, OCR, transcripts, visual
captions, page embeddings, summaries, and model outputs are derived artifacts.

## Adapter Targets

| Capability | Primary targets | Engine use |
| --- | --- | --- |
| Document intelligence | Docling, Marker, olmOCR, Unstructured | PDF/Office/image document parsing, layout, tables, OCR, markdown/JSON export. |
| OCR fallback | Tesseract OCR | Conservative local fallback when richer document intelligence is unavailable. |
| Audio transcription | whisper.cpp, Whisper | Speech-to-text for audio files and extracted video audio. |
| Video processing | FFmpeg | Frame extraction, audio extraction, metadata, transcoding. |
| Visual reasoning | Qwen VL | Image/video/document question answering, visual text reading, localization, and reasoning. |
| Visual document retrieval | ColPali / ColQwen | Page-image retrieval when text extraction alone is not enough. |
| Image-text embeddings | OpenCLIP / SigLIP-style adapters | Cross-modal image/text search and zero-shot tagging. |
| Broad cross-modal embeddings | ImageBind-style adapters | Joint retrieval across image, text, audio, and video-derived signals. |

The code-level registry lives at:

```text
lib/optimal_engine/pipeline/multimodal_tool_registry.ex
```

## Hosted Adapter Options

Hosted services can be useful for teams that want managed quality, throughput,
or a single cross-modal embedding space. They must still use the same adapter
and derivation path as local tools.

Current hosted options to evaluate:

| Capability | Option | Notes |
| --- | --- | --- |
| Unified multimodal embeddings | Google Gemini API / Vertex AI `gemini-embedding-2` | Google's current docs describe it as a multimodal embedding model that maps text, images, video, audio, and documents into one embedding space. |
| Vertex AI compatibility | Google `multimodalembedding@001` | Supports multimodal embeddings for image, text, and video use cases; useful where existing Vertex AI workflows use that model. |
| Document/image/video retrieval embeddings | Voyage multimodal embeddings | Hosted multimodal embeddings for interleaved text and visual data such as screenshots, PDFs, slides, tables, figures, and video. |
| Text/image/PDF multimodal embeddings | Jina embeddings | Hosted multimodal embedding models for text, images, and visually rich documents. |

Hosted adapters should record provider, model, version, request hash, output
hash or reference, cost, latency, security labels, and derivation ledger links.
Credentials belong in deployment secrets or provider-specific secret stores, not
in markdown, package manifests, Source Packages, or Context Packages.

These providers are optional. The engine's contract is the adapter record shape,
not any single provider.

## Recommended Pipelines

### Documents

```text
Source Package
  -> AssetStore
  -> Docling first
  -> Marker / olmOCR / Unstructured fallback or specialist pass
  -> Tesseract fallback for OCR
  -> optional ColPali page retrieval
  -> structured elements, extracted text, images, tables
```

Docling is the default router because it covers multiple formats, local
execution, OCR, advanced PDF layout, exports, audio support, VLM support, and
agent integrations. Marker is valuable for high-accuracy document-to-markdown
and JSON. olmOCR is valuable for scanned and image-based document linearization.
Unstructured remains useful as an ingestion/pre-processing adapter, but its own
documentation calls out limits for the open-source library in production use.

### Images

```text
Source Package
  -> AssetStore
  -> OCR pass when text may exist
  -> Qwen VL visual reasoning when configured
  -> OpenCLIP / ImageBind embedding
  -> Claims only when extraction output is reviewed or policy-accepted
```

Images should not become truth just because a model captioned them. Captions,
OCR, classifications, and visual answers are derived outputs linked to the raw
asset.

### Audio

```text
Source Package
  -> AssetStore
  -> whisper.cpp or Whisper transcript
  -> speaker/time metadata when available
  -> transcript chunks
  -> Claims / Facts / Memory Objects through the normal review lifecycle
```

Transcripts are useful derived evidence, but the audio file remains the preserved
source.

### Video

```text
Source Package
  -> AssetStore
  -> FFmpeg demux
  -> audio transcript
  -> frame/keyframe assets
  -> Qwen VL or embedding pass when configured
  -> episode/workflow extraction when relevant
```

Video becomes a bundle of governed derived artifacts: transcript, keyframes,
frame captions, object/text observations, and timing metadata.

## Storage Shape

The current public backend already supports:

```text
source_packages
assets
asset_adapter_runs
asset_extractions
asset_transcripts
asset_ocr_spans
asset_visual_observations
asset_embedding_refs
derivation_ledger
parser-produced asset metadata
workspace-scoped indexer ingestion
```

`asset_adapter_runs` records:

```text
adapter_id
adapter_role
modality
status
input_hash
output_hash
output_text / output_ref
model_id / model_version
confidence / precision
security_labels / partition_ids
derivation_ledger_id
```

`asset_extractions` is the generic adapter-output projection table. Typed
projection tables hang off it for the modality-specific payloads the engine needs
to recall and review:

```text
asset_transcripts          transcript text, speaker, language, start/end time
asset_ocr_spans            OCR text, page number, bounding box
asset_visual_observations  captions, states, regions, frame times
asset_embedding_refs       embedding model metadata and external/internal vector refs
```

Those rows include or inherit:

```text
workspace_id
source_package_id
asset_id
adapter_run_id
extraction_type
modality
content_text / content_ref
model_id / model_version when applicable
content_hash
confidence
precision
security_labels / partition_ids
created_by
derivation_ledger_id
```

Completed adapter outputs can already enter the truth lifecycle through:

```text
MemoryCore.claim_from_asset_adapter_run(run_id, opts)
```

That bridge preserves the adapter output as a derived Source Package and creates
a pending Claim. It does not automatically promote the model output to a Fact.

Typed extractions can also enter the truth lifecycle through:

```text
MemoryCore.record_asset_extraction(run_id, opts)
MemoryCore.claim_from_asset_extraction(extraction_id, opts)
```

`record_asset_extraction/2` requires a completed adapter run and writes the
generic extraction row plus exactly one typed projection row. Text-bearing
transcripts, OCR spans, and visual observations can become pending Claims.
Reference-only embedding rows remain governed retrieval/index projections and are
not claimable text.

`MemoryCore.run_asset_adapter/3` auto-projects supported completed runs into
those tables by default:

| Adapter role | Default projection |
| --- | --- |
| `document_intelligence` / `ocr` | `asset_ocr_spans` |
| `audio_transcription` | `asset_transcripts` |
| `visual_reasoning` | `asset_visual_observations` |
| `multimodal_embedding` | `asset_embedding_refs` when an embedding ref is available |

Callers can pass `auto_extract: false` when a run should be recorded without an
automatic projection. `MultimodalExtractionParser` now handles the first parser
normalization pass: transcript JSON can become segment-level transcript rows;
document JSON can become page, element, and table OCR rows; and frame JSON can
become visual observations and object-detection rows. Richer adapter-specific
parsers should still cover real tool/provider schemas as needed.

Retrieval can return these projection rows as source-linked Context Package
objects. This lets an agent use a transcript, OCR span, visual observation, or
embedding reference as governed context without treating it as an accepted Fact.

## Deployment

Docker is optional. Local development can run with SQLite and whichever binaries
are available on the machine. Production deployments can package the selected
adapters in Docker or another process manager.

The engine should ask the registry what is configured and available:

```text
MultimodalToolRegistry.recommended_pipeline(:document)
MultimodalToolRegistry.availability()
MemoryCore.run_asset_adapter(asset_id, :docling, command: "docling")
```

API clients can enter the same governed path through:

```text
POST /api/assets
{
  "workspace": "default",
  "filename": "meeting.wav",
  "content_base64": "...",
  "adapter_id": "openai_whisper"
}
```

The route stores the raw file as a Source Package plus workspace-scoped asset
row, then optionally runs the adapter so extraction projection rows are created
without bypassing Memory Core.

Connector adapters can preserve downloaded attachments and files through:

```text
Connectors.preserve_payload_assets(:slack, "message-id", payload,
  connector_id: "slack-main",
  workspace_id: "default"
)
```

The helper accepts attachment lists under `attachments` or `files`, supports
local paths and base64 content, preserves connector origin metadata, and returns
per-attachment errors so a single bad file does not erase the rest of the sync
batch.

Connector sync implementations may also return raw payloads alongside generated
Signals. When those payloads include `attachments` or `files`,
`Connectors.Runner` preserves them through the same Memory Core asset path before
the connector run is marked complete.

Missing binaries should degrade gracefully. The engine should preserve the raw
asset, write a warning or adapter-run failure, and continue the rest of the
pipeline where possible.

## Primary References

- Docling: <https://github.com/docling-project/docling>
- Marker: <https://github.com/datalab-to/marker>
- olmOCR: <https://github.com/allenai/olmocr>
- Unstructured open source: <https://docs.unstructured.io/open-source/introduction/overview>
- Whisper: <https://github.com/openai/whisper>
- whisper.cpp: <https://github.com/ggml-org/whisper.cpp>
- Qwen VL: <https://github.com/QwenLM/Qwen3-VL>
- ColPali / ColQwen: <https://github.com/illuin-tech/colpali>
- OpenCLIP: <https://github.com/mlfoundations/open_clip>
- ImageBind: <https://github.com/facebookresearch/ImageBind>

## Hosted References

- Google Gemini API embeddings: <https://ai.google.dev/gemini-api/docs/embeddings>
- Google Vertex AI / Gemini Enterprise multimodal embeddings: <https://docs.cloud.google.com/gemini-enterprise-agent-platform/models/embeddings/get-multimodal-embeddings>
- Voyage multimodal embeddings: <https://docs.voyageai.com/docs/multimodal-embeddings>
- Jina embeddings: <https://jina.ai/embeddings/>
