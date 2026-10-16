import { describe, expect, it } from 'vitest';
import { coverHue, left, stars } from './media';

describe('coverHue', () => {
	it('gives each title its own, stable colour', () => {
		expect(coverHue('Dune')).toBe(coverHue('Dune'));
		const hues = new Set(
			['Dune', 'Arrival', 'The Dispossessed', 'Severance', 'Atomic Habits'].map(coverHue)
		);
		expect(hues.size).toBeGreaterThanOrEqual(4);
		for (const h of hues) expect(h >= 0 && h < 360 && Number.isInteger(h)).toBe(true);
	});
});

describe('stars and what is left', () => {
	it('draws a rating out of ten as five stars', () => {
		expect(stars(10)).toBe('★★★★★');
		expect(stars(7)).toBe('★★★½☆');
		expect(stars(1)).toBe('½☆☆☆☆');
	});
	it('counts what is left, never below zero', () => {
		expect(left({ progress: 210, total: 611 })).toBe(401);
		expect(left({ progress: 700, total: 611 })).toBe(0);
		expect(left({ progress: 3, total: null })).toBeNull();
	});
});
