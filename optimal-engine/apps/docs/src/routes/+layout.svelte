<script lang="ts">
  import '../app.css';
  import EngineThemeToggle from '$lib/components/EngineThemeToggle.svelte';

  let { children } = $props();
</script>

<div class="shell">
  <header class="glass-header">
    <div class="header-inner">
      <a href="/" class="wordmark">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" aria-hidden="true">
          <circle cx="12" cy="12" r="10" stroke="var(--accent)" stroke-width="1.5" />
          <path d="M8 12h8M12 8v8" stroke="var(--accent)" stroke-width="1.5" stroke-linecap="round" />
        </svg>
        <span>Optimal Engine</span>
        <span class="badge">docs index</span>
      </a>

      <nav class="header-nav" aria-label="Primary">
        <a href="https://github.com/Miosa-osa/OptimalEngine/blob/main/README.md" class="header-link">README</a>
        <a href="https://github.com/Miosa-osa/OptimalEngine/tree/main/docs" class="header-link">Docs</a>
        <a href="https://github.com/Miosa-osa/OptimalEngine/blob/main/docs/reference/backend-readiness.md" class="header-link">Backend</a>
      </nav>

      <div class="header-actions">
        <a
          href="https://github.com/Miosa-osa/OptimalEngine"
          target="_blank"
          rel="noopener"
          class="pill pill-ghost"
          aria-label="GitHub"
        >
          GitHub
        </a>
        <EngineThemeToggle />
      </div>
    </div>
  </header>

  <main class="main-content">
    <div class="content-inner">
      {@render children()}
    </div>
  </main>
</div>

<style>
  .shell {
    display: flex;
    flex-direction: column;
    min-height: 100vh;
  }

  .glass-header {
    position: sticky;
    top: 0;
    z-index: 100;
    height: var(--header-height);
    background: var(--bg-glass);
    backdrop-filter: blur(16px) saturate(1.4);
    -webkit-backdrop-filter: blur(16px) saturate(1.4);
    border-bottom: 1px solid var(--border);
  }

  .header-inner {
    max-width: 1100px;
    margin: 0 auto;
    padding: 0 1.5rem;
    height: 100%;
    display: flex;
    align-items: center;
    gap: 2rem;
  }

  .wordmark {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-weight: 600;
    font-size: 0.9375rem;
    color: var(--text);
    text-decoration: none;
    flex-shrink: 0;
  }

  .wordmark:hover {
    text-decoration: none;
    color: var(--text);
  }

  .badge {
    font-size: 0.6875rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    background: var(--accent-dim);
    color: var(--accent);
    padding: 0.125rem 0.4rem;
    border-radius: var(--r-pill);
    border: 1px solid var(--border-accent);
  }

  .header-nav {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    flex: 1;
  }

  .header-link {
    font-size: 0.875rem;
    color: var(--text-muted);
    padding: 0.375rem 0.625rem;
    border-radius: var(--r-sm);
    text-decoration: none;
  }

  .header-link:hover {
    color: var(--text);
    background: var(--bg-hover);
    text-decoration: none;
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-left: auto;
  }

  .main-content {
    flex: 1;
    padding: 3rem 1.5rem;
  }

  .content-inner {
    max-width: 960px;
    margin: 0 auto;
  }

  @media (max-width: 768px) {
    .header-nav {
      display: none;
    }
  }
</style>
