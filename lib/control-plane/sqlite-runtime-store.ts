import { InvestmentWorkflowSchema, type InvestmentWorkflow } from '@/lib/control-plane/schema';
import { GovernedHandoffSchema, type GovernedHandoff } from '@/lib/control-plane/handoff-schema';
import { AgentRuntimeTaskSchema, CeoInboxItemSchema, RuntimeRedriveRecordSchema, type AgentRuntimeTask, type AgentRuntimeTaskInput, type CeoInboxItem, type CeoInboxItemInput, type RuntimeRedriveRecord } from '@/lib/control-plane/runtime-schema';
import type { InternalRuntimeStore } from '@/lib/control-plane/runtime-store';
import { computeRetryDelayMs } from '@/lib/control-plane/jobs';
import { SqliteGovernedHandoffStore } from '@/lib/control-plane/sqlite-handoff-store';

const INTERNAL_RUNTIME_DDL = `
CREATE TABLE IF NOT EXISTS cp_agent_runtime_tasks (
  task_id TEXT PRIMARY KEY, handoff_id TEXT NOT NULL UNIQUE, workflow_id TEXT,
  agent_id TEXT NOT NULL, action TEXT NOT NULL, summary TEXT NOT NULL,
  state_at_assignment TEXT, priority TEXT NOT NULL, status TEXT NOT NULL, attempts INTEGER NOT NULL,
  created_at TEXT NOT NULL, available_at TEXT NOT NULL, updated_at TEXT NOT NULL,
  retry_policy_json TEXT NOT NULL, lease_owner TEXT, lease_until TEXT, completed_at TEXT, last_error TEXT,
  policy_evidence_json TEXT NOT NULL, payload_json TEXT NOT NULL, result_json TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS cp_agent_runtime_plans (
  task_id TEXT PRIMARY KEY,
  plan_json TEXT NOT NULL,
  saved_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS cp_agent_runtime_tasks_queue_idx ON cp_agent_runtime_tasks(agent_id, status, available_at, lease_until);
CREATE INDEX IF NOT EXISTS cp_agent_runtime_tasks_workflow_idx ON cp_agent_runtime_tasks(workflow_id, created_at);
CREATE TABLE IF NOT EXISTS cp_ceo_inbox (
  inbox_id TEXT PRIMARY KEY, handoff_id TEXT NOT NULL UNIQUE, workflow_id TEXT, category TEXT NOT NULL,
  action TEXT NOT NULL, summary TEXT NOT NULL, state_at_creation TEXT, status TEXT NOT NULL,
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL, acknowledged_at TEXT, resolved_at TEXT,
  resolved_by_control_plane_request_id TEXT, policy_evidence_json TEXT NOT NULL, payload_json TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS cp_ceo_inbox_status_idx ON cp_ceo_inbox(status, category, created_at);
CREATE INDEX IF NOT EXISTS cp_ceo_inbox_workflow_idx ON cp_ceo_inbox(workflow_id, created_at);
CREATE TABLE IF NOT EXISTS cp_runtime_redrives (
  redrive_id TEXT PRIMARY KEY, kind TEXT NOT NULL, target_id TEXT NOT NULL, requested_at TEXT NOT NULL,
  reason TEXT NOT NULL, previous_attempts INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS cp_runtime_redrives_target_idx ON cp_runtime_redrives(kind, target_id, requested_at);
`;

type WorkflowRow = { snapshot_json: string };
type AgentTaskRow = { task_id:string; handoff_id:string; workflow_id:string|null; agent_id:string; action:string; summary:string; state_at_assignment:string|null; priority:string; status:string; attempts:number; created_at:string; available_at:string; updated_at:string; retry_policy_json:string; lease_owner:string|null; lease_until:string|null; completed_at:string|null; last_error:string|null; policy_evidence_json:string; payload_json:string; result_json:string; };
type CeoInboxRow = { inbox_id:string; handoff_id:string; workflow_id:string|null; category:string; action:string; summary:string; state_at_creation:string|null; status:string; created_at:string; updated_at:string; acknowledged_at:string|null; resolved_at:string|null; resolved_by_control_plane_request_id:string|null; policy_evidence_json:string; payload_json:string; };
type HandoffRow = { handoff_id:string; source_message_id:string; source_topic:string; workflow_id:string|null; state_at_event:string|null; kind:string; destination_kind:string; destination_id:string; action:string; summary:string; status:string; attempts:number; created_at:string; available_at:string; retry_policy_json:string; lease_owner:string|null; lease_until:string|null; delivered_at:string|null; last_error:string|null; policy_evidence_json:string; payload_json:string; };
type RuntimeRedriveRow = { redrive_id:string; kind:string; target_id:string; requested_at:string; reason:string; previous_attempts:number; };
const parseJson = <T,>(value:string):T => JSON.parse(value) as T;
const addMs = (iso:string, ms:number) => new Date(Date.parse(iso)+ms).toISOString();

