<script lang="ts">
  import TierStack from '$lib/visuals/TierStack.svelte';
  import Pipeline from '$lib/visuals/Pipeline.svelte';
  import RetrievalFlow from '$lib/visuals/RetrievalFlow.svelte';

  // Deep dive page. Each visual gets its own section with the
  // mechanics expanded around it.
</script>

<svelte:head>
  <title>How it works — Optimal Engine</title>
</svelte:head>

<section class="hiw-hero">
  <div class="container">
    <span class="eyebrow">How your company's second brain works</span>
    <h1>From signal to envelope.</h1>
    <p class="lede">
      A brain has three architectural commitments: separate <strong>raw</strong> memory from
      <strong>derivative</strong> memory, keep a curated <strong>wiki</strong> on top, and deliver one envelope
      to the receiver. The Optimal Engine is your company's brain — every signal flowing through your
      organization, decoded once and curated forever. Everything below is how it composes.
    </p>
  </div>
</section>

<section class="hiw-section">
  <div class="container hiw-split">
    <div class="hiw-copy">
      <span class="eyebrow">Memory</span>
      <h2>Three tiers, one direction of derivation.</h2>
      <p>
        <strong>Tier 1 — Raw sources.</strong> Hash-addressed signal files. The engine never rewrites them.
        This is your audit trail and your replay buffer.
      </p>
      <p>
        <strong>Tier 2 — Derivatives.</strong> Everything machine-maintained: parsed text, chunks at four scales,
        intent labels, embeddings, clusters, graph edges. Rebuildable from Tier 1 at any time.
      </p>
      <p>
        <strong>Tier 3 — Wiki.</strong> The LLM-curated top layer. Audience-aware. Hot citations bind every
        sentence to the raw source it came from.
      </p>
    </div>
    <div class="visual-card hiw-visual">
      <TierStack />
    </div>
  </div>
</section>

<section class="hiw-section hiw-section--alt">
  <div class="container">
    <span class="eyebrow">Pipeline</span>
    <h2>Nine stages. One contract per boundary.</h2>
    <p class="lede">
      Each stage has a single responsibility and a typed contract with the next. Components are swappable.
      The dashed feedback loop is the wiki telling Intake what to prioritize next.
    </p>
    <div class="visual-card">
      <Pipeline />
    </div>
  </div>
</section>

<section class="hiw-section">
  <div class="container">
    <span class="eyebrow">Retrieval</span>
    <h2>Wiki first. Search second. Envelope always.</h2>
    <p class="lede">
      A query asks the wiki before searching anything. If a curated page exists for this audience, you get an
      envelope back in roughly twelve milliseconds. Otherwise we fall through to hybrid vector + BM25 search and
      compose. The receiver always gets exactly one envelope: ACL-scoped, audience-shaped, bandwidth-matched.
    </p>
    <div class="visual-card">
      <RetrievalFlow />
    </div>
  </div>
</section>

<section class="hiw-section hiw-section--alt">
  <div class="container">
    <span class="eyebrow">Why this works</span>
    <h2>Discover once. Curate forever.</h2>
    <p class="lede">
      Classical RAG re-discovers the same facts on every query. Navigation agents walk the same sources every
      time. Optimal Engine does the heavy work at ingest, keeps a curated layer the agent reads first, and
      composes a fitted envelope on the way out. The hot path stays under 200&nbsp;ms because most queries hit
      curated knowledge.
    </p>
  </div>
</section>

<style>
  .hiw-hero {
    padding: 4rem 0 2rem;
  }
  .hiw-hero h1 {
    font-size: clamp(2rem, 4vw, 3rem);
    line-height: 1.1;
    letter-spacing: -0.01em;
    margin: 0.6rem 0 1rem;
  }
  .lede {
    color: var(--text-muted);
    font-size: 1.05rem;
    line-height: 1.6;
    max-width: 760px;
    margin: 0 0 2rem;
  }
  .lede strong { color: var(--text); font-weight: 600; }

  .hiw-section {
    padding: 4rem 0;
  }
  .hiw-section--alt {
    background: linear-gradient(180deg, var(--bg-elevated) 0%, var(--bg) 100%);
    border-block: 1px solid var(--border-soft);
  }
  .hiw-section h2 {
    font-size: clamp(1.6rem, 2.6vw, 2.2rem);
    line-height: 1.2;
    margin: 0.4rem 0 0.6rem;
    letter-spacing: -0.005em;
  }
  .hiw-section p {
    color: var(--text-muted);
    line-height: 1.6;
    max-width: 720px;
  }
  .hiw-section p strong { color: var(--text); font-weight: 600; }

  .hiw-split {
    display: grid;
    grid-template-columns: 1fr 1.05fr;
    gap: 2.5rem;
    align-items: start;
  }
  .hiw-copy { padding-top: 0.5rem; }

  .visual-card {
    background:
      radial-gradient(closest-side at 30% 0%, rgba(126, 168, 255, 0.06), transparent 70%),
      var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: 16px;
    padding: 2rem;
    box-shadow: 0 30px 60px -30px rgba(0, 0, 0, 0.6);
  }

  @media (max-width: 880px) {
    .hiw-split { grid-template-columns: 1fr; }
    .visual-card { padding: 1rem; }
  }
</style>
