<script lang="ts">
  import type { EngineContext } from '../engine.js';
  import type { WikiArticle } from '@optimal-engine/client';
  import { useWiki } from '../hooks/useWiki.svelte.js';

  interface Props {
    engine: EngineContext;
    slug?: string;
    audience?: string;
  }

  let { engine, slug, audience }: Props = $props();

  const wiki = useWiki(engine);

  let showSources = $state(false);

  // When a slug prop is provided, load it immediately
  $effect(() => {
    if (slug) {
      void wiki.openPage(slug, audience);
    }
  });

  function fmtDate(iso?: string): string {
    if (!iso) return '';
    return new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' });
  }

  function parseSources(content: string): string[] {
    const matches = content.matchAll(/\{\{cite:\s*(optimal:\/\/[^\}]+)\}\}/g);
    return [...new Set([...matches].map(m => m[1]))];
  }

  /**
   * Parse wiki content into typed tokens for rendering.
   */
  type Token =
    | { kind: 'heading'; level: 1 | 2 | 3; text: string }
    | { kind: 'directive'; raw: string; verb: string; value: string }
    | { kind: 'paragraph'; parts: InlinePart[] };

  type InlinePart =
    | { type: 'text'; value: string }
    | { type: 'cite'; uri: string }
    | { type: 'wikilink'; target: string }
    | { type: 'bold'; value: string }
    | { type: 'italic'; value: string };

  function parseInline(text: string): InlinePart[] {
    const parts: InlinePart[] = [];
    // Combined regex for all inline patterns
    const re = /(\{\{cite:\s*(optimal:\/\/[^\}]+)\}\}|\[\[([^\]]+)\]\]|\*\*([^*]+)\*\*|_([^_]+)_)/g;
    let last = 0;
    let m: RegExpExecArray | null;
    while ((m = re.exec(text)) !== null) {
      if (m.index > last) {
        parts.push({ type: 'text', value: text.slice(last, m.index) });
      }
      if (m[2]) {
        parts.push({ type: 'cite', uri: m[2].trim() });
      } else if (m[3]) {
        parts.push({ type: 'wikilink', target: m[3] });
      } else if (m[4]) {
        parts.push({ type: 'bold', value: m[4] });
      } else if (m[5]) {
        parts.push({ type: 'italic', value: m[5] });
      }
      last = re.lastIndex;
    }
    if (last < text.length) {
      parts.push({ type: 'text', value: text.slice(last) });
    }
    return parts;
  }

  function parseContent(content: string): Token[] {
    const tokens: Token[] = [];
    const lines = content.split('\n');
    let i = 0;
    while (i < lines.length) {
      const line = lines[i];

      // Headings
      const h3 = line.match(/^###\s+(.*)/);
      const h2 = line.match(/^##\s+(.*)/);
      const h1 = line.match(/^#\s+(.*)/);
      if (h1) { tokens.push({ kind: 'heading', level: 1, text: h1[1] }); i++; continue; }
      if (h2) { tokens.push({ kind: 'heading', level: 2, text: h2[1] }); i++; continue; }
      if (h3) { tokens.push({ kind: 'heading', level: 3, text: h3[1] }); i++; continue; }

      // Directives: {{include:...}} / {{expand:...}}
      const dir = line.match(/^\{\{(include|expand):\s*([^\}]+)\}\}/);
      if (dir) {
        tokens.push({ kind: 'directive', raw: line, verb: dir[1], value: dir[2].trim() });
        i++; continue;
      }

      // Skip blank lines
      if (line.trim() === '') { i++; continue; }

      // Paragraph — accumulate until blank or heading
      const parts = parseInline(line);
      tokens.push({ kind: 'paragraph', parts });
      i++;
    }
    return tokens;
  }

  const tokens = $derived(
    wiki.selected?.content ? parseContent(wiki.selected.content) : []
  );

  const sources = $derived(
    wiki.selected?.content ? parseSources(wiki.selected.content) : []
  );

  function handleWikilink(target: string) {
    void wiki.openPage(target, audience);
  }
</script>

<div class="wv-root">
  {#if wiki.loading}
    <!-- Loading skeleton -->
    <div class="wv-skeleton">
      <div class="wv-skeleton__title"></div>
      <div class="wv-skeleton__meta"></div>
      {#each [1, 2, 3, 4] as n (n)}
        <div class="wv-skeleton__line" style="width: {70 + (n % 3) * 10}%"></div>
      {/each}
    </div>
  {:else if wiki.error}
    <div class="wv-error">{wiki.error}</div>
  {:else if !wiki.selected}
    <!-- Page list -->
    {#if wiki.pages.length === 0}
      <div class="oe-empty">
        <span class="oe-empty__icon">📄</span>
        <span>No wiki pages in this workspace.</span>
      </div>
    {:else}
      <div class="wv-list" role="list">
        {#each wiki.pages as page (page.slug)}
          <button
            class="wv-list-item"
            onclick={() => wiki.openPage(page.slug, audience)}
            aria-label="Open wiki page: {page.title ?? page.slug}"
          >
            <span class="wv-list-item__slug">{page.slug}</span>
            {#if page.title && page.title !== page.slug}
              <span class="wv-list-item__title">{page.title}</span>
            {/if}
            {#if page.audience}
              <span class="oe-chip oe-chip--audience wv-list-audience">{page.audience}</span>
            {/if}
          </button>
        {/each}
      </div>
    {/if}
  {:else}
    <!-- Page view -->
    {@const page = wiki.selected as WikiArticle & { last_curated?: string; version?: string }}

    <!-- Back button (only if navigated in, not from slug prop) -->
    {#if !slug}
      <button class="wv-back" onclick={() => { wiki.selected; void wiki.refresh(); }} aria-label="Back to wiki list">
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none" aria-hidden="true">
          <path d="M8 2L4 6l4 4" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" stroke-linejoin="round"/>
        </svg>
        Wiki
      </button>
    {/if}

    <!-- Page header -->
    <div class="oe-wiki-meta">
      <span class="wv-page-title">{page.title ?? page.slug}</span>
      {#if page.audience}
        <span class="oe-chip oe-chip--audience">{page.audience}</span>
      {/if}
      {#if page.version}
        <span class="oe-chip oe-chip--version">v{page.version as string}</span>
      {/if}
      {#if page.last_curated}
        <span class="wv-meta-ts">Curated {fmtDate(page.last_curated)}</span>
      {/if}
    </div>

    <!-- Page body blocks -->
    <div class="oe-wiki-body">
      {#each tokens as token (JSON.stringify(token))}
        {#if token.kind === 'heading'}
          <div class="oe-heading oe-heading--h{token.level}">{token.text}</div>
        {:else if token.kind === 'directive'}
          <div class="oe-include oe-directive">
            <span class="wv-directive-verb">{'{{' + token.verb + ': '}</span>{token.value}{'}}'}
          </div>
        {:else if token.kind === 'paragraph'}
          <div class="oe-block oe-wiki-para">
            {#each token.parts as part}
              {#if part.type === 'text'}{part.value}{/if}
              {#if part.type === 'cite'}
                <span class="oe-cite" title={part.uri} aria-label="Citation: {part.uri}">{part.uri.replace('optimal://', '')}</span>
              {/if}
              {#if part.type === 'wikilink'}
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <span class="oe-wikilink" role="link" tabindex="0"
                  onclick={() => handleWikilink(part.target)}
                  onkeydown={(e) => { if (e.key === 'Enter') handleWikilink(part.target); }}>
                  {part.target}
                </span>
              {/if}
              {#if part.type === 'bold'}<strong>{part.value}</strong>{/if}
              {#if part.type === 'italic'}<em>{part.value}</em>{/if}
            {/each}
          </div>
        {/if}
      {/each}
    </div>

    <!-- Sources footer -->
    {#if sources.length > 0}
      <div class="oe-wiki-sources">
        <button
          class="oe-wiki-sources__title wv-sources-toggle"
          onclick={() => showSources = !showSources}
          aria-expanded={showSources}
        >
          Sources ({sources.length}) {showSources ? '▴' : '▾'}
        </button>
        {#if showSources}
          <div class="wv-sources-list">
            {#each sources as uri (uri)}
              <a
                href={uri}
                target="_blank"
                rel="noopener noreferrer"
                class="oe-wiki-source-chip"
              >
                {uri}
              </a>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  {/if}
</div>

<style>
  .wv-root {
    display: flex;
    flex-direction: column;
  }

  /* List view */
  .wv-list {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
  }

  .wv-list-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.45rem 0.6rem;
    border-radius: 6px;
    border: none;
    background: none;
    text-align: left;
    cursor: pointer;
    transition: background 0.1s;
    width: 100%;
  }

  .wv-list-item:hover {
    background: var(--dbg2, #131820);
  }

  .wv-list-item__slug {
    font-size: 0.84rem;
    font-family: monospace;
    color: var(--daccent, #7ea8ff);
  }

  .wv-list-item__title {
    font-size: 0.84rem;
    color: var(--dt2, #ccc);
  }

  .wv-list-audience {
    margin-left: auto;
  }

  /* Back button */
  .wv-back {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0.2rem 0.5rem;
    border: none;
    background: none;
    font-size: 0.78rem;
    color: var(--dt3, #888);
    cursor: pointer;
    border-radius: 4px;
    margin-bottom: 0.5rem;
    transition: color 0.1s;
  }

  .wv-back:hover {
    color: var(--dt, #f1f1f3);
  }

  /* Page header */
  .wv-page-title {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--dt, #f1f1f3);
  }

  .wv-meta-ts {
    font-size: 0.72rem;
    color: var(--dt4, #555);
    margin-left: auto;
  }

  /* Directive block */
  .oe-directive {
    background: rgba(187, 126, 255, 0.07);
    border-left: 2px solid #bb7eff;
    font-style: italic;
    font-size: 0.82rem;
    color: var(--dt3, #888);
  }

  .wv-directive-verb {
    font-family: monospace;
    color: #bb7eff;
    font-style: normal;
  }

  /* Sources toggle */
  .wv-sources-toggle {
    background: none;
    border: none;
    cursor: pointer;
    padding: 0;
    font-size: 0.72rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    color: var(--dt4, #555);
    transition: color 0.1s;
  }

  .wv-sources-toggle:hover {
    color: var(--dt3, #888);
  }

  .wv-sources-list {
    margin-top: 0.4rem;
    display: flex;
    flex-wrap: wrap;
    gap: 0.25rem;
  }

  /* Error */
  .wv-error {
    padding: 0.4rem 0.65rem;
    border-radius: 5px;
    background: rgba(248,136,136,0.08);
    border: 1px solid rgba(248,136,136,0.2);
    font-size: 0.82rem;
    color: #f88;
  }

  /* Skeleton */
  .wv-skeleton {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    padding: 0.5rem 0;
  }

  .wv-skeleton__title {
    height: 18px;
    width: 45%;
    border-radius: 4px;
    background: var(--dbg2, #131820);
    animation: wv-pulse 1.4s ease-in-out infinite;
  }

  .wv-skeleton__meta {
    height: 10px;
    width: 30%;
    border-radius: 4px;
    background: var(--dbg2, #131820);
    animation: wv-pulse 1.4s ease-in-out infinite;
  }

  .wv-skeleton__line {
    height: 10px;
    border-radius: 4px;
    background: var(--dbg2, #131820);
    animation: wv-pulse 1.4s ease-in-out infinite;
  }

  @keyframes wv-pulse {
    0%, 100% { opacity: 0.4; }
    50% { opacity: 0.8; }
  }
</style>
