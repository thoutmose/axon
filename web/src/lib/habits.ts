import type { Schemas } from './api';

export type Habit = Schemas['Habit'];
export type HabitCheckin = Schemas['HabitCheckin'];

export const weekdays = [
	{ id: 1, short: 'Mon', long: 'Monday' },
	{ id: 2, short: 'Tue', long: 'Tuesday' },
	{ id: 3, short: 'Wed', long: 'Wednesday' },
	{ id: 4, short: 'Thu', long: 'Thursday' },
	{ id: 5, short: 'Fri', long: 'Friday' },
	{ id: 6, short: 'Sat', long: 'Saturday' },
	{ id: 7, short: 'Sun', long: 'Sunday' }
];

/** "Every day", "Weekdays", or "Mon, Wed, Fri". */
export function scheduleLabel(days: number[]): string {
	const sorted = [...days].sort();
	if (sorted.length === 7) return 'Every day';
	if (sorted.join() === '1,2,3,4,5') return 'Weekdays';
	if (sorted.join() === '6,7') return 'Weekends';
	return sorted.map((d) => weekdays[d - 1].short).join(', ');
}

/** The last n days ending today, oldest first, as ISO dates. */
export function lastDays(today: string, n: number): string[] {
	const end = new Date(`${today}T00:00:00Z`);
	return Array.from({ length: n }, (_, i) => {
		const d = new Date(end);
		d.setUTCDate(end.getUTCDate() - (n - 1 - i));
		return d.toISOString().slice(0, 10);
	});
}

/** ISO weekday (1 = Monday) of an ISO date. */
export function isoWeekday(iso: string): number {
	const wd = new Date(`${iso}T00:00:00Z`).getUTCDay();
	return wd === 0 ? 7 : wd;
}
