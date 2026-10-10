// Engines API client - the workspace's internal automation / multi-agent
// workflow layer. A workflow is a named, ordered list of steps; running it
// executes each step (AI steps call Claude, note/http steps record) and stores
// the combined output as a run. Goes through request<T>() which handles the API
// base, session cookie, CSRF, and the X-Workspace-ID header.
import { request } from "./base";

export type StepType = "ai" | "note" | "http";
export type WorkflowTrigger = "manual" | "scheduled" | "event";
export type WorkflowStatus = "active" | "paused" | "archived";
export type RunStatus = "running" | "done" | "error";

export interface EngineStep {
  type: StepType;
  label: string;
  prompt: string; // used by "ai" steps
  config: string; // used by "note" / "http" steps
}

export interface Workflow {
  id: string;
  name: string;
  description: string;
  trigger: WorkflowTrigger;
  steps: EngineStep[];
  status: WorkflowStatus;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface WorkflowInput {
  name: string;
  description: string;
  trigger: WorkflowTrigger;
  steps: EngineStep[];
  status: WorkflowStatus;
}

export interface EngineRun {
  id: string;
  workflow_id: string;
  status: RunStatus;
  output: string;
  created_at: string;
}

export async function getWorkflows(): Promise<{
  workflows: Workflow[];
  count: number;
}> {
  return request<{ workflows: Workflow[]; count: number }>(`/engines`);
}

export async function createWorkflow(input: WorkflowInput): Promise<Workflow> {
  return request<Workflow>(`/engines`, {
    method: "POST",
    body: JSON.stringify(input),
  });
}

export async function updateWorkflow(
  id: string,
  input: WorkflowInput,
): Promise<Workflow> {
  return request<Workflow>(`/engines/${id}`, {
    method: "PUT",
    body: JSON.stringify(input),
  });
}

export async function deleteWorkflow(
  id: string,
): Promise<{ message: string }> {
  return request<{ message: string }>(`/engines/${id}`, { method: "DELETE" });
}

export async function runWorkflow(id: string): Promise<EngineRun> {
  return request<EngineRun>(`/engines/${id}/run`, {
    method: "POST",
    body: JSON.stringify({}),
    timeout: 120_000,
  });
}

export async function getRuns(
  id: string,
): Promise<{ runs: EngineRun[]; count: number }> {
  return request<{ runs: EngineRun[]; count: number }>(`/engines/${id}/runs`, {
    skipCache: true,
  });
}
