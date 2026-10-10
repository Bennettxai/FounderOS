import { describe, expect, it } from 'vitest';
import { dimFor, liveDot, rosterDot, venturesForAgent, findVenture } from './org';
import type { Venture } from './types';

const V: Venture[] = [
	{ id: 'vantage', label: 'Vantage', kind: 'AI agency', color: '#00ffaa', detail: 'd', brainTag: 'vantage', focus: ['a'], areaAgents: {}, agentIds: ['conductor', 'vantage-sales'] },
	{ id: 'launchpad-cohort', label: 'Launchpad Cohort', kind: 'k', color: '#d9263f', detail: 'd', brainTag: 'aa', focus: ['b'], areaAgents: {}, agentIds: ['conductor', 'whatsapp-worker'] }
];

describe('org helpers', () => {
	it('finds a venture by id; unknown is null (all bright)', () => {
		expect(findVenture(V, 'vantage')?.label).toBe('Vantage');
		expect(findVenture(V, 'brand-deals')).toBeNull();
		expect(findVenture(V, null)).toBeNull();
	});
	it('dims agents outside the venture crew only when a venture is picked', () => {
		expect(dimFor(null, 'whatsapp-worker')).toBe(false);
		expect(dimFor(V[0], 'whatsapp-worker')).toBe(true);
		expect(dimFor(V[0], 'vantage-sales')).toBe(false);
	});
	it('lists the ventures an agent serves (shared agents serve both)', () => {
		expect(venturesForAgent(V, 'conductor').map((v) => v.id)).toEqual(['vantage', 'launchpad-cohort']);
		expect(venturesForAgent(V, 'nobody')).toEqual([]);
	});
	it('maps board and roster statuses onto the kit LED states', () => {
		// lib/org-live LIVE_DOT: running pulses ok, idle is muted (not amber)
		expect(liveDot('running')).toEqual({ fill: 'var(--bn-ok)', pulse: true });
		expect(liveDot('idle')).toEqual({ fill: 'var(--bn-text-2)', pulse: false });
		expect(liveDot('paused')).toEqual({ fill: 'var(--bn-warn)', pulse: false });
		expect(liveDot('error')).toEqual({ fill: 'var(--bn-err)', pulse: false });
		// app/org STATUS_DOT: active is the text color (white), not green;
		// idle muted, training muted + pulse, planned a hollow ring
		expect(rosterDot('active')).toEqual({ fill: 'var(--bn-text)', pulse: false });
		expect(rosterDot('idle')).toEqual({ fill: 'var(--bn-text-2)', pulse: false });
		expect(rosterDot('training')).toEqual({ fill: 'var(--bn-text-2)', pulse: true });
		expect(rosterDot('planned')).toEqual({ fill: null, pulse: false });
	});
});
