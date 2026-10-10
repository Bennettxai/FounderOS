// CRM API Client
import { request } from "../base";
import type {
  Company,
  CompaniesResponse,
  CreateCompanyData,
  UpdateCompanyData,
  Pipeline,
  PipelinesResponse,
  CreatePipelineData,
  UpdatePipelineData,
  PipelineStage,
  StagesResponse,
  CreateStageData,
  UpdateStageData,
  Deal,
  DealsResponse,
  DealStats,
  CreateDealData,
  UpdateDealData,
  CRMActivity,
  ActivitiesResponse,
  CreateActivityData,
  ContactsResponse,
  CRMContact,
  LinkContactData,
  ContactCompanyRelation,
} from "./types";

// ============================================================================
// Companies
// ============================================================================

export async function getCompanies(filters?: {
  industry?: string;
  lifecycle_stage?: string;
  limit?: number;
  offset?: number;
}): Promise<CompaniesResponse> {
  const params = new URLSearchParams();
  if (filters?.industry) params.set("industry", filters.industry);
  if (filters?.lifecycle_stage)
    params.set("lifecycle_stage", filters.lifecycle_stage);
  if (filters?.limit) params.set("limit", filters.limit.toString());
  if (filters?.offset) params.set("offset", filters.offset.toString());
  const query = params.toString();
  const raw = await request<{
    data: Company[];
    pagination?: { total_items?: number };
    companies?: Company[];
    count?: number;
  }>(`/crm/companies${query ? `?${query}` : ""}`);
  // Backend returns paginated { data: [...], pagination: {...} } — unwrap it
  if (
    raw &&
    typeof raw === "object" &&
    "data" in raw &&
    Array.isArray(raw.data)
  ) {
    return {
      companies: raw.data,
      count: raw.pagination?.total_items ?? raw.data.length,
    };
  }
  // Already in expected shape (legacy or direct response)
  if (raw && "companies" in raw && Array.isArray(raw.companies)) {
    return raw as CompaniesResponse;
  }
  return { companies: [], count: 0 };
}

export async function getCompany(id: string): Promise<Company> {
  return request<Company>(`/crm/companies/${id}`);
}

export async function createCompany(data: CreateCompanyData): Promise<Company> {
  return request<Company>("/crm/companies", { method: "POST", body: data });
}

export async function updateCompany(
  id: string,
  data: UpdateCompanyData,
): Promise<Company> {
  return request<Company>(`/crm/companies/${id}`, {
    method: "PUT",
    body: data,
  });
}

export async function deleteCompany(id: string): Promise<void> {
  return request<void>(`/crm/companies/${id}`, { method: "DELETE" });
}

export async function searchCompanies(
  query: string,
  limit?: number,
): Promise<CompaniesResponse> {
  const params = new URLSearchParams();
  params.set("q", query);
  if (limit) params.set("limit", limit.toString());
  const raw = await request<{
    data: Company[];
    pagination?: { total_items?: number };
    companies?: Company[];
    count?: number;
  }>(`/crm/companies/search?${params.toString()}`);
  if (
    raw &&
    typeof raw === "object" &&
    "data" in raw &&
    Array.isArray(raw.data)
  ) {
    return {
      companies: raw.data,
      count: raw.pagination?.total_items ?? raw.data.length,
    };
  }
  if (raw && "companies" in raw && Array.isArray(raw.companies)) {
    return raw as CompaniesResponse;
  }
  return { companies: [], count: 0 };
}

// ============================================================================
// Contact-Company Relations
// ============================================================================

export async function getCompanyContacts(
  companyId: string,
): Promise<ContactsResponse> {
  const raw = await request<{
    data: CRMContact[];
    pagination?: { total_items?: number };
    contacts?: CRMContact[];
    count?: number;
  }>(`/crm/companies/${companyId}/contacts`);
  if (
    raw &&
    typeof raw === "object" &&
    "data" in raw &&
    Array.isArray(raw.data)
  ) {
    return {
      contacts: raw.data,
      count: raw.pagination?.total_items ?? raw.data.length,
    };
  }
  if (raw && "contacts" in raw && Array.isArray(raw.contacts)) {
    return raw as ContactsResponse;
  }
  return { contacts: [], count: 0 };
}

export async function linkContactToCompany(
  companyId: string,
  data: LinkContactData,
): Promise<ContactCompanyRelation> {
  return request<ContactCompanyRelation>(
    `/crm/companies/${companyId}/contacts`,
    { method: "POST", body: data },
  );
}

