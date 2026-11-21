'use client';

/**
 * Client layer for the /agents roster grid. Server-fetched rows come in as
 * plain serialized props; this component owns only UI state — which card's
 * glass panel (if any) is open. Every card is a keyboard-accessible trigger
 * for AgentPanel; the interactive bits already living on a card (Set up
 * link, inline chat, task/cron drawer) stop the click from also opening the
 * panel underneath them.
 */
import Link from 'next/link';
import { useEffect, useState, type KeyboardEvent, type MouseEvent } from 'react';
import { needsSetup } from '@/lib/agent-setup';
import { AgentAvatar } from '@/components/AgentAvatar';
import { AgentChat } from '@/components/AgentChat';
import { AgentWorkPanel } from '@/components/AgentWorkPanel';
import { AgentPanel } from '@/components/AgentPanel';
import { Badge, Dot, SectionHead } from '@/components/terminal';
import { failoverModelOptions } from '@/lib/agent-review';
import type { DeptFilter } from '@/lib/agent-dept';
import type { Agent, AgentCron, AgentMessage, AgentRun, AgentTask, Department, Skill } from '@/lib/schemas';

function stop(e: MouseEvent | KeyboardEvent) {
  e.stopPropagation();
}

function AgentRosterCard({
  agent,
  parent,
  lastRun,
  tasks,
  crons,
  messages,
  avatarSrc,
  onOpen,
  modelOptions,
}: {
  modelOptions: string[];
  agent: Agent;
  parent: Agent | null;
  lastRun: AgentRun | undefined;
  tasks: AgentTask[];
  crons: AgentCron[];
  messages: AgentMessage[];
  avatarSrc: string | null;
  onOpen: () => void;
}) {
  const active = agent.status === 'active';
  return (
    <article
      className="hoverable group relative flex min-h-48 cursor-pointer flex-col rounded-lg-t border bg-os-surface p-6"
      style={{ borderColor: active ? 'color-mix(in oklab, var(--accent) 35%, var(--border))' : 'var(--border)' }}
      role="button"
      tabIndex={0}
      aria-label={`Open ${agent.name} profile`}
      onClick={onOpen}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onOpen();
        }
      }}
    >
      {needsSetup(agent) && (
        <Link
          href={`/agents/setup/${agent.id}`}
          onClick={stop}
          onKeyDown={stop}
          className="absolute -right-2 -top-2 z-10 rounded-[6px] bg-[var(--agent-badge)] px-3 py-1.5 text-[11.5px] font-medium text-white transition-transform hover:-translate-y-[1px]"
        >
          Set up
        </Link>
      )}
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="flex min-w-0 items-center gap-3">
            <AgentAvatar src={avatarSrc} name={agent.name} size={28} />
            <Dot state={agent.status} pulse={active} />
            <h3 className="truncate text-[14.5px] font-medium">{agent.name}</h3>
          </div>
          <div className="mt-1 truncate text-[12px] text-os-dim">{agent.role}</div>
        </div>
        <Badge>{agent.tier}</Badge>
      </div>

      <p className="mt-3 line-clamp-3 text-xs leading-relaxed text-os-muted [text-wrap:pretty]">{agent.description}</p>

      <div className="mt-4 flex flex-wrap gap-1">
        {agent.tools.slice(0, 5).map((tool) => (
          <span
            key={tool}
            className="glass-chip whitespace-nowrap rounded-full px-[7px] py-0.5 font-mono text-[11px] text-os-muted"
          >
            {tool}
          </span>
        ))}
        {agent.tools.length > 5 && (
          <span className="glass-chip rounded-full px-[7px] py-0.5 font-mono text-[11px] text-os-dim">
            +{agent.tools.length - 5}
          </span>
        )}
      </div>

      <div className="mt-auto pt-4">
        <div className="mb-4 flex items-center justify-between gap-4 text-[12px] text-os-dim">
          <span className="truncate">{parent ? `under ${parent.name}` : `instance ${agent.instance}`}</span>
          <span className="shrink-0">{agent.status}</span>
        </div>
        {lastRun && (
          <div className="flex items-center gap-2.5 text-[12px] leading-snug text-os-dim">
            <Dot state={lastRun.ok ? 'ok' : 'fail'} />
            <span className="shrink-0 text-os-muted">{lastRun.ok ? 'ok' : 'failed'}</span>
            <span className="truncate" title={lastRun.summary}>
              last check: {lastRun.summary.slice(0, 56)}
            </span>
          </div>
        )}
        <div onClick={stop} onKeyDown={stop}>
          <AgentChat agentId={agent.id} agentName={agent.name} initialMessages={messages} />
          <AgentWorkPanel agentId={agent.id} initialTasks={tasks} initialCrons={crons} failover={{ agentName: agent.name, model: agent.model, options: modelOptions }} />
        </div>
      </div>
    </article>
  );
}

