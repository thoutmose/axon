import { describe, expect, it } from 'vitest';
import { glasses, litres } from './water';

describe('water', () => {
	it('shows the target as glasses of 250 ml, filled as they are drunk', () => {
		expect(glasses(750, 1600)).toEqual({ filled: 3, total: 7 });
		expect(glasses(2000, 1600)).toEqual({ filled: 8, total: 8 }); // past the target, all filled
		expect(glasses(0, null)).toEqual({ filled: 0, total: 8 }); // no target: eight glasses
		expect(glasses(5000, null).total).toBe(16); // never more than sixteen
	});
	it('writes litres in the interface’s language', () => {
		expect(litres(1250, 'en')).toBe('1.25');
		expect(litres(1250, 'fr')).toBe('1,25');
		expect(litres(2000, 'en')).toBe('2');
	});
});
