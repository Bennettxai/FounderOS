import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fireEvent, render, screen, waitFor, within } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { FinancesPayload } from './types';

const fetchMock = vi.fn();
vi.mock('$lib/founderos/api', () => ({ founderosFetch: (...a: unknown[]) => fetchMock(...a) }));

import FinancesPage from '../../../../routes/(founderos)/os/finances/+page.svelte';

const acct = (o: Partial<FinancesPayload['income']['accounts'][number]>) => ({
	id: 'stripe',
	processor: 'Stripe',
	label: 'Stripe · Launchpad Cohort',
	configured: true,
	live: true,
	income: 6000,
	incomeUpper: null,
	unsplittableCustomers: 0,
	...o
});

function payload(o: Partial<FinancesPayload> = {}): FinancesPayload {
	return {
		generatedAt: '2026-07-20T00:00:00Z',
		thisMonth: '2026-07',
		income: {
			stripe: {
				keyed: true,
				live: true,
				mtdUsd: 6000,
				availableUsd: 1234.5,
				pendingUsd: 10,
				recentCharges: [{ amount: 250000, currency: 'usd', description: 'Retainer · Acme', created: Math.floor(Date.now() / 1000) - 600 }]
			},
			processors: null,
			accounts: [
				acct({}),
				acct({ id: 'stripe-vantage', label: 'Stripe · Vantage', live: false, income: null }),
				acct({ id: 'paykit-lc', processor: 'PayKit', label: 'PayKit · Launchpad Cohort', live: true, income: 1000, incomeUpper: 1500, unsplittableCustomers: 2 })
			],
			totalUsd: 7000,
			totalUpperUsd: 7500,
			hasUnsplittable: true,
			liveCount: 2,
			wiseOutgoing: [{ amountCents: 50000, currency: 'USD', status: 'outgoing_payment_sent', created: '2026-07-10', reference: 'Editor July' }]
		},
		ledger: {
			rows: [
				{ date: '2026-06-03', description: 'AWS', amountCents: 10000, direction: 'out', category: 'Infrastructure', card: 'platinum' },
				{ date: '2026-07-03', description: 'ANTHROPIC', amountCents: 20000, direction: 'out', category: 'Software', card: 'platinum' },
				{ date: '2026-07-04', description: 'AWS', amountCents: 5000, direction: 'out', category: 'Infrastructure', card: 'blue' }
			],
			months: ['2026-06', '2026-07'],
			latestMonth: '2026-07',
			byCategory: [
				{ category: 'Software', total: 200 },
				{ category: 'Infrastructure', total: 50 }
			]
		},
		bank: { series: [{ business: 'Vantage', months: [{ month: '2026-04', creditsCents: 1250000, debitsCents: 425000, netCents: 825000 }] }] },
		cardLanes: [],
		fallback: [
			{ category: 'Contractors', totalCents: 100000 },
			{ category: 'Software', totalCents: 50000 }
		],
		expenses: 250,
		expensesLive: true,
		monthLabel: 'Jul 2026',
		statementIsThisMonth: true,
		netComparable: true,
		netMonthly: 6750,
		volume: {
			headline: 7000,
			upper: 7500,
			chips: [
				{ tone: 'err', text: '$250 out' },
				{ tone: 'ok', text: '+$6,750 net' }
			],
			caption: 'income this month · 2/3 processors live',
			meters: [
				{ label: 'Stripe · Launchpad Cohort', frac: 0.857, display: '$6,000', hue: 'var(--bn-text)' },
				{ label: 'Stripe · Vantage', frac: null, display: 'pull pending', hue: 'var(--bn-text-2)' },
				{ label: 'PayKit · Launchpad Cohort', frac: 0.143, display: '$1,000 – $1,500', hue: 'var(--bn-text-3)' },
				{ label: 'Spent of income · Jul 2026', frac: 0.04, display: '4%', hue: 'var(--bn-warn)' }
			],
			foot: 'expenses from the uploaded Jul 2026 statement',
			insight: { display: '+$6,750', headline: 'kept of $7,000 in this month.', body: '$7,000 in − $250 out.', frac: 0.96 }
		},
		spend: [
			{ label: 'Jun 2026', count: 100 },
			{ label: 'Jul 2026', count: 250 }
		],
		chargeSizes: [
			{ label: '<$100', count: 0 },
			{ label: '$100-500', count: 0 },
			{ label: '$500-2k', count: 0 },
			{ label: '$2k+', count: 1 }
		],
		largestChargeUsd: 2500,
		...o
	};
}

