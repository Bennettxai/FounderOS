import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { ConnectionsBoard } from './types';

vi.mock('$lib/founderos/api', async (orig) => {
	const real = await orig<typeof import('$lib/founderos/api')>();
	return { ...real, founderosFetch: vi.fn() };
});

import { founderosFetch, FounderosApiError } from '$lib/founderos/api';
import Page from '../../../../routes/(founderos)/os/integrations/+page.svelte';

const fetchMock = vi.mocked(founderosFetch);

const tile = (slug: string, name: string, category: string, over: Partial<ConnectionsBoard['catalog'][number]> = {}) => ({
	slug,
	name,
	tagline: `${name} tagline`,
	category,
	connected: false,
	keySaved: false,
	keys: [`${slug.toUpperCase()}_API_KEY`],
	...over
});

function board(over: Partial<ConnectionsBoard> = {}): ConnectionsBoard {
	return {
		connections: [
			{ id: 'slack', name: 'Slack', kind: 'slack', state: 'connected', detail: 'ok' },
			{ id: 'fathom', name: 'Fathom', kind: 'crm', state: 'not_configured', detail: 'FATHOM_API_KEY missing' },
			{ id: 'zernio', name: 'Zernio', kind: 'social', state: 'error', detail: 'HTTP 500' },
			{ id: 'whatsapp', name: 'WhatsApp', kind: 'social', state: 'not_configured', detail: 'no collector has pushed WhatsApp yet' }
		],
		catalog: [
			tile('slack', 'Slack', 'Communication', { connectorId: 'slack', connected: true, keySaved: true, popular: true, keys: ['SLACK_BOT_TOKEN'] }),
			tile('whatsapp', 'WhatsApp', 'Communication', { connectorId: 'whatsapp', keys: [] }),
			tile('fathom', 'Fathom', 'CRM & Sales', { connectorId: 'fathom', popular: true, keys: ['FATHOM_API_KEY'] }),
			tile('zernio', 'Zernio', 'Marketing', { connectorId: 'zernio', keySaved: true, popular: true, keys: ['ZERNIO_API_KEY'] }),
			tile('github', 'GitHub', 'Developer', { popular: true })
		],
		categories: [
			{ name: 'Communication', slugs: ['slack', 'whatsapp'] },
			{ name: 'CRM & Sales', slugs: ['fathom'] },
			{ name: 'Developer', slugs: ['github'] },
			{ name: 'Marketing', slugs: ['zernio'] }
		],
		volume: {
			headline: 1,
			counts: { connected: 1, notConfigured: 2, error: 1, total: 4 },
			chips: [{ tone: 'ok', text: '1 live' }, { text: '2 not configured' }, { tone: 'err', text: '1 erroring' }],
			caption: 'of 4 connector checks · live status, never a stored key alone',
			meters: [{ label: 'Connectors live (1/4)', frac: 0.25, display: '25%', hue: 'var(--bn-ok)' }],
			foot: '5 tools · 4 categories · 1 saved key not live',
			byCategory: [{ label: 'Comm', count: 1 }, { label: 'CRM', count: 0 }],
			topCategory: { name: 'Communication', count: 1 },
			health: [{ label: 'live', count: 1 }, { label: 'unset', count: 2 }, { label: 'error', count: 1 }],
			insight: { value: 1, headline: '1 connector erroring.', body: 'Zernio', frac: 0.25 }
		},
		keys: [
			{ envVar: 'SLACK_BOT_TOKEN', label: 'Slack bot token', group: 'Slack', connectorId: 'slack', present: true },
			{ envVar: 'FATHOM_API_KEY', label: 'Fathom API key', group: 'CRM', connectorId: 'fathom', present: false },
			{ envVar: 'CLAUDE_OAUTH_TOKEN', label: 'Claude plan token', group: 'Usage', present: true }
		],
		oauth: {
			github: {
				slug: 'github', name: 'GitHub', redirectKind: 'any', consoleUrl: 'https://github.com/settings/developers',
				clientIdEnv: 'GITHUB_OAUTH_CLIENT_ID', clientSecretEnv: 'GITHUB_OAUTH_CLIENT_SECRET',
				appConfigured: false, connected: false, expired: false
			},
			slack: {
				slug: 'slack', name: 'Slack', redirectKind: 'https-public', consoleUrl: 'https://api.slack.com/apps',
				clientIdEnv: 'SLACK_OAUTH_CLIENT_ID', clientSecretEnv: 'SLACK_OAUTH_CLIENT_SECRET',
				appConfigured: false, connected: false, expired: false
			}
		},
		...over
	};
}

