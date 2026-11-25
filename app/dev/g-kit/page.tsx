import { notFound } from 'next/navigation';
import { KitGallery } from './KitGallery';

/**
 * Dev-only gallery of the G kit (SPEC S00 §10, lane 1d): every primitive in
 * every state, for screenshot review and the audit scripts. Not in the nav,
 * not served by a production build.
 */
export const metadata = { title: 'Kit gallery' };
export const dynamic = 'force-dynamic';

export default function GKitPage() {
  if (process.env.NODE_ENV === 'production') notFound();
  return <KitGallery />;
}
