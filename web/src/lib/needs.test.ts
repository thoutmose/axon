import { describe, expect, it } from 'vitest';
import { progress } from './needs';

describe('progress towards a daily need', () => {
	it('is the share eaten, capped for the bar, and says when it is over', () => {
		expect(progress(50, 200)).toEqual({ share: 0.25, over: false });
		expect(progress(250, 200)).toEqual({ share: 1, over: true });
		expect(progress(0, 0)).toEqual({ share: 0, over: false });
	});
});
