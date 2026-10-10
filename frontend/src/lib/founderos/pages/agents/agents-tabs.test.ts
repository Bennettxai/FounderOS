import { describe, expect, it } from 'vitest';
import { AGENTS_TABS, parseAgentsTab, tabHref } from './agents-tabs';

describe('agents tabs (FounderOS v1 components/AgentsTabs.tsx)', () => {
	it('is Roster, Needs You, Deliverables, Hermes in that order', () => {
		expect(Object.entries(AGENTS_TABS)).toEqual([
			['roster', 'Roster'],
			['needsyou', 'Needs You'],
			['deliverables', 'Deliverables'],
			['hermes', 'Hermes']
		]);
	});

	it('turns any ?tab= into a real tab; the private build board/tasks ids land on the Roster', () => {
		expect(parseAgentsTab('needsyou')).toBe('needsyou');
		expect(parseAgentsTab(['hermes', 'tasks'])).toBe('hermes');
		expect(parseAgentsTab('board')).toBe('roster');
		expect(parseAgentsTab('tasks')).toBe('roster');
		expect(parseAgentsTab('toString')).toBe('roster');
		expect(parseAgentsTab('nope')).toBe('roster');
		expect(parseAgentsTab(null)).toBe('roster');
		expect(parseAgentsTab(undefined)).toBe('roster');
	});

	it('mirrors the tab into ?tab=, the roster being the bare URL', () => {
		expect(tabHref('http://x/os/agents?tab=hermes&q=1', 'roster')).toBe('/os/agents?q=1');
		expect(tabHref('http://x/os/agents', 'deliverables')).toBe('/os/agents?tab=deliverables');
		expect(tabHref('http://x/os/agents?tab=needsyou#h', 'hermes')).toBe('/os/agents?tab=hermes#h');
	});
});
