<script lang="ts">
	interface Props {
		companyName: string;
		appVersion: string;
	}

	let { companyName, appVersion }: Props = $props();
</script>

<div class="boot-screen">
	<div class="grid-overlay"></div>
	<div class="boot-content">
		<div class="os-logo">
			<span class="logo-name">{companyName}</span>
			<span class="logo-os">OS</span>
		</div>
		<div class="boot-terminal">
			<div class="terminal-line line-1">
				<span class="prompt">$</span>
				<span class="cmd">init</span>
				<span class="args">--workspace</span>
			</div>
			<div class="terminal-line line-2">
				<span class="output">Loading modules</span>
				<span class="cursor">█</span>
			</div>
		</div>
		<div class="boot-loader">
			<div class="loader-segments">
				<div class="segment s1"></div>
				<div class="segment s2"></div>
				<div class="segment s3"></div>
				<div class="segment s4"></div>
				<div class="segment s5"></div>
			</div>
		</div>
	</div>
	<div class="boot-footer">
		<a href="https://osa.dev" target="_blank" rel="noopener noreferrer" class="osa-link">
			<img src="/osa-logo.png" alt="OSA" class="osa-logo" />
		</a>
		<span class="version">v{appVersion}</span>
	</div>
</div>

<style>
	.boot-screen {
		min-height: 100vh;
		display: flex;
		align-items: center;
		justify-content: center;
		background: #fafafa;
		position: relative;
		overflow: hidden;
	}

	.grid-overlay {
		position: absolute;
		inset: 0;
		background-image:
			linear-gradient(rgba(0,0,0,0.02) 1px, transparent 1px),
			linear-gradient(90deg, rgba(0,0,0,0.02) 1px, transparent 1px);
		background-size: 20px 20px;
		pointer-events: none;
	}

	.boot-content {
		text-align: center;
		z-index: 1;
	}

	.os-logo {
		font-family: 'SF Mono', 'Monaco', 'Fira Code', monospace;
		font-size: 42px;
		font-weight: 800;
		letter-spacing: 6px;
		margin-bottom: 40px;
		display: flex;
		align-items: baseline;
		justify-content: center;
		gap: 2px;
	}

	.logo-name {
		color: #111;
		animation: glitch-text 0.3s ease-out;
	}

	.logo-os {
		color: #111;
		font-weight: 400;
		opacity: 0.4;
		font-size: 36px;
	}

	@keyframes glitch-text {
		0% { opacity: 0; transform: translateX(-10px); }
		20% { opacity: 1; transform: translateX(2px); }
		40% { transform: translateX(-1px); }
		60% { transform: translateX(1px); }
		100% { transform: translateX(0); }
	}

	.boot-terminal {
		font-family: 'SF Mono', 'Monaco', monospace;
		font-size: 13px;
		margin-bottom: 32px;
		text-align: left;
		display: inline-block;
	}

	.terminal-line {
		display: flex;
		align-items: center;
		gap: 8px;
		height: 22px;
		opacity: 0;
		animation: type-line 0.2s ease-out forwards;
	}

	.line-1 { animation-delay: 0.1s; }
	.line-2 { animation-delay: 0.3s; }

	@keyframes type-line {
		from { opacity: 0; transform: translateY(5px); }
		to { opacity: 1; transform: translateY(0); }
	}

	.prompt {
		color: #999;
	}

	.cmd {
		color: #111;
		font-weight: 600;
	}

	.args {
		color: #666;
	}

	.output {
		color: #888;
	}

	.cursor {
		color: #111;
		animation: blink-cursor 0.6s step-end infinite;
		font-size: 12px;
	}

	@keyframes blink-cursor {
		0%, 100% { opacity: 1; }
		50% { opacity: 0; }
	}

	.boot-loader {
		display: flex;
		justify-content: center;
	}

	.loader-segments {
		display: flex;
		gap: 4px;
	}

	.segment {
		width: 24px;
		height: 3px;
		background: #e0e0e0;
		position: relative;
		overflow: hidden;
	}

	.segment::after {
		content: '';
		position: absolute;
		inset: 0;
		background: #111;
		transform: translateX(-100%);
		animation: segment-fill 0.5s ease-out forwards;
	}

	.s1::after { animation-delay: 0s; }
	.s2::after { animation-delay: 0.08s; }
	.s3::after { animation-delay: 0.16s; }
	.s4::after { animation-delay: 0.24s; }
	.s5::after { animation-delay: 0.32s; }

	@keyframes segment-fill {
		to { transform: translateX(0); }
	}

	.boot-footer {
		position: absolute;
		bottom: 32px;
		left: 50%;
		transform: translateX(-50%);
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		opacity: 0;
		animation: fade-in 0.3s ease-out 0.5s forwards;
	}

	.osa-link {
		display: block;
		transition: opacity 0.15s ease, transform 0.15s ease;
	}

	.osa-link:hover {
		opacity: 0.8;
		transform: scale(1.05);
	}

	.osa-logo {
		height: 56px;
		width: auto;
		opacity: 0.5;
	}

	.version {
		font-family: 'SF Mono', 'Monaco', monospace;
		font-size: 10px;
		color: #aaa;
		letter-spacing: 1px;
	}

	@keyframes fade-in {
		from { opacity: 0; transform: translateX(-50%) translateY(5px); }
		to { opacity: 1; transform: translateX(-50%) translateY(0); }
	}

	/* ===== DARK MODE ===== */
	:global(.dark) .boot-screen {
		background: #1c1c1e;
	}

	:global(.dark) .grid-overlay {
		background-image:
			linear-gradient(rgba(255,255,255,0.03) 1px, transparent 1px),
			linear-gradient(90deg, rgba(255,255,255,0.03) 1px, transparent 1px);
	}

	:global(.dark) .logo-name {
		color: #f5f5f7;
	}

	:global(.dark) .logo-os {
		color: #f5f5f7;
		opacity: 0.4;
	}

	:global(.dark) .prompt {
		color: #6e6e73;
	}

	:global(.dark) .cmd {
		color: #f5f5f7;
	}

	:global(.dark) .args {
		color: #a1a1a6;
	}

	:global(.dark) .output {
		color: #a1a1a6;
	}

	:global(.dark) .cursor {
		color: #0A84FF;
	}

	:global(.dark) .segment {
		background: #3a3a3c;
	}

	:global(.dark) .segment::after {
		background: #0A84FF;
	}

	:global(.dark) .version {
		color: #6e6e73;
	}
</style>
