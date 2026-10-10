<script lang="ts">
  import HeroVisual from '$lib/visuals/HeroVisual.svelte';
  import Onion from '$lib/visuals/Onion.svelte';
  import RetrievalFlow from '$lib/visuals/RetrievalFlow.svelte';

  const promises = [
    {
      n: '01',
      name: 'Signal integrity',
      detail:
        'Nothing is stored without classification, intent, and citation lineage. The engine knows why each fact exists.'
    },
    {
      n: '02',
      name: 'Scale alignment',
      detail:
        'Every piece of content exists simultaneously as document, section, paragraph, and chunk. Retrieval returns the coarsest scale that answers.'
    },
    {
      n: '03',
      name: 'Modality alignment',
      detail:
        'Text, image, and audio embed into the same 768-dimensional space. A text query can retrieve an image. One retrieval layer, not three.'
    },
    {
      n: '04',
      name: 'Hallucination-proof',
      detail:
        'Every wiki claim cites a raw source. The integrity gate fails closed if any citation does not resolve. The memory layer cannot make things up — it can only surface what was actually said.'
    }
  ];

  // Memory failures the engine cures — taxonomy from Engramme's "Questions in
  // the Wild" study (n=1,940 questions, n=134 participants), mapped to
  // Optimal Engine retrieval flows. Each one is now a typed /api/recall
  // endpoint instead of a generic search.
  const failures = [
    {
      pattern: 'Past actions',
      example: '"What did we decide on pricing last quarter?"',
      flow: 'GET /api/recall/actions',
      mechanism: 'Filtered by intent ∈ { :propose_decision, :commit_action, :record_fact } over the time window.'
    },
    {
      pattern: 'Contact info',
      example: '"Who owns the Firecracker decision?"',
      flow: 'GET /api/recall/who',
      mechanism: 'Resolved against entity graph + node membership; returns principal + relationship to the topic.'
    },
    {
      pattern: 'Schedules',
      example: '"When does the renewal pipeline review run?"',
      flow: 'GET /api/recall/when',
      mechanism: 'Filtered by intent :specify + temporal hint; surfaces date-anchored signals.'
    },
    {
      pattern: 'Object locations',
      example: '"Where do we keep the security-audit notes?"',
      flow: 'GET /api/recall/where',
      mechanism: 'Routed through node topology + asset graph to the specific path on disk.'
    },
    {
      pattern: 'Tasks (open ownership)',
      example: '"What did Bob commit to in last week\'s standup?"',
      flow: 'GET /api/recall/owns',
      mechanism: 'Filtered by intent :commit_action + actor entity; status drawn from follow-up signals.'
    }
  ];

  const invariants = [
    'Tier 1 is append-only. Nothing is rewritten in place.',
    'Tier 2 is fully derivable from Tier 1.',
    'Tier 3 is LLM-owned. Humans write the schema, the curator writes the pages.',
    'Tenant isolation is absolute. No cross-tenant reads, ever.',
    'Permissions propagate by intersection — never union, never inferred.'
  ];

  // Capabilities matrix — what this engine ships, no vendor names
  const capabilities = [
    { dim: 'Enterprise connectors',                value: '14 in v1' },
    { dim: 'Permission-aware retrieval',           value: 'chunk-level + intersection propagation' },
    { dim: 'Signal classification Mode + Genre + Type + Format + Structure',  value: 'every chunk classified at ingest' },
    { dim: 'Per-chunk intent extraction',          value: '10-value enum, deterministic' },
    { dim: 'Tiered disclosure',                    value: 'L0 / L1 / full + bandwidth-matched' },
    { dim: 'Hierarchical chunking',                value: '4 scales: document / section / paragraph / chunk' },
    { dim: 'Multi-modal aligned embeddings',       value: 'text + image + audio in one 768-dim space' },
    { dim: 'Cross-modal retrieval',                value: 'text query → image chunk, no separate index' },
    { dim: 'Hot citations + integrity gate',       value: 'fail-closed; every claim cites' },
    { dim: 'Executable directives',                value: '7-directive grammar in wiki pages' },
    { dim: 'Audience-aware wiki variants',         value: 'sales / legal / exec / engineering' },
    { dim: 'Triggered incremental curation',       value: 'triple-loop SICA learning' },
    { dim: 'Local-first / self-hosted',            value: 'runs on a laptop; on-disk markdown' }
  ];
</script>

