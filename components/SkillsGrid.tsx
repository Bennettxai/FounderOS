'use client';

/**
 * The skill catalog. Compact, icon-led cards packed many-per-row; click a card
 * to expand its real SKILL.md in a reader. The full doc loads on demand
 * (GET /api/skills/[slug]) so the page ships light, unless a card carries inline
 * markdown (the seeded fallback). Rendered with a tiny built-in markdown pass.
 */
import { useEffect, useState, type ReactNode } from 'react';
import {
  X,
  Flame,
  Code2,
  Image as ImageIcon,
  FileSignature,
  Hammer,
  ClipboardList,
  SearchCheck,
  Plug,
  Target,
  Clapperboard,
  Cog,
  Sparkles,
  type LucideIcon,
} from 'lucide-react';
import { SectionHead } from '@/components/terminal';
import { Layer } from '@/components/g/Overlays';

export type SkillCard = {
  id: string; // slug — also the /api/skills/[slug] key
  name: string;
  group: string;
  description: string;
  meta: string; // source path or owner (shown in the reader header)
  filePath: string;
  status?: 'live' | 'learning' | 'planned';
  markdown?: string; // inline (fallback); otherwise fetched by id
};

const STATUS: Record<string, string> = { live: 'var(--ok)', learning: 'var(--warn)', planned: 'var(--text-3)' };
const GROUP_ORDER = ['Spec · build · review', 'Engineering', 'Sales', 'Content', 'Ops', 'Creative', 'Skills', 'Firecrawl'];

/** The tool/nature each skill runs on, as an icon. */
function skillIcon(card: SkillCard): LucideIcon {
  if (card.group === 'Firecrawl') return Flame;
  switch (card.id) {
    case 'codex':
      return Code2;
    case 'nano-banana':
      return ImageIcon;
    case 'proposal-generator':
      return FileSignature;
    case 'mcp-builder':
      return Plug;
    case 'build':
      return Hammer;
    case 'spec':
      return ClipboardList;
    case 'review':
      return SearchCheck;
  }
  switch (card.group) {
    case 'Sales':
      return Target;
    case 'Content':
      return Clapperboard;
    case 'Ops':
      return Cog;
    case 'Engineering':
      return Code2;
    default:
      return Sparkles;
  }
}

/** The mapped icon, or null where skillIcon would fall back to the generic sparkle (GladOS F-S4: an identical icon on every card says nothing). */
export function specificSkillIcon(card: SkillCard): LucideIcon | null {
  const icon = skillIcon(card);
  return icon === Sparkles ? null : icon;
}

