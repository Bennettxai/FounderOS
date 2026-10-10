// GET /api/founderos/pages/roadmap (Go pages/roadmap): FounderOS v1
// PhaseSchema, RoadmapItemSchema and lib/roadmap.ts PhaseProgress.
export type RoadmapStatus = 'done' | 'now' | 'next' | 'later';

export type Phase = { id: string; number: number; title: string; items: string[] };

export type RoadmapItem = {
	id: string;
	title: string;
	quarter: string;
	status: RoadmapStatus;
	departmentId: string | null;
	description: string;
	phaseId: string | null;
};

export type PhaseProgress = { phase: Phase; items: RoadmapItem[]; done: number; total: number; pct: number };

export type RoadmapBody = {
	phases: PhaseProgress[];
	items: RoadmapItem[];
	departments: Record<string, string>;
	shipped: number;
	total: number;
};

/** PATCH /api/founderos/pages/roadmap: the moved row and the whole board back. */
export type RoadmapPatchBody = { item: RoadmapItem; board: RoadmapBody };
