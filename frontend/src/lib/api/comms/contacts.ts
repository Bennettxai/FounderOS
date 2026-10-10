// Contact autocomplete for the compose recipient field.
// Backed by Ghost's /api/comms/contacts/search when available; otherwise the
// orchestrator can fall back to a local frequency cache built from the user's
// own sent/received emails (built in commsEmailUtils.buildLocalContactIndex).
import { request } from "$lib/api/base";
import type { ContactSuggestion } from "./types";

let contactsSearchAvailable: boolean | null = null;

export async function searchContacts(
  q: string,
  limit = 8,
): Promise<ContactSuggestion[]> {
  if (!q.trim()) return [];
  if (contactsSearchAvailable === false) return [];
  try {
    const params = new URLSearchParams({ q, limit: String(limit) });
    const result = await request<ContactSuggestion[] | { contacts: ContactSuggestion[] }>(
      `/comms/contacts/search?${params.toString()}`,
    );
    contactsSearchAvailable = true;
    return Array.isArray(result) ? result : (result.contacts ?? []);
  } catch (err) {
    if (err instanceof Error && /HTTP 404/.test(err.message)) {
      contactsSearchAvailable = false;
      return [];
    }
    throw err;
  }
}

export function __resetContactsSearchProbe(): void {
  contactsSearchAvailable = null;
}
