<!-- FounderOS v1 components/SlidingTabs.tsx, pill variant: fixed-width tabs in an
     8px frame with a 5px marker that slides to the active tab, so it never
     jumps. Mono, bold, upper-case labels; the count rides a small tag that
     goes solid on the active tab. -->
<script lang="ts" generics="T extends string">
	let {
		tabs,
		value,
		onchange,
		tabWidth = 108
	}: { tabs: Array<{ id: T; label: string; count?: number | string | null }>; value: T; onchange: (id: T) => void; tabWidth?: number } = $props();

	const i = $derived(Math.max(0, tabs.findIndex((t) => t.id === value)));
</script>

<div
	data-part="sliding-tabs"
	role="tablist"
	class="bn-border relative grid rounded-[8px] border p-[3px]"
	style="grid-template-columns: repeat({tabs.length}, {tabWidth}px); background: var(--bn-bg)"
>
	<span
		aria-hidden="true"
		data-part="slider"
		class="pointer-events-none absolute bottom-[3px] left-[3px] top-[3px] rounded-[5px]"
		style="width: {tabWidth}px; transform: translateX({i * tabWidth}px); background: var(--bn-surface-2); transition: transform var(--bn-dur-lens, 280ms) var(--bn-ease-lens, ease)"
	></span>
	{#each tabs as t (t.id)}
		{@const active = t.id === value}
		<button
			type="button"
			role="tab"
			aria-selected={active}
			data-lens="c"
			onclick={() => onchange(t.id)}
			class="bn-pressable relative inline-flex h-[30px] items-center justify-center gap-1.5 bg-transparent font-mono text-[10.5px] font-bold uppercase tracking-[.18em] hover:scale-[1.04] hover:shadow-none {active
				? 'bn-text'
				: 'bn-dim hover:text-[color:var(--bn-text-2)]'}"
		>
			{t.label}
			{#if t.count != null}
				<span
					data-part="tab-count"
					class="px-1.5 text-[9.5px] leading-[15px] tracking-normal"
					style={active ? 'background: var(--bn-text); color: var(--bn-accent-ink)' : 'background: var(--bn-surface); color: var(--bn-text-3)'}>{t.count}</span
				>
			{/if}
		</button>
	{/each}
</div>
