<script lang="ts">
	import { User, Loader2 } from 'lucide-svelte';
	import { getUserAvatarUrl, useSession } from '$lib/auth-client';
	import { updateProfileName } from '$lib/api/workspace-admin';

	const session = useSession();

	let name = $state('');
	let initialized = $state(false);
	let saving = $state(false);
	let saved = $state(false);
	let error = $state<string | null>(null);

	// Seed the name field from the session once it resolves.
	$effect(() => {
		const u = $session.data?.user;
		if (u && !initialized) {
			name = u.name ?? '';
			initialized = true;
		}
	});

	const email = $derived($session.data?.user?.email ?? '');
	const image = $derived($session.data?.user?.image ?? '');
	const avatarSource = $derived(getUserAvatarUrl(image));
	let avatarError = $state(false);

	$effect(() => {
		avatarSource;
		avatarError = false;
	});

	async function save() {
		saving = true;
		error = null;
		saved = false;
		try {
			await updateProfileName(name.trim());
			saved = true;
			setTimeout(() => (saved = false), 2500);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to save';
		} finally {
			saving = false;
		}
	}
</script>

<header class="sec-head">
	<div class="sec-icon"><User size={20} strokeWidth={1.8} /></div>
	<div>
		<h2>Profile</h2>
		<p>Your personal account across every workspace.</p>
	</div>
</header>

{#if error}<div class="banner banner--error">{error}</div>{/if}

<div class="card">
	<div class="identity">
		{#if avatarSource && !avatarError}
			<img src={avatarSource} alt="" class="avatar" onerror={() => (avatarError = true)} />
		{:else}
			<div class="avatar avatar--fallback">{(name || email || 'U').charAt(0).toUpperCase()}</div>
		{/if}
		<div>
			<div class="identity-name">{name || 'Your name'}</div>
			<div class="identity-email">{email}</div>
		</div>
	</div>

	<label class="field">
		<span class="field-label">Display name</span>
		<input type="text" bind:value={name} placeholder="Your name" />
	</label>

	<label class="field">
		<span class="field-label">Email</span>
		<input type="email" value={email} disabled />
		<span class="field-hint">Email is tied to your login and can't be changed here.</span>
	</label>

	<div class="actions">
		<button class="btn btn--primary" onclick={save} disabled={saving || !name.trim()}>
			{#if saving}<Loader2 class="spin" size={15} />{/if}
			{saved ? 'Saved' : 'Save'}
		</button>
	</div>
</div>

<style>
	.sec-head { display: flex; gap: 14px; align-items: flex-start; margin-bottom: 22px; }
	.sec-icon { width: 40px; height: 40px; border-radius: 11px; flex-shrink: 0; display: flex; align-items: center; justify-content: center; background: color-mix(in srgb, var(--dt) 8%, transparent); color: var(--dt); }
	.sec-head h2 { margin: 0; font-size: 1.15rem; font-weight: 660; letter-spacing: -0.02em; }
	.sec-head p { margin: 4px 0 0; font-size: 0.84rem; color: var(--dt3); }
	.card { border: 1px solid var(--dbd); border-radius: 14px; padding: 20px; display: flex; flex-direction: column; gap: 18px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.identity { display: flex; align-items: center; gap: 14px; }
	.avatar { width: 52px; height: 52px; border-radius: 50%; object-fit: cover; }
	.avatar--fallback { display: flex; align-items: center; justify-content: center; background: linear-gradient(135deg, #6366f1, #8b5cf6); color: #fff; font-size: 1.2rem; font-weight: 650; }
	.identity-name { font-size: 0.98rem; font-weight: 620; }
	.identity-email { font-size: 0.8rem; color: var(--dt3); }
	.field { display: flex; flex-direction: column; gap: 6px; }
	.field-label { font-size: 0.82rem; font-weight: 580; }
	.field-hint { font-size: 0.74rem; color: var(--dt3); }
	.field input { width: 100%; padding: 9px 12px; border-radius: 9px; border: 1px solid var(--dbd); background: var(--dbg); color: var(--dt); font-size: 0.86rem; outline: none; }
	.field input:focus { border-color: color-mix(in srgb, var(--dt) 40%, transparent); }
	.field input:disabled { opacity: 0.6; }
	.actions { display: flex; justify-content: flex-end; }
	.btn { display: inline-flex; align-items: center; gap: 7px; padding: 9px 16px; border-radius: 9px; font-size: 0.84rem; font-weight: 560; cursor: pointer; border: 1px solid transparent; }
	.btn--primary { background: var(--dt); color: var(--dbg); }
	.btn:disabled { opacity: 0.55; cursor: not-allowed; }
	.banner { padding: 11px 14px; border-radius: 10px; font-size: 0.83rem; margin-bottom: 16px; }
	.banner--error { background: color-mix(in srgb, #ef4444 12%, transparent); color: #ef4444; }
	:global(.spin) { animation: spin 0.9s linear infinite; }
	@keyframes spin { to { transform: rotate(360deg); } }

	@media (max-width: 768px) {
		.card { padding: 16px; }
		.actions .btn { width: 100%; justify-content: center; min-height: 40px; }
		.actions { justify-content: stretch; }
	}

	@media (max-width: 480px) {
		.field input { font-size: 1rem; /* prevent iOS zoom */ min-height: 40px; }
	}
</style>
