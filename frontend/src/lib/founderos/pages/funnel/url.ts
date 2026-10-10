/** /os/funnel view state ↔ its query string (FounderOS v1 app/funnel/page.tsx href()). */
import type { FunnelStage, FunnelVenture } from './types';

export type FunnelView = {
	venture: FunnelVenture | null;
	view: 'live' | 'archive';
	stage: FunnelStage | null;
	layout: 'flow' | 'radial';
	lead: string | null;
};

const VENTURES = new Set(['vantage', 'launchpad-cohort']);
const STAGES = new Set(['first_touch', 'engaged', 'nurtured', 'opted_in', 'converted']);

export function parseFunnelView(search: string): FunnelView {
	const p = new URLSearchParams(search);
	const venture = p.get('venture');
	const stage = p.get('stage');
	return {
		venture: venture && VENTURES.has(venture) ? (venture as FunnelVenture) : null,
		view: p.get('view') === 'archive' ? 'archive' : 'live',
		stage: stage && STAGES.has(stage) ? (stage as FunnelStage) : null,
		layout: p.get('layout') === 'radial' ? 'radial' : 'flow',
		lead: p.get('lead') || null
	};
}

export function funnelHref(v: FunnelView): string {
	const p = new URLSearchParams();
	if (v.venture) p.set('venture', v.venture);
	if (v.view === 'archive') p.set('view', 'archive');
	if (v.stage) p.set('stage', v.stage);
	if (v.layout === 'radial') p.set('layout', 'radial');
	if (v.lead) p.set('lead', v.lead);
	const qs = p.toString();
	return qs ? `/os/funnel?${qs}` : '/os/funnel';
}

/** The data query the backend filters on (venture + stage); layout, view and lead are client-only. */
export function funnelQuery(v: FunnelView): string {
	const p = new URLSearchParams();
	if (v.venture) p.set('venture', v.venture);
	if (v.stage) p.set('stage', v.stage);
	const qs = p.toString();
	return qs ? `/pages/funnel?${qs}` : '/pages/funnel';
}
