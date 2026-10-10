import { fireEvent, render, screen, waitFor } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import SkillsPage from '../../../../routes/(founderos)/os/skills/+page.svelte';
import SkillsGrid from './SkillsGrid.svelte';
import { eyebrowOf, filterCards, matches, parseMarkdown, type Block } from './skills';

const partsOf = (b: Block) => ('parts' in b ? b.parts : []);
const textOf = (b: Block) => ('text' in b ? b.text : undefined);
import type { SkillCard, SkillsBody } from './types';

const cards: SkillCard[] = [
	{ id: 'codex', name: 'codex', group: 'Engineering', kind: 'claude', description: 'Delegate to Codex.', meta: '~/.claude/skills/codex/SKILL.md', filePath: '~/.claude/skills/codex/SKILL.md' },
	{ id: 'superpowers:tdd', name: 'test-driven-development', group: 'Plugin · superpowers', kind: 'claude', description: 'Red green refactor.', meta: '~/.claude/plugins/x/SKILL.md', filePath: '~/.claude/plugins/x/SKILL.md' },
	{ id: 'skill-retrieval', name: 'Knowledge retrieval', group: 'Operator · Ops', kind: 'operator', description: 'Finds things.', meta: 'Conductor', filePath: 'skills/skill-retrieval/SKILL.md', status: 'live', markdown: '# Retrieval\n\n- **bold** step\n\n```\ncode\n```' },
	{ id: 'skill-proposal', name: 'Proposal drafting', group: 'Operator · Sales', kind: 'operator', description: 'Drafts proposals.', meta: 'unassigned', filePath: 'skills/skill-proposal/SKILL.md', status: 'learning', markdown: '# Proposal' }
];

describe('skills logic', () => {
	it('filters: All / Claude Code / Operator / Draft', () => {
		expect(cards.filter((c) => matches(c, 'All'))).toHaveLength(4);
		expect(cards.filter((c) => matches(c, 'Claude Code')).map((c) => c.id)).toEqual(['codex', 'superpowers:tdd']);
		expect(cards.filter((c) => matches(c, 'Operator'))).toHaveLength(2);
		expect(cards.filter((c) => matches(c, 'Draft')).map((c) => c.id)).toEqual(['skill-proposal']);
	});
	it('text filter searches name, description, meta and group', () => {
		expect(filterCards(cards, 'All', 'conductor').map((c) => c.id)).toEqual(['skill-retrieval']);
		expect(filterCards(cards, 'Claude Code', 'SUPERPOWERS').map((c) => c.id)).toEqual(['superpowers:tdd']);
		expect(filterCards(cards, 'All', '  ')).toHaveLength(4);
	});
	it('eyebrow is source · scope for Claude skills, the group for operator skills', () => {
		expect(eyebrowOf(cards[0])).toBe('Claude Code · user');
		expect(eyebrowOf(cards[1])).toBe('Claude Code · plugin');
		expect(eyebrowOf(cards[2])).toBe('Operator · Ops');
	});
	it('markdown: headings, bullets, rules, fences and inline bold/code, never raw HTML', () => {
		const blocks = parseMarkdown('# Title\n## Sub\n### Small\n---\n- item `x`\n\ntext **b**\n```\n<script>\n```');
		expect(blocks.map((b) => b.kind)).toEqual(['h1', 'h2', 'h3', 'hr', 'li', 'gap', 'p', 'pre']);
		expect(partsOf(blocks[4])).toEqual([
			{ kind: 'text', text: 'item ' },
			{ kind: 'code', text: 'x' }
		]);
		expect(partsOf(blocks[6])).toContainEqual({ kind: 'bold', text: 'b' });
		expect(textOf(blocks[7])).toBe('<script>');
	});
	it('an unclosed fence still renders as code', () => {
		expect(parseMarkdown('```\nopen').at(-1)).toEqual({ kind: 'pre', text: 'open' });
	});
});

const json = (body: unknown, status = 200) =>
	new Response(JSON.stringify(body), { status, headers: { 'content-type': 'application/json' } });

let fetchMock: ReturnType<typeof vi.fn>;
beforeEach(() => {
	fetchMock = vi.fn();
	vi.stubGlobal('fetch', fetchMock);
});
afterEach(() => vi.unstubAllGlobals());

