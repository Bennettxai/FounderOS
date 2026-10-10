import { isGuardRefusal } from '$lib/founderos/api';
/**
 * The Conductor dock's pure half, ported from FounderOS v1
 * components/ConductorPanel.tsx and lib/composer.ts.
 * ConductorPanel.svelte is the DOM half. The chat is the real board thread
 * (Paperclip cockpit issue) behind /api/founderos/pages/conductor/*.
 */
import { FounderosApiError } from '../api';

/** Cross-component open signal: the Topbar button and the palette fire it. */
export const CONDUCTOR_OPEN_EVENT = 'conductor:open';

export function openConductor(): void {
	window.dispatchEvent(new CustomEvent(CONDUCTOR_OPEN_EVENT));
}

/** The dock publishes its width here; the /os shell reads it as its right margin. */
export const CONDUCTOR_W_VAR = '--bn-conductor-w';
export const CONDUCTOR_WIDTH_KEY = 'founderos-conductor-w';
export const CONDUCTOR_DEFAULT_W = 380;
export const CONDUCTOR_MIN_W = 300;
export const CONDUCTOR_MAX_W = 760;

export function clampConductorW(w: number): number {
	if (!Number.isFinite(w)) return CONDUCTOR_DEFAULT_W;
	return Math.min(CONDUCTOR_MAX_W, Math.max(CONDUCTOR_MIN_W, Math.round(w)));
}

export type QuickAction = { label: string; prompt: string };

/** GET /pages/conductor/context */
export type ScreenCtx = {
	title: string;
	context: string;
	quickActions?: QuickAction[];
	/** The model in the Conductor seat right now, or null when the board is out. */
	model?: string | null;
};

/** One cockpit comment from GET /pages/conductor/chat. */
export type ThreadMessage = { id: string; body: string; authorType: string; createdAt: string };

export type Turn = {
	id: string;
	role: 'user' | 'assistant';
	content: string;
	routedTo?: string;
	createdAt?: string;
	/** What the turn actually DID, if anything: a one-line proof with a way in. */
	receipt?: { text: string; href?: string };
};

/** The openers used only until the per-screen set lands (never, if the board is out). */
export const FALLBACK_QUICK_ACTIONS: QuickAction[] = [
	{ label: 'What needs me?', prompt: 'What needs my attention right now across the OS?' },
	{ label: 'Board status', prompt: 'Give me the current board status: what is running, blocked, and done today.' },
	{ label: "Today's digest", prompt: 'Summarize today: comms, agents, money, anything unusual.' }
];

/** A clear is local, not a board delete: anything at or before it stays hidden. */
export function threadToTurns(messages: ThreadMessage[], clearedAt: number): Turn[] {
	return messages
		.filter((m) => Date.parse(m.createdAt) > clearedAt)
		.map((m) => {
			const agent = m.authorType === 'agent';
			return {
				id: m.id,
				role: agent ? 'assistant' : 'user',
				content: m.body,
				createdAt: m.createdAt,
				routedTo: agent ? 'Conductor · board' : undefined
			};
		});
}

/** Board turns plus the dock's own local turns (a /ui request and its receipt,
 *  which never go to the board), in time order, so a poll refresh cannot wipe a
 *  receipt. Turns without a time keep their place at the end. */
export function mergeTurns(board: Turn[], local: Turn[]): Turn[] {
	const at = (t: Turn) => (t.createdAt ? Date.parse(t.createdAt) : Number.POSITIVE_INFINITY);
	return [...board, ...local].map((t, i) => ({ t, i })).sort((a, b) => at(a.t) - at(b.t) || a.i - b.i).map(({ t }) => t);
}

export function assistantCount(turns: Turn[]): number {
	return turns.filter((t) => t.role === 'assistant').length;
}

/** The screen rides along so the Conductor knows what the operator is looking at. */
export function withScreen(composed: string, ctx: Pick<ScreenCtx, 'title'> | null): string {
	return ctx ? `${composed}\n\n(the operator is looking at: ${ctx.title})` : composed;
}

/** `/ui <request>` dispatches a coding agent instead of chatting; null for chat. */
export function uiRequest(display: string): string | null {
	const m = /^\/ui\s+([\s\S]+)$/i.exec(display);
	return m ? m[1].trim() : null;
}

export function dispatchTurn(request: string, res: { workspaceId?: string; branch?: string }, now: number): Turn {
	return {
		id: `dispatch-${now}`,
		role: 'assistant',
		createdAt: new Date(now).toISOString(),
		content: `Dispatched a coding agent for: "${request}"\n\nBranch: ${res.branch ?? 'unknown'}\nWorkspace: ${res.workspaceId ?? 'unknown'}`,
		routedTo: 'Superset · coding agent',
		receipt: { text: 'Workspace opened on an isolated branch', href: '/os/agents' }
	};
}

/** After this long a pending reply stops spinning and says it will land later. */
export const SLOW_AFTER_MS = 150_000;
export const SLOW_MESSAGE = 'The CEO run is taking a while. Its reply lands in this thread automatically.';

export const REFUSED_NOTICE = 'Writes are off (FOUNDEROS_WRITES=0), so nothing was sent.';

// Only the write guard's own refusal reads as "writes are off"; any other 409
// (Paperclip not configured, a conflict) keeps the server's reason.
function refused(err: FounderosApiError): boolean {
	return err.status === 409 && isGuardRefusal(err);
}

export function failureMessage(err: unknown): string {
	if (err instanceof FounderosApiError && refused(err)) return REFUSED_NOTICE;
	if (err instanceof Error) return err.message;
	return String(err);
}

/** One line at the composer's type size; keeps an empty box clickable. */
export const COMPOSER_MIN_PX = 20;
/** ~10 lines. Past this the textarea scrolls rather than eating the panel. */
export const COMPOSER_MAX_PX = 200;

/** The composer measures scrollHeight (wrapped lines, not newline count) and this clamps it. */
export function composerHeight(scrollHeight: number, max = COMPOSER_MAX_PX, min = COMPOSER_MIN_PX): number {
	if (!Number.isFinite(scrollHeight) || scrollHeight <= min) return min;
	return Math.min(scrollHeight, max);
}
