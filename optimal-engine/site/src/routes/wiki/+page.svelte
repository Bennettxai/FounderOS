<script lang="ts">
  // Public showcase of the Wiki concept (not the live wiki). Walks
  // through what a curated page is, what hot citations are, and what
  // executable directives look like — using a real-shaped example.

  const sample = {
    slug: 'customer portal-pricing-decision',
    audience: 'sales',
    body: `# Customer Portal partner pricing — Q4

**Decision:** $2,000 per seat per year, 3-year term with annual price protection. [^1] [^2]

**Why $2K not $2.4K:** anchor is incumbent vendor at $2,400. We underprice deliberately to win the
three-year lock-in. [^3]

**Volume forecast:** 500 seats at go-live; $1M ARR.

**Renewal posture:** front-loaded one quarter on signing; annual renewal review every Q3. [^4]

> /audience: sales
> /executable: regenerate-on Slack:#deals events, JIRA priority bumps`,
    citations: [
      { id: 1, source: 'optimal://nodes/operation-weekly-review/signals/2026-04-11-customer-pricing-call.md' },
      { id: 2, source: 'optimal://nodes/product-customer-portal/signals/2026-04-10-microvm-spec.md' },
      { id: 3, source: 'optimal://nodes/project-platform-launch/signals/2026-04-10-first-customer portal-close.md' },
      { id: 4, source: 'optimal://nodes/project-platform-launch/signals/2026-04-15-renewal-pipeline.md' }
    ]
  };
</script>

<svelte:head>
  <title>Wiki — Optimal Engine</title>
</svelte:head>

<section class="wiki-hero">
  <div class="container">
    <span class="eyebrow">Tier 3 · Wiki — your company's working memory</span>
    <h1>The curated layer the agent reads first.</h1>
    <p class="lede">
      A second brain isn't a pile of files — it's a <strong>working theory</strong> of what your company
      knows. The wiki is that theory, an <strong>LLM-maintained</strong> top tier with three load-bearing
      properties: every sentence is anchored to its raw source via <strong>hot citations</strong>, every page is
      shaped for a specific <strong>audience</strong>, and pages can carry <strong>executable directives</strong>
      that tell the engine when to regenerate them.
    </p>
  </div>
</section>

<section class="wiki-sample">
  <div class="container">
    <span class="eyebrow">A real wiki page</span>
    <h2>What a curated page looks like.</h2>

    <div class="wiki-page">
      <header class="wiki-page__head">
        <div class="wiki-page__slug"><code>{sample.slug}</code></div>
        <div class="wiki-page__meta">
          <span class="chip chip--audience">audience: {sample.audience}</span>
          <span class="chip chip--curated">curated</span>
        </div>
      </header>

      <pre class="wiki-page__body">{sample.body}</pre>

      <footer class="wiki-page__cite">
        <div class="eyebrow">Hot citations</div>
        <ol>
          {#each sample.citations as c}
            <li>
              <span class="cite-id">[{c.id}]</span>
              <code>{c.source}</code>
            </li>
          {/each}
        </ol>
      </footer>
    </div>
  </div>
</section>

<section class="wiki-explain">
  <div class="container wiki-three">
    <div class="wiki-feat">
      <div class="wiki-feat__num">01</div>
      <h3>Hot citations</h3>
      <p>Every footnoted claim is anchored to a raw signal. When the source changes, the engine knows which
        wiki sentences to re-curate.</p>
    </div>
    <div class="wiki-feat">
      <div class="wiki-feat__num">02</div>
      <h3>Audience-aware</h3>
      <p>The same underlying knowledge yields different wiki pages for sales / legal / exec / engineering.
        Same facts, different framing, different bandwidth.</p>
    </div>
    <div class="wiki-feat">
      <div class="wiki-feat__num">03</div>
      <h3>Executable directives</h3>
      <p>Pages carry rules — <code>regenerate-on</code>, <code>watch</code>, <code>expire-after</code> — that
        let the engine maintain them without a human in the loop.</p>
    </div>
  </div>
</section>

<style>
  .wiki-hero { padding: 4rem 0 2rem; }
  .wiki-hero h1 {
    font-size: clamp(2rem, 4vw, 3rem);
    line-height: 1.1;
    margin: 0.6rem 0 1rem;
    letter-spacing: -0.01em;
  }
  .lede {
    color: var(--text-muted);
    font-size: 1.05rem;
    line-height: 1.65;
    max-width: 760px;
    margin: 0 0 1rem;
  }
  .lede strong { color: var(--text); font-weight: 600; }

  .wiki-sample { padding: 2rem 0 3rem; }
  .wiki-sample h2 {
    font-size: clamp(1.5rem, 2.4vw, 1.9rem);
    margin: 0.4rem 0 1.5rem;
    letter-spacing: -0.005em;
  }

  .wiki-page {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: 14px;
    overflow: hidden;
  }
  .wiki-page__head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.85rem 1.1rem;
    border-bottom: 1px solid var(--border-soft);
    background: var(--bg);
  }
  .wiki-page__slug code {
    font-size: 0.85rem;
    color: var(--text);
  }
  .wiki-page__meta {
    display: flex;
    gap: 0.5rem;
  }
  .chip {
    padding: 3px 10px;
    border-radius: 999px;
    font-size: 0.72rem;
    font-weight: 600;
    text-transform: lowercase;
  }
  .chip--audience { background: rgba(126, 168, 255, 0.12); color: var(--accent); }
  .chip--curated  { background: rgba(123, 227, 163, 0.10); color: #7be3a3; }

  .wiki-page__body {
    margin: 0;
    padding: 1.4rem 1.6rem;
    color: var(--text);
    font-family: var(--font-sans);
    font-size: 0.95rem;
    line-height: 1.65;
    white-space: pre-wrap;
  }
  .wiki-page__cite {
    border-top: 1px solid var(--border-soft);
    padding: 1rem 1.6rem 1.2rem;
    background: var(--bg);
  }
  .wiki-page__cite ol {
    list-style: none;
    margin: 0.5rem 0 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }
  .wiki-page__cite li {
    display: flex;
    gap: 0.6rem;
    font-size: 0.82rem;
  }
  .cite-id {
    color: var(--accent);
    font-weight: 600;
    flex-shrink: 0;
  }
  .wiki-page__cite code {
    color: var(--text-muted);
    font-family: var(--font-mono);
    font-size: 0.78rem;
    word-break: break-all;
  }

  .wiki-explain {
    padding: 3rem 0 4rem;
    background: linear-gradient(180deg, var(--bg-elevated) 0%, var(--bg) 100%);
    border-block: 1px solid var(--border-soft);
  }
  .wiki-three {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 1rem;
  }
  .wiki-feat {
    background: var(--bg-elevated);
    border: 1px solid var(--border);
    border-radius: 12px;
    padding: 1.4rem 1.4rem 1.6rem;
  }
  .wiki-feat__num {
    color: var(--accent);
    font-size: 0.78rem;
    font-weight: 700;
    letter-spacing: 0.12em;
  }
  .wiki-feat h3 {
    margin: 0.4rem 0 0.5rem;
    font-size: 1.05rem;
  }
  .wiki-feat p {
    color: var(--text-muted);
    margin: 0;
    font-size: 0.9rem;
    line-height: 1.55;
  }
  .wiki-feat code {
    background: var(--bg);
    padding: 1px 6px;
    border-radius: 4px;
    border: 1px solid var(--border);
    font-size: 0.78rem;
    color: var(--text);
  }

  @media (max-width: 720px) {
    .wiki-three { grid-template-columns: 1fr; }
  }
</style>
