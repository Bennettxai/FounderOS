/**
 * Skill types for the SORX skill catalog and decision cards.
 * Schema matches backend: migrations/096_sorx_skills.sql + internal/sorx/types.go
 */

export type SkillTier = 'free' | 'pro' | 'enterprise';
export type SkillTemperature = 'cold' | 'warm' | 'hot';
export type SkillCategory = 'email' | 'messaging' | 'crm' | 'calendar' | 'sync' | 'export' | 'automation';

export interface Skill {
	id: string;
	name: string;
	description: string;
	tier: SkillTier;
	category: SkillCategory;
	enabled: boolean;
	config?: Record<string, unknown>;
}

/** Runtime execution request shown in a decision card */
export interface SkillExecution {
	id: string;
	skill: Skill;
	action: string;
	temperature: SkillTemperature;
	reasoning: string;
	timestamp: Date;
}

// ─── Display Maps ─────────────────────────────────────────────────────────────

export const tierLabels: Record<SkillTier, string> = {
	free: 'Free',
	pro: 'Pro',
	enterprise: 'Enterprise'
};

export const tierColors: Record<SkillTier, string> = {
	free: 'bg-green-100 text-green-700 border-green-200 dark:bg-green-900/30 dark:text-green-300 dark:border-green-800',
	pro: 'bg-blue-100 text-blue-700 border-blue-200 dark:bg-blue-900/30 dark:text-blue-300 dark:border-blue-800',
	enterprise: 'bg-purple-100 text-purple-700 border-purple-200 dark:bg-purple-900/30 dark:text-purple-300 dark:border-purple-800'
};

export const temperatureLabels: Record<SkillTemperature, string> = {
	cold: 'Auto-run',
	warm: 'Confirm',
	hot: 'Approval Required'
};

export const temperatureColors: Record<SkillTemperature, string> = {
	cold: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300',
	warm: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300',
	hot: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
};

export const categoryLabels: Record<SkillCategory, string> = {
	email: 'Email',
	messaging: 'Messaging',
	crm: 'CRM',
	calendar: 'Calendar',
	sync: 'Sync',
	export: 'Export',
	automation: 'Automation'
};

export const categoryColors: Record<SkillCategory, string> = {
	email: 'bg-blue-50 text-blue-600 border-blue-200 dark:bg-blue-900/20 dark:text-blue-400 dark:border-blue-800',
	messaging: 'bg-violet-50 text-violet-600 border-violet-200 dark:bg-violet-900/20 dark:text-violet-400 dark:border-violet-800',
	crm: 'bg-orange-50 text-orange-600 border-orange-200 dark:bg-orange-900/20 dark:text-orange-400 dark:border-orange-800',
	calendar: 'bg-teal-50 text-teal-600 border-teal-200 dark:bg-teal-900/20 dark:text-teal-400 dark:border-teal-800',
	sync: 'bg-cyan-50 text-cyan-600 border-cyan-200 dark:bg-cyan-900/20 dark:text-cyan-400 dark:border-cyan-800',
	export: 'bg-gray-50 text-gray-600 border-gray-200 dark:bg-gray-800 dark:text-gray-400 dark:border-gray-700',
	automation: 'bg-pink-50 text-pink-600 border-pink-200 dark:bg-pink-900/20 dark:text-pink-400 dark:border-pink-800'
};

// ─── Category Icons (SVG path data for inline rendering) ──────────────────────

export const categoryIcons: Record<SkillCategory, string> = {
	email: 'M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z',
	messaging: 'M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z',
	crm: 'M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z',
	calendar: 'M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z',
	sync: 'M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15',
	export: 'M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z',
	automation: 'M13 10V3L4 14h7v7l9-11h-7z'
};
