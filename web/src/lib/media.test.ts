import { describe, expect, it } from 'vitest';
import { coverHue } from './media';

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
