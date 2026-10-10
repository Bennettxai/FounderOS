<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { browser } from '$app/environment';
	import { page } from '$app/stores';
	import { themeStore } from '$lib/stores/themeStore';
	import CookieConsent from '$lib/components/CookieConsent.svelte';
	import { isOperatorView } from '$lib/founderos/chrome/chrome';
	import UpdateBanner from '$lib/components/updates/UpdateBanner.svelte';

	let { children } = $props();

	// Desktop-only: whether we're running inside the Electron shell (frameless
	// window with a hidden inset title bar on macOS).
	let isDesktop = $state(false);
	const dragStripRoutes = ['/login', '/register', '/forgot-password', '/reset-password', '/welcome', '/onboarding', '/auth/callback'];
	const showDesktopDragStrip = $derived(
		isDesktop && dragStripRoutes.some((route) => $page.url.pathname === route || $page.url.pathname.startsWith(`${route}/`))
	);

	// Initialize theme on mount
	onMount(() => {
		// Theme is already applied by the store on creation,
		// but we ensure it's set on the document
		const storedTheme = localStorage.getItem('theme');
		if (storedTheme === 'dark' || storedTheme === 'light' || storedTheme === 'system') {
			themeStore.setTheme(storedTheme);
		}

		isDesktop = browser && 'electron' in window;
	});
</script>

<svelte:head>
	<title>Business OS</title>
	<meta name="description" content="Your internal command center" />
</svelte:head>

{#if showDesktopDragStrip}
	<!--
		Global draggable title-bar strip for the frameless desktop window. The
		authenticated (app) shell has its own drag region, but pre-auth screens
		(login, welcome, onboarding) had none — so the window could not be moved
		from the top while signed out. This thin top strip fixes that everywhere
		without covering interactive content (it sits in the empty traffic-light
		band). Interactive elements set `-webkit-app-region: no-drag` themselves.
	-->
	<div class="desktop-drag-strip" aria-hidden="true"></div>
{/if}

{@render children()}
{#if !isOperatorView($page.url.pathname)}
	<CookieConsent />
{/if}
<UpdateBanner />

<style>
	.desktop-drag-strip {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		height: 28px;
		z-index: 2147483646;
		-webkit-app-region: drag;
	}
</style>
