// Pipelines API client - per-workspace sales/delivery pipelines and their
// stages. The backend exposes these under /crm (CRMHandler); this dedicated
// client is the Pipelines module's own surface over that contract. Goes through
// request<T>() which handles the API base, session cookie, CSRF, and the
// X-Workspace-ID header.
import { request } from "./base";

export type PipelineType = "sales" | "hiring" | "projects" | "custom";
export type StageType = "open" | "won" | "lost";

export interface Pipeline {
  id: string;
  user_id: string;
  name: string;
  description?: string | null;
  pipeline_type?: PipelineType | null;
  currency?: string | null;
  is_default?: boolean | null;
  is_active?: boolean | null;
  color?: string | null;
  icon?: string | null;
  created_at: string;
  updated_at: string;
}

export interface PipelineStage {
  id: string;
  pipeline_id: string;
  name: string;
  description?: string | null;
  position: number;
  probability?: number | null;
  stage_type?: StageType | null;
  rotting_days?: number | null;
  color?: string | null;
  created_at: string;
  updated_at: string;
}

export interface CreatePipelineInput {
  name: string;
  description?: string;
  pipeline_type?: PipelineType;
  currency?: string;
  is_default?: boolean;
  color?: string;
  icon?: string;
}

export interface UpdatePipelineInput {
  name: string;
  description?: string;
  currency?: string;
  color?: string;
  icon?: string;
  is_active?: boolean;
}

export interface CreateStageInput {
  name: string;
  description?: string;
  position: number;
  probability?: number;
  stage_type?: StageType;
  rotting_days?: number;
  color?: string;
}

export interface UpdateStageInput {
  name: string;
  description?: string;
  probability?: number;
  rotting_days?: number;
  color?: string;
}

// ---- Pipelines ----

export async function getPipelines(): Promise<{
  pipelines: Pipeline[];
  count: number;
}> {
  return request<{ pipelines: Pipeline[]; count: number }>("/crm/pipelines", {
    skipCache: true,
  });
}

export function getPipeline(id: string): Promise<Pipeline> {
  return request<Pipeline>(`/crm/pipelines/${id}`, { skipCache: true });
}

export function createPipeline(body: CreatePipelineInput): Promise<Pipeline> {
  return request<Pipeline>("/crm/pipelines", { method: "POST", body });
}

export function updatePipeline(
  id: string,
  body: UpdatePipelineInput,
): Promise<Pipeline> {
  return request<Pipeline>(`/crm/pipelines/${id}`, { method: "PUT", body });
}

export function deletePipeline(id: string): Promise<unknown> {
  return request(`/crm/pipelines/${id}`, { method: "DELETE" });
}

// ---- Stages ----

export async function getPipelineStages(
  pipelineId: string,
): Promise<{ stages: PipelineStage[]; count: number }> {
  return request<{ stages: PipelineStage[]; count: number }>(
    `/crm/pipelines/${pipelineId}/stages`,
    { skipCache: true },
  );
}

export function createPipelineStage(
  pipelineId: string,
  body: CreateStageInput,
): Promise<PipelineStage> {
  return request<PipelineStage>(`/crm/pipelines/${pipelineId}/stages`, {
    method: "POST",
    body,
  });
}

export function updatePipelineStage(
  pipelineId: string,
  stageId: string,
  body: UpdateStageInput,
): Promise<PipelineStage> {
  return request<PipelineStage>(
    `/crm/pipelines/${pipelineId}/stages/${stageId}`,
    { method: "PUT", body },
  );
}

export function deletePipelineStage(
  pipelineId: string,
  stageId: string,
): Promise<unknown> {
  return request(`/crm/pipelines/${pipelineId}/stages/${stageId}`, {
    method: "DELETE",
  });
}
