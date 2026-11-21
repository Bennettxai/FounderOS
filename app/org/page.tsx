import Link from 'next/link';
import { AgentAvatar } from '@/components/AgentAvatar';
import { agentAvatars } from '@/lib/agent-avatars';
import { Users } from 'lucide-react';
import { getDb } from '@/lib/data';
import { buildHierarchy, flattenNodes, type AgentNode } from '@/lib/hierarchy';
import { LIFE_AREAS, lifeAreaForDepartment } from '@/lib/life-map';
import { getVenture, readVentures, ventureAgentSet, venturesForAgent } from '@/lib/ventures-store';
import { ConductorCard } from '@/components/ConductorCard';
import { SparkIcon } from '@/components/SparkIcon';
import { PageHeader } from '@/components/PageHeader';
import type { Agent, AgentStatus } from '@/lib/schemas';
import { OPERATOR_NAME } from '@/lib/operator';
import { getAppearance } from '@/lib/appearance-server';
import { treeFor } from '@/lib/appearance';
import { buildAgentDirectory } from '@/lib/agent-directory';
import { OrgG, type OrgAgentRow } from '@/components/org/OrgG';

export const dynamic = 'force-dynamic';

const STATUS_DOT: Record<AgentStatus, string> = {
  active: 'bg-os-text',
  idle: 'bg-os-muted',
  training: 'bg-os-muted animate-pulse',
  planned: 'border border-os-dim bg-transparent',
};

/** Tiny colored dots showing which ventures an agent serves. */
function VentureDots({ agentId }: { agentId: string }) {
  const serving = venturesForAgent(agentId);
  if (serving.length === 0) return null;
  return (
    <span className="flex shrink-0 items-center gap-0.5">
      {serving.map((v) => (
        <span key={v.id} title={v.label} className="h-1 w-1 rounded-full" style={{ background: ventureColor(v.color) }} />
      ))}
    </span>
  );
}

/** Small black task pill, GladOS-board style. */
function AgentPill({ agent, dim = false }: { agent: Agent; dim?: boolean }) {
  return (
    <div
      title={`${agent.role} — ${agent.description}`}
      className={`flex items-center gap-2.5 rounded-full border border-os-border bg-os-bg px-2.5 py-2.5 ${dim ? 'opacity-20' : ''}`}
    >
      <AgentAvatar src={agentAvatars().get(agent.id) ?? null} name={agent.name} size={18} />
      <span className={`h-1.5 w-1.5 shrink-0 rounded-full ${STATUS_DOT[agent.status]}`} />
      <span className="truncate text-[11.5px] font-medium">{agent.name}</span>
      <VentureDots agentId={agent.id} />
    </div>
  );
}

function AgentNodePill({
  node,
  depth = 0,
  dimFor,
}: {
  node: AgentNode;
  depth?: number;
  dimFor?: (id: string) => boolean;
}) {
  return (
    <div className="space-y-2.5" style={{ paddingLeft: depth ? `${depth * 10}px` : undefined }}>
      <AgentPill agent={node.agent} dim={dimFor?.(node.agent.id) ?? false} />
      {node.children.map((child) => (
        <AgentNodePill key={child.agent.id} node={child} depth={depth + 1} dimFor={dimFor} />
      ))}
    </div>
  );
}

/** A flanking system card next to the AI Head, linked to its view. */
function SystemCard({ href, title, caption }: { href: string; title: string; caption: string }) {
  return (
    <Link
      href={href}
      className="hoverable group block w-44 rounded-lg-t border border-os-border bg-os-surface p-5 text-center"
    >
      <div className="flex justify-center">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg border border-os-border-bright bg-os-bg">
          <SparkIcon size={20} shade="var(--text-3)" />
        </div>
      </div>
      <div className="mt-1.5 text-xs font-medium">{title}</div>
      <div className="text-[11.5px] leading-snug text-os-dim">{caption}</div>
    </Link>
  );
}

/** Life-area hues are per-theme tokens (globals.css --life-*); the hex in lib/life-map is the console fallback. */
function lifeColor(a: { id: string; color: string }): string {
  return `var(--life-${a.id}, ${a.color})`;
}
/** A white venture mark is ink, not literal white, so it reads on White. */
function ventureColor(c: string): string {
  return c.toLowerCase() === '#fafafa' ? 'var(--text)' : c;
}

