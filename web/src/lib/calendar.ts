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
	return new Date(`${month}-01T00:00:00Z`).toLocaleDateString(undefined, {
		month: 'long',
		year: 'numeric',
		timeZone: 'UTC'
	});
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
	const day = (iso: string) => new Date(`${iso}T00:00:00Z`);
	const start = day(from);
	start.setUTCDate(start.getUTCDate() - ((start.getUTCDay() + 6) % 7));
	const end = day(to);
	end.setUTCDate(end.getUTCDate() + ((7 - end.getUTCDay()) % 7));
	const weeks: string[][] = [];
	for (const d = start; d <= end; d.setUTCDate(d.getUTCDate() + 1)) {
		if ((d.getUTCDay() + 6) % 7 === 0) weeks.push([]);
		weeks[weeks.length - 1].push(d.toISOString().slice(0, 10));
	}
	return weeks;
}
