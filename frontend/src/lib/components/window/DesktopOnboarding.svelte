<script lang="ts">
	interface Props {
		step: number;
		onNext: () => void;
		onSkip: () => void;
		onComplete: () => void;
	}

	let { step, onNext, onSkip, onComplete }: Props = $props();
</script>

<div class="onboarding-overlay">
	<div class="onboarding-backdrop"></div>

	{#if step === 0}
		<!-- Highlight Dock -->
		<div class="spotlight-highlight dock-highlight"></div>
		<div class="onboarding-tooltip dock-tooltip">
			<div class="tooltip-content">
				<h3>Your Dock</h3>
				<p>Quick access to all your apps. Click any icon to open it in a window. You can also use the chat bubble for quick AI assistance.</p>
			</div>
			<div class="tooltip-footer">
				<span class="step-indicator">1 of 4</span>
				<div class="tooltip-actions">
					<button class="btn-pill btn-pill-ghost skip-btn" onclick={onSkip}>Skip</button>
					<button class="btn-pill btn-pill-ghost next-btn" onclick={onNext}>Next</button>
				</div>
			</div>
		</div>
	{:else if step === 1}
		<!-- Highlight Menu Bar -->
		<div class="spotlight-highlight menubar-highlight"></div>
		<div class="onboarding-tooltip menubar-tooltip">
			<div class="tooltip-content">
				<h3>Menu Bar</h3>
				<p>Access system functions, view time, and control your workspace from here. Click the company name for quick navigation.</p>
			</div>
			<div class="tooltip-footer">
				<span class="step-indicator">2 of 4</span>
				<div class="tooltip-actions">
					<button class="btn-pill btn-pill-ghost skip-btn" onclick={onSkip}>Skip</button>
					<button class="btn-pill btn-pill-ghost next-btn" onclick={onNext}>Next</button>
				</div>
			</div>
		</div>
	{:else if step === 2}
		<!-- Highlight Desktop Icons -->
		<div class="spotlight-highlight icons-highlight"></div>
		<div class="onboarding-tooltip icons-tooltip">
			<div class="tooltip-content">
				<h3>Desktop Icons</h3>
				<p>Double-click icons to open apps. Drag to rearrange. Right-click for more options like creating folders.</p>
			</div>
			<div class="tooltip-footer">
				<span class="step-indicator">3 of 4</span>
				<div class="tooltip-actions">
					<button class="btn-pill btn-pill-ghost skip-btn" onclick={onSkip}>Skip</button>
					<button class="btn-pill btn-pill-ghost next-btn" onclick={onNext}>Next</button>
				</div>
			</div>
		</div>
	{:else if step === 3}
		<!-- Final welcome -->
		<div class="onboarding-welcome">
			<div class="welcome-icon">
				<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
					<path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
				</svg>
			</div>
			<h2>You're All Set!</h2>
			<p>Press <kbd>⌘</kbd> + <kbd>Space</kbd> anytime to open Spotlight search and find anything quickly.</p>
			<button class="btn-pill btn-pill-ghost get-started-btn" onclick={onComplete}>
				Get Started
			</button>
		</div>
	{/if}
</div>

<style>
	.onboarding-overlay {
		position: fixed;
		inset: 0;
		z-index: 10000;
		pointer-events: auto;
	}

	.onboarding-backdrop {
		position: absolute;
		inset: 0;
		background: transparent;
		pointer-events: none;
	}

	.spotlight-highlight {
		position: fixed;
		background: transparent;
		box-shadow: 0 0 0 9999px rgba(0, 0, 0, 0.75);
		z-index: 10001;
		pointer-events: none;
		animation: pulse-highlight 2s ease-in-out infinite;
	}

	@keyframes pulse-highlight {
		0%, 100% {
			box-shadow: 0 0 0 9999px rgba(0, 0, 0, 0.75), 0 0 0 4px rgba(255, 255, 255, 0.3);
		}
		50% {
			box-shadow: 0 0 0 9999px rgba(0, 0, 0, 0.75), 0 0 0 8px rgba(255, 255, 255, 0.5);
		}
	}

	.dock-highlight {
		bottom: 8px;
		left: 50%;
		transform: translateX(-50%);
		width: calc(100% - 200px);
		max-width: 800px;
		height: 72px;
		border-radius: 20px;
	}

	.menubar-highlight {
		top: 0;
		left: 0;
		right: 0;
		height: 52px;
		border-radius: 0;
		background: transparent;
	}

	.icons-highlight {
		top: 80px;
		left: 20px;
		width: 120px;
		height: 350px;
		border-radius: 16px;
	}

	.onboarding-tooltip {
		position: absolute;
		background: white;
		border-radius: 16px;
		padding: 20px;
		width: 320px;
		box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
		z-index: 10002;
		animation: tooltip-in 0.3s ease-out;
	}

	@keyframes tooltip-in {
		from {
			opacity: 0;
			transform: translateY(10px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	.dock-tooltip {
		bottom: 100px;
		left: 50%;
		transform: translateX(-50%);
	}

	.menubar-tooltip {
		top: 70px;
		left: 50%;
		transform: translateX(-50%);
	}

	.icons-tooltip {
		top: 120px;
		left: 160px;
	}

	.tooltip-content h3 {
		font-size: 18px;
		font-weight: 700;
		color: #111;
		margin: 0 0 8px 0;
	}

	.tooltip-content p {
		font-size: 14px;
		color: #666;
		line-height: 1.5;
		margin: 0;
	}

	.tooltip-footer {
		display: flex;
		align-items: center;
		justify-content: space-between;
		margin-top: 20px;
		padding-top: 16px;
		border-top: 1px solid #eee;
	}

	.step-indicator {
		font-size: 12px;
		color: #999;
		font-weight: 500;
	}

	.tooltip-actions {
		display: flex;
		gap: 10px;
	}

	.skip-btn {
		padding: 8px 16px;
		font-size: 13px;
		font-weight: 500;
		color: #666;
		background: none;
		border: none;
		cursor: pointer;
		border-radius: 8px;
		transition: all 0.15s;
	}

	.skip-btn:hover {
		background: #f5f5f5;
		color: #333;
	}

	.next-btn {
		padding: 8px 20px;
		font-size: 13px;
		font-weight: 600;
		color: white;
		background: #111;
		border: none;
		cursor: pointer;
		border-radius: 8px;
		transition: all 0.15s;
	}

	.next-btn:hover {
		background: #333;
	}

	/* Welcome screen (final step) */
	.onboarding-welcome {
		position: absolute;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		background: white;
		border-radius: 24px;
		padding: 48px;
		text-align: center;
		width: 400px;
		box-shadow: 0 30px 80px rgba(0, 0, 0, 0.3);
		z-index: 10002;
		animation: welcome-in 0.4s ease-out;
	}

	@keyframes welcome-in {
		from {
			opacity: 0;
			transform: translate(-50%, -50%) scale(0.9);
		}
		to {
			opacity: 1;
			transform: translate(-50%, -50%) scale(1);
		}
	}

	.welcome-icon {
		width: 64px;
		height: 64px;
		margin: 0 auto 24px;
		background: linear-gradient(135deg, #10B981 0%, #059669 100%);
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.welcome-icon svg {
		width: 36px;
		height: 36px;
		color: white;
	}

	.onboarding-welcome h2 {
		font-size: 28px;
		font-weight: 700;
		color: #111;
		margin: 0 0 12px 0;
	}

	.onboarding-welcome p {
		font-size: 15px;
		color: #666;
		line-height: 1.6;
		margin: 0 0 28px 0;
	}

	.onboarding-welcome kbd {
		display: inline-block;
		padding: 4px 8px;
		font-family: 'SF Mono', Monaco, monospace;
		font-size: 12px;
		background: #f0f0f0;
		border: 1px solid #ddd;
		border-radius: 4px;
		box-shadow: 0 1px 2px rgba(0,0,0,0.1);
	}

	.get-started-btn {
		padding: 14px 32px;
		font-size: 15px;
		font-weight: 600;
		color: white;
		background: #111;
		border: none;
		cursor: pointer;
		border-radius: 12px;
		transition: all 0.15s;
	}

	.get-started-btn:hover {
		background: #333;
		transform: translateY(-1px);
	}

	/* ===== DARK MODE ===== */
	:global(.dark) .onboarding-tooltip {
		background: #2c2c2e;
		box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
	}

	:global(.dark) .tooltip-content h3 {
		color: #f5f5f7;
	}

	:global(.dark) .tooltip-content p {
		color: #a1a1a6;
	}

	:global(.dark) .tooltip-footer {
		border-top-color: rgba(255, 255, 255, 0.1);
	}

	:global(.dark) .step-indicator {
		color: #6e6e73;
	}

	:global(.dark) .skip-btn {
		color: #a1a1a6;
	}

	:global(.dark) .skip-btn:hover {
		background: #3a3a3c;
		color: #f5f5f7;
	}

	:global(.dark) .next-btn {
		background: #0A84FF;
	}

	:global(.dark) .next-btn:hover {
		background: #409CFF;
	}

	:global(.dark) .onboarding-welcome {
		background: #2c2c2e;
	}

	:global(.dark) .onboarding-welcome h2 {
		color: #f5f5f7;
	}

	:global(.dark) .onboarding-welcome p {
		color: #a1a1a6;
	}

	:global(.dark) .onboarding-welcome kbd {
		background: #3a3a3c;
		border-color: #48484a;
		color: #f5f5f7;
	}

	:global(.dark) .get-started-btn {
		background: #0A84FF;
	}

	:global(.dark) .get-started-btn:hover {
		background: #409CFF;
	}
</style>
