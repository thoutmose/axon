import type { Schemas } from './api';

// The tech watch (EP-73): types and small helpers shared by its pages.
export type WatchItem = Schemas['WatchItem'];
export type WatchSource = Schemas['WatchSource'];
export type WatchTopic = Schemas['WatchTopic'];
export type WatchStats = Schemas['WatchStats'];

export type WatchFilter = {
	status?: '' | 'unread' | 'read' | 'saved' | 'archived' | 'all';
	source?: string;
	topic?: string;
	label?: string;
	q?: string;
	from?: string;
	to?: string;
	outliers?: boolean;
};

/** The query string for a filter, empty values left out. */
export function filterQuery(f: WatchFilter): string {
	const p = new URLSearchParams();
	for (const [k, v] of Object.entries(f)) {
		if (k === 'outliers') {
			if (v) p.set('outliers', '1');
		} else if (v) p.set(k, String(v));
	}
	const s = p.toString();
	return s ? `?${s}` : '';
}

/** "3 h ago", "yesterday", "12 Oct": short, for lists. */
export function ago(iso: string, now = new Date(), locale?: string): string {
	const t = new Date(iso);
	const min = Math.round((now.getTime() - t.getTime()) / 60000);
	if (min < 1) return 'now';
	if (min < 60) return `${min} min`;
	const h = Math.round(min / 60);
	if (h < 24) return `${h} h`;
	const d = Math.round(h / 24);
	if (d < 7) return `${d} d`;
	return t.toLocaleDateString(locale, { day: 'numeric', month: 'short' });
}

/** A schedule in words, for the common cron forms. */
export function describeSchedule(cron: string): string {
	const m = cron.trim().match(/^(\d{1,2}) (\d{1,2}) \* \* (\*|[0-7](?:-[0-7])?)$/);
	if (cron === '0 0 * * *') return 'every day at midnight';
	if (m) {
		const at = `${m[2].padStart(2, '0')}:${m[1].padStart(2, '0')}`;
		if (m[3] === '*') return `every day at ${at}`;
		if (m[3] === '1-5') return `weekdays at ${at}`;
		const days = [
			'Sunday',
			'Monday',
			'Tuesday',
			'Wednesday',
			'Thursday',
			'Friday',
			'Saturday',
			'Sunday'
		];
		if (/^[0-7]$/.test(m[3])) return `${days[Number(m[3])]}s at ${at}`;
	}
	const h = cron.match(/^0 \*\/(\d+) \* \* \*$/);
	if (h) return `every ${h[1]} hours`;
	if (cron === '0 * * * *' || cron === '@hourly') return 'every hour';
	return cron;
}

/** The normalised CSV header, for the import help. */
export const csvHeader = 'url,title,topics,schedule,full_text';
