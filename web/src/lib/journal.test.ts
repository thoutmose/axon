import { describe, expect, it } from 'vitest';
import { emptyDay, hasContent, scaleArt } from './journal';

describe('a journal day', () => {
	it('is worth saving with any one thing in it', () => {
		expect(hasContent(emptyDay())).toBe(false);
		expect(hasContent({ ...emptyDay(), mood: 3 })).toBe(true);
		expect(hasContent({ ...emptyDay(), gratitude: ['', ' Sun ', ''] })).toBe(true);
		expect(hasContent({ ...emptyDay(), note: '   ' })).toBe(false);
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
