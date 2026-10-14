// Certifications (EP-104): what the page says about each one and in all.
export type Cert = {
	id: string;
	name: string;
	code: string;
	status: 'planned' | 'booked' | 'passed' | 'failed';
	exam_on: string | null;
	expires_on: string | null;
	cost_cents: number | null;
	currency: string;
};

/** Whole days from one ISO date to another. */
export const daysBetween = (from: string, to: string) =>
	Math.round((Date.parse(to + 'T00:00:00Z') - Date.parse(from + 'T00:00:00Z')) / 86_400_000);

/** Totals by currency: what the plan will cost, and what was paid (booked, passed, or failed). */
export function costs(items: Cert[]): { currency: string; planned: number; paid: number }[] {
	const by = new Map<string, { currency: string; planned: number; paid: number }>();
	for (const c of items) {
		if (c.cost_cents == null) continue;
		const row = by.get(c.currency) ?? { currency: c.currency, planned: 0, paid: 0 };
		row.planned += c.cost_cents;
		if (c.status !== 'planned') row.paid += c.cost_cents;
		by.set(c.currency, row);
	}
	return [...by.values()];
}

/** The next booked exam from today, if any. */
export function nextExam(items: Cert[], today: string): Cert | null {
	return (
		items
			.filter((c) => c.status === 'booked' && c.exam_on && c.exam_on >= today)
			.sort((a, b) => (a.exam_on! < b.exam_on! ? -1 : 1))[0] ?? null
	);
}

/** Passed certifications expiring within the given days (or already expired). */
export const expiringSoon = (items: Cert[], today: string, within = 90) =>
	items.filter(
		(c) => c.status === 'passed' && c.expires_on && daysBetween(today, c.expires_on) <= within
	);
