import { describe, expect, it } from 'vitest';
import { i18n } from './i18n.svelte';
import { fold, overdue, relativeDay, type Task } from './tasks';

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
	it('speaks the language of the interface', () => {
		i18n.locale = 'fr';
		expect(relativeDay('2026-10-11', '2026-10-10')).toBe('Demain');
		expect(relativeDay('2026-10-16', '2026-10-10')).toBe('ven. 16 oct.');
		i18n.locale = 'en';
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

describe('fold', () => {
	const items = (n: number, p = '') => Array.from({ length: n }, (_, i) => ({ id: `${p}${i}` }));
	it('shows the first five across groups and counts the rest', () => {
		const r = fold(
			[
				['@home', items(7, 'h')],
				['@phone', items(2, 'p')]
			],
			new Set()
		);
		expect(r.groups.map(([k, x]) => [k, x.length])).toEqual([['@home', 5]]);
		expect(r.hidden).toBe(4);
	});
	it('keeps a card just moved there in sight', () => {
		const r = fold([['', items(8)]], new Set(['7']));
		expect(r.groups[0][1].map((x) => x.id)).toEqual(['0', '1', '2', '3', '4', '7']);
		expect(r.hidden).toBe(2);
	});
});
