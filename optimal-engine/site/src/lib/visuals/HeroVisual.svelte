<script lang="ts">
  // Compact hero visual: signals flowing from many sources into a central
  // engine, emerging as a bright contextual envelope on the right.
  //
  //   chat ───┐
  //   mail ───┤        ╱─────╲
  //   doc  ───┼──────►(  ENG  )──────► [ ENVELOPE ]
  //   img  ───┤        ╲─────╱
  //   mic  ───┘
  //
  // Animated dashed connectors signal the live pipeline.

  const sources = [
    { label: 'chat', y: 40 },
    { label: 'mail', y: 90 },
    { label: 'doc',  y: 140 },
    { label: 'img',  y: 190 },
    { label: 'mic',  y: 240 },
    { label: 'tkt',  y: 290 }
  ];
</script>

<svg class="hero-visual" viewBox="0 0 480 340" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Signals flowing through the engine into a contextual envelope">
  <defs>
    <radialGradient id="hv-core" cx="50%" cy="50%" r="50%">
      <stop offset="0%"  stop-color="#7ea8ff" stop-opacity="0.85"/>
      <stop offset="60%" stop-color="#2a3a5e" stop-opacity="0.6"/>
      <stop offset="100%" stop-color="#0d1015" stop-opacity="0"/>
    </radialGradient>
    <linearGradient id="hv-env" x1="0" x2="1" y1="0" y2="1">
      <stop offset="0%"  stop-color="#1a233c"/>
      <stop offset="100%" stop-color="#0d1015"/>
    </linearGradient>
    <marker id="hv-arrow" markerWidth="6" markerHeight="6" refX="6" refY="3" orient="auto">
      <path d="M0,0 L6,3 L0,6 Z" fill="#7ea8ff" opacity="0.7"/>
    </marker>
  </defs>

  <!-- Source chips on left -->
  {#each sources as s, i}
    <g class="hv-src" style="--i: {i}">
      <rect x="10" y={s.y} width="50" height="28" rx="14" fill="#0d1015" stroke="#1c2230"/>
      <text x="35" y={s.y + 18} text-anchor="middle" fill="#9aa0a8" font-size="11" font-weight="500">{s.label}</text>
      <line
        x1="60" y1={s.y + 14}
        x2="190" y2="170"
        stroke="#7ea8ff" stroke-opacity="0.35" stroke-width="1.2"
        stroke-dasharray="3 4"
        marker-end="url(#hv-arrow)"
      />
    </g>
  {/each}

  <!-- Central engine glow -->
  <circle cx="220" cy="170" r="80" fill="url(#hv-core)"/>

  <!-- Engine core ring -->
  <circle cx="220" cy="170" r="44" fill="#0d1015" stroke="#7ea8ff" stroke-width="1.5"/>
  <circle cx="220" cy="170" r="58" fill="none" stroke="#7ea8ff" stroke-opacity="0.3" stroke-dasharray="4 6"/>
  <circle cx="220" cy="170" r="72" fill="none" stroke="#7ea8ff" stroke-opacity="0.15" stroke-dasharray="2 8"/>

  <!-- Engine label -->
  <text x="220" y="166" text-anchor="middle" fill="#7ea8ff" font-size="9" font-weight="700" letter-spacing="2">OPTIMAL</text>
  <text x="220" y="180" text-anchor="middle" fill="#f1f1f3" font-size="11" font-weight="600">ENGINE</text>

  <!-- Engine → envelope -->
  <line x1="264" y1="170" x2="350" y2="170" stroke="#7ea8ff" stroke-width="1.5" stroke-dasharray="4 4" marker-end="url(#hv-arrow)"/>

  <!-- Envelope -->
  <g class="hv-env">
    <rect x="350" y="120" width="120" height="100" rx="10" fill="url(#hv-env)" stroke="#7ea8ff"/>
    <text x="362" y="142" fill="#7ea8ff" font-size="8" font-weight="700" letter-spacing="1.5">CONTEXT</text>
    <line x1="362" y1="148" x2="430" y2="148" stroke="#7ea8ff" stroke-opacity="0.4"/>
    <line x1="362" y1="160" x2="455" y2="160" stroke="#3a4a72"/>
    <line x1="362" y1="170" x2="445" y2="170" stroke="#3a4a72"/>
    <line x1="362" y1="180" x2="450" y2="180" stroke="#3a4a72"/>
    <line x1="362" y1="190" x2="420" y2="190" stroke="#3a4a72"/>
    <circle cx="453" cy="208" r="2.5" fill="#7be3a3"/>
    <text x="430" y="212" text-anchor="end" fill="#7be3a3" font-size="9" font-weight="600">&lt;200ms</text>
  </g>
</svg>

<style>
  .hero-visual {
    width: 100%;
    height: auto;
    display: block;
    filter: drop-shadow(0 30px 60px rgba(126, 168, 255, 0.12));
  }

  /* Stagger source line animations */
  .hv-src line {
    animation: hv-flow 2s linear infinite;
  }
  .hv-src:nth-child(1) line { animation-delay: 0s; }
  .hv-src:nth-child(2) line { animation-delay: 0.15s; }
  .hv-src:nth-child(3) line { animation-delay: 0.30s; }
  .hv-src:nth-child(4) line { animation-delay: 0.45s; }
  .hv-src:nth-child(5) line { animation-delay: 0.60s; }
  .hv-src:nth-child(6) line { animation-delay: 0.75s; }
  @keyframes hv-flow {
    to { stroke-dashoffset: -14; }
  }

  /* Pulse the inner ring */
  .hero-visual circle:nth-of-type(2) {
    animation: hv-ring 3s ease-in-out infinite;
    transform-origin: 220px 170px;
  }
  @keyframes hv-ring {
    0%, 100% { stroke-opacity: 0.6; }
    50%      { stroke-opacity: 1; }
  }

  /* Slow rotation on outer dashed rings */
  .hero-visual circle:nth-of-type(3),
  .hero-visual circle:nth-of-type(4) {
    animation: hv-spin 18s linear infinite;
    transform-origin: 220px 170px;
  }
  .hero-visual circle:nth-of-type(4) {
    animation-direction: reverse;
    animation-duration: 26s;
  }
  @keyframes hv-spin {
    to { transform: rotate(360deg); }
  }

  /* Engine → envelope arrow flow */
  .hero-visual > line {
    animation: hv-flow 1.2s linear infinite;
  }

  @media (prefers-reduced-motion: reduce) {
    .hero-visual *, .hero-visual circle { animation: none !important; }
  }
</style>
