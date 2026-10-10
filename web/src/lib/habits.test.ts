import { describe, expect, it } from 'vitest';
import { isoWeekday, lastDays, scheduleLabel } from './habits';

describe('scheduleLabel', () => {
	it('names common schedules', () => {
		expect(scheduleLabel([1, 2, 3, 4, 5, 6, 7])).toBe('Every day');
		expect(scheduleLabel([5, 4, 3, 2, 1])).toBe('Weekdays');
		expect(scheduleLabel([1, 3, 5])).toBe('Mon, Wed, Fri');
	});
});

describe('lastDays', () => {
	it('ends today and crosses month boundaries', () => {
		expect(lastDays('2026-03-02', 3)).toEqual(['2026-02-28', '2026-03-01', '2026-03-02']);
	});
});

describe('isoWeekday', () => {
	it('counts Monday as 1 and Sunday as 7', () => {
		expect(isoWeekday('2026-10-12')).toBe(1);
		expect(isoWeekday('2026-10-11')).toBe(7);
	});
});