function inline(text: string): ReactNode[] {
  return text.split(/(\*\*[^*]+\*\*|`[^`]+`)/g).map((p, i) => {
    if (p.startsWith('**') && p.endsWith('**')) return <strong key={i} className="font-semibold text-os-text">{p.slice(2, -2)}</strong>;
    if (p.startsWith('`') && p.endsWith('`')) return <code key={i} className="rounded bg-os-surface2 px-1 font-mono text-[12.5px] text-os-accent">{p.slice(1, -1)}</code>;
    return <span key={i}>{p}</span>;
  });
}

function Markdown({ src }: { src: string }) {
  const out: ReactNode[] = [];
  let fence: string[] | null = null;
  let k = 0;
  for (const line of src.split('\n')) {
    if (line.trim().startsWith('```')) {
      if (fence) {
        out.push(<pre key={k++} className="my-2 overflow-x-auto rounded-md border border-os-border bg-os-bg p-3 font-mono text-[12.5px] leading-relaxed text-os-muted">{fence.join('\n')}</pre>);
        fence = null;
      } else fence = [];
      continue;
    }
    if (fence) { fence.push(line); continue; }
    if (/^#\s/.test(line)) out.push(<h1 key={k++} className="mb-1 mt-5 text-[16px] font-bold">{inline(line.slice(2))}</h1>);
    else if (/^##\s/.test(line)) out.push(<h2 key={k++} className="mb-1 mt-4 text-[13.5px] font-semibold text-os-text">{inline(line.slice(3))}</h2>);
    else if (/^###\s/.test(line)) out.push(<h3 key={k++} className="mt-3 text-[13.5px] font-semibold">{inline(line.slice(4))}</h3>);
    else if (/^---\s*$/.test(line)) out.push(<hr key={k++} className="my-2.5 border-os-border" />);
    else if (/^\s*[-*]\s/.test(line))
      out.push(<div key={k++} className="flex gap-3 text-[13.5px] leading-relaxed text-os-muted"><span className="text-os-accent">·</span><span>{inline(line.replace(/^\s*[-*]\s/, ''))}</span></div>);
    else if (line.trim() === '') out.push(<div key={k++} className="h-2" />);
    else out.push(<p key={k++} className="text-[13.5px] leading-relaxed text-os-muted">{inline(line)}</p>);
  }
  if (fence) out.push(<pre key={k++} className="my-2 overflow-x-auto rounded-md border border-os-border bg-os-bg p-3 font-mono text-[12.5px] leading-relaxed text-os-muted">{fence.join('\n')}</pre>);
  return <div>{out}</div>;
}

export function SkillsGrid({
  cards,
  sourceNote,
  query = '',
  openSlug = null,
  onOpened,
}: {
  cards: SkillCard[];
  sourceNote: string;
  /** live filter from the explorer search bar */
  query?: string;
  /** hierarchy panel's "Read SKILL.md" — opens that card's viewer */
  openSlug?: string | null;
  onOpened?: () => void;
}) {
  const [viewing, setViewing] = useState<SkillCard | null>(null);
  const [md, setMd] = useState<string | null>(null);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && setViewing(null);
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, []);

  const open = async (card: SkillCard) => {
    setViewing(card);
    if (card.markdown != null) {
      setMd(card.markdown);
      return;
    }
    setMd(null); // loading
    try {
      const res = await fetch(`/api/skills/${encodeURIComponent(card.id)}`);
      const body = (await res.json()) as { markdown?: string; error?: string };
      setMd(body.markdown ?? `SKILL.md could not be read (${body.error ?? res.status}).`);
    } catch {
      setMd('SKILL.md could not be read.');
    }
  };

  // open requests from the hierarchy panel
  useEffect(() => {
    if (!openSlug) return;
    const card = cards.find((c) => c.id === openSlug);
    if (card) void open(card);
    onOpened?.();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [openSlug]);

  const q = query.trim().toLowerCase();
  const visible =
    q === ''
      ? cards
      : cards.filter(
          (c) =>
            c.name.toLowerCase().includes(q) ||
            c.description.toLowerCase().includes(q) ||
            c.id.toLowerCase().includes(q) ||
            c.group.toLowerCase().includes(q) ||
            c.meta.toLowerCase().includes(q),
        );

  const groups = [...new Set(visible.map((c) => c.group))].sort((a, b) => {
    const ia = GROUP_ORDER.indexOf(a);
    const ib = GROUP_ORDER.indexOf(b);
    return (ia === -1 ? 99 : ia) - (ib === -1 ? 99 : ib) || a.localeCompare(b);
  });

  const ViewingIcon = viewing ? skillIcon(viewing) : Sparkles;

  return (
    <div>
      <p className="mb-4 text-[12.5px] text-os-muted">
        {sourceNote}
        {q !== '' && ` · ${visible.length} match${visible.length === 1 ? '' : 'es'} for “${query.trim()}”`}
      </p>
      {q !== '' && visible.length === 0 && (
        <p className="rounded-md-t border border-dashed border-os-border-strong px-3 py-4 text-center text-[12.5px] text-os-dim">
          no skill matches “{query.trim()}”
        </p>
      )}
      {groups.map((group) => {
        const inGroup = visible.filter((c) => c.group === group);
        return (
          <section key={group} className="mb-8">
            <SectionHead label={group} count={inGroup.length} />
            <div className="grid grid-cols-2 gap-3.5 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6">
              {inGroup.map((c) => {
                const Icon = skillIcon(c);
                return (
                  <button
                    key={c.id}
                    onClick={() => open(c)}
                    title={`${c.name} — open SKILL.md`}
                    className="group flex flex-col gap-2.5 rounded-lg-t border border-os-border bg-os-surface p-2.5 text-left transition-colors hover:border-os-border-strong hover:bg-os-surface2"
                  >
                    <div className="flex items-center gap-2.5">
                      <Icon className="h-[15px] w-[15px] shrink-0 text-os-accent" strokeWidth={1.7} />
                      <span className="min-w-0 flex-1 truncate text-[13px] font-medium group-hover:text-os-text">{c.name}</span>
                      {c.status && <span className="h-1.5 w-1.5 shrink-0 rounded-full" style={{ background: STATUS[c.status] }} />}
                    </div>
                    <p className="line-clamp-2 text-[11.5px] leading-snug text-os-dim">{c.description}</p>
                  </button>
                );
              })}
            </div>
          </section>
        );
      })}

      {/* Portalled to <body>: a frosted ancestor would trap the fixed layer. */}
      {viewing && (
        <Layer>
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-6" onClick={() => setViewing(null)}>
            <div className="c-pop flex max-h-[82vh] w-full max-w-2xl flex-col overflow-hidden" onClick={(e) => e.stopPropagation()}>
              <div className="flex shrink-0 items-center justify-between border-b border-os-border px-5 py-3">
                <span className="flex min-w-0 items-center gap-3 font-mono text-[12.5px] text-os-dim">
                  <ViewingIcon className="h-3.5 w-3.5 shrink-0 text-os-accent" strokeWidth={1.7} />
                  <span className="truncate">{viewing.filePath}</span>
                </span>
                <button onClick={() => setViewing(null)} aria-label="Close" className="shrink-0 text-os-dim transition-colors hover:text-os-text">
                  <X className="h-4 w-4" strokeWidth={1.7} />
                </button>
              </div>
              <div className="overflow-y-auto px-5 py-4">
                {md === null ? (
                  <p className="animate-pulse text-[12.5px] text-os-muted">Loading SKILL.md…</p>
                ) : (
                  <Markdown src={md} />
                )}
              </div>
            </div>
          </div>
        </Layer>
      )}
    </div>
  );
}
