'use client';

/**
 * Canon drift: contradictions between what the knowledge canon says and what
 * gets claimed in incoming context (sales calls, threads). The detection
 * pipeline that would diff live transcripts against the brain store doesn't
 * run yet — the insights render from `lib/drift.ts`; an empty array renders
 * the not-configured state.
 */
import { useState } from 'react';
import Link from 'next/link';
import { CheckCircle2, ChevronDown, ChevronRight, Copy, FileEdit, Mail } from 'lucide-react';
import { SectionHead } from '@/components/terminal';
import { useToast } from '@/components/ui/Toast';
import type { DriftInsight } from '@/lib/drift';

// Nothing on this page sends or writes anything (GladOS UI H35): the draft and
// the proposed canon passage are offered for copying, and the operator sends
// or edits by hand.
export const DRAFT_FOOTER = 'Not sent. Copy it, or send from';

function CopyButton({ text, label, what }: { text: string; label: string; what: string }) {
  const { toast } = useToast();
  const onCopy = async () => {
    try {
      if (!navigator.clipboard) throw new Error('no clipboard');
      await navigator.clipboard.writeText(text);
      toast({ text: `${what} copied`, tone: 'ok' });
    } catch {
      toast({ text: 'Copy failed: select the text and copy it by hand', tone: 'err' });
    }
  };
  return (
    <button type="button" onClick={onCopy} className="c-btn text-[12px]" aria-label={label} title={label}>
      <Copy className="h-3.5 w-3.5" />
      Copy
    </button>
  );
}

// Turns a store-relative path like "Calls/vertex-freight-pricing-21.md" into
// a sentence-case label for the occurrence chip — the raw slug is kept in
// the mono source line underneath once the chip is expanded.
export function humanizeSource(source: string): string {
  const stem = (source.split('/').pop() ?? source).replace(/\.md$/i, '');
  const words = stem
    .split('-')
    .filter((w) => !/^\d+$/.test(w))
    .map((w) => (w.toLowerCase() === 'qbr' ? 'QBR' : w));
  const text = words.join(' ');
  return text.charAt(0).toUpperCase() + text.slice(1);
}

// Static, per-insight demo content for the two expandable actions. This is
// presentation copy for the fixture, not part of the DriftInsight schema —
// a real pipeline would generate both from the live canon diff.
export const PROPOSED_FIX: Record<
  string,
  { removed: string; added: string; to: string; message: string }
> = {
  'drift-tier1-scope': {
    removed:
      'Tier 1 is infrastructure and brain access only. Custom agents are always Tier 2, scoped and priced live — never pre-quoted, never folded into the Tier 1 number.',
    added:
      'Tier 1 is infrastructure and brain access only. Custom agents are always Tier 2, scoped and priced live — never pre-quoted, never folded into the Tier 1 number. Describing Tier 1 as agent-inclusive on a pricing call is a pitch error, not a pricing exception.',
    to: 'sales director',
    message:
      'Three pricing calls this week (Vertex Freight, Kestrel Capital, Juniper & Sons) described Tier 1 as including a custom agent. Canon (Sales/pricing-model.md) is unchanged — Tier 1 is infra + brain only, agents are Tier 2. Flagging so the next pricing call does not repeat it.',
  },
  'drift-lumen-archive-scope': {
    removed:
      'Onboarding scope is the live workflow only — tagging and delivery going forward. Historical archive migration is never bundled; it is quoted and billed as a separate project.',
    added:
      'Onboarding scope is the live workflow only — tagging and delivery going forward. Historical archive migration is never bundled; it is quoted and billed as a separate project. Lumen Studios has been told otherwise on three calls and is owed a scope correction before renewal.',
    to: 'account lead',
    message:
      'Lumen Studios has been told across three calls that historical archive migration is part of onboarding. It is not — Operations/client-onboarding-playbook.md treats it as a separate billed project. Recommend a scope-correction call before the account reaches renewal.',
  },
  'drift-tinderbox-churn': {
    removed: 'Status: active. Sector media. Account lead J. Cole. This page is the source of truth; anything that contradicts it is stale.',
    added:
      'Status: churned. Sector media. Account lead J. Cole. Closed 2026-08-16 — seats had been at zero for six weeks; confirmed via the Q3 QBR prep, not a renewal call.',
    to: 'sales director',
    message:
      'Canon updated 2026-08-16 — Tinderbox Collective moved from active to churned. Seat usage had been at zero for six weeks across the escalation, technical review, and QBR prep.',
  },
};