export async function unlinkContactFromCompany(
  companyId: string,
  relationId: string,
): Promise<void> {
  return request<void>(`/crm/companies/${companyId}/contacts/${relationId}`, {
    method: "DELETE",
  });
}

// ============================================================================
// Pipelines
// ============================================================================

export async function getPipelines(): Promise<PipelinesResponse> {
  const raw = await request<{
    data: Pipeline[];
    pagination?: { total_items?: number };
    pipelines?: Pipeline[];
    count?: number;
  }>("/crm/pipelines", { skipAuthRedirect: true, skipCache: true });
  if (
    raw &&
    typeof raw === "object" &&
    "data" in raw &&
    Array.isArray(raw.data)
  ) {
    return {
      pipelines: raw.data,
      count: raw.pagination?.total_items ?? raw.data.length,
    };
  }
  if (raw && "pipelines" in raw && Array.isArray(raw.pipelines)) {
    return raw as PipelinesResponse;
  }
  return { pipelines: [], count: 0 };
}

export async function getPipeline(id: string): Promise<Pipeline> {
  return request<Pipeline>(`/crm/pipelines/${id}`);
}

export async function createPipeline(
  data: CreatePipelineData,
): Promise<Pipeline> {
  return request<Pipeline>("/crm/pipelines", { method: "POST", body: data });
}

export async function updatePipeline(
  id: string,
  data: UpdatePipelineData,
): Promise<Pipeline> {
  return request<Pipeline>(`/crm/pipelines/${id}`, {
    method: "PUT",
    body: data,
  });
}

export async function deletePipeline(id: string): Promise<void> {
  return request<void>(`/crm/pipelines/${id}`, { method: "DELETE" });
}

// ============================================================================
// Pipeline Stages
// ============================================================================

export async function getPipelineStages(
  pipelineId: string,
): Promise<StagesResponse> {
  const raw = await request<{
    data: PipelineStage[];
    pagination?: { total_items?: number };
    stages?: PipelineStage[];
    count?: number;
  }>(`/crm/pipelines/${pipelineId}/stages`, {
    skipAuthRedirect: true,
    skipCache: true,
  });
  if (
    raw &&
    typeof raw === "object" &&
    "data" in raw &&
    Array.isArray(raw.data)
  ) {
    return {
      stages: raw.data,
      count: raw.pagination?.total_items ?? raw.data.length,
    };
  }
  if (raw && "stages" in raw && Array.isArray(raw.stages)) {
    return raw as StagesResponse;
  }
  return { stages: [], count: 0 };
}

export async function createPipelineStage(
  pipelineId: string,
  data: CreateStageData,
): Promise<PipelineStage> {
  return request<PipelineStage>(`/crm/pipelines/${pipelineId}/stages`, {
    method: "POST",
    body: data,
  });
}

export async function updatePipelineStage(
  pipelineId: string,
  stageId: string,
  data: UpdateStageData,
): Promise<PipelineStage> {
  return request<PipelineStage>(
    `/crm/pipelines/${pipelineId}/stages/${stageId}`,
    { method: "PUT", body: data },
  );
}

export async function deletePipelineStage(
  pipelineId: string,
  stageId: string,
): Promise<void> {
  return request<void>(`/crm/pipelines/${pipelineId}/stages/${stageId}`, {
    method: "DELETE",
  });
}

export async function reorderPipelineStages(
  pipelineId: string,
  stageOrders: { id: string; position: number }[],
): Promise<void> {
  return request<void>(`/crm/pipelines/${pipelineId}/stages/reorder`, {
    method: "POST",
    body: { stage_orders: stageOrders },
  });
}

// ============================================================================
// Deals
// ============================================================================

export async function getDeals(filters?: {
  pipeline_id?: string;
  stage_id?: string;
  status?: string;
  owner_id?: string;
  limit?: number;
  offset?: number;
}): Promise<DealsResponse> {
  const params = new URLSearchParams();
  if (filters?.pipeline_id) params.set("pipeline_id", filters.pipeline_id);
  if (filters?.stage_id) params.set("stage_id", filters.stage_id);
  if (filters?.status) params.set("status", filters.status);
  if (filters?.owner_id) params.set("owner_id", filters.owner_id);
  if (filters?.limit) params.set("limit", filters.limit.toString());
  if (filters?.offset) params.set("offset", filters.offset.toString());
  const query = params.toString();
  const raw = await request<{
    data: Deal[];
    pagination?: { total_items?: number };
    deals?: Deal[];
    count?: number;
  }>(`/crm/deals${query ? `?${query}` : ""}`, {
    skipAuthRedirect: true,
    skipCache: true,
  });
  if (
    raw &&
    typeof raw === "object" &&
    "data" in raw &&
    Array.isArray(raw.data)
  ) {
    return {
      deals: raw.data,
      count: raw.pagination?.total_items ?? raw.data.length,
    };
  }
  if (raw && "deals" in raw && Array.isArray(raw.deals)) {
    return raw as DealsResponse;
  }
  return { deals: [], count: 0 };
}