export default function OrgChartPage({ searchParams }: { searchParams?: { venture?: string } }) {
  const db = getDb();
  const departments = db.departments.all();
  const agents = db.agents.all();
  // The venture lens: same roster, same DB — the switcher just changes which
  // crew lights up. No venture param = everything bright.
  const venture = getVenture(searchParams?.venture ?? '');
  const ventureSet = venture ? ventureAgentSet(venture.id) : null;
  const dimFor = (id: string) => (ventureSet ? !ventureSet.has(id) : false);
  const conductor = agents.find((a) => a.id === 'conductor');
  // Wizard sits in the AI Head slot; the columns are everything else
  const tree = buildHierarchy(
    departments,
    agents.filter((a) => a.id !== 'conductor'),
  );
  const agentNames = Object.fromEntries(agents.map((a) => [a.id, a.name]));
  const lastBroadcast = db.broadcasts.recent(1)[0] ?? null;

  // GladOS tree (glados-ui SPEC S04 §4.7, X5): same hierarchy and lens, kit presentation.
  if (treeFor(getAppearance(), '/org') === 'glados') {
    // a white venture mark is ink on the GladOS tokens
    const gVentureColor = (c: string) => (c.toLowerCase() === '#fafafa' ? 'var(--g-ink)' : c);
    const row = (agent: Agent, depth: number): OrgAgentRow => ({
      id: agent.id,
      name: agent.name,
      role: agent.role,
      status: agent.status,
      instance: agent.instance,
      depth,
      ventureColors: venturesForAgent(agent.id).map((v) => gVentureColor(v.color)),
    });
    const walk = (nodes: AgentNode[], depth: number): OrgAgentRow[] => nodes.flatMap((n) => [row(n.agent, depth), ...walk(n.children, depth + 1)]);
    const columns = tree.departments.map(({ department, roots }) => {
      const leads = roots.filter((r) => r.agent.tier === 'lead');
      const leadIds = new Set(leads.map((r) => r.agent.id));
      const rest = roots.flatMap((r) => (leadIds.has(r.agent.id) ? r.children : [r]));
      return {
        id: department.id,
        name: department.name,
        area: lifeAreaForDepartment(department.id)?.label ?? null,
        leads: leads.map((r) => row(r.agent, 0)),
        workers: walk(rest, 0),
      };
    });
    const ventures = readVentures().map((v) => ({ id: v.id, label: v.label, color: gVentureColor(v.color), kind: v.kind, detail: v.detail, focus: v.focus, brainTag: v.brainTag }));
    return (
      <OrgG
        dir={buildAgentDirectory(db)}
        columns={columns}
        ventures={ventures}
        venture={venture ? ventures.find((v) => v.id === venture.id) ?? null : null}
        ventureAgents={ventureSet ? [...ventureSet] : null}
        operatorName={OPERATOR_NAME}
        routedCount={agents.filter((a) => a.id !== 'conductor').length}
        lastBroadcast={lastBroadcast}
      />
    );
  }

  return (
    <div>
      <PageHeader
        title="Agent hierarchy"
      />

      {/* Venture switcher: Acme / Workshop — one click swaps which
          crew lights up below. All data stays shared. */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <Link
          href="/org"
          className={`rounded-lg border px-3 py-2.5 text-xs font-medium transition-colors ${
            !venture ? 'glass-chip gl-on text-os-text' : 'border-os-border bg-os-surface text-os-muted hover:text-os-text'
          }`}
        >
          All ventures
        </Link>
        {readVentures().map((v) => {
          const active = venture?.id === v.id;
          return (
            <Link
              key={v.id}
              href={`/org?venture=${v.id}`}
              className={`flex items-center gap-2.5 rounded-lg border px-3 py-2.5 text-xs font-medium transition-colors ${
                active ? 'text-os-ink' : 'border-os-border bg-os-surface text-os-muted hover:text-os-text'
              }`}
              style={active ? { background: ventureColor(v.color), borderColor: ventureColor(v.color) } : undefined}
            >
              <span className="h-1.5 w-1.5 rounded-full" style={{ background: active ? 'var(--accent-ink)' : ventureColor(v.color) }} />
              {v.label}
            </Link>
          );
        })}
        {venture && <span className="text-[12.5px] text-os-dim">{venture.kind} · {venture.detail}</span>}
      </div>

      {venture && (
        <div
          className="mb-4 rounded-lg border bg-os-surface px-5 py-3"
          style={{ borderColor: `color-mix(in srgb, ${ventureColor(venture.color)} 40%, transparent)`, boxShadow: `inset 3px 0 0 ${ventureColor(venture.color)}` }}
        >
          <div className="text-[11.5px]" style={{ color: ventureColor(venture.color) }}>
            {venture.label} — executive focus
          </div>
          <ul className="mt-1.5 space-y-1">
            {venture.focus.map((f) => (
              <li key={f} className="flex items-start gap-3 text-[12.5px] text-os-muted">
                <span className="mt-1.5 h-1 w-1 shrink-0 rounded-full" style={{ background: ventureColor(venture.color) }} />
                {f}
              </li>
            ))}
          </ul>
          <div className="mt-3 font-mono text-[11.5px] text-os-dim">
            Optimal Engine tag: #{venture.brainTag} · {ventureSet?.size} agents on this venture
          </div>
        </div>
      )}

      {/* Life-area legend: every crew below is tinted by the part of life it serves */}
      <div className="mb-8 flex flex-wrap items-center gap-5 rounded-lg-t border border-os-border bg-os-surface px-4 py-3">
        <span className="text-[11.5px] text-os-dim">Life areas</span>
        {LIFE_AREAS.map((area) => (
          <span key={area.id} className="flex items-center gap-2.5 text-[11.5px] text-os-muted">
            <span className="h-2 w-2 rounded-full" style={{ background: lifeColor(area) }} />
            {area.label}
          </span>
        ))}
      </div>

      {/* Operator */}
      <div className="flex flex-col items-center">
        <Users className="h-7 w-7 text-os-text" />
        <div className="mt-1 text-base font-medium">{OPERATOR_NAME}</div>
        <div className="text-[12.5px] text-os-dim">Operator</div>
        <div className="mt-3 h-6 w-px bg-os-border-bright" />
        <div className="text-[12.5px] text-os-muted">Wizard (super agent)</div>
        <div className="h-3 w-px bg-os-border-bright" />
      </div>

      {/* AI Head row: Optimal Engine ── Wizard ── Comms Feed */}
      <div className="flex flex-wrap items-center justify-center gap-3 md:flex-nowrap md:gap-0">
        <SystemCard href="/brain" title="Optimal Engine" caption="markdown + pgvector knowledge store" />
        <div className="hidden h-px w-10 bg-os-border-bright md:block" />
        {conductor ? (
          <ConductorCard conductor={conductor} agentNames={agentNames} initialBroadcast={lastBroadcast} />
        ) : (
          <div className="rounded-xl border border-dashed border-os-border px-6 py-4 text-xs text-os-dim">
            conductor missing — run npm run seed
          </div>
        )}
        <div className="hidden h-px w-10 bg-os-border-bright md:block" />
        <SystemCard href="/inbox" title="Comms Feed" caption="Gmail · WhatsApp · Slack, unified" />
      </div>

      {/* Trunk down to the department rail */}
      <div className="mx-auto h-10 w-px bg-os-border-bright" />

      {/* Department crews: sparkle tile → instance agents → worker pills → tools.
          The whole row is centered under the Wizard (mx-auto w-max) and only
          scrolls when it genuinely overflows the viewport. Spacing widens with
          the screen: gap-5 → gap-8 (wide) → gap-12 (ultra / 32"). */}
      <div className="overflow-x-auto overflow-y-hidden pb-4 overscroll-x-contain" data-matrix-scroll>
        <div className="mx-auto w-max">
          {/* Rail inset by half a column (mx-36 = ½ of w-72) so it runs exactly
              center-to-center across the crews — connectors always meet it. */}
          <div className="mx-36 h-px bg-os-border-bright" />
          <div className="flex gap-5 pt-4 wide:gap-8 ultra:gap-12">
          {tree.departments.map(({ department, roots }) => {
            const all = flattenNodes(roots).map((n) => n.agent);
            // Every lead at the department root is an instance slot; everything
            // else (workers, root specialists) renders as a task pill below.
            const instanceAgents = roots.filter((r) => r.agent.tier === 'lead').map((r) => r.agent);
            const instanceIds = new Set(instanceAgents.map((a) => a.id));
            const pillNodes = roots.flatMap((root) =>
              instanceIds.has(root.agent.id) ? root.children : [root],
            );
            const deptTools = [...new Set(all.flatMap((a) => a.tools))];
            const area = lifeAreaForDepartment(department.id);
            return (
              <section
                key={department.id}
                className="org-connector flex w-72 shrink-0 flex-col items-center gap-3.5 wide:px-4 wide:py-5"
              >
                <div className="text-[13px] font-medium">{department.name}</div>
                {area && (
                  <div className="flex items-center gap-2.5 text-[11.5px]" style={{ color: lifeColor(area) }}>
                    <span className="h-1.5 w-1.5 rounded-full" style={{ background: lifeColor(area) }} />
                    {area.label}
                  </div>
                )}
                <div
                  className="group flex h-16 w-16 items-center justify-center rounded-2xl bg-os-raised"
                  style={{ border: `1px solid color-mix(in srgb, ${area ? lifeColor(area) : 'var(--text-3)'} 33%, transparent)` }}
                >
                  <SparkIcon size={34} shade={area ? lifeColor(area) : department.color} />
                </div>

              {instanceAgents.length > 0 && (
                <div className="w-full space-y-2.5">
                  <div className="text-center text-[11.5px] text-os-dim">
                    {department.name} crew
                  </div>
                  {/* Instance slots: each becomes an OpenClaw / Claude Code process on the host */}
                  {instanceAgents.map((agent) => (
                    <div
                      key={agent.id}
                      title={agent.description}
                      className={`w-full rounded-lg-t border border-os-border-bright bg-os-surface px-4 py-3 ${
                        dimFor(agent.id) ? 'opacity-20' : ''
                      }`}
                      style={
                        venture && !dimFor(agent.id)
                          ? { borderColor: `color-mix(in srgb, ${ventureColor(venture.color)} 40%, transparent)` }
                          : undefined
                      }
                    >
                      <div className="flex items-center justify-between gap-3">
                        <div className="flex min-w-0 items-center gap-2.5">
                          <AgentAvatar src={agentAvatars().get(agent.id) ?? null} name={agent.name} size={20} />
                          <span className={`h-1.5 w-1.5 shrink-0 rounded-full ${STATUS_DOT[agent.status]}`} />
                          <span className="truncate text-xs font-medium">{agent.name}</span>
                          <VentureDots agentId={agent.id} />
                        </div>
                        <span className="shrink-0 rounded bg-os-raised px-1.5 py-0.5 font-mono text-[10px] text-os-dim">
                          {agent.instance}
                        </span>
                      </div>
                      <div className="mt-0.5 truncate pl-3 text-[11.5px] text-os-dim">{agent.role}</div>
                    </div>
                  ))}
                </div>
              )}

              {pillNodes.length > 0 && (
                <div className="grid w-full grid-cols-2 gap-2.5">
                  {pillNodes.map((node) => (
                    <AgentNodePill key={node.agent.id} node={node} dimFor={dimFor} />
                  ))}
                </div>
              )}

              {all.length === 0 && (
                <div className="w-full rounded-xl border border-dashed border-os-border px-3 py-5 text-center text-[11.5px] text-os-dim">
                  Agents land here as this department goes live
                </div>
              )}

              {deptTools.length > 0 && (
                <div className="w-full">
                  <div className="rounded-md bg-os-raised px-2 py-1 text-center text-[11px] text-os-muted">
                    Agent tools
                  </div>
                  <div className="mt-1.5 flex flex-wrap justify-center gap-1">
                    {deptTools.map((tool) => (
                      <span key={tool} className="rounded border border-os-border px-1.5 py-0.5 font-mono text-[10px] text-os-muted">
                        {tool}
                      </span>
                    ))}
                  </div>
                </div>
              )}
              </section>
            );
          })}
          </div>
        </div>
      </div>
    </div>
  );
}
