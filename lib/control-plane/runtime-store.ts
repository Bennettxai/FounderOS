import type { InvestmentWorkflow } from '@/lib/control-plane/schema';
import type { GovernedHandoff } from '@/lib/control-plane/handoff-schema';
import type { AgentRuntimeTask, AgentRuntimeTaskInput, CeoInboxItem, CeoInboxItemInput, RuntimeRedriveRecord } from '@/lib/control-plane/runtime-schema';
import type { GovernedHandoffStore } from '@/lib/control-plane/handoff-store';
import type {
  ControlPlaneCommandLedger,
  ControlPlaneDomainEventLedger,
  ControlPlaneJobStore,
} from '@/lib/control-plane/durable';

export interface InternalRuntimeStore
  extends GovernedHandoffStore,
    ControlPlaneCommandLedger,
    ControlPlaneDomainEventLedger,
    ControlPlaneJobStore
{
  listWorkflowSnapshots(): InvestmentWorkflow[];
  upsertAgentRuntimeTask(input: AgentRuntimeTaskInput): AgentRuntimeTask;
  listAgentRuntimeTasks(input?: { agentId?: string; workflowId?: string; status?: AgentRuntimeTask['status']; }): AgentRuntimeTask[];
  getAgentRuntimePlan(taskId: string): unknown | null;
  saveAgentRuntimePlan(input: { taskId: string; workerId: string; savedAt: string; plan: unknown; }): void;
  claimAgentRuntimeTasks(input: { agentId: string; workerId: string; now: string; limit: number; leaseMs: number; }): AgentRuntimeTask[];
  completeAgentRuntimeTask(input: { taskId: string; workerId: string; completedAt: string; result?: Record<string, unknown>; }): AgentRuntimeTask;
  failAgentRuntimeTask(input: { taskId: string; workerId: string; failedAt: string; error: string; retryable?: boolean; }): AgentRuntimeTask;
  redriveAgentRuntimeTask(input: { taskId: string; requestedAt: string; reason: string; }): AgentRuntimeTask;
  redriveHandoff(input: { handoffId: string; requestedAt: string; reason: string; }): GovernedHandoff;
  listRuntimeRedrives(): RuntimeRedriveRecord[];
  upsertCeoInboxItem(input: CeoInboxItemInput): CeoInboxItem;
  listCeoInbox(input?: { status?: CeoInboxItem['status']; category?: CeoInboxItem['category']; workflowId?: string; }): CeoInboxItem[];
  acknowledgeCeoInboxItem(input: { inboxId: string; acknowledgedAt: string; }): CeoInboxItem;
  resolveCeoInboxAfterControlPlane(input: { inboxId: string; controlPlaneRequestId: string; resolvedAt: string; }): CeoInboxItem;
}
