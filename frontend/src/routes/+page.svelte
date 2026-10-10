<!-- Founder OS opens on its operator console. /os checks the session and
     sends a signed-out visitor to /login?next=/os. BusinessOS's marketing
     landing page stays at /landing. -->
<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';

	onMount(() => {
		goto('/os', { replaceState: true });
	});
</script>

<svelte:head>
	<title>FOUNDER OS</title>
</svelte:head>
