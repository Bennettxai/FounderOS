'use client';

/**
 * Voice/text capture straight into the brain-store. The mic uses the
 * browser's built-in speech recognition (Chrome/Safari — no API key);
 * Save writes a real markdown memory file AND embeds it into Optimal Engine immediately
 * (via `gbrain capture`) so agents can retrieve it — degrades to local-only if the
 * knowledgebase is unreachable.
 */
import { useEffect, useRef, useState } from 'react';
import { Mic, MicOff, BrainCircuit, Upload } from 'lucide-react';
import type { Venture } from '@/lib/ventures';

const FOLDERS = ['inbox', 'ideas', 'people', 'companies', 'meetings', 'projects', 'writing'];

type SpeechRecognitionLike = {
  continuous: boolean;
  interimResults: boolean;
  lang: string;
  start(): void;
  stop(): void;
  onresult: ((event: { resultIndex: number; results: ArrayLike<{ isFinal: boolean; 0: { transcript: string } }> }) => void) | null;
  onend: (() => void) | null;
  onerror: ((event: { error: string }) => void) | null;
};

function getRecognizer(): SpeechRecognitionLike | null {
  if (typeof window === 'undefined') return null;
  const Ctor =
    (window as unknown as { SpeechRecognition?: new () => SpeechRecognitionLike }).SpeechRecognition ??
    (window as unknown as { webkitSpeechRecognition?: new () => SpeechRecognitionLike }).webkitSpeechRecognition;
  return Ctor ? new Ctor() : null;
}

// documents we can honestly ingest: anything that reads as text
const TEXTY = /\.(md|markdown|txt|csv|json|ya?ml|html?|log)$/i;
const MAX_DOC_BYTES = 1_000_000; // embedding a monster helps nobody

