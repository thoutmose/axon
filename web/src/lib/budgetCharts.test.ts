import { describe, expect, it } from 'vitest';
import { arc, colourOrder, slices, type CategoryTotal } from './budgetCharts';

const c = (name: string, cents: number, kind: CategoryTotal['kind'] = 'expense') => ({
	name,
	kind,
	cents
});

describe('budget pies', () => {
	it('keep a category’s colour from the year, whatever the period', () => {
		const order = colourOrder(
			[c('Rent', 900), c('Food', 400), c('Salary', 3000, 'income')],
			'expense'
		);
		expect(order).toEqual(['Rent', 'Food']);
		// This month food cost more than rent: the order changes, the colours do not.
		const s = slices([c('Rent', 100), c('Food', 300)], 'expense', order);
		expect(s.map((x) => [x.label, x.color])).toEqual([
			['Food', 'var(--series-2)'],
			['Rent', 'var(--series-1)']
		]);
		expect(s[0].share).toBe(0.75);
	});
	it('fold what has no hue into Other, last', () => {
		const names = 'abcdefghi'.split('');
		const order = names;
		const s = slices(
			names.map((n, i) => c(n, 100 - i)),
			'expense',
			order,
			'Autres'
		);
		expect(s).toHaveLength(8);
		expect(s[7]).toMatchObject({ label: 'Autres', cents: 93 + 92, color: 'var(--muted)' });
	});
	it('leave out empty categories and the other kind', () => {
		expect(slices([c('Rent', 0), c('Salary', 5, 'income')], 'expense', ['Rent'])).toEqual([]);
	});
	it('draw a slice as a closed ring segment', () => {
		expect(arc(0, 0.25, 10, 6, 10)).toMatch(
			/^M10\.00 0\.00A10 10 0 0 1 20\.00 10\.00L16\.00 10\.00A6 6 0 0 0 10\.00 4\.00Z$/
		);
	});
});
