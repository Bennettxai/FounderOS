<script lang="ts">
  import { onMount } from 'svelte';
  import {
    listArchitectures,
    getArchitecture,
    type ArchitectureSummary,
    type ArchitectureDetail,
    type ProcessorSummary
  } from '$lib/api/workspace';

  let archs = $state<ArchitectureSummary[]>([]);
  let processors = $state<ProcessorSummary[]>([]);
  let selected = $state<ArchitectureDetail | null>(null);
  let error = $state<string | null>(null);

  onMount(async () => {
    try {
      const res = await listArchitectures();
      archs = res.architectures;
      processors = res.processors;
      if (archs.length > 0) await select(archs[0].id);
    } catch (e) {
      error = (e as Error).message;
    }
  });

  async function select(id: string) {
    try {
      selected = await getArchitecture(id);
    } catch (e) {
      error = (e as Error).message;
    }
  }

  function modalityColor(m: string): string {
    const map: Record<string, string> = {
      text: '#8af',
      code: '#a8f',
      image: '#f8c',
      audio: '#fc6',
      video: '#fa6',
      time_series: '#6fc',
      table: '#6f8',
      structured: '#cfc',
      graph: '#fcf',
      tensor: '#8cf',
      geo: '#6fa',
      binary: '#888'
    };
    return map[m] ?? '#ccc';
  }
</script>

