// Usage Analytics API Types

// ── Plan / Limits types ───────────────────────────────────────────────────────

export interface PlanUsage {
  ai_calls_today: number;
  ai_calls_limit: number; // -1 = unlimited
  storage_used_bytes: number;
  storage_limit_bytes: number; // -1 = unlimited
  modules_count: number;
  modules_limit: number; // -1 = unlimited
  team_members: number;
  team_members_limit: number; // -1 = unlimited
  plan: string;
}

export interface PlanLimits {
  plan_name: string;
  ai_calls_per_day: number; // -1 = unlimited
  ai_model_tier: string;
  max_modules: number; // -1 = unlimited
  storage_bytes_limit: number; // -1 = unlimited
  max_team_members: number; // -1 = unlimited
  osa_modes: string[];
  compute_cpu_hours_per_month: number; // -1 = unlimited
}

// ── Legacy analytics types ────────────────────────────────────────────────────

export type UsagePeriod = "today" | "week" | "month" | "year" | "all";

export interface UsageSummary {
  total_requests: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_tokens: number;
  total_cost: number;
  period: string;
  start_date: string;
  end_date: string;
}

export interface ProviderUsage {
  provider: string;
  request_count: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_tokens: number;
  total_cost: number;
}

export interface ModelUsage {
  model: string;
  provider: string;
  request_count: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_tokens: number;
  total_cost: number;
}

export interface AgentUsage {
  agent_name: string;
  request_count: number;
  total_input_tokens: number;
  total_output_tokens: number;
  total_tokens: number;
  avg_duration_ms: number;
}

export interface UsageTrendPoint {
  date: string;
  ai_requests: number;
  total_tokens: number;
  estimated_cost: number;
  mcp_requests: number;
  messages_sent: number;
}

export interface MCPToolUsage {
  tool_name: string;
  server_name: string | null;
  request_count: number;
  success_count: number;
  avg_duration_ms: number;
}
