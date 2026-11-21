'use client';

import { useState } from 'react';
import type { Agent, Broadcast } from '@/lib/schemas';
import { ConductorEmblem } from '@/components/ConductorEmblem';

/**
 * The "AI Head" card from the GladOS board: the Wizard super-agent
 * with its chat pill. Sending broadcasts the message to every agent in
 * parallel via POST /api/agents/broadcast; replies expand below.
 */
export function ConductorCard({
  conductor,
  agentNames,
  initialBroadcast,
}: {
  conductor: Agent;
  agentNames: Record<string, string>;
  initialBroadcast: Broadcast | null;
}) {
  const [message, setMessage] = useState('');
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [broadcast, setBroadcast] = useState<Broadcast | null>(initialBroadcast);
  const [showReplies, setShowReplies] = useState(false);

  async function send() {
    const text = message.trim();
    if (!text || sending) return;
    setSending(true);
    setError(null);
    try {
      const res = await fetch('/api/agents/broadcast', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ message: text }),
      });
      if (!res.ok) throw new Error(`broadcast failed (${res.status})`);
      const body = (await res.json()) as { broadcast: Broadcast };
      setBroadcast(body.broadcast);
      setShowReplies(true);
      setMessage('');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSending(false);
    }
  }

  return (
    <div className="hoverable group w-[340px] rounded-lg-t border border-os-border-bright bg-os-surface p-6">
      <div className="text-center text-[12.5px] text-os-muted">AI head</div>

      {/* Living core — breathes and orbits while the Wizard is broadcasting */}
      <div className="mt-3 flex justify-center">
        <ConductorEmblem size={62} thinking={sending} />
      </div>
      <div className="mt-3 text-center text-[14.5px] font-medium">Wizard</div>
      <div className="text-center text-[11.5px] text-os-dim">
        super agent · {conductor.instance} runtime until the dedicated host lands
      </div>

      {/* Chat pill → broadcast to every agent */}
      <div className="mt-4 flex gap-2.5">
        <input
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && send()}
          placeholder="Chat with Wizard — reaches every agent"
          className="min-w-0 flex-1 rounded-sm-t border border-os-border bg-os-bg px-3 py-2.5 text-[13px] text-os-text placeholder:text-os-dim focus:border-os-border-strong focus:outline-none"
          disabled={sending}
        />
        <button
          onClick={send}
          disabled={sending || !message.trim()}
          className="c-btn c-btn-primary h-auto shrink-0 px-3.5 py-2.5 text-[13px]"
        >
          {sending ? '…' : 'Send'}
        </button>
      </div>
      {error && <p className="mt-1.5 text-[12.5px] text-os-muted">✗ {error}</p>}

      {/* Capability pills */}
      <div className="mt-4 grid grid-cols-3 gap-1">
        {['Broadcast', 'Orchestration', 'Instances'].map((cap) => (
          <span key={cap} className="glass-chip truncate rounded-full px-1.5 py-1 text-center text-[11px] text-os-muted">
            {cap}
          </span>
        ))}
      </div>

      {/* Agent tools bar */}
      <div className="mt-3 rounded-md bg-os-raised px-2 py-1 text-center text-[11.5px] text-os-muted">
        Agent tools
      </div>
      <div className="mt-1.5 flex justify-center gap-1">
        {conductor.tools.map((tool) => (
          <span key={tool} className="rounded border border-os-border px-1.5 py-0.5 font-mono text-[10px] text-os-muted">
            {tool}
          </span>
        ))}
      </div>

      {/* Latest broadcast + replies */}
      {broadcast && (
        <div className="mt-4 border-t border-os-border pt-2">
          <button
            onClick={() => setShowReplies((v) => !v)}
            className="flex w-full items-baseline justify-between gap-3 text-left"
          >
            <span className="truncate text-[12.5px] text-os-muted">«{broadcast.message}»</span>
            <span className="shrink-0 text-[11.5px] text-os-dim">
              {broadcast.replies.filter((r) => r.ok).length}/{broadcast.replies.length} ok {showReplies ? '▾' : '▸'}
            </span>
          </button>
          {showReplies && (
            <ul className="mt-3 max-h-56 space-y-1 overflow-y-auto pr-1">
              {broadcast.replies.map((reply) => (
                <li key={reply.id} className="flex items-start gap-2.5 rounded-md bg-os-raised px-2 py-2.5">
                  <span
                    className={`mt-1 h-1 w-1 shrink-0 rounded-full ${reply.ok ? 'bg-os-text' : 'border border-os-dim bg-transparent'}`}
                  />
                  <div className="min-w-0">
                    <span className="text-[11.5px] font-medium">{agentNames[reply.agentId] ?? reply.agentId}</span>
                    <span className="break-words text-[11.5px] leading-relaxed text-os-muted"> — {reply.reply}</span>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </div>
  );
}
