import { describe, expect, it } from 'vitest';
import { emptyDay, fieldsOf, hasContent, scaleArt } from './journal';

describe('a journal day', () => {
	it('is worth saving with any one thing in it', () => {
		expect(hasContent(emptyDay())).toBe(false);
		expect(hasContent({ ...emptyDay(), mood: 3 })).toBe(true);
		expect(hasContent({ ...emptyDay(), gratitude: ['', ' Sun ', ''] })).toBe(true);
		expect(hasContent({ ...emptyDay(), note: '   ' })).toBe(false);
	});
});

describe('an entry written back', () => {
	it('carries only the fields the API takes', () => {
		const stored = {
			...emptyDay(),
			mood: 4,
			day: '2026-10-18',
			updated_at: '2026-10-18T09:00:00Z'
		};
		expect(Object.keys(fieldsOf(stored)).sort()).toEqual([
			'energy',
			'gratitude',
			'mood',
			'note',
			'sleep_hours',
			'stress'
		]);
		expect(fieldsOf(stored).mood).toBe(4);
	});
});

describe('the scales’ drawings', () => {
	it('smile more as the mood rises, and fill the battery with energy', () => {
		const mouth = (step: number) => Number(/Q12 ([\d.]+)/.exec(scaleArt('mood', step))![1]);
		expect(mouth(1)).toBeLessThan(mouth(3));
		expect(mouth(5)).toBeGreaterThan(mouth(3));
		const bars = (step: number) =>
			scaleArt('energy', step).match(/fill="currentColor"/g)?.length ?? 0;
		expect([1, 2, 3, 4, 5].map(bars)).toEqual([0, 1, 2, 3, 4]);
		expect(scaleArt('stress', 1)).not.toBe(scaleArt('stress', 5));
	});
});
