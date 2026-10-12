import { describe, expect, it } from 'vitest';
import { daysUntil, mondayOf, percentLabel } from './goals';

describe('mondayOf', () => {
	it('finds the Monday of the week, Sunday included', () => {
		expect(mondayOf('2026-10-12')).toBe('2026-10-12'); // Monday
		expect(mondayOf('2026-10-10')).toBe('2026-10-05'); // Saturday
		expect(mondayOf('2026-10-11')).toBe('2026-10-05'); // Sunday
		expect(mondayOf('2026-03-01')).toBe('2026-02-23'); // across a month
	});
});

describe('daysUntil', () => {
	it('counts calendar days across daylight-saving changes', () => {
		expect(daysUntil('2026-10-26', '2026-10-24')).toBe(2);
		expect(daysUntil('2026-10-09', '2026-10-10')).toBe(-1);
	});
});

describe('percentLabel', () => {
	it('says when there is nothing to measure', () => {
		expect(percentLabel({ percent: null, method: 'none', done: 0, total: 0, summary: '' })).toBe(
			'No progress measure yet'
		);
		expect(percentLabel({ percent: 40, method: 'items', done: 2, total: 5, summary: '' })).toBe(
			'40%'
		);
	});
});
