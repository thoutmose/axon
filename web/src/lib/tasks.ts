import { api, type Schemas } from './api';

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
	const d = new Date(`${iso}T00:00:00Z`);
	const t = new Date(`${today}T00:00:00Z`);
	const diff = Math.round((d.getTime() - t.getTime()) / 86_400_000);
	if (diff === 0) return 'Today';
	if (diff === 1) return 'Tomorrow';
	if (diff === -1) return 'Yesterday';
	return d.toLocaleDateString(undefined, {
		weekday: 'short',
		day: 'numeric',
		month: 'short',
		timeZone: 'UTC',
		...(d.getUTCFullYear() !== t.getUTCFullYear() ? { year: 'numeric' } : {})
	});
}

export function overdue(task: Task, today: string): boolean {
	return (
		!!task.due_on && task.due_on < today && task.status !== 'done' && task.status !== 'cancelled'
	);
}
