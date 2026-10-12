// Calendar dates as the API sends them: "YYYY-MM-DD", with no time and no
// time zone. Arithmetic is done at UTC midnight so a daylight saving change
// never adds or loses a day, and dates are shown in UTC so the browser's
// time zone never moves them to the day before or after.

const DAY = 86_400_000;

const utc = (iso: string) => Date.parse(`${iso}T00:00:00Z`);
const iso = (ms: number) => new Date(ms).toISOString().slice(0, 10);

/** The browser's own calendar date at d (not UTC's). */
export function localDate(d: Date): string {
	const p = (n: number) => String(n).padStart(2, '0');
	return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

/** The date n days after (or before, when negative) a date. */
export function addDays(day: string, n: number): string {
	return iso(utc(day) + n * DAY);
}

/** Days from one date to another; negative when `to` is earlier. */
export function daysBetween(from: string, to: string): number {
	return Math.round((utc(to) - utc(from)) / DAY);
}

/** ISO weekday of a date: 1 is Monday, 7 is Sunday. */
export function isoWeekday(day: string): number {
	return new Date(utc(day)).getUTCDay() || 7;
}

/** The Monday of the week a date is in. */
export function mondayOf(day: string): string {
	return addDays(day, 1 - isoWeekday(day));
}

/** A date shown as that day, in the browser's language unless one is given. */
export function formatDate(
	day: string,
	options: Intl.DateTimeFormatOptions = {},
	locale?: string
): string {
	return new Date(utc(day)).toLocaleDateString(locale, { ...options, timeZone: 'UTC' });
}