describe('SkillsGrid', () => {
	it('chips carry live counts and narrow the wall; the text filter narrows further', async () => {
		const { container } = render(SkillsGrid, { cards, sourceNote: 'note' });
		expect(screen.getByText('All 4')).toBeTruthy();
		expect(screen.getByText('Claude Code 2')).toBeTruthy();
		expect(screen.getByText('Operator 2')).toBeTruthy();
		expect(screen.getByText('Draft 1')).toBeTruthy();
		expect(screen.getByText('4 of 4')).toBeTruthy();
		await fireEvent.click(screen.getByText('Draft 1'));
		expect(screen.getByText('1 of 4')).toBeTruthy();
		expect(screen.getByText('Proposal drafting')).toBeTruthy();
		await fireEvent.click(screen.getByText('All 4'));
		await fireEvent.input(screen.getByPlaceholderText('filter skills'), { target: { value: 'zzz' } });
		expect(container.textContent).toContain('no skills match "zzz"');
	});

	it('cards read eyebrow / status / title / description / footer meta', () => {
		render(SkillsGrid, { cards, sourceNote: 'note' });
		expect(screen.getAllByText('Claude Code · user').length).toBe(1);
		expect(screen.getByText('learning')).toBeTruthy();
		expect(screen.getByText('Red green refactor.')).toBeTruthy();
		expect(screen.getByText('Conductor')).toBeTruthy();
	});

	it('an operator card opens its inline SKILL.md without a fetch; Escape closes', async () => {
		render(SkillsGrid, { cards, sourceNote: 'note' });
		await fireEvent.click(screen.getByTitle('Knowledge retrieval · open SKILL.md'));
		expect(screen.getByRole('dialog')).toBeTruthy();
		expect(screen.getByText('Retrieval')).toBeTruthy();
		expect(screen.getByText('bold')).toBeTruthy();
		expect(fetchMock).not.toHaveBeenCalled();
		await fireEvent.keyDown(document, { key: 'Escape' });
		expect(screen.queryByRole('dialog')).toBeNull();
	});

	it('a Claude Code card loads SKILL.md from /pages/skills/:slug', async () => {
		fetchMock.mockResolvedValue(json({ markdown: '# TDD\n\nRed first.' }));
		render(SkillsGrid, { cards, sourceNote: 'note' });
		await fireEvent.click(screen.getByTitle('test-driven-development · open SKILL.md'));
		await waitFor(() => expect(screen.getByText('Red first.')).toBeTruthy());
		expect(fetchMock.mock.calls[0][0]).toBe('/api/founderos/pages/skills/superpowers%3Atdd');
	});

	it('a failed read says so in the reader', async () => {
		fetchMock.mockResolvedValue(json({ error: 'skill not found' }, 404));
		render(SkillsGrid, { cards, sourceNote: 'note' });
		await fireEvent.click(screen.getByTitle('codex · open SKILL.md'));
		await waitFor(() => expect(screen.getByRole('dialog').textContent).toContain('SKILL.md could not be read'));
	});

	it('download links to ?download=1 for disk skills', async () => {
		fetchMock.mockResolvedValue(json({ markdown: '# Codex' }));
		render(SkillsGrid, { cards, sourceNote: 'note' });
		await fireEvent.click(screen.getByTitle('codex · open SKILL.md'));
		const link = screen.getByTitle('Download SKILL.md') as HTMLAnchorElement;
		expect(link.getAttribute('href')).toBe('/api/founderos/pages/skills/codex?download=1');
	});
});

describe('SkillsGrid shape (prod SkillsGrid: slab pills, rounded cards, the row lens)', () => {
	it('filter chips are the slab chipClass pills; the active one is solid', () => {
		render(SkillsGrid, { cards, sourceNote: 'note' });
		const all = screen.getByText('All 4');
		expect(all.className).toContain('rounded-full');
		expect(all.className).toContain('bn-filter-chip');
		expect(all.className).toContain('is-on');
		expect(screen.getByText('Draft 1').className).not.toContain('is-on');
		expect((screen.getByPlaceholderText('filter skills') as HTMLElement).className).toContain('rounded-full');
	});
	it('cards are 10px-rounded pressable rows with a round status dot', () => {
		render(SkillsGrid, { cards, sourceNote: 'note' });
		const card = screen.getByTitle('codex · open SKILL.md');
		expect(card.className).toContain('rounded-[10px]');
		expect(card.className).toContain('bn-pressable');
		expect(card.className).toContain('is-row');
		expect(card.getAttribute('data-lens')).toBe('r');
		const dot = card.querySelector('[data-part="status-dot"]') as HTMLElement;
		expect(dot.className).toContain('rounded-full');
	});
	it('the reader is a rounded-xl panel with a rounded download button', async () => {
		render(SkillsGrid, { cards, sourceNote: 'note' });
		await fireEvent.click(screen.getByTitle('Knowledge retrieval · open SKILL.md'));
		expect(screen.getByRole('dialog').className).toContain('rounded-xl');
		expect(screen.getByTitle('Download SKILL.md').className).toContain('rounded-md');
		const pre = screen.getByRole('dialog').querySelector('pre') as HTMLElement;
		expect(pre.className).toContain('rounded-md');
	});
});

const body: SkillsBody = {
	cards,
	sourceNote: '2 skills live from ~/.claude (user + plugins) + 2 operator skills · open any card to read or download its SKILL.md.',
	operatorError: null,
	volume: {
		headline: 4,
		counts: { claude: 2, user: 1, plugin: 1, operator: 2, live: 3, learning: 1, planned: 0 },
		chips: [{ tone: 'ok', text: '3 live' }, { tone: 'warn', text: '1 learning' }],
		caption: '1 user · 1 plugin · 2 operator',
		meters: [{ label: 'Live (3/4)', frac: 0.75, display: '75%', hue: 'var(--bn-ok)' }],
		foot: '4 groups · 1 plugin · 1 tool wired',
		groups: [{ label: 'Engineering', count: 1 }],
		groupCount: 4,
		categories: [{ label: 'Ops', count: 1 }, { label: 'Sales', count: 1 }],
		owners: [{ label: 'Conductor', count: 1 }],
		unassigned: 1,
		insight: { value: 1, headline: '1 learning.', body: 'Proposal drafting', frac: 0.5 }
	}
};

