/** GET /api/founderos/pages/skills and /pages/skills/:slug. */
import type { Meter, SeriesPoint, StatChip } from '$lib/founderos/kit';

export type SkillStatus = 'live' | 'learning' | 'planned';

/** FounderOS v1 SkillsGrid SkillCard. */
export type SkillCard = {
	id: string; // slug, also the /pages/skills/:slug key
	name: string;
	group: string;
	kind: 'claude' | 'operator';
	description: string;
	meta: string; // source path or owner: the card footer
	filePath: string;
	status?: SkillStatus;
	markdown?: string; // inline for operator skills; otherwise fetched by id
};

export type SkillsVolume = {
	headline: number;
	counts: { claude: number; user: number; plugin: number; operator: number; live: number; learning: number; planned: number };
	chips: StatChip[];
	caption: string;
	meters: Meter[];
	foot: string;
	groups: SeriesPoint[];
	groupCount: number;
	categories: SeriesPoint[];
	owners: SeriesPoint[];
	unassigned: number;
	insight: { value: number; headline: string; body: string; frac: number | null };
};

export type SkillsBody = {
	cards: SkillCard[];
	sourceNote: string;
	/** Set when the operator skills table could not be read. */
	operatorError: string | null;
	volume: SkillsVolume;
};
