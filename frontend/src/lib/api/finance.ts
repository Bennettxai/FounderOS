import { request } from "./base";

// Finance is a read layer over the workspace's real revenue data. The backend
// does not expose a dedicated finance/invoices endpoint yet, so financial
// visibility is derived from CRM pipeline deals (the source of every booked and
// pending dollar). When a real billing/invoices endpoint lands, add it here.

export type FinanceDealStatus = "open" | "won" | "lost";

export interface FinanceDeal {
  id: string;
  name: string;
  amount?: number;
  currency?: string;
  status?: FinanceDealStatus;
  stage_name?: string;
  company_name?: string;
  expected_close_date?: string;
  actual_close_date?: string;
  created_at: string;
  updated_at: string;
}

export interface FinanceSummary {
  total_deals: number;
  open_deals: number;
  won_deals: number;
  lost_deals: number;
  open_value: number;
  won_value: number;
  lost_value: number;
  // Present on newer backend versions; derived client-side when absent.
  total_value?: number;
  avg_deal_value?: number;
  win_rate?: number;
}

/**
 * Financial summary for the active workspace: booked (won) revenue, open
 * pipeline value, lost value, and deal counts. Backed by GET /crm/deals/stats.
 */
export async function getFinanceSummary(): Promise<FinanceSummary> {
  return request<FinanceSummary>("/crm/deals/stats", {
    skipAuthRedirect: true,
    skipCache: true,
  });
}

/**
 * Revenue records for the active workspace, optionally filtered by status
 * (won / open / lost). Backed by GET /crm/deals. Normalises the backend's
 * paginated, legacy-array, and legacy-object shapes into a flat list.
 */
export async function getFinanceDeals(filters?: {
  status?: FinanceDealStatus;
  limit?: number;
}): Promise<FinanceDeal[]> {
  const params = new URLSearchParams();
  if (filters?.status) params.set("status", filters.status);
  params.set("limit", String(filters?.limit ?? 200));
  const query = params.toString();

  const raw = await request<{
    data?: FinanceDeal[];
    deals?: FinanceDeal[];
  }>(`/crm/deals${query ? `?${query}` : ""}`, {
    skipAuthRedirect: true,
    skipCache: true,
  });

  if (
    raw &&
    typeof raw === "object" &&
    "data" in raw &&
    Array.isArray(raw.data)
  )
    return raw.data;
  if (
    raw &&
    typeof raw === "object" &&
    "deals" in raw &&
    Array.isArray(raw.deals)
  )
    return raw.deals as FinanceDeal[];
  if (Array.isArray(raw)) return raw as FinanceDeal[];
  return [];
}