<svelte:head>
  <title>Optimal Engine — the second brain of your company</title>
  <meta name="description" content="The Optimal Engine is the second brain of your company. Every comm channel, every doc, every meeting, every signal flowing through your digital world — decoded, curated, delivered as permission-scoped context to any agent in under 200ms." />
</svelte:head>

<!-- HERO -->
<section class="hero">
  <div class="hero__bg" aria-hidden="true"></div>
  <div class="hero__grid"></div>
  <div class="container hero__inner">
    <div class="hero__copy">
      <span class="eyebrow">Open source · Elixir · MIT · Local-first</span>
      <h1 class="hero__title">
        The second brain<br />
        <span class="hero__title-em">of your company.</span>
      </h1>
      <p class="hero__lede">
        Every Slack message, every email, every doc, every meeting, every ticket, every commit — the engine
        ingests, decodes, embeds, clusters, and curates it. Then delivers the right context to the right agent
        (or human) in <strong>under 200&nbsp;ms</strong>, scoped to what the caller is allowed to see.
      </p>
      <p class="hero__sublede">
        Andrej Karpathy calls this Software&nbsp;3.0 — the LLM is the new compute, English the new programming
        language. If the model is the CPU, <strong>context is the memory hierarchy.</strong> The Optimal Engine
        is that hierarchy, organized as your company's brain.
      </p>

      <div class="hero__ctas">
        <a class="btn btn--primary" href="https://github.com/Miosa-osa/OptimalEngine#quick-start" target="_blank" rel="noopener">
          Quick start
        </a>
        <a class="btn btn--ghost" href="/how-it-works">Read the breakdown →</a>
      </div>

      <div class="hero__stats">
        <div><strong>1,075</strong><span>tests passing</span></div>
        <div><strong>14</strong><span>connectors</span></div>
        <div><strong>9-stage</strong><span>pipeline</span></div>
        <div><strong>3-tier</strong><span>memory</span></div>
      </div>
    </div>

    <div class="hero__art">
      <HeroVisual />
    </div>
  </div>
</section>