function rowToAgentTask(row:AgentTaskRow):AgentRuntimeTask { return AgentRuntimeTaskSchema.parse({ taskId:row.task_id, handoffId:row.handoff_id, workflowId:row.workflow_id??undefined, agentId:row.agent_id, action:row.action, summary:row.summary, stateAtAssignment:row.state_at_assignment??undefined, priority:row.priority, status:row.status, attempts:row.attempts, createdAt:row.created_at, availableAt:row.available_at, updatedAt:row.updated_at, retryPolicy:parseJson(row.retry_policy_json), leaseOwner:row.lease_owner??undefined, leaseUntil:row.lease_until??undefined, completedAt:row.completed_at??undefined, lastError:row.last_error??undefined, policyEvidence:parseJson(row.policy_evidence_json), payload:parseJson(row.payload_json), result:parseJson(row.result_json) }); }
function rowToCeo(row:CeoInboxRow):CeoInboxItem { return CeoInboxItemSchema.parse({ inboxId:row.inbox_id, handoffId:row.handoff_id, workflowId:row.workflow_id??undefined, category:row.category, action:row.action, summary:row.summary, stateAtCreation:row.state_at_creation??undefined, status:row.status, createdAt:row.created_at, updatedAt:row.updated_at, acknowledgedAt:row.acknowledged_at??undefined, resolvedAt:row.resolved_at??undefined, resolvedByControlPlaneRequestId:row.resolved_by_control_plane_request_id??undefined, policyEvidence:parseJson(row.policy_evidence_json), payload:parseJson(row.payload_json) }); }
function rowToHandoff(row:HandoffRow):GovernedHandoff { return GovernedHandoffSchema.parse({ handoffId:row.handoff_id, sourceMessageId:row.source_message_id, sourceTopic:row.source_topic, workflowId:row.workflow_id??undefined, stateAtEvent:row.state_at_event??undefined, kind:row.kind, destination:{kind:row.destination_kind,id:row.destination_id}, action:row.action, summary:row.summary, status:row.status, attempts:row.attempts, createdAt:row.created_at, availableAt:row.available_at, retryPolicy:parseJson(row.retry_policy_json), leaseOwner:row.lease_owner??undefined, leaseUntil:row.lease_until??undefined, deliveredAt:row.delivered_at??undefined, lastError:row.last_error??undefined, policyEvidence:parseJson(row.policy_evidence_json), payload:parseJson(row.payload_json) }); }
function rowToRedrive(row:RuntimeRedriveRow):RuntimeRedriveRecord { return RuntimeRedriveRecordSchema.parse({ redriveId:row.redrive_id, kind:row.kind, targetId:row.target_id, requestedAt:row.requested_at, reason:row.reason, previousAttempts:row.previous_attempts }); }
const redriveId = (kind:'agent_task'|'handoff', targetId:string, at:string) => `runtime-redrive:${kind}:${targetId}:${at}`;

