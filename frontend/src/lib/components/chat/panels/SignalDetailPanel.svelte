<script lang="ts">
	import type { OsaMode } from '$lib/stores/osa';

	type Genre = 'DIRECT' | 'INFORM' | 'COMMIT' | 'DECIDE' | 'EXPRESS';

	interface Props {
		mode: OsaMode;
		confidence?: number;
		genre?: Genre;
		docType?: string;
		weight?: number;
	}

	let { mode, confidence, genre, docType, weight }: Props = $props();

	let expanded = $state(false);

	const genreDescriptions: Record<Genre, string> = {
		DIRECT:  'User wants something created or done',
		INFORM:  'User is asking a question or seeking information',
		COMMIT:  'User is making a commitment or planning',
		DECIDE:  'User needs help choosing between options',
		EXPRESS: 'User is expressing a feeling or giving feedback'
	};

	const weightLabel = $derived.by(() => {
		if (!weight) return '';
		if (weight < 0.3) return 'Low complexity';
		if (weight < 0.6) return 'Medium complexity';
		if (weight < 0.8) return 'High complexity';
		return 'Very high complexity';
	});
</script>

{#if genre || docType}
	<div class="mt-1">
		<button
			type="button"
			class="btn-pill btn-pill-ghost btn-pill-xs"
			onclick={() => (expanded = !expanded)}
			aria-expanded={expanded}
			aria-label="{expanded ? 'Hide' : 'Show'} signal classification details"
		>
			{expanded ? 'Hide' : 'Show'} signal details
		</button>

		{#if expanded}
			<div class="mt-1.5 rounded-lg border border-gray-100 bg-gray-50/50 p-3 text-xs space-y-2">
				<div class="grid grid-cols-2 gap-x-4 gap-y-1.5">
					<div>
						<span class="text-gray-400 text-[10px] uppercase tracking-wider">Mode</span>
						<div class="font-medium text-gray-700">{mode}</div>
					</div>

					{#if confidence !== undefined}
						<div>
							<span class="text-gray-400 text-[10px] uppercase tracking-wider">Confidence</span>
							<div class="font-medium text-gray-700">{Math.round(confidence * 100)}%</div>
						</div>
					{/if}

					{#if genre}
						<div>
							<span class="text-gray-400 text-[10px] uppercase tracking-wider">Genre</span>
							<div class="font-medium text-gray-700">{genre}</div>
							<div class="text-gray-400 text-[10px]">{genreDescriptions[genre]}</div>
						</div>
					{/if}

					{#if docType}
						<div>
							<span class="text-gray-400 text-[10px] uppercase tracking-wider">Document Type</span>
							<div class="font-medium text-gray-700 capitalize">{docType}</div>
						</div>
					{/if}

					{#if weight}
						<div>
							<span class="text-gray-400 text-[10px] uppercase tracking-wider">Signal Weight</span>
							<div class="font-medium text-gray-700">{weight.toFixed(1)}</div>
							<div class="text-gray-400 text-[10px]">{weightLabel}</div>
						</div>
					{/if}
				</div>
			</div>
		{/if}
	</div>
{/if}
