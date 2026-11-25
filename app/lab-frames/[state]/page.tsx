/**
 * The frames gate route (02 section 16): /lab-frames/<state>. Standalone: no store, no DB, no /lab layout,
 * nothing from lib/lab/view other than frames.ts. Not linked from any nav.
 */
import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { Suspense } from 'react';
import { FRAME_NOW, FRAME_STATES, frameToday, isFrameState } from '@/lib/lab/view/frames';
import { FrameComponents } from './FrameComponents';
import { FrameTodayView } from './FrameToday';

export const metadata: Metadata = { title: 'Content Lab frames', robots: { index: false, follow: false } };
export const dynamic = 'force-dynamic';
export const dynamicParams = false;

export function generateStaticParams() {
  return FRAME_STATES.map((state) => ({ state }));
}

export default function LabFramePage({ params }: { params: { state: string } }) {
  const state = params.state;
  if (!isFrameState(state)) notFound();
  return (
    <Suspense fallback={null}>
      {state === 'components' ? <FrameComponents /> : <FrameTodayView state={state} data={frameToday(state, new Date(FRAME_NOW))} />}
    </Suspense>
  );
}
