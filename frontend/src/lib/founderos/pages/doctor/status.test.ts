import { describe, expect, it } from 'vitest';
import { doctorStatus } from './status';

describe('doctorStatus', () => {
	it('names failures as failing, in red, ahead of warnings', () => {
		expect(doctorStatus({ ok: 14, warn: 0, fail: 1, total: 15 }, true)).toEqual({ tone: 'err', text: '1 failing', flagged: 1 });
		expect(doctorStatus({ ok: 10, warn: 2, fail: 1, total: 13 }, true)).toEqual({ tone: 'err', text: '1 failing · 2 warnings', flagged: 3 });
	});
	it('warnings alone are amber; nothing flagged is all green', () => {
		expect(doctorStatus({ ok: 13, warn: 1, fail: 0, total: 14 }, true)).toEqual({ tone: 'warn', text: '1 warning', flagged: 1 });
		expect(doctorStatus({ ok: 15, warn: 0, fail: 0, total: 15 }, true)).toEqual({ tone: 'ok', text: 'all green', flagged: 0 });
	});
	it('an unreachable engine says so', () => {
		expect(doctorStatus({ ok: 0, warn: 0, fail: 0, total: 0 }, false)).toEqual({ tone: 'err', text: 'unreachable', flagged: 0 });
	});
});
