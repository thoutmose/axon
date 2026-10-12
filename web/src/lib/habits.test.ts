import { describe, expect, it } from 'vitest';
import { isoWeekday, lastDays, scheduleLabel } from './habits';

describe('scheduleLabel', () => {
	it('names common schedules', () => {
		expect(scheduleLabel([1, 2, 3, 4, 5, 6, 7])).toBe('Every day');
		expect(scheduleLabel([5, 4, 3, 2, 1])).toBe('Weekdays');
		expect(scheduleLabel([1, 3, 5])).toBe('Mon, Wed, Fri');
	});
});

describe('lastDays', () => {
	it('ends today and crosses month boundaries', () => {
		expect(lastDays('2026-03-02', 3)).toEqual(['2026-02-28', '2026-03-01', '2026-03-02']);
	});
});

describe('isoWeekday', () => {
	it('counts Monday as 1 and Sunday as 7', () => {
		expect(isoWeekday('2026-10-12')).toBe(1);
		expect(isoWeekday('2026-10-11')).toBe(7);
	});
});

import { chains, plan, type Habit } from './habits';

const h = (id: string, title: string, stack_after = '', extra: Partial<Habit> = {}) =>
	({
		id,
		title,
		stack_after,
		kind: 'build',
		intention_time: '',
		intention_place: '',
		...extra
	}) as Habit;

describe('plan', () => {
	it('writes habit stacking and implementation intentions in Clear’s formulas', () => {
		expect(plan(h('1', 'Read one page', 'After I pour my coffee'))).toEqual([
			'After I pour my coffee, I will read one page.'
		]);
		expect(
			plan(h('1', 'Meditate', '', { intention_time: '07:00', intention_place: 'in the bedroom' }))
		).toEqual(['I will meditate at 07:00 in the bedroom.']);
		expect(plan(h('1', 'Stretch', '', { intention_place: 'living room' }))).toEqual([
			'I will stretch in living room.'
		]);
		expect(plan(h('1', 'Scroll in bed', '', { kind: 'break' }))).toEqual([
			'I am stopping: scroll in bed.'
		]);
	});
});

describe('chains', () => {
	it('links habits stacked after other habits, first habit first', () => {
		const list = [
			h('a', 'Pour my coffee'),
			h('b', 'Read one page', 'After I pour my coffee'),
			h('c', 'Write three lines', 'After I read one page'),
			h('d', 'Run')
		];
		expect(chains(list).map((c) => c.map((x) => x.id))).toEqual([['a', 'b', 'c']]);
	});
	it('survives a cycle', () => {
		const list = [h('a', 'A', 'After B'), h('b', 'B', 'After A')];
		expect(chains(list)).toEqual([]);
	});
});
