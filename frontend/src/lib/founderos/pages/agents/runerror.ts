import { FounderosApiError, isGuardRefusal } from '$lib/founderos/api';

/** A short, honest line for a failed write: a bridge-guard refusal (any
 *  status, see isGuardRefusal) is not a board failure and says so. */
export function writeFailure(err: unknown, verb = 'run'): string {
	if (err instanceof FounderosApiError) {
		const body = err.body as { error?: string } | undefined;
		if (isGuardRefusal(err)) return `${verb} refused · writes are off`;
		if (err.status === 404) return `${verb} failed · not on this bridge`;
		return `${verb} failed · ${body?.error ?? err.message}`;
	}
	return `${verb} failed · ${err instanceof Error ? err.message : String(err)}`;
}
