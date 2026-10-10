/** The Doctor's one-line state: failures are called failing (red) and come
 *  before warnings (amber), so the header, the core card and the Needs-you
 *  line all read the same counts the same way. */
export type DoctorCounts = { ok: number; warn: number; fail: number; total: number };
export type DoctorStatus = { tone: 'ok' | 'warn' | 'err'; text: string; flagged: number };

export function doctorStatus(c: DoctorCounts, connected: boolean): DoctorStatus {
	if (!connected) return { tone: 'err', text: 'unreachable', flagged: 0 };
	const parts: string[] = [];
	if (c.fail > 0) parts.push(`${c.fail} failing`);
	if (c.warn > 0) parts.push(`${c.warn} warning${c.warn === 1 ? '' : 's'}`);
	const flagged = c.fail + c.warn;
	return { tone: c.fail > 0 ? 'err' : c.warn > 0 ? 'warn' : 'ok', text: parts.join(' · ') || 'all green', flagged };
}
