import { describe, expect, it } from 'vitest';
import { noise } from './sounds';

describe('noise', () => {
	it('stays within range for every kind', () => {
		for (const kind of ['white', 'pink', 'brown', 'rain'] as const) {
			const s = noise(kind, 20_000);
			expect(s.length).toBe(20_000);
			expect(Math.max(...s)).toBeLessThanOrEqual(1);
			expect(Math.min(...s)).toBeGreaterThanOrEqual(-1);
			expect(s.some((x) => x !== 0)).toBe(true);
		}
	});
	it('brown noise changes slowly compared with white noise', () => {
		const step = (s: Float32Array) =>
			s.slice(1).reduce((a, x, i) => a + Math.abs(x - s[i]), 0) / s.length;
		expect(step(noise('brown', 10_000))).toBeLessThan(step(noise('white', 10_000)) / 5);
	});
});
