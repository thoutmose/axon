import { addDays, formatDate, isoWeekday, mondayOf } from './dates';

/** First and last day of the month containing an ISO date or "YYYY-MM". */
export function monthRange(month: string): { from: string; to: string } {
	const [y, m] = month.split('-').map(Number);
	const last = new Date(Date.UTC(y, m, 0)).getUTCDate();
	const mm = String(m).padStart(2, '0');
	return { from: `${y}-${mm}-01`, to: `${y}-${mm}-${String(last).padStart(2, '0')}` };
}

/** "YYYY-MM" shifted by a number of months. */
export function shiftMonth(month: string, by: number): string {
	const [y, m] = month.split('-').map(Number);
	const d = new Date(Date.UTC(y, m - 1 + by, 1));
	return d.toISOString().slice(0, 7);
}

/** "October 2026". */
export function monthLabel(month: string): string {
	return formatDate(`${month}-01`, { month: 'long', year: 'numeric' });
}

/** Groups items by their date, keeping order. */
export function byDay<T extends { date: string }>(items: T[]): [string, T[]][] {
	const groups = new Map<string, T[]>();
	for (const it of items) groups.set(it.date, [...(groups.get(it.date) ?? []), it]);
	return [...groups];
}

/** Weeks (Monday first) covering a "YYYY-MM" month, as ISO dates, padded with the neighbouring months' days. */
export function monthGrid(month: string): string[][] {
	const { from, to } = monthRange(month);
	const weeks: string[][] = [];
	for (let d = mondayOf(from); d <= addDays(mondayOf(to), 6); d = addDays(d, 1)) {
		if (isoWeekday(d) === 1) weeks.push([]);
		weeks[weeks.length - 1].push(d);
	}
	return weeks;
}

/** An event of a followed calendar (EP-58), as the API lists it. */
export type Occurrence = {
	source_id: string;
	title: string;
	location: string;
	start: string;
	end: string;
	all_day: boolean;
	date: string;
	last_date: string;
};

/** A followed calendar's event on one day of the grid, read only. */
export type External = {
	type: 'external';
	kind: 'external';
	id: string;
	date: string;
	title: string;
	done: false;
	color: string;
	time: string;
	location: string;
	calendar: string;
};

/**
 * Followed calendars' events as grid entries: one on each day an event
 * covers, in its calendar's colour, with its start time (from `clock`) unless
 * it is all day.
 */
export function spreadEvents(
	occurrences: Occurrence[],
	sources: { id: string; name: string; color: string }[],
	clock: (iso: string) => string
): External[] {
	const byId = new Map(sources.map((src) => [src.id, src]));
	const out: External[] = [];
	for (const o of occurrences) {
		const src = byId.get(o.source_id);
		const time = o.all_day ? '' : clock(o.start);
		for (let d = o.date; d <= o.last_date; d = addDays(d, 1)) {
			out.push({
				type: 'external',
				kind: 'external',
				id: `${o.source_id}-${o.start}-${d}`,
				date: d,
				title: o.title,
				done: false,
				color: src?.color ?? '#5aa9ff',
				time,
				location: o.location,
				calendar: src?.name ?? ''
			});
		}
	}
	return out;
}
