// Campaigns API client - the per-workspace marketing/outreach campaign registry.
// Goes through request<T>() which handles the API base, session cookie, CSRF,
// and the X-Workspace-ID header.
import { request } from "./base";

export interface Campaign {
  id: string;
  name: string;
  channel: string;
  status: string;
  hook: string;
  description: string;
  cta: string;
  start_date: string | null;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface CampaignInput {
  name: string;
  channel: string;
  status: string;
  hook: string;
  description: string;
  cta: string;
  start_date?: string | null;
}

export async function getCampaigns(
  q?: string,
): Promise<{ campaigns: Campaign[]; count: number }> {
  const query = q ? `?q=${encodeURIComponent(q)}` : "";
  return request<{ campaigns: Campaign[]; count: number }>(
    `/campaigns${query}`,
    { skipCache: true },
  );
}

export function createCampaign(body: CampaignInput): Promise<Campaign> {
  return request<Campaign>(`/campaigns`, { method: "POST", body });
}

export function updateCampaign(
  id: string,
  body: CampaignInput,
): Promise<Campaign> {
  return request<Campaign>(`/campaigns/${id}`, { method: "PUT", body });
}

export function deleteCampaign(id: string): Promise<unknown> {
  return request(`/campaigns/${id}`, { method: "DELETE" });
}
