// Intelligence API client - signals/risks (deterministic, from workspace data)
// plus AI-synthesized insights (recommendations), generated on demand and
// cached per workspace. Goes through request<T>() (base url, cookie, CSRF,
// X-Workspace-ID header).
import { request } from "./base";

export interface Signal {
  kind: "signal" | "risk";
  title: string;
  detail: string;
  severity: "low" | "medium" | "high";
  source_type: "task" | "client" | "project";
  count: number;
}

export interface Insight {
  type: "signal" | "risk" | "recommendation";
  title: string;
  detail: string;
  priority: "low" | "medium" | "high";
}

export interface IntelligenceResponse {
  signals: Signal[];
  insights: Insight[];
  ai_model: string;
  ai_generated: string | null;
  ai_available: boolean;
}

export async function getIntelligence(): Promise<IntelligenceResponse> {
  return request<IntelligenceResponse>(`/intelligence`, { skipCache: true });
}

export async function synthesizeIntelligence(): Promise<{
  insights: Insight[];
  ai_model?: string;
  note?: string;
}> {
  return request(`/intelligence/synthesize`, { method: "POST", timeout: 60_000 });
}
