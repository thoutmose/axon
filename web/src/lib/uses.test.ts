import { describe, expect, it } from 'vitest';
import { PAGES, planFor } from './uses';

describe('planFor', () => {
	it('keeps the pages of the chosen uses and the reflecting ones', () => {
		const { hidden } = planFor(['tasks', 'health']);
		for (const p of [
			'tasks',
			'projects',
			'calendar',
			'focus',
			'health',
			'journal',
			'digest',
			'review'
		])
			expect(hidden).not.toContain(p);
		for (const p of ['notes', 'watch', 'budget', 'media', 'household', 'learn', 'goals'])
			expect(hidden).toContain(p);
	});

	it('lists Today’s sections in their usual order, counts first and progress last', () => {
		expect(planFor(['health', 'tasks']).sections).toEqual([
			'kpis',
			'tasks',
			'milestones',
			'calendar',
			'journal',
			'water',
			'overview'
		]);
	});

	it('hides nothing when every use is chosen', () => {
		const all = planFor([
			'tasks',
			'notes',
			'learning',
			'watch',
			'goals',
			'health',
			'money',
			'media',
			'home'
		]);
		expect(all.hidden).toEqual([]);
		expect(PAGES.length).toBe(21);
	});
});