export function AgentRoster({
  departments,
  agents,
  avatars,
  tasksByAgent,
  cronsByAgent,
  messagesByAgent,
  runsByAgent,
  skills,
  defaultModel,
  dept = null,
}: {
  /** ?dept= (the Social content crew card): only this department's agents */
  dept?: DeptFilter;
  /** F9 failover: the gateway default, for the model select's options. */
  defaultModel: string;
  departments: Department[];
  agents: Agent[];
  avatars: Record<string, string>;
  tasksByAgent: Record<string, AgentTask[]>;
  cronsByAgent: Record<string, AgentCron[]>;
  messagesByAgent: Record<string, AgentMessage[]>;
  runsByAgent: Record<string, AgentRun[]>;
  skills: Skill[];
}) {
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const agentsById = new Map(agents.map((a) => [a.id, a]));
  const selectedAgent = selectedId ? agentsById.get(selectedId) ?? null : null;
  // `/agents?agent=<id>` opens that agent's panel (SPEC C21: Home's agent rows link here).
  useEffect(() => {
    const id = new URL(window.location.href).searchParams.get('agent');
    if (id && agents.some((a) => a.id === id)) setSelectedId(id);
  }, [agents]);

  // The face wall shows agents with a real profile photo; each face opens the
  // same profile panel as the roster cards .
  const withPhoto = agents.filter((a) => avatars[a.id]);
  const faceGroups: Array<[string, Agent[]]> = [
    ['Active', withPhoto.filter((a) => a.status === 'active')],
    ['Inactive', withPhoto.filter((a) => a.status !== 'active')],
  ];

  return (
    <>
      {withPhoto.length > 0 && (
        <div className="mb-10 rounded-lg-t border border-os-border bg-os-surface px-6 py-7">
          {faceGroups
            .filter(([, list]) => list.length > 0)
            .map(([label, list]) => (
              <div key={label} className="mb-7 last:mb-0">
                <div className="mb-5 text-center text-[12.5px] text-os-muted">{label}</div>
                <div className="flex flex-wrap justify-center gap-x-10 gap-y-7">
                  {list.map((a) => (
                    <button
                      key={a.id}
                      onClick={() => setSelectedId(a.id)}
                      aria-label={`Open ${a.name} profile`}
                      className="hoverable flex w-[128px] cursor-pointer flex-col items-center rounded-[12px] p-2 text-center"
                    >
                      <AgentAvatar src={avatars[a.id] ?? null} name={a.name} size={76} />
                      <div className="mt-2.5 text-[13px] font-medium leading-tight">{a.name}</div>
                      <div className="mt-1 text-[11px] leading-snug text-os-dim">{a.role}</div>
                    </button>
                  ))}
                </div>
              </div>
            ))}
        </div>
      )}
      {dept && (
        <div data-dept-filter={dept.department?.id ?? ''} className="mb-5 flex items-center gap-3 text-[13px] text-os-muted">
          <span>
            {dept.department ? `Showing ${dept.department.name} only` : `No department matches “${dept.param}”`}
          </span>
          <Link href="/agents" className="text-[var(--accent)] hover:underline">
            Show every department
          </Link>
        </div>
      )}
      <div className="space-y-8">
        {departments.filter((d) => !dept || d.id === dept.department?.id).map((dept) => {
          const deptAgents = agents.filter((a) => a.departmentId === dept.id);
          if (deptAgents.length === 0) return null;
          return (
            <section key={dept.id}>
              <SectionHead label={dept.name} count={`${deptAgents.length} agents`} />
              <div className="-mt-1 mb-4 text-[13px] text-os-dim">{dept.tagline}</div>
              <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 ultra:grid-cols-5">
                {deptAgents.map((agent) => (
                  <AgentRosterCard
                    key={agent.id}
                    agent={agent}
                    parent={agent.parentId ? agentsById.get(agent.parentId) ?? null : null}
                    lastRun={runsByAgent[agent.id]?.[0]}
                    tasks={tasksByAgent[agent.id] ?? []}
                    crons={cronsByAgent[agent.id] ?? []}
                    messages={messagesByAgent[agent.id] ?? []}
                    avatarSrc={avatars[agent.id] ?? null}
                    onOpen={() => setSelectedId(agent.id)}
                    modelOptions={failoverModelOptions(agents, agent.model, defaultModel)}
                  />
                ))}
              </div>
            </section>
          );
        })}
      </div>

      {selectedAgent && (
        <AgentPanel
          agent={selectedAgent}
          allAgents={agents}
          avatars={avatars}
          departments={departments}
          skills={skills}
          runs={runsByAgent[selectedAgent.id] ?? []}
          onClose={() => setSelectedId(null)}
          onSelectAgent={(id) => setSelectedId(id)}
        />
      )}
    </>
  );
}
