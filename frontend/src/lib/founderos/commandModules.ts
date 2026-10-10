/**
 * The FounderOS v1 section on the Command page (/dashboard): one card per module
 * in the sidebar's "FounderOS" group, in the same order. Hrefs and labels come
 * from the module catalog so the page and the sidebar cannot drift.
 */
import { getEnabledModuleIds, getModuleCatalog } from '$lib/config/workspaceModules';

const DESC: Record<string, string> = {
	'/os/chats': 'Every agent conversation in one hub: threads with the Conductor and each crew, with their replies.',
	'/os/tasks': 'The task board: what each agent owns, what is blocked, and the scheduled jobs that feed it.',
	'/os/roadmap': 'Build phases and the quarter-by-quarter plan for wiring every system into the OS.',
	'/os/reference': 'The reference model: the pillars, systems and domains this operating system is built around.',
	'/os': 'Operator console: pulse row, connections, agent list, what needs you, and operating volume.',
	'/os/comms': 'Unified feed of email, Slack, DMs and messages, plus Plaud and Fathom call recordings.',
	'/os/funnel': 'Client journey for Vantage and Launchpad Cohort: leads, bookings, calls held, payments.',
	'/os/workflows': 'The automations and scheduled runs that move work between agents and tools.',
	'/os/social': 'Zernio growth dashboard: followers, posts and reach across every account.',
	'/os/content': 'Content pipeline, scripts, VSLs and creative in flight.',
	'/os/brand-deals': 'Brand partnerships, deal stages, sent proposals and what each is worth.',
	'/os/finances': 'Stripe, PayKit and bank balances, revenue, spend and the money picture.',
	'/os/trading': 'Robinhood and Phantom: individual and agentic sleeves, open orders, positions, trade log.',
	'/os/clients': 'Every client across both businesses, their stage, payments and last touch.',
	'/os/adpilot': 'Ad campaigns and creative testing, with spend and results.',
	'/os/agents': 'The live agent board and the Conductor, with Tasks, Needs You, Deliverables and Hermes tabs.',
	'/os/skills': 'The skills agents can use, and where each one is wired.',
	'/os/org': 'Org chart: operator, the Conductor, the five pillars and their workers.',
	'/os/brain': 'Optimal Engine memory: search, the knowledge graph and memory health.',
	'/os/doctor': 'Health checks across engines, connectors and services, with fixes.',
	'/os/blueprint': 'How the system fits together: services, data flows and dependencies.',
	'/os/integrations': 'Live Connections board: every connector and its honest status.',
	'/os/usage': 'Token burn across the Claude, ChatGPT and Ollama plans, and the top burners.',
	'/os/analytics': 'Real connector numbers and the snapshots behind the growth charts.',
	'/os/personas': 'Audience personas and the page variants built for them.'
};

const FOUNDEROS = getModuleCatalog().filter((m) => m.group === 'FounderOS');

/** The /os pages switched on for a workspace (its settings; no list = all). */
export function founderosHrefsFor(settings: Record<string, unknown>): Set<string> {
	const enabled = new Set(getEnabledModuleIds(settings));
	return new Set(FOUNDEROS.filter((m) => enabled.has(m.id)).map((m) => m.href));
}

const card = (m: (typeof FOUNDEROS)[number]) => ({ href: m.href, label: m.label, desc: DESC[m.href] ?? '', live: true });

export const FOUNDEROS_COMMAND_GROUP = {
	label: 'FounderOS',
	purpose: 'The FounderOS operator views, reading your connectors and the Optimal Engine.',
	modules: FOUNDEROS.map(card)
};

/** The Command page's FounderOS section for one workspace. */
export function founderosCommandGroup(settings: Record<string, unknown>) {
	const hrefs = founderosHrefsFor(settings);
	return { ...FOUNDEROS_COMMAND_GROUP, modules: FOUNDEROS_COMMAND_GROUP.modules.filter((m) => hrefs.has(m.href)) };
}
