<!-- A detail card's header: title (optionally tinted), sub line, back and close. -->
<script lang="ts">
	import { ArrowLeft, X } from '$lib/founderos/icons';
	let { title, sub, color, onBack, onClose }: { title: string; sub?: string; color?: string; onBack?: () => void; onClose?: () => void } = $props();
</script>

<div class="kg-b flex items-start gap-2 border-b px-3 py-2">
	{#if onBack}
		<button type="button" onclick={onBack} aria-label="Back" class="bn-pressable bn-dim kg-h-text mt-0.5 shrink-0"><ArrowLeft size={14} /></button>
	{/if}
	<div class="min-w-0 flex-1">
		<div class="bn-text truncate text-[12.5px] font-bold" style={color ? `color: ${color}` : undefined}>{title}</div>
		{#if sub}<div class="bn-dim truncate font-mono text-[9.5px]">{sub}</div>{/if}
	</div>
	{#if onClose}
		<button type="button" onclick={onClose} aria-label="Close" class="bn-pressable bn-dim kg-h-text shrink-0"><X size={14} /></button>
	{/if}
</div>
