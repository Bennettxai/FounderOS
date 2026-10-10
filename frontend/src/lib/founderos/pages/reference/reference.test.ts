import { render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import ReferencePage from '../../../../routes/(founderos)/os/reference/+page.svelte';
import DomainGrid from './DomainGrid.svelte';
import type { Domain } from './types';

const domains: Domain[] = [
	{ id: 'brm-1', number: 1, title: 'Command & Memory', color: '#fafafa', items: ['Optimal Engine', 'brain-store markdown'] },
	{ id: 'brm-8', number: 8, title: 'Security', color: '#525252', items: ['No keys in repo'] }
];

const json = (b: unknown, status = 200) =>
	new Response(JSON.stringify(b), { status, headers: { 'content-type': 'application/json' } });

describe('DomainGrid (v1 app/reference)', () => {
	it('one card per domain: zero-padded number, title, item rows', () => {
		const { container } = render(DomainGrid, { domains });
		expect(screen.getByText('01')).toBeTruthy();
		expect(screen.getByText('08')).toBeTruthy();
		expect(screen.getByRole('heading', { level: 2, name: 'Command & Memory' })).toBeTruthy();
		expect(screen.getByText('brain-store markdown').tagName).toBe('LI');
		const cards = container.querySelectorAll('[data-domain]');
		expect(cards).toHaveLength(2);
		// v1 radius scale: cards 10 (rounded-lg-t), item rows 5 (rounded-sm-t)
		expect(cards[0].className).toContain('rounded-[var(--bn-r-panel)]');
		expect(cards[0].querySelector('li')!.className).toContain('rounded-[var(--bn-r-chip)]');
		expect(container.firstElementChild!.className).toContain('xl:grid-cols-4');
	});
});

describe('/os/reference page', () => {
	let fetchMock: ReturnType<typeof vi.fn>;
	beforeEach(() => {
		fetchMock = vi.fn();
		vi.stubGlobal('fetch', fetchMock);
	});
	afterEach(() => vi.unstubAllGlobals());

	it('reads /pages/reference: operating domains eyebrow, Reference Model title', async () => {
		fetchMock.mockResolvedValue(json({ domains }));
		render(ReferencePage);
		expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Reference Model');
		expect(screen.getByText('operating domains')).toBeTruthy();
		await waitFor(() => expect(screen.getByText('Security')).toBeTruthy());
		expect(fetchMock.mock.calls[0][0]).toBe('/api/founderos/pages/reference');
	});

	it('an unreachable source says so', async () => {
		fetchMock.mockResolvedValue(json({ error: 'reference model unavailable: no Postgres' }, 503));
		const { container } = render(ReferencePage);
		await waitFor(() => expect(container.textContent).toContain('reference model unavailable: no Postgres'));
	});

	it('an empty table reads empty', async () => {
		fetchMock.mockResolvedValue(json({ domains: [] }));
		const { container } = render(ReferencePage);
		await waitFor(() => expect(container.textContent).toContain('No domains yet'));
	});
});
