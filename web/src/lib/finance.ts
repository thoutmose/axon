import { api, type Schemas } from './api';

export type Account = Schemas['BudgetAccount'];
export type Category = Schemas['BudgetCategory'];
export type Transaction = Schemas['BudgetTransaction'];
export type Recurring = Schemas['BudgetRecurring'];
export type Rule = Schemas['BudgetRule'];

export const roleNames: Record<string, string> = {
	checking: 'Checking',
	savings: 'Savings',
	cash: 'Cash',
	credit_card: 'Credit card',
	loan: 'Loan',
	investment: 'Investment',
	other: 'Other'
};

export async function loadBasics(): Promise<{
	accounts: Account[];
	categories: Category[];
	currency: string;
}> {
	const [a, c, m] = await Promise.all([
		api<{ items: Account[] }>('/v1/budget/accounts'),
		api<{ items: Category[] }>('/v1/budget/categories'),
		api<{ currency: string }>('/v1/budget')
	]);
	return { accounts: a.items, categories: c.items, currency: m.currency };
}

/** "#food travel" → ["food", "travel"]. */
export function parseTags(text: string): string[] {
	return text
		.split(/[\s,]+/)
		.map((t) => t.replace(/^#/, '').toLowerCase())
		.filter(Boolean);
}

export const frequencies: { label: string; rule: string }[] = [
	{ label: 'Every month', rule: 'FREQ=MONTHLY' },
	{ label: 'Every week', rule: 'FREQ=WEEKLY' },
	{ label: 'Every two weeks', rule: 'FREQ=WEEKLY;INTERVAL=2' },
	{ label: 'Every three months', rule: 'FREQ=MONTHLY;INTERVAL=3' },
	{ label: 'Every year', rule: 'FREQ=YEARLY' },
	{ label: 'Every day', rule: 'FREQ=DAILY' }
];
