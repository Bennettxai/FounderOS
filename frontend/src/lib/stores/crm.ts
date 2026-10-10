// CRM Store - Sales Pipeline Management
import { writable } from "svelte/store";
import * as crmApi from "$lib/api/crm";
import type {
  Company,
  Pipeline,
  PipelineStage,
  Deal,
  DealStats,
  CRMActivity,
  CreateCompanyData,
  UpdateCompanyData,
  CreatePipelineData,
  UpdatePipelineData,
  CreateStageData,
  UpdateStageData,
  CreateDealData,
  UpdateDealData,
  CreateActivityData,
  ActivityType,
} from "$lib/api/crm";

export type CRMViewMode = "kanban" | "list" | "table";

interface CRMFilters {
  pipelineId: string | null;
  stageId: string | null;
  status: "open" | "won" | "lost" | null;
  search: string;
}

interface CRMState {
  // Companies
  companies: Company[];
  currentCompany: Company | null;

  // Pipelines & Stages
  pipelines: Pipeline[];
  currentPipeline: Pipeline | null;
  stages: PipelineStage[];

  // Deals
  deals: Deal[];
  currentDeal: Deal | null;
  dealStats: DealStats | null;

  // Activities
  activities: CRMActivity[];

  // UI State
  loading: boolean;
  error: string | null;
  filters: CRMFilters;
  viewMode: CRMViewMode;
}

const emptyState: CRMState = {
  companies: [],
  currentCompany: null,
  pipelines: [],
  currentPipeline: null,
  stages: [],
  deals: [],
  currentDeal: null,
  dealStats: null,
  activities: [],
  loading: false,
  error: null,
  filters: {
    pipelineId: null,
    stageId: null,
    status: null,
    search: "",
  },
  viewMode: "kanban",
};

