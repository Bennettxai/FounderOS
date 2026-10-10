<!-- Sticky top strip (FounderOS v1 Topbar): `founderos-os / <view>` breadcrumb, then
     three 30px icon buttons (the theme menu, the ⌘K palette, the Conductor
     dock) and the OS emblem in the corner. -->
<script lang="ts">
	import { Bot, Search } from '$lib/founderos/icons';
	import '../kit/kit.css';
	import OsMark from '../kit/OsMark.svelte';
	import { breadcrumbFor, openPalette } from './chrome';
	import { openConductor } from './conductor';
	import ThemeToggle from './ThemeToggle.svelte';

	let { pathname }: { pathname: string } = $props();
	const here = $derived(breadcrumbFor(pathname));
</script>

<div class="bn-topbar sticky top-0 z-30 flex h-[52px] shrink-0 items-center gap-3.5 border-b px-6 backdrop-blur">
	<div data-part="crumb" class="bn-dim flex items-center gap-[7px] whitespace-nowrap font-mono text-[11px] tracking-[0.04em]">
		<span>founder-os</span>
		<span class="opacity-45">/</span>
		<span class="bn-text">{here}</span>
	</div>
	<div class="ml-auto flex items-center gap-2.5">
		<ThemeToggle />
		<button
			type="button"
			onclick={openPalette}
			title="Command palette (⌘K)"
			aria-label="Open the command palette (⌘K)"
			data-lens="c"
			class="bn-pressable is-dark bn-ctl bn-muted grid h-[30px] w-[30px] place-items-center rounded-[6px] border"
		>
			<Search size={14} />
		</button>
		<!-- the agent dock: the Conductor answers about whatever screen you're on -->
		<button
			type="button"
			onclick={openConductor}
			title="Ask the Conductor about this screen"
			aria-label="Open the Conductor agent panel"
			data-lens="c"
			class="bn-pressable is-dark bn-ctl bn-agent bn-muted grid h-[30px] w-[30px] place-items-center rounded-[6px] border"
		>
			<Bot size={14} />
		</button>
		<!-- FounderOS v1 emblem: the brand mark in the top-right corner -->
		<OsMark size={26} class="ml-1 shrink-0" />
	</div>
</div>

<style>
	.bn-topbar {
		border-color: var(--bn-border);
		background: color-mix(in oklab, var(--bn-bg-2) 70%, transparent);
	}
	/* rest colors; the lens (kit.css .bn-pressable) owns hover and press */
	.bn-ctl {
		border-color: var(--bn-border);
		background: var(--bn-surface);
	}
	.bn-agent:hover {
		color: var(--bn-accent);
	}
</style>
