'use client';

import { Component, Fragment, type ErrorInfo, type ReactNode } from 'react';
import { AlertTriangle } from 'lucide-react';
import { OBtn } from '@/components/creative-engine/skin';
import { Layer } from '@/components/g/Overlays';

/**
 * Catches a render error anywhere in the Motion console (S05a FR-2). The
 * console never turns into a white screen: it shows the message with Reset
 * controls (remounts the console on its defaults) and Close.
 */
export class ConsoleErrorBoundary extends Component<{ onClose: () => void; onReset?: () => void; children: ReactNode }, { error: Error | null; resets: number }> {
  state = { error: null as Error | null, resets: 0 };

  static getDerivedStateFromError(error: Error) {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('Motion console render error', error, info.componentStack);
  }

  reset = () => {
    // a stored draft may be what crashed: drop it before remounting on the defaults
    this.props.onReset?.();
    this.setState((s) => ({ error: null, resets: s.resets + 1 }));
  };

  render() {
    const { error, resets } = this.state;
    if (!error) return <Fragment key={resets}>{this.props.children}</Fragment>;
    // Portalled to <body> like the console it stands in for.
    return (
      <Layer>
        <div className="ae-octane ae-o-ground fixed inset-0 z-[130] flex items-center justify-center p-6" role="alertdialog" aria-modal="true" aria-labelledby="ae-console-error-title">
          <div className="ae-o-panel w-full max-w-[460px] p-6">
            <div id="ae-console-error-title" className="flex items-center gap-2 text-[16px] font-medium">
              <AlertTriangle className="h-4 w-4" strokeWidth={2} style={{ color: 'var(--o-red-text)' }} />
              The console hit an error
            </div>
            <p className="mt-3 break-words text-[13px] text-[var(--o-muted)]">The console hit an error: {error.message || 'unknown error'}</p>
            <div className="mt-5 flex justify-end gap-3">
              <OBtn kind="pill" cls="ae-o-pill"  onClick={this.reset}>Reset controls</OBtn>
              <OBtn kind="primary" cls="ae-o-generate"  onClick={this.props.onClose}><i /> Close</OBtn>
            </div>
          </div>
        </div>
      </Layer>
    );
  }
}
