<!--
  ThemeToggle.svelte
  Light/dark mode toggle button
-->
<script lang="ts">
	import { SunIcon, MoonIcon } from './icons';
	import { themeStore } from '$lib/stores/themeStore';

	interface Props {
		isDark?: boolean;
		onToggle?: (isDark: boolean) => void;
		class?: string;
	}

	let { isDark = $bindable(false), onToggle, class: className = '' }: Props = $props();

	// Sync with themeStore
	const unsubscribe = themeStore.subscribe(state => {
		isDark = state.resolvedTheme === 'dark';
	});

	function handleToggle() {
		const newTheme = isDark ? 'light' : 'dark';
		themeStore.setTheme(newTheme);
		isDark = newTheme === 'dark';
		onToggle?.(isDark);
	}
</script>

<button
	type="button"
	class="btn-pill btn-pill-ghost theme-toggle {className}"
	onclick={handleToggle}
	aria-label={isDark ? 'Switch to light mode' : 'Switch to dark mode'}
>
	{#if isDark}
		<SunIcon size={20} />
	{:else}
		<MoonIcon size={20} />
	{/if}
</button>

<style>
	.theme-toggle {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 40px;
		height: 40px;
		border-radius: 50%;
		border: none;
		background-color: var(--secondary, #f9fafb);
		color: var(--foreground, #1f2937);
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.theme-toggle:hover {
		background-color: var(--accent, #f3f4f6);
	}

	:global(.dark) .theme-toggle {
		background-color: var(--secondary, #1a1a1a);
		color: var(--foreground, #f9fafb);
	}

	:global(.dark) .theme-toggle:hover {
		background-color: var(--accent, #2a2a2a);
	}
</style>
