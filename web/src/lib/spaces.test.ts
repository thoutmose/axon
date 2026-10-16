import { describe, expect, it } from 'vitest';
import { spaceFor } from './spaces';

describe('spaceFor', () => {
	it('gives each area its scenery', () => {
		expect(spaceFor('/notes/123')).toBe('notebook');
		expect(spaceFor('/roadmaps')).toBe('library');
		expect(spaceFor('/resources/1')).toBe('library');
		expect(spaceFor('/projects/9')).toBe('blueprint');
		expect(spaceFor('/budget/reports')).toBe('ledger');
		expect(spaceFor('/cards/review')).toBe('library');
		expect(spaceFor('/focus')).toBe('desk');
		expect(spaceFor('/progress')).toBe('summit');
		expect(spaceFor('/collection')).toBe('summit');
		expect(spaceFor('/coffee')).toBe('garden');
	});
	it('gives the tech watch, its graph, its podcast, and the digest their own', () => {
		expect(spaceFor('/watch')).toBe('signal');
		expect(spaceFor('/watch/sources')).toBe('signal');
		expect(spaceFor('/watch/graph')).toBe('constellation');
		expect(spaceFor('/watch/podcast')).toBe('studio');
		expect(spaceFor('/digest')).toBe('dispatch');
		expect(spaceFor('/watchlist')).toBeNull();
	});
	it('leaves pages with their own scenery alone', () => {
		expect(spaceFor('/today')).toBeNull();
		expect(spaceFor('/')).toBeNull();
		expect(spaceFor('/account')).toBeNull();
		expect(spaceFor('/notesy')).toBeNull();
	});
});
