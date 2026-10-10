import { describe, expect, it } from 'vitest';
import type { ContentItem } from '$lib/api/content';
import {
	buildContentBoards,
	contentWorkstream,
	normalizeStatus,
	OWNED_CONTENT_STAGES,
	OWNED_CONTENT_WORKSTREAMS,
	profileInScope,
	scopeProfiles,
	UNASSIGNED_PROFILE,
	workspaceMemberLabel
} from './model';

function item(overrides: Partial<ContentItem>): ContentItem {
	return {
		id: 'content-1',
		title: 'A content idea',
		content_type: 'reel',
		status: 'idea',
		hook: '',
		body: '',
		caption: '',
		cta: '',
		channel: '',
		link: '',
		category: '',
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
		analytics_notes: '',
		created_by: null,
		created_at: '2026-07-10T00:00:00Z',
		updated_at: '2026-07-10T00:00:00Z',
		...overrides
	};
}

describe('ContentOS model', () => {
	it('normalizes legacy API statuses into pipeline stages', () => {
		expect(normalizeStatus('draft')).toBe('scripting');
		expect(normalizeStatus('scheduled')).toBe('to_post');
		expect(normalizeStatus('published')).toBe('posted');
	});

	it('preserves workspace-defined workstreams before applying generic inference', () => {
		expect(contentWorkstream(item({ category: 'Customer stories', content_type: 'story' }))).toBe('Customer stories');
		expect(contentWorkstream(item({ title: 'Weekly newsletter', content_type: 'newsletter' }))).toBe('Email / Newsletter');
	});

	it('remaps retired personal-brand workstreams into the active pipeline lanes', () => {
		expect(contentWorkstream(item({ category: 'Instagram Stories', content_type: 'story' }))).toBe('Organic Content');
		expect(contentWorkstream(item({ category: 'Profile and Bio', content_type: 'copywriting' }))).toBe('Organic Content');
		expect(contentWorkstream(item({ category: 'Ads', content_type: 'video' }))).toBe('Paid Ads');
		expect(contentWorkstream(item({ category: 'Trial reels', content_type: 'reel' }))).toBe('Trial Reels');
	});

	it('builds boards only from workspace records and keeps unassigned content visible', () => {
		const boards = buildContentBoards([
			item({ id: '1', client: 'Northstar', category: 'Founder content' }),
			item({ id: '2', client: 'Northstar', category: 'Founder content', status: 'to_edit' }),
			item({ id: '3', client: '', category: 'Product updates' })
		]);

		expect(boards.map((board) => board.profile)).toEqual(['Northstar', UNASSIGNED_PROFILE]);
		expect(boards[0].workstreams[0].name).toBe('Founder content');
		expect(boards[0].workstreams[0].columns.find((column) => column.stage === 'to_edit')?.items).toHaveLength(1);
	});

	it('seeds Robert-owned boards with the active content workstreams', () => {
		const boards = buildContentBoards([item({ id: '1', client: 'Robert Potter', category: 'Instagram Stories' })], ['Robert Potter']);
		expect(boards[0].workstreams.map((workstream) => workstream.name)).toEqual([...OWNED_CONTENT_WORKSTREAMS].sort((a, b) => a.localeCompare(b)));
		expect(boards[0].workstreams.find((workstream) => workstream.name === 'Organic Content')?.items).toHaveLength(1);
	});

	it('uses the owned-content stages without client approval lanes', () => {
		const boards = buildContentBoards(
			[item({ id: '1', client: 'Robert Potter', category: 'Organic Content', status: 'approved' })],
			['Robert Potter'],
			OWNED_CONTENT_STAGES
		);
		const columns = boards[0].workstreams.find((workstream) => workstream.name === 'Organic Content')?.columns ?? [];
		expect(columns.map((column) => column.stage)).toEqual(['idea', 'scripting', 'to_film', 'to_edit', 'to_post', 'posted', 'archive']);
		expect(columns.find((column) => column.stage === 'to_post')?.items).toHaveLength(1);
	});

	it('uses the active workspace member profile as the assignment label', () => {
		expect(
			workspaceMemberLabel({
				user_id: 'user-1',
				status: 'active',
				profile: { display_name: 'Alex Editor' }
			} as Parameters<typeof workspaceMemberLabel>[0])
		).toBe('Alex Editor');
	});

	it('uses workspace-owned and client profiles instead of global client seeds', () => {
		expect(scopeProfiles('my', ['Agency MIOSA'], ['BetterStem'])).toEqual(['Agency MIOSA']);
		expect(scopeProfiles('clients', ['Agency MIOSA'], ['BetterStem'])).toEqual(['BetterStem']);
		expect(profileInScope('Agency MIOSA', 'my', ['Agency MIOSA'])).toBe(true);
		expect(profileInScope('BetterStem', 'clients', ['Agency MIOSA'])).toBe(true);
		expect(profileInScope('Agency MIOSA', 'clients', ['Agency MIOSA'])).toBe(false);
	});
});
