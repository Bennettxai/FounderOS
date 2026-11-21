'use client';

import { useEffect, useRef, useState } from 'react';
import { Check, CircleCheck, RotateCcw, Send, Circle, Zap, ListChecks } from 'lucide-react';
import { DeleteProjectConfirm } from '@/components/wizard/DeleteProjectConfirm';
import { ConductorEmblem } from '@/components/ConductorEmblem';
import type { WizardProject } from '@/lib/wizard/store';
import { elapsedLabel, relativeTime, shortSessionId } from '@/lib/wizard/helpers';
import { projectProgress, tokensLabel } from '@/lib/wizard/ui';

export function Conversation({
  project,
  pending,
  pendingSeconds,
  engineError,
  now,
  onSend,
  onRename,
  onStatus,
  onTask,
  onDelete,
  onNew,
}: {
  project: WizardProject | null;
  pending: boolean;
  pendingSeconds: number;
  engineError: string | null;
  now: number;
  onSend: (text: string) => void;
  onRename: (name: string) => void;
  onStatus: (status: 'active' | 'done') => void;
  onTask: (taskId: string, state: 'open' | 'done') => void;
  onDelete: () => void;
  onNew: () => void;
}) {
  const [draft, setDraft] = useState('');
  const [editing, setEditing] = useState(false);
  const [nameDraft, setNameDraft] = useState('');
  const scroller = useRef<HTMLDivElement>(null);
  const textarea = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    scroller.current?.scrollTo({ top: scroller.current.scrollHeight });
  }, [project?.messages.length, pending]);

  useEffect(() => {
    setEditing(false);
  }, [project?.id]);

  function submit() {
    const text = draft.trim();
    if (!text || pending) return;
    onSend(text);
    setDraft('');
    if (textarea.current) textarea.current.style.height = 'auto';
  }

  if (!project) {
    return (
      <div className="wz-panel grid place-items-center">
        <div className="text-center">
          <ConductorEmblem size={64} />
          <p className="mt-4 text-[15px] text-os-text">Start a project</p>
          <p className="mt-1 max-w-[320px] text-[12.5px] text-os-muted">Say what you want done. The Wizard plans it, hands work to the crew and tracks it here.</p>
          <button type="button" onClick={onNew} className="wz-new mt-5 inline-flex">
            New project
          </button>
        </div>
      </div>
    );
  }

  const progress = projectProgress(project);
  const sess = shortSessionId(project.sessionId);

  return (
    <div className="wz-panel flex min-h-0 flex-col">
      {/* header */}
      <div className="flex items-center gap-3 border-b border-os-hairline px-5 py-3">
        <div className="min-w-0 flex-1">
          {editing ? (
            <input
              autoFocus
              value={nameDraft}
              onChange={(e) => setNameDraft(e.target.value)}
              onBlur={() => {
                setEditing(false);
                if (nameDraft.trim() && nameDraft.trim() !== project.name) onRename(nameDraft.trim());
              }}
              onKeyDown={(e) => {
                if (e.key === 'Enter') (e.target as HTMLInputElement).blur();
                if (e.key === 'Escape') setEditing(false);
              }}
              className="w-full bg-transparent text-[15px] text-os-text outline-none"
            />
          ) : (
            <button
              type="button"
              onClick={() => {
                setNameDraft(project.name);
                setEditing(true);
              }}
              className="max-w-full truncate text-left text-[15px] text-os-text hover:text-os-accent2"
              title="Rename"
            >
              {project.name}
            </button>
          )}
          <div className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-[11.5px] text-os-dim">
            <span className={`dot ${project.status === 'done' ? 'ok' : 'warn'}`} />
            <span>{project.status === 'done' ? 'done' : 'active'}</span>
            {sess && (
              <>
                <span aria-hidden="true">·</span>
                <span className="font-mono">session {sess}</span>
              </>
            )}
            {project.model && (
              <>
                <span aria-hidden="true">·</span>
                <span className="font-mono">{project.model}</span>
              </>
            )}
            {project.lastUsage && (
              <>
                <span aria-hidden="true">·</span>
                <span className="tabular-nums">{tokensLabel(project.lastUsage.inputTokens + project.lastUsage.outputTokens)} tokens last turn</span>
              </>
            )}
            {project.totalCostUsd > 0 && (
              <>
                <span aria-hidden="true">·</span>
                <span className="tabular-nums">${project.totalCostUsd.toFixed(2)}</span>
              </>
            )}
          </div>
        </div>
        {project.status === 'done' ? (
          <button type="button" onClick={() => onStatus('active')} className="c-btn">
            <RotateCcw className="h-3.5 w-3.5" /> Reopen
          </button>
        ) : (
          <button type="button" onClick={() => onStatus('done')} className="c-btn">
            <CircleCheck className="h-3.5 w-3.5" /> Mark done
          </button>
        )}
        <DeleteProjectConfirm onDelete={onDelete} disabled={pending} />
      </div>

      {/* tasks strip */}
      {project.tasks.length > 0 && (
        <div className="flex flex-wrap items-center gap-1.5 border-b border-os-hairline px-5 py-2.5">
          <span className="mr-1 flex items-center gap-1.5 text-[11.5px] text-os-muted">
            <ListChecks className="h-3.5 w-3.5" />
            {progress?.done}/{progress?.total}
          </span>
          {project.tasks.map((t) => (
            <button
              key={t.id}
              type="button"
              onClick={() => onTask(t.id, t.state === 'done' ? 'open' : 'done')}
              className={`wz-task ${t.state === 'done' ? 'wz-task-done' : ''}`}
              title={`${t.ownerName} · click to ${t.state === 'done' ? 'reopen' : 'mark done'}`}
            >
              {t.state === 'done' ? <Check className="h-3 w-3 shrink-0" strokeWidth={2.5} /> : <Circle className="h-3 w-3 shrink-0" strokeWidth={2} />}
              <span className="min-w-0 truncate">{t.title}</span>
              <span className="shrink-0 whitespace-nowrap text-os-dim">{t.ownerName}</span>
            </button>
          ))}
        </div>
      )}

      {/* messages */}
      <div ref={scroller} className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
        {project.messages.length === 0 && !pending && (
          <p className="pt-6 text-center text-[12.5px] text-os-dim">Say what you need. The first line names the project.</p>
        )}
        <div className="flex flex-col gap-4">
          {project.messages.map((m, i) =>
            m.role === 'user' ? (
              <div key={i} className="flex justify-end">
                <div className="wz-bubble-user animate-wizard-row-in">
                  <p className="whitespace-pre-wrap text-[13.5px] leading-relaxed text-os-text">{m.text}</p>
                  <span className="mt-1 block text-right text-[10.5px] text-os-dim">{relativeTime(m.at, now)}</span>
                </div>
              </div>
            ) : (
              <div key={i} className="flex gap-3 animate-wizard-row-in">
                <ConductorEmblem size={26} className="mt-0.5" />
                <div className="min-w-0 max-w-[82%]">
                  <p className="whitespace-pre-wrap text-[13.5px] leading-relaxed text-os-text">{m.text}</p>
                  {m.receipts && m.receipts.length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-1.5">
                      {m.receipts.map((r, j) => (
                        <span key={j} className={`wz-chip ${r.kind === 'ran-agent' ? 'wz-chip-accent' : ''}`}>
                          {r.kind === 'ran-agent' ? <Zap className="h-3 w-3" /> : r.kind === 'done' ? <Check className="h-3 w-3" /> : <ListChecks className="h-3 w-3" />}
                          {r.label}
                        </span>
                      ))}
                    </div>
                  )}
                  <span className="mt-1 block text-[10.5px] text-os-dim">{relativeTime(m.at, now)}</span>
                </div>
              </div>
            ),
          )}
          {pending && (
            <div className="flex items-center gap-3 text-[12.5px] text-os-muted">
              <ConductorEmblem size={26} thinking />
              Wizard is working · {elapsedLabel(pendingSeconds)}
            </div>
          )}
          {engineError && (
            <div className="glass-well flex items-center gap-2 px-3 py-2 text-[12px] text-os-muted">
              <span className="dot err" />
              {engineError}
            </div>
          )}
        </div>
      </div>

      {/* composer */}
      <div className="px-4 pb-4 pt-2">
        <div className="wz-composer">
          <textarea
            ref={textarea}
            value={draft}
            rows={1}
            placeholder="Ask the Wizard"
            onChange={(e) => {
              setDraft(e.target.value);
              e.target.style.height = 'auto';
              e.target.style.height = `${Math.min(160, e.target.scrollHeight)}px`;
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                e.preventDefault();
                submit();
              }
            }}
            className="max-h-40 w-full resize-none bg-transparent px-3 py-2 text-[13.5px] text-os-text outline-none placeholder:text-os-dim"
          />
          <div className="flex items-center justify-between px-2 pb-1.5">
            <span className="text-[10.5px] text-os-dim">⌘⏎ to send</span>
            <button type="button" onClick={submit} disabled={pending || !draft.trim()} className="wz-send">
              <Send className="h-3.5 w-3.5" strokeWidth={2.2} />
              Send
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