function createCRMStore() {
  const { subscribe, update } = writable<CRMState>({ ...emptyState });

  return {
    subscribe,

    // ============ Company Methods ============

    async loadCompanies(filters?: {
      industry?: string;
      lifecycle_stage?: string;
    }) {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const response = await crmApi.getCompanies(filters);
        update((s) => ({
          ...s,
          companies: response.companies,
          loading: false,
        }));
      } catch (error) {
        console.error("Failed to load companies:", error);
        update((s) => ({
          ...s,
          loading: false,
          error:
            error instanceof Error ? error.message : "Failed to load companies",
        }));
      }
    },

    async loadCompany(id: string) {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const company = await crmApi.getCompany(id);
        update((s) => ({ ...s, currentCompany: company, loading: false }));
        return company;
      } catch (error) {
        console.error("Failed to load company:", error);
        update((s) => ({
          ...s,
          loading: false,
          error:
            error instanceof Error ? error.message : "Failed to load company",
        }));
        return null;
      }
    },

    async createCompany(data: CreateCompanyData) {
      try {
        const company = await crmApi.createCompany(data);
        update((s) => ({ ...s, companies: [company, ...s.companies] }));
        return company;
      } catch (error) {
        console.error("Failed to create company:", error);
        throw error;
      }
    },

    async updateCompany(id: string, data: UpdateCompanyData) {
      try {
        const company = await crmApi.updateCompany(id, data);
        update((s) => ({
          ...s,
          companies: s.companies.map((c) => (c.id === id ? company : c)),
          currentCompany:
            s.currentCompany?.id === id ? company : s.currentCompany,
        }));
        return company;
      } catch (error) {
        console.error("Failed to update company:", error);
        throw error;
      }
    },

    async deleteCompany(id: string) {
      try {
        await crmApi.deleteCompany(id);
        update((s) => ({
          ...s,
          companies: s.companies.filter((c) => c.id !== id),
          currentCompany: s.currentCompany?.id === id ? null : s.currentCompany,
        }));
      } catch (error) {
        console.error("Failed to delete company:", error);
        throw error;
      }
    },

    async searchCompanies(query: string) {
      try {
        const response = await crmApi.searchCompanies(query);
        return response.companies;
      } catch (error) {
        console.error("Failed to search companies:", error);
        throw error;
      }
    },

    // ============ Pipeline Methods ============

    async loadPipelines() {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const response = await Promise.race([
          crmApi.getPipelines(),
          new Promise<never>((_, reject) =>
            setTimeout(() => reject(new Error("CRM API timeout")), 3000),
          ),
        ]);
        const pipelines = response.pipelines;
        if (pipelines.length === 0) {
          update((s) => ({
            ...s,
            pipelines: [],
            currentPipeline: null,
            stages: [],
            deals: [],
            dealStats: null,
            filters: { ...s.filters, pipelineId: null, stageId: null },
            loading: false,
          }));
          return;
        }
        const fallbackPipeline =
          pipelines.find((p) => p.is_default) || pipelines[0];
        let selectedId = fallbackPipeline.id;
        update((s) => {
          const kept =
            s.currentPipeline &&
            pipelines.find((p) => p.id === s.currentPipeline!.id)
              ? s.currentPipeline
              : fallbackPipeline;
          selectedId = kept.id;
          return { ...s, pipelines, currentPipeline: kept };
        });
        if (selectedId) {
          this.loadPipelineStages(selectedId);
          this.loadDeals({ pipeline_id: selectedId });
          this.loadDealStats(selectedId);
        }
        update((s) => ({ ...s, loading: false }));
      } catch (error) {
        console.error("Failed to load CRM pipelines:", error);
        update((s) => ({
          ...s,
          pipelines: [],
          currentPipeline: null,
          stages: [],
          deals: [],
          dealStats: null,
          loading: false,
          error:
            error instanceof Error ? error.message : "Failed to load pipelines",
        }));
      }
    },

    async createPipeline(data: CreatePipelineData) {
      try {
        const pipeline = await crmApi.createPipeline(data);
        update((s) => ({ ...s, pipelines: [...s.pipelines, pipeline] }));
        return pipeline;
      } catch (error) {
        console.error("Failed to create pipeline:", error);
        throw error;
      }
    },

    async updatePipeline(id: string, data: UpdatePipelineData) {
      try {
        const pipeline = await crmApi.updatePipeline(id, data);
        update((s) => ({
          ...s,
          pipelines: s.pipelines.map((p) => (p.id === id ? pipeline : p)),
          currentPipeline:
            s.currentPipeline?.id === id ? pipeline : s.currentPipeline,
        }));
        return pipeline;
      } catch (error) {
        console.error("Failed to update pipeline:", error);
        throw error;
      }
    },

    async deletePipeline(id: string) {
      try {
        await crmApi.deletePipeline(id);
        update((s) => ({
          ...s,
          pipelines: s.pipelines.filter((p) => p.id !== id),
          currentPipeline:
            s.currentPipeline?.id === id ? null : s.currentPipeline,
        }));
      } catch (error) {
        console.error("Failed to delete pipeline:", error);
        throw error;
      }
    },

    selectPipeline(pipeline: Pipeline) {
      update((s) => ({
        ...s,
        currentPipeline: pipeline,
        filters: { ...s.filters, pipelineId: pipeline.id, stageId: null },
      }));
      this.loadPipelineStages(pipeline.id);
      this.loadDeals({ pipeline_id: pipeline.id });
    },

    // ============ Stage Methods ============

    async loadPipelineStages(pipelineId: string) {
      try {
        const response = await Promise.race([
          crmApi.getPipelineStages(pipelineId),
          new Promise<never>((_, reject) =>
            setTimeout(() => reject(new Error("Stages API timeout")), 3000),
          ),
        ]);
        update((s) => ({
          ...s,
          stages: response.stages.sort((a, b) => a.position - b.position),
        }));
      } catch (error) {
        console.error("Failed to load stages:", error);
        update((s) => ({
          ...s,
          stages: [],
          error:
            error instanceof Error ? error.message : "Failed to load stages",
        }));
      }
    },

    async createStage(pipelineId: string, data: CreateStageData) {
      try {
        const stage = await crmApi.createPipelineStage(pipelineId, data);
        update((s) => ({
          ...s,
          stages: [...s.stages, stage].sort((a, b) => a.position - b.position),
        }));
        return stage;
      } catch (error) {
        console.error("Failed to create stage:", error);
        throw error;
      }
    },

    async updateStage(
      pipelineId: string,
      stageId: string,
      data: UpdateStageData,
    ) {
      try {
        const stage = await crmApi.updatePipelineStage(
          pipelineId,
          stageId,
          data,
        );
        update((s) => ({
          ...s,
          stages: s.stages.map((st) => (st.id === stageId ? stage : st)),
        }));
        return stage;
      } catch (error) {
        console.error("Failed to update stage:", error);
        throw error;
      }
    },

    async deleteStage(pipelineId: string, stageId: string) {
      try {
        await crmApi.deletePipelineStage(pipelineId, stageId);
        update((s) => ({
          ...s,
          stages: s.stages.filter((st) => st.id !== stageId),
        }));
      } catch (error) {
        console.error("Failed to delete stage:", error);
        throw error;
      }
    },

    async reorderStages(
      pipelineId: string,
      stageOrders: { id: string; position: number }[],
    ) {
      try {
        await crmApi.reorderPipelineStages(pipelineId, stageOrders);
        update((s) => ({
          ...s,
          stages: s.stages
            .map((st) => {
              const order = stageOrders.find((o) => o.id === st.id);
              return order ? { ...st, position: order.position } : st;
            })
            .sort((a, b) => a.position - b.position),
        }));
      } catch (error) {
        console.error("Failed to reorder stages:", error);
        throw error;
      }
    },

    // ============ Deal Methods ============

    async loadDeals(filters?: {
      pipeline_id?: string;
      stage_id?: string;
      status?: string;
      owner_id?: string;
    }) {
      try {
        const response = await Promise.race([
          crmApi.getDeals(filters),
          new Promise<never>((_, reject) =>
            setTimeout(() => reject(new Error("Deals API timeout")), 3000),
          ),
        ]);
        update((s) => ({ ...s, deals: response.deals }));
      } catch (error) {
        console.error("Failed to load deals:", error);
        update((s) => ({
          ...s,
          deals: [],
          error: error instanceof Error ? error.message : "Failed to load deals",
        }));
      }
    },

    async loadDeal(id: string) {
      update((s) => ({ ...s, loading: true, error: null }));
      try {
        const deal = await Promise.race([
          crmApi.getDeal(id),
          new Promise<never>((_, reject) =>
            setTimeout(() => reject(new Error("Deal API timeout")), 3000),
          ),
        ]);
        update((s) => ({ ...s, currentDeal: deal, loading: false }));
        // Also load activities for this deal
        this.loadDealActivities(id);
        return deal;
      } catch (error) {
        console.error("Failed to load deal:", error);
        update((s) => ({
          ...s,
          currentDeal: null,
          loading: false,
          error: error instanceof Error ? error.message : "Failed to load deal",
        }));
        return null;
      }
    },

    async createDeal(data: CreateDealData) {
      try {
        const deal = await crmApi.createDeal(data);
        update((s) => ({ ...s, deals: [deal, ...s.deals] }));
        return deal;
      } catch (error) {
        console.error("Failed to create deal:", error);
        throw error;
      }
    },

    async updateDeal(id: string, data: UpdateDealData) {
      try {
        const deal = await crmApi.updateDeal(id, data);
        update((s) => ({
          ...s,
          deals: s.deals.map((d) => (d.id === id ? deal : d)),
          currentDeal: s.currentDeal?.id === id ? deal : s.currentDeal,
        }));
        return deal;
      } catch (error) {
        console.error("Failed to update deal:", error);
        throw error;
      }
    },

    async deleteDeal(id: string) {
      try {
        await crmApi.deleteDeal(id);
        update((s) => ({
          ...s,
          deals: s.deals.filter((d) => d.id !== id),
          currentDeal: s.currentDeal?.id === id ? null : s.currentDeal,
        }));
      } catch (error) {
        console.error("Failed to delete deal:", error);
        throw error;
      }
    },

    async moveDealToStage(dealId: string, stageId: string) {
      try {
        const deal = await crmApi.moveDealToStage(dealId, stageId);
        update((s) => ({
          ...s,
          deals: s.deals.map((d) => (d.id === dealId ? deal : d)),
          currentDeal: s.currentDeal?.id === dealId ? deal : s.currentDeal,
        }));
        return deal;
      } catch (error) {
        console.error("Failed to move deal:", error);
        throw error;
      }
    },

    async updateDealStatus(
      dealId: string,
      status: string,
      lostReason?: string,
    ) {
      try {
        const deal = await crmApi.updateDealStatus(dealId, status, lostReason);
        update((s) => ({
          ...s,
          deals: s.deals.map((d) => (d.id === dealId ? deal : d)),
          currentDeal: s.currentDeal?.id === dealId ? deal : s.currentDeal,
        }));
        return deal;
      } catch (error) {
        console.error("Failed to update deal status:", error);
        throw error;
      }
    },

    async loadDealStats(pipelineId?: string) {
      try {
        const stats = await Promise.race([
          crmApi.getDealStats(pipelineId),
          new Promise<never>((_, reject) =>
            setTimeout(() => reject(new Error("Stats API timeout")), 3000),
          ),
        ]);
        update((s) => ({ ...s, dealStats: stats }));
        return stats;
      } catch (error) {
        console.error("Failed to load deal stats:", error);
        update((s) => ({
          ...s,
          dealStats: null,
          error:
            error instanceof Error ? error.message : "Failed to load deal stats",
        }));
      }
    },

    // ============ Activity Methods ============

    async loadActivities(filters?: {
      activity_type?: string;
      is_completed?: boolean;
    }) {
      try {
        const response = await crmApi.getActivities(filters);
        update((s) => ({ ...s, activities: response.activities }));
      } catch (error) {
        console.error("Failed to load activities:", error);
      }
    },

    async loadDealActivities(dealId: string) {
      try {
        const response = await Promise.race([
          crmApi.getDealActivities(dealId),
          new Promise<never>((_, reject) =>
            setTimeout(() => reject(new Error("Activities timeout")), 3000),
          ),
        ]);
        update((s) => ({ ...s, activities: response.activities }));
      } catch (error) {
        console.error("Failed to load deal activities:", error);
        update((s) => ({ ...s, activities: [] }));
      }
    },

    async createActivity(data: CreateActivityData) {
      try {
        const activity = await crmApi.createActivity(data);
        update((s) => ({ ...s, activities: [activity, ...s.activities] }));
        return activity;
      } catch (error) {
        console.error("Failed to create activity:", error);
        throw error;
      }
    },

    async completeActivity(activityId: string, outcome?: string) {
      try {
        const activity = await crmApi.completeActivity(activityId, outcome);
        update((s) => ({
          ...s,
          activities: s.activities.map((a) =>
            a.id === activityId ? activity : a,
          ),
        }));
        return activity;
      } catch (error) {
        console.error("Failed to complete activity:", error);
        throw error;
      }
    },

    async deleteActivity(activityId: string) {
      try {
        await crmApi.deleteActivity(activityId);
        update((s) => ({
          ...s,
          activities: s.activities.filter((a) => a.id !== activityId),
        }));
      } catch (error) {
        console.error("Failed to delete activity:", error);
        throw error;
      }
    },

    // ============ Filter & View Methods ============

    setFilters(filters: Partial<CRMFilters>) {
      update((s) => ({
        ...s,
        filters: { ...s.filters, ...filters },
      }));
    },

    clearFilters() {
      update((s) => ({
        ...s,
        filters: {
          pipelineId: s.currentPipeline?.id || null,
          stageId: null,
          status: null,
          search: "",
        },
      }));
    },

    setViewMode(mode: CRMViewMode) {
      update((s) => ({ ...s, viewMode: mode }));
    },

    clearCurrentDeal() {
      update((s) => ({ ...s, currentDeal: null, activities: [] }));
    },

    clearCurrentCompany() {
      update((s) => ({ ...s, currentCompany: null }));
    },

    clearError() {
      update((s) => ({ ...s, error: null }));
    },

  };
}

