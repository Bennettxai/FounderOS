// Learning Store - Personalization and Feedback State Management
import { writable } from 'svelte/store';
import {
	getPersonalizationProfile,
	updatePersonalizationProfile,
	recordFeedback,
	detectPatterns,
	getLearnings,
	type PersonalizationProfile,
	type DetectedPattern,
	type Learning,
	type FeedbackInput,
	type FeedbackEntry
} from '$lib/api/learning';

interface LearningState {
	profile: PersonalizationProfile | null;
	patterns: DetectedPattern[];
	learnings: Learning[];
	feedbackHistory: FeedbackEntry[];
	loading: boolean;
	profileLoading: boolean;
	patternsLoading: boolean;
	error: string | null;
}

function createLearningStore() {
	const { subscribe, update } = writable<LearningState>({
		profile: null,
		patterns: [],
		learnings: [],
		feedbackHistory: [],
		loading: false,
		profileLoading: false,
		patternsLoading: false,
		error: null
	});

	return {
		subscribe,

		/**
		 * Load the user's personalization profile
		 */
		async loadProfile() {
			update((s) => ({ ...s, profileLoading: true, error: null }));
			try {
				const profile = await getPersonalizationProfile();
				update((s) => ({ ...s, profile, profileLoading: false }));
				return profile;
			} catch (error) {
				console.error('Failed to load personalization profile:', error);
				update((s) => ({
					...s,
					profileLoading: false,
					error: error instanceof Error ? error.message : 'Failed to load profile'
				}));
				throw error;
			}
		},

		/**
		 * Update the user's personalization profile
		 */
		async updateProfile(data: Partial<PersonalizationProfile>) {
			update((s) => ({ ...s, profileLoading: true, error: null }));
			try {
				await updatePersonalizationProfile(data);
				update((s) => ({
					...s,
					profile: s.profile ? { ...s.profile, ...data } : null,
					profileLoading: false
				}));
			} catch (error) {
				console.error('Failed to update personalization profile:', error);
				update((s) => ({
					...s,
					profileLoading: false,
					error: error instanceof Error ? error.message : 'Failed to update profile'
				}));
				throw error;
			}
		},

		/**
		 * Record feedback for a message, artifact, or other target
		 */
		async recordFeedback(input: FeedbackInput): Promise<FeedbackEntry> {
			try {
				const entry = await recordFeedback(input);
				update((s) => ({
					...s,
					feedbackHistory: [entry, ...s.feedbackHistory]
				}));
				return entry;
			} catch (error) {
				console.error('Failed to record feedback:', error);
				throw error;
			}
		},

		/**
		 * Detect patterns from user interactions
		 */
		async detectPatterns() {
			update((s) => ({ ...s, patternsLoading: true, error: null }));
			try {
				const patterns = await detectPatterns();
				update((s) => ({ ...s, patterns, patternsLoading: false }));
				return patterns;
			} catch (error) {
				console.error('Failed to detect patterns:', error);
				update((s) => ({
					...s,
					patternsLoading: false,
					error: error instanceof Error ? error.message : 'Failed to detect patterns'
				}));
				throw error;
			}
		},

		/**
		 * Load learnings (optionally filtered by agent type)
		 */
		async loadLearnings(agentType?: string, limit?: number) {
			update((s) => ({ ...s, loading: true, error: null }));
			try {
				const learnings = await getLearnings(agentType, limit);
				update((s) => ({ ...s, learnings, loading: false }));
				return learnings;
			} catch (error) {
				console.error('Failed to load learnings:', error);
				update((s) => ({
					...s,
					loading: false,
					error: error instanceof Error ? error.message : 'Failed to load learnings'
				}));
				throw error;
			}
		},

		/**
		 * Clear error state
		 */
		clearError() {
			update((s) => ({ ...s, error: null }));
		},

		/**
		 * Reset store to initial state
		 */
		reset() {
			update(() => ({
				profile: null,
				patterns: [],
				learnings: [],
				feedbackHistory: [],
				loading: false,
				profileLoading: false,
				patternsLoading: false,
				error: null
			}));
		}
	};
}

export const learning = createLearningStore();
