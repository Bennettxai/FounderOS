import type { ContentItem, ContentPriority, ContentStatus, ContentType } from '$lib/api/content';
import type { WorkspaceMember } from '$lib/api/workspaces/types';
import {
	CLIENT_CONTENT_PROFILES,
	CONTENT_THEMES,
	CONTENT_THEME_COLORS,
	CONTENT_THEME_DEFINITIONS,
	OWNED_CONTENT_PROFILES,
	OWNED_CONTENT_WORKSTREAMS
} from './workspaceContentDefaults';

export {
	CLIENT_CONTENT_PROFILES,
	CONTENT_THEMES,
	CONTENT_THEME_COLORS,
	CONTENT_THEME_DEFINITIONS,
	OWNED_CONTENT_PROFILES,
	OWNED_CONTENT_WORKSTREAMS
} from './workspaceContentDefaults';
export type { ContentThemeDefinition } from './workspaceContentDefaults';

export type StageId =
	| 'idea'
	| 'scripting'
	| 'to_film'
	| 'to_edit'
	| 'client_review'
	| 'approved'
	| 'to_post'
	| 'posted'
	| 'archive';

export type ContentForm = {
	title: string;
	content_type: ContentType;
	status: StageId;
	hook: string;
	body: string;
	caption: string;
	cta: string;
	channel: string;
	link: string;
	category: string;
	theme: string;
	client: string;
	campaign: string;
	owner: string;
	editor: string;
	priority: ContentPriority;
	due_date: string;
	publish_date: string;
	asset_link: string;
	review_link: string;
	revision_notes: string;
	notes: string;
	views: number;
	reach: number;
	likes: number;
	comments: number;
	saves: number;
	shares: number;
	reposts: number;
	follows: number;
	profile_activity: number;
	accounts_engaged: number;
	avg_watch_time_seconds: number;
	retention_rate: number;
	analytics_notes: string;
};

export type MemberWithProfile = WorkspaceMember & {
	email?: string | null;
	name?: string | null;
	display_name?: string | null;
	profile?: {
		display_name?: string | null;
		work_email?: string | null;
	} | null;
};

export type ContentColumn = { stage: StageId; items: ContentItem[] };
export type ContentWorkstream = { name: string; items: ContentItem[]; columns: ContentColumn[] };
export type ContentBoard = { profile: string; items: ContentItem[]; workstreams: ContentWorkstream[] };
export type ContentScope = 'my' | 'clients';
export type ContentThemeConfig = { name: string; color: string };
export type ContentCardSettings = {
	compact: boolean;
	showHook: boolean;
	showMeta: boolean;
	showLinks: boolean;
	showAnalytics: boolean;
};

export const DEFAULT_WORKSTREAM = 'Organic Content';
export const UNASSIGNED_PROFILE = 'Unassigned profile';

export const DEFAULT_CARD_SETTINGS: ContentCardSettings = {
	compact: false,
	showHook: true,
	showMeta: true,
	showLinks: true,
	showAnalytics: true
};

export function defaultThemeConfig(): ContentThemeConfig[] {
	return CONTENT_THEMES.map((name) => ({ name, color: CONTENT_THEME_COLORS[name] ?? '#0f766e' }));
}

export const OWNED_STRATEGIC_CONTENT_TOPICS = [...CONTENT_THEMES];
export const INITIAL_STAGE_CARD_LIMIT = 3;
export const STAGE_CARD_INCREMENT = 3;

export const STAGES: StageId[] = [
	'idea',
	'scripting',
	'to_film',
	'to_edit',
	'client_review',
	'approved',
	'to_post',
	'posted',
	'archive'
];

export const OWNED_CONTENT_STAGES: StageId[] = STAGES.filter((stage) => stage !== 'client_review' && stage !== 'approved');
export const CLIENT_CONTENT_STAGES: StageId[] = STAGES;

