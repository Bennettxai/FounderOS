<script lang="ts">
	import {
		HelpCircle,
		BarChart3,
		Database,
		Plug,
		Library,
		Bell,
		Keyboard,
		LifeBuoy,
		Mail,
		ExternalLink
	} from 'lucide-svelte';

	const modules = [
		{
			icon: BarChart3,
			title: 'Analytics',
			desc: 'Live counts and 30-day trends across tasks, projects, clients, offers, content, and campaigns. Read-only — nothing is changed here.'
		},
		{
			icon: Database,
			title: 'Data',
			desc: 'An inventory of every data entity in your workspace and how many records each holds. Export is coming soon; no destructive actions.'
		},
		{
			icon: Plug,
			title: 'Connectors',
			desc: 'Browse available integrations (CRMs, calendars, comms) and see which are connected. Connect or manage them from Settings > Integrations.'
		},
		{
			icon: Library,
			title: 'Resources',
			desc: 'A shared library of SOPs, guides, templates, and links. Add, edit, search, and delete resources so your team can find them fast.'
		},
		{
			icon: Bell,
			title: 'Notifications',
			desc: 'Your activity feed. Filter by unread, mark items read, or dismiss them. Delivery preferences live in Settings.'
		}
	];

	const shortcuts = [
		{ keys: ['⌘', 'K'], label: 'Open command palette / search' },
		{ keys: ['⌘', '/'], label: 'Toggle this help' },
		{ keys: ['G', 'then', 'H'], label: 'Go to Home / Dashboard' },
		{ keys: ['Esc'], label: 'Close a dialog or modal' }
	];

	const support = [
		{ icon: Mail, label: 'Email support', desc: 'Reach the team directly', href: 'mailto:support@businessos.dev' },
		{ icon: LifeBuoy, label: 'Documentation', desc: 'Guides and reference for every module', href: 'https://docs.businessos.dev' }
	];
</script>

<svelte:head><title>Help Center - BusinessOS</title></svelte:head>

<div class="page">
	<header class="page-header">
		<div class="page-icon"><HelpCircle size={22} strokeWidth={1.8} /></div>
		<div class="head-text">
			<h1 class="page-title">Help Center</h1>
			<p class="page-desc">How the Manage modules work, keyboard basics, and where to get support.</p>
		</div>
	</header>

	<section>
		<h2 class="sec-title">What each module does</h2>
		<div class="cards">
			{#each modules as m}
				{@const Ico = m.icon}
				<div class="card">
					<div class="card-icon"><Ico size={17} strokeWidth={1.8} /></div>
					<div class="card-body">
						<span class="card-title">{m.title}</span>
						<p class="card-desc">{m.desc}</p>
					</div>
				</div>
			{/each}
		</div>
	</section>

	<section>
		<h2 class="sec-title"><Keyboard size={15} /> Keyboard basics</h2>
		<ul class="shortcuts">
			{#each shortcuts as s}
				<li class="shortcut">
					<span class="shortcut-label">{s.label}</span>
					<span class="keys">
						{#each s.keys as k}
							{#if k === 'then'}<span class="then">then</span>{:else}<kbd>{k}</kbd>{/if}
						{/each}
					</span>
				</li>
			{/each}
		</ul>
		<p class="hint">Shortcuts may vary by platform; on Windows/Linux use Ctrl in place of ⌘.</p>
	</section>

	<section>
		<h2 class="sec-title"><LifeBuoy size={15} /> Get support</h2>
		<ul class="link-list">
			{#each support as item}
				{@const Ico = item.icon}
				<li>
					<a href={item.href} target="_blank" rel="noopener noreferrer" class="link-card">
						<div class="link-icon"><Ico size={16} strokeWidth={1.8} /></div>
						<div class="link-card-body">
							<span class="link-label">{item.label}</span>
							<span class="link-desc">{item.desc}</span>
						</div>
						<ExternalLink size={15} strokeWidth={1.8} class="link-external" />
					</a>
				</li>
			{/each}
		</ul>
	</section>
</div>

<style>
	.page { display: flex; flex-direction: column; gap: 26px; padding: 28px 32px; height: 100%; overflow-y: auto; }
	.page-header { display: flex; align-items: flex-start; gap: 14px; }
	.page-icon { width: 42px; height: 42px; border-radius: 10px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.head-text { flex: 1; min-width: 0; }
	.page-title { font-size: 1.25rem; font-weight: 650; color: var(--dt); letter-spacing: -0.02em; margin: 0; }
	.page-desc { font-size: 0.875rem; color: var(--dt3); margin: 2px 0 0; }

	section { display: flex; flex-direction: column; gap: 12px; }
	.sec-title { display: flex; align-items: center; gap: 7px; font-size: 0.82rem; font-weight: 650; text-transform: uppercase; letter-spacing: 0.05em; color: var(--dt2); margin: 0; }

	.cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 12px; }
	.card { display: flex; gap: 12px; border: 1px solid var(--dbd); border-radius: 12px; padding: 14px 16px; background: color-mix(in srgb, var(--dt) 2%, transparent); }
	.card-icon { width: 34px; height: 34px; border-radius: 8px; background: color-mix(in srgb, var(--dt) 6%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.card-body { display: flex; flex-direction: column; gap: 3px; }
	.card-title { font-size: 0.9rem; font-weight: 600; color: var(--dt); }
	.card-desc { font-size: 0.82rem; color: var(--dt3); margin: 0; line-height: 1.5; }

	.shortcuts { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; border: 1px solid var(--dbd); border-radius: 12px; overflow: hidden; max-width: 560px; }
	.shortcut { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 11px 16px; border-bottom: 1px solid var(--dbd); }
	.shortcut:last-child { border-bottom: none; }
	.shortcut-label { font-size: 0.85rem; color: var(--dt2); }
	.keys { display: flex; align-items: center; gap: 5px; }
	kbd { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 0.74rem; padding: 2px 7px; border-radius: 5px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 5%, transparent); color: var(--dt2); min-width: 20px; text-align: center; }
	.then { font-size: 0.72rem; color: var(--dt3); }
	.hint { font-size: 0.78rem; color: var(--dt3); margin: 0; }

	.link-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; max-width: 560px; }
	.link-card { display: flex; align-items: center; gap: 12px; padding: 14px 16px; border-radius: 10px; border: 1px solid var(--dbd); background: color-mix(in srgb, var(--dt) 3%, transparent); text-decoration: none; transition: background 150ms ease, border-color 150ms ease; }
	.link-card:hover { background: color-mix(in srgb, var(--dt) 7%, transparent); border-color: color-mix(in srgb, var(--dt) 20%, transparent); }
	.link-icon { width: 32px; height: 32px; border-radius: 8px; background: color-mix(in srgb, var(--dt) 6%, transparent); color: var(--dt2); display: flex; align-items: center; justify-content: center; flex-shrink: 0; }
	.link-card-body { display: flex; flex-direction: column; gap: 2px; flex: 1; }
	.link-label { font-size: 0.88rem; font-weight: 600; color: var(--dt); }
	.link-desc { font-size: 0.8rem; color: var(--dt3); }
	.link-card :global(.link-external) { color: var(--dt4, var(--dt3)); flex-shrink: 0; }

	@media (max-width: 768px) { .page { padding: 16px 18px; } .cards { grid-template-columns: 1fr; } }
	@media (max-width: 480px) { .page { padding: 12px 14px; } }
</style>
