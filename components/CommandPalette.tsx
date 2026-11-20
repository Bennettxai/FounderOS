'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { filterCommands, type Command } from '@/lib/palette';
// Digit shortcuts (1–9) jump to views — derived from the sidebar's visible order
// so they stay in lockstep with the nav.
import { digitViewsFor } from '@/lib/nav';
import { GEmpty, GList, GRow, GSearch, GTag } from '@/components/g';
import { cx } from '@/components/g/cx';
import type { UiStyle } from '@/lib/appearance';
import { useUiStyle } from '@/lib/use-ui-style';
import { useMeltEnd, useMeltPhase } from '@/lib/use-melt';

function isTyping(): boolean {
  const el = document.activeElement;
  if (!el) return false;
  const tag = el.tagName;
  return tag === 'INPUT' || tag === 'TEXTAREA' || (el as HTMLElement).isContentEditable;
}

/**
 * The palette (Cmd/Ctrl+K). Glass (brief §3.2, §4): the scrim is the face's sibling, never its
 * parent, so the face keeps its own strong frost; a press on the scrim closes; on close the face
 * melts out (lib/use-melt) while the shortcuts already treat the palette as closed.
 */
export function CommandPalette({ commands, ui = 'classic', off = [] }: { commands: Command[]; ui?: UiStyle; /** modules switched off in Settings */ off?: string[] }) {
  const g = useUiStyle(ui) === 'glados';
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [index, setIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const faceRef = useRef<HTMLDivElement>(null);
  const melt = useMeltPhase(open);
  const melting = melt.melting;
  const onMeltEnd = useMeltEnd(faceRef, melting, melt.end);

  const hits = useMemo(() => filterCommands(commands, query).slice(0, 9), [commands, query]);

  const close = useCallback(() => {
    setOpen(false);
    setQuery('');
    setIndex(0);
  }, []);

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        setOpen((v) => !v);
      } else if (e.key === 'Escape') {
        // GSearch clears a non-empty query first and marks the key handled
        if (e.defaultPrevented) return;
        close();
      } else if (!open && /^[1-9]$/.test(e.key) && !e.metaKey && !e.ctrlKey && !e.altKey && !isTyping()) {
        const href = digitViewsFor(off)[Number(e.key) - 1];
        if (href) router.push(href);
      }
    }
    function onOpen() {
      setOpen(true);
    }
    window.addEventListener('keydown', onKey);
    window.addEventListener('glados:palette', onOpen);
    return () => {
      window.removeEventListener('keydown', onKey);
      window.removeEventListener('glados:palette', onOpen);
    };
  }, [close, open, router, off]);

  useEffect(() => {
    if (open) inputRef.current?.focus();
  }, [open]);

  // Closed but still melting: the face keeps no focus, so no key reaches its input.
  useEffect(() => {
    const active = document.activeElement;
    if (melting && active instanceof HTMLElement && faceRef.current?.contains(active)) active.blur();
  }, [melting]);

  function go(command: Command | undefined) {
    if (!command) return;
    close();
    // shell actions (Toggle sidebar) fire their window event instead of navigating
    if (command.action) window.dispatchEvent(new CustomEvent(command.action));
    else if (!command.href) return;
    else if (command.href.startsWith('http')) window.open(command.href, '_blank');
    else router.push(command.href);
  }

  if (!melt.mounted) return null;

  const onListKey = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      setIndex((i) => Math.min(i + 1, hits.length - 1));
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      setIndex((i) => Math.max(i - 1, 0));
    } else if (e.key === 'Enter') {
      go(hits[index]);
    }
  };

  if (g) {
    // GPalette (S00 §5.5): a radius-20 card on the --g-scrim, GSearch pill, GRow results
    return (
      <div className="g-palette-layer" data-g="GPalette" data-melting={melting ? '' : undefined}>
        <div className="gl-scrim" aria-hidden onClick={close} />
        <div
          ref={faceRef}
          className={cx('g-palette glass-face--strong', melting && 'is-melting')}
          role="dialog"
          aria-modal="true"
          aria-label="Command palette"
          onAnimationEnd={onMeltEnd}
        >
          <GSearch
            ref={inputRef}
            value={query}
            onValueChange={(v) => {
              setQuery(v);
              setIndex(0);
            }}
            onKeyDown={onListKey}
            placeholder="Jump anywhere: views, agents, tools"
            aria-label="Search views, agents and tools"
            autoComplete="off"
          />
          {hits.length === 0 ? (
            <GEmpty inline title="No matches" body="Try a view, an agent or a tool by name." className="g-palette-empty" />
          ) : (
            <div
              className="g-palette-list"
              onPointerMove={(e) => {
                const li = (e.target as HTMLElement).closest('li');
                if (!li?.parentElement) return;
                const i = Array.prototype.indexOf.call(li.parentElement.children, li);
                if (i >= 0 && i !== index) setIndex(i);
              }}
            >
              <GList label="Results">
                {hits.map((command, i) => (
                  <GRow
                    key={command.id}
                    title={command.label}
                    trailing={command.hint ? <GTag size="xs">{command.hint}</GTag> : undefined}
                    selected={i === index}
                    onClick={() => go(command)}
                  />
                ))}
              </GList>
            </div>
          )}
        </div>
      </div>
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-[14vh]" data-melting={melting ? '' : undefined}>
      <div className="gl-scrim" aria-hidden onClick={close} />
      <div
        ref={faceRef}
        className={cx('c-pop glass-face--strong relative w-[560px] max-w-[calc(100vw-48px)] overflow-hidden', melting && 'is-melting')}
        role="dialog"
        aria-modal="true"
        aria-label="Command palette"
        onAnimationEnd={onMeltEnd}
      >
        <input
          ref={inputRef}
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setIndex(0);
          }}
          onKeyDown={(e) => {
            if (e.key === 'ArrowDown') {
              e.preventDefault();
              setIndex((i) => Math.min(i + 1, hits.length - 1));
            } else if (e.key === 'ArrowUp') {
              e.preventDefault();
              setIndex((i) => Math.max(i - 1, 0));
            } else if (e.key === 'Enter') {
              go(hits[index]);
            }
          }}
          placeholder="Jump anywhere — views, agents, tools…"
          className="w-full border-b border-os-border bg-transparent px-[18px] py-4 font-mono text-sm text-os-text outline-none placeholder:text-os-dim"
        />
        <ul className="max-h-80 overflow-y-auto p-1.5">
          {hits.length === 0 && (
            <li className="px-4 py-6 text-center font-mono text-xs text-os-dim">No matches</li>
          )}
          {hits.map((command, i) => (
            <li key={command.id}>
              <button
                onClick={() => go(command)}
                onMouseEnter={() => setIndex(i)}
                className={`flex w-full items-center gap-3.5 rounded-sm-t px-3 py-[9px] text-left text-[14px] ${
                  i === index ? 'bg-[var(--accent-soft)] text-os-text' : 'text-os-muted'
                }`}
              >
                <span className="w-3 shrink-0 opacity-60">{i === index ? '›' : ''}</span>
                <span className="min-w-0 flex-1 truncate">{command.label}</span>
                {command.hint && (
                  <span className="shrink-0 font-mono text-[11px] text-os-dim">
                    {command.hint}
                  </span>
                )}
              </button>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
