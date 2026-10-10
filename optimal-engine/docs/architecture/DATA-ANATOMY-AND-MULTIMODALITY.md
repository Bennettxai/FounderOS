# Data Anatomy And Multimodality

This document defines the shape of data inside Optimal Engine.

The short version:

```text
Source Package = preserved evidence
Data Point = typed payload inside or derived from a Source Package
Data Architecture = schema for that data point
Field = typed slot with modality, shape, and processor
Chunk / Feature / Embedding = rebuildable processing output
Signal = classified interpretation
Claim / Fact / Memory = governed truth lifecycle
```

Do not collapse these into one object. They answer different questions.

## Core Anatomy

Every incoming thing should pass through this anatomy:

```text
Command or raw input
  -> Scope Envelope
  -> Source Package
  -> Data Point
  -> Data Architecture
  -> Fields
  -> Processors
  -> Parsed Representation
  -> Chunk Tree / Feature Outputs
  -> Signal
  -> Claim / Fact / Memory lifecycle
```

The Source Package is the evidence container. The Data Architecture describes
what kind of payload is inside it. Processors produce rebuildable outputs.
Memory Core decides what becomes institutional truth.

## Objects

| Object | Meaning | Truth or projection? | Owner |
| --- | --- | --- | --- |
| Scope Envelope | Actor, workspace, operation class, permissions, source type, and Node hints. | Governance context | Command Gateway / Governance |
| Source Package | Preserved evidence with hash, source URI, source time, trust, retention, security, and raw/verbatim archive link. | Durable evidence truth | Memory Core / Source Intake |
| Raw Artifact | Binary or text payload: file, media, connector payload, record, transcript, tool result. | Durable evidence truth | Source Intake / Asset Store |
| Data Point | Structured representation of one source payload or one extracted unit. | Durable or derived, depending on origin | Data Architecture / Memory Core |
| Data Architecture | Declares fields, primary modality, granularity, retention, and processor bindings. | Durable schema/config | Data Architecture |
| Field | One typed slot in a data point with modality, dimensions, required flag, and processor hint. | Schema/config | Data Architecture |
| Parsed Representation | Text, structure, assets, timestamps, regions, rows, frames, or records extracted from source. | Rebuildable derivation | Pipeline / Parser |
| Chunk Tree | Document/section/paragraph/chunk or modality-specific decomposition. | Rebuildable projection | Pipeline / Retrieval |
| Feature / Embedding | Processor output such as vector, transcript, OCR text, caption, numeric features, entities. | Rebuildable projection | Processor / Retrieval |
| Signal | Classified processing unit using Signal dimensions and routing metadata. | Interpretation/projection until promoted | Signal Pipeline |
| Claim | Source-backed assertion not yet accepted as true. | Durable assertion | Memory Core |
| Fact | Accepted assertion with evidence, scope, time, confidence, and precision. | Durable truth | Memory Core |
| Memory Object | Institutional meaning around sources, claims, facts, relationships, and time. | Durable memory truth | Memory Core |

## Modality Taxonomy

Optimal Engine must support these modalities as first-class data shapes:

| Modality | Examples | Default processing | Retrieval surface |
| --- | --- | --- | --- |
| `text` | notes, transcripts, docs, emails, prompts | parse, classify, chunk, embed | FTS, vector, citations |
| `code` | files, diffs, configs, SQL | parse symbols/hunks, classify, embed | code search, dependency links |
| `image` | screenshots, diagrams, scans, photos | OCR, caption, vision embedding, regions | visual search, OCR text, source preview |
| `audio` | calls, meetings, voice memos | transcription, speaker/time metadata, audio/text embedding | transcript search, time offsets |
| `video` | demos, webinars, walkthroughs | scenes, shots, keyframes, transcript, captions | scene/frame recall, transcript recall |
| `time_series` | telemetry, prices, metrics, vitals | windows, samples, statistical features | range queries, anomaly/feature search |
| `table` | CSV, spreadsheets, tabular exports | rows, columns, typed cells, summaries | row/column filters, table snippets |
| `structured` | tickets, CRM deals, invoices, API JSON | schema validation, field extraction, status/party links | structured filters + notes retrieval |
| `graph` | dependency graph, org graph, concept map | nodes, edges, typed relationships | graph traversal and scoped recall |
| `tensor` | model features, activations, arrays | shape validation, processor-specific features | model/feature inspection |
| `geo` | locations, routes, polygons | spatial indexing, region metadata | map/spatial filters |
| `binary` | unknown or opaque files | hash, metadata, quarantine or parser fallback | source preview/download only until parsed |

