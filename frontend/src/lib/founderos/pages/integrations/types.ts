/** GET /api/founderos/pages/connections: the Connections board (spec 6.10). */
import type { Connection } from '$lib/founderos/connections';
import type { Meter, SeriesPoint, StatChip } from '$lib/founderos/kit';

export type CatalogEntry = {
	slug: string;
	name: string;
	tagline: string;
	category: string;
	connectorId?: string;
	popular?: boolean;
	/** True only when the linked connector answered "connected" on this load. */
	connected: boolean;
	/** Every connect key resolves (planted or in ~/.founderos/.env); never "connected". */
	keySaved: boolean;
	/** Env names Connect asks for; [] = guidance only (local setup). */
	keys: string[];
};

export type CategoryGroup = { name: string; slugs: string[] };

/** One "Browse by category" row: the category's tiles and how many are live. */
export type BrowseCategory = { label: string; count: number; connected: number; entries: CatalogEntry[] };

export type IntegrationsVolume = {
	headline: number;
	counts: { connected: number; notConfigured: number; error: number; total: number };
	chips: StatChip[];
	caption: string;
	meters: Meter[];
	foot: string;
	byCategory: SeriesPoint[];
	topCategory: { name: string; count: number } | null;
	health: SeriesPoint[];
	insight: { value: number; headline: string; body: string; frac: number };
};

export type KeySlot = {
	envVar: string;
	label: string;
	group: string;
	hint?: string;
	connectorId?: string;
	/** A value resolves for this slot. The value itself never leaves the server. */
	present: boolean;
	/** The last-4 mask ("••••ab12"), when the server sends one; never the value. */
	masked?: string;
};

export type OAuthReadiness = {
	slug: string;
	name: string;
	redirectKind: 'any' | 'loopback' | 'https-public';
	consoleUrl: string;
	clientIdEnv: string;
	clientSecretEnv: string;
	appConfigured: boolean;
	connected: boolean;
	expired: boolean;
};

export type ConnectionsBoard = {
	connections: Connection[];
	catalog: CatalogEntry[];
	categories: CategoryGroup[];
	volume: IntegrationsVolume;
	keys: KeySlot[];
	oauth: Record<string, OAuthReadiness>;
};

export type KeyTestResult = { ok: boolean; ms: number; detail?: string };
