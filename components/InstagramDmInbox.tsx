'use client';

import { Instagram, Send } from 'lucide-react';
import type { DmThread } from '@/lib/social';
import { initials, relativeTime, useDmInbox } from '@/components/social/use-dm-inbox';

/**
 * The Instagram DM inbox: thread list + conversation pane, with a reply box that
 * sends for real via ManyChat (POST /api/social/dm/reply). Seeded until the
 * ManyChat webhook feeds it live; a message with source 'manychat' flips the
 * "live" pip. Sending is honest — a failed/unconfigured send shows inline and
 * appends nothing. State lives in useDmInbox (shared with the GladOS tree).
 */
export function InstagramDmInbox({ threads: initial, nowMs }: { threads: DmThread[]; nowMs: number }) {
  const { threads, active, setSelected, draft, setDraft, sending, error, isLive, send } = useDmInbox(initial);

  return (
    <div>
      {/* Header — Instagram channel + honest seeded/live status. */}
      <div className="mb-4 flex items-center gap-3 border-b border-os-border pb-3">
        <div className="inline-flex gap-1 rounded-md-t border border-os-border bg-os-surface p-1">
          <span className="flex items-center gap-3 rounded-[5px] bg-[var(--accent-soft)] px-3.5 py-2.5 text-[13px] font-medium text-os-accent">
            <Instagram className="h-3.5 w-3.5" />
            Instagram
            <span className="c-chip">{threads.length}</span>
          </span>
        </div>
        <span className="ml-auto flex items-center gap-2 text-[12.5px] text-os-muted">
          <span className={`dot ${isLive ? 'ok' : 'off'}`} />
          {isLive ? 'Live · ManyChat' : 'Seeded · goes live via the ManyChat webhook'}
        </span>
      </div>

      {threads.length === 0 ? (
        <div className="rounded-lg-t border border-os-border bg-os-surface px-5 py-8 text-center text-[13px] text-os-dim">
          No DMs yet. They appear here as ManyChat posts them to /api/webhooks/manychat.
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-px overflow-hidden rounded-lg-t border border-os-border bg-os-border md:grid-cols-[minmax(0,320px)_minmax(0,1fr)]">
          {/* Thread list */}
          <div className="flex max-h-[520px] flex-col overflow-y-auto bg-os-surface">
            {threads.map((t) => {
              const on = t.subscriberId === active?.subscriberId;
              return (
                <button
                  key={t.subscriberId}
                  onClick={() => setSelected(t.subscriberId)}
                  className={`flex items-center gap-4 border-b border-os-border px-3.5 py-3 text-left transition-colors ${
                    on ? 'bg-os-surface2' : 'hover:bg-os-surface2/50'
                  }`}
                >
                  <span className="grid h-9 w-9 shrink-0 place-items-center rounded-full border border-os-border-strong bg-os-bg text-[12.5px] font-medium text-os-muted">
                    {initials(t.name)}
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-baseline gap-3">
                      <span className="truncate text-[13.5px] font-medium text-os-text">{t.name}</span>
                      <span className="ml-auto shrink-0 font-mono text-[11.5px] text-os-dim">{relativeTime(t.last.ts, nowMs)}</span>
                    </div>
                    <div className="flex items-center gap-2.5">
                      <span className="truncate text-[12.5px] text-os-dim">
                        {t.last.direction === 'out' ? 'You: ' : ''}
                        {t.last.text}
                      </span>
                      {t.unreplied && <span className="dot warn ml-auto" title="needs reply" />}
                    </div>
                  </div>
                </button>
              );
            })}
          </div>

          {/* Conversation + reply box */}
          <div className="flex max-h-[520px] flex-col bg-os-surface">
            {active && (
              <>
                <div className="flex items-center gap-3 border-b border-os-border px-4 py-3">
                  <Instagram className="h-3.5 w-3.5 text-os-dim" />
                  <span className="text-[13.5px] font-medium text-os-text">{active.name}</span>
                  {active.handle && <span className="font-mono text-[12.5px] text-os-dim">@{active.handle}</span>}
                  {active.unreplied && (
                    <span className="ml-auto flex items-center gap-1.5 text-[12px] text-os-muted">
                      <span className="dot warn" />
                      Needs reply
                    </span>
                  )}
                </div>

                <div className="flex flex-1 flex-col gap-3.5 overflow-y-auto px-4 py-4">
                  {active.messages.map((m) => {
                    const out = m.direction === 'out';
                    return (
                      <div key={m.id} className={`flex flex-col ${out ? 'items-end' : 'items-start'}`}>
                        <div
                          className={`glass-well max-w-[78%] px-3 py-3 text-[13.5px] leading-relaxed ${
                            out ? 'gl-on-rim text-os-text' : 'text-os-text'
                          }`}
                        >
                          {m.text}
                        </div>
                        <div className="mt-1 flex items-center gap-2.5 px-1 text-[11px] text-os-dim">
                          {m.tag && <span className="text-os-muted">{m.tag}</span>}
                          <span className="font-mono">{relativeTime(m.ts, nowMs)}</span>
                        </div>
                      </div>
                    );
                  })}
                </div>

                {/* Reply box — sends for real via ManyChat. */}
                <div className="border-t border-os-border px-3 py-3">
                  {error && (
                    <p className="mb-2 flex items-center gap-2 text-[12px] text-os-muted">
                      <span className="dot err" />
                      {error}
                    </p>
                  )}
                  <div className="flex items-end gap-3">
                    <textarea
                      value={draft}
                      onChange={(e) => setDraft(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                          e.preventDefault();
                          void send();
                        }
                      }}
                      rows={1}
                      placeholder={`Reply to ${active.name}…  (⌘↵ to send)`}
                      className="max-h-28 min-h-[38px] flex-1 resize-none rounded-md-t border border-os-border bg-os-bg px-3 py-3 text-[13px] text-os-text placeholder:text-os-dim focus:border-os-border-strong focus:outline-none"
                    />
                    <button
                      onClick={() => void send()}
                      disabled={sending || !draft.trim()}
                      className="c-btn c-btn-primary shrink-0"
                    >
                      <Send className="h-3.5 w-3.5" />
                      {sending ? 'Sending' : 'Send'}
                    </button>
                  </div>
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
