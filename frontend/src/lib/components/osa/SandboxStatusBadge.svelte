<script lang="ts">
	import { Loader2 } from 'lucide-svelte';
	import type { SandboxStatus } from '$lib/types/sandbox';
	import { getSandboxStatusColor, getSandboxStatusBgColor } from '$lib/types/sandbox';

	interface Props {
		status: SandboxStatus;
		showLabel?: boolean;
		size?: 'sm' | 'md' | 'lg';
	}

	let { status, showLabel = true, size = 'md' }: Props = $props();

	const STATUS_LABELS: Record<SandboxStatus, string> = {
		none: 'None',
		pending: 'Pending',
		deploying: 'Deploying',
		running: 'Running',
		stopped: 'Stopped',
		failed: 'Failed',
		removing: 'Removing'
	};

	let textColor = $derived(getSandboxStatusColor(status));
	let bgColor = $derived(getSandboxStatusBgColor(status));
	let isAnimated = $derived(status === 'deploying' || status === 'running');

	let sizeClasses = $derived({
		sm: { badge: 'px-1.5 py-0.5 text-xs', dot: 'w-1.5 h-1.5', icon: 'w-3 h-3' },
		md: { badge: 'px-2 py-1 text-xs', dot: 'w-2 h-2', icon: 'w-3.5 h-3.5' },
		lg: { badge: 'px-2.5 py-1.5 text-sm', dot: 'w-2.5 h-2.5', icon: 'w-4 h-4' }
	}[size]);
</script>

<span class="inline-flex items-center gap-1.5 rounded-full font-medium {bgColor} {sizeClasses.badge}">
	{#if status === 'deploying' || status === 'removing'}
		<Loader2 class="{sizeClasses.icon} {textColor} animate-spin" />
	{:else}
		<span
			class="rounded-full {sizeClasses.dot} {textColor}"
			class:animate-pulse={status === 'running'}
			style="background-color: currentColor;"
		></span>
	{/if}

	{#if showLabel}
		<span class={textColor}>{STATUS_LABELS[status]}</span>
	{/if}
</span>
