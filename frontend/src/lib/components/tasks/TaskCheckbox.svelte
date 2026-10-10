<script lang="ts">
	import { fly } from 'svelte/transition';

	type TaskStatus = 'todo' | 'in_progress' | 'in_review' | 'done' | 'blocked';

	interface Props {
		status: TaskStatus;
		size?: 'sm' | 'md' | 'lg';
		disabled?: boolean;
		onStatusChange?: (status: TaskStatus) => void;
	}

	let { status, size = 'md', disabled = false, onStatusChange }: Props = $props();

	const sizeClasses = {
		sm: 'w-4 h-4',
		md: 'w-5 h-5',
		lg: 'w-6 h-6'
	};

	function cycleStatus() {
		if (disabled) return;

		const statusOrder: TaskStatus[] = ['todo', 'in_progress', 'in_review', 'done'];
		const currentIndex = statusOrder.indexOf(status);
		const nextStatus = statusOrder[(currentIndex + 1) % statusOrder.length];
		onStatusChange?.(nextStatus);
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ' ') {
			e.preventDefault();
			cycleStatus();
		}
	}
</script>

<button
	onclick={cycleStatus}
	onkeydown={handleKeydown}
	{disabled}
	class="tc-box {sizeClasses[size]} tc-box--{status}
		{disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}"
	title={status.replace('_', ' ')}
>
	{#if status === 'todo'}
		<!-- Empty -->
	{:else if status === 'in_progress'}
		<svg class="w-3 h-3" viewBox="0 0 16 16" fill="currentColor">
			<path d="M8 0a8 8 0 1 0 0 16A8 8 0 0 0 8 0zM1 8a7 7 0 0 1 7-7v14a7 7 0 0 1-7-7z"/>
		</svg>
	{:else if status === 'in_review'}
		<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
		</svg>
	{:else if status === 'done'}
		<svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24" in:fly={{ y: -5, duration: 200 }}>
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
		</svg>
	{:else if status === 'blocked'}
		<svg class="w-3 h-3" fill="currentColor" viewBox="0 0 24 24">
			<rect x="6" y="10" width="12" height="4" rx="1" />
		</svg>
	{/if}
</button>

<style>
	.tc-box {
		flex-shrink: 0;
		border-radius: 0.25rem;
		border: 1.5px solid var(--dbd, #d1d5db);
		display: flex;
		align-items: center;
		justify-content: center;
		transition: all 0.15s;
		background: transparent;
		color: var(--dt3, #888);
	}
	.tc-box:focus-visible {
		outline: none;
		box-shadow: 0 0 0 2px var(--dt, #111);
	}

	/* todo: empty border */
	.tc-box--todo {
		border-color: var(--dbd, #d1d5db);
	}
	.tc-box--todo:hover {
		border-color: var(--dt3, #888);
	}

	/* in_progress: half filled */
	.tc-box--in_progress {
		border-color: var(--dt3, #888);
		background: var(--dbg2, #f5f5f5);
		color: var(--dt2, #555);
	}

	/* in_review */
	.tc-box--in_review {
		border-color: var(--dt3, #888);
		background: var(--dbg2, #f5f5f5);
		color: var(--dt2, #555);
	}

	/* done: filled dark */
	.tc-box--done {
		border-color: var(--dt, #111);
		background: var(--dt, #111);
		color: #fff;
	}

	/* blocked */
	.tc-box--blocked {
		border-color: var(--dt3, #888);
		background: var(--dbg2, #f5f5f5);
		color: var(--dt3, #888);
	}
</style>
