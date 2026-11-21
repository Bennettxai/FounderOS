'use client';

import { Send, Clock, Paperclip, X } from 'lucide-react';
import { Badge } from '@/components/terminal';
import type { SocialPost } from '@/lib/schemas';
import { COMPOSER_PLATFORMS as PLATFORMS, fmtWhen, usePostComposer } from '@/components/social/use-post-composer';

export { QUEUE_UNDO_MS } from '@/components/social/use-post-composer';

export function PostComposer({ initialPosts }: { initialPosts: SocialPost[] }) {
  const {
    caption, setCaption, selected, toggle, mediaUrl, setMediaUrl, scheduledFor, setScheduledFor, busy, error, submit, queued, remove,
  } = usePostComposer(initialPosts);

  return (
    <div className="grid grid-cols-1 gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(280px,0.7fr)]">
      {/* Composer */}
      <div className="rounded-lg-t border border-os-border bg-os-surface p-6">
        <textarea
          value={caption}
          onChange={(e) => setCaption(e.target.value)}
          placeholder="Write a caption"
          rows={4}
          className="w-full resize-none rounded-sm-t border border-os-border bg-os-surface2 px-3 py-3.5 text-[14px] leading-relaxed text-os-text outline-none placeholder:text-os-dim focus:border-os-border-strong"
        />
        <div className="mt-4 flex flex-wrap gap-2.5">
          {PLATFORMS.map((p) => {
            const on = selected.has(p.id);
            return (
              <button
                key={p.id}
                type="button"
                onClick={() => toggle(p.id)}
                className={`rounded-sm-t border px-2.5 py-1 text-[12.5px] transition-colors ${
                  on
                    ? 'border-[var(--accent-line)] bg-[var(--accent-soft)] text-os-accent'
                    : 'border-os-border text-os-dim hover:bg-os-surface2 hover:text-os-muted'
                }`}
              >
                {p.label}
              </button>
            );
          })}
        </div>
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <label className="flex flex-1 items-center gap-3 rounded-sm-t border border-os-border bg-os-surface2 px-2.5 py-2.5">
            <Paperclip className="h-3.5 w-3.5 shrink-0 text-os-dim" />
            <input
              value={mediaUrl}
              onChange={(e) => setMediaUrl(e.target.value)}
              placeholder="media URL (optional)"
              className="w-full bg-transparent font-mono text-[12.5px] text-os-text outline-none placeholder:text-os-dim"
            />
          </label>
          <label className="flex items-center gap-3 rounded-sm-t border border-os-border bg-os-surface2 px-2.5 py-2.5">
            <Clock className="h-3.5 w-3.5 shrink-0 text-os-dim" />
            <input
              type="datetime-local"
              value={scheduledFor}
              onChange={(e) => setScheduledFor(e.target.value)}
              className="bg-transparent font-mono text-[12.5px] text-os-text outline-none [color-scheme:dark]"
            />
          </label>
        </div>
        <div className="mt-4 flex items-center justify-between gap-4">
          <span className="text-[12px] text-os-dim">
            Queues only — the Social agent publishes on its next run.
          </span>
          <button onClick={submit} disabled={busy} className="c-btn c-btn-primary">
            {busy ? 'Queuing…' : <><Send className="h-[13px] w-[13px]" /> Queue post</>}
          </button>
        </div>
        {error && (
          <p className="mt-3 flex items-center gap-2 text-[12.5px] text-os-muted">
            <span className="dot err" />
            {error}
          </p>
        )}
      </div>

      {/* Queue */}
      <div className="rounded-lg-t border border-os-border bg-os-surface p-1">
        <div className="flex items-center justify-between px-3 py-3.5">
          <span className="text-[13px] font-medium text-os-muted">Queue</span>
          <span className="c-chip">{queued.length} pending</span>
        </div>
        <div className="flex max-h-[280px] flex-col gap-1 overflow-y-auto px-1 pb-1">
          {queued.length === 0 && (
            <p className="px-3 py-6 text-center text-[12.5px] text-os-dim">Nothing queued yet</p>
          )}
          {queued.map((post) => (
            <div key={post.id} className="glass-well px-3 py-3.5">
              <p className="line-clamp-2 text-[13px] leading-snug text-os-text">{post.caption}</p>
              <div className="mt-3 flex flex-wrap items-center gap-2.5">
                {post.platforms.map((p) => (
                  <span key={p} className="font-mono text-[11px] text-os-dim">
                    {p}
                  </span>
                ))}
                <span className="ml-auto">
                  <Badge tone={post.scheduledFor ? 'warn' : 'accent'}>
                    {post.scheduledFor ? fmtWhen(post.scheduledFor) : 'queued'}
                  </Badge>
                </span>
                <button
                  type="button"
                  onClick={() => remove(post)}
                  aria-label="Remove from queue"
                  title="Remove from queue (Undo for 5 seconds)"
                  className="grid h-6 w-6 shrink-0 place-items-center rounded-sm-t border border-os-border text-os-dim transition-colors hover:bg-os-surface3 hover:text-os-text focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--accent)]"
                >
                  <X className="h-3 w-3" />
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