describe('/os/skills page', () => {
	it('wears the v1 slab: header chip, Skill Library + Skill Volume, Categories / Owners / Drafts, then the wall', async () => {
		fetchMock.mockResolvedValue(json(body));
		const { container } = render(SkillsPage);
		expect(screen.getByRole('heading', { level: 1 }).textContent).toBe('Skills');
		await waitFor(() => expect(screen.getByText('All 4')).toBeTruthy());
		expect(fetchMock.mock.calls[0][0]).toBe('/api/founderos/pages/skills');
		expect(container.textContent).toContain('2 Claude Code skills on disk · 2 operator skills · 4 groups');
		// the header chip: live count · drafts (the insight value)
		expect(screen.getByText('3 live · 1 drafts')).toBeTruthy();
		// section order matches v1 app/skills/page.tsx
		const titles = [...container.querySelectorAll('h2')].map((h) => h.textContent?.trim());
		expect(titles).toEqual(['Skill Library', 'Skill Volume', 'Categories', 'Owners', 'Skills']);
		expect(screen.getByText('skills per source group')).toBeTruthy();
		expect(screen.getByText('operator skills')).toBeTruthy();
		expect(screen.getByText('agents wielding them')).toBeTruthy();
		// Skill Library: group count + plugin/user chips + the largest group
		expect(screen.getByText('1 from plugins')).toBeTruthy();
		expect(screen.getByText('1 user')).toBeTruthy();
		expect(container.textContent).toContain('largest · Engineering with 1');
		// Skill Volume: headline chips, caption, meters and foot
		expect(screen.getByText('3 live')).toBeTruthy();
		expect(screen.getByText('1 learning')).toBeTruthy();
		expect(container.textContent).toContain('1 user · 1 plugin · 2 operator');
		expect(container.textContent).toContain('Live (3/4)');
		expect(container.textContent).toContain('4 groups · 1 plugin · 1 tool wired');
		// Categories and Owners captions, the unassigned chip
		expect(container.textContent).toContain('most in Ops · 1');
		expect(container.textContent).toContain('Conductor holds the most · 1');
		expect(screen.getByText('1 unassigned')).toBeTruthy();
		// the Drafts gradient card
		expect(screen.getByText('Drafts')).toBeTruthy();
		expect(screen.getByText('1 learning.')).toBeTruthy();
		// the wall: title + "n of m" + source note
		expect(screen.getByText('4 of 4')).toBeTruthy();
		expect(container.textContent).toContain('2 skills live from ~/.claude');
		const agents = screen.getByRole('link', { name: 'Agents' });
		expect(agents.getAttribute('href')).toBe('/os/agents');
		// the slab PILL: rounded, pressable
		expect(agents.className).toContain('rounded-full');
		expect(agents.className).toContain('bn-pill');
	});

	it('no skills on disk reads honestly: none on disk, operator group only', async () => {
		const noDisk: SkillsBody = {
			...body,
			cards: cards.slice(2),
			sourceNote: '2 operator skills (no ~/.claude/skills on this machine) · open any card to read or download its SKILL.md.',
			volume: { ...body.volume, headline: 2, counts: { claude: 0, user: 0, plugin: 0, operator: 2, live: 1, learning: 1, planned: 0 }, groups: [{ label: 'Operator', count: 2 }], groupCount: 1 }
		};
		fetchMock.mockResolvedValue(json(noDisk));
		const { container } = render(SkillsPage);
		await waitFor(() => expect(container.textContent).toContain('0 Claude Code skills on disk · 2 operator skills · 1 groups'));
		expect(screen.getByText('0 from plugins')).toBeTruthy();
		expect(container.textContent).toContain('no ~/.claude/skills on this machine');
	});

	it('operator skills unreachable: the chip and the cards say so, disk skills still show', async () => {
		fetchMock.mockResolvedValue(json({ ...body, cards: cards.slice(0, 2), operatorError: 'operator skills unavailable: no Postgres' }));
		const { container } = render(SkillsPage);
		await waitFor(() => expect(container.textContent).toContain('operator skills unavailable: no Postgres'));
		expect(screen.getByText('All 2')).toBeTruthy();
	});

	it('backend down reads unknown, never zero skills', async () => {
		fetchMock.mockResolvedValue(json({ error: 'backend down' }, 503));
		const { container } = render(SkillsPage);
		await waitFor(() => expect(container.textContent).toContain('Skills could not be read'));
		expect(container.textContent).not.toContain('0 Claude Code skills');
	});
});
