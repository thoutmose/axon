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

/** The four laws, or their inversions for a habit to break (Atomic Habits). */
export const laws = {
	build: [
		{ field: 'cue', law: 'Make it obvious', prompt: 'Where will you see the cue?' },
		{
			field: 'craving',
			law: 'Make it attractive',
			prompt: 'Why does it matter to you, or what will you pair it with?'
		},
		{ field: 'two_minute', law: 'Make it easy', prompt: 'The two-minute version' },
		{ field: 'reward', law: 'Make it satisfying', prompt: 'An immediate reward afterwards' }
	],
	break: [
		{ field: 'cue', law: 'Make it invisible', prompt: 'How will you remove the cue?' },
		{ field: 'craving', law: 'Make it unattractive', prompt: 'What does it really cost you?' },
		{ field: 'two_minute', law: 'Make it difficult', prompt: 'What friction will you add?' },
		{
			field: 'reward',
			law: 'Make it unsatisfying',
			prompt: 'What cost or accountability will follow?'
		}
	]
} as const;

type PlanFields = Pick<
	Habit,
	'title' | 'kind' | 'stack_after' | 'intention_time' | 'intention_place'
>;

/** "After I pour my coffee, I will read one page." */
function stackSentence(h: PlanFields): string | null {
	const after = h.stack_after
		.trim()
		.replace(/^after\s+/i, '')
		.replace(/[.,]$/, '');
	if (!after) return null;
	return `After ${after}, I will ${lower(h.title)}.`;
}

const lower = (s: string) => s.charAt(0).toLowerCase() + s.slice(1);

/** The plan in James Clear's formulas, as the person would say it. */
export function plan(h: PlanFields): string[] {
	if (h.kind === 'break') return [`I am stopping: ${lower(h.title)}.`];
	const out: string[] = [];
	const stack = stackSentence(h);
	if (stack) out.push(stack);
	if (h.intention_time || h.intention_place) {
		const when = h.intention_time ? ` at ${h.intention_time}` : '';
		const where = h.intention_place
			? ` ${h.intention_place.replace(/^(in|at|on)?\s*/i, (m) => (m.trim() ? m : 'in '))}`
			: '';
		out.push(`I will ${lower(h.title)}${when}${where}.`);
	}
	return out;
}

/**
 * Habit stacks: a habit whose "after" names another habit follows it. Chains
 * are returned first habit first; habits outside any chain are left out.
 */
export function chains(habits: Habit[]): Habit[][] {
	const norm = (s: string) =>
		s
			.toLowerCase()
			.replace(/^after\s+(i\s+)?/, '')
			.replace(/[.,]$/, '')
			.trim();
	const byTitle = new Map(habits.map((h) => [norm(h.title), h]));
	const next = new Map<string, Habit[]>();
	const hasParent = new Set<string>();
	for (const h of habits) {
		const parent = byTitle.get(norm(h.stack_after));
		if (parent && parent.id !== h.id) {
			next.set(parent.id, [...(next.get(parent.id) ?? []), h]);
			hasParent.add(h.id);
		}
	}
	const out: Habit[][] = [];
	for (const root of habits) {
		if (hasParent.has(root.id) || !next.has(root.id)) continue;
		const chain = [root];
		const seen = new Set([root.id]);
		let cur = root;
		while (next.get(cur.id)?.length) {
			const child = next.get(cur.id)![0];
			if (seen.has(child.id)) break;
			chain.push(child);
			seen.add(child.id);
			cur = child;
		}
		out.push(chain);
	}
	return out;
}
