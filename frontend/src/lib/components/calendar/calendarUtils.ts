import type { CalendarEvent, MeetingType } from "$lib/api";

export type ViewMode = "day" | "week" | "month" | "agenda";

export interface SyncStats {
  totalEvents: number;
  googleEvents: number;
  localEvents: number;
  dateRange: { from: string | null; to: string | null } | null;
  lastSync: string | null;
}

export interface EventFormData {
  title: string;
  description: string;
  start_date: string;
  start_time: string;
  end_date: string;
  end_time: string;
  all_day: boolean;
  location: string;
  meeting_type: MeetingType | "";
  meeting_link: string;
}

export const weekDays = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
export const hours = Array.from({ length: 24 }, (_, i) => i);

export function getWeekDays(weekStart = 0): string[] {
  return weekDays.map((_, index) => weekDays[(index + weekStart) % 7]);
}

/**
 * Meeting type color definitions using raw hex values.
 * These are applied as inline CSS custom properties (--cal-ev-color)
 * to avoid Tailwind color class dependencies.
 */
export const meetingTypeColorValues: Record<string, string> = {
  team: "#3b82f6",
  sales: "#10b981",
  client: "#8b5cf6",
  onboarding: "#eab308",
  kickoff: "#f97316",
  implementation: "#06b6d4",
  standup: "#6366f1",
  planning: "#ec4899",
  review: "#14b8a6",
  one_on_one: "#f43f5e",
  retrospective: "#f59e0b",
  internal: "#64748b",
  external: "#059669",
  other: "#6b7280",
  default: "#3b82f6",
};

export function getEventColor(event: CalendarEvent): string {
  if (event.color_hex && /^#[0-9a-f]{6}$/i.test(event.color_hex)) {
    return event.color_hex;
  }
  const type = event.meeting_type || "default";
  if (type === "other" && event.source === "google") return "#34a853";
  return meetingTypeColorValues[type] || meetingTypeColorValues.default;
}

/**
 * @deprecated Use getEventColor() + inline styles instead.
 * Kept for backward compat during migration.
 */
export const meetingTypeColors: Record<
  string,
  { bg: string; border: string; text: string }
> = {
  team: { bg: "bg-blue-100", border: "border-blue-300", text: "text-blue-800" },
  sales: {
    bg: "bg-green-100",
    border: "border-green-300",
    text: "text-green-800",
  },
  client: {
    bg: "bg-purple-100",
    border: "border-purple-300",
    text: "text-purple-800",
  },
  onboarding: {
    bg: "bg-yellow-100",
    border: "border-yellow-300",
    text: "text-yellow-800",
  },
  kickoff: {
    bg: "bg-orange-100",
    border: "border-orange-300",
    text: "text-orange-800",
  },
  implementation: {
    bg: "bg-cyan-100",
    border: "border-cyan-300",
    text: "text-cyan-800",
  },
  standup: {
    bg: "bg-indigo-100",
    border: "border-indigo-300",
    text: "text-indigo-800",
  },
  planning: {
    bg: "bg-pink-100",
    border: "border-pink-300",
    text: "text-pink-800",
  },
  review: {
    bg: "bg-teal-100",
    border: "border-teal-300",
    text: "text-teal-800",
  },
  one_on_one: {
    bg: "bg-rose-100",
    border: "border-rose-300",
    text: "text-rose-800",
  },
  retrospective: {
    bg: "bg-amber-100",
    border: "border-amber-300",
    text: "text-amber-800",
  },
  internal: {
    bg: "bg-slate-100",
    border: "border-slate-300",
    text: "text-slate-800",
  },
  external: {
    bg: "bg-emerald-100",
    border: "border-emerald-300",
    text: "text-emerald-800",
  },
  other: {
    bg: "bg-gray-100",
    border: "border-gray-300",
    text: "text-gray-800",
  },
  default: {
    bg: "bg-blue-50",
    border: "border-blue-200",
    text: "text-blue-700",
  },
};

