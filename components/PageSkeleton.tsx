/**
 * The instant-paint fallback every route shows on click, before its
 * `force-dynamic` server render (live DB/connector reads) finishes.
 *
 * This is the fix for "the button doesn't respond to my click": without a
 * `loading.tsx`, the App Router keeps the OLD page fully painted and does
 * nothing else until the new page's entire render completes, so a click
 * looks dead for however long the server takes. Once a `loading.tsx` exists,
 * Next swaps to it the instant navigation starts -- same render budget as
 * before, but the screen answers back immediately instead of freezing.
 *
 * It is also what makes prefetching possible at all on a fully dynamic page:
 * Next can only prefetch UP TO the nearest loading boundary, so a route with
 * none has nothing to preload. Every sidebar link sits in the viewport, so
 * Next prefetches this shell for all of them as soon as the shell exists --
 * the "preload while I'm using it" half of the ask, for free.
 *
 * No page-specific content on purpose, but the shape is the Brand Deals slab
 * every page now lands in (2026-09-24): title row, a 2fr/1fr hero with a
 * volume card's meter tracks, then a row of three. A generic shape avoids
 * duplicating each page's real layout just to guess at it, and it is the same
 * `animate-pulse` + hairline-border block already used for lazy-loaded graphs
 * (BrainGraphView, AudienceConsistencyLazy)  -  one visual language, not two.
 */
function Block({ className = '' }: { className?: string }) {
  return <div className={`animate-pulse rounded-[12px] border border-os-border bg-os-surface ${className}`} />;
}

/** The volume card in outline: a big numeral, a caption, three meter tracks. */
function VolumeOutline() {
  return (
    <div className="flex h-72 flex-col rounded-[12px] border border-os-border bg-os-surface px-6 py-5">
      <div className="h-4 w-32 animate-pulse rounded-full bg-os-border" />
      <div className="mt-5 h-10 w-40 animate-pulse rounded-[8px] bg-os-border" />
      <div className="mt-6 flex flex-1 flex-col justify-around border-t border-os-border pt-4">
        {[0.7, 0.45, 0.85].map((w) => (
          <div key={w} data-skel-meter>
            <div className="mb-2 h-2.5 w-24 animate-pulse rounded-full bg-os-border" />
            <div className="h-[10px] w-full rounded-full" style={{ background: 'color-mix(in oklab, var(--text) 6%, transparent)' }}>
              <div className="h-full animate-pulse rounded-full bg-os-border" style={{ width: `${w * 100}%` }} />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export function PageSkeleton() {
  return (
    <div className="os-slab">
      <div className="mb-7 flex items-end justify-between gap-4">
        <div>
          <Block className="mb-3 h-2.5 w-32" />
          <Block className="h-11 w-64" />
          <Block className="mt-3 h-2.5 w-80 max-w-full" />
        </div>
        <Block className="h-9 w-32 !rounded-full" />
      </div>
      <div className="grid grid-cols-[2fr_1fr] gap-6 max-[1200px]:grid-cols-1">
        <Block className="h-72" />
        <VolumeOutline />
      </div>
      <div className="mt-6 grid grid-cols-3 gap-6 max-[1200px]:grid-cols-1">
        <Block className="h-44" />
        <Block className="h-44" />
        <Block className="h-44" />
      </div>
      <Block className="mt-6 h-48" />
    </div>
  );
}
