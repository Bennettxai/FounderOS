<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { Calendar, Mail, MessageSquare } from 'lucide-svelte';
	import { Toaster, toast } from 'svelte-sonner';
	import {
		commsStreamStatus,
		connectCommsStream,
		type CommsStreamHandle,
	} from '$lib/api/comms';

	let { children } = $props();

	const tabs = [
		{ href: '/communication/calendar', label: 'Calendar', icon: Calendar },
		{ href: '/communication/email', label: 'Email', icon: Mail },
		{ href: '/communication/channels', label: 'Channels', icon: MessageSquare },
	];

	const isActiveTab = (tabHref: string) => {
		return $page.url.pathname.startsWith(tabHref);
	};

	// ─── Realtime stream lifecycle ───
	// One EventSource for the whole comms hub. Tabs share the connection via
	// the subscriber API in $lib/api/comms/stream.
	let streamHandle: CommsStreamHandle | null = null;
	let reconnectingToastId: string | number | null = null;
	let lostToastId: string | number | null = null;
	let reconnectingTimer: ReturnType<typeof setTimeout> | null = null;
	let lostTimer: ReturnType<typeof setTimeout> | null = null;

	function clearReconnectingToast() {
		if (reconnectingToastId !== null) {
			toast.dismiss(reconnectingToastId);
			reconnectingToastId = null;
		}
		if (reconnectingTimer) {
			clearTimeout(reconnectingTimer);
			reconnectingTimer = null;
		}
	}

	function clearLostToast() {
		if (lostToastId !== null) {
			toast.dismiss(lostToastId);
			lostToastId = null;
		}
		if (lostTimer) {
			clearTimeout(lostTimer);
			lostTimer = null;
		}
	}

	$effect(() => {
		const status = $commsStreamStatus;
		if (status === 'connected') {
			clearReconnectingToast();
			clearLostToast();
			return;
		}
		if (status !== 'reconnecting' && status !== 'connecting') return;

		// Only schedule the toasts once per disconnect window.
		if (!reconnectingTimer && reconnectingToastId === null) {
			reconnectingTimer = setTimeout(() => {
				reconnectingToastId = toast.loading('Reconnecting to live updates…', {
					duration: Infinity,
				});
			}, 5_000);
		}
		if (!lostTimer && lostToastId === null) {
			lostTimer = setTimeout(() => {
				clearReconnectingToast();
				lostToastId = toast.error(
					'Live updates disconnected — refresh if it persists',
					{ duration: Infinity },
				);
			}, 60_000);
		}
	});

	onMount(() => {
		streamHandle = connectCommsStream();
		return () => {
			streamHandle?.close();
			streamHandle = null;
			clearReconnectingToast();
			clearLostToast();
		};
	});
</script>

<div class="ch-layout">
	<!-- Header with Tabs -->
	<div class="ch-layout__header">
		<div class="ch-layout__header-inner">
			<h1 class="ch-layout__title">Communication Hub</h1>
			<nav class="ch-layout__tabs">
				{#each tabs as tab}
					<a
						href={tab.href}
						class="ch-layout__tab"
						class:ch-layout__tab--active={isActiveTab(tab.href)}
						aria-label="{tab.label} tab"
					>
						<span class="ch-layout__tab-icon">
							<tab.icon size={16} strokeWidth={1.5} />
						</span>
						{tab.label}
					</a>
				{/each}
			</nav>
		</div>
	</div>

	<!-- Content -->
	<div class="ch-layout__content">
		{@render children()}
	</div>
</div>

<Toaster
	position="bottom-right"
	richColors
	closeButton
	toastOptions={{
		style: 'background: var(--dbg2); color: var(--dt); border: 1px solid var(--dbd);'
	}}
/>

<style>
	.ch-layout {
		flex: 1 1 0%;
		min-height: 0;
		min-width: 0;
		display: flex;
		flex-direction: column;
		overflow: hidden;
	}

	.ch-layout__header {
		border-bottom: 1px solid var(--dbd);
		background: var(--dbg);
		flex-shrink: 0;
	}

	.ch-layout__header-inner {
		padding: var(--space-4) var(--space-6) 0;
	}

	.ch-layout__title {
		font-size: var(--text-2xl);
		font-weight: var(--font-bold);
		color: var(--dt);
		margin: 0 0 var(--space-4) 0;
	}

	.ch-layout__tabs {
		display: flex;
		gap: var(--space-2);
	}

	.ch-layout__tab {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: 6px var(--space-4);
		font-size: 13px;
		font-weight: 500;
		color: var(--dt3);
		text-decoration: none;
		border-radius: var(--radius-sm) var(--radius-sm) 0 0;
		border-bottom: 2px solid transparent;
		position: relative;
		transition: color 150ms ease, background 150ms ease;
		/* pull bottom of pill flush with header border */
		margin-bottom: -1px;
	}

	.ch-layout__tab-icon {
		display: flex;
		align-items: center;
		flex-shrink: 0;
	}

	.ch-layout__tab:hover {
		color: var(--dt2);
		background: var(--dbg3);
	}

	.ch-layout__tab--active {
		color: var(--bos-nav-active);
		background: var(--bos-nav-active-bg);
		border-bottom-color: var(--bos-nav-active);
	}

	.ch-layout__tab--active:hover {
		background: var(--bos-nav-active-bg);
		color: var(--bos-nav-active);
	}

	.ch-layout__content {
		flex: 1;
		min-height: 0;
		overflow: hidden;
		display: flex;
		flex-direction: column;
	}
</style>