<div class="archs">
  <aside class="archs__list">
    <header>Data architectures <small>{archs.length}</small></header>
    {#if error}
      <pre class="error">{error}</pre>
    {/if}
    <ul>
      {#each archs as a}
        <li>
          <button
            class:active={selected?.id === a.id}
            onclick={() => select(a.id)}
          >
            <div class="name">{a.name}<small>v{a.version}</small></div>
            <div class="desc">{a.description}</div>
            <div class="meta">
              <span class="pill" style="color: {modalityColor(a.modality_primary)}">
                {a.modality_primary}
              </span>
              <span class="gran">{a.granularity.join(' → ')}</span>
              <span class="fcount">{a.field_count} fields</span>
            </div>
          </button>
        </li>
      {/each}
    </ul>

    <section class="processors">
      <header>Registered processors <small>{processors.length}</small></header>
      <ul class="proc-list">
        {#each processors as p}
          <li>
            <span class="proc-id">{p.id}</span>
            <span class="pill" style="color: {modalityColor(p.modality)}">{p.modality}</span>
            <span class="emits">emits {p.emits.join(', ')}</span>
          </li>
        {/each}
      </ul>
    </section>
  </aside>

  <section class="archs__detail">
    {#if !selected}
      <p class="muted">Pick an architecture on the left.</p>
    {:else}
      <header class="archs__detail-header">
        <h2>{selected.name}</h2>
        <span class="pill pill-lg" style="color: {modalityColor(selected.modality_primary)}">
          {selected.modality_primary}
        </span>
      </header>

      <p class="muted">{selected.description}</p>

      <div class="block">
        <h3>Granularity</h3>
        <div class="granularity">
          {#each selected.granularity as g, i}
            <span class="gran-step">{g}</span>
            {#if i < selected.granularity.length - 1}
              <span class="gran-arrow">→</span>
            {/if}
          {/each}
        </div>
      </div>

      <div class="block">
        <h3>Fields ({selected.fields.length})</h3>
        <table>
          <thead>
            <tr>
              <th>name</th>
              <th>modality</th>
              <th>shape</th>
              <th>req</th>
              <th>processor</th>
              <th>description</th>
            </tr>
          </thead>
          <tbody>
            {#each selected.fields as f}
              <tr>
                <td class="name">{f.name}</td>
                <td>
                  <span class="pill" style="color: {modalityColor(f.modality)}">{f.modality}</span>
                </td>
                <td class="shape">
                  {#if f.dims.length > 0}
                    [{f.dims.map((d) => (d === 'any' ? '*' : String(d))).join(' × ')}]
                  {:else}
                    —
                  {/if}
                </td>
                <td>{f.required ? '✓' : ''}</td>
                <td class="proc">
                  {#if f.processor}
                    {f.processor}
                  {:else}
                    <span class="muted">—</span>
                  {/if}
                </td>
                <td class="desc">{f.description ?? ''}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <div class="block">
        <h3>Philosophy</h3>
        <p class="muted">
          Fields bind to processors by atom, not by module. Swap a text embedder for a
          vision model by changing <code>processor:</code> in the architecture — no code
          path in the engine changes. Any field is a data point; any processor is a model
          or algorithm. LLM, vision model, classical ML, rule engine, agent — none are
          privileged.
        </p>
      </div>
    {/if}
  </section>
</div>

<style>
  .archs {
    display: grid;
    grid-template-columns: 360px 1fr;
    height: calc(100vh - 64px);
    background: #0b0d10;
    color: #ddd;
  }
  aside,
  section {
    overflow: auto;
  }
  .archs__list {
    border-right: 1px solid #1a1e25;
    display: flex;
    flex-direction: column;
  }
  header {
    padding: 0.75rem 1rem;
    color: #888;
    font-size: 0.85rem;
    border-bottom: 1px solid #1a1e25;
    background: #0b0d10;
    position: sticky;
    top: 0;
    z-index: 1;
  }
  header small {
    color: #555;
    margin-left: 0.5rem;
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .archs__list > ul li button {
    display: block;
    width: 100%;
    text-align: left;
    padding: 0.75rem 1rem;
    background: none;
    border: none;
    border-bottom: 1px solid #111;
    color: inherit;
    cursor: pointer;
  }
  .archs__list > ul li button:hover,
  .archs__list > ul li button.active {
    background: #161a21;
  }
  .name {
    font-weight: 600;
    margin-bottom: 0.25rem;
  }
  .name small {
    color: #666;
    margin-left: 0.4rem;
    font-weight: 400;
  }
  .desc {
    color: #888;
    font-size: 0.8rem;
    margin-bottom: 0.4rem;
  }
  .meta {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.75rem;
    color: #666;
  }
  .pill {
    padding: 2px 8px;
    background: #111;
    border: 1px solid #222;
    border-radius: 999px;
    font-size: 0.7rem;
    font-weight: 600;
  }
  .pill-lg {
    padding: 4px 12px;
    font-size: 0.85rem;
  }
  .gran {
    color: #888;
  }
  .fcount {
    color: #666;
  }
  .processors {
    padding-bottom: 1rem;
  }
  .proc-list li {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.4rem 1rem;
    font-size: 0.8rem;
    border-bottom: 1px solid #111;
  }
  .proc-id {
    font-weight: 600;
    color: #ccc;
    flex: 1;
  }
  .emits {
    color: #666;
    font-size: 0.75rem;
  }
  .archs__detail {
    padding: 1.5rem 2rem;
  }
  .archs__detail-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    margin-bottom: 1rem;
  }
  h2 {
    margin: 0;
    font-size: 1.1rem;
  }
  .block {
    margin-top: 1.5rem;
    padding-top: 1rem;
    border-top: 1px solid #1a1e25;
  }
  .block h3 {
    margin: 0 0 0.75rem 0;
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: #888;
    font-weight: 600;
  }
  .granularity {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }
  .gran-step {
    padding: 4px 10px;
    background: #111;
    border: 1px solid #222;
    border-radius: 4px;
    font-size: 0.85rem;
  }
  .gran-arrow {
    color: #555;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    font-size: 0.85rem;
  }
  th {
    text-align: left;
    padding: 0.4rem 0.6rem;
    color: #666;
    font-size: 0.7rem;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    font-weight: 500;
    border-bottom: 1px solid #222;
  }
  td {
    padding: 0.5rem 0.6rem;
    border-bottom: 1px solid #141820;
    vertical-align: top;
  }
  td.name {
    font-weight: 600;
    color: #cdf;
  }
  td.shape {
    font-family: 'SF Mono', Menlo, monospace;
    color: #888;
    font-size: 0.8rem;
  }
  td.proc {
    font-family: 'SF Mono', Menlo, monospace;
    color: #8af;
    font-size: 0.8rem;
  }
  td.desc {
    color: #888;
    font-size: 0.8rem;
  }
  .muted {
    color: #888;
  }
  code {
    font-family: 'SF Mono', Menlo, monospace;
    background: #0e1116;
    padding: 0 4px;
    border-radius: 3px;
  }
  .error {
    color: #f88;
    padding: 0.5rem;
  }
</style>
