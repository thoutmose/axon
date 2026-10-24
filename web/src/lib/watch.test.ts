import { describe, expect, it } from 'vitest';
import {
	ago,
	describeSchedule,
	exportURL,
	fetchedWhen,
	filterQuery,
	lastDays,
	latestFetch,
	sourceLate,
	watchStale,
	shareQuery,
	trendWord
} from './watch';
import { fr } from './i18n.fr';

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
	it('finds the latest fetch and warns after 24 hours without one', () => {
		const now = new Date('2026-10-24T12:00:00Z');
		const src = (last: string | null, enabled = true, created = '2026-10-01T00:00:00Z') => ({
			last_fetched_at: last,
			enabled,
			created_at: created
		});
		expect(latestFetch([src(null), src('2026-10-24T01:00:00Z'), src('2026-10-23T01:00:00Z')])).toBe(
			'2026-10-24T01:00:00Z'
		);
		expect(latestFetch([src(null)])).toBeNull();
		expect(watchStale([src('2026-10-24T01:00:00Z')], now)).toBe(false);
		expect(watchStale([src('2026-10-23T12:00:01Z')], now)).toBe(false);
		expect(watchStale([src('2026-10-23T11:59:00Z')], now)).toBe(true);
		// Never fetched: followed over a day ago warns, followed 5 min ago does not.
		expect(watchStale([src(null)], now)).toBe(true);
		expect(watchStale([src(null, true, '2026-10-24T11:55:00Z')], now)).toBe(false);
		// A new source does not hide an old one stuck for days.
		expect(
			watchStale([src(null, true, '2026-10-24T11:55:00Z'), src('2026-10-20T00:00:00Z')], now)
		).toBe(true);
		expect(watchStale([], now)).toBe(false);
		expect(watchStale([src(null, false)], now)).toBe(false);
		// A source turned off but fetched recently does not hide the others.
		expect(watchStale([src('2026-10-24T11:00:00Z', false), src('2026-10-20T00:00:00Z')], now)).toBe(
			true
		);
	});
	it('says when a source was fetched, as a date from a week on', () => {
		const now = new Date('2026-10-24T12:00:00Z');
		expect(fetchedWhen('2026-10-24T11:59:40Z', now)).toEqual({ key: 'Fetched just now' });
		expect(fetchedWhen('2026-10-24T11:55:00Z', now)).toEqual({
			key: 'Fetched {n} min ago',
			vars: { n: 5 }
		});
		expect(fetchedWhen('2026-10-24T09:00:00Z', now)).toEqual({
			key: 'Fetched {n} h ago',
			vars: { n: 3 }
		});
		expect(fetchedWhen('2026-10-21T12:00:00Z', now)).toEqual({
			key: 'Fetched {n} d ago',
			vars: { n: 3 }
		});
		const old = fetchedWhen('2026-10-12T12:00:00Z', now, 'fr-FR');
		expect(old.key).toBe('Fetched on {date}');
		expect(old.vars?.date).toBe('12 oct. 2026');
		expect(fetchedWhen('2026-10-12T12:00:00Z', now, 'en-GB').vars?.date).toBe('12 Oct 2026');
		expect(fr['Fetched on {date}']).toBe('Récupérée le {date}');
		expect(fr['Fetched {n} d ago']).toBe('Récupérée il y a {n} j');
	});
	it('marks an enabled source late after 24 hours without a fetch', () => {
		const now = new Date('2026-10-24T12:00:00Z');
		const old = '2026-10-01T00:00:00Z';
		const src = (last: string | null, enabled = true, created_at = old) => ({
			last_fetched_at: last,
			enabled,
			created_at
		});
		expect(sourceLate(src('2026-10-24T01:00:00Z'), now)).toBe(false);
		expect(sourceLate(src('2026-10-23T11:00:00Z'), now)).toBe(true);
		// Never fetched: counted from when it was followed.
		expect(sourceLate(src(null), now)).toBe(true);
		expect(sourceLate(src(null, true, '2026-10-24T11:55:00Z'), now)).toBe(false);
		expect(sourceLate(src(null, true, '2026-10-23T11:00:00Z'), now)).toBe(true);
		expect(sourceLate(src(null, false), now)).toBe(false);
	});
	it('does not print NaN for a date it cannot read', () => {
		expect(fetchedWhen('not a date')).toEqual({ key: 'Never fetched' });
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
