'use client';

/**
 * Full-screen glass profile overlay for one agent — opened from a roster
 * card on /agents. Blurs the whole page behind it, shows the agent's
 * derived workflow, its real connections, who it works with (via the real
 * parent/child edges in the seeded roster), plain-language logic, and its
 * real run history from the runs table. There is no input here: the old
 * "Send a directive" box sent nothing (honesty H25). Talking to the agent
 * happens in its real chat (POST /api/agents/<id>/chat).
 */
import { useEffect } from 'react';
import { X } from 'lucide-react';
import { useMeltClose } from '@/lib/use-melt';
import type { Agent, AgentRun, Department, Skill } from '@/lib/schemas';
import { AgentAvatar } from '@/components/AgentAvatar';
import { Badge, Dot } from '@/components/terminal';
import { buildLogicRules, buildStepTree, connectedChips, resolveWorksWith } from '@/app/agents/panel-data';

function relativeTime(iso: string): string {
  const diffMs = Date.now() - new Date(iso).getTime();
  const mins = Math.round(diffMs / 60000);
  if (mins < 1) return 'just now';
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.round(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.round(hours / 24);
  return `${days}d ago`;
}

export function AgentPanel({
  agent,
  allAgents,
  avatars,
  departments,
  skills,
  runs,
  onClose,
  onSelectAgent,
}: {
  agent: Agent;
  allAgents: Agent[];
  avatars: Record<string, string>;
  departments: Department[];
  skills: Skill[];
  runs: AgentRun[];
  onClose: () => void;
  onSelectAgent: (id: string) => void;
}) {
  // Close melts the face (lib/use-melt); the roster unmounts it once it has melted.
  const melt = useMeltClose(onClose);
  const close = melt.close;
  useEffect(() => {
    const prevOverflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') close();
    }
    window.addEventListener('keydown', onKey);
    return () => {
      document.body.style.overflow = prevOverflow;
      window.removeEventListener('keydown', onKey);
    };
  }, [close]);

  const department = departments.find((d) => d.id === agent.departmentId) ?? null;
  const ownedSkills = skills.filter((s) => s.ownerAgentId === agent.id);
  const worksWith = resolveWorksWith(agent, allAgents);
  const parentEntry = worksWith.find((w) => w.relation === 'parent');
  const latestRun = runs[0] ?? null;
  const steps = buildStepTree(agent, latestRun, ownedSkills);
  const chips = connectedChips(agent, ownedSkills);
  const rules = buildLogicRules(agent, worksWith, ownedSkills, latestRun);

  return (
    <div className="fixed inset-0 z-[100] flex items-center justify-center p-6" role="presentation" {...melt.layer}>
      {/* The scrim is the face's sibling (glass brief §3.2): it frosts and dims the page; the panel frosts on its own. */}
      <div className="gl-scrim" aria-hidden onClick={close} />
      <div
        ref={melt.faceRef}
        onAnimationEnd={melt.onAnimationEnd}
        className={`glass-panel glass-face--strong gl-materialise relative flex w-full max-w-[880px] flex-col overflow-hidden${melt.melting ? ' is-melting' : ''}`}
        style={{ maxHeight: '86vh' }}
        role="dialog"
        aria-modal="true"
        aria-label={`${agent.name} profile`}
      >
        <button
          onClick={close}
          aria-label="Close"
          className="absolute right-4 top-4 z-10 grid h-8 w-8 place-items-center rounded-full border border-[rgb(var(--mist-rgb) / 0.14)] bg-black/30 text-os-dim transition-colors hover:text-os-text"
        >
          <X className="h-4 w-4" />
        </button>

        <div className="flex-1 overflow-y-auto px-7 py-7">
          {/* Identity row */}
          <div className="flex items-start gap-4 pr-10">
            <AgentAvatar src={avatars[agent.id] ?? null} name={agent.name} size={64} />
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center gap-2.5">
                <h2 className="text-[20px] font-medium leading-tight">{agent.name}</h2>
                <Dot state={agent.status} pulse={agent.status === 'active'} />
                <span className="text-[12.5px] text-os-dim">{agent.status}</span>
              </div>
              <div className="mt-1 text-[13px] text-os-muted">{agent.role}</div>
              <div className="mt-2.5 flex flex-wrap items-center gap-1.5">
                <Badge>{agent.tier}</Badge>
                {department && <Badge>{department.name}</Badge>}
              </div>
              <div className="mt-2 text-[12px] text-os-dim">
                {parentEntry ? `under ${parentEntry.agent.name}` : `instance ${agent.instance}`}
              </div>
            </div>
          </div>

          {/* What it does */}
          <section className="mt-7">
            <span className="glass-label">What it does</span>
            <div className="mt-3.5 flex flex-col">
              {steps.length === 0 && <div className="text-[13px] text-os-dim">Nothing derivable yet.</div>}
              {steps.map((step, i) => (
                <div key={step.id} className="flex gap-3">
                  <div className="flex flex-col items-center">
                    <span className="agentpanel-node-dot mt-3.5" />
                    {i < steps.length - 1 && <span className="agentpanel-connector flex-1" />}
                  </div>
                  <div className="agentpanel-step glass-well mb-2.5 flex-1 px-3.5 py-2.5 text-[13px] leading-snug text-os-muted">
                    {step.label}
                  </div>
                </div>
              ))}
            </div>
          </section>

          {/* Connected to */}
          <section className="mt-7">
            <span className="glass-label">Connected to</span>
            <div className="mt-3.5 flex flex-wrap gap-1.5">
              {chips.length === 0 && <span className="text-[12.5px] text-os-dim">Nothing wired yet.</span>}
              {chips.map((c) => (
                <span key={c} className="glass-chip px-3 py-1 font-mono text-[11.5px] text-os-muted">
                  {c}
                </span>
              ))}
            </div>
          </section>

          {/* Works with */}
          {worksWith.length > 0 && (
            <section className="mt-7">
              <span className="glass-label">Works with</span>
              <div className="mt-3.5 flex flex-wrap gap-2">
                {worksWith.map((w) => (
                  <button
                    key={w.agent.id}
                    onClick={() => onSelectAgent(w.agent.id)}
                    className="glass-chip flex items-center gap-2 py-1.5 pl-1.5 pr-3 text-[12.5px] text-os-muted transition-colors hover:text-os-text"
                  >
                    <AgentAvatar src={avatars[w.agent.id] ?? null} name={w.agent.name} size={20} />
                    <span className="font-medium text-os-text">{w.agent.name}</span>
                    <span className="text-os-dim">{w.relation === 'parent' ? 'parent' : 'reports in'}</span>
                  </button>
                ))}
              </div>
            </section>
          )}

          {/* Logic */}
          <section className="mt-7">
            <span className="glass-label">Logic</span>
            <ul className="mt-3.5 flex flex-col gap-2">
              {rules.map((rule, i) => (
                <li key={i} className="flex gap-2.5 text-[13px] leading-relaxed text-os-muted">
                  <span className="mt-[7px] h-1 w-1 shrink-0 rounded-full bg-os-dim" />
                  <span>{rule}</span>
                </li>
              ))}
            </ul>
          </section>

          {/* Run feed */}
          <section className="mt-7">
            <span className="glass-label">Run feed</span>
            <div className="mt-3.5 flex flex-col gap-2">
              {runs.length === 0 && (
                <div className="text-[12.5px] text-os-dim">No runs recorded yet.</div>
              )}
              {runs.map((run) => (
                <div key={run.id} className="flex items-center gap-3 text-[12.5px]">
                  <Dot state={run.ok ? 'ok' : 'fail'} />
                  <span className="shrink-0 font-mono text-[11px] text-os-dim">{relativeTime(run.startedAt)}</span>
                  <span className="min-w-0 flex-1 truncate text-os-muted">{run.summary}</span>
                </div>
              ))}
            </div>
          </section>
        </div>
      </div>

      <style>{`
        .agentpanel-node-dot {
          width: 6px;
          height: 6px;
          border-radius: 50%;
          background: var(--accent);
          flex-shrink: 0;
        }
        .agentpanel-connector {
          width: 1px;
          min-height: 14px;
          background: rgb(var(--mist-rgb) /  0.14);
        }
        .agentpanel-step {
          border-radius: 10px;
          border: 1px solid rgb(var(--mist-rgb) /  0.1);
          background: rgba(20, 34, 31, 0.35);
        }
      `}</style>
    </div>
  );
}
