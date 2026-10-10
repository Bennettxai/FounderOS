// Pure helpers behind the /chats hub (FounderOS v1 components/ChatHub.tsx).
import type { ConversationSummary } from './types';

/** The board Conductor's pinned row: the live cockpit thread, not a direct chat. */
export const CONDUCTOR = '__board_conductor__';

/** Compact rail age: 'now', '5m', '3h', '2d'. */
export function railAgo(iso: string, now = Date.now()): string {
	const ms = now - Date.parse(iso);
	if (!Number.isFinite(ms) || ms < 0) return 'now';
	const m = Math.floor(ms / 60_000);
	if (m < 1) return 'now';
	if (m < 60) return `${m}m`;
	const h = Math.floor(m / 60);
	if (h < 24) return `${h}h`;
	return `${Math.floor(h / 24)}d`;
}

/** Up to two initials for the collapsed avatar strip. */
export function initials(name: string): string {
	return name
		.split(/[\s·]+/)
		.filter(Boolean)
		.slice(0, 2)
		.map((w) => w[0]!.toUpperCase())
		.join('');
}

/** The rail after a reply: this conversation moves to the top with its last line. */
export function bumpSummary(
	summaries: ConversationSummary[],
	agentId: string,
	agentName: string,
	last: { content: string; createdAt: string },
	count: number
): ConversationSummary[] {
	return [
		{ agentId, agentName, lastMessage: last.content.replace(/\s+/g, ' ').slice(0, 140), lastAt: last.createdAt, messageCount: count },
		...summaries.filter((c) => c.agentId !== agentId)
	];
}
