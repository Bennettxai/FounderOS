<!--
  Onboarding wizard — 5 steps, single page.
  Steps:
    0  Welcome / your name
    1  Create your organization
    2  Create your first workspace
    3  How it works
    4  Done
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { getSession } from '$lib/auth-client';
  import { wizardStore } from '$lib/stores/onboardingStore';
  import { onboardingStore } from '$lib/stores/onboardingStore';
  import {
    createOrganization,
    listMyOrganizations,
  } from '$lib/api/organizations';
  import { createWorkspace, getWorkspaces } from '$lib/api/workspaces';
  import { currentWorkspace } from '$lib/stores/workspaces';
  import { switchWorkspace } from '$lib/stores/workspaces';

  // -------------------------------------------------------------------------
  // State
  // -------------------------------------------------------------------------

  const TOTAL_STEPS = 5;

  let w = $derived($wizardStore);

  // per-step local form state
  let nameInput = $state('');
  let orgInput = $state('');
  let wsInput = $state('');

  let loading = $state(false);
  let error = $state('');

  // -------------------------------------------------------------------------
  // Bootstrap: pre-fill name from session; detect existing org/workspace
  // -------------------------------------------------------------------------

  onMount(async () => {
    const session = await getSession();
    const sessionName = session.data?.user?.name ?? '';

    // If wizard already has a stored name, use that; else seed from session
    if (!w.displayName && sessionName) {
      wizardStore.patch({ displayName: sessionName });
      nameInput = sessionName;
    } else {
      nameInput = w.displayName;
    }

    orgInput = w.orgName;
    wsInput = w.workspaceName;

    // If org already exists (e.g. auto-provisioned at signup), skip step 1
    if (!w.orgId) {
      try {
        const orgs = await listMyOrganizations();
        if (orgs.length > 0) {
          wizardStore.patch({ orgId: orgs[0].id, orgName: orgs[0].name });
          orgInput = orgs[0].name;
          // If also has workspace, skip to how-it-works
          if (w.step < 2) {
            const wss = await getWorkspaces();
            if (wss.length > 0) {
              wizardStore.patch({
                workspaceId: wss[0].id,
                workspaceName: wss[0].name,
                step: 3,
              });
            } else if (w.step < 2) {
              wizardStore.patch({ step: 2 });
            }
          }
        }
      } catch {
        // non-fatal — user just proceeds normally
      }
    }
  });

  // -------------------------------------------------------------------------
  // Navigation helpers
  // -------------------------------------------------------------------------

  function back() {
    error = '';
    const prev = Math.max(0, w.step - 1);
    wizardStore.patch({ step: prev });
  }

  // -------------------------------------------------------------------------
  // Step handlers
  // -------------------------------------------------------------------------

  async function submitName() {
    const name = nameInput.trim();
    if (!name) {
      error = 'Please enter your name.';
      return;
    }
    error = '';
    wizardStore.patch({ displayName: name, step: 1 });
  }

  async function submitOrg() {
    const name = orgInput.trim();
    if (!name) {
      error = 'Please enter your organization name.';
      return;
    }
    error = '';
    loading = true;

    try {
      // If already has an org (pre-detected), skip creation
      if (w.orgId) {
        wizardStore.patch({ orgName: name, step: 2 });
        return;
      }

      const org = await createOrganization({ name });
      wizardStore.patch({ orgId: org.id, orgName: org.name, step: 2 });
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to create organization. Please try again.';
    } finally {
      loading = false;
    }
  }

  async function submitWorkspace() {
    const name = wsInput.trim();
    if (!name) {
      error = 'Please name your workspace.';
      return;
    }
    error = '';
    loading = true;

    try {
      // If already has a workspace, skip creation
      if (w.workspaceId) {
        wizardStore.patch({ workspaceName: name, step: 3 });
        return;
      }

      const wsData: import('$lib/api/workspaces').CreateWorkspaceData = { name };
      if (w.orgId) wsData.organization_id = w.orgId;
      const ws = await createWorkspace(wsData);
      wizardStore.patch({ workspaceId: ws.id, workspaceName: ws.name, step: 3 });
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to create workspace. Please try again.';
    } finally {
      loading = false;
    }
  }

  function toHowItWorks() {
    error = '';
    wizardStore.patch({ step: 4 });
  }

  async function finish() {
    loading = true;
    error = '';

    try {
      // Activate the workspace
      if (w.workspaceId) {
        await switchWorkspace(w.workspaceId);
      }

      // Mark onboarding complete on backend
      await onboardingStore.complete();

      // Clear wizard state so a future re-visit does not replay
      wizardStore.reset();
      onboardingStore.reset();

      goto('/');
    } catch (e) {
      error = e instanceof Error ? e.message : 'Something went wrong. Please try again.';
      loading = false;
    }
  }

  // -------------------------------------------------------------------------
  // Key handler — Enter submits
  // -------------------------------------------------------------------------

  function handleKeydown(e: KeyboardEvent, submit: () => void) {
    if (e.key === 'Enter') submit();
  }