beforeEach(() => {
	fetchMock.mockReset();
});
afterEach(() => vi.restoreAllMocks());

async function loaded(b = board()) {
	fetchMock.mockImplementation(async (path: string) => {
		if (path === '/pages/connections') return b as never;
		return { ok: true } as never;
	});
	const r = render(Page);
	await waitFor(() => expect(r.container.querySelector('[data-part="volume"]')).toBeTruthy());
	return r;
}

describe('/os/integrations: title row', () => {
	it('reads GET /pages/connections and titles the board', async () => {
		await loaded();
		expect(fetchMock).toHaveBeenCalledWith('/pages/connections');
		expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Connections');
		expect(screen.getByText(/5 tools · 4 connector checks/)).toBeTruthy();
		expect(screen.getByText('1 live · 1 erroring')).toBeTruthy();
		expect(screen.getByRole('link', { name: 'API keys' }).getAttribute('href')).toBe('#api-keys');
	});
});

describe('/os/integrations: v1 layout (app/integrations/page.tsx)', () => {
	it('hero row: Your connected tools (only the live tiles) beside Connection Volume', async () => {
		const { container } = await loaded();
		const hero = container.querySelector('[data-part="connected-tools"]') as HTMLElement;
		expect(within(hero).getByText('Your connected tools')).toBeTruthy();
		expect(within(hero).getByText('1 of 5')).toBeTruthy();
		expect(within(hero).getByText('catalog tools whose connector answered on this load')).toBeTruthy();
		expect([...hero.querySelectorAll<HTMLElement>('[data-slug]')].map((t) => t.dataset.slug)).toEqual(['slack']);
		const vol = container.querySelector('[data-part="volume"]') as HTMLElement;
		expect(within(vol).getByText('Connection Volume')).toBeTruthy();
		expect(within(vol).getByText('of 4 connector checks · live status, never a stored key alone')).toBeTruthy();
		expect(within(vol).getByText('Connectors live (1/4)')).toBeTruthy();
		expect(within(vol).getByText('5 tools · 4 categories · 1 saved key not live')).toBeTruthy();
	});

	it('nothing live reads as v1’s empty state', async () => {
		const b = board();
		b.catalog = b.catalog.map((c) => ({ ...c, connected: false }));
		const { container } = await loaded(b);
		const hero = container.querySelector('[data-part="connected-tools"]') as HTMLElement;
		expect(within(hero).getByText('Nothing is connected yet. Pick a tool below and connect it.')).toBeTruthy();
	});

	it('second row: By Category, Connector Health and exactly one Needs-you insight', async () => {
		const { container } = await loaded();
		const cat = container.querySelector('[data-part="by-category"]') as HTMLElement;
		expect(within(cat).getByText('By Category')).toBeTruthy();
		expect(within(cat).getByText('connected tools')).toBeTruthy();
		expect(within(cat).getByText('Communication')).toBeTruthy();
		expect(within(cat).getByText('most connected · 1 live')).toBeTruthy();
		const health = container.querySelector('[data-part="health"]') as HTMLElement;
		expect(within(health).getByText('Connector Health')).toBeTruthy();
		expect(within(health).getByText('this load')).toBeTruthy();
		expect(within(health).getByText('connector checks, live · unset · error')).toBeTruthy();
		expect(container.querySelectorAll('.bn-insight').length).toBe(1);
		expect(screen.getByText('1 connector erroring.')).toBeTruthy();
	});

	it('Popular: the popular tools in catalog order', async () => {
		const { container } = await loaded();
		const pop = container.querySelector('[data-part="popular"]') as HTMLElement;
		expect(within(pop).getByText('Popular')).toBeTruthy();
		expect(within(pop).getByText('4')).toBeTruthy();
		expect([...pop.querySelectorAll<HTMLElement>('[data-slug]')].map((t) => t.dataset.slug)).toEqual(['slack', 'fathom', 'zernio', 'github']);
	});

	it('Browse by category: All plus a pill per category; the first category opens, a pill narrows to that one', async () => {
		const { container } = await loaded();
		const browse = container.querySelector('[data-part="browse"]') as HTMLElement;
		expect(within(browse).getByText('Browse by category')).toBeTruthy();
		expect(within(browse).getByRole('button', { name: 'All · 5' }).getAttribute('aria-pressed')).toBe('true');
		const toggles = [...browse.querySelectorAll<HTMLElement>('[data-part="category-toggle"]')];
		expect(toggles.map((t) => t.getAttribute('aria-expanded'))).toEqual(['true', 'false', 'false', 'false']);
		const open = () => [...browse.querySelectorAll<HTMLElement>('[data-slug]')].map((t) => t.dataset.slug);
		expect(open()).toEqual(['slack', 'whatsapp']);
		await fireEvent.click(within(browse).getByRole('button', { name: 'Developer · 1' }));
		expect(browse.querySelectorAll('[data-part="category-toggle"]').length).toBe(1);
		expect(open()).toEqual(['github']);
		await fireEvent.click(browse.querySelector('[data-part="category-toggle"]') as HTMLElement);
		expect(open()).toEqual([]);
	});

	it('a tile is v1’s row: brand logo beside the name and tagline, the status footer under them', async () => {
		const { container } = await loaded();
		const slack = container.querySelector('[data-slug="slack"]') as HTMLElement;
		expect(slack.querySelector('[data-mark="handmade"]')).toBeTruthy();
		expect(within(slack).getByText('Slack tagline')).toBeTruthy();
		expect(slack.className).toContain('min-h-[112px]');
		// v1 shows the OAuth strip on the tile itself, above the status row
		expect(within(slack).getByText(/public https redirect/)).toBeTruthy();
	});
});

