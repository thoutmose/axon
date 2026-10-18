import { describe, expect, it } from 'vitest';
import { ruleLabel } from './finance';

describe('ruleLabel', () => {
	it('says a rule in words', () => {
		expect(ruleLabel('FREQ=MONTHLY')).toBe('Every month');
		expect(ruleLabel('FREQ=MONTHLY;BYMONTHDAY=2')).toBe('Every month, on day 2');
		expect(ruleLabel('FREQ=WEEKLY;INTERVAL=2')).toBe('Every two weeks');
		expect(ruleLabel('FREQ=HOURLY')).toBe('FREQ=HOURLY');
	});
});
