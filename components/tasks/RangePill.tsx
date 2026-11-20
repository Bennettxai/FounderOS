'use client';

import { useRouter, useSearchParams, usePathname } from 'next/navigation';
import { useState, useRef, useEffect } from 'react';
import { Calendar, ChevronDown, Download } from 'lucide-react';
import { TASK_RANGES, type RangeId } from '@/lib/tasks-metrics';

/**
 * The reference's "Last 30 days" pill and black Export button
 * (docs/TASKS-REDESIGN.md §5, zone 0). The range lives in the URL, the way
 * /funnel keeps its view state, so a reload or a shared link reproduces the
 * page exactly — and every metric below reads the same server-side value.
 */
export function RangePill({ range }: { range: RangeId }) {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const current = TASK_RANGES.find((r) => r.id === range) ?? TASK_RANGES[1];

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener('mousedown', onDown);
    return () => document.removeEventListener('mousedown', onDown);
  }, [open]);

  const pick = (id: RangeId) => {
    const next = new URLSearchParams(params.toString());
    next.set('range', id);
    router.replace(`${pathname}?${next.toString()}`, { scroll: false });
    setOpen(false);
  };

  return (
    <div className="flex items-center gap-2">
      <div className="relative" ref={ref}>
        <button
          onClick={() => setOpen((o) => !o)}
          className="flex items-center gap-2 rounded-md-t border border-os-border bg-os-surface2 px-3 py-1.5 text-[12.5px] text-os-muted transition-colors hover:bg-os-surface3"
          aria-expanded={open}
        >
          <Calendar className="h-3.5 w-3.5" strokeWidth={1.6} />
          {current.label}
          <ChevronDown className={`h-3.5 w-3.5 transition-transform ${open ? 'rotate-180' : ''}`} strokeWidth={1.6} />
        </button>
        {open && (
          <div className="glass-solid absolute right-0 z-20 mt-1 w-[168px] overflow-hidden rounded-md-t border border-os-border bg-os-surface shadow-lg">
            {TASK_RANGES.map((r) => (
              <button
                key={r.id}
                onClick={() => pick(r.id)}
                className={`block w-full px-3 py-2 text-left text-[12.5px] transition-colors hover:bg-os-surface3 ${
                  r.id === range ? 'text-[var(--accent)]' : 'text-os-muted'
                }`}
              >
                {r.label}
              </button>
            ))}
          </div>
        )}
      </div>

      <a
        href={`/api/tasks/export?range=${range}`}
        className="flex items-center gap-2 rounded-md-t bg-os-text px-3 py-1.5 text-[12.5px] font-medium text-os-bg transition-opacity hover:opacity-85"
        download
      >
        <Download className="h-3.5 w-3.5" strokeWidth={1.8} />
        Export
      </a>
    </div>
  );
}
