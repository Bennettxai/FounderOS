/**
 * The FounderOS primary navigation, ported from FounderOS v1 lib/nav.ts (the
 * published demo): same groups, order and labels, every href under /os. The
 * Sidebar renders these groups in order and the CommandPalette derives its
 * digit (1–9) shortcuts from the same visible order, so the two cannot drift.
 *
 * One deliberate change: 'G-Brain' is 'Brain'. G-Brain is gone; the view shows
 * the Optimal Engine.
 */
import type { Component } from 'svelte';
import {
	BarChart3,
	BookOpen,
	Brain,
	Calendar,
	FolderKanban,
	Inbox,
	LayoutDashboard,
	Monitor,
	Settings,
	CandlestickChart,
	Clapperboard,
	Crosshair,
	Filter,
	Gauge,
	Handshake,
	Home,
	Layers,
	LayoutGrid,
	ListChecks,
	Map as MapIcon,
	MessagesSquare,
	MessageSquare,
	Network,
	Plug,
	Share2,
	Sparkles,
	Stethoscope,
	Users,
	Wallet,
	Waypoints,
	Workflow
} from '$lib/founderos/icons';

// lucide-svelte ships legacy class-style typings; the runtime value is a Svelte component.
type Icon = Component<Record<string, unknown>>;
const icon = (c: unknown) => c as Icon;

export type NavItem = { href: string; label: string; icon: Icon };
/** `title` is the sidebar heading; `sub` is the palette's group line (FounderOS v1
 *  heads the last group 'Variants' in the sidebar but 'Library' in the palette). */
export type NavGroup = { title: string; sub: string; items: NavItem[] };

export const OS_ROOT = '/os';

export const NAV_OPERATE: NavItem[] = [
	{ href: '/os', label: 'Home', icon: icon(Home) },
	{ href: '/os/comms', label: 'Comms', icon: icon(MessageSquare) },
	{ href: '/os/funnel', label: 'Funnel', icon: icon(Filter) },
	{ href: '/os/workflows', label: 'Workflows', icon: icon(Workflow) },
	{ href: '/os/social', label: 'Social', icon: icon(Share2) },
	{ href: '/os/content', label: 'Content', icon: icon(Clapperboard) },
	{ href: '/os/brand-deals', label: 'Brand Deals', icon: icon(Handshake) },
	{ href: '/os/finances', label: 'Finances', icon: icon(Wallet) },
	{ href: '/os/trading', label: 'Trading', icon: icon(CandlestickChart) },
	// Last in Operate on purpose, outside the nine digit shortcuts.
	{ href: '/os/adpilot', label: 'AdPilot', icon: icon(Crosshair) }
];

export const NAV_AGENTS: NavItem[] = [
	{ href: '/os/agents', label: 'Agents', icon: icon(Users) },
	{ href: '/os/chats', label: 'Chats', icon: icon(MessagesSquare) },
	{ href: '/os/tasks', label: 'Tasks', icon: icon(ListChecks) },
	{ href: '/os/skills', label: 'Skills', icon: icon(Sparkles) },
	{ href: '/os/org', label: 'Org Chart', icon: icon(Network) },
	// The system drawn from its own registries.
	{ href: '/os/blueprint', label: 'Blueprint', icon: icon(Waypoints) }
];

export const NAV_INTELLIGENCE: NavItem[] = [
	{ href: '/os/brain', label: 'Brain', icon: icon(Brain) },
	{ href: '/os/doctor', label: 'Doctor', icon: icon(Stethoscope) }
];

export const NAV_SYSTEM: NavItem[] = [
	{ href: '/os/integrations', label: 'Connections', icon: icon(Plug) },
	{ href: '/os/usage', label: 'Usage', icon: icon(Gauge) },
	{ href: '/os/roadmap', label: 'Roadmap', icon: icon(MapIcon) },
	{ href: '/os/analytics', label: 'Analytics', icon: icon(BarChart3) },
	{ href: '/os/reference', label: 'Reference Model', icon: icon(LayoutGrid) }
];

export const NAV_LIBRARY: NavItem[] = [{ href: '/os/personas', label: 'Personas', icon: icon(Layers) }];

export const NAV_GROUPS: NavGroup[] = [
	{ title: 'Operate', sub: 'Operate', items: NAV_OPERATE },
	{ title: 'Agents', sub: 'Agents', items: NAV_AGENTS },
	{ title: 'Intelligence', sub: 'Intelligence', items: NAV_INTELLIGENCE },
	{ title: 'System', sub: 'System', items: NAV_SYSTEM },
	{ title: 'Variants', sub: 'Library', items: NAV_LIBRARY }
];

export const NAV_ITEMS: NavItem[] = NAV_GROUPS.flatMap((g) => g.items);

/** BusinessOS primitives kept beside the operator's own groups (sidebar only; the
 *  palette's digit shortcuts stay the operator's). Agents and Tasks are left out:
 *  The operator's own versions are above. */
export const NAV_BUSINESSOS: NavItem[] = [
	{ href: '/dashboard', label: 'Command', icon: icon(LayoutDashboard) },
	{ href: '/knowledge', label: 'Knowledge', icon: icon(BookOpen) },
	{ href: '/inbox', label: 'Inbox', icon: icon(Inbox) },
	{ href: '/calendar', label: 'Calendar', icon: icon(Calendar) },
	{ href: '/communication', label: 'Communications', icon: icon(MessageSquare) },
	{ href: '/projects', label: 'Projects', icon: icon(FolderKanban) },
	{ href: '/window', label: 'Desktop', icon: icon(Monitor) },
	{ href: '/settings', label: 'Settings', icon: icon(Settings) }
];

/** Visible top-to-bottom order across all groups. */
export const NAV_ORDER: string[] = NAV_ITEMS.map((n) => n.href);

/** Digit keys 1–9 jump to the first nine views in visible order. */
export const DIGIT_VIEWS: string[] = NAV_ORDER.slice(0, 9);

/** Home is active only on /os itself; every other item also on its sub-routes. */
export function isActive(href: string, pathname: string): boolean {
	return pathname === href || (href !== OS_ROOT && pathname.startsWith(`${href}/`));
}

export function labelFor(href: string): string | undefined {
	return NAV_ITEMS.find((n) => n.href === href)?.label;
}
