import { request } from "../base";
import { withOAuthRedirect } from "$lib/utils/platform";
import type {
  CalendarEvent,
  CreateCalendarEventData,
  UpdateCalendarEventData,
  ScheduleRequest,
  ScheduleProposal,
  TaskSuggestion,
  TaskSuggestionsResponse,
} from "./types";

// ============================================
// Calendar API - Uses tool-specific OAuth (calendar scopes only)
// Routes under /integrations/google_calendar/* for isolated OAuth
// ============================================

// Google OAuth / sync routes (isolated calendar scopes).
const CALENDAR_BASE = "/integrations/google_calendar";
// Native event store. Events work standalone here; Google/Outlook sync INTO it.
const EVENTS_BASE = "/calendar";

export async function getCalendarEvents(filters?: {
  start?: string;
  end?: string;
  meetingType?: string;
  contextId?: string;
  projectId?: string;
  clientId?: string;
}) {
  const params = new URLSearchParams();
  if (filters?.start) params.set("start", filters.start);
  if (filters?.end) params.set("end", filters.end);
  if (filters?.meetingType) params.set("meeting_type", filters.meetingType);
  if (filters?.contextId) params.set("context_id", filters.contextId);
  if (filters?.projectId) params.set("project_id", filters.projectId);
  if (filters?.clientId) params.set("client_id", filters.clientId);
  const query = params.toString();
  return request<CalendarEvent[]>(
    `${EVENTS_BASE}/events${query ? `?${query}` : ""}`,
  );
}

export async function getCalendarEvent(id: string) {
  return request<CalendarEvent>(`${EVENTS_BASE}/events/${id}`);
}

export async function createCalendarEvent(data: CreateCalendarEventData) {
  return request<CalendarEvent>(`${EVENTS_BASE}/events`, {
    method: "POST",
    body: data,
  });
}

export async function updateCalendarEvent(
  id: string,
  data: UpdateCalendarEventData,
) {
  return request<CalendarEvent>(`${EVENTS_BASE}/events/${id}`, {
    method: "PUT",
    body: data,
  });
}

export async function deleteCalendarEvent(id: string) {
  return request(`${EVENTS_BASE}/events/${id}`, { method: "DELETE" });
}

export async function syncCalendar(): Promise<{
  message: string;
  synced_count: number;
}> {
  return request<{ message: string; synced_count: number }>(
    `${CALENDAR_BASE}/sync`,
    { method: "POST" },
  );
}

export async function getTodayEvents() {
  return request<CalendarEvent[]>(`${EVENTS_BASE}/today`);
}

export async function getUpcomingEvents(limit?: number) {
  const params = new URLSearchParams();
  if (limit) params.set("limit", String(limit));
  const query = params.toString();
  return request<CalendarEvent[]>(
    `${EVENTS_BASE}/upcoming${query ? `?${query}` : ""}`,
  );
}

// Calendar-specific OAuth functions (isolated scopes)
export async function getCalendarConnectionStatus() {
  return request<{
    connected: boolean;
    authorized: boolean;
    email?: string;
    last_sync?: string | null;
    scopes?: string[];
  }>(
    `${CALENDAR_BASE}/status`,
  );
}

export async function getCalendarAuthUrl() {
  return request<{ auth_url: string }>(
    withOAuthRedirect(`${CALENDAR_BASE}/auth`),
  );
}

export async function disconnectCalendar() {
  return request(`${CALENDAR_BASE}/disconnect`, { method: "POST" });
}

// ============================================
// AI-Assisted Scheduling API
// ============================================

export async function proposeSchedule(
  scheduleRequest: ScheduleRequest,
): Promise<ScheduleProposal> {
  return request<ScheduleProposal>(`${CALENDAR_BASE}/schedule/propose`, {
    method: "POST",
    body: scheduleRequest,
  });
}

export async function createEventFromProposal(
  scheduleRequest: ScheduleRequest,
  selectedSlot: { start: string; end: string },
): Promise<CalendarEvent> {
  return request<CalendarEvent>(`${CALENDAR_BASE}/schedule/create`, {
    method: "POST",
    body: {
      title: scheduleRequest.title,
      description: scheduleRequest.description,
      start_time: selectedSlot.start,
      end_time: selectedSlot.end,
      attendees: scheduleRequest.attendees.map((email) => ({ email })),
      meeting_type: scheduleRequest.meeting_type,
      project_id: scheduleRequest.project_id,
      client_id: scheduleRequest.client_id,
    },
  });
}

export async function getTaskSuggestions(
  contextId?: string,
  projectId?: string,
): Promise<TaskSuggestionsResponse> {
  const params = new URLSearchParams();
  if (contextId) params.set("context_id", contextId);
  if (projectId) params.set("project_id", projectId);
  const query = params.toString();
  const response = await request<
    Partial<TaskSuggestionsResponse> & { suggestions?: TaskSuggestion[] }
  >(`/calendar/task-suggestions${query ? `?${query}` : ""}`);
  const generatedAt = response.generated_at ?? new Date().toISOString();

  return {
    suggestions: (response.suggestions ?? []).map(normalizeTaskSuggestion),
    generated_at: generatedAt,
    events_analyzed: response.events_analyzed ?? 0,
    analysis_period: response.analysis_period ?? {
      start: generatedAt,
      end: generatedAt,
    },
  };
}

function normalizeTaskSuggestion(suggestion: TaskSuggestion): TaskSuggestion {
  return {
    ...suggestion,
    related_event_title:
      suggestion.related_event_title ?? suggestion.related_event ?? "",
    suggested_due_date:
      suggestion.suggested_due_date ?? suggestion.suggested_due,
    description: suggestion.description ?? suggestion.reason,
    confidence: suggestion.confidence ?? 0.85,
  };
}
