<script lang="ts">
	// Wave 2 ships Mode B (plain textarea) only — Mode A (DocumentEditor-backed
	// rich text) is deferred to Wave 3 per the implementation decision logged in
	// the wave kickoff. The component still keeps a `mode` prop on the surface so
	// the orchestrator and the rest of the spec align with future work.

	type Mode = 'plain' | 'rich';

	interface Props {
		value: string;
		mode?: Mode;
		placeholder?: string;
		onChange: (value: string) => void;
	}

	let {
		value,
		mode = 'plain',
		placeholder = 'Write your message…',
		onChange,
	}: Props = $props();

	function handleInput(event: Event) {
		const target = event.target as HTMLTextAreaElement;
		onChange(target.value);
	}
</script>

<div class="cm-email-body" data-mode={mode}>
	<textarea
		class="bos-textarea cm-email-body__textarea"
		{value}
		{placeholder}
		oninput={handleInput}
		aria-label="Email body"
	></textarea>
</div>

<style>
	.cm-email-body {
		display: flex;
		flex-direction: column;
		min-height: 240px;
		max-height: 60vh;
		background: var(--dbg);
	}

	.cm-email-body__textarea {
		flex: 1;
		width: 100%;
		min-height: 240px;
		padding: var(--space-4);
		background: var(--dbg);
		border: none;
		outline: none;
		resize: vertical;
		font-family: inherit;
		font-size: var(--text-sm);
		color: var(--dt);
		line-height: 1.65;
	}

	.cm-email-body__textarea::placeholder {
		color: var(--dt4);
	}
</style>
