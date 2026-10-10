// Calendar API Types
// NOTE: GoogleConnectionStatus moved to integrations/types.ts

export type MeetingType = 'team' | 'sales' | 'onboarding' | 'kickoff' | 'implementation' | 'standup' | 'retrospective' | 'planning' | 'review' | 'one_on_one' | 'client' | 'internal' | 'external' | 'other';
export type EventSource = 'google' | 'businessos';

export interface CalendarAttendee {
  email: string;
  name?: string;
  response_status?: string;
}

export interface ExternalLink {
  name: string;
  url: string;
  type?: string;
}

export interface ActionItem {
  id: string;
  text: string;
  completed: boolean;
  assignee_id?: string;
  due_date?: string;
}

export interface CalendarEvent {
  id: string;
  user_id: string;
  google_event_id: string | null;
  calendar_id: string | null;
  title: string | null;
  description: string | null;
  start_time: string;
  end_time: string;
  all_day: boolean;
  location: string | null;
  attendees: CalendarAttendee[];
  status: string | null;
  visibility: string | null;
  html_link: string | null;
  source: EventSource;
  color_id?: string | null;
  color_hex?: string | null;
  meeting_type: MeetingType;
  context_id: string | null;
  project_id: string | null;
  client_id: string | null;
  recording_url: string | null;
  meeting_link: string | null;
  external_links: ExternalLink[];
  meeting_notes: string | null;
  meeting_summary: string | null;
  action_items: ActionItem[];
  synced_at: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreateCalendarEventData {
  title: string;
  description?: string;
  start_time: string;
  end_time: string;
  all_day?: boolean;
  location?: string;
  attendees?: CalendarAttendee[];
  meeting_type?: MeetingType;
  context_id?: string;
  project_id?: string;
  client_id?: string;
  recording_url?: string;
  meeting_link?: string;
  external_links?: ExternalLink[];
  meeting_notes?: string;
  action_items?: ActionItem[];
}

export interface UpdateCalendarEventData {
  title?: string;
  description?: string;
  start_time?: string;
  end_time?: string;
  all_day?: boolean;
  location?: string;
  attendees?: CalendarAttendee[];
  meeting_type?: MeetingType;
  context_id?: string | null;
  project_id?: string | null;
  client_id?: string | null;
  recording_url?: string;
  meeting_link?: string;
  external_links?: ExternalLink[];
  meeting_notes?: string;
  action_items?: ActionItem[];
}

// Scheduling Types (for AI-assisted meeting scheduling)
export interface TimePreferences {
  preferred_start_hour: number;
  preferred_end_hour: number;
  avoid_back_to_back: boolean;
  buffer_minutes: number;
  preferred_days: number[]; // 0-6 (Sunday-Saturday)
}

export interface ScheduleRequest {
  title: string;
  description?: string;
  duration_minutes: number;
  attendees: string[];
  search_start: string;
  search_end: string;
  preferences?: TimePreferences;
  meeting_type?: MeetingType;
  project_id?: string;
  client_id?: string;
}

export interface ProposedSlot {
  start: string;
  end: string;
  score: number;
  reason: string;
}

export interface ScheduleProposal {
  request: ScheduleRequest;
  proposed_slots: ProposedSlot[];
  created_at: string;
}

export type SuggestionType = 'meeting_prep' | 'prep' | 'follow_up' | 'deadline' | 'reminder';
export type SuggestionPriority = 'high' | 'medium' | 'low';

export interface TaskSuggestion {
  type: SuggestionType;
  title: string;
  description: string;
  related_event_id?: string;
  related_event?: string;
  related_event_title: string;
  suggested_due: string;
  suggested_due_date: string;
  priority: SuggestionPriority;
  reason: string;
  confidence: number;
}

export interface TaskSuggestionsResponse {
  suggestions: TaskSuggestion[];
  generated_at: string;
  events_analyzed: number;
  analysis_period: {
    start: string;
    end: string;
  };
}
