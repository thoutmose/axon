import { describe, expect, it } from 'vitest';
import { overdue, relativeDay, type Task } from './tasks';

describe('relativeDay', () => {
	it('names nearby days', () => {
		expect(relativeDay('2026-10-10', '2026-10-10')).toBe('Today');
		expect(relativeDay('2026-10-11', '2026-10-10')).toBe('Tomorrow');
		expect(relativeDay('2026-10-09', '2026-10-10')).toBe('Yesterday');
	});
	it('does not drift across daylight-saving changes', () => {
		expect(relativeDay('2026-10-26', '2026-10-25')).toBe('Tomorrow');
		expect(relativeDay('2026-03-30', '2026-03-29')).toBe('Tomorrow');
	});
});

describe('overdue', () => {
	const task = (due: string | null, status: Task['status'] = 'next') =>
		({ due_on: due, status }) as Task;
	it('is true only for open tasks due before today', () => {
		expect(overdue(task('2026-10-09'), '2026-10-10')).toBe(true);
		expect(overdue(task('2026-10-10'), '2026-10-10')).toBe(false);
		expect(overdue(task('2026-10-09', 'done'), '2026-10-10')).toBe(false);
		expect(overdue(task(null), '2026-10-10')).toBe(false);
	});
});
