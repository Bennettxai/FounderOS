/** GET /api/founderos/pages/personas (FounderOS v1 PersonaSchema). */
export type PersonaPillar = { name: string; focus: string; agents: string[] };

export type Persona = {
	id: string;
	order: number;
	name: string;
	archetype: string;
	tagline: string;
	summary: string;
	accent: string;
	northStar: string;
	pillars: PersonaPillar[];
	connectors: string[];
	metrics: string[];
	brainUse: string;
	signaturePlay: string;
};

export type PersonasBody = { personas: Persona[] };