## Field Contract

Each field declares:

```text
name
modality
dims
required
processor
description
```

Examples:

```text
body:        modality=text, required=true,  processor=text_embedder
image:       modality=image, required=false, processor=image_embedder
audio_track: modality=audio, required=false, processor=audio_embedder
payload:     modality=structured, required=true
samples:     modality=time_series, dims=[:any, 2], processor=ts_feature_extractor
```

The field contract is how the engine avoids pretending every source is text.

## Multimodal Source Flow

```text
source file / connector payload
  -> Source Package
  -> Data Architecture selection
  -> field validation
  -> parser / extractor
  -> modality-specific decomposition
  -> processor outputs
  -> Signal classification
  -> Claim candidates
  -> governed truth lifecycle
```

Examples:

```text
meeting video
  -> Source Package
  -> multimodal_media
  -> video field + transcript field + thumbnails field + audio field
  -> scene/shot/frame chunks + transcript utterances
  -> text, vision, and audio embeddings
  -> Signals and Claims linked to time offsets and frame/source links
```

```text
CRM deal record
  -> Source Package
  -> structured_record
  -> payload, title, notes, status, parties
  -> structured filters + text embeddings for notes
  -> Claims about deal status, parties, dates, and obligations
```

```text
screenshot
  -> Source Package
  -> image_asset
  -> image field + OCR text + caption
  -> image embedding + OCR chunks
  -> Claims only if supported by visible source spans or review
```

## Storage Meaning

Use one physical database if needed, but do not confuse storage classes.

| Storage class | Examples | Rule |
| --- | --- | --- |
| Evidence truth | Source Packages, raw artifacts, content hashes, source URI, archive URI. | Preserve before interpretation. |
| Schema/config | Data Architectures, fields, processor bindings. | Version when behavior changes. |
| Derived projections | parsed text, chunks, embeddings, captions, OCR, summaries. | Rebuild from Source Packages and processor versions. |
| Interpretation | Signal, Claim candidates, extracted relationships. | Link to source spans and processor runs. |
| Accepted truth | Facts, Memory Objects, reviewed Relationship Edges. | Promote through policy/review/evidence. |
| Working state | Active Memory Pools, observations, pending Claims. | Task-scoped until promoted. |
| Human projections | markdown, wiki, HTML, app views, packages. | Export records and drift checks required. |

## Current Code Reality

Already present:

- `OptimalEngine.Architecture.Field` defines modalities and field shape.
- `OptimalEngine.Architecture.Architecture` defines Data Architecture specs.
- Built-in architectures include text, image, audio, structured, time-series,
  code, and multimodal media shapes.
- `OptimalEngine.Pipeline.Decomposer.Chunk` carries chunk scale, modality, and
  asset reference.
- `OptimalEngine.Pipeline.Embedder` dispatches text, image, audio, and video
  fallback paths.
- `OptimalEngine.MemoryCore.SourcePackage` preserves source metadata and text
  evidence for the first Memory Core slice.

Still needed:

- `raw_artifacts` or `asset_store` for binary source preservation.
- Source Package support for structured payloads and binary/archive references
  beyond `raw_text`.
- Data Architecture selection inside source-first intake.
- Processor-run records connected to Source Packages, chunks, Claims, and
  Derivation Ledger.
- Source spans for non-text modalities: time offsets, frame IDs, regions, row
  IDs, cell ranges, graph node/edge IDs.
- Multimodal retrieval packages that cite native source locations, not only text
  snippets.

## Non-Negotiable Rule

Multimodality is not an add-on retrieval feature. It is part of the evidence
model.

```text
Every modality must preserve source evidence.
Every processor output must be rebuildable.
Every generated assertion must cite native source locations.
Every accepted Fact must be promoted from evidence, not from an embedding.
```