export function getEventColors(event: CalendarEvent): {
  bg: string;
  border: string;
  text: string;
} {
  const type = event.meeting_type || "default";
  return meetingTypeColors[type] || meetingTypeColors.default;
}

export function isToday(date: Date): boolean {
  const today = new Date();
  return (
    date.getFullYear() === today.getFullYear() &&
    date.getMonth() === today.getMonth() &&
    date.getDate() === today.getDate()
  );
}

export function formatHour(hour: number): string {
  if (hour === 0) return "12 AM";
  if (hour === 12) return "12 PM";
  return hour > 12 ? `${hour - 12} PM` : `${hour} AM`;
}

/**
 * Format an hour-of-day label respecting the workspace time format preference.
 * 24h => "13:00", 12h => "1 PM" (matches the compact gutter style).
 */
export function formatHourFmt(hour: number, use24h: boolean): string {
  if (use24h) return `${String(hour).padStart(2, "0")}:00`;
  return formatHour(hour);
}

/**
 * Format an event timestamp respecting the workspace time format preference.
 * 24h => "13:00", 12h => "1:00 PM".
 */
export function formatTimeFmt(value: string | Date, use24h: boolean): string {
  try {
    const d = value instanceof Date ? value : new Date(value);
    return d.toLocaleTimeString(undefined, {
      hour: use24h ? "2-digit" : "numeric",
      minute: "2-digit",
      hour12: !use24h,
    });
  } catch {
    return "";
  }
}

/** Strip dangerous HTML, allow basic formatting tags only. */
export function sanitizeHtml(html: string): string {
  if (!html) return "";
  const temp = document.createElement("div");
  temp.innerHTML = html;
  temp
    .querySelectorAll("script, style, iframe, object, embed")
    .forEach((el) => el.remove());
  temp.querySelectorAll("*").forEach((el) => {
    Array.from(el.attributes).forEach((attr) => {
      if (
        attr.name.startsWith("on") ||
        (attr.name === "href" && attr.value.startsWith("javascript:"))
      ) {
        el.removeAttribute(attr.name);
      }
    });
  });
  return temp.innerHTML;
}

function floatingDate(value: string): Date {
  const parsed = new Date(value);
  return new Date(
    parsed.getUTCFullYear(),
    parsed.getUTCMonth(),
    parsed.getUTCDate(),
  );
}

/**
 * Google all-day values are dates, not moments in time. The backend stores
 * their date parts at UTC midnight, so rebuild them as local floating dates
 * before rendering. Timed events remain real instants and use local timezone
 * conversion normally.
 */
export function getEventDisplayStart(event: CalendarEvent): Date {
  return event.all_day
    ? floatingDate(event.start_time)
    : new Date(event.start_time);
}

export function getEventDisplayEnd(event: CalendarEvent): Date {
  return event.all_day
    ? floatingDate(event.end_time)
    : new Date(event.end_time);
}

function startOfLocalDay(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate());
}

function nextLocalDay(date: Date): Date {
  const next = startOfLocalDay(date);
  next.setDate(next.getDate() + 1);
  return next;
}

export function isEventOnDate(event: CalendarEvent, date: Date): boolean {
  const dayStart = startOfLocalDay(date);
  const dayEnd = nextLocalDay(date);
  const eventStart = getEventDisplayStart(event);
  const eventEnd = getEventDisplayEnd(event);

  // Both Google all-day end dates and timed ranges use an exclusive end.
  return eventStart < dayEnd && eventEnd > dayStart;
}

export interface TimedEventSegment {
  start: Date;
  end: Date;
  startMinute: number;
  endMinute: number;
}

