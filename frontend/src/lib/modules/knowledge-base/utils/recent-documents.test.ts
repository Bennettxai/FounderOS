import { describe, expect, it } from "vitest";
import {
  addRecentDocument,
  formatRecentTimeAgo,
  readRecentDocuments,
} from "./recent-documents";

function createStorage(): Storage {
  const values = new Map<string, string>();
  return {
    get length() {
      return values.size;
    },
    clear: () => values.clear(),
    getItem: (key) => values.get(key) ?? null,
    key: (index) => Array.from(values.keys())[index] ?? null,
    removeItem: (key) => values.delete(key),
    setItem: (key, value) => values.set(key, value),
  };
}

describe("recent documents", () => {
  it("adds a document to the front and deduplicates existing entries", () => {
    const storage = createStorage();
    addRecentDocument(storage, "recent", { id: "a.md", title: "A" }, 1000);
    const next = addRecentDocument(storage, "recent", { id: "a.md", title: "A updated" }, 2000);

    expect(next).toEqual([{ id: "a.md", title: "A updated", openedAt: 2000 }]);
  });

  it("drops corrupt storage instead of throwing in the route", () => {
    const storage = createStorage();
    storage.setItem("recent", "{bad json");

    expect(readRecentDocuments(storage, "recent")).toEqual([]);
    expect(storage.getItem("recent")).toBeNull();
  });

  it("formats recent timestamps", () => {
    const now = 60 * 60 * 1000;

    expect(formatRecentTimeAgo(now - 30 * 1000, now)).toBe("just now");
    expect(formatRecentTimeAgo(now - 5 * 60 * 1000, now)).toBe("5m ago");
    expect(formatRecentTimeAgo(now - 60 * 60 * 1000, now)).toBe("1h ago");
  });
});
