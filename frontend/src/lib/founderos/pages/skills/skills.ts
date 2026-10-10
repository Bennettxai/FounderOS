/** Pure logic behind the /skills card wall (FounderOS v1 components/SkillsGrid.tsx). */
import type { SkillCard } from './types';

export type Filter = 'All' | 'Claude Code' | 'Operator' | 'Draft';
export const FILTERS: Filter[] = ['All', 'Claude Code', 'Operator', 'Draft'];

export const isDraft = (c: SkillCard) => c.status != null && c.status !== 'live';

export const matches = (c: SkillCard, f: Filter) =>
	f === 'All' ? true : f === 'Draft' ? isDraft(c) : f === 'Claude Code' ? c.kind === 'claude' : c.kind === 'operator';

export function filterCards(cards: SkillCard[], f: Filter, query: string): SkillCard[] {
	const q = query.trim().toLowerCase();
	return cards.filter((c) => matches(c, f) && (q === '' || `${c.name} ${c.description} ${c.meta} ${c.group}`.toLowerCase().includes(q)));
}

/** What sits above the title: the card's source and scope. */
export const eyebrowOf = (c: SkillCard) => (c.kind === 'claude' ? `Claude Code · ${c.id.includes(':') ? 'plugin' : 'user'}` : c.group);

export type IconKey = 'flame' | 'code' | 'image' | 'signature' | 'plug' | 'hammer' | 'spec' | 'review' | 'target' | 'content' | 'ops' | 'sparkles';

/** The tool or nature each skill runs on, as an icon key. */
export function iconKeyOf(c: SkillCard): IconKey {
	if (c.group === 'Firecrawl') return 'flame';
	const byId: Record<string, IconKey> = {
		codex: 'code',
		'nano-banana': 'image',
		'proposal-generator': 'signature',
		'mcp-builder': 'plug',
		build: 'hammer',
		spec: 'spec',
		review: 'review'
	};
	if (byId[c.id]) return byId[c.id];
	const byGroup: Record<string, IconKey> = { Sales: 'target', Content: 'content', Ops: 'ops', Engineering: 'code' };
	return byGroup[c.group.replace(/^Operator · /, '')] ?? 'sparkles';
}

export type Inline = { kind: 'text' | 'bold' | 'code'; text: string };
export type Block =
	| { kind: 'h1' | 'h2' | 'h3' | 'p' | 'li'; parts: Inline[] }
	| { kind: 'hr' | 'gap' }
	| { kind: 'pre'; text: string };

export function inline(text: string): Inline[] {
	return text
		.split(/(\*\*[^*]+\*\*|`[^`]+`)/g)
		.filter((p) => p !== '')
		.map((p) =>
			p.startsWith('**') && p.endsWith('**') && p.length > 4
				? { kind: 'bold' as const, text: p.slice(2, -2) }
				: p.startsWith('`') && p.endsWith('`') && p.length > 2
					? { kind: 'code' as const, text: p.slice(1, -1) }
					: { kind: 'text' as const, text: p }
		);
}

/** The reader's small markdown: headings, bullets, rules, fences, inline bold and code.
 *  Everything renders as text nodes, so a SKILL.md can never inject markup. */
export function parseMarkdown(src: string): Block[] {
	const out: Block[] = [];
	let fence: string[] | null = null;
	for (const line of src.split('\n')) {
		if (line.trim().startsWith('```')) {
			if (fence) {
				out.push({ kind: 'pre', text: fence.join('\n') });
				fence = null;
			} else fence = [];
			continue;
		}
		if (fence) {
			fence.push(line);
			continue;
		}
		if (/^#\s/.test(line)) out.push({ kind: 'h1', parts: inline(line.slice(2)) });
		else if (/^##\s/.test(line)) out.push({ kind: 'h2', parts: inline(line.slice(3)) });
		else if (/^###\s/.test(line)) out.push({ kind: 'h3', parts: inline(line.slice(4)) });
		else if (/^---\s*$/.test(line)) out.push({ kind: 'hr' });
		else if (/^\s*[-*]\s/.test(line)) out.push({ kind: 'li', parts: inline(line.replace(/^\s*[-*]\s/, '')) });
		else if (line.trim() === '') out.push({ kind: 'gap' });
		else out.push({ kind: 'p', parts: inline(line) });
	}
	if (fence) out.push({ kind: 'pre', text: fence.join('\n') });
	return out;
}
