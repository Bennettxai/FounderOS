// Pure helpers behind the review panel (FounderOS v1 components/TaskReviewPanel.tsx
// + lib/board-approvals.ts ACTION_TEXT). The bridge classifies the Needs you
// queue; the panel also opens Deliverables rows the bridge did not classify,
// so it reads the ask off the file name the same way (the Go Classify rules).

export const ACTION_TEXT: Record<string, string> = {
	staged: 'The agent already wrote this. Approve to send it. Dismiss to bin it.',
	decision: 'Only you can make this call. Approve to go ahead. Dismiss to drop it.',
	gate: 'Work is paused until you answer. Approve to unblock it. Dismiss to keep it closed.',
	request: 'Someone asked you for something. Approve to act on it. Dismiss to decline.',
	draft: 'A draft is waiting on you. Approve to publish it. Dismiss to send it back.',
	done: 'Finished work. Nothing is being asked of you.',
	output: 'Reference output. Nothing is being asked of you.'
};

const DONE = /(UNBLOCKED|RESOLVED|FINAL|COMPLETE)/i;
const ASK: [RegExp, string][] = [
	[/STAGED/i, 'staged'],
	[/\bdecision\b|-decision-/i, 'decision'],
	[/-gate-|\bgate\b/i, 'gate'],
	[/-request-|\brequest\b/i, 'request'],
	[/-draft-|\bdraft\b/i, 'draft']
];

/** What approving and dismissing this file actually do, in a sentence. */
export function actionTextOf(name: string): string {
	if (DONE.test(name)) return ACTION_TEXT.done;
	for (const [re, kind] of ASK) if (re.test(name)) return ACTION_TEXT[kind];
	if (/DELIVER-BEFORE-\d{4}Z/i.test(name)) return ACTION_TEXT.request;
	return ACTION_TEXT.output;
}

/**
 * A `.json` deliverable is an envelope the agent would post; the document is
 * inside it. Falls back to the raw text when it is not the expected shape.
 */
export function jsonBodyOf(name: string, text: string): string {
	if (!/\.json$/i.test(name)) return text;
	try {
		const parsed: unknown = JSON.parse(text);
		if (parsed && typeof parsed === 'object') {
			for (const key of ['body', 'comment', 'text', 'content', 'markdown']) {
				const v = (parsed as Record<string, unknown>)[key];
				if (typeof v === 'string' && v.trim()) return v;
			}
		}
	} catch {
		/* truncated or not JSON at all: show what we have */
	}
	return text;
}
