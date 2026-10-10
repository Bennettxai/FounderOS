<script lang="ts">
	import type { CustomModule } from '$lib/types/modules';
	import { categoryLabels } from '$lib/types/modules';
	import { getCategoryColor } from '$lib/constants/colors';
	import { getModuleIcon } from '$lib/components/modules/moduleIcons';
	import { Download, Star } from 'lucide-svelte';

	interface Props {
		module: CustomModule;
		compact?: boolean;
		onClick?: () => void;
	}

	let { module, compact = false, onClick }: Props = $props();

	const catColor = $derived(getCategoryColor(module.category));
	const ModIcon = $derived(getModuleIcon(module.icon, module.category));

	function fmtNum(n: number): string {
		if (n >= 1000) return (n / 1000).toFixed(n >= 10000 ? 0 : 1) + 'K';
		return String(n);
	}
</script>

{#if compact}
	<!-- List / Compact Row -->
	<button onclick={onClick} class="am-row" aria-label="View {module.name}">
		<div class="am-row__icon" style="background: {catColor}; color: var(--bos-surface-on-color);">
			<ModIcon class="w-4 h-4" />
		</div>
		<div class="am-row__info">
			<span class="am-row__name">{module.name}</span>
			<span class="am-row__desc">{module.description}</span>
		</div>
		<span class="am-row__cat" style="background: color-mix(in srgb, {catColor} 8%, transparent); color: {catColor};">
			{categoryLabels[module.category]}
		</span>
		<div class="am-row__stats">
			<span class="am-row__stat"><Download class="w-3 h-3" />{fmtNum(module.install_count)}</span>
			<span class="am-row__stat"><Star class="w-3 h-3" />{fmtNum(module.star_count)}</span>
		</div>
		<span class="am-row__version">v{module.version}</span>
		{#if module.creator_name}
			<span class="am-row__author">{module.creator_name}</span>
		{/if}
	</button>
{:else}
	<!-- Grid Card -->
	<button onclick={onClick} class="am-card" aria-label="View {module.name}">
		<!-- Top: Icon + Title + Badge -->
		<div class="am-card__header">
			<div class="am-card__icon" style="background: {catColor}; color: var(--bos-surface-on-color);">
				<ModIcon class="w-5 h-5" />
			</div>
			<div class="am-card__title-area">
				<h3 class="am-card__name">{module.name}</h3>
				<span class="am-card__version">v{module.version}</span>
			</div>
		</div>

		<!-- Description -->
		<p class="am-card__desc">{module.description}</p>

		<!-- Footer: Category + Stats + Author -->
		<div class="am-card__footer">
			<span class="am-card__cat" style="background: color-mix(in srgb, {catColor} 8%, transparent); color: {catColor};">
				{categoryLabels[module.category]}
			</span>
			<div class="am-card__meta">
				<span class="am-card__stat" title="Installs">
					<Download class="w-3 h-3" />
					{fmtNum(module.install_count)}
				</span>
				<span class="am-card__stat" title="Stars">
					<Star class="w-3 h-3" />
					{fmtNum(module.star_count)}
				</span>
				{#if module.creator_name}
					<span class="am-card__author">{module.creator_name}</span>
				{/if}
			</div>
		</div>
	</button>
{/if}

<style>
	/* ══════════════════════════════════════════════════════════════ */
	/*  MODULE CARD v2 (am-card-) — Foundation Design Tokens        */
	/* ══════════════════════════════════════════════════════════════ */

	/* ── Grid Card ─────────────────────────────────────────────── */
	.am-card {
		display: flex;
		flex-direction: column;
		width: 100%;
		text-align: left;
		padding: 16px;
		border-radius: 12px;
		border: 1px solid var(--dbd2);
		background: var(--dbg);
		cursor: pointer;
		transition: border-color 0.15s, box-shadow 0.15s, transform 0.15s;
	}
	.am-card:hover {
		border-color: var(--dbd);
		box-shadow: 0 4px 16px rgba(0, 0, 0, 0.06);
		transform: translateY(-2px);
	}

	.am-card__header {
		display: flex;
		align-items: center;
		gap: 10px;
		margin-bottom: 10px;
	}
	.am-card__icon {
		width: 38px;
		height: 38px;
		border-radius: 10px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}
	.am-card__title-area {
		flex: 1;
		min-width: 0;
	}
	.am-card__name {
		font-size: 14px;
		font-weight: 600;
		color: var(--dt);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
		line-height: 1.3;
	}
	.am-card:hover .am-card__name {
		color: var(--dt);
	}
	.am-card__version {
		font-size: 11px;
		color: var(--dt4);
		font-weight: 500;
	}

	.am-card__desc {
		font-size: 12px;
		color: var(--dt3);
		line-height: 1.5;
		display: -webkit-box;
		-webkit-line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
		margin-bottom: 12px;
		flex: 1;
	}

	.am-card__footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 8px;
		padding-top: 10px;
		border-top: 1px solid var(--dbd2);
	}
	.am-card__cat {
		display: inline-flex;
		align-items: center;
		padding: 2px 8px;
		border-radius: 999px;
		font-size: 10px;
		font-weight: 600;
		white-space: nowrap;
		flex-shrink: 0;
	}
	.am-card__meta {
		display: flex;
		align-items: center;
		gap: 10px;
		min-width: 0;
	}
	.am-card__stat {
		display: inline-flex;
		align-items: center;
		gap: 3px;
		font-size: 11px;
		color: var(--dt3);
	}
	.am-card__author {
		font-size: 11px;
		color: var(--dt4);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	/* ── List / Compact Row ────────────────────────────────────── */
	.am-row {
		display: flex;
		align-items: center;
		gap: 12px;
		width: 100%;
		text-align: left;
		padding: 10px 14px;
		border-radius: 10px;
		border: 1px solid var(--dbd2);
		background: var(--dbg);
		cursor: pointer;
		transition: border-color 0.15s, background 0.15s;
	}
	.am-row:hover {
		border-color: var(--dbd);
		background: var(--dbg2);
	}
	.am-row__icon {
		width: 30px;
		height: 30px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		flex-shrink: 0;
	}
	.am-row__info {
		flex: 1;
		min-width: 0;
		display: flex;
		flex-direction: column;
		gap: 1px;
	}
	.am-row__name {
		font-size: 13px;
		font-weight: 600;
		color: var(--dt);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.am-row__desc {
		font-size: 11px;
		color: var(--dt3);
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.am-row__cat {
		display: inline-flex;
		align-items: center;
		padding: 2px 8px;
		border-radius: 999px;
		font-size: 10px;
		font-weight: 600;
		white-space: nowrap;
		flex-shrink: 0;
	}
	.am-row__stats {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-shrink: 0;
	}
	.am-row__stat {
		display: inline-flex;
		align-items: center;
		gap: 3px;
		font-size: 11px;
		color: var(--dt3);
	}
	.am-row__version {
		font-size: 11px;
		color: var(--dt4);
		flex-shrink: 0;
	}
	.am-row__author {
		font-size: 11px;
		color: var(--dt4);
		flex-shrink: 0;
		white-space: nowrap;
	}
</style>
