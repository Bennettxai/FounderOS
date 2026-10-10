<script lang="ts">
	import {
		desktopSettings,
		type WindowAnimationType,
		type AnimationSpeed
	} from '$lib/stores/desktopStore';
</script>

<!-- Window Animations -->
<div class="section">
	<label class="section-title">Window Animations</label>
	<p class="section-subtitle">Customize how windows open, close, and minimize</p>

	<div class="settings-row-group">
		<div class="settings-row">
			<div class="settings-row-label">
				<span class="settings-label-text">Open Animation</span>
				<span class="settings-label-desc">Effect when windows appear</span>
			</div>
			<select
				class="settings-select"
				value={$desktopSettings.windowAnimations.openAnimation}
				onchange={(e) => desktopSettings.setWindowAnimations({ openAnimation: e.currentTarget.value as WindowAnimationType })}
			>
				<option value="none">None</option>
				<option value="fade">Fade</option>
				<option value="scale">Scale</option>
				<option value="slide">Slide</option>
				<option value="bounce">Bounce</option>
			</select>
		</div>

		<div class="settings-row">
			<div class="settings-row-label">
				<span class="settings-label-text">Close Animation</span>
				<span class="settings-label-desc">Effect when windows disappear</span>
			</div>
			<select
				class="settings-select"
				value={$desktopSettings.windowAnimations.closeAnimation}
				onchange={(e) => desktopSettings.setWindowAnimations({ closeAnimation: e.currentTarget.value as WindowAnimationType })}
			>
				<option value="none">None</option>
				<option value="fade">Fade</option>
				<option value="scale">Scale</option>
				<option value="slide">Slide</option>
			</select>
		</div>

		<div class="settings-row">
			<div class="settings-row-label">
				<span class="settings-label-text">Animation Speed</span>
				<span class="settings-label-desc">How fast animations play</span>
			</div>
			<select
				class="settings-select"
				value={$desktopSettings.windowAnimations.speed}
				onchange={(e) => desktopSettings.setWindowAnimations({ speed: e.currentTarget.value as AnimationSpeed })}
			>
				<option value="fast">Fast</option>
				<option value="normal">Normal</option>
				<option value="slow">Slow</option>
			</select>
		</div>
	</div>
</div>

<style>
	.section {
		margin-top: 32px;
		padding-top: 24px;
		border-top: 1px solid #e5e5e5;
		margin-bottom: 24px;
	}

	.section-title {
		font-size: 13px;
		font-weight: 600;
		color: #333;
		display: block;
		margin-bottom: 12px;
	}

	.section-subtitle {
		font-size: 11px;
		color: #999;
		margin: -4px 0 8px 0;
	}

	.settings-row-group {
		display: flex;
		flex-direction: column;
		gap: 0;
		background: #fafafa;
		border: 1px solid #e5e5e5;
		border-radius: 8px;
		overflow: hidden;
	}

	.settings-row {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 12px 16px;
		border-bottom: 1px solid #e5e5e5;
	}

	.settings-row:last-child {
		border-bottom: none;
	}

	.settings-row-label {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.settings-label-text {
		font-size: 13px;
		font-weight: 500;
		color: #333;
	}

	.settings-label-desc {
		font-size: 11px;
		color: #888;
	}

	.settings-select {
		padding: 8px 32px 8px 12px;
		background: white;
		border: 1px solid #e0e0e0;
		border-radius: 6px;
		font-size: 13px;
		color: #333;
		cursor: pointer;
		min-width: 120px;
		appearance: none;
		background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%23666' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpolyline points='6 9 12 15 18 9'%3E%3C/polyline%3E%3C/svg%3E");
		background-repeat: no-repeat;
		background-position: right 10px center;
		transition: border-color 0.15s ease, box-shadow 0.15s ease;
	}

	.settings-select:hover {
		border-color: #999;
	}

	.settings-select:focus {
		outline: none;
		border-color: #666;
		box-shadow: 0 0 0 2px rgba(0, 0, 0, 0.08);
	}
</style>
