<script lang="ts">
  // 9-stage pipeline visual.
  //
  //   INTAKE → PARSE → DECOMPOSE → CLASSIFY → EMBED  ┐
  //                                                  │
  //                                                  ▼
  //   CURATE ← CLUSTER ← STORE ← ROUTE  ─────────────┘
  //      │
  //      └ feedback loop back to INTAKE (dashed)
  //
  // Phases color-coded:
  //   ingest  (blue)   1–2
  //   process (purple) 3–5
  //   route   (cyan)   6–7
  //   curate  (green)  8–9

  type Stage = { n: number; name: string; phase: 'ingest' | 'process' | 'route' | 'curate' };

  const top: Stage[] = [
    { n: 1, name: 'Intake',    phase: 'ingest'  },
    { n: 2, name: 'Parse',     phase: 'ingest'  },
    { n: 3, name: 'Decompose', phase: 'process' },
    { n: 4, name: 'Classify',  phase: 'process' },
    { n: 5, name: 'Embed',     phase: 'process' }
  ];

  const bottom: Stage[] = [
    { n: 6, name: 'Route',   phase: 'route'  },
    { n: 7, name: 'Store',   phase: 'route'  },
    { n: 8, name: 'Cluster', phase: 'curate' },
    { n: 9, name: 'Curate',  phase: 'curate' }
  ];

  const phaseColor: Record<Stage['phase'], string> = {
    ingest:  '#7ea8ff',
    process: '#bb7eff',
    route:   '#5fcfd4',
    curate:  '#7be3a3'
  };

  let active = $state<number | null>(null);
</script>

<div class="pipeline">
  <svg viewBox="0 0 920 280" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="Nine-stage Optimal Engine pipeline">
    <defs>
      <marker id="pl-arrow" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto">
        <path d="M0,0 L8,4 L0,8 Z" fill="#3a4a72"/>
      </marker>
      <marker id="pl-arrow-soft" markerWidth="8" markerHeight="8" refX="7" refY="4" orient="auto">
        <path d="M0,0 L8,4 L0,8 Z" fill="#7ea8ff" opacity="0.4"/>
      </marker>
    </defs>

    <!-- Top row connection -->
    <path d="M 100,60 L 820,60" stroke="#3a4a72" stroke-width="1.5" fill="none" stroke-dasharray="2 4"/>

    <!-- Top row stages -->
    {#each top as s, i}
      {@const x = 80 + i * 170}
      <g
        class="pl-stage"
        class:active={active === s.n}
        onmouseenter={() => (active = s.n)}
        onmouseleave={() => (active = null)}
        role="button" tabindex="0"
      >
        <circle cx={x} cy="60" r="28" fill="#0d1015" stroke={phaseColor[s.phase]} stroke-width="2"/>
        <text x={x} y="65" text-anchor="middle" fill={phaseColor[s.phase]} font-size="13" font-weight="700">{s.n}</text>
        <text x={x} y="110" text-anchor="middle" fill="#f1f1f3" font-size="12" font-weight="600">{s.name}</text>
      </g>
    {/each}

    <!-- Bend down from stage 5 to stage 6 -->
    <path
      d="M 760,60 Q 870,60 870,140 Q 870,220 800,220"
      stroke="#3a4a72" stroke-width="1.5" fill="none" stroke-dasharray="2 4"
      marker-end="url(#pl-arrow)"
    />

    <!-- Bottom row connection -->
    <path d="M 800,220 L 100,220" stroke="#3a4a72" stroke-width="1.5" fill="none" stroke-dasharray="2 4"/>

    <!-- Bottom row stages (right-to-left visually) -->
    {#each bottom as s, i}
      {@const x = 760 - i * 170}
      <g
        class="pl-stage"
        class:active={active === s.n}
        onmouseenter={() => (active = s.n)}
        onmouseleave={() => (active = null)}
        role="button" tabindex="0"
      >
        <circle cx={x} cy="220" r="28" fill="#0d1015" stroke={phaseColor[s.phase]} stroke-width="2"/>
        <text x={x} y="225" text-anchor="middle" fill={phaseColor[s.phase]} font-size="13" font-weight="700">{s.n}</text>
        <text x={x} y="170" text-anchor="middle" fill="#f1f1f3" font-size="12" font-weight="600">{s.name}</text>
      </g>
    {/each}

    <!-- Feedback loop: Curate (9) back to Intake (1) -->
    <path
      d="M 80,220 Q 30,220 30,140 Q 30,60 60,60"
      stroke="#7ea8ff" stroke-width="1.5" fill="none" stroke-dasharray="4 4" opacity="0.4"
      marker-end="url(#pl-arrow-soft)"
    />
    <text x="42" y="142" fill="#7ea8ff" font-size="9" font-weight="600" letter-spacing="1.5" opacity="0.7" transform="rotate(-90 42 142)">FEEDBACK</text>

    <!-- Phase legend -->
    <g class="pl-legend" font-size="10" font-weight="600">
      <g transform="translate(80, 268)">
        <rect width="10" height="10" rx="2" fill="#7ea8ff"/>
        <text x="16" y="9" fill="#9aa0a8">Ingest</text>
      </g>
      <g transform="translate(180, 268)">
        <rect width="10" height="10" rx="2" fill="#bb7eff"/>
        <text x="16" y="9" fill="#9aa0a8">Process</text>
      </g>
      <g transform="translate(290, 268)">
        <rect width="10" height="10" rx="2" fill="#5fcfd4"/>
        <text x="16" y="9" fill="#9aa0a8">Route + Store</text>
      </g>
      <g transform="translate(420, 268)">
        <rect width="10" height="10" rx="2" fill="#7be3a3"/>
        <text x="16" y="9" fill="#9aa0a8">Curate</text>
      </g>
    </g>
  </svg>
</div>

<style>
  .pipeline {
    width: 100%;
  }
  .pipeline svg {
    width: 100%;
    height: auto;
    display: block;
  }
  .pl-stage {
    cursor: pointer;
    transition: filter 0.2s ease;
  }
  .pl-stage:hover circle,
  .pl-stage.active circle {
    filter: drop-shadow(0 0 12px currentColor);
  }
  .pl-stage:hover text,
  .pl-stage.active text {
    fill: var(--text);
  }

  /* Animate the dashed connector lines */
  .pipeline svg path[stroke-dasharray] {
    animation: pl-flow 6s linear infinite;
  }
  @keyframes pl-flow {
    to { stroke-dashoffset: -120; }
  }
  @media (prefers-reduced-motion: reduce) {
    .pipeline svg path[stroke-dasharray] { animation: none; }
  }
</style>
