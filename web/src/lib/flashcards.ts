import type { Schemas } from './api';

export type Flashcard = Schemas['Flashcard'];

/** "now", "10 min", "1 day", "3 days", "2 mo", "1.5 yr": what a rating does. */
export function interval(from: Date, to: Date): string {
	const ms = to.getTime() - from.getTime();
	if (ms < 60_000) return 'now';
	const min = Math.round(ms / 60_000);
	if (min < 60) return `${min} min`;
	const h = Math.round(min / 60);
	if (h < 24) return `${h} h`;
	const d = Math.round(h / 24);
	if (d < 31) return `${d} ${d === 1 ? 'day' : 'days'}`;
	const mo = d / 30.4;
	if (mo < 12) return `${Math.round(mo)} mo`;
	const yr = Math.round((d / 365) * 10) / 10;
	return `${yr} yr`;
}

/** A shuffled sample of n items (Fisher–Yates), for practice tests. */
export function sample<T>(items: T[], n: number, random = Math.random): T[] {
	const a = [...items];
	for (let i = a.length - 1; i > 0; i--) {
		const j = Math.floor(random() * (i + 1));
		[a[i], a[j]] = [a[j], a[i]];
	}
	return a.slice(0, n);
}

export const ratings = [
	{ value: 1, label: 'Again', key: '1', next: 'again' },
	{ value: 2, label: 'Hard', key: '2', next: 'hard' },
	{ value: 3, label: 'Good', key: '3', next: 'good' },
	{ value: 4, label: 'Easy', key: '4', next: 'easy' }
] as const;
