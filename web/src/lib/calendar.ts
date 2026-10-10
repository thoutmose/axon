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
