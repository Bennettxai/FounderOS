'use client';

import { useEffect, useState } from 'react';
import { usePathname } from 'next/navigation';
import { Search } from 'lucide-react';
import { BRAND } from '@/lib/brand';
import { WizardIcon } from '@/lib/nav';
import { GButton } from '@/components/g';
import type { UiStyle } from '@/lib/appearance';
import { useUiStyle } from '@/lib/use-ui-style';
import { LegacyTopbarMark } from '@/components/legacy/Shell';
import {
  CONDUCTOR_STATE_EVENT,
  CONDUCTOR_TOGGLE_EVENT,
  type ConductorStateDetail,
} from '@/lib/conductor-launcher';

const SEGMENT_LABELS: Record<string, string> = {
  '': 'home',
  social: 'social',
  comms: 'comms',
  agents: 'agents',
  org: 'org-chart',
  brain: 'optimal-engine',
  integrations: 'connections',
  roadmap: 'roadmap',
  analytics: 'analytics',
  database: 'brain-files',
};

export function openPalette() {
  window.dispatchEvent(new CustomEvent('glados:palette'));
}

export function Topbar({ ui = 'classic' }: { ui?: UiStyle } = {}) {
  const g = useUiStyle(ui) === 'glados';
  const legacy = useUiStyle(ui) === 'legacy';
  const pathname = usePathname();
  const segment = pathname.split('/')[1] ?? '';
  const here = SEGMENT_LABELS[segment] ?? segment;
  // the Wizard dock reports open/closed so the wand can toggle it and say so
  const [dockOpen, setDockOpen] = useState(false);
  useEffect(() => {
    const onState = (e: Event) => setDockOpen(!!(e as CustomEvent<ConductorStateDetail>).detail?.open);
    window.addEventListener(CONDUCTOR_STATE_EVENT, onState);
    return () => window.removeEventListener(CONDUCTOR_STATE_EVENT, onState);
  }, []);
  // GladOS: the bar has no edge at rest; a --g-line shadow appears once the page scrolls (S00 §5.5)
  const [scrolled, setScrolled] = useState(false);
  useEffect(() => {
    if (!g) return;
    const onScroll = () => setScrolled(window.scrollY > 0);
    onScroll();
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  }, [g]);
  const toggleDock = () => window.dispatchEvent(new CustomEvent(CONDUCTOR_TOGGLE_EVENT));

  if (g) {
    // GTopbar (S00 §5.5): sans breadcrumb, kit icon buttons, no AdPilot accent override (C30)
    return (
      <div className="os-topbar g-topbar" data-g="GTopbar" data-scrolled={scrolled || undefined}>
        <div className="g-crumb">
          <span>{BRAND.name.toLowerCase()}</span>
          <span aria-hidden className="g-crumb-sep">
            /
          </span>
          <span className="g-crumb-here">{here}</span>
        </div>
        <div className="g-topbar-actions">
          <GButton variant="icon" onClick={openPalette} title="Command palette (⌘K)" aria-label="Open the command palette">
            <Search size={16} strokeWidth={1.6} aria-hidden />
          </GButton>
          <GButton
            variant="icon"
            onClick={toggleDock}
            title={dockOpen ? 'Close the Wizard' : 'Ask the Wizard about this screen'}
            aria-label={dockOpen ? 'Close the Wizard' : 'Open the Wizard'}
            aria-expanded={dockOpen}
          >
            <WizardIcon size={16} strokeWidth={1.6} aria-hidden />
          </GButton>
        </div>
      </div>
    );
  }

  return (
    <div
      className="os-topbar sticky top-0 z-30 flex h-[52px] shrink-0 items-center gap-5 border-b border-os-border px-6"
      // AdPilot's chrome follows the page accent through theme tokens
      // (.os-topbar[data-route='adpilot'] in app/globals.css, SPEC G1).
      data-route={pathname.startsWith('/adpilot') ? 'adpilot' : undefined}
    >
      <div className="flex items-center gap-[7px] whitespace-nowrap font-mono text-[12.5px] tracking-[0.04em] text-os-dim">
        <span>{BRAND.name.toLowerCase()}</span>
        <span className="opacity-45">/</span>
        <span className="text-os-text">{here}</span>
      </div>
      <div className="ml-auto flex items-center gap-2.5">
        <button
          onClick={openPalette}
          title="Command palette (⌘K)"
          className="grid h-[30px] w-[30px] place-items-center rounded-sm-t border border-os-border-strong bg-os-surface2 text-os-muted transition-colors hover:bg-os-surface3 hover:text-os-text"
        >
          <Search className="h-3.5 w-3.5" strokeWidth={1.7} />
        </button>
        {/* the agent dock: the Wizard answers about whatever screen you're on.
            The wand toggles the dock (S08 FR-7). */}
        <button
          type="button"
          onClick={() => window.dispatchEvent(new CustomEvent(CONDUCTOR_TOGGLE_EVENT))}
          title={dockOpen ? 'Close the Wizard' : 'Ask the Wizard about this screen'}
          aria-label={dockOpen ? 'Close the Wizard' : 'Open the Wizard'}
          aria-expanded={dockOpen}
          className="grid h-[30px] w-[30px] place-items-center rounded-sm-t border border-os-border-strong bg-os-surface2 text-os-muted transition-colors hover:bg-os-surface3 hover:text-os-text aria-expanded:text-os-text"
        >
          <WizardIcon className="h-3.5 w-3.5" strokeWidth={1.7} aria-hidden />
        </button>
        {/* Bennett Legacy: the brand mark in the corner, as FounderOS's topbar (L0.4 shell rules) */}
        {legacy && <LegacyTopbarMark />}
      </div>
    </div>
  );
}
