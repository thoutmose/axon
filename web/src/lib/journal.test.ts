import { describe, expect, it } from 'vitest';
import { emptyDay, hasContent } from './journal';

describe('a journal day', () => {
	it('is worth saving with any one thing in it', () => {
		expect(hasContent(emptyDay())).toBe(false);
		expect(hasContent({ ...emptyDay(), mood: 3 })).toBe(true);
		expect(hasContent({ ...emptyDay(), gratitude: ['', ' Sun ', ''] })).toBe(true);
		expect(hasContent({ ...emptyDay(), note: '   ' })).toBe(false);
	});
});
