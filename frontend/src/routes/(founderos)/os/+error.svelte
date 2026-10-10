<!-- The /os not-found and error page: Next.js's default not-found as FounderOS v1
     shows it, inside the chrome: "404 | This page could not be found."
     centred, the status and the message split by a hairline. -->
<script lang="ts">
	import { page } from '$app/state';

	const message = $derived(page.status === 404 ? 'This page could not be found.' : page.error?.message || 'Something went wrong.');
</script>

<svelte:head>
	<title>{page.status}: {message}</title>
</svelte:head>

<div data-part="not-found" class="flex min-h-[calc(100vh-52px-68px)] flex-col items-center justify-center text-center">
	<div class="flex items-center">
		<h1 class="bn-nf-rule mr-5 inline-block border-r pr-[23px] align-top text-[24px] font-medium leading-[49px]">{page.status}</h1>
		<div class="inline-block">
			<h2 class="m-0 text-[14px] font-normal leading-[49px]">{message}</h2>
		</div>
	</div>
</div>

<style>
	[data-part='not-found'] {
		font-family:
			system-ui,
			'Segoe UI',
			Roboto,
			Helvetica,
			Arial,
			sans-serif;
		color: var(--bn-text);
	}
	.bn-nf-rule {
		border-color: color-mix(in oklab, var(--bn-text) 30%, transparent);
	}
</style>
