<script lang="ts">
	/**
	 * ChartView - Chart visualization for app templates
	 * Lightweight SVG-based charts without external dependencies
	 */

	import type { Field } from '../types/field';
	import type { ChartViewConfig } from '../types/view';
	import { TemplateSkeleton } from '../primitives';

	interface Props {
		config: ChartViewConfig;
		fields: Field[];
		data: Record<string, unknown>[];
		loading?: boolean;
		onpointclick?: (record: Record<string, unknown>) => void;
	}

	let {
		config,
		fields,
		data,
		loading = false,
		onpointclick
	}: Props = $props();

	const chartWidth = 600;
	const chartHeight = 400;
	const padding = { top: 40, right: 40, bottom: 60, left: 60 };

	// Colors for chart elements
	const colors = [
		'var(--tpl-accent-primary)',
		'var(--tpl-status-success)',
		'var(--tpl-status-warning)',
		'var(--tpl-status-error)',
		'var(--tpl-accent-secondary)',
		'var(--tpl-status-info)'
	];

	// Aggregate data based on config
	const chartData = $derived(() => {
		if (!data.length) return [];

		const aggregated = new Map<string, number>();

		data.forEach(record => {
			const xValue = String(record[config.xAxisField] || 'Unknown');
			const yValue = Number(record[config.yAxisField] || 0);

			if (config.aggregation === 'count') {
				aggregated.set(xValue, (aggregated.get(xValue) || 0) + 1);
			} else if (config.aggregation === 'sum') {
				aggregated.set(xValue, (aggregated.get(xValue) || 0) + yValue);
			} else if (config.aggregation === 'average') {
				// For average, we need to track sum and count
				const key = `${xValue}_sum`;
				const countKey = `${xValue}_count`;
				aggregated.set(key, (aggregated.get(key) || 0) + yValue);
				aggregated.set(countKey, (aggregated.get(countKey) || 0) + 1);
			} else {
				aggregated.set(xValue, yValue);
			}
		});

		// If averaging, calculate final values
		if (config.aggregation === 'average') {
			const finalData: { label: string; value: number }[] = [];
			const labels = new Set<string>();
			aggregated.forEach((_, key) => {
				if (!key.endsWith('_sum') && !key.endsWith('_count')) return;
				const label = key.replace(/_sum$|_count$/, '');
				labels.add(label);
			});
			labels.forEach(label => {
				const sum = aggregated.get(`${label}_sum`) || 0;
				const count = aggregated.get(`${label}_count`) || 1;
				finalData.push({ label, value: sum / count });
			});
			return finalData;
		}

		return Array.from(aggregated.entries()).map(([label, value]) => ({ label, value }));
	});

	// Calculate scales
	const maxValue = $derived(Math.max(...chartData().map(d => d.value), 1));
	const innerWidth = chartWidth - padding.left - padding.right;
	const innerHeight = chartHeight - padding.top - padding.bottom;

	// Generate Y-axis ticks
	const yTicks = $derived(() => {
		const tickCount = 5;
		const step = maxValue / tickCount;
		return Array.from({ length: tickCount + 1 }, (_, i) => Math.round(step * i));
	});

	// Get bar dimensions
	function getBarX(index: number): number {
		const barCount = chartData().length;
		const barWidth = innerWidth / barCount;
		return padding.left + barWidth * index + barWidth * 0.1;
	}

	function getBarWidth(): number {
		const barCount = chartData().length;
		return (innerWidth / barCount) * 0.8;
	}

	function getBarHeight(value: number): number {
		return (value / maxValue) * innerHeight;
	}

	// Get point position for line/area charts
	function getPointX(index: number): number {
		const pointCount = chartData().length;
		if (pointCount === 1) return padding.left + innerWidth / 2;
		return padding.left + (innerWidth / (pointCount - 1)) * index;
	}

	function getPointY(value: number): number {
		return padding.top + innerHeight - (value / maxValue) * innerHeight;
	}

	// Generate line path
	const linePath = $derived(() => {
		const points = chartData().map((d, i) => `${getPointX(i)},${getPointY(d.value)}`);
		return `M ${points.join(' L ')}`;
	});

	// Generate area path
	const areaPath = $derived(() => {
		const points = chartData().map((d, i) => `${getPointX(i)},${getPointY(d.value)}`);
		const baseline = padding.top + innerHeight;
		return `M ${padding.left},${baseline} L ${points.join(' L ')} L ${getPointX(chartData().length - 1)},${baseline} Z`;
	});

	// Pie chart calculations
	const pieData = $derived(() => {
		const total = chartData().reduce((sum, d) => sum + d.value, 0);
		let currentAngle = 0;

		return chartData().map((d, i) => {
			const angle = (d.value / total) * 360;
			const startAngle = currentAngle;
			currentAngle += angle;
			return {
				...d,
				startAngle,
				endAngle: currentAngle,
				percentage: ((d.value / total) * 100).toFixed(1),
				color: colors[i % colors.length]
			};
		});
	});

	function polarToCartesian(centerX: number, centerY: number, radius: number, angleInDegrees: number) {
		const angleInRadians = ((angleInDegrees - 90) * Math.PI) / 180.0;
		return {
			x: centerX + radius * Math.cos(angleInRadians),
			y: centerY + radius * Math.sin(angleInRadians)
		};
	}

	function describeArc(x: number, y: number, radius: number, startAngle: number, endAngle: number) {
		const start = polarToCartesian(x, y, radius, endAngle);
		const end = polarToCartesian(x, y, radius, startAngle);
		const largeArcFlag = endAngle - startAngle <= 180 ? '0' : '1';
		return `M ${start.x} ${start.y} A ${radius} ${radius} 0 ${largeArcFlag} 0 ${end.x} ${end.y}`;
	}

	function describeSlice(x: number, y: number, radius: number, startAngle: number, endAngle: number) {
		const start = polarToCartesian(x, y, radius, endAngle);
		const end = polarToCartesian(x, y, radius, startAngle);
		const largeArcFlag = endAngle - startAngle <= 180 ? '0' : '1';
		return `M ${x} ${y} L ${start.x} ${start.y} A ${radius} ${radius} 0 ${largeArcFlag} 0 ${end.x} ${end.y} Z`;
	}

	const pieCenterX = chartWidth / 2;
	const pieCenterY = chartHeight / 2;
	const pieRadius = Math.min(innerWidth, innerHeight) / 2 - 20;
	const donutInnerRadius = pieRadius * 0.6;
