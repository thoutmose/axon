import { describe, expect, it } from 'vitest';
import { addDays, between, goalTemplates, milestonePlans } from './entityTemplates';

describe('entity templates', () => {
	it('spreads milestones between today and the target date', () => {
		expect(between('2026-01-01', '2026-01-11', 0.5)).toBe('2026-01-06');
		expect(between('2026-01-01', '2026-01-11', 1)).toBe('2026-01-11');
		expect(addDays('2026-12-30', 3)).toBe('2027-01-02');
	});
	it('keeps every share within the goal', () => {
		for (const g of goalTemplates) for (const [, s] of g.milestones) expect(s).toBeGreaterThan(0);
		for (const p of milestonePlans) for (const [, s] of p.steps) expect(s).toBeLessThanOrEqual(1);
	});
	it('only measures with a target that differs from the start', () => {
		for (const g of goalTemplates)
			if (g.measure) expect(g.measure.target).not.toBe(g.measure.start);
	});
});
