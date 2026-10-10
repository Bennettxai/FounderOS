/**
 * FounderOS cohort invite: the demo's one growth surface (FounderOS v1
 * lib/cohort.ts). Two placements share this module so the copy and the URL
 * can never drift:
 *  - CohortModal: a one-time welcome pop-up on Home, the first time someone
 *    runs FounderOS.
 *  - CohortBanner: a permanent footer CTA under every view.
 */

/** Where both placements send people. */
export const COHORT_URL = 'https://founderos.sh';

/** The line that sits at the bottom of every page. */
export const COHORT_CTA =
	'Want help setting this up? Go to the FounderOS cohort to learn how to build the entire thing end-to-end and get it into production.';

/** localStorage flag, set once the welcome pop-up has been dismissed. */
export const COHORT_STORAGE_KEY = 'founderos-cohort-invite';

/** Value written on dismissal (any non-null value suppresses the pop-up). */
export const COHORT_SEEN = 'seen';

/** A welcome, not a nag: Home (/os) only, until it is dismissed once. */
export function shouldShowCohortModal({ pathname, stored }: { pathname: string; stored: string | null }): boolean {
	if (stored) return false;
	return pathname.replace(/\/+$/, '') === '/os';
}