describe('/os/integrations: tiles and connect flow', () => {
	it('a connected tile carries the status in its border; a saved key reads saved, not connected', async () => {
		const { container } = await loaded();
		const slack = container.querySelector('[data-slug="slack"]') as HTMLElement;
		const zernio = container.querySelector('[data-slug="zernio"]') as HTMLElement;
		expect(slack.hasAttribute('data-connected')).toBe(true);
		expect(within(slack).getByText('Connected')).toBeTruthy();
		expect(zernio.hasAttribute('data-connected')).toBe(false);
		expect(within(zernio).getByText('Key saved')).toBeTruthy();
	});

	it('a disconnect that cannot remove the key everywhere says where it still resolves', async () => {
		const { container } = await loaded();
		const reason = 'still resolves: ZERNIO_API_KEY from env.local; remove it there';
		const base = fetchMock.getMockImplementation()!;
		fetchMock.mockImplementation(async (path: string, init?: { method?: string }) => {
			if (init?.method === 'DELETE') throw new FounderosApiError(409, reason, { ok: false, stillResolves: ['ZERNIO_API_KEY'], error: reason });
			return base(path, init as never);
		});
		const zernio = container.querySelector('[data-slug="zernio"]') as HTMLElement;
		await fireEvent.click(within(zernio).getByRole('button', { name: 'Disconnect' }));
		await waitFor(() => expect(within(zernio).getByText(/still resolves: ZERNIO_API_KEY from env.local/)).toBeTruthy());
	});

	it('guidance-only tiles show Setup with the live detail, not a form', async () => {
		const { container } = await loaded();
		const wa = container.querySelector('[data-slug="whatsapp"]') as HTMLElement;
		const setup = within(wa).getByText('Setup');
		expect(setup.getAttribute('title')).toBe('no collector has pushed WhatsApp yet');
	});

	it('Connect opens a field per key and saves through /pages/connections/connect, then reloads', async () => {
		const { container } = await loaded();
		const fathom = container.querySelector('[data-slug="fathom"]') as HTMLElement;
		await fireEvent.click(within(fathom).getByRole('button', { name: '+ Connect' }));
		const input = within(fathom).getByPlaceholderText('FATHOM_API_KEY') as HTMLInputElement;
		expect(input.type).toBe('password');
		await fireEvent.input(input, { target: { value: 'fth-1' } });
		await fireEvent.click(within(fathom).getByRole('button', { name: /Save & connect/ }));
		await waitFor(() =>
			expect(fetchMock).toHaveBeenCalledWith('/pages/connections/connect', {
				method: 'POST',
				json: { slug: 'fathom', values: { FATHOM_API_KEY: 'fth-1' } }
			})
		);
		await waitFor(() => expect(fetchMock.mock.calls.filter((c) => c[0] === '/pages/connections').length).toBe(2));
	});

	it('an empty field never posts', async () => {
		const { container } = await loaded();
		const fathom = container.querySelector('[data-slug="fathom"]') as HTMLElement;
		await fireEvent.click(within(fathom).getByRole('button', { name: '+ Connect' }));
		await fireEvent.click(within(fathom).getByRole('button', { name: /Save & connect/ }));
		expect(within(fathom).getByText('every field is required')).toBeTruthy();
		expect(fetchMock.mock.calls.some((c) => c[0] === '/pages/connections/connect')).toBe(false);
	});

	it('the OAuth strip sits on the tile above the key row; a public-https provider gets no button', async () => {
		const b = board();
		b.oauth = { ...b.oauth, fathom: { ...b.oauth.slack, slug: 'fathom', name: 'Fathom', consoleUrl: 'https://fathom.video' } };
		const { container } = await loaded(b);
		const fathom = container.querySelector('[data-slug="fathom"]') as HTMLElement;
		expect(within(fathom).getByText(/public https redirect/)).toBeTruthy();
		expect(fathom.querySelector('a[href*="/oauth/"]')).toBeNull();
		const github = container.querySelector('[data-slug="github"]') as HTMLElement;
		expect(within(github).getByText(/Register an app at/)).toBeTruthy();
		expect(within(github).getByText('GITHUB_OAUTH_CLIENT_ID')).toBeTruthy();
	});
});

