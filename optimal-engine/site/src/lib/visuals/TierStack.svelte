<script lang="ts">
  // Animated SVG showing the 3-tier memory architecture.
  //
  //   Tier 3 — Wiki (curated, audience-aware)
  //       ▲ DERIVE       ▼ CURATE
  //   Tier 2 — Derivatives (machine-maintained)
  //       ▲ DERIVE       ▼ CURATE
  //   Tier 1 — Raw sources (immutable, append-only)
  //
  // Hot-citation lines connect a wiki page to its underlying derivatives
  // and raw signals. Small source-type icons sit inside Tier 1.

  let active = $state(2); // 0=raw, 1=derivatives, 2=wiki

  const tiers = [
    { id: 0, name: 'Raw sources',  tag: 'Immutable · append-only',     y: 250 },
    { id: 1, name: 'Derivatives',  tag: 'Machine-maintained',          y: 140 },
    { id: 2, name: 'Wiki',         tag: 'LLM-curated · audience-aware', y: 30  }
  ];

  // Source-type glyphs in Tier 1
  const sources = ['chat', 'mail', 'doc', 'img', 'mic', 'tkt'];
</script>

<div class="tier-stack">
  <svg viewBox="0 0 560 360" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Three-tier memory architecture: raw sources, derivatives, wiki">
    <defs>
      <linearGradient id="ts-tier0" x1="0" x2="1" y1="0" y2="0">
        <stop offset="0%"  stop-color="#1a1f2a"/>
        <stop offset="100%" stop-color="#0f131a"/>
      </linearGradient>
      <linearGradient id="ts-tier1" x1="0" x2="1" y1="0" y2="0">
        <stop offset="0%"  stop-color="#1f2a3d"/>
        <stop offset="100%" stop-color="#131a26"/>
      </linearGradient>
      <linearGradient id="ts-tier2" x1="0" x2="1" y1="0" y2="0">
        <stop offset="0%"  stop-color="#2a3a5e"/>
        <stop offset="100%" stop-color="#1a233c"/>
      </linearGradient>
      <linearGradient id="ts-glow" x1="0" x2="0" y1="0" y2="1">
        <stop offset="0%"  stop-color="#7ea8ff" stop-opacity="0.5"/>
        <stop offset="100%" stop-color="#7ea8ff" stop-opacity="0"/>
      </linearGradient>
      <marker id="ts-arrow-down" markerWidth="10" markerHeight="10" refX="5" refY="9" orient="auto">
        <path d="M0,0 L10,0 L5,9 Z" fill="#7ea8ff"/>
      </marker>
      <marker id="ts-arrow-up" markerWidth="10" markerHeight="10" refX="5" refY="1" orient="auto">
        <path d="M0,10 L10,10 L5,1 Z" fill="#bb7eff"/>
      </marker>
    </defs>

    <!-- Hot citation lines: wiki page anchored to derivatives + raw -->
    <g class="ts-citations" stroke="#7ea8ff" stroke-width="1" fill="none" opacity="0.35">
      <path d="M120,75 C 140,110 200,160 240,185" stroke-dasharray="2 3"/>
      <path d="M120,75 C 160,140 280,250 340,290" stroke-dasharray="2 3"/>
      <path d="M120,75 C 100,120 130,200 180,290" stroke-dasharray="2 3"/>
    </g>

    <!-- Tier 1: Raw -->
    <g
      class="ts-tier"
      class:active={active === 0}
      onmouseenter={() => (active = 0)}
      role="button"
      tabindex="0"
    >
      <rect x="60" y="250" width="440" height="80" rx="10" fill="url(#ts-tier0)" stroke="#1c2230" stroke-width="1"/>
      <text x="80" y="278" fill="#9aa0a8" font-size="10" font-weight="600" letter-spacing="1.2">TIER 1</text>
      <text x="80" y="298" fill="#f1f1f3" font-size="16" font-weight="600">Raw sources</text>
      <text x="80" y="316" fill="#6b7280" font-size="11">Immutable · append-only · hash-addressed</text>

      <!-- Source-type chips -->
      <g class="ts-sources" font-size="9" fill="#9aa0a8" font-weight="500">
        {#each sources as s, i}
          <g transform="translate({270 + i * 36}, 285)">
            <rect x="0" y="0" width="30" height="22" rx="4" fill="#0d1015" stroke="#1c2230"/>
            <text x="15" y="14" text-anchor="middle">{s}</text>
          </g>
        {/each}
      </g>
    </g>

    <!-- Tier 2: Derivatives -->
    <g
      class="ts-tier"
      class:active={active === 1}
      onmouseenter={() => (active = 1)}
      role="button"
      tabindex="0"
    >
      <rect x="60" y="140" width="440" height="80" rx="10" fill="url(#ts-tier1)" stroke="#243044" stroke-width="1"/>
      <text x="80" y="168" fill="#9aa0a8" font-size="10" font-weight="600" letter-spacing="1.2">TIER 2</text>
      <text x="80" y="188" fill="#f1f1f3" font-size="16" font-weight="600">Derivatives</text>
      <text x="80" y="206" fill="#6b7280" font-size="11">SQLite · FTS5 · vectors · graph · clusters</text>

      <!-- Cluster blob glyphs -->
      <g class="ts-clusters">
        {#each [0,1,2,3,4] as i}
          <circle cx={310 + i * 32} cy={180} r={4 + (i % 3)} fill="#7ea8ff" opacity="0.5"/>
          <circle cx={325 + i * 32} cy={195} r={2 + (i % 2)} fill="#7ea8ff" opacity="0.35"/>
          <circle cx={300 + i * 32} cy={170} r="3" fill="#7ea8ff" opacity="0.25"/>
        {/each}
      </g>
    </g>

    <!-- Tier 3: Wiki -->
    <g
      class="ts-tier"
      class:active={active === 2}
      onmouseenter={() => (active = 2)}
      role="button"
      tabindex="0"
    >
      <rect x="60" y="30" width="440" height="80" rx="10" fill="url(#ts-tier2)" stroke="#3a4a72" stroke-width="1"/>
      <text x="80" y="58" fill="#9aa0a8" font-size="10" font-weight="600" letter-spacing="1.2">TIER 3</text>
      <text x="80" y="78" fill="#f1f1f3" font-size="16" font-weight="600">Wiki</text>
      <text x="80" y="96" fill="#9aa0a8" font-size="11">LLM-maintained · audience-aware · read first</text>

      <!-- Wiki page glyph -->
      <g transform="translate(390, 50)">
        <rect width="80" height="44" rx="4" fill="#0d1015" stroke="#3a4a72"/>
        <line x1="8" y1="10" x2="60" y2="10" stroke="#7ea8ff" stroke-width="2"/>
        <line x1="8" y1="18" x2="68" y2="18" stroke="#3a4a72" stroke-width="1"/>
        <line x1="8" y1="24" x2="55" y2="24" stroke="#3a4a72" stroke-width="1"/>
        <line x1="8" y1="30" x2="64" y2="30" stroke="#3a4a72" stroke-width="1"/>
        <line x1="8" y1="36" x2="40" y2="36" stroke="#3a4a72" stroke-width="1"/>
        <circle cx="68" cy="36" r="2" fill="#7ea8ff"/>
      </g>
    </g>

    <!-- DERIVE arrow (going up, left side) -->
    <g class="ts-flow ts-flow--up">
      <line
        x1="36" y1="320" x2="36" y2="50"
        stroke="#bb7eff" stroke-width="1.5" stroke-dasharray="6 4"
        marker-end="url(#ts-arrow-up)"
      />
      <text x="14" y="200" fill="#bb7eff" font-size="10" font-weight="600" letter-spacing="2" transform="rotate(-90 14 200)">DERIVE</text>
    </g>

    <!-- CURATE arrow (going down, right side) -->
    <g class="ts-flow ts-flow--down">
      <line
        x1="524" y1="50" x2="524" y2="320"
        stroke="#7ea8ff" stroke-width="1.5" stroke-dasharray="6 4"
        marker-end="url(#ts-arrow-down)"
      />
      <text x="540" y="190" fill="#7ea8ff" font-size="10" font-weight="600" letter-spacing="2" transform="rotate(90 540 190)">CURATE</text>
    </g>
  </svg>

  <div class="tier-stack__caption">
    <strong>Information flows up.</strong> Raw signals get parsed, decomposed, embedded, clustered.
    <strong>Intent flows down.</strong> The wiki tells the lower tiers what matters and why.
    Hot citations bind every wiki sentence back to the raw source it came from.
  </div>
</div>

<style>
  .tier-stack {
    width: 100%;
  }
  .tier-stack svg {
    width: 100%;
    height: auto;
    display: block;
  }
  .ts-tier {
    cursor: pointer;
    transition: transform 0.2s ease, filter 0.2s ease;
    transform-origin: center;
  }
  .ts-tier:hover,
  .ts-tier.active {
    filter: drop-shadow(0 0 18px rgba(126, 168, 255, 0.35));
  }
  .ts-tier.active rect {
    stroke: var(--accent);
  }
  .ts-flow line {
    animation: dash 1.6s linear infinite;
  }
  @keyframes dash {
    to { stroke-dashoffset: -20; }
  }
  .ts-clusters circle {
    animation: pulse 2.4s ease-in-out infinite;
  }
  .ts-clusters circle:nth-child(3n) { animation-delay: 0.4s; }
  .ts-clusters circle:nth-child(3n+1) { animation-delay: 0.8s; }
  @keyframes pulse {
    0%, 100% { opacity: 0.25; }
    50%      { opacity: 0.6; }
  }

  .tier-stack__caption {
    margin-top: 1.2rem;
    color: var(--text-muted);
    font-size: 0.9rem;
    line-height: 1.6;
    max-width: 720px;
  }
  .tier-stack__caption strong {
    color: var(--text);
    font-weight: 600;
  }

  @media (prefers-reduced-motion: reduce) {
    .ts-flow line, .ts-clusters circle { animation: none; }
  }
</style>
