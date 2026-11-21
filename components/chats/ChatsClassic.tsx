'use client';

import Link from 'next/link';
import { useEffect, useRef } from 'react';
import { ArrowLeft, PanelLeftClose, PanelLeftOpen, Plus, RotateCcw, Search, Send, Wrench } from 'lucide-react';
import { PageHeader } from '@/components/PageHeader';
import { AgentAvatar } from '@/components/AgentAvatar';
import { ConductorEmblem } from '@/components/ConductorEmblem';
import { SetupLink } from '@/components/SetupLink';
import { relativeTime } from '@/lib/wizard/helpers';
import type { ChatsView } from '@/lib/chats-view';
import { useChats, WIZARD, type ChatsState } from './useChats';

/**
 * /chats, Classic tree (founderos-ui pages/chats.md §7): PageHeader, one
 * bordered shell with the conversation rail and the open thread, the Classic
 * Wizard's bubbles and composer. Same state hook as the GladOS tree.
 */
export function ChatsClassic({ view, initialView }: { view: ChatsView; initialView: 'rail' | 'thread' }) {
  const c = useChats(view, { initialView });
  const startNew = () => {
    c.setPicking(true);
    c.back();
    if (c.collapsed) c.toggleRail();
  };
  return (
    <div className="ch-c-page">
      <PageHeader
        title="Chats"
        eyebrow="Every conversation with your agents"
        right={
          <div className="flex items-center gap-2.5">
            {view.sample && (
              <span className="c-chip" data-sample-badge="" title={view.mode === 'filming' ? 'Sample data: Filming threads' : 'Sample data: sample threads (?sample=1)'}>
                Sample
              </span>
            )}
            <button type="button" className="c-btn c-btn-primary" onClick={startNew} aria-expanded={c.picking}>
              <Plus className="h-3.5 w-3.5" strokeWidth={2} aria-hidden /> New chat
            </button>
          </div>
        }
      />
      <div className="ch-c-shell rounded-lg-t border border-os-border bg-os-surface">
        <div className="ch-c-grid" data-view={c.mobileView} data-collapsed={c.collapsed || undefined}>
          <Rail c={c} />
          <Strip c={c} />
          <Thread c={c} />
        </div>
      </div>
    </div>
  );
}

function Face({ id, src, name, size }: { id: string; src: string | null; name: string; size: number }) {
  return id === WIZARD ? <ConductorEmblem size={size} glyph="wand" /> : <AgentAvatar src={src} name={name} size={size} />;
}

