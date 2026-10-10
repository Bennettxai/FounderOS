import { describe, expect, it } from "vitest";
import type { ContentItem } from "$lib/api/content";
import {
  contentProfiles,
  contentToCalendarEvents,
  filterCalendarContent,
  missedContent,
} from "./calendarProjection";

function item(overrides: Partial<ContentItem> = {}): ContentItem {
  return {
    id: "content-1",
    title: "Launch reel",
    content_type: "reel",
    status: "scheduled",
    hook: "A useful hook",
    body: "",
    caption: "Caption",
    cta: "",
    channel: "Instagram",
    link: "",
    category: "Organic Content",
    theme: "",
    client: "Client Alpha",
    campaign: "",
    owner: "",
    editor: "",
    priority: "normal",
    due_date: "2026-07-11",
    publish_date: "2026-07-12",
    asset_link: "",
    review_link: "",
    revision_notes: "",
    notes: "",
    views: 0,
    reach: 0,
    likes: 0,
    comments: 0,
    saves: 0,
    shares: 0,
    reposts: 0,
    follows: 0,
    profile_activity: 0,
    accounts_engaged: 0,
    avg_watch_time_seconds: 0,
    retention_rate: 0,
    analytics_notes: "",
    created_by: null,
    created_at: "2026-07-10T00:00:00Z",
    updated_at: "2026-07-10T00:00:00Z",
    ...overrides,
  };
}

describe("content calendar projection", () => {
  it("derives profiles only from workspace content", () => {
    expect(contentProfiles([item(), item({ id: "2", client: "Client Beta" }), item({ id: "3", client: "" })])).toEqual([
      "Client Alpha",
      "Client Beta",
    ]);
  });

  it("removes archived and duplicate calendar records", () => {
    const duplicate = item({ id: "duplicate" });
    expect(filterCalendarContent([item(), duplicate, item({ id: "archived", status: "archive" })])).toHaveLength(1);
  });

  it("projects publish and filming dates without a hardcoded person", () => {
    const events = contentToCalendarEvents([item({ client: "" })]);
    expect(events).toHaveLength(2);
    expect(events[0].description).toContain("Workspace");
    expect(events[0].description).not.toContain("Robert");
    expect(events[1].id).toBe("content-film-content-1");
  });

  it("identifies overdue unpublished content", () => {
    const today = new Date(2026, 6, 10);
    expect(missedContent([item({ publish_date: "2026-07-09" })], today)).toHaveLength(1);
    expect(missedContent([item({ publish_date: "2026-07-09", status: "posted" })], today)).toHaveLength(0);
  });
});
