<script lang="ts">
	import { goto } from '$app/navigation';
	import { useSession } from '$lib/auth-client';

	let { children } = $props();

	const session = useSession();

	$effect(() => {
		if (!$session.isPending && !$session.data) {
			goto('/login');
		}
	});
</script>

{#if $session.isPending}
	<div class="min-h-screen flex items-center justify-center bg-white">
		<div class="animate-spin h-8 w-8 border-2 border-gray-900 border-t-transparent rounded-full"></div>
	</div>
{:else if $session.data}
	<div class="h-screen w-screen overflow-hidden bg-white">
		{@render children()}
	</div>
{/if}
