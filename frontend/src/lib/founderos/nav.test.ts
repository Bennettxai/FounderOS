import { existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';
import { DIGIT_VIEWS, NAV_GROUPS, NAV_ITEMS, NAV_ORDER, isActive, labelFor } from './nav';

// Hard-coded from FounderOS v1 lib/nav.ts (the published demo, 2026-10-08), in
// visible order, with every href under /os and G-Brain renamed Brain: G-Brain is
// gone and the view shows the Optimal Engine.
const EXPECTED: Array<[group: string, label: string, href: string]> = [
	['Operate', 'Home', '/os'],
	['Operate', 'Comms', '/os/comms'],
	['Operate', 'Funnel', '/os/funnel'],
	['Operate', 'Workflows', '/os/workflows'],
	['Operate', 'Social', '/os/social'],
	['Operate', 'Content', '/os/content'],
	['Operate', 'Brand Deals', '/os/brand-deals'],
	['Operate', 'Finances', '/os/finances'],
	['Operate', 'Trading', '/os/trading'],
	['Operate', 'AdPilot', '/os/adpilot'],
	['Agents', 'Agents', '/os/agents'],
	['Agents', 'Chats', '/os/chats'],
	['Agents', 'Tasks', '/os/tasks'],
	['Agents', 'Skills', '/os/skills'],
	['Agents', 'Org Chart', '/os/org'],
	['Agents', 'Blueprint', '/os/blueprint'],
	['Intelligence', 'Brain', '/os/brain'],
	['Intelligence', 'Doctor', '/os/doctor'],
	['System', 'Connections', '/os/integrations'],
	['System', 'Usage', '/os/usage'],
	['System', 'Roadmap', '/os/roadmap'],
	['System', 'Analytics', '/os/analytics'],
	['System', 'Reference Model', '/os/reference'],
	['Variants', 'Personas', '/os/personas']
];

describe('FounderOS nav', () => {
	it('matches FounderOS v1: groups, order, labels, /os hrefs', () => {
		const flat = NAV_GROUPS.flatMap((g) => g.items.map((i) => [g.title, i.label, i.href]));
		expect(flat).toEqual(EXPECTED);
	});

	it('group titles are the sidebar headings, in order', () => {
		expect(NAV_GROUPS.map((g) => g.title)).toEqual(['Operate', 'Agents', 'Intelligence', 'System', 'Variants']);
	});

	it('G-Brain is gone: the knowledge view is Brain', () => {
		expect(NAV_ITEMS.some((i) => /g-?brain/i.test(i.label))).toBe(false);
		expect(labelFor('/os/brain')).toBe('Brain');
	});

	it('every item has an icon component', () => {
		for (const item of NAV_ITEMS) expect(item.icon, item.href).toBeTruthy();
	});

	it('NAV_ORDER is the visible order', () => {
		expect(NAV_ORDER).toEqual(EXPECTED.map(([, , href]) => href));
	});

	it('digits 1–9 map to the first nine visible items, like FounderOS v1', () => {
		expect(DIGIT_VIEWS).toEqual([
			'/os',
			'/os/comms',
			'/os/funnel',
			'/os/workflows',
			'/os/social',
			'/os/content',
			'/os/brand-deals',
			'/os/finances',
			'/os/trading'
		]);
	});

	it('active state: Home only on /os exactly; others also on their sub-routes', () => {
		expect(isActive('/os', '/os')).toBe(true);
		expect(isActive('/os', '/os/comms')).toBe(false);
		expect(isActive('/os/social', '/os/social')).toBe(true);
		expect(isActive('/os/social', '/os/social/instagram')).toBe(true);
		expect(isActive('/os/social', '/os/socials')).toBe(false);
	});

	it('every nav target has a page, so navigation never 404s', () => {
		const routes = resolve(__dirname, '../../routes/(founderos)');
		for (const href of NAV_ORDER) {
			expect(existsSync(`${routes}${href}/+page.svelte`), `${href} needs a +page.svelte`).toBe(true);
		}
	});
});
