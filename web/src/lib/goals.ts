import type { Schemas } from './api';

export type Goal = Schemas['Goal'];
export type Milestone = Schemas['Milestone'];
export type Measure = Schemas['Measure'];
export type Progress = Schemas['Progress'];

/** The Monday of the week containing an ISO date: weekly check-ins live there. */
export function mondayOf(iso: string): string {
	const d = new Date(`${iso}T00:00:00Z`);
	d.setUTCDate(d.getUTCDate() - ((d.getUTCDay() + 6) % 7));
	return d.toISOString().slice(0, 10);
}

/** "50%" or "No progress measure yet", for labels and screen readers. */
export function percentLabel(p: Progress): string {
	return p.percent === null ? 'No progress measure yet' : `${p.percent}%`;
}

/** Days from today to a target date; negative when it has passed. */
export function daysUntil(target: string, today: string): number {
	return Math.round(
		(new Date(`${target}T00:00:00Z`).getTime() - new Date(`${today}T00:00:00Z`).getTime()) /
			86_400_000
	);
}
