'use client';

import Link from 'next/link';
import { useEffect, useState } from 'react';

export function FunnelLayoutToggle({ layout, archived, options }: {
  layout: 'flow' | 'radial'; archived: boolean;
  options: { id: 'flow' | 'radial'; label: string; href: string }[];
}) {
  const [selected, setSelected] = useState(layout);
  useEffect(() => setSelected(layout), [layout, archived]);
  return <nav aria-label="Funnel graph view" className="relative isolate grid grid-cols-2 rounded-ctl border border-os-border p-0.5 font-mono text-[10px] uppercase tracking-wide">
    <span aria-hidden="true" className={`pointer-events-none absolute bottom-0.5 left-0.5 top-0.5 -z-10 w-[calc(50%-2px)] bg-[var(--accent-soft)] transition-transform duration-200 motion-reduce:transition-none ${archived ? 'opacity-40' : ''}`}
      style={{ transform: `translateX(${selected === 'radial' ? '100%' : '0%'})` }} />
    {options.map(option => <Link key={option.id} href={option.href} scroll={false} data-lens="c"
      onClick={() => setSelected(option.id)} aria-current={!archived && layout === option.id ? 'page' : undefined}
      className={`pressable px-3 py-1.5 text-center focus-visible:outline focus-visible:outline-2 ${selected === option.id ? 'text-os-accent' : 'text-os-dim hover:text-os-muted'}`}>
      {option.label}
    </Link>)}
  </nav>;
}
