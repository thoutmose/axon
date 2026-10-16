import { t } from './i18n.svelte';
import { api, type Schemas } from './api';
import { daysBetween, formatDate } from './dates';

export type Task = Schemas['Task'];
export type TaskFields = Schemas['TaskFields'];

/** Recurrence presets offered in the editor; anything else is "custom". */
export const recurrencePresets: { label: string; rule: string | null }[] = [
	{ label: 'Does not repeat', rule: null },
	{ label: 'Every day', rule: 'FREQ=DAILY' },
	{ label: 'Every weekday', rule: 'FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR' },
	{ label: 'Every week', rule: 'FREQ=WEEKLY' },
	{ label: 'Every month', rule: 'FREQ=MONTHLY' },
	{ label: 'Every year', rule: 'FREQ=YEARLY' }
];

export function capture(title: string, extra: TaskFields = {}): Promise<Task> {
	return api<Task>('/v1/tasks', { method: 'POST', body: { title, ...extra } });
}

/** Human date relative to today ("Today", "Tomorrow", "Mon 12 Oct"). */
export function relativeDay(iso: string, today: string): string {
	const diff = daysBetween(today, iso);
	if (diff === 0) return t('Today');
	if (diff === 1) return t('Tomorrow');
	if (diff === -1) return t('Yesterday');
	return formatDate(iso, {
		weekday: 'short',
		day: 'numeric',
		month: 'short',
		...(iso.slice(0, 4) !== today.slice(0, 4) ? { year: 'numeric' } : {})
	});
}

export function overdue(task: Task, today: string): boolean {
	return (
		!!task.due_on && task.due_on < today && task.status !== 'done' && task.status !== 'cancelled'
	);
}

/** Cards a board column shows before "Show N more" (EP-128). */
export const BOARD_LIMIT = 5;

/**
 * The cards of a folded column: the first `limit` of each group, and any card
 * in `keep` (one just moved there, so it never vanishes). `hidden` counts the rest.
 */
export function fold<T extends { id: string }>(
	groups: [string, T[]][],
	keep: Set<string>,
	limit = BOARD_LIMIT
): { groups: [string, T[]][]; hidden: number } {
	let hidden = 0;
	const out = groups.map(([key, items]): [string, T[]] => {
		const shown = items.filter((x, i) => i < limit || keep.has(x.id));
		hidden += items.length - shown.length;
		return [key, shown];
	});
	return { groups: out, hidden };
}
