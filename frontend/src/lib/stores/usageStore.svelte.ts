/**
 * Plan Usage Store
 * Fetches current plan quota data from /api/usage and /api/usage/limits.
 * Provides helpers for percentage calculation, over-limit detection, and
 * byte formatting — all consumed by PlanUsageSection.svelte.
 *
 * Singleton factory pattern — matches dashboardDataStore.svelte.ts conventions.
 */

import { api } from "$lib/api";
import type { PlanUsage, PlanLimits } from "$lib/api/usage/types";

export type UsageMetric = "ai_calls" | "storage" | "modules" | "team_members";

function createUsageStore() {
  // ── State ──────────────────────────────────────────────────────────────────
  let planUsage = $state<PlanUsage | null>(null);
  let planLimits = $state<PlanLimits | null>(null);
  let isLoading = $state(false);
  let error = $state<string | null>(null);

  // ── Load ───────────────────────────────────────────────────────────────────

  async function fetchUsage(): Promise<void> {
    isLoading = true;
    error = null;
    try {
      const [usage, limits] = await Promise.all([
        api.getPlanUsage(),
        api.getPlanLimits(),
      ]);
      planUsage = usage;
      planLimits = limits;
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load plan usage";
    } finally {
      isLoading = false;
    }
  }

  // ── Helpers ────────────────────────────────────────────────────────────────

  /**
   * Returns a 0–100 percentage for the given metric.
   * Returns 0 when the limit is -1 (unlimited) or when data is absent.
   */
  function getUsagePercentage(metric: UsageMetric): number {
    if (!planUsage) return 0;

    switch (metric) {
      case "ai_calls":
        if (planUsage.ai_calls_limit === -1) return 0;
        return Math.min(
          100,
          Math.round(
            (planUsage.ai_calls_today / planUsage.ai_calls_limit) * 100,
          ),
        );
      case "storage":
        if (planUsage.storage_limit_bytes === -1) return 0;
        return Math.min(
          100,
          Math.round(
            (planUsage.storage_used_bytes / planUsage.storage_limit_bytes) *
              100,
          ),
        );
      case "modules":
        if (planUsage.modules_limit === -1) return 0;
        return Math.min(
          100,
          Math.round((planUsage.modules_count / planUsage.modules_limit) * 100),
        );
      case "team_members":
        if (planUsage.team_members_limit === -1) return 0;
        return Math.min(
          100,
          Math.round(
            (planUsage.team_members / planUsage.team_members_limit) * 100,
          ),
        );
      default:
        return 0;
    }
  }

  /**
   * Returns true when the metric has hit its hard limit.
   */
  function isOverLimit(metric: UsageMetric): boolean {
    if (!planUsage) return false;

    switch (metric) {
      case "ai_calls":
        return (
          planUsage.ai_calls_limit !== -1 &&
          planUsage.ai_calls_today >= planUsage.ai_calls_limit
        );
      case "storage":
        return (
          planUsage.storage_limit_bytes !== -1 &&
          planUsage.storage_used_bytes >= planUsage.storage_limit_bytes
        );
      case "modules":
        return (
          planUsage.modules_limit !== -1 &&
          planUsage.modules_count >= planUsage.modules_limit
        );
      case "team_members":
        return (
          planUsage.team_members_limit !== -1 &&
          planUsage.team_members >= planUsage.team_members_limit
        );
      default:
        return false;
    }
  }

  /**
   * Returns true when ANY metric is above the given threshold (default 80%).
   */
  function isAnyMetricAbove(threshold = 80): boolean {
    const metrics: UsageMetric[] = [
      "ai_calls",
      "storage",
      "modules",
      "team_members",
    ];
    return metrics.some((m) => getUsagePercentage(m) >= threshold);
  }

  return {
    get planUsage() {
      return planUsage;
    },
    get planLimits() {
      return planLimits;
    },
    get isLoading() {
      return isLoading;
    },
    get error() {
      return error;
    },
    fetchUsage,
    getUsagePercentage,
    isOverLimit,
    isAnyMetricAbove,
  };
}

// Singleton instance — shared across the usage page tree.
export const usageStore = createUsageStore();
