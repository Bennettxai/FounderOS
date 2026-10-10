<!-- Legacy step removed. Redirect to wizard. -->
<script lang="ts">
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  onMount(() => goto('/onboarding', { replaceState: true }));
</script>
