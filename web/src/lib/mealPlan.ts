import { t } from './i18n.svelte';

/** What the meal plan form asks the Chef & Nutritionist for (EP-90). */
export type MealPlanForm = {
	days: number;
	people: number;
	meals: string[];
	budget: string;
	currency: string;
	restrictions: string[];
	minutes: number;
	other: string;
};

/** Restrictions and allergies offered as checkboxes, with how they read. */
export const restrictionNames: Record<string, string> = {
	vegetarian: 'vegetarian',
	vegan: 'vegan',
	pescatarian: 'pescatarian',
	halal: 'halal',
	kosher: 'kosher',
	gluten: 'gluten-free',
	lactose: 'lactose-free',
	nuts: 'nut allergy',
	peanuts: 'peanut allergy',
	eggs: 'egg allergy',
	shellfish: 'shellfish allergy',
	soy: 'soy allergy'
};

export const mealNames: Record<string, string> = {
	breakfast: 'breakfast',
	lunch: 'lunch',
	dinner: 'dinner',
	snack: 'snacks'
};

/** The request, in the interface's language. */
export function mealRequest(f: MealPlanForm): string {
	const and = (items: string[]) =>
		items.length > 1
			? `${items.slice(0, -1).join(', ')} ${t('and')} ${items[items.length - 1]}`
			: (items[0] ?? '');
	const parts = [
		t(f.days === 1 ? 'Plan {days} day of {meals}' : 'Plan {days} days of {meals}', {
			days: f.days,
			meals: and(f.meals.map((m) => t(mealNames[m] ?? m)))
		}) +
			' ' +
			t(f.people === 1 ? 'for {people} person.' : 'for {people} people.', { people: f.people }),
		f.budget.trim()
			? t('Budget: {amount} {currency} in all.', { amount: f.budget.trim(), currency: f.currency })
			: t('Budget: none set.'),
		t('Dietary restrictions and allergies: {list}.', {
			list: f.restrictions.length
				? f.restrictions.map((r) => t(restrictionNames[r] ?? r)).join(', ')
				: t('none')
		}),
		t('At most {minutes} minutes of active cooking per meal.', { minutes: f.minutes })
	];
	if (f.other.trim()) parts.push(t('Other constraints: {other}', { other: f.other.trim() }));
	return parts.join(' ');
}
