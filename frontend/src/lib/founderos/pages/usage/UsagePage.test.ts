import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { OllamaBoard, UsageBody } from './usage';
import { claude } from './fixtures';

vi.mock('$lib/founderos/api', () => ({
	founderosFetch: vi.fn(),
	founderosUrl: (p: string) => `/api/founderos${p}`
}));

import { founderosFetch } from '$lib/founderos/api';
import UsagePage from '../../../../routes/(founderos)/os/usage/+page.svelte';

const fetchMock = vi.mocked(founderosFetch);

const ollama: OllamaBoard = {
	state: 'up',
	plan: 'Ollama Pro',
	models: [{ name: 'gpt-oss:120b-cloud', cloud: true, host: 'Ollama · mini' }],
	requests: { hour: { chat: 1, embed: 2 }, session: { chat: 3, embed: 4 }, day: { chat: 5, embed: 7 }, week: { chat: 20, embed: 30 } },
	note: 'from the server log',
	machines: [{ id: 'ollama-mini', label: 'Ollama · mini', source: 'push', capturedAt: new Date().toISOString(), lastActivity: null, stale: false }]
};

const full: UsageBody = { generatedAt: new Date().toISOString(), claude, codex: null, ollama };
const empty: UsageBody = { generatedAt: new Date().toISOString(), claude: null, codex: null, ollama: null };

beforeEach(() => {
	fetchMock.mockReset();
});
afterEach(() => vi.useRealTimers());