export const STAGE_META: Record<StageId, { label: string; color: string }> = {
	idea: { label: 'Idea', color: '#737373' },
	scripting: { label: 'Scripting', color: '#2f80c0' },
	to_film: { label: 'To Film', color: '#c6534a' },
	to_edit: { label: 'To Edit', color: '#bb4f91' },
	client_review: { label: 'Client Review', color: '#8b5cf6' },
	approved: { label: 'Approved', color: '#16a34a' },
	to_post: { label: 'To Post', color: '#c79a16' },
	posted: { label: 'Posted', color: '#2f8f5b' },
	archive: { label: 'Archive', color: '#7a7a7a' }
};

export const TYPES: ContentType[] = [
	'script',
	'copywriting',
	'carousel',
	'image',
	'video',
	'post',
	'reel',
	'story',
	'newsletter',
	'podcast',
	'thread',
	'article',
	'other'
];

export const CHANNELS = [
	'Instagram',
	'TikTok',
	'YouTube',
	'YouTube Shorts',
	'LinkedIn',
	'Facebook',
	'X',
	'Newsletter',
	'Podcast',
	'Website'
];

export function emptyContentForm(): ContentForm {
	return {
		title: '',
		content_type: 'reel',
		status: 'idea',
		hook: '',
		body: '',
		caption: '',
		cta: '',
		channel: '',
		link: '',
		category: DEFAULT_WORKSTREAM,
		theme: '',
		client: '',
		campaign: '',
		owner: '',
		editor: '',
		priority: 'normal',
		due_date: '',
		publish_date: '',
		asset_link: '',
		review_link: '',
		revision_notes: '',
		notes: '',
		views: 0,
		reach: 0,
		likes: 0,
		comments: 0,
		saves: 0,
		shares: 0,
		reposts: 0,
		follows: 0,
		profile_activity: 0,
		accounts_engaged: 0,
		avg_watch_time_seconds: 0,
		retention_rate: 0,
		analytics_notes: ''
	};
}

export function analyticsEngagement(item: Pick<ContentItem, 'likes' | 'comments' | 'saves' | 'shares' | 'reposts'>): number {
	return (item.likes || 0) + (item.comments || 0) + (item.saves || 0) + (item.shares || 0) + (item.reposts || 0);
}

export function hasAnalytics(item: Pick<ContentItem, 'views' | 'reach' | 'likes' | 'comments' | 'saves' | 'shares' | 'reposts' | 'follows' | 'profile_activity' | 'accounts_engaged'>): boolean {
	return Boolean(
		(item.views || 0) ||
			(item.reach || 0) ||
			(item.likes || 0) ||
			(item.comments || 0) ||
			(item.saves || 0) ||
			(item.shares || 0) ||
			(item.reposts || 0) ||
			(item.follows || 0) ||
			(item.profile_activity || 0) ||
			(item.accounts_engaged || 0)
	);
}

export function formatMetric(value: number): string {
	if (!value) return '0';
	return Intl.NumberFormat('en', { notation: value >= 10000 ? 'compact' : 'standard', maximumFractionDigits: 1 }).format(value);
}

export function normalizeStatus(status: ContentStatus): StageId {
	if (status === 'draft' || (status as string) === 'scripting_priority') return 'scripting';
	if (status === 'scheduled') return 'to_post';
	if (status === 'published') return 'posted';
	if ((status as string) === 'revisions') return 'to_edit';
	if (STAGES.includes(status as StageId)) return status as StageId;
	return 'idea';
}

export function visibleStage(status: ContentStatus, stages: StageId[]): StageId {
	const normalized = normalizeStatus(status);
	if (stages.includes(normalized)) return normalized;
	if ((normalized === 'client_review' || normalized === 'approved') && stages.includes('to_post')) return 'to_post';
	return stages[0] ?? 'idea';
}

