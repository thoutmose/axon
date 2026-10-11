import { describe, expect, it } from 'vitest';
import { chord, goTo } from './shortcuts';

describe('shortcuts', () => {
	it('maps g chords to pages, each key once', () => {
		expect(chord('t')).toBe('/tasks');
		expect(chord('s')).toBe('/account/preferences');
		expect(chord('z')).toBeUndefined();
		const keys = goTo.map((s) => s.keys);
		expect(new Set(keys).size).toBe(keys.length);
	});
});