describe('/os/integrations: API keys', () => {
	it('says where v2 keeps the keys: ~/.founderos/.env, never v1’s .env.local or the private bridge', async () => {
		const { container } = await loaded();
		const keys = container.querySelector('#api-keys') as HTMLElement;
		expect(keys.textContent).toContain('Stored in ~/.founderos/.env, applied live. Values shown masked — the OS never echoes a secret back.');
		expect(keys.textContent).not.toMatch(/bridge|env\.local/);
	});

	it('lists slots by group, set or not set, never a value', async () => {
		const { container } = await loaded();
		const keys = container.querySelector('#api-keys') as HTMLElement;
		expect(within(keys).getByText('Slack')).toBeTruthy();
		const slackRow = keys.querySelector('[data-env="SLACK_BOT_TOKEN"]') as HTMLElement;
		expect(within(slackRow).getByText('••••••••')).toBeTruthy();
		const fathomRow = keys.querySelector('[data-env="FATHOM_API_KEY"]') as HTMLElement;
		expect(within(fathomRow).getByText('not set')).toBeTruthy();
		expect(within(fathomRow).queryByRole('button', { name: 'test' })).toBeNull();
		const claudeRow = keys.querySelector('[data-env="CLAUDE_OAUTH_TOKEN"]') as HTMLElement;
		expect(within(claudeRow).queryByRole('button', { name: 'test' })).toBeNull();
	});

	it('test runs the real connector check and shows the elapsed ms', async () => {
		const { container } = await loaded();
		fetchMock.mockImplementation(async (path: string) =>
			(path === '/pages/admin/keys/test' ? { ok: true, state: 'connected', detail: 'ok', ms: 212 } : board()) as never
		);
		const row = container.querySelector('[data-env="SLACK_BOT_TOKEN"]') as HTMLElement;
		await fireEvent.click(within(row).getByRole('button', { name: 'test' }));
		await waitFor(() => expect(within(row).getByText('✓ 212ms')).toBeTruthy());
		expect(fetchMock).toHaveBeenCalledWith('/pages/admin/keys/test', { method: 'POST', json: { envVar: 'SLACK_BOT_TOKEN' } });
	});

	it('a failing test stays red with the far end’s answer', async () => {
		const { container } = await loaded();
		fetchMock.mockImplementation(async (path: string) =>
			(path === '/pages/admin/keys/test' ? { ok: false, state: 'error', detail: 'HTTP 401', ms: 90 } : board()) as never
		);
		const row = container.querySelector('[data-env="SLACK_BOT_TOKEN"]') as HTMLElement;
		await fireEvent.click(within(row).getByRole('button', { name: 'test' }));
		await waitFor(() => expect(within(row).getByText('✗ failed')).toBeTruthy());
		expect(within(container.querySelector('#api-keys') as HTMLElement).getByText(/SLACK_BOT_TOKEN: HTTP 401/)).toBeTruthy();
	});

	it('set saves a slot through the connect route', async () => {
		const { container } = await loaded();
		const row = container.querySelector('[data-env="FATHOM_API_KEY"]') as HTMLElement;
		await fireEvent.click(within(row).getByRole('button', { name: 'set' }));
		const input = within(row).getByPlaceholderText('paste value') as HTMLInputElement;
		await fireEvent.input(input, { target: { value: 'k-1' } });
		await fireEvent.click(within(row).getByRole('button', { name: 'Save' }));
		await waitFor(() =>
			expect(fetchMock).toHaveBeenCalledWith('/pages/connections/connect', { method: 'POST', json: { slot: 'FATHOM_API_KEY', value: 'k-1' } })
		);
	});
});

