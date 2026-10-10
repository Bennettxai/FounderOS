<script lang="ts">
  import { Bot, Briefcase, Code2, Headphones, Sparkles } from 'lucide-svelte';

  interface Quickstart {
    name: string;
    desc: string;
    href: string;
    icon: 'briefcase' | 'code' | 'headset' | 'sparkle';
  }

  interface Props {
    quickstarts: readonly Quickstart[];
    onNavigate: (href: string) => void;
    onNewAgent: () => void;
  }

  let { quickstarts, onNavigate, onNewAgent }: Props = $props();
</script>

<div class="aes-empty-full">
  <div class="aes-empty-full__icon" aria-hidden="true">
    <Bot size={28} strokeWidth={1.5} />
  </div>
  <h2 class="aes-empty-full__title">No agents yet</h2>
  <p class="aes-empty-full__subtitle">Create your first custom AI agent or start from a template</p>

  <div class="aes-quickstart">
    {#each quickstarts as qs}
      <button class="aes-qs" onclick={() => onNavigate(qs.href)}>
        <div class="aes-qs__icon" aria-hidden="true">
          {#if qs.icon === 'briefcase'}
            <Briefcase size={16} strokeWidth={2} />
          {:else if qs.icon === 'code'}
            <Code2 size={16} strokeWidth={2} />
          {:else if qs.icon === 'headset'}
            <Headphones size={16} strokeWidth={2} />
          {:else if qs.icon === 'sparkle'}
            <Sparkles size={16} strokeWidth={2} />
          {/if}
        </div>
        <p class="aes-qs__name">{qs.name}</p>
        <p class="aes-qs__desc">{qs.desc}</p>
      </button>
    {/each}
  </div>

  <p class="aes-empty-full__or">or</p>
  <button onclick={onNewAgent} class="btn-cta">Start from Scratch</button>
</div>

<style>
  .aes-empty-full {
    text-align: center;
    padding: 3.5rem 2rem;
    background: var(--dbg, #fff);
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 0.75rem;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.75rem;
  }
  .aes-empty-full__icon {
    width: 3.5rem;
    height: 3.5rem;
    border-radius: 14px;
    background: var(--dbg2, #f5f5f5);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--dt3, #888);
    margin-bottom: 0.25rem;
  }
  .aes-empty-full__title {
    font-size: 1.125rem;
    font-weight: 700;
    color: var(--dt, #111);
    margin: 0;
  }
  .aes-empty-full__subtitle {
    font-size: 0.875rem;
    color: var(--dt3, #888);
    margin: 0 0 0.5rem;
    max-width: 340px;
  }
  .aes-empty-full__or {
    font-size: 0.8125rem;
    color: var(--dt4, #bbb);
    margin: 0;
  }

  .aes-quickstart {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.75rem;
    width: 100%;
    max-width: 480px;
  }
  .aes-qs {
    background: var(--dbg, #fff);
    border: 1px solid var(--dbd, #e0e0e0);
    border-radius: 12px;
    padding: 1.25rem;
    cursor: pointer;
    text-align: left;
    transition: border-color 0.15s, box-shadow 0.15s, transform 0.15s;
    display: flex;
    flex-direction: column;
    gap: 0.375rem;
  }
  .aes-qs:hover {
    border-color: var(--bos-accent-blue, #3b82f6);
    box-shadow: 0 4px 16px rgba(59, 130, 246, 0.08);
    transform: translateY(-1px);
  }
  .aes-qs__icon {
    width: 2rem;
    height: 2rem;
    border-radius: 8px;
    background: var(--dbg2, #f5f5f5);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--dt2, #555);
    margin-bottom: 0.25rem;
  }
  .aes-qs__name {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--dt, #111);
    margin: 0;
  }
  .aes-qs__desc {
    font-size: 0.75rem;
    color: var(--dt3, #888);
    margin: 0;
  }
</style>