function ConfidenceChip({ confidence }: { confidence: DriftInsight['confidence'] }) {
  return (
    <span className="glass-chip px-2.5 py-1 text-[10.5px] text-os-muted">
      {confidence === 'high' ? 'High confidence' : 'Medium confidence'}
    </span>
  );
}

function DriftCard({ insight }: { insight: DriftInsight }) {
  const resolved = insight.state === 'resolved';
  const [collapsed, setCollapsed] = useState(resolved);
  const [openOcc, setOpenOcc] = useState<Set<number>>(new Set());
  const [showDiff, setShowDiff] = useState(false);
  const [showDraft, setShowDraft] = useState(false);
  const fix = PROPOSED_FIX[insight.id];

  const toggleOcc = (i: number) => {
    setOpenOcc((prev) => {
      const next = new Set(prev);
      if (next.has(i)) next.delete(i);
      else next.add(i);
      return next;
    });
  };

  return (
    <div data-resolved={resolved || undefined} className={`glass-panel-violet hoverable p-5 ${resolved ? 'gl-dim' : ''}`}>
      {/* claim + resolved state */}
      <button
        onClick={() => resolved && setCollapsed((c) => !c)}
        className={`flex w-full items-start gap-2 text-left ${resolved ? 'cursor-pointer' : 'cursor-default'}`}
      >
        {resolved && (
          <span className="mt-0.5 shrink-0 text-os-dim">
            {collapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
          </span>
        )}
        <span className="flex-1 text-[14.5px] leading-snug text-os-text">{insight.claim}</span>
        {resolved && (
          <span className="glass-chip mt-0.5 flex shrink-0 items-center gap-1.5 px-2.5 py-1 text-[10.5px] text-os-muted">
            <CheckCircle2 className="h-3 w-3" />
            Resolved
          </span>
        )}
      </button>

      {!collapsed && (
        <div className="mt-4 flex flex-col gap-4">
          {/* occurrence strip */}
          <div>
            <div className="glass-label mb-1.5">Occurrences</div>
            <div className="flex flex-wrap gap-1.5">
              {insight.occurrences.map((occ, i) => (
                <button
                  key={`${occ.source}-${i}`}
                  onClick={() => toggleOcc(i)}
                  className="glass-chip flex items-center gap-1.5 px-2.5 py-1 text-[11px] text-os-muted transition-colors hover:text-os-text"
                  title={occ.source}
                >
                  {humanizeSource(occ.source)} <span className="text-os-dim">·</span> {occ.date}
                </button>
              ))}
            </div>
            {openOcc.size > 0 && (
              <div className="mt-2 grid gap-2">
                {insight.occurrences.map((occ, i) =>
                  openOcc.has(i) ? (
                    <div key={`${occ.source}-${i}-excerpt`} className="glass-well p-3">
                      <div className="flex items-center justify-between gap-3 text-[10.5px] text-os-dim">
                        <span>{occ.speaker}</span>
                        <span className="font-mono">{occ.source}</span>
                      </div>
                      <p className="mt-1.5 font-mono text-[11px] leading-relaxed text-os-muted">&ldquo;{occ.excerpt}&rdquo;</p>
                    </div>
                  ) : null,
                )}
              </div>
            )}
          </div>

          {/* canon passage */}
          <div className="glass-well p-3">
            <div className="flex items-center justify-between gap-3">
              <span className="glass-label">Canon</span>
              <span className="font-mono text-[10.5px] text-os-dim">{insight.canon.file}</span>
            </div>
            <p className="mt-1.5 text-[12px] leading-relaxed text-os-muted">{insight.canon.passage}</p>
          </div>

          {/* footer: confidence, detected, actions */}
          <div className="flex flex-wrap items-center gap-2">
            <ConfidenceChip confidence={insight.confidence} />
            <span className="font-mono text-[10.5px] text-os-dim">detected {insight.detectedAt}</span>
            <span className="flex-1" />
            {/* edge 9: with no proposed fix there is nothing to reveal, so no buttons */}
            {fix && (
              <>
                <button type="button" onClick={() => setShowDiff((v) => !v)} aria-expanded={showDiff} className="c-btn text-[12px]">
                  <FileEdit className="h-3.5 w-3.5" />
                  Update canon
                </button>
                <button type="button" onClick={() => setShowDraft((v) => !v)} aria-expanded={showDraft} className="c-btn text-[12px]">
                  <Mail className="h-3.5 w-3.5" />
                  Draft correction
                </button>
              </>
            )}
          </div>

          {showDiff && fix && (
            <div className="glass-well p-3 font-mono text-[11px] leading-relaxed">
              <div className="mb-1.5 text-[10.5px] uppercase tracking-[0.2em] text-os-dim">Proposed diff · {insight.canon.file}</div>
              <div className="rounded-[6px] bg-[var(--drift-del-bg)] px-2 py-1.5 text-[var(--drift-del-fg)]">− {fix.removed}</div>
              <div className="mt-1 rounded-[6px] bg-[var(--drift-add-bg)] px-2 py-1.5 text-[var(--diff-add-fg)]">+ {fix.added}</div>
              <div className="mt-2 flex flex-wrap items-center justify-between gap-2 font-sans">
                <span className="text-[10.5px] text-os-dim">Not written. Copy the new passage into {insight.canon.file}.</span>
                <CopyButton text={fix.added} label={`Copy the new passage for ${insight.canon.file}`} what="New passage" />
              </div>
            </div>
          )}

          {showDraft && fix && (
            <div className="glass-well p-3">
              <div className="text-[11px] text-os-dim">
                To: <span className="text-os-muted">{fix.to}</span>
              </div>
              <p className="mt-1.5 text-[12px] leading-relaxed text-os-muted">{fix.message}</p>
              <div className="mt-2 flex flex-wrap items-center justify-between gap-2">
                <span className="text-[10.5px] text-os-dim">
                  {DRAFT_FOOTER}{' '}
                  <Link href="/inbox" className="text-os-muted underline underline-offset-2 hover:text-os-text">
                    Comms
                  </Link>
                  .
                </span>
                <CopyButton text={fix.message} label={`Copy the draft to the ${fix.to}`} what="Draft" />
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export function DriftRail({ insights }: { insights: DriftInsight[] }) {
  // Empty insights = the detection pipeline is not configured. The rail still
  // shows itself honestly (production ships with zero insights until the
  // pipeline runs); the labeled demo sequence only appears when insights exist.
  if (insights.length === 0) {
    return (
      <div className="mt-6">
        <SectionHead label="Drift" count="not configured" />
        <div
          className="mt-3 rounded-[16px] border border-dashed px-5 py-6 text-[12px] leading-relaxed"
          style={{ borderColor: 'var(--drift-empty-border)', color: 'var(--text-2, var(--drift-empty-fg))' }}
        >
          Canon-drift detection compares claims in incoming calls and threads against the knowledge canon and
          surfaces contradictions here. The extraction pipeline is not configured yet — no insights to show.
        </div>
      </div>
    );
  }
  return (
    <div className="mt-6">
      <SectionHead label="Drift" count={`${insights.filter((i) => i.state !== 'resolved').length} open`} />
      <p className="-mt-1 mb-4 max-w-[92ch] text-[12px] leading-relaxed text-os-muted">
        Claims made in incoming calls and threads, checked against the knowledge canon. When the same
        contradiction repeats across independent sources, it surfaces here with the evidence on both sides.
      </p>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {insights.map((insight) => (
          <DriftCard key={insight.id} insight={insight} />
        ))}
      </div>
    </div>
  );
}
