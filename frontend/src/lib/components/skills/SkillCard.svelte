<!--
	SkillCard.svelte
	Visual card for an individual SORX skill in the catalog grid.
	Shows name, description, category icon, tier badge, and enabled toggle.
-->
<script lang="ts">
	import type { Skill } from '$lib/types/skills';
	import {
		tierLabels,
		tierColors,
		categoryLabels,
		categoryColors,
		categoryIcons
	} from '$lib/types/skills';

	interface Props {
		skill: Skill;
		onToggle?: (skill: Skill) => void;
		onClick?: (skill: Skill) => void;
	}

	let { skill, onToggle, onClick }: Props = $props();

	/** Map category to icon background tint */
	const categoryIconBg: Record<string, string> = {
		email: 'bg-blue-100 dark:bg-blue-900/30',
		messaging: 'bg-violet-100 dark:bg-violet-900/30',
		crm: 'bg-orange-100 dark:bg-orange-900/30',
		calendar: 'bg-teal-100 dark:bg-teal-900/30',
		sync: 'bg-cyan-100 dark:bg-cyan-900/30',
		export: 'bg-gray-100 dark:bg-gray-800',
		automation: 'bg-pink-100 dark:bg-pink-900/30'
	};

	const categoryIconColor: Record<string, string> = {
		email: 'text-blue-600 dark:text-blue-400',
		messaging: 'text-violet-600 dark:text-violet-400',
		crm: 'text-orange-600 dark:text-orange-400',
		calendar: 'text-teal-600 dark:text-teal-400',
		sync: 'text-cyan-600 dark:text-cyan-400',
		export: 'text-gray-600 dark:text-gray-400',
		automation: 'text-pink-600 dark:text-pink-400'
	};

	/** Category-matched hover border color */
	const categoryHoverBorder: Record<string, string> = {
		email: 'hover:border-blue-300 dark:hover:border-blue-700',
		messaging: 'hover:border-violet-300 dark:hover:border-violet-700',
		crm: 'hover:border-orange-300 dark:hover:border-orange-700',
		calendar: 'hover:border-teal-300 dark:hover:border-teal-700',
		sync: 'hover:border-cyan-300 dark:hover:border-cyan-700',
		export: 'hover:border-gray-400 dark:hover:border-gray-600',
		automation: 'hover:border-pink-300 dark:hover:border-pink-700'
	};
</script>

<div
	onclick={() => onClick?.(skill)}
	onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); onClick?.(skill); } }}
	role="button"
	tabindex="0"
	class="skill-card group w-full cursor-pointer text-left rounded-xl border p-5 transition-all duration-200 hover:shadow-lg hover:scale-[1.02]
		{skill.enabled
			? `border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-900 ${categoryHoverBorder[skill.category] ?? 'hover:border-gray-300 dark:hover:border-gray-600'}`
			: 'border-gray-200/60 bg-gray-50 opacity-60 dark:border-gray-800 dark:bg-gray-900/50 hover:opacity-80 hover:border-gray-300 dark:hover:border-gray-700'}"
>
	<!-- Header: Icon + Category + Tier -->
	<div class="mb-3 flex items-start justify-between">
		<div class="flex items-center gap-3">
			<!-- Category Icon (tinted by category) -->
			<div class="flex h-10 w-10 items-center justify-center rounded-lg transition-transform duration-200 group-hover:scale-110
				{categoryIconBg[skill.category] ?? 'bg-gray-100 dark:bg-gray-800'}">
				<svg class="h-5 w-5 {categoryIconColor[skill.category] ?? 'text-gray-600 dark:text-gray-300'}" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
					<path d={categoryIcons[skill.category]} />
				</svg>
			</div>
			<!-- Category Badge -->
			<span class="rounded-full border px-2.5 py-0.5 text-xs font-medium {categoryColors[skill.category]}">
				{categoryLabels[skill.category]}
			</span>
		</div>
		<!-- Tier Badge -->
		<span class="rounded-full border px-2.5 py-0.5 text-xs font-semibold uppercase {tierColors[skill.tier]}">
			{tierLabels[skill.tier]}
		</span>
	</div>

	<!-- Name -->
	<h3 class="mb-1.5 font-mono text-base font-semibold transition-colors
		{skill.enabled
			? 'text-gray-900 group-hover:text-blue-600 dark:text-white dark:group-hover:text-blue-400'
			: 'text-gray-500 dark:text-gray-500'}">
		{skill.name}
	</h3>

	<!-- Description -->
	<p class="mb-4 text-sm leading-relaxed line-clamp-2
		{skill.enabled
			? 'text-gray-500 dark:text-gray-400'
			: 'text-gray-400 dark:text-gray-600'}">
		{skill.description}
	</p>

	<!-- Footer: Status + Toggle -->
	<div class="flex items-center justify-between border-t pt-3
		{skill.enabled
			? 'border-gray-100 dark:border-gray-800'
			: 'border-gray-100/60 dark:border-gray-800/60'}">
		<!-- Status Dot -->
		<div class="flex items-center gap-1.5">
			<span
				class="h-2 w-2 rounded-full {skill.enabled ? 'bg-green-500' : 'bg-gray-300 dark:bg-gray-600'}"
				aria-hidden="true"
			></span>
			<span class="text-sm {skill.enabled ? 'text-gray-500 dark:text-gray-400' : 'text-gray-400 dark:text-gray-600'}">
				{skill.enabled ? 'Active' : 'Inactive'}
			</span>
		</div>

		<!-- Toggle Switch -->
		<button
			onclick={(e: MouseEvent) => {
				e.stopPropagation();
				onToggle?.(skill);
			}}
			role="switch"
			aria-checked={skill.enabled}
			aria-label="Toggle {skill.name}"
			class="relative inline-flex h-5 w-9 items-center rounded-full transition-colors
				{skill.enabled
					? 'bg-blue-600 dark:bg-blue-500'
					: 'bg-gray-200 dark:bg-gray-700'}"
		>
			<span
				class="inline-block h-3.5 w-3.5 rounded-full bg-white shadow-sm transition-transform
					{skill.enabled ? 'translate-x-4' : 'translate-x-0.5'}"
			></span>
		</button>
	</div>
</div>
