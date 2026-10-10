// Shared formatting helpers for the platform admin sections.

export function fmtDate(s: string): string {
  try {
    return new Date(s).toLocaleDateString(undefined, {
      month: "short",
      day: "numeric",
      year: "numeric",
    });
  } catch {
    return "";
  }
}

export function initials(
  name: string | null | undefined,
  email: string,
): string {
  return (name || email || "?").charAt(0).toUpperCase();
}
