<!-- Theme picker (FounderOS v1 components/ThemeToggle.tsx): a palette chip in the
     Topbar that opens the theme menu, every skin with its swatch trio, name and
     one-line feel. Picking one flips data-founderos-theme on <html> and persists
     it; app.html's init script applies it before the next first paint. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { Check, Palette } from '$lib/founderos/icons';
	import '../kit/kit.css';
	import { applyFounderosTheme, FOUNDEROS_THEMES, DEFAULT_FOUNDEROS_THEME, resolveTheme, THEME_META, type FounderosTheme } from '../theme/theme';

	let theme = $state<FounderosTheme>(DEFAULT_FOUNDEROS_THEME);
	let open = $state(false);
	let root: HTMLDivElement | undefined = $state();

	onMount(() => {
		theme = resolveTheme(document.documentElement.getAttribute('data-founderos-theme'));
	});

	// close on outside click or Escape
	$effect(() => {
		if (!open) return;
		const onDown = (e: PointerEvent) => {
			if (!root?.contains(e.target as Node)) open = false;
		};
		const onKey = (e: KeyboardEvent) => {
			if (e.key === 'Escape') open = false;
		};
		window.addEventListener('pointerdown', onDown);
		window.addEventListener('keydown', onKey);
		return () => {
			window.removeEventListener('pointerdown', onDown);
			window.removeEventListener('keydown', onKey);
		};
	});

	function apply(next: FounderosTheme) {
		applyFounderosTheme(next);
		theme = next;
		open = false;
	}
</script>

<div bind:this={root} class="relative">
	<button
		type="button"
		onclick={() => (open = !open)}
		title="Choose a theme"
		aria-label="Choose a theme"
		aria-haspopup="menu"
		aria-expanded={open}
		class="bn-pressable bn-theme-chip grid h-[30px] w-[30px] place-items-center rounded-[5px] border"
	>
		<Palette size={14} />
	</button>

	{#if open}
		<div role="menu" aria-label="Theme" class="bn-theme-menu absolute right-0 top-9 z-50 w-56 rounded-[5px] border p-1">
			<div data-part="menu-title" class="bn-dim px-2 pb-1 pt-1.5 font-mono text-[9px] uppercase tracking-[0.16em]">Theme</div>
			{#each FOUNDEROS_THEMES as t (t)}
				{@const meta = THEME_META[t]}
				<button
					type="button"
					role="menuitemradio"
					aria-checked={t === theme}
					onclick={() => apply(t)}
					class="bn-pressable bn-theme-item flex w-full items-center gap-2.5 rounded-[5px] px-2 py-1.5 text-left"
				>
					<span data-part="swatch" class="flex shrink-0 -space-x-1">
						{#each meta.swatch as c, i (i)}
							<span class="bn-swatch h-3.5 w-3.5 rounded-full border" style="background: {c}"></span>
						{/each}
					</span>
					<span class="min-w-0 flex-1">
						<span data-part="name" class="bn-text block text-[11px] font-semibold leading-tight">{meta.name}</span>
						<span class="bn-dim block truncate font-mono text-[9px]">{meta.blurb}</span>
					</span>
					{#if t === theme}<Check size={14} class="bn-accent shrink-0" />{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>

<style>
	/* ThemeToggle.tsx is a plain pressable: it brightens its own border */
	.bn-theme-chip {
		border-color: var(--bn-border);
		background: var(--bn-surface);
		color: var(--bn-text-2);
	}
	.bn-theme-chip:hover {
		border-color: var(--bn-border-strong);
		color: var(--bn-text);
	}
	.bn-theme-menu {
		border-color: var(--bn-border-strong);
		background: var(--bn-surface);
		box-shadow: 0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1);
	}
	.bn-theme-item:hover,
	.bn-theme-item[aria-checked='true'] {
		background: var(--bn-surface-2);
	}
	.bn-swatch {
		border-color: var(--bn-border-strong);
	}
</style>
