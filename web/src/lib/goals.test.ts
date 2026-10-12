import { describe, expect, it } from 'vitest';
import { percentLabel } from './goals';

describe('percentLabel', () => {
	it('says when there is nothing to measure', () => {
		expect(percentLabel({ percent: null, method: 'none', done: 0, total: 0, summary: '' })).toBe(
			'No progress measure yet'
		);
		expect(percentLabel({ percent: 40, method: 'items', done: 2, total: 5, summary: '' })).toBe(
			'40%'
		);
	});
});