</script>

<svelte:head>
  <title>Get started - BusinessOS</title>
</svelte:head>

<div class="wizard-shell">

  <!-- Progress bar -->
  <div class="progress-bar" role="progressbar" aria-valuenow={w.step + 1} aria-valuemin={1} aria-valuemax={TOTAL_STEPS}>
    {#each Array(TOTAL_STEPS) as _, i}
      <div class="progress-dot" class:active={i === w.step} class:done={i < w.step}></div>
    {/each}
  </div>

  <!-- Step panels -->
  <div class="panel">

    <!-- ------------------------------------------------------------------ -->
    <!-- Step 0: Welcome / name -->
    <!-- ------------------------------------------------------------------ -->
    {#if w.step === 0}
      <div class="step" role="main">
        <p class="eyebrow">Welcome</p>
        <h1 class="heading">What should we call you?</h1>
        <p class="sub">This is how you appear to your team.</p>

        <div class="field-group">
          <input
            class="text-input"
            type="text"
            placeholder="Your name"
            bind:value={nameInput}
            onkeydown={(e) => handleKeydown(e, submitName)}
            aria-label="Your name"
            autofocus
          />
          {#if error}<p class="error-msg" role="alert">{error}</p>{/if}
        </div>

        <div class="actions">
          <button class="btn-primary" onclick={submitName}>
            Continue
          </button>
        </div>
      </div>

    <!-- ------------------------------------------------------------------ -->
    <!-- Step 1: Create organization -->
    <!-- ------------------------------------------------------------------ -->
    {:else if w.step === 1}
      <div class="step" role="main">
        <p class="eyebrow">Step 1 of 3</p>
        <h1 class="heading">Name your organization</h1>
        <p class="sub">An organization is your whole account - the company group you invite people to.</p>

        <div class="field-group">
          {#if w.orgId}
            <div class="prefilled-note">
              Organization already set up: <strong>{w.orgName}</strong>
            </div>
          {:else}
            <input
              class="text-input"
              type="text"
              placeholder="e.g. Acme Corp"
              bind:value={orgInput}
              onkeydown={(e) => handleKeydown(e, submitOrg)}
              aria-label="Organization name"
              autofocus
              disabled={loading}
            />
          {/if}
          {#if error}<p class="error-msg" role="alert">{error}</p>{/if}
        </div>

        <div class="actions">
          <button class="btn-primary" onclick={submitOrg} disabled={loading}>
            {#if loading}Creating...{:else}Continue{/if}
          </button>
          <button class="btn-ghost" onclick={back} disabled={loading}>Back</button>
        </div>
      </div>

    <!-- ------------------------------------------------------------------ -->
    <!-- Step 2: Create workspace -->
    <!-- ------------------------------------------------------------------ -->
    {:else if w.step === 2}
      <div class="step" role="main">
        <p class="eyebrow">Step 2 of 3</p>
        <h1 class="heading">Create your first workspace</h1>
        <p class="sub">A workspace is a work area - a company, client, brand, or department. Org members get added to specific workspaces.</p>

        <div class="field-group">
          {#if w.workspaceId}
            <div class="prefilled-note">
              Workspace already exists: <strong>{w.workspaceName}</strong>
            </div>
          {:else}
            <input
              class="text-input"
              type="text"
              placeholder="e.g. Main Business"
              bind:value={wsInput}
              onkeydown={(e) => handleKeydown(e, submitWorkspace)}
              aria-label="Workspace name"
              autofocus
              disabled={loading}
            />
          {/if}
          {#if error}<p class="error-msg" role="alert">{error}</p>{/if}
        </div>

        <div class="actions">
          <button class="btn-primary" onclick={submitWorkspace} disabled={loading}>
            {#if loading}Creating...{:else}Continue{/if}
          </button>
          <button class="btn-ghost" onclick={back} disabled={loading}>Back</button>
        </div>
      </div>

    <!-- ------------------------------------------------------------------ -->
    <!-- Step 3: How it works -->
    <!-- ------------------------------------------------------------------ -->
    {:else if w.step === 3}
      <div class="step" role="main">
        <p class="eyebrow">Step 3 of 3</p>
        <h1 class="heading">How it works</h1>

        <ul class="how-list">
          <li>
            <span class="how-icon" aria-hidden="true">&#9632;</span>
            <div>
              <strong>Workspaces</strong> are your work areas - one per company, client, or team. Each has its own data and members.
            </div>
          </li>
          <li>
            <span class="how-icon" aria-hidden="true">&#9632;</span>
            <div>
              <strong>Modules</strong> (Tasks, Projects, Clients...) can be configured per workspace or shared across your organization.
            </div>
          </li>
          <li>
            <span class="how-icon" aria-hidden="true">&#9632;</span>
            <div>
              <strong>Invite your team</strong> to the organization first, then add them to the workspaces they need.
            </div>
          </li>
        </ul>

        <div class="actions">
          <button class="btn-primary" onclick={toHowItWorks}>
            Got it
          </button>
          <button class="btn-ghost" onclick={back}>Back</button>
        </div>
      </div>

    <!-- ------------------------------------------------------------------ -->
    <!-- Step 4: Done -->
    <!-- ------------------------------------------------------------------ -->
    {:else if w.step === 4}
      <div class="step" role="main">
        <div class="done-icon" aria-hidden="true">&#10003;</div>
        <h1 class="heading">You're all set, {w.displayName || 'there'}.</h1>
        <p class="sub">
          Your organization <strong>{w.orgName}</strong> and workspace <strong>{w.workspaceName}</strong> are ready.
        </p>

        {#if error}<p class="error-msg" role="alert">{error}</p>{/if}

        <div class="actions">
          <button class="btn-primary" onclick={finish} disabled={loading}>
            {#if loading}Opening...{:else}Open BusinessOS{/if}
          </button>
        </div>
      </div>
    {/if}

  </div>
</div>

<style>
  /* Shell */
  .wizard-shell {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 2rem 1rem;
    background-color: var(--dbg);
    gap: 2rem;
  }

  /* Progress dots */
  .progress-bar {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }

  .progress-dot {
    width: 0.5rem;
    height: 0.5rem;
    border-radius: 50%;
    background-color: var(--dbd);
    transition: background-color 0.2s ease, transform 0.2s ease;
  }

  .progress-dot.active {
    background-color: var(--dt);
    transform: scale(1.25);
  }

  .progress-dot.done {
    background-color: var(--dt3);
  }

  /* Panel card */
  .panel {
    width: 100%;
    max-width: 480px;
    background-color: var(--dbg2);
    border: 1px solid var(--dbd);
    border-radius: 1rem;
    padding: 2.5rem 2rem;
  }

  /* Step layout */
  .step {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  /* Typography */
  .eyebrow {
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    color: var(--dt3);
    margin: 0;
  }

  .heading {
    font-size: 1.75rem;
    font-weight: 700;
    line-height: 1.2;
    letter-spacing: -0.02em;
    color: var(--dt);
    margin: 0;
  }

  .sub {
    font-size: 0.9375rem;
    color: var(--dt2);
    line-height: 1.6;
    margin: 0;
  }

  /* Field group */
  .field-group {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .text-input {
    width: 100%;
    padding: 0.75rem 1rem;
    font-size: 1rem;
    font-family: inherit;
    background-color: var(--dbg3);
    border: 1px solid var(--dbd);
    border-radius: 0.625rem;
    color: var(--dt);
    outline: none;
    transition: border-color 0.15s ease;
    box-sizing: border-box;
  }

  .text-input:focus {
    border-color: var(--dt3);
  }

  .text-input:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .text-input::placeholder {
    color: var(--dt3);
  }

  .error-msg {
    font-size: 0.875rem;
    color: #ef4444;
    margin: 0;
  }

  .prefilled-note {
    font-size: 0.9375rem;
    color: var(--dt2);
    padding: 0.75rem 1rem;
    background-color: var(--dbg3);
    border: 1px solid var(--dbd);
    border-radius: 0.625rem;
  }

  .prefilled-note strong {
    color: var(--dt);
  }

  /* Actions */
  .actions {
    display: flex;
    flex-direction: column;
    gap: 0.625rem;
  }

  .btn-primary {
    width: 100%;
    padding: 0.75rem 1.5rem;
    font-size: 0.9375rem;
    font-weight: 600;
    font-family: inherit;
    background-color: var(--dt);
    color: var(--dbg);
    border: none;
    border-radius: 0.625rem;
    cursor: pointer;
    transition: opacity 0.15s ease;
  }

  .btn-primary:hover:not(:disabled) {
    opacity: 0.88;
  }

  .btn-primary:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  .btn-ghost {
    width: 100%;
    padding: 0.625rem 1.5rem;
    font-size: 0.875rem;
    font-weight: 500;
    font-family: inherit;
    background: transparent;
    color: var(--dt2);
    border: none;
    border-radius: 0.625rem;
    cursor: pointer;
    transition: color 0.15s ease;
  }

  .btn-ghost:hover:not(:disabled) {
    color: var(--dt);
  }

  .btn-ghost:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }

  /* How it works list */
  .how-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .how-list li {
    display: flex;
    gap: 0.875rem;
    align-items: flex-start;
    font-size: 0.9375rem;
    color: var(--dt2);
    line-height: 1.5;
  }

  .how-list li strong {
    color: var(--dt);
  }

  .how-icon {
    color: var(--dt3);
    font-size: 0.5rem;
    margin-top: 0.45rem;
    flex-shrink: 0;
  }

  /* Done screen */
  .done-icon {
    width: 3rem;
    height: 3rem;
    border-radius: 50%;
    background-color: var(--dbg3);
    border: 1px solid var(--dbd);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 1.25rem;
    color: var(--dt);
  }

  /* Mobile */
  @media (max-width: 640px) {
    .panel {
      padding: 2rem 1.25rem;
    }

    .heading {
      font-size: 1.5rem;
    }
  }

  @media (max-width: 480px) {
    .wizard-shell {
      padding: 1.25rem 0;
      justify-content: flex-start;
      padding-top: 1.5rem;
      gap: 1.25rem;
    }

    .panel {
      width: 100%;
      max-width: 100%;
      border-radius: 0.75rem;
      padding: 1.5rem 1rem;
      border-left: none;
      border-right: none;
      border-radius: 0;
    }

    .heading {
      font-size: 1.375rem;
    }

    .sub {
      font-size: 0.875rem;
    }

    .text-input {
      font-size: 1rem;
      padding: 0.875rem 1rem;
      /* prevent iOS zoom on focus */
    }

    .btn-primary {
      min-height: 3rem;
      font-size: 1rem;
    }

    .btn-ghost {
      min-height: 2.75rem;
    }

    .progress-bar {
      padding: 0 1rem;
    }

    .progress-dot {
      width: 0.625rem;
      height: 0.625rem;
    }

    .how-list li {
      font-size: 0.875rem;
    }

    .done-icon {
      width: 2.5rem;
      height: 2.5rem;
    }
  }

  @media (max-width: 320px) {
    .wizard-shell {
      padding-top: 1rem;
    }

    .panel {
      padding: 1.25rem 0.875rem;
    }
  }
</style>
