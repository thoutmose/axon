import { describe, expect, it } from 'vitest';
import { verdict } from './calibration';

const cal = (reviews: number, remembered: number, target = 0.9) => ({
	reviews,
	remembered,
	target
});

describe('verdict', () => {
	it('waits for enough reviews to judge', () => {
		expect(verdict(cal(19, 0.5))).toBe('few');
	});
	it('says when recall is below, on, or above the target (5 points either way)', () => {
		expect(verdict(cal(200, 0.84))).toBe('below');
		expect(verdict(cal(200, 0.86))).toBe('on');
		expect(verdict(cal(200, 0.94))).toBe('on');
		expect(verdict(cal(200, 0.96))).toBe('above');
	});
});
