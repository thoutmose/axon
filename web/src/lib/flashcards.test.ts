import { describe, expect, it } from 'vitest';
import { interval, sample } from './flashcards';

describe('interval', () => {
	const t = new Date('2026-01-01T00:00:00Z');
	const after = (ms: number) => new Date(t.getTime() + ms);
	it('reads like Anki', () => {
		expect(interval(t, after(30_000))).toBe('now');
		expect(interval(t, after(10 * 60_000))).toBe('10 min');
		expect(interval(t, after(5 * 3_600_000))).toBe('5 h');
		expect(interval(t, after(86_400_000))).toBe('1 day');
		expect(interval(t, after(9 * 86_400_000))).toBe('9 days');
		expect(interval(t, after(70 * 86_400_000))).toBe('2 mo');
		expect(interval(t, after(550 * 86_400_000))).toBe('1.5 yr');
	});
});

describe('sample', () => {
	it('takes n distinct items', () => {
		const s = sample([1, 2, 3, 4, 5, 6], 4);
		expect(s.length).toBe(4);
		expect(new Set(s).size).toBe(4);
		expect(sample([1, 2], 5).length).toBe(2);
	});
});
