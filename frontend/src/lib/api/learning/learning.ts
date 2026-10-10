// Learning API - Pedro Tasks Integration
import { request } from '../base';
import type {
  FeedbackInput,
  FeedbackEntry,
  PersonalizationProfile,
  BehaviorObservation,
  DetectedPattern,
  Learning
} from './types';

/**
 * Record user feedback on AI outputs
 */
export async function recordFeedback(input: FeedbackInput): Promise<FeedbackEntry> {
  return request<FeedbackEntry>('/learning/feedback', {
    method: 'POST',
    body: input
  });
}

/**
 * Record a user behavior observation
 */
export async function observeBehavior(observation: BehaviorObservation): Promise<{ status: string }> {
  return request<{ status: string }>('/learning/behavior', {
    method: 'POST',
    body: observation
  });
}

/**
 * Get user's personalization profile
 */
export async function getPersonalizationProfile(): Promise<PersonalizationProfile> {
  return request<PersonalizationProfile>('/learning/profile');
}

/**
 * Update user's personalization profile
 */
export async function updatePersonalizationProfile(
  profile: Partial<PersonalizationProfile>
): Promise<{ status: string }> {
  return request<{ status: string }>('/learning/profile', {
    method: 'PUT',
    body: profile
  });
}

/**
 * Refresh profile from detected patterns
 */
export async function refreshProfile(): Promise<{ status: string }> {
  return request<{ status: string }>('/learning/profile/refresh', {
    method: 'POST'
  });
}

/**
 * Detect patterns from user behavior
 */
export async function detectPatterns(): Promise<DetectedPattern[]> {
  return request<DetectedPattern[]>('/learning/patterns');
}

/**
 * Get learnings for a specific context
 */
export async function getLearnings(
  agentType?: string,
  limit: number = 20
): Promise<Learning[]> {
  const params = new URLSearchParams();
  if (agentType) params.append('agent_type', agentType);
  params.append('limit', limit.toString());

  const query = params.toString();
  return request<Learning[]>(`/learning/learnings${query ? `?${query}` : ''}`);
}

/**
 * Mark a learning as applied
 */
export async function applyLearning(
  learningId: string,
  successful: boolean
): Promise<{ status: string }> {
  return request<{ status: string }>(`/learning/learnings/${learningId}/apply`, {
    method: 'POST',
    body: JSON.stringify({ successful })
  });
}
