'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { KeyRound, LogOut, ShieldCheck, ShieldOff, Trash2 } from 'lucide-react';

type Mode = 'idle' | 'set' | 'change' | 'rotate' | 'remove';

const COPY: Record<Exclude<Mode, 'idle'>, { title: string; button: string; needsCurrent: boolean; needsNew: boolean }> = {
  set: { title: 'Set the operator password', button: 'Set password', needsCurrent: false, needsNew: true },
  change: { title: 'Change the operator password', button: 'Change password', needsCurrent: true, needsNew: true },
  rotate: { title: 'Sign out every other device', button: 'Sign out others', needsCurrent: true, needsNew: false },
  remove: { title: 'Remove the password', button: 'Remove password', needsCurrent: true, needsNew: false },
};

/**
 * The password manager on /setup. One card, two states:
 *   no password → the OS is open on this machine only; offer Set password.
 *   password    → offer Change, Sign out others (rotate the session), Remove.
 * Every write goes through /api/auth/password, which re-checks the current
 * password itself. The hash lands in .env.local on the host, never in git.
 */
export function SetupSecurity({ configured }: { configured: boolean }) {
  const { mode, current, setCurrent, next, setNext, confirm, setConfirm, busy, error, notice, open, close, submit, spec } = usePasswordForm();
  return (
    <SecurityClassic
      {...{ configured, mode, current, setCurrent, next, setNext, confirm, setConfirm, busy, error, notice, open, close, submit, spec }}
    />
  );
}

/** Password form state and submit, shared by both trees of /setup (GladOS: components/setup/SetupG.tsx). */
export function usePasswordForm() {
  const router = useRouter();
  const [mode, setMode] = useState<Mode>('idle');
  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [confirm, setConfirm] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');

  const open = (m: Exclude<Mode, 'idle'>) => {
    setMode(m);
    setCurrent('');
    setNext('');
    setConfirm('');
    setError('');
    setNotice('');
  };

  const close = () => {
    setMode('idle');
    setError('');
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (mode === 'idle') return;
    const spec = COPY[mode];
    if (spec.needsNew && next !== confirm) {
      setError('the two new passwords do not match');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const body =
        mode === 'set'
          ? { action: 'set', password: next }
          : mode === 'change'
            ? { action: 'change', current, password: next }
            : { action: mode, current };
      const res = await fetch('/api/auth/password', {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify(body),
      });
      const data = (await res.json().catch(() => null)) as { error?: string } | null;
      if (!res.ok) {
        setError(data?.error ?? 'could not save');
        return;
      }
      setNotice(
        mode === 'set'
          ? 'Password set. Sign-in is now required on every device.'
          : mode === 'change'
            ? 'Password changed. Every other device has been signed out.'
            : mode === 'rotate'
              ? 'Every other device has been signed out.'
              : 'Password removed. The console is open on this machine only.',
      );
      setMode('idle');
      router.refresh();
    } catch {
      setError('could not reach the server');
    } finally {
      setBusy(false);
    }
  };

  const spec = mode === 'idle' ? null : COPY[mode];
  return { mode, current, setCurrent, next, setNext, confirm, setConfirm, busy, error, notice, open, close, submit, spec };
}

type PasswordForm = ReturnType<typeof usePasswordForm>;