export async function getDeal(id: string): Promise<Deal> {
  return request<Deal>(`/crm/deals/${id}`, {
    skipAuthRedirect: true,
    skipCache: true,
  });
}

export async function createDeal(data: CreateDealData): Promise<Deal> {
  return request<Deal>("/crm/deals", {
    method: "POST",
    body: data,
    skipAuthRedirect: true,
  });
}

export async function updateDeal(
  id: string,
  data: UpdateDealData,
): Promise<Deal> {
  return request<Deal>(`/crm/deals/${id}`, {
    method: "PUT",
    body: data,
    skipAuthRedirect: true,
  });
}

export async function deleteDeal(id: string): Promise<void> {
  return request<void>(`/crm/deals/${id}`, {
    method: "DELETE",
    skipAuthRedirect: true,
  });
}

export async function moveDealToStage(
  dealId: string,
  stageId: string,
): Promise<Deal> {
  return request<Deal>(`/crm/deals/${dealId}/stage`, {
    method: "PATCH",
    body: { stage_id: stageId },
    skipAuthRedirect: true,
  });
}

export async function updateDealStatus(
  dealId: string,
  status: string,
  lostReason?: string,
): Promise<Deal> {
  return request<Deal>(`/crm/deals/${dealId}/status`, {
    method: "PATCH",
    body: { status, lost_reason: lostReason },
    skipAuthRedirect: true,
  });
}

export async function getDealStats(pipelineId?: string): Promise<DealStats> {
  const params = pipelineId ? `?pipeline_id=${pipelineId}` : "";
  return request<DealStats>(`/crm/deals/stats${params}`, {
    skipAuthRedirect: true,
    skipCache: true,
  });
}

// ============================================================================
// Activities
// ============================================================================

export async function getActivities(filters?: {
  activity_type?: string;
  is_completed?: boolean;
  limit?: number;
  offset?: number;
}): Promise<ActivitiesResponse> {
  const params = new URLSearchParams();
  if (filters?.activity_type)
    params.set("activity_type", filters.activity_type);
  if (filters?.is_completed !== undefined)
    params.set("is_completed", filters.is_completed.toString());
  if (filters?.limit) params.set("limit", filters.limit.toString());
  if (filters?.offset) params.set("offset", filters.offset.toString());
  const query = params.toString();
  const raw = await request<{
    data: CRMActivity[];
    pagination?: { total_items?: number };
    activities?: CRMActivity[];
    count?: number;
  }>(`/crm/activities${query ? `?${query}` : ""}`);
  if (
    raw &&
    typeof raw === "object" &&
    "data" in raw &&
    Array.isArray(raw.data)
  ) {
    return {
      activities: raw.data,
      count: raw.pagination?.total_items ?? raw.data.length,
    };
  }
  if (raw && "activities" in raw && Array.isArray(raw.activities)) {
    return raw as ActivitiesResponse;
  }
  return { activities: [], count: 0 };
}

export async function getDealActivities(
  dealId: string,
): Promise<ActivitiesResponse> {
  const raw = await request<{
    data: CRMActivity[];
    pagination?: { total_items?: number };
    activities?: CRMActivity[];
    count?: number;
  }>(`/crm/deals/${dealId}/activities`, {
    skipAuthRedirect: true,
    skipCache: true,
  });
  if (
    raw &&
    typeof raw === "object" &&
    "data" in raw &&
    Array.isArray(raw.data)
  ) {
    return {
      activities: raw.data,
      count: raw.pagination?.total_items ?? raw.data.length,
    };
  }
  if (raw && "activities" in raw && Array.isArray(raw.activities)) {
    return raw as ActivitiesResponse;
  }
  return { activities: [], count: 0 };
}

export async function createActivity(
  data: CreateActivityData,
): Promise<CRMActivity> {
  return request<CRMActivity>("/crm/activities", {
    method: "POST",
    body: data,
  });
}

export async function completeActivity(
  activityId: string,
  outcome?: string,
): Promise<CRMActivity> {
  return request<CRMActivity>(`/crm/activities/${activityId}/complete`, {
    method: "POST",
    body: { outcome },
  });
}

export async function deleteActivity(activityId: string): Promise<void> {
  return request<void>(`/crm/activities/${activityId}`, { method: "DELETE" });
}
