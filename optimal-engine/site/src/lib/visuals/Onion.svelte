<script lang="ts">
  // Onion visualization of the context architecture.
  //
  // 8 concentric rings, raw signal at the core, envelope on the outside.
  // The agent sits beyond the outermost ring and peels INWARD only as
  // far as it needs. Most queries terminate at the wiki ring.
  //
  // Click any ring to load its full granular detail in the side panel.

  type Layer = {
    n: number;
    name: string;
    short: string;        // 1-line frame
    stage: string;
    module: string;
    inputT: string;
    outputT: string;
    storage: string;
    color: string;
    artifacts: { label: string; value: string }[];
  };

  const LAYERS: Layer[] = [
    {
      n: 0,
      name: 'Raw Signal',
      short: 'Tier 1 · plain markdown files + assets on disk. Append-only. Git-friendly.',
      stage: 'Stage 1 · Intake',
      module: 'OptimalEngine.Intake',
      inputT: '{source, format_hint, payload}',
      outputT: '%RawSignal{id, content_hash, origin, received_at, raw}',
      storage: '<org>/<workspace>/nodes/<node>/signals/YYYY-MM-DD-slug.md  ·  context.md  ·  signal.md  ·  assets/  ·  signals table (scoped by workspace_id)',
      color: '#fc9e6c',
      artifacts: [
        { label: 'On-disk shape',     value: 'nodes/<node>/{ context.md · signal.md · signals/YYYY-MM-DD-slug.md }  +  assets/  +  architectures/*.yaml  +  topology.yaml' },
        { label: 'Signal file',       value: 'YYYY-MM-DD-slug.md — append-only. One file per event: transcript, decision, plan, note. Convention: lowercase-kebab.' },
        { label: 'context.md',        value: 'Persistent ground truth per node. Edited in place when facts change. The "permanent" companion to signals.' },
        { label: 'signal.md',         value: 'Rolling weekly status per node. Overwritten each cycle. The temporal layer.' },
        { label: 'Frontmatter (YAML)', value: 'title · genre · mode · node · sn_ratio · entities · authored_at — engine reads these directly off the .md file' },
        { label: 'Body convention',   value: '## Summary  → engine pulls for L0 (~100 tok)\n## Key points → L1 (~2K tok)\n## Detail     → full L2' },
        { label: 'ID format',         value: 'sha256:<64-hex>  — content-addressed, dedup at intake' },
        { label: 'Mutability',        value: 'NEVER rewrites in place. Every edit is a new dated file at a new path. git diff works out of the box.' },
        { label: 'What lives here',   value: 'markdown signals · PDFs · images · audio · video · transcripts · attachments — all human-readable on disk' }
      ]
    },
    {
      n: 1,
      name: 'Parsed',
      short: 'Format → text + structure + assets. Markdown is the native lingua franca.',
      stage: 'Stage 2 · Parse',
      module: 'OptimalEngine.Parser → Parser.<Format>',
      inputT: '%RawSignal{}',
      outputT: '%ParsedDoc{text, structure, assets, modality}',
      storage: 'In-memory pipeline value (feeds Decompose). Frontmatter split off here.',
      color: '#d68fff',
      artifacts: [
        { label: 'Markdown (.md)',      value: 'NATIVE — primary on-disk format. YAML frontmatter peeled off; body kept as-is. .txt .rst .adoc treated similarly.' },
        { label: 'Frontmatter parser',  value: 'yaml_elixir splits the leading --- block from the body; fields populate %Classification at this stage.' },
        { label: 'Data formats',        value: 'YAML / JSON / TOML — yaml_elixir · Jason — for architectures/*.yaml and topology.yaml configs' },
        { label: 'Tabular',             value: 'CSV / TSV — NimbleCSV — each row becomes a paragraph chunk' },
        { label: 'HTML',                value: 'Floki — strips chrome, keeps headings + links' },
        { label: 'Source code',         value: 'tree-sitter or native — 30+ extensions, code modality preserved' },
        { label: 'PDF',                 value: 'pdftotext shell or pdf_extract — pages preserved as section boundaries' },
        { label: 'Office',              value: '.docx .pptx .xlsx — zip + OOXML — slides become sections' },
        { label: 'Images',              value: '.png .jpg .gif .webp — tesseract OCR for text, original asset retained at assets/<hash>' },
        { label: 'Audio',               value: '.mp3 .wav .m4a .ogg .flac — whisper.cpp local server — transcript is the parsed text, audio asset retained' },
        { label: 'Video',               value: '.mp4 .mov .webm — ffmpeg → frames (→ image backend) + audio (→ whisper)' },
        { label: 'Structure preserved', value: 'every heading · page · slide · timestamp · code-block boundary — Decompose depends on these' }
      ]
    },
    {
      n: 2,
      name: 'Chunks',
      short: 'Hierarchical decomposition. Four scales, parent-linked.',
      stage: 'Stage 3 · Decompose',
      module: 'OptimalEngine.Decomposer',
      inputT: '%ParsedDoc{}',
      outputT: '%ChunkTree{root, nodes: [%Chunk{id, parent_id, scale, offset, length, text, modality, asset_ref}]}',
      storage: '.optimal/index.db → chunks table (SQLite)',
      color: '#bb7eff',
      artifacts: [
        { label: ':document scale',  value: 'full source. One per signal.' },
        { label: ':section scale',   value: 'one per heading / page / slide / structural unit.' },
        { label: ':paragraph scale', value: 'one per paragraph / code block / table row group.' },
        { label: ':chunk scale',     value: '~512 tokens, sliding window with 64-token overlap, respects paragraph boundaries.' },
        { label: 'Hard rules',       value: 'never split structural boundaries · never split mid-word · code respects function/class boundaries' },
        { label: 'Round-trip',       value: 'chunk concat = byte-identical text (invariant)' }
      ]
    },
    {
      n: 3,
      name: 'Classify',
      short: 'Mode + Genre + Type + Format + Structure + intent + confidence, per chunk per scale.',
      stage: 'Stage 4 · Classify + IntentExtract',
      module: 'OptimalEngine.Classifier · OptimalEngine.IntentExtractor',
      inputT: '%Chunk{}',
      outputT: '%Classification{mode, genre, type, format, structure, intent, sn_ratio, confidence}',
      storage: '.optimal/index.db → classifications table — one row per chunk',
      color: '#7eb6ff',
      artifacts: [
        { label: 'M — Mode',     value: ':linguistic · :visual · :code · :data · :mixed' },
        { label: 'G — Genre',    value: ':spec · :brief · :plan · :transcript · :report · :ADR · :note (+ many)' },
        { label: 'T — Type',     value: ':direct · :inform · :commit · :decide · :express' },
        { label: 'F — Format',   value: ':markdown · :code · :json · :cli_output · :diff' },
        { label: 'W — Structure', value: 'genre-specific internal skeleton' },
        { label: 'Intent (10-value enum)', value: ':request_info · :propose_decision · :record_fact · :express_concern · :commit_action · :reference · :narrate · :reflect · :specify · :measure' },
        { label: 'sn_ratio',     value: 'Signal-to-noise estimate per chunk' },
        { label: 'Confidence',   value: 'heuristics-first; Ollama-augmented when available' }
      ]
    },
    {
      n: 4,
      name: 'Embed',
      short: '768-dim aligned vector space. Text · image · audio.',
      stage: 'Stage 5 · Embed',
      module: 'OptimalEngine.Embedder (Ollama + Whisper)',
      inputT: '%Chunk{}',
      outputT: '%Embedding{chunk_id, model, dim: 768, vector: [float], modality}',
      storage: '.optimal/index.db → embeddings table — vectors indexed for cosine retrieval (FTS5 + sqlite-vec)',
      color: '#5fcfd4',
      artifacts: [
        { label: 'Text',  value: 'nomic-embed-text-v1.5  ·  768-dim  ·  Ollama /api/embeddings' },
        { label: 'Image', value: 'nomic-embed-vision-v1.5  ·  768-dim  ·  ALIGNED with text' },
        { label: 'Audio', value: 'whisper.cpp → transcript → text embedder  ·  768-dim' },
        { label: 'Code',  value: 'nomic-embed-text-v1.5  ·  code chunks embed as text' },
        { label: 'Alignment invariant', value: 'a text query embedding can retrieve an image chunk because both live in the same 768-d space. This is the whole product.' },
        { label: 'Local · zero cloud',  value: 'all models run on the host; no signal leaves the box.' }
      ]
    },
    {
      n: 5,
      name: 'Cluster',
      short: 'HDBSCAN over a weighted feature graph. Themes auto-named.',
      stage: 'Stage 8 · Cluster',
      module: 'OptimalEngine.Clusterer',
      inputT: '%Embedding{} (and metadata)',
      outputT: '%Cluster{id, theme, intent_dominant, member_chunk_ids, centroid_vector}',
      storage: '.optimal/index.db → clusters · cluster_members tables',
      color: '#7be3a3',
      artifacts: [
        { label: 'Algorithm',     value: 'HDBSCAN — incremental add per new chunk; full rebuild only via mix optimal.cluster.rebuild' },
        { label: 'Feature blend', value: '0.60 · embedding  +  0.20 · entity_overlap  +  0.15 · intent_match  +  0.05 · node_affinity' },
        { label: 'Theme naming',  value: 'Ollama auto-names each cluster from top-N member chunks' },
        { label: 'Centroid',      value: 'mean of member vectors — used for cluster-expand at retrieval' },
        { label: 'Intent_dominant', value: 'majority intent across members — used for intent-match retrieval boost' }
      ]
    },
    {
      n: 6,
      name: 'Wiki',
      short: 'Tier 3 · markdown files at .wiki/ · LLM-curated · hot-cited · git-diffable.',
      stage: 'Stage 9 · Curate',
      module: 'OptimalEngine.Wiki.Curator · .Directives · .Integrity · .Page',
      inputT: '%Cluster{} + new chunks + schema',
      outputT: '%WikiPage{slug, frontmatter, body, citations, audience, version}',
      storage: '<org>/<workspace>/.wiki/<slug>.md  +  <workspace>/.wiki/SCHEMA.md  +  .optimal/index.db → wiki_pages · citations (scoped by workspace_id)',
      color: '#7ea8ff',
      artifacts: [
        { label: 'On disk',           value: '.wiki/<slug>.md — plain markdown with YAML frontmatter. Human-readable, git-trackable, diff-friendly. Curator writes; humans can edit.' },
        { label: '.wiki/SCHEMA.md',   value: 'Governance rules the curator honors — naming, audience policy, citation requirements. Edit this to steer curation.' },
        { label: 'Frontmatter',       value: 'title · last_curated · curated_by · source_count · audience · version' },
        { label: 'Page anatomy',      value: '## Summary → ## Open threads → ## Related → ## Incoming  (each section is parsed structurally)' },
        { label: 'Hot citations',     value: 'every factual claim carries {{cite: optimal://...}} — no claim survives without one' },
        { label: 'Directive — {{cite}}',    value: '{{cite: optimal://nodes/operation-weekly-review/signals/2026-04-11-customer-pricing-call.md}} — binds a sentence to its raw source' },
        { label: 'Directive — {{include}}', value: '{{include: optimal://...}} — inlines content from another URI on render' },
        { label: 'Directive — {{expand}}',  value: '{{expand: ed-counter-offer-options}} — sub-query placeholder; resolved on demand' },
        { label: 'Directive — {{search}}',  value: '{{search: pricing AND Q4}} — live query embedded in the page' },
        { label: 'Directive — {{table}}',   value: '{{table: deals where status=active}} — structured tabular pull' },
        { label: 'Directive — {{trace}}',   value: '{{trace: pricing-anchor}} — evidence trail across raw sources' },
        { label: 'Directive — {{recent}}',  value: '{{recent: project-platform-launch within=14d}} — time-bounded slice of related signals' },
        { label: '[[wikilinks]]',     value: '[[other-page]] — page-to-page cross references in addition to URI citations' },
        { label: 'Audience variants', value: 'one signal → N pages: sales / legal / exec / engineering. Filter at curation time, not query time. Strictly safer.' },
        { label: 'Integrity gate',    value: 'every claim cites · every citation resolves · schema rules pass · contradictions surface — fail closed, write nothing if any fails' },
        { label: 'Trigger',           value: 'on store.chunk.indexed event → compute affected pages (cited cluster/entity overlap) → enqueue curator job' },
        { label: 'Versioning',        value: 'every curation writes a new .md version + diff; rollback via git or via wiki_pages history' }
      ]
    },
    {
      n: 7,
      name: 'Envelope',
      short: 'Composed for the receiver. ACL-scoped. Audience-shaped. Bandwidth-matched.',
      stage: 'Retrieval · Compose',
      module: 'OptimalEngine.Composer · ContextAssembler · IntentAnalyzer',
      inputT: '%Query{q, audience, format, bandwidth, principal}',
      outputT: '%Envelope{body, format, sources, warnings, trace}',
      storage: 'Returned to caller — sources cite back as optimal:// URIs that resolve to the .md files on disk; logged to .optimal/index.db → events for audit',
      color: '#f1f1f3',
      artifacts: [
        { label: 'Format options',   value: ':plain · :markdown · :claude · :openai (tool-result)' },
        { label: 'Audiences',        value: 'default · sales · legal · exec · engineering — selects audience-aware wiki variant' },
        { label: 'Bandwidth tiers',  value: 'L0 (~100 tok abstract) · L1 (~2K tok summary) · full' },
        { label: 'Sources',          value: 'every URI cited — agent can drill back into any tier' },
        { label: 'Trace',            value: 'wiki_hit? · n_candidates · n_delivered · truncated? · elapsed_ms · intent decoded' },
        { label: 'ACL scoping',      value: 'principal × resource × action — enforced at retrieval, never inferred' },
        { label: 'Wall-clock target', value: 'p50 < 50 ms (wiki hit) · p99 < 200 ms · cold path p99 < 2 s' }
      ]
    }
  ];

  let active = $state(6); // start on Wiki — most-asked layer
  const layer = $derived(LAYERS[active]);

  // Geometry — all rings live in a 700×700 viewBox, center (350,350).
  const CENTER = 350;
  const RADIUS_INNER = 48;
  const RADIUS_OUTER = 320;
  const STEP = (RADIUS_OUTER - RADIUS_INNER) / LAYERS.length; // ~34
