import { writable } from "svelte/store";
import { browser } from "$app/environment";
import { getApiBaseUrl, getCSRFToken } from "$lib/api/base";

// ─── Types ───────────────────────────────────────────────────────────────────

export interface Runtime {
  name: string;
  status: "active" | "idle" | "stopped";
  memory_mb: number;
  uptime_seconds: number;
}

export interface Computer {
  id: string;
  status: "provisioning" | "active" | "hibernating" | "stopped" | "error";
  plan: string;
  ram_gb: number;
  cpus: number;
  storage_gb: number;
  storage_used_gb: number;
  domain: string;
  runtimes: Runtime[];
  created_at: string;
  updated_at: string;
}

export interface ComputerMetrics {
  cpu_percent: number;
  ram_used_mb: number;
  ram_total_mb: number;
  storage_used_gb: number;
  storage_total_gb: number;
  network_in_mb: number;
  network_out_mb: number;
  uptime_seconds: number;
}

export interface Subscription {
  plan: string;
  price_monthly: number;
  credits_total: number;
  credits_used: number;
  seats_used: number;
  billing_cycle: string;
  next_billing_at: string;
  status: string;
}

export interface PlanInfo {
  id: string;
  name: string;
  price_monthly: number;
  ram_gb: number;
  cpus: number;
  storage_gb: number;
  credits_included: number;
  features: string[];
}

// ─── Store State ─────────────────────────────────────────────────────────────

interface ComputerState {
  computer: Computer | null;
  metrics: ComputerMetrics | null;
  runtimes: Runtime[];
  subscription: Subscription | null;
  plans: PlanInfo[];
  loading: boolean;
  error: string | null;
}

// ─── Store ───────────────────────────────────────────────────────────────────

function createComputerStore() {
  const initialState: ComputerState = {
    computer: null,
    metrics: null,
    runtimes: [],
    subscription: null,
    plans: [],
    loading: false,
    error: null,
  };

  const { subscribe, update } = writable<ComputerState>(initialState);

  function buildHeaders(includeContentType = false): Record<string, string> {
    const headers: Record<string, string> = {};
    const csrfToken = getCSRFToken();
    if (csrfToken) headers["X-CSRF-Token"] = csrfToken;
    if (includeContentType) headers["Content-Type"] = "application/json";
    return headers;
  }

  return {
    subscribe,

    /** Load the current computer — GET /api/computer */
    async loadComputer(): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/computer`, {
          method: "GET",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(8000),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const computer: Computer = await res.json();

        update((s) => ({ ...s, computer, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load computer";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Load compute metrics — GET /api/computer/metrics */
    async loadMetrics(): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/computer/metrics`, {
          method: "GET",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(8000),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const metrics: ComputerMetrics = await res.json();

        update((s) => ({ ...s, metrics, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load metrics";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Load runtimes — GET /api/computer/runtimes */
    async loadRuntimes(): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/computer/runtimes`, {
          method: "GET",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(8000),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const data: { runtimes?: Runtime[] } = await res.json();
        const runtimes = Array.isArray(data.runtimes) ? data.runtimes : [];

        update((s) => ({ ...s, runtimes, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load runtimes";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Load billing subscription — GET /api/billing/subscription */
    async loadSubscription(): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/billing/subscription`, {
          method: "GET",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(8000),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const subscription: Subscription = await res.json();

        update((s) => ({ ...s, subscription, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load subscription";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Load available plans — GET /api/billing/plans */
    async loadPlans(): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/billing/plans`, {
          method: "GET",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(8000),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const data: { plans?: PlanInfo[] } = await res.json();
        const plans = Array.isArray(data.plans) ? data.plans : [];

        update((s) => ({ ...s, plans, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to load plans";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Provision a new computer — POST /api/computer */
    async createComputer(plan: string): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/computer`, {
          method: "POST",
          headers: buildHeaders(true),
          credentials: "include",
          signal: AbortSignal.timeout(15000),
          body: JSON.stringify({ plan }),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const computer: Computer = await res.json();

        update((s) => ({ ...s, computer, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to create computer";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Upgrade the computer plan — PUT /api/computer/upgrade */
    async upgradeComputer(plan: string): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/computer/upgrade`, {
          method: "PUT",
          headers: buildHeaders(true),
          credentials: "include",
          signal: AbortSignal.timeout(15000),
          body: JSON.stringify({ plan }),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const computer: Computer = await res.json();

        update((s) => ({ ...s, computer, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to upgrade computer";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Delete the computer — DELETE /api/computer */
    async deleteComputer(): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/computer`, {
          method: "DELETE",
          headers: buildHeaders(),
          credentials: "include",
          signal: AbortSignal.timeout(15000),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        update((s) => ({
          ...s,
          computer: null,
          metrics: null,
          runtimes: [],
          loading: false,
        }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to delete computer";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Start a named runtime — POST /api/computer/runtimes/{name}/start */
    async startRuntime(name: string): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(
          `${getApiBaseUrl()}/computer/runtimes/${encodeURIComponent(name)}/start`,
          {
            method: "POST",
            headers: buildHeaders(true),
            credentials: "include",
            signal: AbortSignal.timeout(15000),
            body: JSON.stringify({}),
          },
        );

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        // Refresh runtimes list after state change
        update((s) => ({ ...s, loading: false }));
        await this.loadRuntimes();
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to start runtime";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Stop a named runtime — POST /api/computer/runtimes/{name}/stop */
    async stopRuntime(name: string): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(
          `${getApiBaseUrl()}/computer/runtimes/${encodeURIComponent(name)}/stop`,
          {
            method: "POST",
            headers: buildHeaders(true),
            credentials: "include",
            signal: AbortSignal.timeout(15000),
            body: JSON.stringify({}),
          },
        );

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        // Refresh runtimes list after state change
        update((s) => ({ ...s, loading: false }));
        await this.loadRuntimes();
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to stop runtime";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Subscribe to a billing plan — POST /api/billing/subscribe */
    async subscribePlan(plan: string): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/billing/subscribe`, {
          method: "POST",
          headers: buildHeaders(true),
          credentials: "include",
          signal: AbortSignal.timeout(15000),
          body: JSON.stringify({ plan }),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const subscription: Subscription = await res.json();

        update((s) => ({ ...s, subscription, loading: false }));
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to subscribe";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Purchase additional credits — POST /api/billing/credits/purchase */
    async purchaseCredits(amount: number): Promise<void> {
      if (!browser) return;

      update((s) => ({ ...s, loading: true, error: null }));

      try {
        const res = await fetch(`${getApiBaseUrl()}/billing/credits/purchase`, {
          method: "POST",
          headers: buildHeaders(true),
          credentials: "include",
          signal: AbortSignal.timeout(15000),
          body: JSON.stringify({ amount }),
        });

        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        // Reload subscription to reflect updated credit balance
        update((s) => ({ ...s, loading: false }));
        await this.loadSubscription();
      } catch (err) {
        const message =
          err instanceof Error ? err.message : "Failed to purchase credits";
        update((s) => ({ ...s, loading: false, error: message }));
      }
    },

    /** Dismiss the current error */
    clearError(): void {
      update((s) => ({ ...s, error: null }));
    },
  };
}

export const computerStore = createComputerStore();
