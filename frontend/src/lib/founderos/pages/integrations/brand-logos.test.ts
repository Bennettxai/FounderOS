import { render } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import BrandLogo from './BrandLogo.svelte';
import { brandMark, brandMarkKind } from './brand-logos';

// FounderOS v1 lib/integrations-catalog.ts slugs (G-Brain's tile is the Optimal Engine here).
const CATALOG = [
	'slack', 'gmail', 'whatsapp', 'discord', 'telegram', 'zoom', 'manychat',
	'airtable', 'googlesheets', 'googledocs', 'clickup', 'trello', 'coda', 'wispr',
	'hubspot', 'salesforce', 'zendesk', 'intercom',
	'github', 'linear', 'jira', 'vercel', 'sentry', 'gitlab',
	'googlecalendar', 'calendly', 'caldotcom', 'googlemeet',
	'stripe', 'stripe-vantage', 'quickbooks', 'xero', 'paypal', 'wise', 'plaid', 'robinhood', 'paykit', 'phantom',
	'mailchimp', 'googleanalytics', 'meta', 'beehiiv', 'buffer', 'hootsuite', 'zernio', 'skool', 'trakyo', 'fathom', 'plaud', 'docusign',
	'googledrive', 'dropbox', 'box', 'onedrive', 'obsidian',
	'openai', 'anthropic', 'zapier', 'make', 'n8n', 'optimal-engine', 'paperclip', 'ollama',
	'figma', 'canva', 'miro', 'loom', 'typeform', 'vidalytics', 'arcads'
];

describe('brand marks (FounderOS v1 lib/brand-logos.tsx)', () => {
	it('every catalog slug resolves to a mark; a typo does not', () => {
		for (const slug of CATALOG) expect(brandMarkKind(slug), slug).not.toBe('none');
		expect(brandMarkKind('not-a-brand')).toBe('none');
	});

	it('picks the same source prod does: handmade, vector, icon, raster, then lettermark', () => {
		expect(brandMarkKind('slack')).toBe('handmade');
		expect(brandMarkKind('openai')).toBe('vector');
		expect(brandMarkKind('zernio')).toBe('vector');
		expect(brandMarkKind('gmail')).toBe('icon');
		expect(brandMarkKind('stripe-vantage')).toBe('icon'); // wears the parent's mark
		expect(brandMarkKind('beehiiv')).toBe('raster');
		expect(brandMarkKind('docusign')).toBe('raster');
		for (const s of ['trakyo', 'wispr', 'plaud', 'optimal-engine']) expect(brandMarkKind(s)).toBe('lettermark');
	});

	it('an icon keeps its brand hex; a dark brand is lightened on the dark tile', () => {
		const gmail = brandMark('gmail', 'Gmail');
		expect(gmail.kind).toBe('icon');
		if (gmail.kind === 'icon') expect(gmail.fill.toLowerCase()).toBe('#ea4335');
		const github = brandMark('github', 'GitHub');
		if (github.kind === 'icon') expect(github.fill).toBe('#e6e7ea');
	});

	it('a lettermark is the tinted initial', () => {
		const t = brandMark('trakyo', 'Trakyo');
		expect(t).toMatchObject({ kind: 'lettermark', initial: 'T', color: '#EAB308' });
	});
});

describe('BrandLogo', () => {
	it('renders the kind it resolved, sized like prod (radius 28%, glyph 56%)', () => {
		const { container } = render(BrandLogo, { slug: 'slack', name: 'Slack', size: 34 });
		const tile = container.querySelector('[data-mark]') as HTMLElement;
		expect(tile.dataset.mark).toBe('handmade');
		expect(tile.getAttribute('style')).toContain('width: 34px');
		expect(tile.querySelector('svg')?.getAttribute('width')).toBe('19');
	});

	it('a raster mark loads the vendored PNG', () => {
		const { container } = render(BrandLogo, { slug: 'beehiiv', name: 'beehiiv', size: 34 });
		expect(container.querySelector('img')?.getAttribute('src')).toBe('/logos/integrations/beehiiv.png');
	});

	it('a lettermark shows the initial', () => {
		const { container } = render(BrandLogo, { slug: 'plaud', name: 'Plaud', size: 34 });
		expect(container.textContent?.trim()).toBe('P');
	});
});
