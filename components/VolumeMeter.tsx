/**
 * The Deal Volume bar: a label/value row over a rounded track whose hatched,
 * glowing fill sweeps out from the left on mount. Brand Deals introduced it;
 * Home wears it too (Alex, 2026-09-24: "I like the way the deal volume bars
 * go out"). Pure presentation: every caller passes its own numbers.
 */
const EASE = 'cubic-bezier(.2,.7,.2,1)';

export function VolumeMeter({ label, frac: rawFrac, display, hue, delay = 0 }: { label: string; frac: number; display: string; hue: string; delay?: number }) {
  const frac = Number.isFinite(rawFrac) && rawFrac > 0 ? Math.max(0.02, Math.min(1, rawFrac)) : 0.02;
  return (
    <div>
      <div className="flex items-baseline justify-between gap-3">
        <span className="text-[13.5px] text-os-muted">{label}</span>
        <span className="text-[14px] font-semibold tabular-nums">{display}</span>
      </div>
      <div className="mt-2 h-[10px] overflow-hidden rounded-full" style={{ background: 'color-mix(in oklab, var(--text) 8%, transparent)' }}>
        <div
          className="vol-fill h-full rounded-full"
          style={{
            width: `${frac * 100}%`,
            background: `linear-gradient(90deg, transparent 72%, color-mix(in oklab, ${hue} 60%, white) 100%), repeating-linear-gradient(45deg, ${hue}, ${hue} 6px, color-mix(in oklab, ${hue} 45%, transparent) 6px, color-mix(in oklab, ${hue} 45%, transparent) 12px)`,
            boxShadow: `0 0 14px color-mix(in oklab, ${hue} 45%, transparent), inset 0 0 5px color-mix(in oklab, ${hue} 55%, transparent)`,
            animation: `vol-meter-in 1.2s ${EASE} ${delay}ms both`,
          }}
        />
      </div>
    </div>
  );
}