export const crm = createCRMStore();

// ============ UI Helper Constants ============

export const dealStatusColors: Record<string, string> = {
  open: "bg-blue-50 text-blue-700 border-blue-200",
  won: "bg-emerald-50 text-emerald-700 border-emerald-200",
  lost: "bg-red-50 text-red-700 border-red-200",
};

export const dealStatusLabels: Record<string, string> = {
  open: "Open",
  won: "Won",
  lost: "Lost",
};

export const dealPriorityColors: Record<string, string> = {
  low: "bg-gray-100 text-gray-600",
  medium: "bg-blue-50 text-blue-700",
  high: "bg-amber-50 text-amber-700",
  urgent: "bg-red-50 text-red-700",
};

export const dealPriorityLabels: Record<string, string> = {
  low: "Low",
  medium: "Medium",
  high: "High",
  urgent: "Urgent",
};

export const activityTypeColors: Record<ActivityType, string> = {
  call: "bg-blue-50 text-blue-700",
  email: "bg-purple-50 text-purple-700",
  meeting: "bg-green-50 text-green-700",
  demo: "bg-amber-50 text-amber-700",
  note: "bg-gray-50 text-gray-700",
  task: "bg-indigo-50 text-indigo-700",
  lunch: "bg-orange-50 text-orange-700",
  deadline: "bg-red-50 text-red-700",
  other: "bg-gray-100 text-gray-600",
};

export const activityTypeLabels: Record<ActivityType, string> = {
  call: "Call",
  email: "Email",
  meeting: "Meeting",
  demo: "Demo",
  note: "Note",
  task: "Task",
  lunch: "Lunch",
  deadline: "Deadline",
  other: "Other",
};

export const lifecycleStageColors: Record<string, string> = {
  lead: "bg-gray-100 text-gray-700",
  opportunity: "bg-blue-50 text-blue-700",
  customer: "bg-emerald-50 text-emerald-700",
  churned: "bg-red-50 text-red-700",
  partner: "bg-purple-50 text-purple-700",
};

export const lifecycleStageLabels: Record<string, string> = {
  lead: "Lead",
  opportunity: "Opportunity",
  customer: "Customer",
  churned: "Churned",
  partner: "Partner",
};

// Format currency for display
export function formatCurrency(
  amount: number | undefined,
  currency = "USD",
): string {
  if (amount === undefined || amount === null) return "-";
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency,
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(amount);
}

// Format deal probability
export function formatProbability(probability: number | undefined): string {
  if (probability === undefined || probability === null) return "-";
  return `${Math.round(probability)}%`;
}
