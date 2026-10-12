import { describe, expect, it } from 'vitest';
import {
	byDay,
	monthGrid,
	monthRange,
	shiftMonth,
	spreadEvents,
	type Occurrence
} from './calendar';

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

describe('spreadEvents', () => {
	const work = { id: 's1', name: 'Work', color: '#ff0000' };
	const at = (o: Partial<Occurrence>): Occurrence => ({
		source_id: 's1',
		title: 'Offsite',
		location: 'Lyon',
		start: '2026-10-12T07:00:00Z',
		end: '2026-10-12T09:00:00Z',
		all_day: false,
		date: '2026-10-12',
		last_date: '2026-10-12',
		...o
	});

	it("puts an event on each of its days, in its calendar's colour", () => {
		const out = spreadEvents([at({ all_day: true, last_date: '2026-10-14' })], [work], () => '');
		expect(out.map((e) => e.date)).toEqual(['2026-10-12', '2026-10-13', '2026-10-14']);
		expect(new Set(out.map((e) => e.id)).size).toBe(3); // keyed apart
		expect(out[0]).toMatchObject({
			color: '#ff0000',
			calendar: 'Work',
			time: '',
			location: 'Lyon'
		});
	});

	it('shows the start time of timed events, and a default colour for an unknown calendar', () => {
		const out = spreadEvents([at({ source_id: 'gone' })], [work], (iso) => `at ${iso}`);
		expect(out).toHaveLength(1);
		expect(out[0]).toMatchObject({
			time: 'at 2026-10-12T07:00:00Z',
			color: '#5aa9ff',
			calendar: ''
		});
	});
});
