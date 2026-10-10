<script lang="ts">
  import '../app.css';
  import { page } from '$app/stores';

  let { children } = $props<{ children?: () => any }>();

  function isActive(path: string): boolean {
    if (path === '/') return $page.url.pathname === '/';
    return $page.url.pathname.startsWith(path);
  }
</script>

<header class="site-header glass">
  <div class="site-header__inner container">
    <a class="site-header__brand" href="/">
      <span class="site-header__mark">◇</span>
      <span class="site-header__name">Optimal Engine</span>
    </a>
    <nav class="site-header__nav">
      <a href="/how-it-works" class:active={isActive('/how-it-works')}>How it works</a>
      <a href="/connectors" class:active={isActive('/connectors')}>Connectors</a>
      <a href="/wiki" class:active={isActive('/wiki')}>Wiki</a>
      <a href="https://github.com/Miosa-osa/OptimalEngine" target="_blank" rel="noopener">GitHub ↗</a>
    </nav>
    <a class="btn btn--primary" href="https://github.com/Miosa-osa/OptimalEngine#quick-start" target="_blank" rel="noopener">
      Get started
    </a>
  </div>
</header>

<main>
  {@render children?.()}
</main>

<footer class="site-footer">
  <div class="container site-footer__inner">
    <div class="site-footer__col">
      <div class="site-footer__brand">◇ Optimal Engine</div>
      <p class="site-footer__tag">The second brain of your company.</p>
    </div>
    <div class="site-footer__col">
      <div class="eyebrow">Product</div>
      <a href="/how-it-works">How it works</a>
      <a href="/connectors">Connectors</a>
      <a href="/wiki">Wiki</a>
    </div>
    <div class="site-footer__col">
      <div class="eyebrow">Develop</div>
      <a href="https://github.com/Miosa-osa/OptimalEngine" target="_blank" rel="noopener">GitHub</a>
      <a href="https://github.com/Miosa-osa/OptimalEngine/blob/main/docs/architecture/ENGINE-STRUCTURE.md" target="_blank" rel="noopener">Architecture</a>
      <a href="https://github.com/Miosa-osa/OptimalEngine/blob/main/docs/guides/getting-started.md" target="_blank" rel="noopener">Getting started</a>
    </div>
    <div class="site-footer__col">
      <div class="eyebrow">License</div>
      <span class="site-footer__muted">MIT</span>
    </div>
  </div>
</footer>

<style>
  .site-header {
    position: sticky;
    top: 0;
    z-index: 50;
    border-radius: 0;
    border-left: none;
    border-right: none;
    border-top: none;
  }
  .site-header__inner {
    display: flex;
    align-items: center;
    gap: 1.25rem;
    padding: 0.7rem 1.5rem;
  }
  .site-header__brand {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    color: var(--text);
    font-weight: 600;
    text-decoration: none;
  }
  .site-header__brand:hover { text-decoration: none; }
  .site-header__mark {
    color: var(--accent);
    font-size: 1.05rem;
  }
  .site-header__name {
    font-size: 0.95rem;
    letter-spacing: 0.2px;
  }
  .site-header__nav {
    flex: 1;
    display: flex;
    gap: 0.25rem;
    margin-left: 1rem;
  }
  .site-header__nav a {
    color: var(--text-muted);
    font-size: 0.85rem;
    padding: 0.35rem 0.7rem;
    border-radius: 6px;
    transition: background 0.12s, color 0.12s;
  }
  .site-header__nav a:hover {
    color: var(--text);
    background: var(--bg-elevated-2);
    text-decoration: none;
  }
  .site-header__nav a.active {
    color: var(--accent);
    background: var(--accent-soft);
  }

  .site-footer {
    border-top: 1px solid var(--border);
    margin-top: 4rem;
    padding: 3rem 0 2rem;
    background: var(--bg);
  }
  .site-footer__inner {
    display: grid;
    grid-template-columns: 1.5fr repeat(3, 1fr);
    gap: 2rem;
  }
  .site-footer__col {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }
  .site-footer__brand {
    font-weight: 600;
    color: var(--text);
  }
  .site-footer__tag {
    color: var(--text-muted);
    margin: 0.25rem 0 0;
    font-size: 0.85rem;
  }
  .site-footer__col a {
    color: var(--text-muted);
    font-size: 0.85rem;
  }
  .site-footer__col a:hover {
    color: var(--text);
  }
  .site-footer__muted {
    color: var(--text-subtle);
    font-size: 0.85rem;
  }

  @media (max-width: 720px) {
    .site-header__nav { display: none; }
    .site-footer__inner { grid-template-columns: 1fr 1fr; }
  }
</style>
