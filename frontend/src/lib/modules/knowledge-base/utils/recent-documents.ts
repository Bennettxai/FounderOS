export interface RecentDocumentItem {
  id: string;
  title: string;
  openedAt: number;
}

const MAX_RECENT = 30;

function isRecentDocumentItem(value: unknown): value is RecentDocumentItem {
  if (!value || typeof value !== "object") return false;
  const item = value as Partial<RecentDocumentItem>;
  return (
    typeof item.id === "string" &&
    typeof item.title === "string" &&
    typeof item.openedAt === "number"
  );
}

export function readRecentDocuments(
  storage: Storage,
  key: string,
): RecentDocumentItem[] {
  try {
    const parsed = JSON.parse(storage.getItem(key) || "[]");
    return Array.isArray(parsed) ? parsed.filter(isRecentDocumentItem) : [];
  } catch {
    storage.removeItem(key);
    return [];
  }
}

export function addRecentDocument(
  storage: Storage,
  key: string,
  item: Omit<RecentDocumentItem, "openedAt">,
  openedAt = Date.now(),
): RecentDocumentItem[] {
  const stored = readRecentDocuments(storage, key);
  const filtered = stored.filter((recent) => recent.id !== item.id);
  const next = [{ ...item, openedAt }, ...filtered].slice(0, MAX_RECENT);
  storage.setItem(key, JSON.stringify(next));
  return next;
}

export function formatRecentTimeAgo(timestamp: number, now = Date.now()): string {
  const diff = Math.max(0, now - timestamp);
  const mins = Math.floor(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}
