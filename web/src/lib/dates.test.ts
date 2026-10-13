import { afterEach, describe, expect, it, vi } from 'vitest';
import {
	addDays,
	appLocale,
	daysBetween,
	formatDate,
	isoWeekday,
	localDate,
	mondayOf
} from './dates';
import { i18n } from './i18n.svelte';

// The browser's time zone, for the tests that change it.
afterEach(() => {
	vi.unstubAllEnvs();
});

describe('dates', () => {
	it('adds days across months, years, and leap days', () => {
		expect(addDays('2026-12-30', 3)).toBe('2027-01-02');
		expect(addDays('2028-02-28', 1)).toBe('2028-02-29');
		expect(addDays('2026-03-01', -1)).toBe('2026-02-28');
		expect(addDays('2026-10-12', 0)).toBe('2026-10-12');
	});

	it('counts days between dates, negative when the second is earlier', () => {
		expect(daysBetween('2026-10-12', '2026-10-19')).toBe(7);
		expect(daysBetween('2026-10-12', '2026-10-11')).toBe(-1);
		// Across a daylight saving change, a day is still a day.
		expect(daysBetween('2026-03-28', '2026-03-30')).toBe(2);
	});

	it('finds the Monday of a week, and the ISO weekday', () => {
		expect(mondayOf('2026-10-12')).toBe('2026-10-12'); // a Monday
		expect(mondayOf('2026-10-18')).toBe('2026-10-12'); // the Sunday after
		expect(mondayOf('2027-01-01')).toBe('2026-12-28');
		expect(mondayOf('2026-03-01')).toBe('2026-02-23'); // across a month
		expect(isoWeekday('2026-10-12')).toBe(1);
		expect(isoWeekday('2026-10-18')).toBe(7);
	});

	it("gives the browser's own calendar date, not UTC's", () => {
		vi.stubEnv('TZ', 'America/Los_Angeles');
		// 23:30 on the 12th in Los Angeles is already the 13th in UTC.
		expect(localDate(new Date('2026-10-13T06:30:00Z'))).toBe('2026-10-12');
		vi.stubEnv('TZ', 'Pacific/Kiritimati');
		expect(localDate(new Date('2026-10-12T11:00:00Z'))).toBe('2026-10-13');
	});

	it('shows a date as that day, wherever the browser is', () => {
		for (const tz of ['America/Los_Angeles', 'UTC', 'Pacific/Kiritimati']) {
			vi.stubEnv('TZ', tz);
			expect(formatDate('2026-10-12', { day: 'numeric', month: 'long' }, 'en-GB')).toBe(
				'12 October'
			);
		}
		expect(formatDate('2026-10-12', undefined, 'en-GB')).toBe('12/10/2026');
	});

	it('writes dates in the language of the interface', () => {
		i18n.locale = 'fr';
		expect(appLocale()).toMatch(/^fr/);
		expect(formatDate('2026-10-12', { day: 'numeric', month: 'long' })).toBe('12 octobre');
		i18n.locale = 'en';
		expect(appLocale() ?? 'en').toMatch(/^en/);
	});
});