export class SqliteInternalRuntimeStore extends SqliteGovernedHandoffStore implements InternalRuntimeStore {
  constructor(dbPath:string){ super(dbPath); this.db.exec(INTERNAL_RUNTIME_DDL); }
  listWorkflowSnapshots():InvestmentWorkflow[]{ const rows=this.db.prepare(`SELECT snapshot_json FROM cp_workflows ORDER BY updated_at DESC, rowid DESC`).all() as WorkflowRow[]; return rows.map(r=>InvestmentWorkflowSchema.parse(parseJson(r.snapshot_json))); }
  upsertAgentRuntimeTask(input:AgentRuntimeTaskInput):AgentRuntimeTask { const t=AgentRuntimeTaskSchema.parse(input); this.db.prepare(`INSERT OR IGNORE INTO cp_agent_runtime_tasks (task_id,handoff_id,workflow_id,agent_id,action,summary,state_at_assignment,priority,status,attempts,created_at,available_at,updated_at,retry_policy_json,lease_owner,lease_until,completed_at,last_error,policy_evidence_json,payload_json,result_json) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`).run(t.taskId,t.handoffId,t.workflowId??null,t.agentId,t.action,t.summary,t.stateAtAssignment??null,t.priority,t.status,t.attempts,t.createdAt,t.availableAt,t.updatedAt,JSON.stringify(t.retryPolicy),t.leaseOwner??null,t.leaseUntil??null,t.completedAt??null,t.lastError??null,JSON.stringify(t.policyEvidence),JSON.stringify(t.payload),JSON.stringify(t.result)); return this.readAgentTask(t.taskId); }
  listAgentRuntimeTasks(input:{agentId?:string;workflowId?:string;status?:AgentRuntimeTask['status']}={}):AgentRuntimeTask[]{ const c:string[]=[]; const p:string[]=[]; if(input.agentId){c.push('agent_id = ?');p.push(input.agentId)} if(input.workflowId){c.push('workflow_id = ?');p.push(input.workflowId)} if(input.status){c.push('status = ?');p.push(input.status)} const where=c.length?`WHERE ${c.join(' AND ')}`:''; const rows=this.db.prepare(`SELECT * FROM cp_agent_runtime_tasks ${where} ORDER BY CASE priority WHEN 'critical' THEN 0 WHEN 'high' THEN 1 ELSE 2 END, created_at DESC, rowid DESC`).all(...p) as AgentTaskRow[]; return rows.map(rowToAgentTask); }
  getAgentRuntimePlan(taskId: string): unknown | null {
    const row = this.db.prepare('SELECT plan_json FROM cp_agent_runtime_plans WHERE task_id = ?')
      .get(taskId) as { plan_json: string } | undefined;
    return row ? parseJson<unknown>(row.plan_json) : null;
  }

  saveAgentRuntimePlan(input: { taskId: string; workerId: string; savedAt: string; plan: unknown; }): void {
    const transaction = this.db.transaction(() => {
      const current = this.readAgentTask(input.taskId);
      if (current.status !== 'running' || current.leaseOwner !== input.workerId) {
        throw new Error('agent_task_plan_lease_mismatch:' + input.taskId);
      }
      const encoded = JSON.stringify(input.plan);
      if (typeof encoded !== 'string') throw new Error('agent_task_plan_not_serializable');
      const prior = this.db.prepare('SELECT plan_json FROM cp_agent_runtime_plans WHERE task_id = ?')
        .get(input.taskId) as { plan_json: string } | undefined;
      if (prior) {
        if (prior.plan_json !== encoded) throw new Error('agent_task_plan_conflict:' + input.taskId);
        return;
      }
      this.db.prepare('INSERT INTO cp_agent_runtime_plans (task_id, plan_json, saved_at) VALUES (?, ?, ?)')
        .run(input.taskId, encoded, input.savedAt);
    });
    transaction();
  }

