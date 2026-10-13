import { describe, expect, it } from 'vitest';
import { fitSize } from './photo';

describe('a photo made small enough to send', () => {
	it('keeps the shape and never enlarges', () => {
		expect(fitSize(4032, 3024, 1568)).toEqual({ width: 1568, height: 1176 });
		expect(fitSize(3024, 4032, 1568)).toEqual({ width: 1176, height: 1568 });
		expect(fitSize(800, 600, 1568)).toEqual({ width: 800, height: 600 });
	});
});