export function contentWorkstream(item: Pick<ContentItem, 'category' | 'content_type' | 'title'>): string {
	const category = normalizeWorkstreamName(item.category);
	if (category) return category;
	const raw = `${item.content_type || ''} ${item.title || ''}`.toLowerCase();
	if (raw.includes('trial')) return 'Trial Reels';
	if (raw.includes('ad') || raw.includes('paid')) return 'Paid Ads';
	if (raw.includes('newsletter') || raw.includes('email')) return 'Email / Newsletter';
	if (raw.includes('podcast') || raw.includes('youtube') || raw.includes('long form')) return 'Long Form';
	if (raw.includes('linkedin')) return 'LinkedIn Posts';
	return DEFAULT_WORKSTREAM;
}

export function normalizeWorkstreamName(value: string | null | undefined): string {
	const raw = value?.trim();
	if (!raw) return '';
	const normalized = raw.toLowerCase().replaceAll('&', 'and');
	if (normalized.includes('instagram stories') || normalized === 'stories') return DEFAULT_WORKSTREAM;
	if (normalized.includes('profile') || normalized.includes('bio')) return DEFAULT_WORKSTREAM;
	if (normalized.includes('trial')) return 'Trial Reels';
	if (normalized.includes('paid') || normalized === 'ads' || normalized.includes('ad creative')) return 'Paid Ads';
	if (normalized.includes('linkedin')) return 'LinkedIn Posts';
	if (normalized.includes('organic')) return DEFAULT_WORKSTREAM;
	return raw;
}

export function contentProfile(item: Pick<ContentItem, 'client'>): string {
	return item.client?.trim() || UNASSIGNED_PROFILE;
}

export function scopeProfiles(
	scope: ContentScope,
	ownedProfiles: string[] = OWNED_CONTENT_PROFILES,
	clientProfiles: string[] = CLIENT_CONTENT_PROFILES
): string[] {
	return scope === 'clients' ? clientProfiles : ownedProfiles;
}

export function profileInScope(
	profile: string,
	scope: ContentScope,
	ownedProfiles: string[] = OWNED_CONTENT_PROFILES,
	clientProfiles: string[] = CLIENT_CONTENT_PROFILES
): boolean {
	if (scope === 'my') return ownedProfiles.includes(profile);
	if (clientProfiles.length > 0) return clientProfiles.includes(profile);
	return profile !== UNASSIGNED_PROFILE && !ownedProfiles.includes(profile);
}

export function buildContentBoards(
	items: ContentItem[],
	seedProfiles: string[] = [],
	stages: StageId[] = STAGES,
	ownedProfiles: string[] = OWNED_CONTENT_PROFILES
): ContentBoard[] {
	const profiles = unique([...seedProfiles, ...items.map(contentProfile)]);
	return profiles.map((profile) => {
		const profileItems = items.filter((item) => contentProfile(item) === profile);
		const seededWorkstreams = profileInScope(profile, 'my', ownedProfiles) ? OWNED_CONTENT_WORKSTREAMS : [DEFAULT_WORKSTREAM];
		const workstreams = unique([...seededWorkstreams, ...profileItems.map(contentWorkstream)]);
		return {
			profile,
			items: profileItems,
			workstreams: workstreams.map((name) => {
				const workstreamItems = profileItems.filter((item) => contentWorkstream(item) === name);
				return {
					name,
					items: workstreamItems,
					columns: stages.map((stage) => ({
						stage,
						items: workstreamItems.filter((item) => visibleStage(item.status, stages) === stage)
					}))
				};
			})
		};
	});
}

export function workspaceMemberLabel(member: MemberWithProfile): string {
	return (
		member.profile?.display_name ||
		member.display_name ||
		member.name ||
		member.profile?.work_email ||
		member.email ||
		member.user_id
	);
}

export function unique(values: string[]): string[] {
	return Array.from(new Set(values.filter(Boolean))).sort((a, b) => a.localeCompare(b));
}
