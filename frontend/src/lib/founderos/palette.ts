/**
 * The ⌘K palette's command list and filter, ported from FounderOS v1
 * lib/palette.ts: one box, three groups. Go to (the nav), Run (every agent in
 * the registry, POST /api/founderos/agents/:id/run) and Ask (Conductor prompts,
 * POST /api/founderos/pages/conductor/chat, plus a Brain jump). The off-OS
 * external links (localhost:4000, Remotion…) are FounderOS-v1-host specific and
 * are not carried over.
 */
import { DIGIT_VIEWS, NAV_GROUPS } from './nav';

export type PaletteKind = 'go' | 'run' | 'ask';
export type PaletteScope = 'all' | PaletteKind;
export const PALETTE_SCOPES: PaletteScope[] = ['all', 'go', 'run', 'ask'];

/** The slice of an agent row the palette needs (GET /api/founderos/agents). */
export type PaletteAgent = { id: string; name: string; role: string };

export type PaletteCommand = {
	id: string;
	kind: PaletteKind;
	title: string;
	sub: string;
	glyph: string;
	/** go: the route. ask: an optional page jump instead of a prompt. */
	href?: string;
	aliases?: string[];
	/** run: target for POST /api/founderos/agents/:id/run */
	agentId?: string;
	/** ask: the message POSTed to the Conductor chat */
	prompt?: string;
};

// Search vocabulary the labels alone don't cover (FounderOS v1 NAV_ALIASES), with
// the knowledge view now answering to the Optimal Engine instead of gbrain.
const NAV_ALIASES: Record<string, string[]> = {
	'/os': ['home', 'console', 'dashboard', 'overview'],
	'/os/comms': ['inbox', 'email', 'messages', 'slack', 'whatsapp', 'unread'],
	'/os/funnel': ['clients', 'journey', 'pipeline', 'vantage'],
	'/os/workflows': ['automations', 'crons'],
	'/os/social': ['instagram', 'tiktok', 'youtube', 'followers', 'zernio'],
	'/os/content': ['videos', 'posts', 'carousel'],
	'/os/brand-deals': ['sponsors', 'partnerships'],
	'/os/finances': ['money', 'stripe', 'revenue', 'expenses'],
	'/os/trading': ['robinhood', 'phantom', 'markets', 'portfolio'],
	'/os/agents': ['roster', 'workforce', 'deliverables'],
	'/os/chats': ['conversations', 'threads'],
	'/os/tasks': ['todo', 'board', 'kanban'],
	'/os/skills': ['sops', 'playbooks'],
	'/os/org': ['hierarchy', 'chart', 'structure', 'pillars'],
	'/os/brain': ['optimal', 'engine', 'knowledge', 'memory', 'graph', 'recall'],
	'/os/doctor': ['health', 'diagnostics', 'checks'],
	'/os/integrations': ['connections', 'tools', 'creds', 'status'],
	'/os/usage': ['tokens', 'burn', 'quota', 'seats'],
	'/os/analytics': ['metrics', 'numbers'],
	'/os/personas': ['templates', 'variants']
};

const ASK: PaletteCommand[] = [
	{
		id: 'ask-attention',
		kind: 'ask',
		glyph: '?',
		title: 'What needs my attention?',
		sub: 'Conductor · triage comms and the board',
		prompt: 'What needs my attention right now? Triage comms and the board and give me the top items.'
	},
	{
		id: 'ask-status',
		kind: 'ask',
		glyph: '?',
		title: 'Agent status report',
		sub: 'Conductor · last run of every agent',
		prompt: 'Give me a status report: the last run of every agent and anything that failed.'
	},
	{
		id: 'ask-day',
		kind: 'ask',
		glyph: '?',
		title: 'Plan my day',
		sub: 'Conductor · calendar + open work',
		prompt: 'Plan my day from the calendar and the open work on the board.'
	},
	{
		id: 'ask-brain',
		kind: 'ask',
		glyph: '◎',
		title: 'Search the Brain',
		sub: 'open the knowledge core',
		href: '/os/brain',
		aliases: ['optimal', 'knowledge', 'query', 'recall']
	}
];

export function buildPaletteCommands(agents: PaletteAgent[] = []): PaletteCommand[] {
	const runs: PaletteCommand[] = agents.map((a) => ({
		id: `run-${a.id}`,
		kind: 'run',
		title: a.name,
		sub: a.role,
		glyph: '▸',
		agentId: a.id,
		aliases: [a.name.split(/\s+/)[0].toLowerCase()]
	}));
	return [...goCommands(), ...runs, ...ASK];
}

function goCommands(): PaletteCommand[] {
	return NAV_GROUPS.flatMap((g) =>
		g.items.map(({ href, label }) => ({
			id: `go-${href}`,
			kind: 'go' as const,
			title: label,
			sub: g.sub,
			glyph: '›',
			href,
			aliases: NAV_ALIASES[href]
		}))
	);
}

/** Scope narrows to one kind; then every query word must match somewhere in
 *  title / sub / aliases / kind words, so "run inbox" finds the Inbox agent and
 *  "jump comms" finds the view (FounderOS v1 filterPalette). */
export function filterPalette(commands: PaletteCommand[], query: string, scope: PaletteScope = 'all'): PaletteCommand[] {
	const words = query.trim().toLowerCase().split(/\s+/).filter(Boolean);
	return commands.filter((c) => {
		if (scope !== 'all' && c.kind !== scope) return false;
		if (words.length === 0) return true;
		const kindWords = c.kind === 'go' ? 'go open jump' : c.kind;
		const hay = `${c.title} ${c.sub} ${c.aliases?.join(' ') ?? ''} ${kindWords}`.toLowerCase();
		return words.every((w) => hay.includes(w));
	});
}

/** The view a bare digit key jumps to, or null. */
export function digitTarget(key: string): string | null {
	if (!/^[1-9]$/.test(key)) return null;
	return DIGIT_VIEWS[Number(key) - 1] ?? null;
}
