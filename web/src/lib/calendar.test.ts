import { describe, expect, it } from 'vitest';
import { byDay, monthGrid, monthRange, shiftMonth } from './calendar';

describe('monthRange', () => {
	it('covers the whole month, leap years included', () => {
		expect(monthRange('2026-10')).toEqual({ from: '2026-10-01', to: '2026-10-31' });
		expect(monthRange('2028-02')).toEqual({ from: '2028-02-01', to: '2028-02-29' });
		expect(monthRange('2026-02-15')).toEqual({ from: '2026-02-01', to: '2026-02-28' });
	});
});

describe('shiftMonth', () => {
	it('crosses year boundaries', () => {
		expect(shiftMonth('2026-12', 1)).toBe('2027-01');
		expect(shiftMonth('2026-01', -1)).toBe('2025-12');
	});
});

describe('byDay', () => {
	it('groups consecutive items by date', () => {
		const groups = byDay([
			{ date: '2026-10-01', id: 'a' },
			{ date: '2026-10-01', id: 'b' },
			{ date: '2026-10-03', id: 'c' }
		]);
		expect(groups.map(([d, items]) => [d, items.length])).toEqual([
			['2026-10-01', 2],
			['2026-10-03', 1]
		]);
	});
});

describe('monthGrid', () => {
	it('fills whole Monday-first weeks around the month', () => {
		const weeks = monthGrid('2026-10'); // 1 October 2026 is a Thursday
		expect(weeks).toHaveLength(5);
		expect(weeks[0][0]).toBe('2026-09-28');
		expect(weeks[0][3]).toBe('2026-10-01');
		expect(weeks[4][6]).toBe('2026-11-01');
		expect(weeks.every((w) => w.length === 7)).toBe(true);
	});
	it('does not pad a month that starts on Monday and ends on Sunday', () => {
		const weeks = monthGrid('2027-02');
		expect(weeks[0][0]).toBe('2027-02-01');
		expect(weeks.at(-1)?.at(-1)).toBe('2027-02-28');
	});
});
