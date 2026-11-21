'use client';

import { useEffect, useState } from 'react';
import { X } from 'lucide-react';
import type { Department } from '@/lib/schemas';
import type { DesignedAgent } from '@/lib/wizard/designed';
import { Layer } from '@/components/g/Overlays';

/** Centered modal. Files an agent spec as "designed" — nothing claims a runtime. */
export function CreateAgentModal({
  open,
  departments,
  onClose,
  onCreated,
}: {
  open: boolean;
  departments: Department[];
  onClose: () => void;
  onCreated: (agent: DesignedAgent) => void;
}) {
  const [name, setName] = useState('');
  const [departmentId, setDepartmentId] = useState(departments[0]?.id ?? '');
  const [purpose, setPurpose] = useState('');
  const [tools, setTools] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === 'Escape' && onClose();
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  if (!open) return null;

  async function submit() {
    if (!name.trim() || !departmentId || !purpose.trim() || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      const res = await fetch('/api/wizard/designed', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, departmentId, purpose, tools }),
      });
      const body = await res.json();
      if (!res.ok || !body.ok) throw new Error(body.error ?? `filing failed (${res.status})`);
      onCreated(body.agent as DesignedAgent);
      setName('');
      setPurpose('');
      setTools('');
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }

  const field = 'mt-1 w-full rounded-[10px] border border-os-border bg-os-surface2 px-3 py-2 text-[13px] text-os-text placeholder:text-os-dim focus:border-os-border-strong focus:outline-none';

  // Portalled to <body>: under a frosted ancestor the fixed layer would be trapped.
  return (
    <Layer>
      <div className="fixed inset-0 z-40 grid place-items-center p-6">
        <button type="button" aria-label="Close" onClick={onClose} className="absolute inset-0 bg-black/55 backdrop-blur-[2px]" />
        <div role="dialog" aria-modal="true" className="c-pop glass-face--strong gl-materialise relative w-[440px] max-w-full p-6">
          <div className="mb-1 flex items-center justify-between">
            <span className="text-[16px] text-os-text">Create an agent</span>
            <button type="button" aria-label="Close" onClick={onClose} className="wz-icon-btn">
              <X className="h-4 w-4" />
            </button>
          </div>
          <p className="mb-5 text-[12px] text-os-dim">Files a spec under designed agents. It appears ghosted in the crew and on the Blueprint until it is wired.</p>

          <label className="mb-3 block text-[12px] text-os-muted">
            Name
            <input value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Invoice reconciler" className={field} />
          </label>
          <label className="mb-3 block text-[12px] text-os-muted">
            Department
            <select value={departmentId} onChange={(e) => setDepartmentId(e.target.value)} className={field}>
              {departments.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.name}
                </option>
              ))}
            </select>
          </label>
          <label className="mb-3 block text-[12px] text-os-muted">
            Purpose
            <textarea value={purpose} onChange={(e) => setPurpose(e.target.value)} placeholder="What would it actually do?" rows={4} className={`${field} resize-none`} />
          </label>
          <label className="mb-4 block text-[12px] text-os-muted">
            Tools
            <input value={tools} onChange={(e) => setTools(e.target.value)} placeholder="comma separated, e.g. gmail, attio" className={field} />
          </label>

          {error && (
            <p className="mb-3 flex items-center gap-2 text-[12px] text-os-muted">
              <span className="dot err" /> {error}
            </p>
          )}

          <div className="flex gap-2.5">
            <button type="button" onClick={onClose} className="c-btn">
              Cancel
            </button>
            <button type="button" onClick={submit} disabled={submitting || !name.trim() || !purpose.trim()} className="c-btn c-btn-primary flex-1">
              {submitting ? 'Filing' : 'File as designed'}
            </button>
          </div>
        </div>
      </div>
    </Layer>
  );
}