  claimAgentRuntimeTasks(input:{agentId:string;workerId:string;now:string;limit:number;leaseMs:number}):AgentRuntimeTask[]{ if(input.limit<=0) throw new Error('agent_task_claim_limit_must_be_positive'); if(input.leaseMs<=0) throw new Error('agent_task_claim_lease_must_be_positive'); const leaseUntil=addMs(input.now,input.leaseMs); const tx=this.db.transaction(()=>{ const candidates=this.db.prepare(`SELECT task_id FROM cp_agent_runtime_tasks WHERE agent_id = ? AND available_at <= ? AND (status='queued' OR (status='running' AND lease_until IS NOT NULL AND lease_until <= ?)) ORDER BY CASE priority WHEN 'critical' THEN 0 WHEN 'high' THEN 1 ELSE 2 END, available_at, created_at, rowid LIMIT ?`).all(input.agentId,input.now,input.now,input.limit) as {task_id:string}[]; const claim=this.db.prepare(`UPDATE cp_agent_runtime_tasks SET status='running',attempts=attempts+1,updated_at=?,lease_owner=?,lease_until=? WHERE task_id=? AND agent_id=? AND (status='queued' OR (status='running' AND lease_until IS NOT NULL AND lease_until <= ?))`); const out:AgentRuntimeTask[]=[]; for(const x of candidates){ const r=claim.run(input.now,input.workerId,leaseUntil,x.task_id,input.agentId,input.now); if(r.changes===1) out.push(this.readAgentTask(x.task_id)); } return out; }); return tx(); }
  completeAgentRuntimeTask(input:{taskId:string;workerId:string;completedAt:string;result?:Record<string,unknown>}):AgentRuntimeTask { const r=this.db.prepare(`UPDATE cp_agent_runtime_tasks SET status='completed',completed_at=?,updated_at=?,lease_owner=NULL,lease_until=NULL,last_error=NULL,result_json=? WHERE task_id=? AND status='running' AND lease_owner=?`).run(input.completedAt,input.completedAt,JSON.stringify(input.result??{}),input.taskId,input.workerId); if(r.changes!==1) throw new Error(`agent_task_complete_lease_mismatch:${input.taskId}`); return this.readAgentTask(input.taskId); }
  failAgentRuntimeTask(input:{taskId:string;workerId:string;failedAt:string;error:string;retryable?:boolean}):AgentRuntimeTask { const tx=this.db.transaction(()=>{ const cur=this.readAgentTask(input.taskId); if(cur.status!=='running'||cur.leaseOwner!==input.workerId) throw new Error(`agent_task_fail_lease_mismatch:${input.taskId}`); const exhausted=input.retryable===false||cur.attempts>=cur.retryPolicy.maxAttempts; const next=addMs(input.failedAt,computeRetryDelayMs(cur.retryPolicy,cur.attempts)); this.db.prepare(`UPDATE cp_agent_runtime_tasks SET status=?,available_at=?,updated_at=?,lease_owner=NULL,lease_until=NULL,last_error=? WHERE task_id=?`).run(exhausted?'dead_letter':'queued',exhausted?cur.availableAt:next,input.failedAt,input.error,input.taskId); return this.readAgentTask(input.taskId); }); return tx(); }
  redriveAgentRuntimeTask(input:{taskId:string;requestedAt:string;reason:string}):AgentRuntimeTask { const tx=this.db.transaction(()=>{ const cur=this.readAgentTask(input.taskId); if(cur.status!=='dead_letter') throw new Error(`agent_task_not_dead_letter:${input.taskId}`); this.db.prepare(`INSERT INTO cp_runtime_redrives (redrive_id,kind,target_id,requested_at,reason,previous_attempts) VALUES (?,'agent_task',?,?,?,?)`).run(redriveId('agent_task',input.taskId,input.requestedAt),input.taskId,input.requestedAt,input.reason,cur.attempts); this.db.prepare(`UPDATE cp_agent_runtime_tasks SET status='queued',attempts=0,available_at=?,updated_at=?,lease_owner=NULL,lease_until=NULL,last_error=NULL WHERE task_id=? AND status='dead_letter'`).run(input.requestedAt,input.requestedAt,input.taskId); return this.readAgentTask(input.taskId); }); return tx(); }
  redriveHandoff(input:{handoffId:string;requestedAt:string;reason:string}):GovernedHandoff { const tx=this.db.transaction(()=>{ const row=this.db.prepare(`SELECT * FROM cp_handoffs WHERE handoff_id=?`).get(input.handoffId) as HandoffRow|undefined; if(!row) throw new Error(`handoff_not_found:${input.handoffId}`); const cur=rowToHandoff(row); if(cur.status!=='dead_letter') throw new Error(`handoff_not_dead_letter:${input.handoffId}`); this.db.prepare(`INSERT INTO cp_runtime_redrives (redrive_id,kind,target_id,requested_at,reason,previous_attempts) VALUES (?,'handoff',?,?,?,?)`).run(redriveId('handoff',input.handoffId,input.requestedAt),input.handoffId,input.requestedAt,input.reason,cur.attempts); this.db.prepare(`UPDATE cp_handoffs SET status='pending',attempts=0,available_at=?,lease_owner=NULL,lease_until=NULL,last_error=NULL WHERE handoff_id=? AND status='dead_letter'`).run(input.requestedAt,input.handoffId); return rowToHandoff(this.db.prepare(`SELECT * FROM cp_handoffs WHERE handoff_id=?`).get(input.handoffId) as HandoffRow); }); return tx(); }
  listRuntimeRedrives():RuntimeRedriveRecord[]{ return (this.db.prepare(`SELECT * FROM cp_runtime_redrives ORDER BY requested_at DESC, rowid DESC`).all() as RuntimeRedriveRow[]).map(rowToRedrive); }
  upsertCeoInboxItem(input:CeoInboxItemInput):CeoInboxItem { const i=CeoInboxItemSchema.parse(input); this.db.prepare(`INSERT OR IGNORE INTO cp_ceo_inbox (inbox_id,handoff_id,workflow_id,category,action,summary,state_at_creation,status,created_at,updated_at,acknowledged_at,resolved_at,resolved_by_control_plane_request_id,policy_evidence_json,payload_json) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`).run(i.inboxId,i.handoffId,i.workflowId??null,i.category,i.action,i.summary,i.stateAtCreation??null,i.status,i.createdAt,i.updatedAt,i.acknowledgedAt??null,i.resolvedAt??null,i.resolvedByControlPlaneRequestId??null,JSON.stringify(i.policyEvidence),JSON.stringify(i.payload)); return this.readCeo(i.inboxId); }
  listCeoInbox(input:{status?:CeoInboxItem['status'];category?:CeoInboxItem['category'];workflowId?:string}={}):CeoInboxItem[]{ const c:string[]=[]; const p:string[]=[]; if(input.status){c.push('status = ?');p.push(input.status)} if(input.category){c.push('category = ?');p.push(input.category)} if(input.workflowId){c.push('workflow_id = ?');p.push(input.workflowId)} const where=c.length?`WHERE ${c.join(' AND ')}`:''; return (this.db.prepare(`SELECT * FROM cp_ceo_inbox ${where} ORDER BY created_at DESC, rowid DESC`).all(...p) as CeoInboxRow[]).map(rowToCeo); }
  acknowledgeCeoInboxItem(input:{inboxId:string;acknowledgedAt:string}):CeoInboxItem { const r=this.db.prepare(`UPDATE cp_ceo_inbox SET status=CASE WHEN status='pending' THEN 'acknowledged' ELSE status END,acknowledged_at=COALESCE(acknowledged_at,?),updated_at=? WHERE inbox_id=? AND status IN ('pending','acknowledged')`).run(input.acknowledgedAt,input.acknowledgedAt,input.inboxId); if(r.changes!==1) throw new Error(`ceo_inbox_not_acknowledgeable:${input.inboxId}`); return this.readCeo(input.inboxId); }
  resolveCeoInboxAfterControlPlane(input:{inboxId:string;controlPlaneRequestId:string;resolvedAt:string}):CeoInboxItem { const cmd=this.db.prepare(`SELECT outcome FROM cp_commands WHERE request_id=?`).get(input.controlPlaneRequestId) as {outcome:string}|undefined; if(!cmd) throw new Error(`ceo_inbox_control_plane_request_not_found:${input.controlPlaneRequestId}`); if(cmd.outcome!=='succeeded') throw new Error(`ceo_inbox_control_plane_request_not_succeeded:${input.controlPlaneRequestId}`); const r=this.db.prepare(`UPDATE cp_ceo_inbox SET status='resolved',resolved_at=?,resolved_by_control_plane_request_id=?,updated_at=? WHERE inbox_id=? AND status IN ('pending','acknowledged')`).run(input.resolvedAt,input.controlPlaneRequestId,input.resolvedAt,input.inboxId); if(r.changes!==1) throw new Error(`ceo_inbox_not_resolvable:${input.inboxId}`); return this.readCeo(input.inboxId); }
  private readAgentTask(id:string):AgentRuntimeTask { const row=this.db.prepare(`SELECT * FROM cp_agent_runtime_tasks WHERE task_id=?`).get(id) as AgentTaskRow|undefined; if(!row) throw new Error(`agent_runtime_task_not_found:${id}`); return rowToAgentTask(row); }
  private readCeo(id:string):CeoInboxItem { const row=this.db.prepare(`SELECT * FROM cp_ceo_inbox WHERE inbox_id=?`).get(id) as CeoInboxRow|undefined; if(!row) throw new Error(`ceo_inbox_item_not_found:${id}`); return rowToCeo(row); }
}
