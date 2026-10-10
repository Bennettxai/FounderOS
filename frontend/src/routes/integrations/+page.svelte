<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { ArrowRight, CalendarDays, Check } from 'lucide-svelte';

	const provider = $derived($page.url.searchParams.get('connected') ?? 'integration');
	const isCalendar = $derived(provider === 'google_calendar');
	const providerName = $derived(isCalendar ? 'Google Calendar' : provider.replaceAll('_', ' '));
	const appPath = $derived(isCalendar ? '/calendar' : '/settings');
	const deepLink = $derived(`businessos://app${appPath}`);

	onMount(() => {
		const timer = window.setTimeout(() => {
			window.location.href = deepLink;
		}, 650);
		return () => window.clearTimeout(timer);
	});
</script>

<svelte:head>
	<title>{providerName} connected - BusinessOS</title>
	<meta name="description" content={`${providerName} is connected to BusinessOS.`} />
</svelte:head>

<main class="completion-shell">
	<section class="completion-panel" aria-labelledby="completion-title">
		<header class="window-bar" aria-hidden="true">
			<span class="window-dot dot-red"></span>
			<span class="window-dot dot-yellow"></span>
			<span class="window-dot dot-green"></span>
			<span class="window-title">BusinessOS connection</span>
		</header>

		<div class="completion-body">
			<div class="service-icon">
				{#if isCalendar}<CalendarDays size={25} strokeWidth={1.8} />{:else}<Check size={25} strokeWidth={2} />{/if}
			</div>
			<div class="status-line"><span class="status-dot"></span>Connection complete</div>
			<h1 id="completion-title">{providerName} is connected</h1>
			<p>BusinessOS can now securely sync your calendar data. The desktop app should reopen automatically.</p>

			<div class="actions">
				<a class="primary-action" href={deepLink}>Open Calendar <ArrowRight size={16} /></a>
				<a class="secondary-action" href={appPath}>Continue in browser</a>
			</div>
			<p class="close-hint">You can close this browser window after BusinessOS opens.</p>
		</div>
	</section>
</main>

<style>
	:global(html) { background: #f5f5f4; }
	:global(body) { margin: 0; }
	.completion-shell {
		min-height: 100vh;
		display: grid;
		place-items: center;
		padding: 24px;
		box-sizing: border-box;
		background:
			linear-gradient(rgba(24, 24, 27, 0.035) 1px, transparent 1px),
			linear-gradient(90deg, rgba(24, 24, 27, 0.035) 1px, transparent 1px),
			#f5f5f4;
		background-size: 32px 32px;
		color: #18181b;
		font-family: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
	}
	.completion-panel {
		width: min(460px, 100%);
		border: 1px solid #d6d3d1;
		border-radius: 8px;
		background: #ffffff;
		box-shadow: 0 24px 70px rgba(28, 25, 23, 0.14);
		overflow: hidden;
	}
	.window-bar {
		height: 38px;
		display: flex;
		align-items: center;
		gap: 7px;
		padding: 0 14px;
		border-bottom: 1px solid #e7e5e4;
		background: #fafaf9;
	}
	.window-dot { width: 10px; height: 10px; border-radius: 50%; }
	.dot-red { background: #ff5f57; }
	.dot-yellow { background: #febc2e; }
	.dot-green { background: #28c840; }
	.window-title {
		margin-left: auto;
		color: #a8a29e;
		font-size: 11px;
		font-weight: 650;
		text-transform: uppercase;
	}
	.completion-body { padding: 34px; }
	.service-icon {
		width: 48px;
		height: 48px;
		display: grid;
		place-items: center;
		margin-bottom: 24px;
		border: 1px solid #d6d3d1;
		border-radius: 8px;
		background: #fafaf9;
		color: #18181b;
	}
	.status-line {
		display: flex;
		align-items: center;
		gap: 8px;
		margin-bottom: 10px;
		color: #15803d;
		font-size: 12px;
		font-weight: 700;
		text-transform: uppercase;
	}
	.status-dot {
		width: 7px;
		height: 7px;
		border-radius: 50%;
		background: #22c55e;
		box-shadow: 0 0 0 5px rgba(34, 197, 94, 0.12);
	}
	h1 { margin: 0 0 12px; font-size: 28px; line-height: 1.15; font-weight: 760; }
	p { margin: 0; color: #57534e; font-size: 14px; line-height: 1.6; }
	.actions { display: flex; align-items: center; gap: 10px; margin-top: 28px; }
	.primary-action,
	.secondary-action {
		height: 40px;
		display: inline-flex;
		align-items: center;
		justify-content: center;
		gap: 8px;
		padding: 0 15px;
		border-radius: 6px;
		font-size: 13px;
		font-weight: 650;
		text-decoration: none;
	}
	.primary-action { background: #18181b; color: #ffffff; }
	.primary-action:hover { background: #27272a; }
	.secondary-action { border: 1px solid #d6d3d1; color: #44403c; background: #ffffff; }
	.secondary-action:hover { background: #fafaf9; }
	.close-hint { margin-top: 18px; color: #a8a29e; font-size: 12px; }
	@media (max-width: 520px) {
		.completion-shell { padding: 14px; }
		.completion-body { padding: 28px 22px; }
		.actions { align-items: stretch; flex-direction: column; }
		.primary-action, .secondary-action { width: 100%; box-sizing: border-box; }
	}
</style>
