<!-- /os/content/lead-magnets (spec 6.14): the status pills are ?status= links. -->
<script lang="ts">
	import { page } from '$app/state';
	import LeadMagnetsView from '$lib/founderos/pages/content/LeadMagnetsView.svelte';
</script>

<LeadMagnetsView status={page.url.searchParams.get('status')} />
