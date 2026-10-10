// Offers API client - the per-workspace offers catalog (name + price + status +
// what's included). Goes through request<T>() which handles the API base, session
// cookie, CSRF, and the X-Workspace-ID header.
import { request } from "./base";

export type OfferStatus = "active" | "draft" | "archived";
// Where this offer sits in the Growth System Audit ladder.
export type OfferTier = "audit" | "phase-1" | "phase-2" | "lane";

export interface Offer {
  id: string;
  name: string;
  tier: OfferTier;
  price: string;
  status: OfferStatus;
  promise: string;
  description: string;
  includes: string;
  cta: string;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface OfferInput {
  name: string;
  tier: OfferTier;
  price: string;
  status: OfferStatus;
  promise: string;
  description: string;
  includes: string;
  cta: string;
}

export async function getOffers(
  q?: string,
): Promise<{ offers: Offer[]; count: number }> {
  const query = q ? `?q=${encodeURIComponent(q)}` : "";
  return request<{ offers: Offer[]; count: number }>(`/offers${query}`, {
    skipCache: true,
  });
}

export function createOffer(body: OfferInput): Promise<Offer> {
  return request<Offer>(`/offers`, { method: "POST", body });
}

export function updateOffer(id: string, body: OfferInput): Promise<Offer> {
  return request<Offer>(`/offers/${id}`, { method: "PUT", body });
}

export function deleteOffer(id: string): Promise<unknown> {
  return request(`/offers/${id}`, { method: "DELETE" });
}
