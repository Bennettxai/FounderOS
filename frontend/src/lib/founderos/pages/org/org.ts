// Pure helpers behind the /os/org board (lib/ventures.ts lookups and the
// org-live LED map, on the kit's Dot states).
import type { Venture } from './types';

export function findVenture(ventures: Venture[], id: string | null | undefined): Venture | null {
	if (!id) return null;
	return ventures.find((v) => v.id === id) ?? null;
}

/** Venture lens: no venture = everything bright. */
export function dimFor(venture: Venture | null, agentId: string): boolean {
	return venture ? !venture.agentIds.includes(agentId) : false;
}

export function venturesForAgent(ventures: Venture[], agentId: string): Venture[] {
	return ventures.filter((v) => v.agentIds.includes(agentId));
}

/** An org LED: a fill color, or null for prod's hollow ring; pulse = Tailwind animate-pulse. */
export type OrgDotLook = { fill: string | null; pulse: boolean };

/** Board status -> LED (lib/org-live LIVE_DOT: running pulses ok, idle muted, paused warn, error err). */
export function liveDot(status: string): OrgDotLook {
	switch (status) {
		case 'running':
			return { fill: 'var(--bn-ok)', pulse: true };
		case 'paused':
			return { fill: 'var(--bn-warn)', pulse: false };
		case 'error':
			return { fill: 'var(--bn-err)', pulse: false };
		default:
			return { fill: 'var(--bn-text-2)', pulse: false };
	}
}

/** Roster status -> LED (app/org STATUS_DOT: active the text color, idle muted,
 *  training muted + pulse, planned a hollow ring). Unknown reads hollow. */
export function rosterDot(status: string): OrgDotLook {
	switch (status) {
		case 'active':
			return { fill: 'var(--bn-text)', pulse: false };
		case 'idle':
			return { fill: 'var(--bn-text-2)', pulse: false };
		case 'training':
			return { fill: 'var(--bn-text-2)', pulse: true };
		default:
			return { fill: null, pulse: false };
	}
}
