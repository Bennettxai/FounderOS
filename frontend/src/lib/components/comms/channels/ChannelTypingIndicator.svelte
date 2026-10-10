<script lang="ts">
	// Slack/Teams-style "Sarah is typing…" indicator. UI shell only — Wave 3
	// realtime (Fantem) wires the visibility flag to live presence events.
	//
	// Names are passed by the orchestrator; this component decides the copy
	// (so "Sarah, Bob, and 2 others are typing…" stays in one place).

	interface Props {
		// Display names of currently-typing users. Empty list ⇒ component renders nothing.
		names?: string[];
	}

	let { names = [] }: Props = $props();

	const visible = $derived(names.length > 0);

	const label = $derived.by(() => {
		if (names.length === 0) return '';
		if (names.length === 1) return `${names[0]} is typing`;
		if (names.length === 2) return `${names[0]} and ${names[1]} are typing`;
		if (names.length === 3) {
			return `${names[0]}, ${names[1]} and ${names[2]} are typing`;
		}
		return `${names[0]}, ${names[1]} and ${names.length - 2} others are typing`;
	});
</script>

{#if visible}
	<div
		class="cm-channels-typing"
		role="status"
		aria-live="polite"
		aria-label={label}
	>
		<span class="cm-channels-typing__dots" aria-hidden="true">
			<span class="cm-channels-typing__dot"></span>
			<span class="cm-channels-typing__dot"></span>
			<span class="cm-channels-typing__dot"></span>
		</span>
		<span class="cm-channels-typing__label">{label}</span>
	</div>
{/if}

<style>
	.cm-channels-typing {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-1) var(--space-5);
		font-size: var(--text-xs);
		color: var(--dt3);
		background: var(--dbg);
		border-top: 1px solid var(--dbd);
	}

	.cm-channels-typing__dots {
		display: inline-flex;
		align-items: center;
		gap: 3px;
	}

	.cm-channels-typing__dot {
		width: 4px;
		height: 4px;
		border-radius: var(--radius-full);
		background: var(--dt3);
		animation: cm-channels-typing-fade 1.2s ease-in-out infinite;
	}

	.cm-channels-typing__dot:nth-child(2) {
		animation-delay: 0.18s;
	}

	.cm-channels-typing__dot:nth-child(3) {
		animation-delay: 0.36s;
	}

	.cm-channels-typing__label {
		font-style: italic;
	}

	@keyframes cm-channels-typing-fade {
		0%, 100% {
			opacity: 0.25;
			transform: translateY(0);
		}
		40% {
			opacity: 1;
			transform: translateY(-1px);
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.cm-channels-typing__dot {
			animation: none;
			opacity: 0.6;
		}
	}
</style>
