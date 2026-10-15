import { describe, expect, it } from 'vitest';
import {
	ago,
	describeSchedule,
	exportURL,
	filterQuery,
	lastDays,
	shareQuery,
	trendWord
} from './watch';

describe('watch helpers', () => {
	it('builds a filter query without empty values', () => {
		expect(filterQuery({})).toBe('');
		expect(filterQuery({ status: 'unread', q: 'rust lang', outliers: false, label: '' })).toBe(
			'?status=unread&q=rust+lang'
		);
		expect(filterQuery({ outliers: true })).toBe('?outliers=1');
	});
	it('says how long ago', () => {
		const now = new Date('2026-10-14T12:00:00Z');
		expect(ago('2026-10-14T11:58:00Z', now)).toBe('2 min');
		expect(ago('2026-10-14T09:00:00Z', now)).toBe('3 h');
		expect(ago('2026-10-12T12:00:00Z', now)).toBe('2 d');
	});
	it('describes common schedules', () => {
		expect(describeSchedule('0 0 * * *')).toBe('every day at midnight');
		expect(describeSchedule('30 7 * * *')).toBe('every day at 07:30');
		expect(describeSchedule('0 8 * * 1-5')).toBe('weekdays at 08:00');
		expect(describeSchedule('0 9 * * 1')).toBe('Mondays at 09:00');
		expect(describeSchedule('0 */6 * * *')).toBe('every 6 hours');
		expect(describeSchedule('5 4 1 * *')).toBe('5 4 1 * *');
	});
});

describe('the news graph', () => {
	it('names a trend', () => {
		expect(trendWord(2)).toBe('rising');
		expect(trendWord(-2)).toBe('fading');
		expect(trendWord(0.4)).toBe('steady');
	});
	it('covers the last days, today included', () => {
		expect(lastDays(7, new Date('2026-10-14T12:00:00Z'))).toEqual({
			from: '2026-10-07',
			to: '2026-10-15'
		});
	});
});

describe('exports and shares', () => {
	it('builds a download address from the filter', () => {
		expect(exportURL('pdf', {})).toBe('/api/v1/watch/export?format=pdf');
		expect(exportURL('csv', { kind: 'video', label: 'ai', status: '' })).toBe(
			'/api/v1/watch/export?kind=video&label=ai&format=csv'
		);
	});
	it('shares the filter without outliers', () => {
		expect(shareQuery({ label: 'ai', outliers: true, from: '2026-10-01' })).toBe(
			'label=ai&from=2026-10-01'
		);
	});
});
