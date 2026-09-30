'use client';

import { useState } from 'react';
import { ArrowUpRight, Loader2, Plus } from 'lucide-react';
import { SlabCard, PILL, chipClass } from '@/components/slab';
import type { PaperclipIssue } from '@/lib/connectors/paperclip';

/**
 * The board queue on /tasks (mock 5f): live Paperclip issues, the real org's
 * work queue, plus a composer that creates a REAL issue the Conductor routes.
 * The routing chips are a hint carried in the issue description; the board's
 * one-assignee rule means the Conductor still does the actual triage. Sits
 * above the local kanban; the two queues are honestly separate systems.
 *
 * 2026-09-24: a Brand Deals SlabCard, routing chips in the slab's pill shape,
 * roomy rows with rounded status pills like the deal list.
 */
const STATUS_TONE: Record<string, string> = {
  done: 'var(--ok)',
  in_progress: 'var(--warn)',
  blocked: 'var(--err)',
};

const ROUTES = ['Conductor routes', 'TECH', 'Sales'] as const;
type Route = (typeof ROUTES)[number];

function ago(iso: string | null): string {
  if (!iso) return '';
  const ms = Date.now() - Date.parse(iso);
  if (!Number.isFinite(ms) || ms < 0) return 'now';
  const m = Math.floor(ms / 60_000);
  if (m < 60) return `${m}m`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h`;
  return `${Math.floor(h / 24)}d`;
}

/** Rounded status pill, tinted by the same tone the queue always used. */
const pillStyle = (status: string) => {
  const tone = STATUS_TONE[status] ?? 'var(--text-2)';
  return { color: tone, background: `color-mix(in oklab, ${tone} 14%, transparent)` };
};

export function BoardTasks({ initialIssues, boardUrl, i = 7 }: { initialIssues: PaperclipIssue[]; boardUrl: string | null; i?: number }) {
  const [issues, setIssues] = useState(initialIssues);
  const [title, setTitle] = useState('');
  const [route, setRoute] = useState<Route>('Conductor routes');
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const create = async () => {
    const t = title.trim();
    if (!t || sending) return;
    setSending(true);
    setError(null);
    try {
      const res = await fetch('/api/board/tasks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: t,
          ...(route === 'Conductor routes' ? {} : { description: `Route to the ${route} pillar.` }),
        }),
      });
      const body = (await res.json()) as { issue?: PaperclipIssue; error?: string };
      if (!res.ok || !body.issue) throw new Error(body.error ?? `HTTP ${res.status}`);
      setIssues((prev) => [body.issue!, ...prev]);
      setTitle('');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSending(false);
    }
  };

  return (
    <SlabCard
      i={i}
      className="mt-6"
      title="Board queue"
      sub={`${issues.length} open issues`}
      action={
        boardUrl && (
          <a href={boardUrl} target="_blank" rel="noreferrer" className={PILL}>
            open board <ArrowUpRight className="h-3.5 w-3.5" />
          </a>
        )
      }
    >
      <div className="px-6 pt-4">
        {/* composer: a real issue for the real org */}
        <div className="mb-3 flex items-center gap-2">
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && create()}
            placeholder="Hand the company work · one line · the Conductor routes it"
            className="min-w-0 flex-1 rounded-full border border-os-border bg-os-bg px-4 py-2 text-[13px] text-os-text placeholder:text-os-dim focus:border-os-border-strong focus:outline-none"
          />
          <button
            onClick={create}
            disabled={sending || !title.trim()}
            className="pressable flex shrink-0 items-center gap-1.5 rounded-full bg-os-accent px-4 py-2 text-[13px] font-semibold text-os-ink disabled:opacity-40"
          >
            {sending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Plus className="h-3.5 w-3.5" />}
            Create issue
          </button>
        </div>
        <div className="mb-4 flex flex-wrap items-center gap-2">
          {ROUTES.map((r) => (
            <button key={r} onClick={() => setRoute(r)} className={chipClass(route === r)}>
              {r}
            </button>
          ))}
        </div>
        {error && <div className="mb-3 text-[12px] text-os-err">Board rejected it: {error}</div>}
      </div>

      {issues.length === 0 ? (
        <div className="border-t border-os-border px-6 py-8 text-center text-[12.5px] text-os-dim">Board unreachable or empty · the live queue shows here.</div>
      ) : (
        <ul className="border-t border-os-border">
          {issues.slice(0, 10).map((issue) => (
            <li
              key={issue.id}
              className="grid grid-cols-[1fr_auto_auto] items-center gap-4 border-b border-os-border px-6 py-3.5 last:border-0 max-[700px]:grid-cols-[1fr_auto]"
            >
              <div className="min-w-0">
                <div className="truncate text-[13.5px] font-medium text-os-text">{issue.title}</div>
                <div className="truncate font-mono text-[11px] text-os-dim">
                  {issue.identifier}
                  {issue.assigneeName && ` · ${issue.assigneeName}`}
                </div>
              </div>
              <span className="rounded-full px-2.5 py-0.5 font-mono text-[10.5px] uppercase tracking-[0.12em]" style={pillStyle(issue.status)}>
                {issue.status.replace(/_/g, ' ')}
              </span>
              <span className="w-[40px] text-right font-mono text-[11px] text-os-dim max-[700px]:hidden">{issue.updatedAt ? ago(issue.updatedAt) : ''}</span>
            </li>
          ))}
        </ul>
      )}
    </SlabCard>
  );
}
