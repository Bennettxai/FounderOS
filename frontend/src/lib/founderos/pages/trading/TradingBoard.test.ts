import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { fireEvent, render, within } from '@testing-library/svelte';
import { describe, expect, test, vi } from 'vitest';
import TradingBoard from './TradingBoard.svelte';
import { tradingFixture } from './fixture';

const src = (f: string) => readFileSync(resolve(__dirname, f), 'utf8');

function board(over = {}) {
	const onRefresh = vi.fn();
	const r = render(TradingBoard, { data: tradingFixture(over), onRefresh });
	return { ...r, onRefresh };
}

const part = (c: HTMLElement, name: string) => c.querySelector(`[data-part="${name}"]`) as HTMLElement;

describe('the trading slab', () => {
	test('the title row: total across accounts, freshness, monitor-only, a refresh button', async () => {
		const { getByText, getByRole, onRefresh } = board();
		expect(getByText('Trading')).toBeTruthy();
		expect(getByText(/\$2,000\.00 across 2 accounts \+ wallet · synced 30m ago/)).toBeTruthy();
		expect(getByText('live · robinhood')).toBeTruthy();
		expect(getByText('monitor-only · refreshes 60s')).toBeTruthy();
		await fireEvent.click(getByRole('button', { name: 'Refresh feed' }));
		expect(onRefresh).toHaveBeenCalledTimes(1);
	});

	test('no Individual / Agentic / Phantom SOL top row: the board opens on the sleeve (prod TradingBoard.tsx)', () => {
		const { container, queryByText } = board();
		expect(part(container, 'top-row')).toBeNull();
		expect(queryByText('Phantom SOL')).toBeNull();
		const hero = part(container, 'hero-row');
		const title = container.querySelector('h1, h2');
		expect(hero).toBeTruthy();
		expect(title!.compareDocumentPosition(hero) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
	});

	test('no wallet: the title never claims one', () => {
		const { getByText, queryByText } = board({ phantom: null, phantomStatus: { state: 'error', detail: 'Solana RPC did not return a balance: timeout' } });
		expect(getByText(/\$1,800\.00 across 2 accounts · synced 30m ago/)).toBeTruthy();
		expect(queryByText(/\+ wallet/)).toBeNull();
	});

	test('an unpriced wallet is not counted into the total as if it were included', () => {
		const f = tradingFixture();
		const { getByText, queryByText } = board({ phantom: { ...f.phantom!, usdPerSol: null, usdValue: null } });
		expect(getByText(/\$1,800\.00 across 2 accounts · wallet value unknown · synced 30m ago/)).toBeTruthy();
		expect(queryByText(/\+ wallet/)).toBeNull();
	});
});

/**
 * Prod components/trading/TradingBoard.tsx (the code, not the CLAUDE.md prose):
 * a hero row of the sleeve's value line beside the Accounts card, then a
 * second row of reasoning, positions and the one gradient card.
 */
describe('the hero row: the sleeve beside the Accounts card', () => {
	test('one 2fr/1fr grid, sleeve first, Accounts second', () => {
		const { container } = board();
		const row = part(container, 'hero-row');
		expect(row.className).toMatch(/\bgrid\b/);
		expect(row.className).toContain('grid-cols-[2fr_1fr]');
		const cells = [...row.children] as HTMLElement[];
		expect(cells.map((c) => c.getAttribute('data-part'))).toEqual(['sleeve', 'accounts']);
		expect(within(cells[0]).getByText('Agent · the sleeve')).toBeTruthy();
		expect(within(cells[1]).getByText('Accounts')).toBeTruthy();
	});

	test('the Accounts card: status detail, the headline, a meter per account plus the wallet, the foot', () => {
		const { container } = board();
		const acc = part(container, 'accounts');
		expect(acc.textContent).toContain('2 accounts · updated 30 min ago');
		expect(acc.textContent).toContain('Individual · read-only to agents');
		expect(acc.textContent).toContain('Agentic · agent may trade');
		expect(acc.textContent).toContain('Phantom · 2 SOL');
		expect(acc.textContent).toContain('only the agentic sleeve can be traded by an agent');
	});

	test('card heads carry the meta under the title and a kebab menu', () => {
		const { container, getByRole } = board();
		for (const t of ['Agent · the sleeve', 'Accounts', 'Agent · reasoning', 'Positions', 'Open orders']) {
			expect(getByRole('button', { name: `${t} menu` })).toBeTruthy();
		}
		const head = part(container, 'sleeve').querySelector('[data-part="tb-head"]') as HTMLElement;
		const meta = head.querySelector('[data-part="card-meta"]') as HTMLElement;
		expect(meta.textContent).toContain('1 trade');
		expect(head.querySelector('h2')!.compareDocumentPosition(meta) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
	});

	test('the sleeve graph draws from the account history, and switches account', async () => {
		const { container, getByRole } = board();
		const sleeve = part(container, 'sleeve');
		expect(sleeve.querySelector('svg[role="img"]')!.getAttribute('aria-label')).toContain('$590.00 to $600.00');
		await fireEvent.click(getByRole('button', { name: 'Individual' }));
		expect(within(sleeve).getByText(/No value series yet/)).toBeTruthy();
	});

	test('a sleeve that has not traded says it is sitting in cash', () => {
		const f = tradingFixture();
		const { getByText } = board({ view: { ...f.view, agent: { ...f.view.agent, hasActed: false, tradeCount: 0, lastTradeAt: null } } });
		expect(getByText(/sitting in cash/)).toBeTruthy();
		expect(getByText('no trades placed yet')).toBeTruthy();
	});
});

describe('the second row: reasoning, positions, the next move', () => {
	test('one 1.15fr/1fr/.85fr grid in that order', () => {
		const { container } = board();
		const row = part(container, 'second-row');
		expect(row.className).toContain('grid-cols-[1.15fr_1fr_.85fr]');
		const cells = [...row.children] as HTMLElement[];
		expect(cells).toHaveLength(3);
		expect(cells[0].getAttribute('data-part')).toBe('reasoning');
		expect(cells[1].getAttribute('data-part')).toBe('positions');
		expect(cells[2].textContent).toContain('Agent · next move');
		expect(part(container, 'hero-row').compareDocumentPosition(row) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
	});

	test('fifty ticker rows scroll inside a capped list instead of stretching the row', () => {
		const rows = Array.from({ length: 50 }, (_, i) => ({ ticker: `T${i}`, score: i, verdict: 'watch', reason: 'r' }));
		const f = tradingFixture();
		const { container } = board({ analysis: { ...f.analysis!, rows } });
		const list = part(container, 'reasoning-list');
		expect(list.querySelectorAll('li')).toHaveLength(50);
		expect(list.className).toContain('overflow-y-auto');
		expect(list.className).toContain('max-h-[260px]');
	});

	test('the positions list is capped at 236px', () => {
		const { container } = board();
		const ul = part(container, 'positions').querySelector('ul') as HTMLElement;
		expect(ul.className).toContain('max-h-[236px]');
	});

	test('the reasoning filters its rows by the verdicts present in the run', async () => {
		const { container } = board();
		const panel = within(part(container, 'reasoning'));
		expect(panel.getByRole('button', { name: 'all' })).toBeTruthy();
		await fireEvent.click(panel.getByRole('button', { name: 'watch' }));
		const list = part(container, 'reasoning-list');
		expect(list.textContent).toContain('SPY');
		expect(list.textContent).not.toContain('QQQ');
		expect(list.textContent).toContain('unscored');
	});

	test('no analysis yet says so plainly', () => {
		const { getByText } = board({ analysis: null });
		expect(getByText(/No analysis pushed yet/)).toBeTruthy();
	});
});

describe('below the second row', () => {
	test('sections in order: open orders, trade log, limits', () => {
		const { container } = board();
		const order = ['open-orders', 'trade-log', 'limits'].map((p) => part(container, p));
		for (let i = 1; i < order.length; i++) {
			expect(order[i - 1].compareDocumentPosition(order[i]) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
		}
		expect(part(container, 'second-row').compareDocumentPosition(order[0]) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
	});

	test('open orders say who placed them, agent vs you', () => {
		const { container } = board();
		const orders = part(container, 'open-orders');
		const items = orders.querySelectorAll('li');
		expect(items).toHaveLength(2);
		expect(items[0].textContent).toContain('SPY');
		expect(items[0].textContent).toContain('agent');
		expect(items[0].textContent).toContain('$25.00');
		expect(items[1].textContent).toContain('you');
		expect(items[1].textContent).toContain('limit $250.00');
		expect(orders.textContent).toContain('2 working');
	});

	test('no live orders: nothing working', () => {
		const { container } = board({ openOrders: [] });
		expect(part(container, 'open-orders').textContent).toContain('nothing working');
		expect(part(container, 'open-orders').textContent).toContain('No live orders at the broker.');
	});

	test('positions list every holding with its P&L', () => {
		const { container } = board();
		const p = part(container, 'positions');
		expect(p.textContent).toContain('NVDA');
		expect(p.textContent).toContain('-$12.00');
		expect(p.textContent).toContain('+$2.00');
	});

	test('the trade log filters by who and outcome, and the rationale opens in a drawer', async () => {
		const { container, getByRole, queryByRole } = board();
		const log = part(container, 'trade-log');
		expect(log.textContent).toContain('2 of 2');
		await fireEvent.click(within(log).getByRole('button', { name: 'Rejected 1' }));
		expect(log.textContent).toContain('1 of 2');
		expect(log.textContent).toContain('TSLA');
		expect(log.textContent).not.toContain('Dip under');
		await fireEvent.click(within(log).getByRole('button', { name: 'Agent 1' }));
		await fireEvent.click(within(log).getByRole('button', { name: /QQQ/ }));
		const drawer = getByRole('complementary');
		expect(drawer.textContent).toContain('Dip under the 20-day band; index core top-up.');
		expect(drawer.textContent).toContain('Markets Agent');
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(queryByRole('complementary')).toBeNull();
	});

	test("the limits wear prod's editor (inputs, autopilot switch, save) but are held read-only: nothing on the port can arm the agent", async () => {
		const { container } = board();
		const l = part(container, 'limits');
		for (const label of ['Autonomy budget', 'Max per trade', 'Max position', 'Risk per trade', 'Max positions', 'Trades per day', 'Kill-switch floor']) {
			expect(l.textContent).toContain(label);
		}
		expect(l.textContent).toContain('code defaults');
		const inputs = [...l.querySelectorAll('input')] as HTMLInputElement[];
		expect(inputs).toHaveLength(7);
		for (const i of inputs) expect(i.readOnly).toBe(true);
		const sw = within(l).getByRole('switch', { name: 'Autopilot' });
		expect(sw.getAttribute('aria-checked')).toBe('false');
		expect((sw as HTMLButtonElement).disabled).toBe(true);
		await fireEvent.click(sw);
		expect(sw.getAttribute('aria-checked')).toBe('false');
		expect(l.textContent).toContain('OFF: the daily run proposes and logs every order as a dry run');
		const save = within(l).getByRole('button', { name: 'save' }) as HTMLButtonElement;
		expect(save.disabled).toBe(true);
		// v1's footer: nothing beside SAVE until a save or a clamp has something to say
		expect(l.textContent).not.toMatch(/bridge/i);
		expect(l.querySelector('footer')?.textContent?.trim().toLowerCase()).toBe('save');
	});

	test('no orders and no analysis: v1 counts 0 runs recorded and says the agent has not reported in', () => {
		const { container } = board({ openOrders: [], analysis: null });
		const v = container.querySelector('.bn-insight [data-part="insight-value"]') as HTMLElement;
		expect(v.textContent?.trim()).toBe('0');
		expect(container.querySelector('.bn-insight')?.textContent).toContain('runs recorded. The agent has not reported in.');
	});

	test('exactly one insight card: the agent next move', () => {
		const { container } = board();
		expect(container.querySelectorAll('.bn-insight')).toHaveLength(1);
		expect(container.textContent).toContain('Agent · next move');
	});
});

describe('honest freshness and empty states', () => {
	test.each([
		['seeded', 'seeded · awaiting the feed'],
		['stale', 'stale · 27d ago'],
		['none', 'no feed']
	] as const)('%s', (state, badge) => {
		const f = tradingFixture();
		const label = state === 'stale' ? 'synced 27d ago' : state === 'seeded' ? 'seeded · example rows' : 'no feed yet';
		const { getByText } = board({ view: { ...f.view, freshness: { state, label } } });
		expect(getByText(badge)).toBeTruthy();
	});

	test('no account data: the status detail and limits still shown', () => {
		const detail = 'Robinhood not feeding the OS yet.';
		const { getByText, container } = board({ accounts: [], history: {}, positions: [], activity: [], openOrders: [], analysis: null, status: { id: 'robinhood', name: 'Robinhood', state: 'not_configured', detail } });
		expect(getByText('No account data yet')).toBeTruthy();
		expect(getByText(detail)).toBeTruthy();
		expect(part(container, 'limits')).toBeTruthy();
		expect(part(container, 'hero-row')).toBeNull();
	});
});

// Round 2 (prod 2026-10 snapshot): prod's rounded furniture on every control.
describe('prod shapes', () => {
	test('monitor-only is a round pill, refresh a round pressable', () => {
		const { getByText, getByRole } = board();
		expect(getByText('monitor-only · refreshes 60s').className).toContain('rounded-full');
		const r = getByRole('button', { name: 'Refresh feed' });
		expect(r.className).toContain('rounded-full');
		expect(r.className).toContain('bn-pressable');
	});

	test("account switch and reasoning verdicts are the kit's 24px toggle pills", () => {
		const { container, getByRole } = board();
		const ind = getByRole('button', { name: 'Individual' });
		expect(ind.className).toContain('bn-toggle-chip');
		expect(ind.getAttribute('data-on')).toBe('false');
		expect(getByRole('button', { name: 'Agentic' }).getAttribute('data-on')).toBe('true');
		const all = within(part(container, 'reasoning')).getByRole('button', { name: 'all' });
		expect(all.className).toContain('bn-toggle-chip');
	});

	test('the trade log filters are the slab filter chips', () => {
		const { container } = board();
		const all = within(part(container, 'trade-log')).getByRole('button', { name: 'All 2' });
		expect(all.className).toContain('bn-filter-chip');
		expect(all.className).toContain('is-on');
		expect(within(part(container, 'trade-log')).getByRole('button', { name: 'Agent 1' }).className).not.toContain('is-on');
	});

	test("card heads stay out of the kit's Home card-head rules (no 10px foot, no pill-styled menu rows)", () => {
		const { container } = board();
		expect(container.querySelector('[data-part="card-head"]')).toBeNull();
		expect(container.querySelectorAll('[data-part="tb-head"]').length).toBeGreaterThanOrEqual(5);
	});

	test("the kebab is round; its menu a rounded glass panel of plain rows", async () => {
		const { getByRole } = board();
		const k = getByRole('button', { name: 'Accounts menu' });
		expect(k.className).toContain('rounded-full');
		await fireEvent.click(k);
		const menu = getByRole('menu');
		expect(menu.className).toMatch(/rounded-\[8px\]/);
		const items = within(menu).getAllByRole('menuitem');
		expect(items.map((i) => i.textContent!.trim())).toEqual(['Refresh from the feed', 'Open Robinhood']);
	});

	test('the stat strip and the reasoning notes are 10px rounded panels', () => {
		const { container } = board();
		expect(part(container, 'sleeve-stats').className).toMatch(/rounded-\[10px\]/);
		expect((part(container, 'reasoning').querySelector('p') as HTMLElement).className).toMatch(/rounded-\[10px\]/);
	});

	test("buy/sell read as prod's small action tag, not a bordered badge", () => {
		const { container } = board();
		const tag = part(container, 'trade-log').querySelector('[data-part="action-tag"]') as HTMLElement;
		expect(tag.textContent!.trim()).toMatch(/^(buy|sell)$/);
		expect(tag.className).toContain('text-[9px]');
		expect(tag.className).not.toContain('border');
	});

	test('position and trade rows are pressable rows; symbol and account sit tight', () => {
		const { container } = board();
		const li = part(container, 'positions').querySelector('li') as HTMLElement;
		expect(li.className).toContain('is-row');
		expect(li.getAttribute('data-lens')).toBe('r');
		expect(li.querySelector('div')!.textContent).toMatch(/^QQQagentic/);
		const row = part(container, 'trade-log').querySelector('button[data-lens="r"]') as HTMLElement;
		expect(row.className).toContain('is-row');
	});
});
