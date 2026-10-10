<script lang="ts">
  import CodeBlock from '$lib/components/CodeBlock.svelte';

  const setup = `git clone https://github.com/Miosa-osa/OptimalEngine.git
cd OptimalEngine
mix deps.get
mix compile
mix optimal.reality_check`;
</script>

<svelte:head>
  <title>Optimal Engine Docs Index</title>
</svelte:head>

<section class="hero">
  <div class="hero-eyebrow">Current docs surface</div>
  <h1>Backend-first operating engine for human and AI workspaces.</h1>
  <p class="hero-lead">
    Optimal Engine organizes messy sources into workspace topology, Signals,
    Claims, Facts, Memory Objects, Context Packages, Active Memory Pools,
    workflow/Skill Packages, governed tool calls, and projection surfaces.
  </p>
</section>

<section class="notice">
  <h2>Canonical docs live in markdown</h2>
  <p>
    This Svelte app is intentionally a lightweight index. The canonical product
    and architecture documentation lives in the repository markdown under
    <code>docs/</code>, <code>README.md</code>, and <code>GETTING_STARTED.md</code>.
  </p>
</section>

<section class="quick">
  <h2>Local verification</h2>
  <CodeBlock code={setup} lang="bash" />
</section>

<section class="grid" aria-label="Documentation links">
  <a class="doc-card" href="https://github.com/Miosa-osa/OptimalEngine/blob/main/README.md">
    <span class="doc-title">README</span>
    <span class="doc-desc">Product framing, hierarchy, layers, tiers, stages, rhythm, CLI, and surfaces.</span>
  </a>

  <a class="doc-card" href="https://github.com/Miosa-osa/OptimalEngine/blob/main/docs/README.md">
    <span class="doc-title">Docs Map</span>
    <span class="doc-desc">Canonical reading order for workspace setup, storage, signal theory, loops, packages, and integrations.</span>
  </a>

  <a class="doc-card" href="https://github.com/Miosa-osa/OptimalEngine/blob/main/docs/architecture/ENGINE-STRUCTURE.md">
    <span class="doc-title">Engine Structure</span>
    <span class="doc-desc">Organization, workspace, Nodes, layers, stores, projections, agents, and loops.</span>
  </a>

  <a class="doc-card" href="https://github.com/Miosa-osa/OptimalEngine/blob/main/docs/architecture/STORAGE-AND-PROJECTION-MAP.md">
    <span class="doc-title">Storage Map</span>
    <span class="doc-desc">Which substrate stores what, which layer owns meaning, and which surfaces display state.</span>
  </a>

  <a class="doc-card" href="https://github.com/Miosa-osa/OptimalEngine/blob/main/docs/reference/backend-readiness.md">
    <span class="doc-title">Backend Readiness</span>
    <span class="doc-desc">What is built now, what is only a spine, and which checks prove the runtime works.</span>
  </a>

  <a class="doc-card" href="https://github.com/Miosa-osa/OptimalEngine/blob/main/docs/concepts/signal-theory.md">
    <span class="doc-title">Signal Theory</span>
    <span class="doc-desc">Mode, Genre, Type, Format, and Structure for routing noisy input correctly.</span>
  </a>
</section>

<style>
  .hero {
    padding: 3rem 0 2.25rem;
    border-bottom: 1px solid var(--border);
    margin-bottom: 2rem;
  }

  .hero-eyebrow {
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--accent);
    margin-bottom: 1rem;
  }

  .hero h1 {
    max-width: 760px;
    font-size: 2.4rem;
    font-weight: 700;
    margin-bottom: 1rem;
  }

  .hero-lead {
    max-width: 760px;
    font-size: 1.05rem;
    line-height: 1.7;
  }

  .notice,
  .quick {
    margin-bottom: 2rem;
  }

  .grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.875rem;
    margin-bottom: 3rem;
  }

  .doc-card {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    padding: 1rem 1.125rem;
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: var(--r-md);
    text-decoration: none;
  }

  .doc-card:hover {
    border-color: var(--border-accent);
    background: var(--bg-hover);
    text-decoration: none;
  }

  .doc-title {
    color: var(--text);
    font-weight: 600;
  }

  .doc-desc {
    color: var(--text-muted);
    font-size: 0.875rem;
    line-height: 1.5;
  }

  @media (max-width: 760px) {
    .hero h1 {
      font-size: 2rem;
    }

    .grid {
      grid-template-columns: 1fr;
    }
  }
</style>