// braces: a function returned from beforeEach is run as its teardown
beforeEach(() => {
	fetchMock.mockReset();
});

describe('/os/finances', () => {
	it('shows a loading state, then the slab with every section of the FounderOS v1 page', async () => {
		fetchMock.mockResolvedValue(payload());
		render(FinancesPage);
		expect(screen.getByText(/loading finances/i)).toBeTruthy();
		await screen.findByRole('heading', { level: 1, name: 'Finances' });
		expect(fetchMock).toHaveBeenCalledWith('/pages/finances');

		for (const t of ['Money Volume', 'Where it goes', 'Spend by month', 'Charge sizes', 'Income · by business', 'Income · by processor', 'Recent income', 'Outgoing · Wise']) {
			expect(screen.getByText(t), t).toBeTruthy();
		}
		// meta line: band, statement month, processors live, Stripe balance
		expect(screen.getByText(/\$7,000 – \$7,500 in this month · \$250 out \(Jul 2026 statement\) · 2\/3 processors live · Stripe balance \$1,234\.50/)).toBeTruthy();
		expect(screen.getByText('+$6,750 net /mo')).toBeTruthy();
		expect(screen.getByText('Kept this month')).toBeTruthy();
		expect(screen.getByText('Retainer · Acme')).toBeTruthy();
		expect(screen.getByText('Editor July')).toBeTruthy();
		expect(screen.getByText('$2,500.00')).toBeTruthy(); // largest charge
	});

	it('processor rows: live, key set (pull pending), and a bounded month says so', async () => {
		fetchMock.mockResolvedValue(payload());
		render(FinancesPage);
		const section = (await screen.findByText('Income · by processor')).closest('.bn-card') as HTMLElement;
		const rows = within(section).getAllByTestId('processor-row');
		expect(rows).toHaveLength(3);
		expect(rows[1].textContent).toContain('—');
		expect(rows[1].textContent).toContain('pull pending');
		expect(rows[1].textContent).toContain('key set');
		expect(rows[2].textContent).toContain('2 repeat customers · split unavailable');
		expect(rows[2].textContent).toContain('– $1,500');
	});

	it('the expense panel steps months and redraws from the ledger rows', async () => {
		fetchMock.mockResolvedValue(payload());
		render(FinancesPage);
		const total = await screen.findByTestId('expenses-total');
		expect(total.textContent).toBe('$250 · Jul 2026');
		await fireEvent.click(screen.getByRole('button', { name: 'Previous month' }));
		expect(screen.getByTestId('expenses-total').textContent).toBe('$100 · Jun 2026');
		await fireEvent.click(screen.getByRole('button', { name: 'View full expenditure statement' }));
		expect(screen.getByText('// full expenditure statement')).toBeTruthy();
		await fireEvent.keyDown(window, { key: 'Escape' });
		await waitFor(() => expect(screen.queryByText('// full expenditure statement')).toBeNull());
	});

	it('is honest when Stripe is not live and nothing is uploaded: set fees, no fake balance', async () => {
		const p = payload({
			ledger: { rows: [], months: [], latestMonth: null, byCategory: [] },
			bank: { series: [] },
			expenses: 1500,
			expensesLive: false,
			monthLabel: null,
			largestChargeUsd: null,
			spend: []
		});
		p.income.stripe = { keyed: false, live: false, mtdUsd: null, availableUsd: 0, pendingUsd: 0, recentCharges: [] };
		p.income.wiseOutgoing = null;
		fetchMock.mockResolvedValue(p);
		render(FinancesPage);
		await screen.findByRole('heading', { level: 1, name: 'Finances' });
		expect(screen.getByText(/Stripe balance needs a live key/)).toBeTruthy();
		expect(screen.getByText(/\(set fees\)/)).toBeTruthy();
		expect(screen.getByText(/set fees · upload a card statement for real months/)).toBeTruthy();
		expect(screen.getByText('No card statements uploaded yet.')).toBeTruthy();
		expect(screen.queryByText('Recent income')).toBeNull();
		expect(screen.queryByText('Outgoing · Wise')).toBeNull();
		expect(screen.queryByText('Income · by business')).toBeNull();
		expect(screen.getByTestId('largest-charge').textContent).toBe(' - ');
		// v1: Spend by month and Charge sizes count zero, never a dash or "unknown"
		expect(screen.getByText('upload a card statement to chart spend')).toBeTruthy();
		expect(screen.getByText('recent charges')).toBeTruthy();
		expect(screen.queryByText(/charges unknown/)).toBeNull();
	});

	it('surfaces an unreachable ledger instead of reading it as empty', async () => {
		fetchMock.mockResolvedValue(payload({ ledger: { rows: [], months: [], latestMonth: null, byCategory: [], error: 'finances store: no database' } }));
		render(FinancesPage);
		expect(await screen.findByText(/ledger unreachable: finances store: no database/)).toBeTruthy();
	});

	it('no processor answering reads $0 in this month, as v1 does', async () => {
		const base = payload();
		const accounts = base.income.accounts.map((a) => ({ ...a, live: false, income: null, incomeUpper: null }));
		fetchMock.mockResolvedValue(payload({ income: { ...base.income, accounts } }));
		render(FinancesPage);
		expect(await screen.findByText(/^\$0 in this month · \$250 out/)).toBeTruthy();
		expect(screen.queryByText(/income unknown/)).toBeNull();
	});

	it('shows the error when the page payload cannot be read', async () => {
		fetchMock.mockImplementation(async () => {
			throw new Error('Bad gateway (HTTP 502)');
		});
		render(FinancesPage);
		expect(await screen.findByText(/finances unreachable: Bad gateway \(HTTP 502\)/)).toBeTruthy();
	});

	it('uploads a card statement under the chosen lane, then reloads the page data', async () => {
		fetchMock.mockImplementation(async (path: string) => {
			if (path === '/pages/finances/statements') return { inserted: 1, parsed: 1, card: 'gold', uploadedMonths: ['2026-07'], months: ['2026-06', '2026-07'] };
			return payload();
		});
		const { container } = render(FinancesPage);
		await screen.findByRole('heading', { level: 1, name: 'Finances' });
		await fireEvent.click(screen.getByRole('button', { name: 'Gold · Personal' }));
		const input = container.querySelector('input[type="file"]') as HTMLInputElement;
		const file = new File(['Date,Description,Amount\n07/15/2026,AWS,-57.00'], 'july.csv', { type: 'text/csv' });
		await fireEvent.change(input, { target: { files: [file] } });
		await screen.findByText('✓ 1 new of 1 parsed rows · Jul 2026');
		const call = fetchMock.mock.calls.find((c) => c[0] === '/pages/finances/statements')!;
		expect(call[1].method).toBe('POST');
		const form = call[1].body as FormData;
		expect(form.get('card')).toBe('gold');
		expect((form.get('file') as File).name).toBe('july.csv');
		expect(fetchMock.mock.calls.filter((c) => c[0] === '/pages/finances')).toHaveLength(2);
	});

	it('a bank statement goes to the bank route and reports the business month', async () => {
		fetchMock.mockImplementation(async (path: string) => {
			if (path === '/pages/finances/bank-statement') return { summary: { business: 'Vantage', month: '2026-04', creditsCents: 1250000 } };
			return payload();
		});
		const { container } = render(FinancesPage);
		await screen.findByRole('heading', { level: 1, name: 'Finances' });
		await fireEvent.click(screen.getByRole('button', { name: 'Bank statement · income' }));
		const input = container.querySelector('input[type="file"]') as HTMLInputElement;
		await fireEvent.change(input, { target: { files: [new File(['%PDF'], 'april.pdf', { type: 'application/pdf' })] } });
		await screen.findByText('✓ Vantage 2026-04: $12,500 in');
		const form = fetchMock.mock.calls.find((c) => c[0] === '/pages/finances/bank-statement')![1].body as FormData;
		expect(form.get('card')).toBeNull();
	});

	it('an upload failure reads as a failure', async () => {
		fetchMock.mockImplementation(async (path: string) => {
			if (path === '/pages/finances/statements') throw new Error('no parseable rows (HTTP 400)');
			return payload();
		});
		const { container } = render(FinancesPage);
		await screen.findByRole('heading', { level: 1, name: 'Finances' });
		const input = container.querySelector('input[type="file"]') as HTMLInputElement;
		await fireEvent.change(input, { target: { files: [new File(['x'], 'x.csv', { type: 'text/csv' })] } });
		expect(await screen.findByText('✗ no parseable rows (HTTP 400)')).toBeTruthy();
	});
});