function Rail({ c }: { c: ChatsState }) {
  const empty = c.allThreadsCount <= 1 && !c.search.trim();
  return (
    <aside className="ch-c-rail">
      <div className="flex items-center justify-between gap-2 px-4 pb-2 pt-3.5">
        <span className="text-[13px] font-medium text-os-muted">{c.picking ? 'New chat' : 'Conversations'}</span>
        {c.picking ? (
          <button type="button" onClick={() => c.setPicking(false)} className="text-[12px] text-os-dim transition-colors hover:text-os-text" title="Back to conversations (Esc)">
            Back
          </button>
        ) : (
          <button type="button" onClick={c.toggleRail} aria-label="Collapse the conversation list" className="wz-icon-btn ch-c-wide-only">
            <PanelLeftClose className="h-3.5 w-3.5" strokeWidth={1.7} aria-hidden />
          </button>
        )}
      </div>
      <div className="px-3 pb-2">
        <label className="ch-c-search">
          <Search className="h-3.5 w-3.5 shrink-0 text-os-dim" strokeWidth={1.7} aria-hidden />
          <input
            type="search"
            value={c.search}
            onChange={(e) => c.setSearch(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Escape' && c.search) {
                e.preventDefault();
                c.setSearch('');
              }
            }}
            placeholder={c.picking ? 'Search agents' : 'Search threads'}
            aria-label={c.picking ? 'Search agents' : 'Search threads'}
          />
        </label>
      </div>
      <div className="ch-c-rail-list">
        {c.picking ? (
          c.roster.length > 0 ? (
            <ul aria-label="Agents without a thread">
              {c.roster.map((r) => (
                <li key={r.id}>
                  <button type="button" className="ch-c-row" onClick={() => c.select(r.id)}>
                    <Face id={r.id} src={r.avatar} name={r.name} size={28} />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-[13px] text-os-text">{r.name}</span>
                      <span className="block truncate text-[11.5px] text-os-dim">{r.description || 'Agent'}</span>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          ) : c.rosterTotal === 0 && c.view.agentCount === 0 ? (
            <p className="px-4 py-3 text-[12.5px] leading-relaxed text-os-dim">
              No agents yet.{' '}
              <Link href="/agents" className="text-os-accent hover:underline">
                Open Agents
              </Link>
            </p>
          ) : (
            <p className="px-4 py-3 text-[12.5px] text-os-dim">{c.rosterTotal === 0 ? 'Every agent already has a thread.' : 'No agent matches that search.'}</p>
          )
        ) : (
          <>
            <ul aria-label="Conversations">
              {c.threads.map((t) => (
                <li key={t.agentId}>
                  <button type="button" className="ch-c-row" aria-pressed={t.agentId === c.selected} onClick={() => c.select(t.agentId)}>
                    <Face id={t.agentId} src={t.avatar} name={t.name} size={30} />
                    <span className="min-w-0 flex-1">
                      <span className="flex items-center gap-2">
                        <span className="truncate text-[13px] text-os-text">{t.name}</span>
                        {t.pinned && <span className="ch-c-pin">pinned</span>}
                        {t.lastAt && <span className="ml-auto shrink-0 font-mono text-[10.5px] text-os-dim">{relativeTime(t.lastAt, c.now)}</span>}
                      </span>
                      <span className="block truncate text-[11.5px] text-os-dim">{t.lastMessage ?? (t.pinned ? t.role : 'No messages yet')}</span>
                    </span>
                  </button>
                </li>
              ))}
            </ul>
            {empty && <p className="px-4 py-3 text-[12.5px] leading-relaxed text-os-dim">No direct chats yet. Start one with New chat.</p>}
            {!empty && c.threads.length === 0 && <p className="px-4 py-3 text-[12.5px] text-os-dim">No threads match that search.</p>}
          </>
        )}
      </div>
    </aside>
  );
}

function Strip({ c }: { c: ChatsState }) {
  return (
    <aside className="ch-c-strip">
      <button type="button" onClick={c.toggleRail} aria-label="Show the conversation list" className="wz-icon-btn">
        <PanelLeftOpen className="h-3.5 w-3.5" strokeWidth={1.7} aria-hidden />
      </button>
      <ul aria-label="Conversations" className="flex flex-col items-center gap-2">
        {c.threads.map((t) => (
          <li key={t.agentId}>
            <button type="button" title={t.name} aria-label={`Open ${t.name}`} aria-pressed={t.agentId === c.selected} onClick={() => c.select(t.agentId)} className="ch-c-strip-btn">
              <Face id={t.agentId} src={t.avatar} name={t.name} size={28} />
            </button>
          </li>
        ))}
      </ul>
    </aside>
  );
}

function Thread({ c }: { c: ChatsState }) {
  const h = c.header;
  const scroller = useRef<HTMLDivElement>(null);
  const field = useRef<HTMLTextAreaElement>(null);
  useEffect(() => {
    const el = scroller.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [c.selected, c.messages.length, c.pending.length, c.routed.length, c.status]);
  useEffect(() => {
    if (field.current && !c.draft) field.current.style.height = 'auto';
  }, [c.draft]);
  const composer = c.view.composer;
  const nothing = c.status === 'ready' && c.messages.length === 0 && c.pending.length === 0 && c.routed.length === 0;
  const submit = () => c.send(c.draft);

  return (
    <section className="ch-c-thread" aria-label={`Conversation with ${h.name}`}>
      <header className="flex items-center gap-3 border-b border-os-border px-5 py-3.5">
        <button type="button" onClick={c.back} aria-label="Back to conversations" className="wz-icon-btn ch-c-narrow-only">
          <ArrowLeft className="h-4 w-4" strokeWidth={1.7} aria-hidden />
        </button>
        <Face id={h.agentId} src={h.avatar} name={h.name} size={34} />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span className="truncate text-[14px] font-medium text-os-text">{h.name}</span>
            {!h.wizard && <span className="c-chip">direct line</span>}
          </div>
          <p className="truncate font-mono text-[11px] text-os-dim">{h.wizard ? (h.projectName ? `latest project · ${h.projectName}` : 'your conductor') : h.role || 'agent'}</p>
        </div>
        <Link href={h.wizard ? '/wizard' : `/agents?agent=${encodeURIComponent(h.agentId)}`} className="c-btn shrink-0">
          {h.wizard ? 'Open Wizard' : 'Open agent'}
        </Link>
      </header>
      <p className="border-b border-os-border px-5 py-2 text-[12px] text-os-dim">
        {h.wizard ? 'The Wizard hands your message to the agent best placed to answer. The reply also lands in that agent’s thread.' : `Direct line: ${h.name} can read your data here, not change it.`}
      </p>

      <div ref={scroller} className="ch-c-msgs" aria-live="polite">
        {c.status === 'loading' && c.messages.length === 0 && (
          <div className="flex flex-col gap-3" aria-label="Loading the thread">
            <span className="ch-c-skel ml-auto h-10 w-[46%]" />
            <span className="ch-c-skel h-4 w-[60%]" />
            <span className="ch-c-skel h-4 w-[36%]" />
          </div>
        )}
        {c.status === 'error' && (
          <div className="glass-well flex items-center gap-2.5 px-3 py-2 text-[12.5px] text-os-muted" role="alert">
            <span className="dot err" /> {c.error ?? 'Could not load the thread.'}
            <button type="button" onClick={c.reload} className="ml-auto inline-flex items-center gap-1.5 text-os-text hover:underline">
              <RotateCcw className="h-3 w-3" strokeWidth={1.8} aria-hidden /> Retry
            </button>
          </div>
        )}
        {nothing && (
          <div className="grid h-full place-items-center text-center">
            <div className="max-w-[340px]">
              <p className="text-[14px] text-os-text">{h.wizard ? 'No Wizard conversation yet' : `No messages with ${h.name} yet`}</p>
              <p className="mt-1.5 text-[12.5px] leading-relaxed text-os-dim">
                {h.wizard ? 'Your latest Wizard project shows here. Ask below and the Wizard routes it.' : 'Ask what it can see. It reads its connectors to answer.'}
              </p>
            </div>
          </div>
        )}
        <div className="flex flex-col gap-4">
          {c.messages.map((m) =>
            m.role === 'tool' ? (
              <div key={m.id} className="flex items-center gap-1.5 pl-[38px] font-mono text-[11px] text-os-dim">
                <Wrench className="h-3 w-3" strokeWidth={1.7} aria-hidden /> read {m.tools.join(', ') || 'a tool'}
              </div>
            ) : m.role === 'user' ? (
              <div key={m.id} className="flex justify-end">
                <div className="wz-bubble-user">
                  <p className="whitespace-pre-wrap text-[13.5px] leading-relaxed text-os-text">{m.content}</p>
                  <span className="mt-1 block text-right text-[10.5px] text-os-dim">{relativeTime(m.createdAt, c.now)}</span>
                </div>
              </div>
            ) : (
              <div key={m.id} className="flex gap-3">
                <Face id={h.agentId} src={h.avatar} name={h.name} size={26} />
                <div className="min-w-0 max-w-[82%]">
                  <p className="whitespace-pre-wrap text-[13.5px] leading-relaxed text-os-text">{m.content}</p>
                  <span className="mt-1 block text-[10.5px] text-os-dim">{relativeTime(m.createdAt, c.now)}</span>
                </div>
              </div>
            ),
          )}
          {c.routed.map((r) => (
            <div key={r.key} className="ch-c-new flex flex-col gap-4">
              <div className="flex justify-end">
                <div className="wz-bubble-user">
                  <p className="whitespace-pre-wrap text-[13.5px] leading-relaxed text-os-text">{r.text}</p>
                </div>
              </div>
              <div className="flex gap-3">
                <AgentAvatar src={r.routedAvatar} name={r.routedName} size={26} />
                <div className="min-w-0 max-w-[82%]">
                  <p className="whitespace-pre-wrap text-[13.5px] leading-relaxed text-os-text">{r.reply}</p>
                  <span className="mt-1 flex items-center gap-2 text-[10.5px] text-os-dim">
                    Answered by {r.routedName}
                    <button type="button" onClick={() => c.select(r.routedTo)} className="text-os-accent hover:underline">
                      Open thread
                    </button>
                  </span>
                </div>
              </div>
            </div>
          ))}
          {c.pending.map((p) => (
            <div key={p.key} className="ch-c-new flex flex-col gap-3">
              <div className="flex justify-end">
                <div className="wz-bubble-user" data-state={p.state}>
                  <p className="whitespace-pre-wrap text-[13.5px] leading-relaxed text-os-text">{p.text}</p>
                  {p.state === 'sending' && <span className="mt-1 block text-right text-[10.5px] text-os-dim">sending</span>}
                </div>
              </div>
              {p.state === 'sending' ? (
                <div className="flex items-center gap-3 text-[12.5px] text-os-muted">
                  <Face id={h.agentId} src={h.avatar} name={h.name} size={26} /> Thinking
                </div>
              ) : (
                <div className="glass-well flex items-center gap-2.5 px-3 py-2 text-[12.5px] text-os-muted" role="alert">
                  <span className="dot err" /> Not sent. {p.error}
                  <button type="button" onClick={() => c.retry(p.key)} className="ml-auto inline-flex items-center gap-1.5 text-os-text hover:underline">
                    <RotateCcw className="h-3 w-3" strokeWidth={1.8} aria-hidden /> Retry
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      <div className="px-4 pb-4 pt-2">
        {composer.reason === 'not_configured' && (
          <div className="mb-2 flex flex-wrap items-center gap-2.5 text-[12.5px] text-os-muted">
            <span className="c-chip">Not configured</span>
            Agent chat needs a model key.
            <SetupLink href={c.view.model.setupHref} hint="Add the AI Gateway key the agent chat uses" />
          </div>
        )}
        {composer.reason === 'sample' && <p className="mb-2 text-[12.5px] text-os-dim">Sample threads. Sending is off in this preview.</p>}
        <div className="wz-composer" data-disabled={!composer.enabled || undefined}>
          <textarea
            ref={field}
            value={c.draft}
            rows={1}
            disabled={!composer.enabled}
            aria-label={`Message ${h.name}`}
            placeholder={`Message ${h.name}…`}
            onChange={(e) => {
              c.setDraft(e.target.value);
              e.target.style.height = 'auto';
              e.target.style.height = `${Math.min(160, e.target.scrollHeight)}px`;
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
                e.preventDefault();
                submit();
              }
            }}
            className="max-h-40 w-full resize-none bg-transparent px-3 py-2.5 text-[13.5px] text-os-text outline-none placeholder:text-os-dim disabled:cursor-not-allowed"
          />
          <div className="flex items-center justify-between gap-2 px-2 pb-1.5">
            <span className="text-[10.5px] text-os-dim">{composer.enabled ? 'Enter sends · Shift+Enter for a new line' : ''}</span>
            <button type="button" onClick={submit} disabled={!composer.enabled || c.busy || !c.draft.trim()} className="wz-send">
              <Send className="h-3.5 w-3.5" strokeWidth={2.2} aria-hidden />
              Send
            </button>
          </div>
        </div>
      </div>
    </section>
  );
}
