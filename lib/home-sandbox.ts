import type { Agent, Department, SopTask } from '@/lib/schemas';

export type SandboxTaskPreview = {
  agent: string;
  department: string;
  task: string;
};

/** The landing preview follows the team map's one-SOP-task-per-agent links. */
export function previewSandboxTasks(tasks: SopTask[], agents: Agent[], departments: Department[]): SandboxTaskPreview[] {
  const agentById = new Map(agents.map((agent) => [agent.id, agent]));
  const departmentById = new Map(departments.map((department) => [department.id, department]));
  const seenAgents = new Set<string>();
  const previews: SandboxTaskPreview[] = [];

  for (const task of tasks) {
    if (task.assigneeKind !== 'agent' || seenAgents.has(task.assigneeId)) continue;
    const agent = agentById.get(task.assigneeId);
    if (!agent) continue;
    const department = departmentById.get(task.departmentId);
    seenAgents.add(agent.id);
    previews.push({ agent: agent.name, department: department?.name ?? 'Team', task: task.title });
    if (previews.length === 3) break;
  }
  return previews;
}

export type SandboxSpeedSample = { name: string; latencyMs: number; widthPct: number };

/** Fixed, clearly labeled demo timings. The lowest latency anchors the chart at 100%. */
export function sandboxSpeed(): SandboxSpeedSample[] {
  const samples = [
    { name: 'Atlas', latencyMs: 57 },
    { name: 'Quill', latencyMs: 84 },
    { name: 'Flint', latencyMs: 112 },
  ];
  const fastest = Math.min(...samples.map((sample) => sample.latencyMs));
  return samples.map((sample) => ({ ...sample, widthPct: Math.round((fastest / sample.latencyMs) * 100) }));
}
