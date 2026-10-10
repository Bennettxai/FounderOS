// Learning API Types - Pedro Tasks Integration

export type FeedbackTargetType = 'message' | 'artifact' | 'memory' | 'suggestion' | 'agent_response';
export type FeedbackType = 'thumbs_up' | 'thumbs_down' | 'correction' | 'comment' | 'rating';
export type TonePreference = 'formal' | 'casual' | 'professional' | 'friendly';
export type VerbosityPreference = 'concise' | 'balanced' | 'detailed';
export type FormatPreference = 'prose' | 'bullets' | 'structured' | 'mixed';

export interface FeedbackInput {
  target_type: FeedbackTargetType;
  target_id: string;
  feedback_type: FeedbackType;
  feedback_value?: string;
  rating?: number;
  conversation_id?: string;
  agent_type?: string;
  focus_mode?: string;
  original_content?: string;
  expected_content?: string;
}

export interface FeedbackEntry {
  id: string;
  user_id: string;
  target_type: FeedbackTargetType;
  target_id: string;
  feedback_type: FeedbackType;
  feedback_value?: string;
  rating?: number;
  conversation_id?: string;
  agent_type?: string;
  focus_mode?: string;
  original_content?: string;
  expected_content?: string;
  was_processed: boolean;
  processed_at?: string;
  resulting_learning_id?: string;
  created_at: string;
}

export interface PersonalizationProfile {
  id?: string;
  user_id: string;
  preferred_tone: TonePreference;
  preferred_verbosity: VerbosityPreference;
  preferred_format: FormatPreference;
  prefers_examples: boolean;
  prefers_analogies: boolean;
  prefers_code_samples: boolean;
  prefers_visual_aids: boolean;
  expertise_areas: string[];
  learning_areas: string[];
  common_topics: string[];
  timezone?: string;
  preferred_working_hours?: {
    start: string;
    end: string;
    days: number[];
  };
  most_active_hours: number[];
  total_conversations: number;
  total_feedback_given: number;
  positive_feedback_ratio: number;
  profile_completeness: number;
  last_profile_update?: string;
  created_at?: string;
  updated_at?: string;
}

export interface BehaviorObservation {
  pattern_type: string;
  pattern_key: string;
  pattern_value: string;
}

export interface DetectedPattern {
  id: string;
  user_id: string;
  pattern_type: string;
  pattern_key: string;
  pattern_value: string;
  pattern_description?: string;
  observation_count: number;
  confidence_score: number;
  first_observed_at: string;
  last_observed_at: string;
  is_applied: boolean;
  applied_in_prompt: boolean;
  is_active: boolean;
}

export interface Learning {
  id: string;
  user_id: string;
  learning_type: string;
  learning_content: string;
  learning_summary?: string;
  source_type: string;
  source_id?: string;
  source_context?: string;
  confidence_score: number;
  times_applied: number;
  last_applied_at?: string;
  successful_applications: number;
  created_memory_id?: string;
  created_fact_key?: string;
  category?: string;
  tags: string[];
  was_validated: boolean;
  validated_at?: string;
  validation_result?: string;
  validation_notes?: string;
  is_active: boolean;
  superseded_by?: string;
  created_at: string;
  updated_at: string;
}

export interface LearningState {
  profile: PersonalizationProfile | null;
  patterns: DetectedPattern[];
  learnings: Learning[];
  feedbackHistory: FeedbackEntry[];
  loading: boolean;
  error: string | null;
}
