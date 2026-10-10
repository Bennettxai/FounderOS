import { request } from "../base";
import type {
  PlanUsage,
  PlanLimits,
  UsageSummary,
  ProviderUsage,
  ModelUsage,
  AgentUsage,
  UsageTrendPoint,
  MCPToolUsage,
} from "./types";

// ── Plan / Limits endpoints ───────────────────────────────────────────────────

export async function getPlanUsage() {
  return request<PlanUsage>("/usage");
}

export async function getPlanLimits() {
  return request<PlanLimits>("/usage/limits");
}

// ── Analytics endpoints ───────────────────────────────────────────────────────

export async function getUsageSummary(
  period: "today" | "week" | "month" | "all" = "month",
) {
  return request<UsageSummary>(`/usage/summary?period=${period}`);
}

export async function getUsageByProvider(
  period: "today" | "week" | "month" | "year" = "month",
) {
  return request<ProviderUsage[]>(`/usage/providers?period=${period}`);
}

export async function getUsageByModel(
  period: "today" | "week" | "month" | "year" = "month",
) {
  return request<ModelUsage[]>(`/usage/models?period=${period}`);
}

export async function getUsageByAgent(
  period: "today" | "week" | "month" | "year" = "month",
) {
  return request<AgentUsage[]>(`/usage/agents?period=${period}`);
}

export async function getUsageTrend() {
  return request<UsageTrendPoint[]>("/usage/trend");
}

export async function getMCPUsage(
  period: "today" | "week" | "month" | "year" = "month",
) {
  return request<MCPToolUsage[]>(`/usage/mcp?period=${period}`);
}