/** Return the visible portion of a timed event inside one local calendar day. */
export function getTimedEventSegment(
  event: CalendarEvent,
  date: Date,
): TimedEventSegment | null {
  if (event.all_day) return null;

  const dayStart = startOfLocalDay(date);
  const dayEnd = nextLocalDay(date);
  const eventStart = new Date(event.start_time);
  const eventEnd = new Date(event.end_time);
  if (eventStart >= dayEnd || eventEnd <= dayStart) return null;

  const start = eventStart > dayStart ? eventStart : dayStart;
  const end = eventEnd < dayEnd ? eventEnd : dayEnd;
  const startMinute = start === dayStart ? 0 : start.getHours() * 60 + start.getMinutes();
  const endMinute = end.getTime() === dayEnd.getTime() ? 1440 : end.getHours() * 60 + end.getMinutes();

  return { start, end, startMinute, endMinute };
}

export function getEventsForDate(
  events: CalendarEvent[],
  date: Date,
): CalendarEvent[] {
  return events.filter((event) => isEventOnDate(event, date));
}

export function filterEventsBySelectedMembers(
  events: CalendarEvent[],
  selectedMemberIds: ReadonlySet<string>,
  hasWorkspaceMembers: boolean,
): CalendarEvent[] {
  if (!hasWorkspaceMembers) return events;
  return events.filter((event) => selectedMemberIds.has(event.user_id));
}

export function getEventsForHour(
  events: CalendarEvent[],
  date: Date,
  hour: number,
): CalendarEvent[] {
  return events.filter((event) => {
    if (event.all_day) return false; // all-day events render in their own row
    const eventStart = new Date(event.start_time);
    return (
      eventStart.getFullYear() === date.getFullYear() &&
      eventStart.getMonth() === date.getMonth() &&
      eventStart.getDate() === date.getDate() &&
      eventStart.getHours() === hour
    );
  });
}

export function buildDateRange(
  viewMode: ViewMode,
  currentDate: Date,
): { start: Date; end: Date } {
  const start = new Date(currentDate);
  const end = new Date(currentDate);

  if (viewMode === "day") {
    start.setHours(0, 0, 0, 0);
    end.setHours(23, 59, 59, 999);
  } else if (viewMode === "week") {
    start.setDate(start.getDate() - start.getDay());
    start.setHours(0, 0, 0, 0);
    end.setDate(start.getDate() + 6);
    end.setHours(23, 59, 59, 999);
  } else if (viewMode === "month") {
    start.setDate(1);
    start.setHours(0, 0, 0, 0);
    end.setMonth(end.getMonth() + 1, 0);
    end.setHours(23, 59, 59, 999);
  } else {
    // agenda — 30 days from today
    start.setHours(0, 0, 0, 0);
    end.setDate(end.getDate() + 30);
    end.setHours(23, 59, 59, 999);
  }

  return { start, end };
}

export function buildMonthData(
  currentDate: Date,
  dateRange: { start: Date; end: Date },
  weekStart = 0,
): Date[][] {
  const firstDayOfMonth = (dateRange.start.getDay() - weekStart + 7) % 7;
  const daysInMonth = new Date(dateRange.end).getDate();

  const weeks: Date[][] = [];
  let currentWeek: Date[] = [];

  for (let i = 0; i < firstDayOfMonth; i++) {
    const prevDate = new Date(dateRange.start);
    prevDate.setDate(prevDate.getDate() - (firstDayOfMonth - i));
    currentWeek.push(prevDate);
  }

  for (let day = 1; day <= daysInMonth; day++) {
    const date = new Date(
      currentDate.getFullYear(),
      currentDate.getMonth(),
      day,
    );
    currentWeek.push(date);
    if (currentWeek.length === 7) {
      weeks.push(currentWeek);
      currentWeek = [];
    }
  }

  if (currentWeek.length > 0) {
    const nextMonth = new Date(dateRange.end);
    nextMonth.setDate(nextMonth.getDate() + 1);
    while (currentWeek.length < 7) {
      currentWeek.push(new Date(nextMonth));
      nextMonth.setDate(nextMonth.getDate() + 1);
    }
    weeks.push(currentWeek);
  }

  return weeks;
}
