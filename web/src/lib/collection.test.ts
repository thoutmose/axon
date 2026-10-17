import { describe, expect, it } from 'vitest';
import { achievement, cardName, nextStep, groups } from './collection';

const card = (over: Record<string, unknown>) =>
	({
		id: 'x',
		family: 'practice',
		title: '',
		kind: 'hard',
		tier: '',
		value: 0,
		unit: 'problems',
		next: { tier: 'bronze', target: 1 },
		...over
	}) as Parameters<typeof cardName>[0];

describe('the collection (EP-119)', () => {
	it('names roadmap, project, and goal cards after them, and says what is left (EP-170)', () => {
		const goal = card({
			family: 'goal',
			title: 'Run 10 km',
			value: 50,
			unit: 'percent',
			next: { tier: 'gold', target: 100 }
		});
		expect(cardName(goal)).toBe('Run 10 km');
		expect(nextStep(goal)).toBe('100 % of its milestones reached for gold');
		expect(cardName(card({ family: 'milestones' }))).toBe('Milestones reached');
		const g = groups([
			goal,
			card({ family: 'project', title: 'pipekit' }),
			card({ family: 'exams' })
		]);
		expect(g.map((x) => x.id)).toEqual(['goals', 'projects', 'milestones']);
	});
	it('names a card by its skill or habit, or by what it counts', () => {
		expect(cardName(card({ family: 'habit', title: 'Read 20 pages' }))).toBe('Read 20 pages');
		expect(cardName(card({}))).toBe('Hard problems');
		expect(cardName(card({ family: 'streak' }))).toBe('Daily streak');
	});
	it('says what the next tier takes, in whole numbers', () => {
		expect(nextStep(card({ value: 0 }))).toBe('1 more solved for bronze');
		expect(
			nextStep(
				card({ family: 'habit', value: 7, unit: 'days', next: { tier: 'silver', target: 21 } })
			)
		).toBe('14 more days in a row for silver');
		expect(
			nextStep(
				card({ family: 'skill', value: 3.2, unit: 'hours', next: { tier: 'silver', target: 10 } })
			)
		).toBe('7 more hours of focus on it for silver');
		expect(nextStep(card({ family: 'skill', next: { tier: 'bronze', target: 0 } }))).toBe(
			'Start learning it for bronze'
		);
		expect(
			nextStep(card({ family: 'skill', tier: 'silver', next: { tier: 'gold', target: 0 } }))
		).toBe('Mark it learned for gold');
		expect(nextStep(card({ tier: 'gold', next: null }))).toBe('Complete');
	});
	it('groups skills, habits, and the rest', () => {
		const g = groups([card({ family: 'skill' }), card({ family: 'habit' }), card({})]);
		expect(g.map((x) => [x.id, x.cards.length])).toEqual([
			['skills', 1],
			['habits', 1],
			['milestones', 1]
		]);
	});
	it('says what an earned card stands for', () => {
		expect(achievement(card({ family: 'habit', tier: 'silver', value: 23 }))).toBe(
			'Kept 23 scheduled days in a row.'
		);
		expect(achievement(card({ tier: 'bronze', value: 3 }))).toBe('3 problems solved unaided.');
	});
});