<!-- THESIS STRIP — four promises -->
<section class="section thesis">
  <div class="container">
    <span class="eyebrow">Four promises a brain must keep</span>
    <h2 class="section__title">If any one fails, it isn't a brain — it's a search engine wearing a brain's clothes.</h2>
    <div class="promises">
      {#each promises as p}
        <div class="promise">
          <div class="promise__num">{p.n}</div>
          <h3 class="promise__name">{p.name}</h3>
          <p class="promise__detail">{p.detail}</p>
        </div>
      {/each}
    </div>
  </div>
</section>

<!-- MEMORY FAILURES CURED — Engramme-flavored, but Optimal-grounded -->
<section class="section section--alt">
  <div class="container">
    <div class="section__header">
      <span class="eyebrow">Memory failures the engine cures</span>
      <h2 class="section__title">The five things people actually forget at work — and the typed recall flows for each.</h2>
      <p class="section__lede">
        A Harvard memory study found people fail at five specific patterns: past actions, contact info, schedules,
        object locations, and ownership of open tasks. Each of these is a different shape of query — a generic
        search engine treats them all the same. We don't. Each pattern has its own typed endpoint and its own
        retrieval intent filter, so the engine knows exactly what kind of memory you're trying to recover.
      </p>
    </div>

    <div class="failures">
      {#each failures as f, i}
        <div class="failure">
          <div class="failure__head">
            <span class="failure__num">{String(i + 1).padStart(2, '0')}</span>
            <h3 class="failure__name">{f.pattern}</h3>
          </div>
          <div class="failure__example">{f.example}</div>
          <div class="failure__flow"><code>{f.flow}</code></div>
          <p class="failure__mechanism">{f.mechanism}</p>
        </div>
      {/each}
    </div>
  </div>
</section>

<!-- COUNTER-POSITIONING — vs Engramme + the proprietary memory layer crowd -->
<section class="section section--counter">
  <div class="container">
    <div class="counter">
      <div class="counter__col">
        <span class="eyebrow eyebrow--neg">The proprietary memory layer</span>
        <h3 class="counter__title">Closed model · black box · SaaS-only</h3>
        <ul class="counter__list">
          <li>Architecture is proprietary; you can't audit it.</li>
          <li>Hosted in their cloud. Your signals leave your network.</li>
          <li>Consumer-first. Enterprise is an afterthought.</li>
          <li>"Trust us, we'll get to your industry eventually."</li>
        </ul>
      </div>
      <div class="counter__col counter__col--us">
        <span class="eyebrow eyebrow--pos">Optimal Engine</span>
        <h3 class="counter__title">Open source · self-hosted · enterprise-grade</h3>
        <ul class="counter__list">
          <li>Every line of the 9-stage pipeline is auditable.</li>
          <li>Runs on a laptop. Your signals never leave the box.</li>
          <li>Workspaces, ACLs, audit log, 14 connectors — shipped today.</li>
          <li>Open the markdown files in any editor. Grep. Git diff. Done.</li>
        </ul>
      </div>
    </div>
    <p class="counter__footer">
      Lossy compression isn't a bug — it's the reason brains work. The engine prunes noise, promotes signal, and
      lets the wiki forget what doesn't matter. <strong>Curated memory beats infinite memory.</strong>
    </p>
  </div>
</section>

<!-- ANTI-RAG FRAMING -->
<section class="section section--frame">
  <div class="container">
    <span class="eyebrow">The problem with the default move</span>
    <h2 class="frame__title">
      "Ingest everything into a vector DB, chunk, embed, and pray" is not a memory system.
      <br />It's a stochastic search engine wearing memory's clothes.
    </h2>
    <div class="frame__grid">
      <div class="frame__card">
        <div class="frame__bad">×</div>
        <h3>Classical RAG</h3>
        <p>Re-discovers the same facts on every query. No curation. No permissions baked in. No grain awareness.
        At 10K employees × 50 queries/day, you're paying retriever cost on every hit.</p>
      </div>
      <div class="frame__card">
        <div class="frame__warn">~</div>
        <h3>Navigation agents</h3>
        <p>Walk sources on every query the way coding agents walk a repo. Lower hallucination, higher latency.
        Memory still lives in the operator's head. No persistent curated layer.</p>
      </div>
      <div class="frame__card frame__card--good">
        <div class="frame__good">✓</div>
        <h3>Optimal Engine</h3>
        <p>Discover once. Curate forever. The wiki answers most queries in milliseconds. Hybrid retrieval is the
        fall-through, not the default. Permissions filter at curation time, not query time.</p>
      </div>
    </div>
  </div>
</section>

<!-- ON-DISK — every signal is a markdown file with YAML frontmatter -->
<section class="section section--disk">
  <div class="container">
    <div class="section__header">
      <span class="eyebrow">On disk · markdown all the way down</span>
      <h2 class="section__title">Organizations contain workspaces. Workspaces contain nodes. Nodes contain signals.</h2>
      <p class="section__lede">
        Every level is a folder. Every signal is a markdown file with YAML frontmatter. Every wiki page is more
        markdown. The only non-text artifact is a SQLite database for derivatives — and that's fully rebuildable
        from the files. <strong>Open, edit, grep, git diff.</strong> Nothing locked in a black box.
      </p>
    </div>

    <div class="disk">
      <pre class="disk__tree" aria-label="On-disk layout">
<span class="disk__c">sample-corp/</span>                              <span class="disk__t">Organization (= tenant)</span>
├── <span class="disk__d">engineering-brain/</span>                  <span class="disk__t">Workspace · its own knowledge base</span>
│   ├── <span class="disk__d">nodes/</span>                       <span class="disk__t">Tier 1 — raw, append-only, human-edited</span>
│   │   ├── product-customer-portal/                <span class="disk__t">a node — recursive folder tree</span>
│   │   │   ├── <span class="disk__f">context.md</span>         <span class="disk__t">persistent ground truth</span>
│   │   │   ├── <span class="disk__f">signal.md</span>          <span class="disk__t">rolling weekly status</span>
│   │   │   └── <span class="disk__d">signals/</span>           <span class="disk__t">append-only event log</span>
│   │   │       ├── <span class="disk__f">2026-04-10-microvm-spec.md</span>
│   │   │       └── <span class="disk__f">2026-04-15-data-arch-review.md</span>
│   │   └── operation-weekly-review/
│   │       └── signals/
│   │           └── <span class="disk__f">2026-04-11-customer-pricing-call.md</span>
│   ├── <span class="disk__d">.wiki/</span>                       <span class="disk__t">Tier 3 — LLM-curated, more markdown</span>
│   │   ├── <span class="disk__f">SCHEMA.md</span>               <span class="disk__t">governance rules the curator honors</span>
│   │   └── <span class="disk__f">core-platform-architecture.md</span>
│   ├── <span class="disk__d">architectures/</span>               <span class="disk__t">user-defined data-point schemas</span>
│   │   └── <span class="disk__f">customer_requirement.yaml</span>
│   └── <span class="disk__d">assets/</span>                      <span class="disk__t">hash-addressed binaries</span>
│       └── &lt;hash&gt;.&lt;ext&gt;
│
├── <span class="disk__d">sales-brain/</span>                        <span class="disk__t">Another workspace — same shape, isolated data</span>
│   ├── nodes/
│   ├── .wiki/
│   └── ...
│
└── <span class="disk__d">.optimal/</span>                            <span class="disk__t">Tier 2 — derivatives, all workspaces, rebuildable</span>
    ├── <span class="disk__f">index.db</span>                        <span class="disk__t">SQLite + FTS5 + sqlite-vec, rows scoped by workspace_id</span>
    └── <span class="disk__f">config.yaml</span>
</pre>

      <div class="disk__file">
        <header class="disk__file-head">
          <code>sample-corp/engineering-brain/nodes/product-customer-portal/signals/2026-04-10-microvm-spec.md</code>
          <span class="disk__chip">Tier 1 · raw signal</span>
        </header>
<pre class="disk__sample"><span class="disk__y">---</span>
<span class="disk__k">title</span>: <span class="disk__v">Core platform — microVM isolation spec</span>
<span class="disk__k">genre</span>: <span class="disk__v">spec</span>
<span class="disk__k">mode</span>: <span class="disk__v">linguistic</span>
<span class="disk__k">node</span>: <span class="disk__v">product-customer-portal</span>
<span class="disk__k">authored_at</span>: <span class="disk__v">2026-04-10T10:00:00Z</span>
<span class="disk__k">sn_ratio</span>: <span class="disk__v">0.85</span>
<span class="disk__k">entities</span>:
  - {`{`} name: <span class="disk__v">"Bob"</span>, type: person {`}`}
  - {`{`} name: <span class="disk__v">"Carol"</span>, type: person {`}`}
  - {`{`} name: <span class="disk__v">"microVM"</span>, type: concept {`}`}
<span class="disk__y">---</span>

<span class="disk__h">## Summary</span>      <span class="disk__t">→ engine pulls for L0 (~100 tok)</span>

Per-tenant microVM isolation on the core compute plane. Each tenant
gets an isolated VM; an in-VM daemon runs inside.

<span class="disk__h">## Key points</span>   <span class="disk__t">→ engine pulls for L1 (~2K tok)</span>

- Strong process + network isolation per tenant
- Cold-start target: p95 under 100 ms
- Warm-pool policy per SLA tier

<span class="disk__h">## Detail</span>       <span class="disk__t">→ full L2 / decomposed into chunks</span>

…
</pre>

        <footer class="disk__file-foot">
          <span>YAML frontmatter populates classification at parse time. Body sections map directly to the L0 / L1 / L2 disclosure tiers.</span>
        </footer>
      </div>
    </div>
  </div>
</section>

<!-- ONION — eight concentric layers, full granular detail in side panel -->
<section id="how-it-works" class="section section--onion">
  <div class="container">
    <div class="section__header">
      <span class="eyebrow">Architecture · the eight layers</span>
      <h2 class="section__title">Eight concentric layers, raw signal at the core.</h2>
      <p class="section__lede">
        Click any ring to see exactly what lives in it — module name, input struct, output struct, storage
        table, every artifact. The agent sits outside and peels inward only as far as it needs. Most queries
        terminate at the wiki ring. The hot path stays under 200&nbsp;ms because most queries hit curated
        knowledge.
      </p>
    </div>

    <div class="visual-card visual-card--onion">
      <Onion />
    </div>
  </div>
</section>

<!-- FIVE INVARIANTS -->
<section class="section section--alt">
  <div class="container">
    <div class="section__header">
      <span class="eyebrow">Five invariants</span>
      <h2 class="section__title">Violate any of these and the engine is broken.</h2>
      <p class="section__lede">
        These aren't preferences. They are the load-bearing commitments that keep raw, derived, and curated
        from collapsing into each other.
      </p>
    </div>
    <ol class="invariants">
      {#each invariants as inv, i}
        <li class="invariant">
          <span class="invariant__num">{String(i + 1).padStart(2, '0')}</span>
          <span class="invariant__text">{inv}</span>
        </li>
      {/each}
    </ol>
  </div>
</section>

<!-- RETRIEVAL FLOW -->
<section class="section">
  <div class="container">
    <div class="section__header">
      <span class="eyebrow">Retrieval</span>
      <h2 class="section__title">Wiki first. Search second. Envelope always.</h2>
      <p class="section__lede">
        Most agent queries never touch the retriever — the curated wiki already answered them. When the wiki
        misses, hybrid retrieval (BM25 + vector + graph + intent-match + cluster-expand + temporal-decay) picks
        up. The receiver always gets one envelope: ACL-scoped, audience-shaped, bandwidth-matched.
      </p>
    </div>

    <div class="visual-card">
      <RetrievalFlow />
    </div>
  </div>
</section>

<!-- CAPABILITIES — what this engine ships -->
<section class="section">
  <div class="container">
    <div class="section__header">
      <span class="eyebrow">Capabilities</span>
      <h2 class="section__title">Every primitive shipped, every decision deliberate.</h2>
      <p class="section__lede">
        This is the full surface — auditable in code, runnable on a laptop, isolated per workspace.
      </p>
    </div>

    <div class="capabilities">
      {#each capabilities as cap}
        <div class="capability">
          <div class="capability__dim">{cap.dim}</div>
          <div class="capability__value">{cap.value}</div>
        </div>
      {/each}
    </div>
  </div>
</section>

<!-- CTA -->
<section class="section section--cta">
  <div class="container cta">
    <div>
      <h2>Run it locally in five minutes.</h2>
      <p>Elixir 1.17, Node 20, a C toolchain. <code>make install &amp;&amp; make bootstrap &amp;&amp; make dev</code>.</p>
    </div>
    <div class="cta__actions">
      <a class="btn btn--primary" href="https://github.com/Miosa-osa/OptimalEngine#quick-start" target="_blank" rel="noopener">Quick start</a>
      <a class="btn btn--ghost" href="https://github.com/Miosa-osa/OptimalEngine" target="_blank" rel="noopener">Star on GitHub</a>
    </div>
  </div>
</section>

<style>
  /* HERO */
  .hero {
    position: relative;
    overflow: hidden;
    padding: 5rem 0 4.5rem;
  }
  .hero__bg {
    position: absolute;
    inset: -30% -10% auto -10%;
    height: 90%;
    background:
      radial-gradient(closest-side at 30% 30%, rgba(126, 168, 255, 0.22), transparent 70%),
      radial-gradient(closest-side at 75% 60%, rgba(187, 126, 255, 0.16), transparent 70%),
      radial-gradient(closest-side at 50% 80%, rgba(95, 207, 212, 0.10), transparent 70%);
    filter: blur(40px);
    pointer-events: none;
    z-index: 0;
  }
  .hero__grid {
    position: absolute;
    inset: 0;
    background-image:
      linear-gradient(to right, rgba(255,255,255,0.025) 1px, transparent 1px),
      linear-gradient(to bottom, rgba(255,255,255,0.025) 1px, transparent 1px);
    background-size: 32px 32px;
    mask-image: radial-gradient(ellipse 80% 70% at 50% 30%, black 30%, transparent 80%);
    pointer-events: none;
    z-index: 0;
  }
  .hero__inner {
    position: relative;
    z-index: 1;
    display: grid;
    grid-template-columns: 1.05fr 0.95fr;
    gap: 2.5rem;
    align-items: center;
  }
  .hero__title {
    font-size: clamp(2.4rem, 5.5vw, 4.4rem);
    line-height: 1.04;
    letter-spacing: -0.02em;
    margin: 0.6rem 0 1.4rem;
    font-weight: 600;
  }
  .hero__title-em {
    background: linear-gradient(120deg, var(--text) 0%, var(--accent) 100%);
    -webkit-background-clip: text;
    background-clip: text;
    color: transparent;
  }
  .hero__lede {
    font-size: clamp(1rem, 1.4vw, 1.18rem);
    line-height: 1.6;
    color: var(--text);
    max-width: 580px;
    margin: 0 0 1rem;
  }
  .hero__lede strong {
    color: var(--text);
    font-weight: 600;
  }
  .hero__sublede {
    font-size: 0.95rem;
    line-height: 1.55;
    color: var(--text-muted);
    max-width: 580px;
    margin: 0 0 2rem;
    padding-left: 0.85rem;
    border-left: 2px solid var(--accent-soft);
  }
  .hero__sublede strong {
    color: var(--text);
    font-weight: 600;
  }
  .hero__ctas {
    display: flex;
    gap: 0.6rem;
    margin-bottom: 2.5rem;
  }
  .hero__stats {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 1.25rem;
    max-width: 580px;
  }
  .hero__stats div {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }
  .hero__stats strong {
    font-size: 1.25rem;
    color: var(--text);
    font-weight: 600;
  }
  .hero__stats span {
    color: var(--text-muted);
    font-size: 0.78rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  .hero__art { position: relative; }

  /* SECTION SHELL */
  .section__header {
    margin-bottom: 2.5rem;
    max-width: 760px;
  }
  .section__title {
    font-size: clamp(1.8rem, 3.2vw, 2.6rem);
    line-height: 1.15;
    letter-spacing: -0.01em;
    margin: 0.5rem 0 0.8rem;
    font-weight: 600;
  }
  .section__lede {
    color: var(--text-muted);
    margin: 0;
    font-size: 1.02rem;
    line-height: 1.6;
  }
  :global(.section__lede code) {
    background: var(--bg-elevated-2);
    padding: 1px 6px;
    border-radius: 4px;
    border: 1px solid var(--border);
    color: var(--text);
    font-size: 0.85em;
  }
  :global(.section__lede a) {
    color: var(--accent);
    border-bottom: 1px dashed currentColor;
  }
  .section--alt {
    background: linear-gradient(180deg, var(--bg-elevated) 0%, var(--bg) 100%);
    border-block: 1px solid var(--border-soft);
  }
  .section--frame {
    padding: 4rem 0;
  }

  /* THESIS / PROMISES */
  .thesis { padding: 4rem 0; }
  .promises {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 1rem;
    margin-top: 1.8rem;
  }
  @media (max-width: 1100px) {
    .promises { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  }
  .promise {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 1.5rem 1.5rem 1.7rem;
    position: relative;
    overflow: hidden;
  }
  .promise::before {
    content: '';
    position: absolute;
    inset: 0 0 auto 0;
    height: 1px;
    background: linear-gradient(90deg, transparent, var(--accent), transparent);
    opacity: 0.4;
  }
  .promise__num {
    color: var(--accent);
    font-size: 0.78rem;
    font-weight: 700;
    letter-spacing: 0.12em;
  }
  .promise__name {
    margin: 0.4rem 0 0.5rem;
    font-size: 1.15rem;
    font-weight: 600;
    letter-spacing: -0.005em;
  }
  .promise__detail {
    color: var(--text-muted);
    margin: 0;
    font-size: 0.92rem;
    line-height: 1.55;
  }

  /* MEMORY FAILURES */
  .failures {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 0.85rem;
  }
  .failure {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 1.3rem 1.4rem;
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
  }
  .failure__head {
    display: flex;
    align-items: baseline;
    gap: 0.6rem;
  }
  .failure__num {
    color: var(--accent);
    font-size: 0.78rem;
    font-weight: 700;
    letter-spacing: 0.12em;
  }
  .failure__name {
    margin: 0;
    font-size: 1.05rem;
    font-weight: 600;
  }
  .failure__example {
    color: var(--text-muted);
    font-style: italic;
    font-size: 0.92rem;
    line-height: 1.5;
  }
  .failure__flow code {
    display: inline-block;
    background: var(--bg);
    border: 1px solid var(--border);
    color: var(--accent);
    padding: 4px 10px;
    border-radius: 4px;
    font-family: var(--font-mono);
    font-size: 0.78rem;
  }
  .failure__mechanism {
    margin: 0;
    color: var(--text-subtle);
    font-size: 0.85rem;
    line-height: 1.55;
  }

  /* COUNTER-POSITIONING */
  .section--counter {
    padding: 4rem 0;
  }
  .counter {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
  }
  .counter__col {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: 14px;
    padding: 1.6rem 1.6rem 1.7rem;
  }
  .counter__col--us {
    border-color: var(--accent);
    box-shadow: 0 0 0 1px var(--accent-soft);
    background: linear-gradient(180deg, var(--bg-elevated) 0%, rgba(126, 168, 255, 0.04) 100%);
  }
  .eyebrow--neg {
    color: var(--text-subtle);
  }
  .eyebrow--pos {
    color: var(--accent);
  }
  .counter__title {
    font-size: 1.15rem;
    font-weight: 600;
    margin: 0.4rem 0 1rem;
    letter-spacing: -0.005em;
  }
  .counter__list {
    margin: 0;
    padding: 0;
    list-style: none;
    display: flex;
    flex-direction: column;
    gap: 0.45rem;
  }
  .counter__list li {
    color: var(--text-muted);
    font-size: 0.92rem;
    line-height: 1.5;
    padding-left: 1.1rem;
    position: relative;
  }
  .counter__list li::before {
    content: '·';
    position: absolute;
    left: 0;
    color: var(--accent);
    font-weight: 700;
  }
  .counter__col--us .counter__list li {
    color: var(--text);
  }
  .counter__footer {
    margin: 1.5rem 0 0;
    color: var(--text-muted);
    font-size: 0.95rem;
    line-height: 1.6;
    text-align: center;
    max-width: 760px;
    margin-left: auto;
    margin-right: auto;
  }
  .counter__footer strong {
    color: var(--text);
    font-weight: 600;
  }

  /* INVARIANTS */
  .invariants {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.7rem;
  }
  .invariant {
    display: flex;
    align-items: flex-start;
    gap: 0.85rem;
    padding: 1rem 1.2rem;
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: 10px;
  }
  .invariant__num {
    color: var(--accent);
    font-weight: 700;
    font-size: 0.78rem;
    letter-spacing: 0.08em;
    flex-shrink: 0;
    padding-top: 2px;
  }
  .invariant__text {
    color: var(--text);
    font-size: 0.92rem;
    line-height: 1.5;
  }

  /* Visual card — frames every diagram */
  .visual-card {
    background:
      radial-gradient(closest-side at 30% 0%, rgba(126, 168, 255, 0.06), transparent 70%),
      var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 2rem;
    box-shadow: 0 30px 60px -30px rgba(0, 0, 0, 0.6);
  }
  .visual-card--onion {
    padding: 2.5rem 2rem 2rem;
  }
  .section--onion { padding-bottom: 5rem; }

  /* ON-DISK CONVENTION SECTION */
  .section--disk {
    background: linear-gradient(180deg, var(--bg-elevated) 0%, var(--bg) 100%);
    border-block: 1px solid var(--border-soft);
  }
  .disk {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
  }
  .disk__tree,
  .disk__sample {
    margin: 0;
    padding: 1.4rem 1.5rem;
    background: var(--bg);
    border: 1px solid var(--border);
    border-radius: 12px;
    color: var(--text);
    font-family: var(--font-mono);
    font-size: 0.78rem;
    line-height: 1.7;
    overflow-x: auto;
    white-space: pre;
  }
  .disk__tree {
    box-shadow: 0 30px 60px -30px rgba(0, 0, 0, 0.5);
  }
  .disk__file {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: 12px;
    overflow: hidden;
    display: flex;
    flex-direction: column;
    box-shadow: 0 30px 60px -30px rgba(0, 0, 0, 0.5);
  }
  .disk__file-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
    padding: 0.7rem 1rem;
    background: var(--bg);
    border-bottom: 1px solid var(--border-soft);
  }
  .disk__file-head code {
    font-family: var(--font-mono);
    font-size: 0.8rem;
    color: var(--text);
  }
  .disk__chip {
    padding: 3px 10px;
    border-radius: 999px;
    font-size: 0.68rem;
    font-weight: 600;
    text-transform: lowercase;
    background: rgba(252, 158, 108, 0.12);
    color: #fc9e6c;
  }
  .disk__file .disk__sample {
    border: none;
    border-radius: 0;
    background: var(--bg-elevated);
  }
  .disk__file-foot {
    padding: 0.7rem 1rem;
    border-top: 1px solid var(--border-soft);
    background: var(--bg);
    color: var(--text-muted);
    font-size: 0.78rem;
  }

  /* Syntax tokens */
  .disk__c { color: var(--text); font-weight: 600; }
  .disk__d { color: var(--accent); font-weight: 600; }
  .disk__f { color: var(--text); }
  .disk__t { color: var(--text-subtle); }
  .disk__y { color: #bb7eff; font-weight: 600; }
  .disk__k { color: #5fcfd4; }
  .disk__v { color: var(--text); }
  .disk__h { color: #7be3a3; font-weight: 600; }

  @media (max-width: 880px) {
    .disk { grid-template-columns: 1fr; }
  }

  /* ANTI-RAG FRAME */
  .frame__title {
    font-size: clamp(1.3rem, 2vw, 1.65rem);
    line-height: 1.4;
    font-weight: 500;
    color: var(--text);
    max-width: 880px;
    margin: 0.5rem 0 2.5rem;
    letter-spacing: -0.005em;
  }
  .frame__grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 1rem;
  }
  .frame__card {
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 1.4rem 1.4rem 1.6rem;
    background: var(--bg-elevated);
  }
  .frame__card h3 {
    margin: 0.6rem 0 0.4rem;
    font-size: 1rem;
    font-weight: 600;
  }
  .frame__card p {
    color: var(--text-muted);
    margin: 0;
    font-size: 0.92rem;
    line-height: 1.55;
  }
  .frame__card--good {
    border-color: var(--accent);
    box-shadow: 0 0 0 1px var(--accent-soft);
    background: linear-gradient(180deg, var(--bg-elevated) 0%, rgba(126, 168, 255, 0.04) 100%);
  }
  .frame__bad,
  .frame__warn,
  .frame__good {
    width: 28px;
    height: 28px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: 999px;
    font-weight: 700;
    font-size: 0.95rem;
  }
  .frame__bad { background: rgba(248, 136, 136, 0.12); color: var(--bad); }
  .frame__warn { background: rgba(252, 198, 102, 0.12); color: #fc6; }
  .frame__good { background: var(--accent-soft); color: var(--accent); }

  /* CAPABILITIES — single-column on mobile, two on desktop */
  .capabilities {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.5rem;
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 1rem;
    background: var(--bg-elevated);
  }
  .capability {
    display: grid;
    grid-template-columns: 1fr 1.4fr;
    gap: 1rem;
    padding: 0.7rem 0.85rem;
    align-items: baseline;
    border-radius: 8px;
  }
  .capability:hover { background: var(--bg-elevated-2); }
  .capability__dim {
    color: var(--text-muted);
    font-size: 0.85rem;
    font-weight: 500;
  }
  .capability__value {
    color: var(--text);
    font-size: 0.85rem;
  }
  @media (max-width: 720px) {
    .capabilities { grid-template-columns: 1fr; }
    .capability { grid-template-columns: 1fr; gap: 0.2rem; }
  }

  /* COMPARE — legacy, kept for back-compat with any old refs */
  .compare {
    border: 1px solid var(--border);
    border-radius: 12px;
    overflow: hidden;
    background: var(--bg-elevated);
    font-size: 0.85rem;
  }
  .compare__row {
    display: grid;
    grid-template-columns: 1.6fr repeat(5, 1fr) 1.3fr;
    border-top: 1px solid var(--border-soft);
  }
  .compare__row:first-child { border-top: none; }
  .compare__row > div {
    padding: 0.7rem 0.85rem;
    font-size: 0.83rem;
  }
  .compare__row--head > div {
    color: var(--text-muted);
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-weight: 600;
    background: var(--bg);
  }
  .compare__col--us,
  .compare__row .compare__cell--good {
    background: rgba(126, 168, 255, 0.05);
    color: var(--text);
  }
  .compare__dim { color: var(--text-muted); font-weight: 500; }
  .compare__cell { color: var(--text-subtle); }
  .compare__cell--good { color: var(--text); font-weight: 500; }

  /* CTA */
  .section--cta { border-top: 1px solid var(--border-soft); }
  .cta {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 2rem;
    padding: 3rem 2rem;
    background:
      radial-gradient(closest-side at 0% 0%, rgba(126, 168, 255, 0.08), transparent 70%),
      var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: 16px;
  }
  .cta h2 {
    margin: 0 0 0.4rem;
    font-size: 1.5rem;
    font-weight: 600;
    letter-spacing: -0.01em;
  }
  .cta p {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.95rem;
  }
  .cta code {
    background: var(--bg);
    padding: 2px 8px;
    border-radius: 4px;
    border: 1px solid var(--border);
    color: var(--text);
    font-size: 0.85rem;
  }
  .cta__actions {
    display: flex;
    gap: 0.6rem;
    flex-shrink: 0;
  }

  /* RESPONSIVE */
  @media (max-width: 980px) {
    .hero__inner { grid-template-columns: 1fr; }
    .hero__art { max-width: 480px; margin: 0 auto; }
    .promises { grid-template-columns: 1fr; }
    .invariants { grid-template-columns: 1fr; }
    .counter { grid-template-columns: 1fr; }
  }
  @media (max-width: 720px) {
    .frame__grid { grid-template-columns: 1fr; }
    .compare__row { grid-template-columns: 1.6fr 1fr 1.3fr; }
    .compare__row > div:nth-child(3),
    .compare__row > div:nth-child(4),
    .compare__row > div:nth-child(5),
    .compare__row > div:nth-child(6) { display: none; }
    .cta { flex-direction: column; align-items: flex-start; }
    .visual-card { padding: 1rem; }
  }
</style>