/** `g`: the compact capture card under the GladOS style (glados-ui S05 F05.12), same behaviour, kit controls. */
export function BrainDump({ compact = false, g = false }: { compact?: boolean; g?: boolean }) {
  const [text, setText] = useState('');
  const [title, setTitle] = useState('');
  const [folder, setFolder] = useState('inbox');
  const [tags, setTags] = useState<string[]>([]);
  // Ventures live in the data directory (server-side); fetch the tag list once.
  const [ventures, setVentures] = useState<Venture[]>([]);
  useEffect(() => {
    fetch('/api/ventures')
      .then((r) => (r.ok ? r.json() : { ventures: [] }))
      .then((d: { ventures?: Venture[] }) => setVentures(d.ventures ?? []))
      .catch(() => setVentures([]));
  }, []);
  const [listening, setListening] = useState(false);
  const [supported, setSupported] = useState(true);
  const [dragOver, setDragOver] = useState(false);
  const [status, setStatus] = useState<{
    kind: 'idle' | 'saving' | 'saved' | 'error';
    detail?: string;
    embedded?: boolean;
    slug?: string;
  }>({ kind: 'idle' });
  const recRef = useRef<SpeechRecognitionLike | null>(null);
  const baseTextRef = useRef('');
  const fileRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    setSupported(getRecognizer() !== null);
  }, []);

  const stopListening = () => {
    recRef.current?.stop();
    recRef.current = null;
    setListening(false);
  };

  const startListening = () => {
    const rec = getRecognizer();
    if (!rec) return;
    baseTextRef.current = text ? `${text.trim()} ` : '';
    rec.continuous = true;
    rec.interimResults = true;
    rec.lang = 'en-US';
    rec.onresult = (event) => {
      let transcript = '';
      for (let i = 0; i < event.results.length; i++) transcript += event.results[i][0].transcript;
      setText(baseTextRef.current + transcript.trim());
    };
    rec.onend = () => setListening(false);
    rec.onerror = (e) => {
      setListening(false);
      setStatus({ kind: 'error', detail: `mic: ${e.error}` });
    };
    recRef.current = rec;
    setStatus({ kind: 'idle' });
    setListening(true);
    rec.start();
  };

  const save = async () => {
    if (!text.trim()) return;
    stopListening();
    setStatus({ kind: 'saving' });
    try {
      const res = await fetch('/api/brain/dump', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ text, title: title || undefined, folder, tags }),
      });
      const body = await res.json();
      if (!res.ok) throw new Error(body.error?.toString() ?? 'save failed');
      setStatus({ kind: 'saved', detail: body.relPath, embedded: body.embedded, slug: body.slug });
      setText('');
      setTitle('');
    } catch (err) {
      setStatus({ kind: 'error', detail: err instanceof Error ? err.message : 'save failed' });
    }
  };

  // dropped documents flow through the same dump pipeline, one note per file,
  // filename (sans extension) as the title so retrieval reads human
  const ingestFiles = async (files: File[]) => {
    const texty = files.filter((f) => TEXTY.test(f.name) || f.type.startsWith('text/'));
    const skipped = files.length - texty.length;
    if (texty.length === 0) {
      setStatus({ kind: 'error', detail: 'only text documents for now (.md .txt .csv .json …)' });
      return;
    }
    setStatus({ kind: 'saving' });
    let saved = 0;
    let embedded = 0;
    try {
      for (const f of texty) {
        if (f.size > MAX_DOC_BYTES) throw new Error(`${f.name} is over 1MB`);
        const content = await f.text();
        if (!content.trim()) continue;
        const res = await fetch('/api/brain/dump', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ text: content, title: f.name.replace(/\.[^.]+$/, ''), folder, tags }),
        });
        const body = await res.json();
        if (!res.ok) throw new Error(body.error?.toString() ?? `saving ${f.name} failed`);
        saved += 1;
        if (body.embedded) embedded += 1;
      }
      setStatus({
        kind: 'saved',
        detail: `${saved} doc${saved === 1 ? '' : 's'}${skipped ? ` · ${skipped} skipped (not text)` : ''}`,
        embedded: embedded === saved,
      });
    } catch (err) {
      setStatus({ kind: 'error', detail: err instanceof Error ? err.message : 'ingest failed' });
    }
  };

  const dropProps = {
    onDragOver: (e: React.DragEvent) => {
      e.preventDefault();
      setDragOver(true);
    },
    onDragLeave: () => setDragOver(false),
    onDrop: (e: React.DragEvent) => {
      e.preventDefault();
      setDragOver(false);
      void ingestFiles([...e.dataTransfer.files]);
    },
  };

  if (compact && g) {
    const statusText =
      status.kind === 'saving'
        ? 'Saving…'
        : status.kind === 'saved'
          ? `${status.embedded ? 'Embedded' : 'Saved'} · ${status.slug ?? status.detail}`
          : status.kind === 'error'
            ? (status.detail ?? 'Save failed')
            : 'Type, talk, drop or upload. Cmd Enter saves.';
    return (
      <div {...dropProps} className="kg-dump" data-drag={dragOver || undefined}>
        <textarea
          value={text}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) void save();
          }}
          rows={2}
          aria-label="Dump into the brain"
          placeholder={dragOver ? 'Drop it: one note per document' : 'Dump into the brain, or drop documents'}
          className="g-input kg-dump-text"
          data-g="GTextarea"
        />
        <input
          ref={fileRef}
          type="file"
          multiple
          tabIndex={-1}
          aria-hidden
          data-g-exempt="hidden native file picker, opened by the Upload button"
          accept=".md,.markdown,.txt,.csv,.json,.yaml,.yml,.html,.htm,.log,text/*"
          onChange={(e) => {
            void ingestFiles([...(e.target.files ?? [])]);
            e.currentTarget.value = '';
          }}
          className="hidden"
        />
        <div className="kg-dump-foot">
          <span className="kg-cap kg-trunc" data-state={status.kind} role="status">
            {statusText}
          </span>
          {supported && (
            <button type="button" className="g-btn" data-variant="icon" data-size="sm" aria-pressed={listening} onClick={listening ? stopListening : startListening} aria-label={listening ? 'Stop dictation' : 'Start dictation'} title={listening ? 'Stop dictation' : 'Start dictation'}>
              {listening ? <MicOff size={13} strokeWidth={1.8} aria-hidden="true" /> : <Mic size={13} strokeWidth={1.8} aria-hidden="true" />}
            </button>
          )}
          <button type="button" className="g-btn" data-size="sm" onClick={() => fileRef.current?.click()} title="Choose documents to upload">
            <Upload size={12} strokeWidth={1.8} aria-hidden="true" /> Upload
          </button>
          <button type="button" className="g-btn" data-variant="ink" data-size="sm" onClick={save} disabled={!text.trim() || status.kind === 'saving'} title={!text.trim() ? 'Type something first' : undefined}>
            Save
          </button>
        </div>
      </div>
    );
  }

  if (compact) {
    // ONE untitled part : a wide, short capture bar tucked across the
    // top-right whitespace beside the title — type, talk, drop, or upload
    // documents into the brain. Horizontal, not tall: the graph owns the space
    // directly under the title.
    return (
      <div
        {...dropProps}
        className={`w-full rounded-lg-t border bg-os-surface p-2 transition-colors ${
          dragOver ? 'border-os-accent' : 'border-os-border'
        }`}
      >
        <div className="relative">
          <textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) void save();
            }}
            rows={2}
            placeholder={dragOver ? 'drop it — one note per document' : 'dump into the brain… or drop documents'}
            className="w-full resize-none border-0 bg-transparent px-1.5 py-1 pr-9 text-[13px] leading-relaxed text-os-text placeholder:text-os-dim focus:outline-none"
          />
          {supported && (
            <button
              onClick={listening ? stopListening : startListening}
              title={listening ? 'Stop dictation' : 'Start dictation'}
              className={`absolute right-0 top-0 flex h-6 w-6 items-center justify-center rounded-sm-t border transition-colors ${
                listening
                  ? 'animate-pulse border-os-err bg-os-err text-white'
                  : 'border-os-border bg-os-surface text-os-muted hover:text-os-text'
              }`}
            >
              {listening ? <MicOff className="h-3 w-3" /> : <Mic className="h-3 w-3" />}
            </button>
          )}
        </div>

        {/* native file picker — a real Upload button alongside drag-and-drop */}
        <input
          ref={fileRef}
          type="file"
          multiple
          accept=".md,.markdown,.txt,.csv,.json,.yaml,.yml,.html,.htm,.log,text/*"
          onChange={(e) => {
            void ingestFiles([...(e.target.files ?? [])]);
            e.currentTarget.value = '';
          }}
          className="hidden"
        />

        <div className="mt-1.5 flex items-center gap-2.5 border-t border-os-border px-1.5 pt-1.5">
          <span className="min-w-0 flex-1 truncate text-[11.5px] text-os-dim">
            {status.kind === 'saving'
              ? 'Saving…'
              : status.kind === 'saved'
                ? `✓ ${status.embedded ? 'Embedded' : 'Saved'} · ${status.slug ?? status.detail}`
                : status.kind === 'error'
                  ? `✗ ${status.detail}`
                  : 'Text · voice · drag or upload'}
          </span>
          <button
            onClick={() => fileRef.current?.click()}
            title="Choose documents to upload"
            className="flex shrink-0 items-center gap-1 rounded-sm-t border border-os-border-strong bg-os-surface2 px-2 py-0.5 text-[11.5px] text-os-muted transition-colors hover:bg-os-surface3 hover:text-os-text"
          >
            <Upload className="h-3 w-3" strokeWidth={1.7} />
            Upload
          </button>
          <button
            onClick={save}
            disabled={!text.trim() || status.kind === 'saving'}
            className="shrink-0 rounded-sm-t bg-os-accent px-2.5 py-0.5 text-[11.5px] font-medium text-os-ink transition-colors hover:bg-os-accent2 disabled:opacity-30"
          >
            Save
          </button>
        </div>
      </div>
    );
  }

  return (
    <div
      {...dropProps}
      className={`rounded-lg-t border bg-os-surface p-6 transition-colors ${dragOver ? 'border-os-accent' : 'border-os-border'}`}
    >
      <div className="flex items-center gap-3">
        <BrainCircuit className="h-4 w-4 text-os-muted" strokeWidth={1.6} />
        <h3 className="text-sm font-medium">Brain dump</h3>
        <span className="text-[12.5px] text-os-dim">
          speak, type, or drop documents → saved to the brain-store & embedded into Optimal Engine
        </span>
      </div>

      <div className="mt-4 flex flex-col gap-3">
        <div className="flex gap-3">
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="Title (optional — derived from first words)"
            className="flex-1 rounded-sm-t border border-os-border bg-os-surface2 px-3 py-3 text-xs text-os-text placeholder:text-os-dim focus:border-os-border-strong focus:outline-none"
          />
          <select
            value={folder}
            onChange={(e) => setFolder(e.target.value)}
            className="rounded-sm-t border border-os-border bg-os-surface2 px-2 py-3 text-xs text-os-muted focus:outline-none"
          >
            {FOLDERS.map((f) => (
              <option key={f} value={f}>{f}/</option>
            ))}
          </select>
        </div>

        <div className="relative">
          <textarea
            value={text}
            onChange={(e) => setText(e.target.value)}
            rows={4}
            placeholder={supported ? 'Dump it here, or hit the mic and just talk…' : 'Type your dump (voice needs Chrome/Safari)…'}
            className="w-full resize-y rounded-sm-t border border-os-border bg-os-surface2 px-3 py-3 pr-12 text-sm leading-relaxed text-os-text placeholder:text-os-dim focus:border-os-border-strong focus:outline-none"
          />
          {supported && (
            <button
              onClick={listening ? stopListening : startListening}
              title={listening ? 'Stop dictation' : 'Start dictation'}
              className={`absolute right-2 top-2 flex h-8 w-8 items-center justify-center rounded-sm-t border transition-colors ${
                listening
                  ? 'animate-pulse border-os-err bg-os-err text-white'
                  : 'border-os-border bg-os-surface text-os-muted hover:text-os-text'
              }`}
            >
              {listening ? <MicOff className="h-4 w-4" /> : <Mic className="h-4 w-4" />}
            </button>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <span className="text-[11.5px] text-os-dim">Tag venture</span>
          {ventures.map((v) => {
            const on = tags.includes(v.brainTag);
            return (
              <button
                key={v.id}
                onClick={() => setTags((prev) => (on ? prev.filter((t) => t !== v.brainTag) : [...prev, v.brainTag]))}
                className={`glass-chip flex items-center gap-1 rounded-full px-2 py-0.5 text-[11.5px] transition-colors ${
                  on ? 'glass-chip gl-on text-os-text' : 'text-os-muted hover:text-os-text'
                }`}
                style={on ? ({ '--gl-on': v.color } as React.CSSProperties) : undefined}
              >
                #{v.brainTag}
              </button>
            );
          })}

          <button
            onClick={save}
            disabled={!text.trim() || status.kind === 'saving'}
            className="ml-auto rounded-sm-t bg-os-accent px-4 py-2.5 text-xs font-medium text-os-ink transition-colors hover:bg-os-accent2 disabled:opacity-30"
          >
            {status.kind === 'saving' ? 'Saving…' : 'Save to brain'}
          </button>
        </div>

        {status.kind === 'saved' && (
          <p className="flex items-center gap-2.5 text-[12.5px] text-os-muted">
            <span className="dot ok" />
            {status.embedded
              ? `Saved & embedded → ${status.slug ?? status.detail} — retrievable by agents now`
              : `Saved → brain-store/${status.detail} — embed pending (Optimal Engine unreachable, will sync later)`}
          </p>
        )}
        {status.kind === 'error' && (
          <p className="flex items-center gap-2.5 text-[12.5px] text-os-muted">
            <span className="dot err" /> {status.detail}
          </p>
        )}
      </div>
    </div>
  );
}
