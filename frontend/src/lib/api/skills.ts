import { request } from './base';

// ─── Types (matching backend response shapes) ────────────────────────────────

export interface BackendSkill {
	name: string;
	description: string;
	version: string;
	priority: number;
	tools_used: string[];
}

export interface BackendSkillDetail extends BackendSkill {
	content: string;
	references: string[];
}

export interface SkillGroup {
	[group: string]: string[];
}

// ─── API Functions ───────────────────────────────────────────────────────────

/** List all enabled skills — GET /api/v1/skills */
export async function listSkills(): Promise<{ skills: BackendSkill[]; count: number }> {
	return request('/skills');
}

/** Get a specific skill's full content — GET /api/v1/skills/:name */
export async function getSkill(name: string): Promise<BackendSkillDetail> {
	return request(`/skills/${encodeURIComponent(name)}`);
}

/** Validate a skill — GET /api/v1/skills/:name/validate */
export async function validateSkill(name: string): Promise<{ skill: string; valid: boolean; issues: string[] }> {
	return request(`/skills/${encodeURIComponent(name)}/validate`);
}

/** Reload skills from disk — POST /api/v1/skills/reload */
export async function reloadSkills(): Promise<{ message: string; count: number }> {
	return request('/skills/reload', { method: 'POST' });
}

/** Get skill groups — GET /api/v1/skills/groups */
export async function getSkillGroups(): Promise<{ groups: SkillGroup; settings: Record<string, unknown> }> {
	return request('/skills/groups');
}
