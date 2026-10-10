<!--
  Onboarding layout — auth guard only.
  The wizard itself lives in +page.svelte and manages its own step UI.
-->
<script lang="ts">
  import { onMount, type Snippet } from 'svelte';
  import { goto } from '$app/navigation';
  import { browser } from '$app/environment';
  import { getSession, checkOnboardingStatus } from '$lib/auth-client';

  interface Props {
    children: Snippet;
  }

  let { children }: Props = $props();

  let checking = $state(true);
  let authed = $state(false);

  onMount(async () => {
    if (!browser) return;

    const session = await getSession();
    if (!session.data?.user) {
      goto('/login');
      return;
    }

    authed = true;

    const status = await checkOnboardingStatus();
    if (!status.needsOnboarding) {
      goto('/window');
      return;
    }

    checking = false;
  });
</script>

{#if checking}
  <div class="loading-screen">
    <div class="spinner"></div>
  </div>
{:else if authed}
  {@render children()}
{/if}

<style>
  .loading-screen {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background-color: var(--dbg);
  }

  .spinner {
    width: 2rem;
    height: 2rem;
    border: 2px solid var(--dbd);
    border-top-color: var(--dt);
    border-radius: 50%;
    animation: spin 0.7s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }
</style>