</script>

<div class="onion">
  <div class="onion__viz">
    <svg viewBox="0 0 700 700" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Concentric layers of the Optimal Engine context architecture">
      <defs>
        {#each LAYERS as l}
          <radialGradient id="onion-grad-{l.n}" cx="50%" cy="50%" r="50%">
            <stop offset="0%"  stop-color={l.color} stop-opacity="0.32"/>
            <stop offset="60%" stop-color={l.color} stop-opacity="0.10"/>
            <stop offset="100%" stop-color={l.color} stop-opacity="0.04"/>
          </radialGradient>
        {/each}
        <filter id="onion-glow" x="-50%" y="-50%" width="200%" height="200%">
          <feGaussianBlur stdDeviation="3" result="blur"/>
          <feMerge>
            <feMergeNode in="blur"/>
            <feMergeNode in="SourceGraphic"/>
          </feMerge>
        </filter>
      </defs>

      <!-- Rings drawn from outside in so smaller rings sit on top -->
      {#each [...LAYERS].reverse() as l}
        {@const r = RADIUS_INNER + (l.n + 1) * STEP}
        {@const isActive = l.n === active}
        <g
          class="onion-ring"
          class:active={isActive}
          onclick={() => (active = l.n)}
          onkeydown={(e: KeyboardEvent) => { if (e.key === 'Enter' || e.key === ' ') active = l.n; }}
          role="button"
          tabindex="0"
          aria-label={l.name + ' layer'}
        >
          <circle
            cx={CENTER} cy={CENTER} r={r}
            fill={`url(#onion-grad-${l.n})`}
            stroke={l.color}
            stroke-width={isActive ? 2.5 : 1}
            stroke-opacity={isActive ? 0.95 : 0.45}
            filter={isActive ? 'url(#onion-glow)' : ''}
          />
          <!-- Layer label sits on the right edge of its ring -->
          <g transform="translate({CENTER + r - STEP / 2}, {CENTER}) rotate(-90)">
            <text
              text-anchor="middle"
              fill={isActive ? l.color : 'var(--text-muted)'}
              font-size="11"
              font-weight={isActive ? '700' : '600'}
              letter-spacing="0.12em"
              style="text-transform: uppercase;"
            >
              {l.name}
            </text>
          </g>
        </g>
      {/each}

      <!-- Center dot (raw nucleus) -->
      <circle cx={CENTER} cy={CENTER} r="6" fill="#fc9e6c">
        <animate attributeName="r" values="6;9;6" dur="2.4s" repeatCount="indefinite"/>
      </circle>

      <!-- Agent outside, top-right, peels inward -->
      <g class="onion-agent" transform="translate(560, 100)">
        <rect x="-46" y="-18" width="92" height="36" rx="18" fill="#0d1015" stroke="#1c2230"/>
        <text x="0" y="5" text-anchor="middle" fill="#9aa0a8" font-size="11" font-weight="600">AGENT</text>
        <!-- Arrow from agent inward -->
        <line
          x1="-22" y1="14"
          x2={(CENTER + RADIUS_OUTER * 0.7) - 560} y2={(CENTER - RADIUS_OUTER * 0.4) - 100}
          stroke="#7ea8ff" stroke-width="1.2" stroke-dasharray="4 4" stroke-opacity="0.65"
        >
          <animate attributeName="stroke-dashoffset" values="0;-16" dur="1.8s" repeatCount="indefinite"/>
        </line>
        <text x="-8" y="40" text-anchor="middle" fill="#7ea8ff" font-size="9" font-weight="600" letter-spacing="1.5" opacity="0.75">PEELS INWARD</text>
      </g>

      <!-- Stage badges along the bottom -->
      <g transform="translate(40, 660)" font-size="10" fill="#6b7280">
        <text x="0" y="0">Center: raw signal</text>
        <text x="320" y="0" text-anchor="middle">→ derived → curated → composed →</text>
        <text x="660" y="0" text-anchor="end">Outermost: envelope</text>
      </g>
    </svg>

    <ol class="onion__legend" aria-label="Layers">
      {#each LAYERS as l}
        <li>
          <button
            type="button"
            class="onion__chip"
            class:active={l.n === active}
            style="--chip-color: {l.color}"
            onclick={() => (active = l.n)}
          >
            <span class="onion__chip-dot"></span>
            <span class="onion__chip-num">L{l.n}</span>
            <span class="onion__chip-name">{l.name}</span>
          </button>
        </li>
      {/each}
    </ol>
  </div>

  <aside class="onion__detail" aria-live="polite">
    <header class="onion__detail-head" style="--accent-layer: {layer.color}">
      <div class="onion__detail-meta">
        <span class="onion__detail-tag">L{layer.n} · {layer.stage}</span>
        <h3 class="onion__detail-title">{layer.name}</h3>
        <p class="onion__detail-short">{layer.short}</p>
      </div>
    </header>

    <dl class="onion__io">
      <div>
        <dt>Module</dt>
        <dd><code>{layer.module}</code></dd>
      </div>
      <div>
        <dt>Input</dt>
        <dd><code>{layer.inputT}</code></dd>
      </div>
      <div>
        <dt>Output</dt>
        <dd><code>{layer.outputT}</code></dd>
      </div>
      <div>
        <dt>Stored</dt>
        <dd>{layer.storage}</dd>
      </div>
    </dl>

    <section class="onion__artifacts">
      <h4>What's actually in this layer</h4>
      <dl>
        {#each layer.artifacts as a}
          <div class="onion__artifact">
            <dt>{a.label}</dt>
            <dd>{a.value}</dd>
          </div>
        {/each}
      </dl>
    </section>
  </aside>
</div>

<style>
  .onion {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(320px, 460px);
    gap: 2rem;
    align-items: start;
  }

  .onion__viz {
    position: relative;
  }
  .onion__viz svg {
    width: 100%;
    height: auto;
    display: block;
  }

  .onion-ring {
    cursor: pointer;
    transition: filter 0.18s ease, opacity 0.18s ease;
  }
  .onion-ring:hover { opacity: 0.92; }
  .onion-ring.active { filter: brightness(1.08); }

  .onion-agent rect { transition: stroke 0.18s ease; }

  /* Layer chips below the ring */
  .onion__legend {
    list-style: none;
    margin: 1rem 0 0;
    padding: 0;
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    justify-content: center;
  }
  .onion__chip {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.35rem 0.7rem 0.35rem 0.55rem;
    border-radius: 999px;
    border: 1px solid var(--border);
    background: var(--bg-elevated);
    color: var(--text-muted);
    font: inherit;
    font-size: 0.78rem;
    cursor: pointer;
    transition: border-color 0.15s ease, color 0.15s ease, background 0.15s ease;
  }
  .onion__chip:hover { color: var(--text); border-color: var(--chip-color); }
  .onion__chip.active {
    color: var(--text);
    border-color: var(--chip-color);
    background: color-mix(in srgb, var(--chip-color) 10%, var(--bg-elevated));
  }
  .onion__chip-dot {
    display: inline-block;
    width: 8px; height: 8px;
    border-radius: 999px;
    background: var(--chip-color);
  }
  .onion__chip-num {
    color: var(--text-subtle);
    font-weight: 600;
    font-size: 0.72rem;
  }
  .onion__chip.active .onion__chip-num { color: var(--chip-color); }
  .onion__chip-name {
    font-weight: 500;
  }

  /* Detail panel */
  .onion__detail {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 1.4rem 1.4rem 1.6rem;
    position: sticky;
    top: 1rem;
  }
  .onion__detail-head {
    border-left: 3px solid var(--accent-layer);
    padding-left: 0.85rem;
    margin-bottom: 1.25rem;
  }
  .onion__detail-tag {
    color: var(--text-subtle);
    font-size: 0.7rem;
    font-weight: 700;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .onion__detail-title {
    margin: 0.25rem 0 0.4rem;
    font-size: 1.5rem;
    font-weight: 600;
    letter-spacing: -0.01em;
    color: var(--text);
  }
  .onion__detail-short {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.9rem;
    line-height: 1.55;
  }

  .onion__io {
    margin: 0 0 1.25rem;
    display: grid;
    gap: 0.55rem;
    padding: 0.9rem 1rem;
    background: var(--bg);
    border: 1px solid var(--border-soft);
    border-radius: 10px;
  }
  .onion__io > div {
    display: grid;
    grid-template-columns: 70px 1fr;
    gap: 0.6rem;
    align-items: baseline;
  }
  .onion__io dt {
    color: var(--text-subtle);
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-weight: 700;
  }
  .onion__io dd {
    margin: 0;
    color: var(--text);
    font-size: 0.82rem;
    line-height: 1.5;
  }
  .onion__io code {
    font-family: var(--font-mono);
    color: var(--text);
    font-size: 0.78rem;
    background: var(--bg-elevated-2);
    padding: 1px 6px;
    border-radius: 4px;
    border: 1px solid var(--border);
    word-break: break-word;
  }

  .onion__artifacts h4 {
    margin: 0 0 0.7rem;
    font-size: 0.78rem;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.1em;
    font-weight: 700;
  }
  .onion__artifacts > dl {
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }
  .onion__artifact {
    display: grid;
    grid-template-columns: 130px 1fr;
    gap: 0.7rem;
    padding: 0.6rem 0.8rem;
    background: var(--bg);
    border: 1px solid var(--border-soft);
    border-radius: 8px;
    align-items: baseline;
  }
  .onion__artifact dt {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.72rem;
    font-weight: 600;
    letter-spacing: 0.04em;
  }
  .onion__artifact dd {
    margin: 0;
    color: var(--text);
    font-size: 0.84rem;
    line-height: 1.55;
    font-family: var(--font-mono);
    font-size: 0.78rem;
  }

  @media (max-width: 980px) {
    .onion {
      grid-template-columns: 1fr;
    }
    .onion__detail {
      position: static;
    }
    .onion__artifact {
      grid-template-columns: 1fr;
      gap: 0.25rem;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .onion-agent line, .onion__viz circle[cx] animate { animation: none; }
  }
</style>