describe('/os/integrations: production look (round 2)', () => {
	it('the API keys header action is the slab pill, rounded like production', async () => {
		await loaded();
		const a = screen.getByRole('link', { name: 'API keys' });
		expect(a.className).toContain('bn-pill');
		expect(a.className).toContain('rounded-full');
	});

	it('a tile is a lens row on the 12px tile radius', async () => {
		const { container } = await loaded();
		const slack = container.querySelector('[data-slug="slack"]') as HTMLElement;
		expect(slack.dataset.lens).toBe('r');
		expect(slack.className).toContain('bn-pressable');
		expect(slack.className).toContain('is-row');
		expect(slack.className).toContain('rounded-[var(--bn-r-tile)]');
	});

	it('Managed reads in the text colour, as production renders it', async () => {
		const b = board();
		b.catalog[0] = { ...b.catalog[0], keySaved: false };
		const { container } = await loaded(b);
		const managed = within(container.querySelector('[data-slug="slack"]') as HTMLElement).getByText('Managed');
		expect(managed.className).toContain('bn-text');
		expect(managed.className).not.toContain('bn-dim');
	});

	it('the connect panel: a rounded field, a pill Cancel and the primary pressable Save & connect', async () => {
		const { container } = await loaded();
		const fathom = container.querySelector('[data-slug="fathom"]') as HTMLElement;
		await fireEvent.click(within(fathom).getByRole('button', { name: '+ Connect' }));
		const input = within(fathom).getByPlaceholderText('FATHOM_API_KEY');
		expect(input.className).toContain('rounded-[var(--bn-r-ctl)]');
		const cancel = within(fathom).getByRole('button', { name: 'Cancel' });
		expect(cancel.className).toContain('rounded-full');
		expect(cancel.className).toContain('bn-pressable');
		const save = within(fathom).getByRole('button', { name: /Save & connect/ });
		expect(save.dataset.tone).toBe('primary');
	});

	it('+ Connect and Setup are lens controls', async () => {
		const { container } = await loaded();
		const connect = within(container.querySelector('[data-slug="fathom"]') as HTMLElement).getByRole('button', { name: '+ Connect' });
		expect(connect.className).toContain('bn-pressable');
		expect(connect.dataset.lens).toBe('c');
	});

	it('API keys: the section sits 40px below the card top, each group a rounded lens panel', async () => {
		const { container } = await loaded();
		const keys = container.querySelector('#api-keys') as HTMLElement;
		expect((keys.querySelector('section') as HTMLElement).className).toContain('mt-10');
		const group = keys.querySelector('[data-group="Slack"]') as HTMLElement;
		expect(group.className).toContain('rounded-[var(--bn-r-panel)]');
		expect(group.className).toContain('is-row');
		expect(group.dataset.lens).toBe('r');
	});

	it('API keys: a set slot shows the blank mask with reveal; reveal shows only the masked tail the server sent', async () => {
		const b = board();
		b.keys = [
			{ envVar: 'SLACK_BOT_TOKEN', label: 'Slack bot token', group: 'Slack', connectorId: 'slack', present: true, masked: '••••ab12' },
			{ envVar: 'CLAUDE_OAUTH_TOKEN', label: 'Claude plan token', group: 'Usage', present: true },
			{ envVar: 'FATHOM_API_KEY', label: 'Fathom API key', group: 'CRM', connectorId: 'fathom', present: false }
		];
		const { container } = await loaded(b);
		const row = container.querySelector('[data-env="SLACK_BOT_TOKEN"]') as HTMLElement;
		expect(within(row).getByText('••••••••')).toBeTruthy();
		await fireEvent.click(within(row).getByRole('button', { name: 'reveal' }));
		expect(within(row).getByText('••••ab12')).toBeTruthy();
		expect(within(row).getByRole('button', { name: 'hide' })).toBeTruthy();
		// an older bridge sends no tail: reveal says only that it is set
		const claude = container.querySelector('[data-env="CLAUDE_OAUTH_TOKEN"]') as HTMLElement;
		await fireEvent.click(within(claude).getByRole('button', { name: 'reveal' }));
		expect(within(claude).getByText('set')).toBeTruthy();
		const fathom = container.querySelector('[data-env="FATHOM_API_KEY"]') as HTMLElement;
		expect(within(fathom).queryByRole('button', { name: 'reveal' })).toBeNull();
		expect(within(fathom).getByText('not set')).toBeTruthy();
	});

	it('API keys: test is the secondary pressable and the dots are round', async () => {
		const { container } = await loaded();
		const row = container.querySelector('[data-env="SLACK_BOT_TOKEN"]') as HTMLElement;
		expect(within(row).getByRole('button', { name: 'test' }).dataset.tone).toBe('secondary');
		expect((row.querySelector('.bn-slot-dot') as HTMLElement).className).toContain('rounded-full');
	});
});

