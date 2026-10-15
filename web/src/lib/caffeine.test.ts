import { describe, expect, it } from 'vitest';
import { advice, drinkNames } from './caffeine';
import { fr } from './i18n.fr';

const day = (over: Partial<Parameters<typeof advice>[0]>) => ({
	total_mg: 95,
	limit_mg: 400,
	cutoff: '2026-10-15T14:12:00Z',
	at_bedtime_mg: 10,
	...over
});

describe('caffeine advice (EP-117)', () => {
	it('says when to stop, then that it is past, then that tonight is already too much', () => {
		expect(advice(day({}), new Date('2026-10-15T09:00:00Z')).key).toBe('before');
		expect(advice(day({}), new Date('2026-10-15T15:00:00Z')).key).toBe('past');
		expect(advice(day({ cutoff: null, at_bedtime_mg: 60 }), new Date()).key).toBe('none');
	});
	it('puts the daily limit first', () => {
		expect(advice(day({ total_mg: 420 }), new Date('2026-10-15T09:00:00Z')).key).toBe('over');
	});
	it('names every drink, in French too', () => {
		const names = Object.values(drinkNames);
		expect(names.filter((n) => !(n in fr))).toEqual([]);
	});
});