</script>

<div class="tpl-chart-view">
	{#if loading}
		<div class="tpl-chart-loading">
			<TemplateSkeleton variant="rectangular" width="100%" height="400px" />
		</div>
	{:else if data.length === 0}
		<div class="tpl-chart-empty">
			<p>No data to display</p>
		</div>
	{:else}
		<svg viewBox="0 0 {chartWidth} {chartHeight}" class="tpl-chart-svg">
			{#if config.chartType === 'bar'}
				<!-- Bar Chart -->
				{#if config.showGrid}
					{#each yTicks() as tick}
						<line
							x1={padding.left}
							y1={getPointY(tick)}
							x2={chartWidth - padding.right}
							y2={getPointY(tick)}
							class="tpl-chart-grid-line"
						/>
					{/each}
				{/if}

				{#each chartData() as item, i}
					<rect
						x={getBarX(i)}
						y={padding.top + innerHeight - getBarHeight(item.value)}
						width={getBarWidth()}
						height={getBarHeight(item.value)}
						fill={colors[i % colors.length]}
						class="tpl-chart-bar"
						onclick={() => onpointclick?.(data[i])}
					/>
					{#if config.showLabels}
						<text
							x={getBarX(i) + getBarWidth() / 2}
							y={padding.top + innerHeight - getBarHeight(item.value) - 5}
							text-anchor="middle"
							class="tpl-chart-label"
						>
							{Math.round(item.value)}
						</text>
					{/if}
				{/each}

				<!-- X Axis Labels -->
				{#each chartData() as item, i}
					<text
						x={getBarX(i) + getBarWidth() / 2}
						y={chartHeight - padding.bottom + 20}
						text-anchor="middle"
						class="tpl-chart-axis-label"
					>
						{item.label.length > 10 ? item.label.slice(0, 10) + '...' : item.label}
					</text>
				{/each}

			{:else if config.chartType === 'line' || config.chartType === 'area'}
				<!-- Line/Area Chart -->
				{#if config.showGrid}
					{#each yTicks() as tick}
						<line
							x1={padding.left}
							y1={getPointY(tick)}
							x2={chartWidth - padding.right}
							y2={getPointY(tick)}
							class="tpl-chart-grid-line"
						/>
					{/each}
				{/if}

				{#if config.chartType === 'area'}
					<path d={areaPath()} class="tpl-chart-area" />
				{/if}

				<path d={linePath()} class="tpl-chart-line" />

				{#each chartData() as item, i}
					<circle
						cx={getPointX(i)}
						cy={getPointY(item.value)}
						r="6"
						class="tpl-chart-point"
						onclick={() => onpointclick?.(data[i])}
					/>
					{#if config.showLabels}
						<text
							x={getPointX(i)}
							y={getPointY(item.value) - 12}
							text-anchor="middle"
							class="tpl-chart-label"
						>
							{Math.round(item.value)}
						</text>
					{/if}
				{/each}

				<!-- X Axis Labels -->
				{#each chartData() as item, i}
					<text
						x={getPointX(i)}
						y={chartHeight - padding.bottom + 20}
						text-anchor="middle"
						class="tpl-chart-axis-label"
					>
						{item.label.length > 10 ? item.label.slice(0, 10) + '...' : item.label}
					</text>
				{/each}

			{:else if config.chartType === 'pie' || config.chartType === 'donut'}
				<!-- Pie/Donut Chart -->
				{#each pieData() as slice}
					{#if config.chartType === 'donut'}
						<path
							d={describeArc(pieCenterX, pieCenterY, pieRadius, slice.startAngle, slice.endAngle)}
							fill="none"
							stroke={slice.color}
							stroke-width={pieRadius - donutInnerRadius}
							class="tpl-chart-slice"
						/>
					{:else}
						<path
							d={describeSlice(pieCenterX, pieCenterY, pieRadius, slice.startAngle, slice.endAngle)}
							fill={slice.color}
							class="tpl-chart-slice"
						/>
					{/if}
				{/each}

				{#if config.chartType === 'donut'}
					<!-- Center text for donut -->
					<text x={pieCenterX} y={pieCenterY} text-anchor="middle" class="tpl-chart-donut-total">
						{chartData().reduce((sum, d) => sum + d.value, 0)}
					</text>
					<text x={pieCenterX} y={pieCenterY + 16} text-anchor="middle" class="tpl-chart-donut-label">
						Total
					</text>
				{/if}

			{/if}

			<!-- Y Axis -->
			{#if config.chartType !== 'pie' && config.chartType !== 'donut'}
				<line
					x1={padding.left}
					y1={padding.top}
					x2={padding.left}
					y2={chartHeight - padding.bottom}
					class="tpl-chart-axis"
				/>
				{#each yTicks() as tick}
					<text
						x={padding.left - 10}
						y={getPointY(tick) + 4}
						text-anchor="end"
						class="tpl-chart-axis-label"
					>
						{tick}
					</text>
				{/each}

				<!-- X Axis -->
				<line
					x1={padding.left}
					y1={chartHeight - padding.bottom}
					x2={chartWidth - padding.right}
					y2={chartHeight - padding.bottom}
					class="tpl-chart-axis"
				/>
			{/if}
		</svg>

		{#if config.showLegend && (config.chartType === 'pie' || config.chartType === 'donut')}
			<div class="tpl-chart-legend">
				{#each pieData() as slice}
					<div class="tpl-chart-legend-item">
						<span class="tpl-chart-legend-color" style:background={slice.color}></span>
						<span class="tpl-chart-legend-label">{slice.label}</span>
						<span class="tpl-chart-legend-value">{slice.percentage}%</span>
					</div>
				{/each}
			</div>
		{/if}
	{/if}
</div>

<style>
	.tpl-chart-view {
		padding: var(--tpl-space-4);
		display: flex;
		flex-direction: column;
		align-items: center;
	}

	.tpl-chart-loading,
	.tpl-chart-empty {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 400px;
	}

	.tpl-chart-empty {
		color: var(--tpl-text-muted);
		font-family: var(--tpl-font-sans);
	}

	.tpl-chart-svg {
		width: 100%;
		max-width: 800px;
		height: auto;
	}

	.tpl-chart-grid-line {
		stroke: var(--tpl-border-subtle);
		stroke-width: 1;
	}

	.tpl-chart-axis {
		stroke: var(--tpl-border-default);
		stroke-width: 1;
	}

	.tpl-chart-axis-label {
		font-family: var(--tpl-font-sans);
		font-size: 11px;
		fill: var(--tpl-text-muted);
	}

	.tpl-chart-label {
		font-family: var(--tpl-font-sans);
		font-size: 12px;
		font-weight: 500;
		fill: var(--tpl-text-primary);
	}

	.tpl-chart-bar {
		cursor: pointer;
		transition: opacity var(--tpl-transition-fast);
	}

	.tpl-chart-bar:hover {
		opacity: 0.8;
	}

	.tpl-chart-line {
		fill: none;
		stroke: var(--tpl-accent-primary);
		stroke-width: 2;
		stroke-linecap: round;
		stroke-linejoin: round;
	}

	.tpl-chart-area {
		fill: var(--tpl-accent-primary);
		opacity: 0.2;
	}

	.tpl-chart-point {
		fill: var(--tpl-accent-primary);
		stroke: var(--tpl-bg-primary);
		stroke-width: 2;
		cursor: pointer;
		transition: r var(--tpl-transition-fast);
	}

	.tpl-chart-point:hover {
		r: 8;
	}

	.tpl-chart-slice {
		cursor: pointer;
		transition: opacity var(--tpl-transition-fast);
	}

	.tpl-chart-slice:hover {
		opacity: 0.85;
	}

	.tpl-chart-donut-total {
		font-family: var(--tpl-font-sans);
		font-size: 24px;
		font-weight: 600;
		fill: var(--tpl-text-primary);
	}

	.tpl-chart-donut-label {
		font-family: var(--tpl-font-sans);
		font-size: 12px;
		fill: var(--tpl-text-muted);
	}

	.tpl-chart-legend {
		display: flex;
		flex-wrap: wrap;
		gap: var(--tpl-space-4);
		margin-top: var(--tpl-space-4);
		justify-content: center;
	}

	.tpl-chart-legend-item {
		display: flex;
		align-items: center;
		gap: var(--tpl-space-2);
		font-family: var(--tpl-font-sans);
		font-size: var(--tpl-text-sm);
	}

	.tpl-chart-legend-color {
		width: 12px;
		height: 12px;
		border-radius: var(--tpl-radius-sm);
	}

	.tpl-chart-legend-label {
		color: var(--tpl-text-secondary);
	}

	.tpl-chart-legend-value {
		color: var(--tpl-text-muted);
	}
</style>
