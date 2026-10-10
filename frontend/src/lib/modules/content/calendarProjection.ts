import type { CalendarEvent } from "$lib/api/calendar/types";
import type { ContentItem } from "$lib/api/content";
import { contentWorkstream } from "./model";

export const contentCalendarColors: Record<string, string> = {
  "contentos-linkedin-posts": "#2563eb",
  "contentos-organic-content": "#f59e0b",
  "contentos-paid-ads": "#475569",
  "contentos-trial-reels": "#7c3aed",
};

export type ContentCalendarSource = keyof typeof contentCalendarColors;

export const contentCalendarSources: { id: ContentCalendarSource; label: string }[] = [
  { id: "contentos-linkedin-posts", label: "LinkedIn Posts" },
  { id: "contentos-organic-content", label: "Organic Content" },
  { id: "contentos-paid-ads", label: "Paid Ads" },
  { id: "contentos-trial-reels", label: "Trial Reels" },
];

type EventMeta = {
  label: string;
  userId: keyof typeof contentCalendarColors;
  titlePrefix: string;
  hour: number;
};

function contentRaw(item: ContentItem): string {
  return `${item.category || ""} ${item.theme || ""} ${item.content_type || ""} ${item.title || ""} ${item.channel || ""}`.toLowerCase();
}

export function contentTypeLabel(item: ContentItem): string {
  const raw = contentRaw(item);
  if (raw.includes("story")) return "Instagram Stories";
  if (raw.includes("linkedin")) return "LinkedIn Posts";
  if (raw.includes("paid") || raw.includes("ad ") || raw.includes("ads") || raw.includes("offer")) return "Paid Ads";
  if (raw.includes("trial")) return "Trial Reels";
  return "Organic Content";
}

function eventMeta(item: ContentItem): EventMeta {
  const workstream = contentWorkstream(item);
  if (workstream === "LinkedIn Posts") {
    return { label: "LinkedIn Posts", userId: "contentos-linkedin-posts", titlePrefix: "LinkedIn", hour: 10 };
  }
  if (workstream === "Paid Ads") {
    return { label: "Paid Ads", userId: "contentos-paid-ads", titlePrefix: "Paid Ad", hour: 14 };
  }
  if (workstream === "Trial Reels") {
    return { label: "Trial Reels", userId: "contentos-trial-reels", titlePrefix: "Trial Reel", hour: 15 };
  }
  return { label: "Organic Content", userId: "contentos-organic-content", titlePrefix: "Organic", hour: 15 };
}

export function contentCalendarSource(item: ContentItem): ContentCalendarSource {
  return eventMeta(item).userId;
}

function localDateToIso(date: string, hour: number): string {
  const [year, month, day] = date.split("-").map(Number);
  return new Date(year, (month || 1) - 1, day || 1, hour, 0, 0, 0).toISOString();
}

function calendarKey(item: ContentItem): string {
  const date = item.publish_date || item.due_date || "";
  return [
    date,
    item.client?.trim().toLowerCase() || "",
    item.channel?.trim().toLowerCase() || "",
    item.title?.trim().toLowerCase() || "",
    item.content_type || "",
    item.category?.trim().toLowerCase() || "",
  ].join("|");
}

export function contentProfiles(items: ContentItem[]): string[] {
  return Array.from(
    new Set(items.map((item) => item.client?.trim()).filter((value): value is string => Boolean(value))),
  ).sort((a, b) => a.localeCompare(b));
}

export function filterCalendarContent(items: ContentItem[], profile = "all", sources?: ContentCalendarSource[]): ContentItem[] {
  const sourceSet = sources ? new Set(sources) : null;
  const seen = new Set<string>();
  return items.filter((item) => {
    if (item.status === "archive") return false;
    if (profile !== "all" && item.client?.trim() !== profile) return false;
    if (sourceSet && !sourceSet.has(contentCalendarSource(item))) return false;
    const key = calendarKey(item);
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

export function missedContent(items: ContentItem[], today = new Date()): ContentItem[] {
  const threshold = new Date(today.getFullYear(), today.getMonth(), today.getDate()).getTime();
  return items.filter((item) => {
    if (!item.publish_date || ["posted", "published", "archive"].includes(item.status)) return false;
    const [year, month, day] = item.publish_date.split("-").map(Number);
    return new Date(year, (month || 1) - 1, day || 1).getTime() < threshold;
  });
}

export function contentToCalendarEvents(items: ContentItem[]): CalendarEvent[] {
  return items.flatMap((item) => {
    const events: CalendarEvent[] = [];
    const meta = eventMeta(item);
    const profile = item.client?.trim() || "Workspace";
    const base = {
      user_id: meta.userId,
      google_event_id: null,
      calendar_id: null,
      title: item.title || "Untitled content",
      description: `<strong>${meta.label}</strong><br>${profile}${item.channel ? ` · ${item.channel}` : ""}${item.hook ? `<br><br>${item.hook}` : ""}${item.caption ? `<br><br><strong>Caption</strong><br>${item.caption}` : ""}`,
      all_day: false,
      location: item.channel || null,
      attendees: [],
      status: item.status,
      visibility: null,
      html_link: null,
      source: "businessos" as const,
      meeting_type: "other" as const,
      context_id: item.id,
      project_id: null,
      client_id: item.client || null,
      recording_url: null,
      meeting_link: null,
      external_links: item.link ? [{ name: "Live link", url: item.link, type: "content" }] : [],
      meeting_notes: item.notes || null,
      meeting_summary: null,
      action_items: [],
      synced_at: null,
      created_at: item.created_at,
      updated_at: item.updated_at,
    };

    if (item.publish_date) {
      events.push({
        ...base,
        id: `content-${meta.titlePrefix.toLowerCase()}-${item.id}`,
        title: `${meta.titlePrefix}: ${item.title}`,
        start_time: localDateToIso(item.publish_date, meta.hour),
        end_time: localDateToIso(item.publish_date, meta.hour + 1),
      });
    }
    if (item.due_date) {
      events.push({
        ...base,
        user_id: meta.userId,
        id: `content-film-${item.id}`,
        title: `Film: ${item.title}`,
        start_time: localDateToIso(item.due_date, 11),
        end_time: localDateToIso(item.due_date, 12),
      });
    }
    return events;
  });
}