// FounderOS v1 app/finances/page.tsx + MonthlyExpenses.tsx + StatementUploader.tsx:
// the demo's page shape, not the later private build's.
describe('/os/finances matches FounderOS v1', () => {
	const order = (a: Node, b: Node) => !!(a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING);

	it('Where it goes and Money Volume lead; Income · by business sits under the second row, above the processors', async () => {
		fetchMock.mockResolvedValue(payload());
		render(FinancesPage);
		const income = await screen.findByText('Income · by business');
		expect(screen.getByText('bank deposits')).toBeTruthy();
		const where = screen.getByText('Where it goes');
		expect(order(where, screen.getByText('Money Volume'))).toBe(true);
		expect(order(screen.getByText('Kept this month'), income)).toBe(true);
		expect(order(income, screen.getByText('Income · by processor'))).toBe(true);
	});

	it('month pills are the ledger months, with no statement checklist', async () => {
		fetchMock.mockResolvedValue(payload());
		const { container } = render(FinancesPage);
		const jun = await screen.findByRole('button', { name: 'Jun 2026' });
		expect(jun.getAttribute('data-incomplete')).toBeNull();
		expect(screen.getByRole('button', { name: 'Jul 2026' })).toBeTruthy();
		expect(container.querySelector('[data-part="statement-checklist"]')).toBeNull();
		expect(screen.queryByText(/Statements · /)).toBeNull();
	});

	it('an uncategorized month charts its categories, never a merchant split', async () => {
		const rows = [
			{ date: '2026-07-03', description: 'GOOGLE *GSUITE', amountCents: 11500, direction: 'out' as const, category: 'Uncategorized', card: 'blue' as const },
			{ date: '2026-07-04', description: 'DINER SEATTLE WA', amountCents: 4500, direction: 'out' as const, category: 'Uncategorized', card: 'blue' as const }
		];
		fetchMock.mockResolvedValue(payload({ ledger: { rows, months: ['2026-07'], latestMonth: '2026-07', byCategory: [{ category: 'Uncategorized', total: 160 }] } }));
		render(FinancesPage);
		await screen.findByText('Where it goes');
		expect(screen.getAllByText('Uncategorized').length).toBeGreaterThan(0);
		expect(screen.queryByText(/by merchant/)).toBeNull();
		expect(screen.queryByText('Google')).toBeNull();
	});

	it('with nothing uploaded the donut splits the declared set fees, with a share legend', async () => {
		const p = payload({ ledger: { rows: [], months: [], latestMonth: null, byCategory: [] }, expenses: 1500, expensesLive: false, monthLabel: null });
		fetchMock.mockResolvedValue(p);
		const { container } = render(FinancesPage);
		await screen.findByText('Where it goes');
		expect(screen.getByTestId('expenses-total').textContent).toBe('$1,500 /mo');
		const pie = container.querySelector('svg[aria-label="Monthly expenses by category"]') as SVGElement;
		expect(pie).toBeTruthy();
		expect(pie.querySelectorAll('path')).toHaveLength(2);
		expect(pie.textContent).toContain('per month');
		expect(container.textContent).toContain('66.7%');
		expect(container.textContent).toContain('33.3%');
		expect(screen.queryByText(/one charge line/)).toBeNull();
	});

	it('the uploader is v1\'s panel: Upload statements, four lanes, then the upload button', async () => {
		fetchMock.mockResolvedValue(payload());
		render(FinancesPage);
		const panel = await screen.findByRole('region', { name: 'Upload statements' });
		expect(within(panel).getByText('Upload statements')).toBeTruthy();
		const lanes = within(panel)
			.getAllByRole('button')
			.map((b) => b.textContent?.trim());
		expect(lanes).toEqual(['Gold · Personal', 'Platinum · Business', 'Business Blue · Vantage', 'Bank statement · income']);
		expect(panel.textContent).toContain('Drop a CSV or PDF here, or pick one below.');
		expect(panel.textContent).toContain('Card statement: categorized spend and subscriptions for this lane.');
		expect(panel.textContent).not.toMatch(/bridge/i);
		expect(order(screen.getByText('Where it goes'), panel)).toBe(true);
	});

	it('Income · by processor is a full-width card, not half a row', async () => {
		fetchMock.mockResolvedValue(payload());
		render(FinancesPage);
		const card = (await screen.findByText('Income · by processor')).closest('.bn-card') as HTMLElement;
		expect(card.parentElement?.className ?? '').not.toMatch(/grid-cols-2/);
	});

	it('a money band never breaks across lines: the headline unit and meter read as one', async () => {
		fetchMock.mockResolvedValue(payload());
		const { container } = render(FinancesPage);
		await screen.findByText('Money Volume');
		expect(container.textContent).toContain('–⁠ $7,500');
		expect(container.textContent).toContain('$1,000 –⁠ $1,500');
	});
});

