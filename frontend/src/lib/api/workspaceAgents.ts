// Workspace Agents API client — the new agent system (replaces Dalya). Defines
// AI agents (name, role, model, system prompt) per workspace and runs them
// against a Claude model. Goes through request<T>() (base url, cookie, CSRF,
// X-Workspace-ID header).
import { request } from "./base";

export interface WorkspaceAgent {
  id: string;
  name: string;
  role: string;
  description: string;
  model: string;
  system_prompt: string;
  status: string;
  created_by: string | null;
  created_at: string;
  updated_at: string;
}

export interface WorkspaceAgentRun {
  id: string;
  agent_id: string;
  input: string;
  output: string;
  model: string;
  status: string; // "done" | "error"
  created_at: string;
}

export interface AgentModel {
  id: string;
  label: string;
  hint: string;
}

// The Claude models an agent can be configured with (mirrors the backend allow-list).
export const AGENT_MODELS: AgentModel[] = [
  { id: "claude-sonnet-4-5-20250929", label: "Sonnet 4.5", hint: "Balanced (default)" },
  { id: "claude-opus-4-1-20250805", label: "Opus 4", hint: "Most capable" },
  { id: "claude-haiku-4-5-20251001", label: "Haiku 3.5", hint: "Fast & cheap" },
];

export const DEFAULT_AGENT_MODEL = "claude-sonnet-4-5-20250929";

export function modelLabel(id: string): string {
  return AGENT_MODELS.find((m) => m.id === id)?.label ?? id;
}

export interface AgentInput {
  name: string;
  role?: string;
  description?: string;
  model?: string;
  system_prompt?: string;
  status?: string;
}

export interface RunResult {
  output: string;
  model: string;
  status: string;
  run_id: string;
  created_at: string;
}

export async function listAgents(): Promise<{
  agents: WorkspaceAgent[];
  count: number;
  ai_available: boolean;
}> {
  return request(`/workspace-agents`, { skipCache: true });
}

export async function createAgent(input: AgentInput): Promise<WorkspaceAgent> {
  return request(`/workspace-agents`, { method: "POST", body: input });
}

export async function updateAgent(
  id: string,
  input: AgentInput,
): Promise<WorkspaceAgent> {
  return request(`/workspace-agents/${id}`, { method: "PUT", body: input });
}

export async function deleteAgent(id: string): Promise<{ message: string }> {
  return request(`/workspace-agents/${id}`, { method: "DELETE" });
}

export async function runAgent(
  id: string,
  input: string,
): Promise<RunResult> {
  return request(`/workspace-agents/${id}/run`, {
    method: "POST",
    body: { input },
    timeout: 120_000,
  });
}

export async function listRuns(
  id: string,
): Promise<{ runs: WorkspaceAgentRun[]; count: number }> {
  return request(`/workspace-agents/${id}/runs`, { skipCache: true });
}