function SecurityClassic({
  configured, mode, current, setCurrent, next, setNext, confirm, setConfirm, busy, error, notice, open, close, submit, spec,
}: PasswordForm & { configured: boolean }) {
  const field =
    'w-full rounded-md-t border border-os-border bg-os-bg px-3 py-1.5 font-mono text-[12px] outline-none focus:border-os-border-strong';

  return (
    <div id="security" className="scroll-mt-24 rounded-lg-t border border-os-border bg-os-surface px-5 py-4">
      <div className="flex flex-wrap items-center gap-3">
        {configured ? (
          <ShieldCheck size={16} strokeWidth={1.8} className="shrink-0 text-os-accent" />
        ) : (
          <ShieldOff size={16} strokeWidth={1.8} className="shrink-0 text-os-dim" />
        )}
        <span className="text-[13.5px] font-medium">Operator password</span>
        <span className="glass-chip inline-flex items-center gap-2 rounded-full px-3 py-1 text-[11.5px] text-os-muted">
          <span className={configured ? 'dot ok' : 'dot warn'} /> {configured ? 'Password set' : 'Not configured'}
        </span>
        {mode === 'idle' && (
          <span className="ml-auto flex flex-wrap items-center gap-2">
            {configured ? (
              <>
                <button onClick={() => open('change')} className="c-btn h-auto px-3 py-1 text-[11.5px]">
                  <KeyRound size={12} strokeWidth={1.8} /> Change
                </button>
                <button onClick={() => open('rotate')} className="c-btn h-auto px-3 py-1 text-[11.5px]">
                  <LogOut size={12} strokeWidth={1.8} /> Sign out others
                </button>
                <button onClick={() => open('remove')} className="c-btn h-auto px-3 py-1 text-[11.5px]">
                  <Trash2 size={12} strokeWidth={1.8} /> Remove
                </button>
              </>
            ) : (
              <button onClick={() => open('set')} className="c-btn c-btn-primary h-auto px-3.5 py-1.5 text-[12px]">
                Set password
              </button>
            )}
          </span>
        )}
      </div>

      <p className="mt-1.5 pl-[28px] text-[12.5px] leading-relaxed text-os-muted [overflow-wrap:anywhere]">
        {configured
          ? 'Every page and API route asks for this password. It is stored as a scrypt hash in .env.local on this host and is never in git. Forgot it: delete the GLADOS_PASSWORD_HASH line from .env.local, restart, and set a new one here.'
          : 'No password yet, so the console answers only on this machine: the server refuses to listen on any other address until one exists. Set a password to sign in from other devices on your network.'}
      </p>

      {notice && (
        <p className="mt-2 flex items-center gap-2 pl-[28px] text-[12px] text-os-muted">
          <span className="dot ok" /> {notice}
        </p>
      )}

      {spec && (
        <form onSubmit={submit} className="mt-4 max-w-[420px] space-y-2.5 pl-[28px]">
          <div className="text-[12.5px] font-medium">{spec.title}</div>
          {spec.needsCurrent && (
            <input
              type="password"
              value={current}
              onChange={(e) => setCurrent(e.target.value)}
              placeholder="Current password"
              autoComplete="current-password"
              autoFocus
              className={field}
            />
          )}
          {spec.needsNew && (
            <>
              <input
                type="password"
                value={next}
                onChange={(e) => setNext(e.target.value)}
                placeholder="New password (8+ characters)"
                autoComplete="new-password"
                autoFocus={!spec.needsCurrent}
                className={field}
              />
              <input
                type="password"
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                placeholder="Repeat new password"
                autoComplete="new-password"
                className={field}
              />
            </>
          )}
          {mode === 'remove' && (
            <p className="text-[12px] text-os-dim">
              Removing it signs this device out and puts the OS back to loopback-only until a new password is set.
            </p>
          )}
          {error && (
            <p className="flex items-center gap-2 text-[12px] text-os-muted">
              <span className="dot err" /> {error}
            </p>
          )}
          <div className="flex items-center gap-2 pt-1">
            <button
              type="submit"
              disabled={busy || (spec.needsCurrent && !current) || (spec.needsNew && (!next || !confirm))}
              className="c-btn c-btn-primary h-auto px-3.5 py-1.5 text-[12px] disabled:opacity-40"
            >
              {busy ? 'Saving…' : spec.button}
            </button>
            <button type="button" onClick={close} className="text-[11.5px] text-os-dim transition-colors hover:text-os-text">
              cancel
            </button>
          </div>
        </form>
      )}
    </div>
  );
}
