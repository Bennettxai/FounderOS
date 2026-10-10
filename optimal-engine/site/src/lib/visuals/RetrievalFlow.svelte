<script lang="ts">
  // Retrieval flow visual.
  //
  //                           ┌──────────────┐
  //   USER QUERY ─→ AGENT ─→  │ TIER 3: WIKI │  ─yes(hot)→ ENVELOPE
  //                           └──────┬───────┘                ▲
  //                                  │ no                     │
  //                                  ▼                        │
  //                           ┌──────────────┐                 │
  //                           │ TIER 2: SEARCH│ → COMPOSER ────┘
  //                           │ vec + BM25   │
  //                           └──────────────┘
  //
  // Envelope is: ACL-scoped, audience-shaped, bandwidth-matched.
  //   wiki hit:    ~12 ms
  //   chunks path: ~80 ms
  //   target SLA:  < 200 ms

  let phase = $state<'idle' | 'wiki' | 'fallthrough' | 'envelope'>('idle');

  function play() {
    phase = 'wiki';
    setTimeout(() => (phase = 'fallthrough'), 800);
    setTimeout(() => (phase = 'envelope'), 1600);
    setTimeout(() => (phase = 'idle'), 3200);
  }
</script>

<div class="rflow">
  <svg viewBox="0 0 880 280" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Retrieval flow: query, wiki check, fall-through to hybrid search, envelope">
    <defs>
      <marker id="rf-arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto">
        <path d="M0,0 L8,4 L0,8 Z" fill="#9aa0a8"/>
      </marker>
      <marker id="rf-arrow-hot" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto">
        <path d="M0,0 L8,4 L0,8 Z" fill="#7be3a3"/>
      </marker>
      <marker id="rf-arrow-cold" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto">
        <path d="M0,0 L8,4 L0,8 Z" fill="#bb7eff"/>
      </marker>
      <linearGradient id="rf-env" x1="0" y1="0" x2="1" y2="0">
        <stop offset="0%"  stop-color="#1a233c"/>
        <stop offset="100%" stop-color="#0d1015"/>
      </linearGradient>
    </defs>

    <!-- Query box -->
    <g class="rf-node">
      <rect x="20" y="100" width="140" height="60" rx="10" fill="#0d1015" stroke="#1c2230"/>
      <text x="90" y="125" text-anchor="middle" fill="#9aa0a8" font-size="9" font-weight="600" letter-spacing="1.5">QUERY</text>
      <text x="90" y="145" text-anchor="middle" fill="#f1f1f3" font-size="12" font-style="italic">"customer portal pricing"</text>
    </g>

    <!-- Arrow query → wiki -->
    <line x1="160" y1="130" x2="280" y2="80" stroke="#9aa0a8" stroke-width="1.5" marker-end="url(#rf-arrow)"/>
    <text x="200" y="100" fill="#9aa0a8" font-size="10">ask</text>

    <!-- Wiki box -->
    <g class="rf-node" class:hot={phase === 'wiki' || phase === 'envelope'}>
      <rect x="280" y="40" width="220" height="80" rx="10" fill="#1a233c" stroke={phase === 'wiki' || phase === 'envelope' ? '#7be3a3' : '#3a4a72'} stroke-width="2"/>
      <text x="296" y="62" fill="#7be3a3" font-size="9" font-weight="600" letter-spacing="1.5">TIER 3 · WIKI</text>
      <text x="296" y="82" fill="#f1f1f3" font-size="13" font-weight="600">Curated page hit?</text>
      <text x="296" y="100" fill="#9aa0a8" font-size="10">audience-aware · hot citations</text>
      <text x="478" y="112" text-anchor="end" fill="#7be3a3" font-size="10" font-weight="600">~12ms</text>
    </g>

    <!-- Hot path: wiki hit → envelope -->
    <path
      d="M 500,80 L 690,80 L 690,120"
      stroke={phase === 'wiki' || phase === 'envelope' ? '#7be3a3' : '#2a3340'}
      stroke-width="2"
      fill="none"
      stroke-dasharray="6 4"
      marker-end="url(#rf-arrow-hot)"
    />
    <text x="600" y="72" fill="#7be3a3" font-size="10" font-weight="600">yes</text>

    <!-- Fallthrough: wiki miss → search -->
    <path
      d="M 390,120 L 390,180"
      stroke={phase === 'fallthrough' ? '#bb7eff' : '#2a3340'}
      stroke-width="2"
      fill="none"
      marker-end="url(#rf-arrow-cold)"
    />
    <text x="400" y="155" fill="#bb7eff" font-size="10" font-weight="600">no</text>

    <!-- Search box -->
    <g class="rf-node" class:hot={phase === 'fallthrough'}>
      <rect x="280" y="180" width="220" height="80" rx="10" fill="#131a26" stroke={phase === 'fallthrough' ? '#bb7eff' : '#243044'} stroke-width="2"/>
      <text x="296" y="202" fill="#bb7eff" font-size="9" font-weight="600" letter-spacing="1.5">TIER 2 · HYBRID SEARCH</text>
      <text x="296" y="222" fill="#f1f1f3" font-size="13" font-weight="600">Vector + BM25 + graph</text>
      <text x="296" y="240" fill="#9aa0a8" font-size="10">classify intent · expand · rank</text>
      <text x="478" y="252" text-anchor="end" fill="#bb7eff" font-size="10" font-weight="600">~80ms</text>
    </g>

    <!-- Search → composer → envelope -->
    <path
      d="M 500,220 L 580,220 L 580,150 L 690,150"
      stroke={phase === 'fallthrough' ? '#bb7eff' : '#2a3340'}
      stroke-width="2"
      fill="none"
      stroke-dasharray="6 4"
      marker-end="url(#rf-arrow-cold)"
    />
    <text x="540" y="212" fill="#bb7eff" font-size="10" font-weight="600">compose</text>

    <!-- Envelope -->
    <g class="rf-node" class:hot={phase === 'envelope'}>
      <rect x="690" y="120" width="170" height="120" rx="10" fill="url(#rf-env)" stroke={phase === 'envelope' ? '#7ea8ff' : '#3a4a72'} stroke-width="2"/>
      <text x="706" y="142" fill="#7ea8ff" font-size="9" font-weight="600" letter-spacing="1.5">ENVELOPE</text>
      <text x="706" y="162" fill="#f1f1f3" font-size="13" font-weight="600">Right grain.</text>
      <text x="706" y="180" fill="#f1f1f3" font-size="13" font-weight="600">Right scope.</text>
      <line x1="706" y1="190" x2="844" y2="190" stroke="#243044"/>
      <text x="706" y="206" fill="#9aa0a8" font-size="10">· ACL-scoped</text>
      <text x="706" y="220" fill="#9aa0a8" font-size="10">· audience-shaped</text>
      <text x="706" y="234" fill="#9aa0a8" font-size="10">· bandwidth-matched</text>
    </g>

    <!-- SLA marker -->
    <g transform="translate(20, 30)">
      <rect width="120" height="24" rx="999" fill="#0d1015" stroke="#1c2230"/>
      <circle cx="14" cy="12" r="4" fill="#7be3a3"/>
      <text x="26" y="16" fill="#9aa0a8" font-size="10" font-weight="600">SLA &lt; 200 ms</text>
    </g>
  </svg>

  <button class="rflow__play" type="button" onclick={play} aria-label="Replay retrieval animation">
    {phase === 'idle' ? '▶ Animate retrieval' : '… running'}
  </button>
</div>

<style>
  .rflow {
    width: 100%;
  }
  .rflow svg {
    width: 100%;
    height: auto;
    display: block;
  }
  .rf-node {
    transition: filter 0.3s ease;
  }
  .rf-node.hot {
    filter: drop-shadow(0 0 18px rgba(126, 168, 255, 0.45));
  }
  .rf-node.hot rect {
    transition: stroke 0.3s ease;
  }

  .rflow__play {
    margin-top: 0.75rem;
    background: var(--bg-elevated);
    color: var(--text-muted);
    border: 1px solid var(--border);
    border-radius: 999px;
    padding: 0.4rem 0.9rem;
    font-size: 0.78rem;
    cursor: pointer;
    font: inherit;
    font-size: 0.78rem;
  }
  .rflow__play:hover {
    color: var(--text);
    border-color: var(--accent);
  }
</style>
