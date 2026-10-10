import { describe, expect, it } from 'vitest';
import {
	bucketPostingByDay,
	dateAxis,
	followerBarModel,
	forwardFill,
	formatFollowers,
	formatPct,
	growthOf,
	inRange,
	pieSlices,
	platformsPresent,
	platformLabel,
	likeToViewRatio,
	averageLikeToView,
	formatRatioPct,
	SAMPLE_POSTS,
	emailChips,
	emailCaption,
	agoFrom,
	matrixCols
} from './lib';

describe('formatting', () => {
	it('followers and growth never fake a zero', () => {
		expect(formatFollowers(null)).toBe('—');
		expect(formatFollowers(53970)).toBe('53,970');
		expect(formatPct(null)).toBe('—');
		expect(formatPct(1.234)).toBe('+1.23%');
		expect(formatPct(-12.5)).toBe('-12.5%');
		expect(platformLabel('twitter')).toBe('X');
		expect(platformLabel('facebook')).toBe('Facebook');
	});
});

describe('posting activity (lib/posting-activity.ts)', () => {
	it('dateAxis is inclusive UTC days', () => {
		expect(dateAxis('2026-09-28', '2026-09-30')).toEqual(['2026-09-28', '2026-09-29', '2026-09-30']);
		expect(dateAxis('2026-09-30', '2026-09-28')).toEqual([]);
	});
	it('buckets cross-posts per platform and drops off-axis posts', () => {
		const axis = ['2026-09-29', '2026-09-30'];
		const days = bucketPostingByDay(
			[
				{ date: '2026-09-30', platforms: ['instagram', 'tiktok'] },
				{ date: '2026-09-30', platforms: ['instagram'] },
				{ date: '2026-01-01', platforms: ['youtube'] }
			],
			axis
		);
		expect(days[1]).toEqual({ date: '2026-09-30', counts: { instagram: 2, tiktok: 1 }, total: 3 });
		expect(days[0].total).toBe(0);
	});
	it('forward-fills and orders platforms', () => {
		expect(forwardFill([{ date: '2026-09-29', value: 5 }], ['2026-09-28', '2026-09-29', '2026-09-30'])).toEqual([null, 5, 5]);
		expect(platformsPresent([{ date: 'x', platforms: ['zzz', 'tiktok', 'instagram'] }], ['instagram', 'tiktok'])).toEqual([
			'instagram',
			'tiktok',
			'zzz'
		]);
	});
});

describe('series range growth (SocialStatStrip)', () => {
	const pts = [
		{ date: '2026-08-01', value: 100 },
		{ date: '2026-09-15', value: 110 },
		{ date: '2026-09-30', value: 121 }
	];
	it('slices a trailing window and measures first→last', () => {
		expect(inRange(pts, 30).map((p) => p.date)).toEqual(['2026-09-15', '2026-09-30']);
		expect(growthOf(inRange(pts, 30))).toBeCloseTo(10);
		expect(growthOf(inRange(pts, 'all'))).toBeCloseTo(21);
		expect(growthOf([pts[0]])).toBeNull();
	});
});

describe('charts (lib/social-chart.ts)', () => {
	it('pie slices drop null/zero channels and close the circle', () => {
		const s = pieSlices([
			{ key: 'a', label: 'A', value: 75 },
			{ key: 'b', label: 'B', value: 25 },
			{ key: 'c', label: 'C', value: null }
		]);
		expect(s.map((x) => x.key)).toEqual(['a', 'b']);
		expect(s[0].startAngle).toBe(-90);
		expect(s[1].endAngle).toBeCloseTo(270);
		expect(pieSlices([])).toEqual([]);
	});
	it('follower bars keep a floor below the minimum and mark deltas', () => {
		const m = followerBarModel([
			{ date: '2026-09-01', followers: 1000 },
			{ date: '2026-09-02', followers: 1100 },
			{ date: '2026-09-03', followers: 1050 }
		]);
		expect(m.bars[0].delta).toBeNull();
		expect(m.bars[1].delta).toBe(100);
		expect(m.bars[2].delta).toBe(-50);
		expect(m.floor).toBeLessThan(1000);
		expect(m.bars.every((b) => b.h > 0 && b.h <= 1)).toBe(true);
		expect(m.bars[0].xLabel).toBe('09-01');
		expect(followerBarModel([]).bars).toEqual([]);
	});
});

describe('engagement (lib/engagement.ts)', () => {
	it('like-to-view is a percentage, null without views', () => {
		expect(likeToViewRatio(1104, 12400)).toBeCloseTo(8.903, 2);
		expect(likeToViewRatio(5, 0)).toBeNull();
		expect(averageLikeToView([{ likes: 1, views: 10 }, { likes: 3, views: 10 }, { likes: 1, views: 0 }])).toBeCloseTo(20);
		expect(averageLikeToView([])).toBeNull();
		expect(formatRatioPct(8.903)).toBe('8.9%');
		expect(formatRatioPct(null)).toBe('—');
	});

	it('the sample posts carry prod copy and an honest average', () => {
		expect(SAMPLE_POSTS).toHaveLength(5);
		expect(SAMPLE_POSTS[0].caption).toBe('3 agents that run my business while I sleep');
		expect(formatRatioPct(averageLikeToView(SAMPLE_POSTS))).toBe('8.6%');
	});
});

describe('newsletter header (app/social/page.tsx)', () => {
	const g = (d7: number | null, d30: number | null) => ({ d7, d30, d60: null, allTime: null });
	it('chips only for windows with a real baseline, toned by sign', () => {
		expect(emailChips(g(-0.14, 21.8))).toEqual([
			{ tone: 'err', text: '-0.14% 7d' },
			{ tone: 'ok', text: '+21.8% 30d' }
		]);
		expect(emailChips(g(null, null))).toEqual([]);
	});
	it('caption names the reading date, or says there is none', () => {
		expect(emailCaption('2026-09-30T12:00:00Z')).toBe('subscribers · Beehiiv reading 2026-09-30');
		expect(emailCaption(null)).toBe('no Beehiiv reading yet · add BEEHIIV_API_KEY');
	});
	it('agoFrom: 2h / 3d, blank for a missing or future stamp', () => {
		const now = Date.parse('2026-09-30T12:00:00Z');
		expect(agoFrom('2026-09-30T10:00:00Z', now)).toBe('2h');
		expect(agoFrom('2026-09-27T12:00:00Z', now)).toBe('3d');
		expect(agoFrom('2026-09-30T11:59:50Z', now)).toBe('1m');
		expect(agoFrom(null, now)).toBe('');
		expect(agoFrom('2026-10-01T00:00:00Z', now)).toBe('');
	});
});

describe('matrixCols (the open-rate DotMatrix labels)', () => {
	it('undoes the backend numbering and keeps keys unique with invisible marks', () => {
		const cols = matrixCols([{ label: 'Sep 9', count: 3 }, { label: 'Sep 9 2', count: 2 }, { label: 'Sep 10', count: 1 }, { label: 'Sep 9', count: 4 }]);
		expect(cols.map((c) => c.label.replace(/​/g, ''))).toEqual(['Sep 9', 'Sep 9', 'Sep 10', 'Sep 9']);
		expect(new Set(cols.map((c) => c.label)).size).toBe(4);
		expect(cols.map((c) => c.count)).toEqual([3, 2, 1, 4]);
		expect(cols[0].label).toBe('Sep 9');
	});
});
