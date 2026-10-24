import type { Schemas } from './api';

// The tech watch (EP-110): types and small helpers shared by its pages.
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
	kind?: '' | 'article' | 'video' | 'podcast' | 'news';
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

/**
 * When a source was last fetched, as an English key and its values for t():
 * minutes, hours, days, then the date itself from a week on, so the sentence
 * never reads "12 Oct ago".
 */
export function fetchedWhen(
	iso: string,
	now = new Date(),
	locale?: string
): { key: string; vars?: Record<string, string | number> } {
	const at = new Date(iso);
	if (Number.isNaN(at.getTime())) return { key: 'Never fetched' };
	const min = Math.round((now.getTime() - at.getTime()) / 60000);
	if (min < 1) return { key: 'Fetched just now' };
	if (min < 60) return { key: 'Fetched {n} min ago', vars: { n: min } };
	const h = Math.round(min / 60);
	if (h < 24) return { key: 'Fetched {n} h ago', vars: { n: h } };
	const d = Math.round(h / 24);
	if (d < 7) return { key: 'Fetched {n} d ago', vars: { n: d } };
	const date = at.toLocaleDateString(locale, { day: 'numeric', month: 'short', year: 'numeric' });
	return { key: 'Fetched on {date}', vars: { date } };
}

const DAY_MS = 24 * 3600 * 1000;

/**
 * True when an enabled source was not fetched in the last 24 hours. A source
 * never fetched counts from when it was followed, so a new one is not late.
 */
export function sourceLate(
	s: Pick<WatchSource, 'last_fetched_at' | 'enabled' | 'created_at'>,
	now = new Date()
): boolean {
	if (!s.enabled) return false;
	const since = Date.parse(s.last_fetched_at ?? s.created_at);
	return now.getTime() - since > DAY_MS;
}

/** The most recent fetch across the sources, or null when none ever ran. */
export function latestFetch(sources: Pick<WatchSource, 'last_fetched_at'>[]): string | null {
	let best: string | null = null;
	for (const s of sources) {
		if (s.last_fetched_at && (!best || Date.parse(s.last_fetched_at) > Date.parse(best))) {
			best = s.last_fetched_at;
		}
	}
	return best;
}

/**
 * True when no enabled source was fetched in the last 24 hours and at least
 * one of them is late: sources all followed within the day do not warn.
 */
export function watchStale(
	sources: Pick<WatchSource, 'last_fetched_at' | 'enabled' | 'created_at'>[],
	now = new Date()
): boolean {
	// Only enabled sources count: a source turned off but fetched recently
	// must not hide that the others are late.
	const on = sources.filter((s) => s.enabled);
	const last = latestFetch(on);
	if (last && now.getTime() - Date.parse(last) <= DAY_MS) return false;
	return on.some((s) => sourceLate(s, now));
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

export type WatchGraph = Schemas['WatchGraph'];

/** How a concept is moving: its trend score in words. */
export function trendWord(trend: number): 'rising' | 'fading' | 'steady' {
	if (trend > 1.5) return 'rising';
	if (trend < -1.5) return 'fading';
	return 'steady';
}

/** The from and to of the last `days` days, as dates, for the graph. */
export function lastDays(days: number, now = new Date()): { from: string; to: string } {
	const to = new Date(now.getTime() + 86400000);
	const from = new Date(now.getTime() - days * 86400000);
	const day = (d: Date) => d.toISOString().slice(0, 10);
	return { from: day(from), to: day(to) };
}

export type WatchShare = Schemas['WatchShare'];

/** Export formats, in the order they are offered. */
export const exportFormats: [string, string][] = [
	['pdf', 'PDF'],
	['ods', 'Spreadsheet (ODS)'],
	['csv', 'CSV'],
	['md', 'Markdown'],
	['zip', 'Feed notes (zip)'],
	['json', 'JSON']
];

/** The download address of the news a filter selects. */
export function exportURL(format: string, f: WatchFilter): string {
	const qs = filterQuery(f);
	return `/api/v1/watch/export${qs ? qs + '&' : '?'}format=${format}`;
}

/** The filter as a share link keeps it (no outliers, no paging). */
export function shareQuery(f: WatchFilter): string {
	const { outliers: _o, ...rest } = f;
	return filterQuery(rest).slice(1);
}
