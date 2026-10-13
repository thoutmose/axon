import { describe, expect, it } from 'vitest';
import { mealRequest } from './mealPlan';

describe('the request a meal plan form sends to the chef', () => {
	it('says everything the chef needs, in plain words', () => {
		expect(
			mealRequest({
				days: 3,
				people: 2,
				meals: ['breakfast', 'dinner'],
				budget: '40',
				currency: 'EUR',
				restrictions: ['vegetarian', 'nuts'],
				minutes: 30,
				other: '  No oven. '
			})
		).toBe(
			'Plan 3 days of breakfast and dinner for 2 people. Budget: 40 EUR in all. Dietary restrictions and allergies: vegetarian, nut allergy. At most 30 minutes of active cooking per meal. Other constraints: No oven.'
		);
	});
	it('says when there is no budget, restriction, or other constraint', () => {
		expect(
			mealRequest({
				days: 1,
				people: 1,
				meals: ['lunch'],
				budget: '',
				currency: 'EUR',
				restrictions: [],
				minutes: 15,
				other: ''
			})
		).toBe(
			'Plan 1 day of lunch for 1 person. Budget: none set. Dietary restrictions and allergies: none. At most 15 minutes of active cooking per meal.'
		);
	});
});