describe('/os/integrations: honest states', () => {
	it('loading reads as checking, not zero', async () => {
		let release: (b: ConnectionsBoard) => void = () => {};
		fetchMock.mockImplementation(() => new Promise((r) => (release = r as never)) as never);
		const { container } = render(Page);
		expect(container.textContent).toContain('checking connectors');
		expect(container.textContent).not.toMatch(/\b0 live\b/);
		release(board());
		await waitFor(() => expect(container.querySelector('[data-part="volume"]')).toBeTruthy());
	});

	it('an unreachable board is an error, never an empty board', async () => {
		fetchMock.mockImplementation(async () => {
			throw new FounderosApiError(503, 'backend down');
		});
		const { container } = render(Page);
		await waitFor(() => expect(container.textContent).toContain('connections unreachable'));
		expect(container.textContent).toContain('backend down');
		expect(container.textContent).not.toContain('Nothing is connected yet');
		expect(container.querySelector('[data-part="volume"]')).toBeNull();
	});
});

describe('/os/integrations: source contract', () => {
	const src = readFileSync(resolve(__dirname, '../../../../routes/(founderos)/os/integrations/+page.svelte'), 'utf8');
	it('composes the slab kit, not a placeholder', () => {
		expect(src).toContain('SlabTitle');
		expect(src).not.toMatch(/PortingPlaceholder|bridge:partial/);
	});
	it('the hero is the 2fr/1fr split and stagger indices run 1..8 as in v1', () => {
		expect(src).toContain('grid-cols-[2fr_1fr]');
		const loaded = src.slice(src.indexOf('<!-- Hero row'));
		const idx = [...loaded.matchAll(/\bi=\{(\d+)\}/g)].map((m) => m[1]);
		expect(idx).toEqual(['1', '2', '3', '4', '5', '6', '7', '8']);
	});
	it('By Category dots wear v1\'s --ramp-1 (brain-2), not the muted text colour', () => {
		expect(src).toContain('<DotMatrix cols={v.byCategory} hue="var(--bn-brain-2)" />');
	});
	it('no raw hex colours', () => {
		expect(src).not.toMatch(/#[0-9a-fA-F]{3,6}\b/);
	});
});
