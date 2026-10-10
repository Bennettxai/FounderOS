// The /agents tab set (FounderOS v1 components/AgentsTabs.tsx): Roster (the
// live board + Conductor rail), Needs You, Deliverables, Hermes. /os/tasks is
// its own page again, so the private build's Tasks tab is gone; parseAgentsTab
// turns any ?tab= value into a real tab, so a stale link (?tab=tasks, the old
// ?tab=board) lands on the Roster, never a blank.
export const AGENTS_TABS = {
	roster: 'Roster',
	needsyou: 'Needs You',
	deliverables: 'Deliverables',
	hermes: 'Hermes'
} as const;

export type AgentsTabId = keyof typeof AGENTS_TABS;

export function parseAgentsTab(raw: string | string[] | undefined | null): AgentsTabId {
	const v = Array.isArray(raw) ? raw[0] : raw;
	return v && Object.prototype.hasOwnProperty.call(AGENTS_TABS, v) ? (v as AgentsTabId) : 'roster';
}

/** The same page with the tab mirrored into ?tab= (the Roster is the bare URL). */
export function tabHref(current: string, id: AgentsTabId): string {
	const url = new URL(current, 'http://localhost');
	if (id === 'roster') url.searchParams.delete('tab');
	else url.searchParams.set('tab', id);
	return `${url.pathname}${url.search}${url.hash}`;
}