describe('/os/usage', () => {
	it('opens with the slab title row and a raw-reading pill to the bridge endpoint', async () => {
		fetchMock.mockResolvedValue(full);
		render(UsagePage);
		expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Usage');
		expect(screen.getByText('token burn · one plan per provider', { exact: false })).toBeTruthy();
		const raw = screen.getByRole('link', { name: /raw reading/i });
		expect(raw.getAttribute('href')).toBe('/api/founderos/pages/usage');
		expect(fetchMock).toHaveBeenCalledWith('/pages/usage');
	});

	it('shows a loading line, then an honest failure when the read fails', async () => {
		let fail!: () => void;
		fetchMock.mockImplementation(() => new Promise((_, r) => (fail = () => r(new Error('backend down (HTTP 502)')))));
		render(UsagePage);
		expect(screen.getByText(/reading local transcripts/i)).toBeTruthy();
		await waitFor(() => expect(fetchMock).toHaveBeenCalled());
		fail();
		await waitFor(() => expect(screen.getByText('usage read failed: backend down (HTTP 502)')).toBeTruthy());
	});

	it('renders the hero, second row and plan cards from one reading', async () => {
		fetchMock.mockResolvedValue(full);
		render(UsagePage);
		await waitFor(() => expect(screen.getByText('Token Burn')).toBeTruthy());
		for (const t of ['Burn Volume', 'Burn by Model', 'Plan Limits', 'Plans']) expect(screen.getByText(t)).toBeTruthy();
		expect(screen.getByText('Claude Max 20x · active')).toBeTruthy();
		expect(screen.getByText('Ollama Pro · active')).toBeTruthy();
		// Claude's breakdown is open by default: where it burns + top burners
		expect(screen.getByText('where it burns')).toBeTruthy();
		expect(screen.getAllByText('Conductor').length).toBeGreaterThan(0);
		// ChatGPT/Codex was never pushed: v1's empty card, not zeros
		expect(screen.getByText('no sessions found on any reporting machine')).toBeTruthy();
	});

	it('clicking a plan card opens where its tokens went', async () => {
		fetchMock.mockResolvedValue(full);
		render(UsagePage);
		await waitFor(() => expect(screen.getByText('Ollama Pro · active')).toBeTruthy());
		await fireEvent.click(screen.getByText('Ollama Pro · active'));
		expect(screen.getByText('requests by window · chat vs embeddings')).toBeTruthy();
		expect(screen.getByText('cloud · bills plan')).toBeTruthy();
		expect(screen.queryByText('where it burns')).toBeNull();
	});

	it('the window tabs switch the breakdown window', async () => {
		fetchMock.mockResolvedValue(full);
		render(UsagePage);
		await waitFor(() => expect(screen.getByText('where it burns')).toBeTruthy());
		const week = screen.getByRole('tab', { name: '7 days' });
		await fireEvent.click(week);
		expect(week.getAttribute('aria-selected')).toBe('true');
		const detail = document.querySelector('[data-part="plan-detail"]') as HTMLElement;
		expect(within(detail).getByText('1.6M')).toBeTruthy();
	});

	it('nothing pushed reads unknown, never zero', async () => {
		fetchMock.mockResolvedValue({ ...empty, errors: { stored: 'no database: stored usage pushes unreadable' } });
		render(UsagePage);
		await waitFor(() => expect(screen.getByText('No machine has reported usage yet.')).toBeTruthy());
		expect(screen.getAllByText('no sessions found on any reporting machine').length).toBe(2);
		expect(screen.getByText('no machine has reported Ollama yet')).toBeTruthy();
		// v1 copy only: no private-build "unknown · not zero" footers
		expect(screen.queryByText(/not zero|not down/)).toBeNull();
		expect(screen.getByText(/stored: no database/)).toBeTruthy();
		expect(screen.queryByText('Nothing burned this week.')).toBeNull();
	});

	it('flags a stale pushed machine', async () => {
		const stale = { ...claude, machines: [{ ...claude.machines[0], stale: true, capturedAt: new Date(Date.now() - 2 * 86400_000).toISOString() }] };
		fetchMock.mockResolvedValue({ ...full, claude: stale });
		render(UsagePage);
		await waitFor(() => expect(screen.getByText(/· stale/)).toBeTruthy());
	});

	it('polls every 10s', async () => {
		vi.useFakeTimers();
		fetchMock.mockResolvedValue(full);
		render(UsagePage);
		expect(fetchMock).toHaveBeenCalledTimes(1);
		await vi.advanceTimersByTimeAsync(10_000);
		expect(fetchMock).toHaveBeenCalledTimes(2);
	});

	it('v1 layout: hero (burn line + Burn Volume), then models / limits / top burner, then the Plans', async () => {
		fetchMock.mockResolvedValue(full);
		const { container } = render(UsagePage);
		await waitFor(() => expect(screen.getByText('Token Burn')).toBeTruthy());
		expect(container.textContent).toContain(
			'Claude · ChatGPT / Codex · Ollama · local file parsing only · no paid calls · refreshes every 10s'
		);
		// Burn Volume: no subtitle, the official 5h chip, the lanes as one column of meters with the burn foot
		expect(screen.queryByText('cache reads not counted')).toBeNull();
		expect(screen.getByText('Claude 5h 42%')).toBeTruthy();
		expect(container.querySelector('[data-part="burn-lanes"]')).toBeNull();
		const meters = container.querySelector('[data-part="meters"]') as HTMLElement;
		expect(meters.className).not.toContain('grid-cols-2');
		expect(within(meters).getByText('Agent board')).toBeTruthy();
		expect(within(meters).getByText('Crons & headless')).toBeTruthy();
		expect(screen.getByText('burn = input + output + cache writes · cache reads not counted')).toBeTruthy();
		// section order: hero, then the second row, then Plans
		const text = container.textContent ?? '';
		const at = (t: string) => text.indexOf(t);
		expect(at('Token Burn')).toBeLessThan(at('Burn Volume'));
		expect(at('Burn Volume')).toBeLessThan(at('Burn by Model'));
		expect(at('Burn by Model')).toBeLessThan(at('Plan Limits'));
		expect(at('Plan Limits')).toBeLessThan(at('Top burner · 7 days'));
		expect(at('Top burner · 7 days')).toBeLessThan(at('Plans'));
		// Plans counts providers the v1 way: reported Claude/Codex plans + Ollama
		expect(screen.getByText('2 providers')).toBeTruthy();
		expect(container.textContent).toContain('refreshes every 10s · local file parsing only · no paid calls');
	});

	it('prod shapes: rounded pills and chips, 12px pressable plan cards, a hairline 12px detail, round bars and dots', async () => {
		fetchMock.mockResolvedValue(full);
		const { container } = render(UsagePage);
		await waitFor(() => expect(screen.getByText('where it burns')).toBeTruthy());
		expect(screen.getByRole('link', { name: /raw reading/i }).className).toContain('rounded-full');
		expect(screen.getByRole('button', { name: 'Ollama' }).className).toContain('rounded-full');
		expect(screen.getByRole('tab', { name: 'today' }).className).toContain('rounded-full');
		const cards = container.querySelectorAll('[data-part="plan-card"]');
		for (const c of cards) expect(c.className).toContain('rounded-[12px]');
		const claudeCard = cards[0] as HTMLElement;
		expect(claudeCard.className).toContain('bn-pressable');
		expect(claudeCard.className).toContain('is-row');
		// the hairline is a real border colour, not Tailwind's border-width reading of var()
		expect(claudeCard.className).not.toContain('border-[var(');
		const detail = container.querySelector('[data-part="plan-detail"]') as HTMLElement;
		expect(detail.className).toContain('rounded-[12px]');
		expect(detail.className).toContain('bn-border');
		for (const bar of container.querySelectorAll('[data-part="lane-bar"]')) expect(bar.className).toContain('rounded-full');
		for (const d of container.querySelectorAll('[data-part="legend-dot"]')) expect(d.className).toContain('rounded-full');
		expect(container.querySelector('[data-part="top-model"]')!.className).toContain('rounded-full');
	});

	it('the token-burn line is the kit StepLine at its default 150 high (v1)', async () => {
		fetchMock.mockResolvedValue(full);
		const { container } = render(UsagePage);
		await waitFor(() => expect(screen.getByText('Token Burn')).toBeTruthy());
		const svg = container.querySelector('[data-part="burn-line"] svg');
		expect(svg?.getAttribute('viewBox')).toBe('0 0 600 150');
	});

	it('a pushed official Claude gauge fills the plan card, the 5h chip and the Plan Limits rows', async () => {
		const claudeOfficial = {
			...claude,
			official: {
				session: { usedPercent: 37, windowMinutes: 300, resetsAt: new Date(Date.now() + 2 * 3600_000).toISOString() },
				weekly: { usedPercent: 64, windowMinutes: 10080, resetsAt: new Date(Date.now() + 3 * 86400_000).toISOString() }
			},
			note: 'burn measured from transcripts; limit % is the official gauge from this login'
		};
		fetchMock.mockResolvedValue({ ...full, claude: claudeOfficial });
		render(UsagePage);
		await waitFor(() => expect(screen.getByText('Claude 5h 37%')).toBeTruthy());
		expect(screen.getByText('Session limit · 5h')).toBeTruthy();
		expect(screen.getByText('Weekly limit · 7d')).toBeTruthy();
		expect(screen.getByText('Claude · 5h session')).toBeTruthy();
		expect(screen.getByText('Claude · weekly')).toBeTruthy();
		// the card's weekly meter and the Plan Limits row read the same gauge
		expect(screen.getAllByText('64% · resets in 3d').length).toBe(2);
		expect(screen.queryByText(/no official gauge reported/)).toBeNull();
		expect(screen.getByText('burn measured from transcripts; limit % is the official gauge from this login')).toBeTruthy();
	});

	it('without an official Claude gauge the card says so and Plan Limits stays honest', async () => {
		fetchMock.mockResolvedValue({ ...full, claude: { ...claude, official: null } });
		render(UsagePage);
		await waitFor(() => expect(screen.getByText(/no official gauge reported/)).toBeTruthy());
		expect(screen.queryByText(/^Claude 5h/)).toBeNull();
		expect(screen.queryByText('Claude · 5h session')).toBeNull();
		expect(screen.getByText('No provider reported an official gauge. Burn is measured from transcripts.')).toBeTruthy();
	});
});
