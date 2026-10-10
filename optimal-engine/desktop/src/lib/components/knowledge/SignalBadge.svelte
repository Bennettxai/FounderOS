<script lang="ts">
	import type { OsaMode } from '$lib/stores/osa';

	type Genre = 'DIRECT' | 'INFORM' | 'COMMIT' | 'DECIDE' | 'EXPRESS';

	interface Props {
		mode: OsaMode;
		confidence?: number;
		genre?: Genre;
		docType?: string;
		weight?: number;
		compact?: boolean;
	}

	let { mode, confidence, genre, docType, weight, compact = false }: Props = $props();

	const modeConfig: Record<OsaMode, { bg: string; text: string; dot: string }> = {
		BUILD:    { bg: 'bg-indigo-50',  text: 'text-indigo-700',  dot: 'bg-indigo-500'  },
		ASSIST:   { bg: 'bg-green-50',   text: 'text-green-700',   dot: 'bg-green-500'   },
		ANALYZE:  { bg: 'bg-violet-50',  text: 'text-violet-700',  dot: 'bg-violet-500'  },
		EXECUTE:  { bg: 'bg-amber-50',   text: 'text-amber-700',   dot: 'bg-amber-500'   },
		MAINTAIN: { bg: 'bg-slate-100',  text: 'text-slate-600',   dot: 'bg-slate-400'   }
	};

	const genreConfig: Record<Genre, { bg: string; text: string }> = {
		DIRECT:  { bg: 'bg-orange-50',  text: 'text-orange-700'  },
		INFORM:  { bg: 'bg-blue-50',    text: 'text-blue-700'    },
		COMMIT:  { bg: 'bg-emerald-50', text: 'text-emerald-700' },
		DECIDE:  { bg: 'bg-rose-50',    text: 'text-rose-700'    },
		EXPRESS: { bg: 'bg-pink-50',    text: 'text-pink-700'    }
	};

	const genreLabels: Record<Genre, string> = {
		DIRECT:  'Action',
		INFORM:  'Question',
		COMMIT:  'Commit',
		DECIDE:  'Decision',
		EXPRESS: 'Feedback'
	};

	const mConfig = $derived(modeConfig[mode]);
	const gConfig = $derived(genre ? genreConfig[genre] : null);
	const modeLabel = $derived(
		confidence !== undefined
			? `${mode} ${Math.round(confidence * 100)}%`
			: mode
	);
</script>

<div class="inline-flex items-center gap-1 flex-wrap">
	<!-- Mode badge -->
	<span
		class="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-[11px] font-semibold tracking-wide {mConfig.bg} {mConfig.text}"
		aria-label="Signal mode: {modeLabel}"
	>
		<span class="h-1.5 w-1.5 rounded-full {mConfig.dot}" aria-hidden="true"></span>
		{compact ? mode : modeLabel}
	</span>

	<!-- Genre badge -->
	{#if genre && gConfig}
		<span
			class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-medium tracking-wide {gConfig.bg} {gConfig.text}"
			aria-label="Genre: {genre}"
		>
			{genreLabels[genre]}
		</span>
	{/if}

	<!-- DocType badge -->
	{#if docType}
		<span
			class="inline-flex items-center rounded-full px-1.5 py-0.5 text-[10px] font-medium tracking-wide bg-gray-100 text-gray-600"
			aria-label="Document type: {docType}"
		>
			{docType}
		</span>
	{/if}
</div>
