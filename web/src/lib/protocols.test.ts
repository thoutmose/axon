import { describe, expect, it } from 'vitest';
import { fr } from './i18n.fr';
import { evidenceNames, protocols } from './protocols';

// Huberman's protocols (EP-118): each one sourced, and in French.
describe('protocols', () => {
	it('each has a reason, a source over https, and a habit to start', () => {
		const ids = new Set(protocols.map((p) => p.id));
		expect(ids.size).toBe(protocols.length);
		for (const p of protocols) {
			expect(p.sources.length, p.id).toBeGreaterThan(0);
			for (const s of p.sources) expect(s.url, p.id).toMatch(/^https:\/\//);
			expect(p.habit.two_minute, p.id).not.toBe('');
		}
		// What no study tested says so.
		expect(protocols.find((p) => p.id === 'delay-caffeine')?.evidence).toBe('debated');
	});
	it('has French for everything shown', () => {
		const texts = [
			...Object.values(evidenceNames),
			...protocols.flatMap((p) =>
				[
					p.title,
					p.how,
					p.why,
					p.caution,
					p.habit.title,
					p.habit.identity,
					p.habit.stack_after,
					p.habit.intention_place,
					p.habit.cue,
					p.habit.craving,
					p.habit.two_minute
				].filter((s): s is string => !!s)
			)
		];
		expect([...new Set(texts.filter((s) => !(s in fr)))]).toEqual([]);
	});
});