// Port of tests/finances-slab.test.ts: /finances wears the Brand Deals slab,
// built from the kit, with every piece of the v1 page on it.
describe('/finances on the slab kit', () => {
	const view = readFileSync(resolve(__dirname, 'FinancesView.svelte'), 'utf8');
	const panel = readFileSync(resolve(__dirname, 'MonthlyExpenses.svelte'), 'utf8');

	it('floats as one slab with the kit title row', () => {
		expect(view).toMatch(/from '\$lib\/founderos\/kit'/);
		expect(view).toContain('<Slab>');
		expect(view).toContain('<SlabTitle');
		expect(view).toContain('title="Finances"');
		expect(view).not.toContain('<PageHeader');
	});

	it('Where it goes beside Money Volume, Income · by business after the second row', () => {
		expect(view).toMatch(/grid-cols-\[2fr_1fr\]/);
		expect(view).toContain('title="Money Volume"');
		expect(view).toContain('<BigStat');
		expect(view).toContain('<MeterStack');
		expect(view.indexOf('<MonthlyExpenses')).toBeLessThan(view.indexOf('title="Money Volume"'));
		expect(view.indexOf('<InsightCard')).toBeLessThan(view.indexOf('<BusinessIncomeChart'));
		expect(view.indexOf('<BusinessIncomeChart')).toBeLessThan(view.indexOf('Income · by processor'));
	});

	it('a second row: spend step line, charge-size dots, and exactly one insight card', () => {
		expect(view).toContain('<StepLine');
		expect(view).toContain('<DotMatrix');
		expect((view.match(/<InsightCard/g) ?? []).length).toBe(1);
	});

	it('keeps every piece of the v1 page, recent income above Wise', () => {
		for (const s of ['<StatementUploader', '<MonthlyExpenses', '<BusinessIncomeChart', 'Income · by processor', 'Recent income', 'Outgoing · Wise']) {
			expect(view, s).toContain(s);
		}
		expect(view.indexOf('Recent income')).toBeLessThan(view.indexOf('Outgoing · Wise'));
		expect(view).not.toContain('statementCoverage');
	});

	it('card stagger indices are distinct', () => {
		const idx = [...view.matchAll(/<(?:SlabCard|InsightCard)[^>]*\bi=\{(\d+)\}/g)].map((m) => Number(m[1]));
		expect(idx.length).toBeGreaterThanOrEqual(6);
		expect(new Set(idx).size).toBe(idx.length);
	});

	it('house rules: no raw hex, no transition-all/colors', () => {
		for (const src of [view, panel]) {
			expect(src).not.toMatch(/#[0-9a-fA-F]{3,6}\b/);
			expect(src).not.toMatch(/transition-(colors|all)\b/);
		}
	});

	it('the expense panel: slab card head, month chips, Deal Volume meters, the full statement', () => {
		expect(panel).toContain('text-[19px]');
		expect(panel).toContain('<VolumeMeter');
		expect(panel).toContain('categoryTotals');
		expect(panel).not.toContain('statement-checklist');
		expect(panel).toContain('<SharePie');
		expect(panel).toContain('<ExpenditureReport');
	});
});

describe('/os/finances shapes and meters', () => {
	it('Money Volume meters read their display, never "unknown": a pull pending says so, a live $0 reads $0', async () => {
		const p = payload();
		p.volume.meters = [
			{ label: 'Stripe · Launchpad Cohort', frac: null, display: '$0', hue: 'var(--bn-text)' },
			{ label: 'Stripe · Vantage', frac: null, display: 'pull pending', hue: 'var(--bn-text-2)' },
			{ label: 'Spent of income · Sep 2026', frac: null, display: 'no statement this month', hue: 'var(--bn-warn)' }
		];
		fetchMock.mockResolvedValue(p);
		const { container } = render(FinancesPage);
		await screen.findByText('Money Volume');
		const meters = container.querySelector('[data-part="meters"]') as HTMLElement;
		expect(meters.textContent).not.toContain('unknown');
		expect(meters.textContent).toContain('$0');
		expect(meters.textContent).toContain('pull pending');
		expect(meters.textContent).toContain('no statement this month');
		expect(meters.querySelectorAll('[data-unknown]').length).toBe(0);
	});

	it('header actions are the slab pills (rounded), Open Stripe the accent pill', async () => {
		fetchMock.mockResolvedValue(payload());
		render(FinancesPage);
		const up = await screen.findByRole('link', { name: /Upload statement/ });
		expect(up.className).toContain('rounded-full');
		expect(up.className).toContain('bn-pill');
		const stripe = screen.getByRole('link', { name: /Open Stripe/ });
		expect(stripe.className).toContain('bn-pill-accent');
		expect(stripe.className).toContain('rounded-full');
	});

	it("Income · by business toggles are the kit's 24px toggle pills, and switch the card", async () => {
		fetchMock.mockResolvedValue(payload());
		render(FinancesPage);
		const out = await screen.findByRole('button', { name: 'Money out' });
		expect(out.className).toContain('bn-toggle-chip');
		expect(out.className).toContain('rounded-full');
		expect(out.getAttribute('data-on')).toBe('false');
		await fireEvent.click(out);
		expect(out.getAttribute('data-on')).toBe('true');
		expect(screen.getByText('outflow · bank debits')).toBeTruthy();
		const all = screen.getByRole('button', { name: 'All' });
		expect(all.className).toContain('bn-toggle-chip');
		expect(all.getAttribute('data-on')).toBe('true');
	});

	it('month pills are rounded filter chips; the steppers rounded controls', async () => {
		fetchMock.mockResolvedValue(payload());
		render(FinancesPage);
		const jul = await screen.findByRole('button', { name: 'Jul 2026' });
		expect(jul.className).toContain('bn-filter-chip');
		expect(jul.className).toContain('rounded-full');
		expect(jul.className).toContain('is-on');
		expect(screen.getByRole('button', { name: 'Jun 2026' }).className).not.toContain('is-on');
		expect(screen.getByRole('button', { name: 'Previous month' }).className).toMatch(/rounded-\[6px\]/);
		expect(screen.getByRole('button', { name: /View full expenditure statement/ }).className).toContain('rounded-full');
	});

	it('Where it goes draws v1\'s full-size meters, amount only', async () => {
		fetchMock.mockResolvedValue(payload());
		render(FinancesPage);
		const label = await screen.findByText('Software', { selector: 'span.text-\\[13\\.5px\\]' });
		expect(label).toBeTruthy();
		expect(screen.getAllByText('$200').length).toBeGreaterThan(0);
		expect(screen.queryByText(/\$200 · \d+%/)).toBeNull();
	});

	it('the chosen upload lane is outlined in accent; the panel is a dashed rounded drop zone', async () => {
		fetchMock.mockResolvedValue(payload());
		render(FinancesPage);
		const panel = await screen.findByRole('region', { name: 'Upload statements' });
		expect(panel.className).toMatch(/rounded-\[10px\]/);
		expect(panel.className).toContain('border-dashed');
		const plat = screen.getByRole('button', { name: 'Platinum · Business' });
		expect(plat.getAttribute('aria-pressed')).toBe('true');
		expect(plat.className).toContain('fin-lane');
	});

	it('Largest, processor tags and recent-income amounts are rounded pills', async () => {
		fetchMock.mockResolvedValue(payload());
		render(FinancesPage);
		const largest = await screen.findByTestId('largest-charge');
		expect((largest.parentElement as HTMLElement).className).toContain('rounded-full');
		const live = screen.getAllByText('live')[0];
		expect(live.className).toContain('rounded-full');
		expect(screen.getByText('+$2,500.00').className).toContain('rounded-full');
	});
});

describe('/finances hues ride prod ramp tokens', () => {
	const view = readFileSync(resolve(__dirname, 'FinancesView.svelte'), 'utf8');
	const panel = readFileSync(resolve(__dirname, 'MonthlyExpenses.svelte'), 'utf8');
	it('spend step line and category meters take --ramp-4, charge dots --ramp-1', async () => {
		const { RAMP_1, RAMP_4 } = await import('./spend-report');
		expect(RAMP_1).toBe('var(--bn-brain-2)');
		expect(RAMP_4).toBe('color-mix(in oklab, var(--bn-brain-1) 70%, var(--bn-accent))');
		expect(view).toMatch(/<StepLine[^>]*hue=\{RAMP_4\}/);
		expect(view).toMatch(/<DotMatrix[^>]*hue=\{RAMP_1\}/);
		expect(panel).toMatch(/hue=\{RAMP_4\}/);
	});
});
