import { describe, expect, it } from 'vitest';
import { spaceFor } from './spaces';

describe('spaceFor', () => {
	it('gives each area its scenery', () => {
		expect(spaceFor('/notes/123')).toBe('notebook');
		expect(spaceFor('/roadmaps')).toBe('library');
		expect(spaceFor('/resources/1')).toBe('library');
		expect(spaceFor('/projects/9')).toBe('blueprint');
		expect(spaceFor('/budget/reports')).toBe('ledger');
	});
	it('leaves pages with their own scenery alone', () => {
		expect(spaceFor('/today')).toBeNull();
		expect(spaceFor('/')).toBeNull();
		expect(spaceFor('/account')).toBeNull();
		expect(spaceFor('/notesy')).toBeNull();
	});
});
