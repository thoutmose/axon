import { describe, expect, it } from 'vitest';
import { costs, daysBetween, expiringSoon, nextExam, type Cert } from './certs';

const c = (over: Partial<Cert>): Cert => ({
	id: Math.random().toString(),
	name: 'X',
	code: '',
	status: 'planned',
	exam_on: null,
	expires_on: null,
	cost_cents: null,
	currency: 'USD',
	...over
});

describe('certifications', () => {
	it('adds up what the plan costs and what was paid', () => {
		expect(
			costs([
				c({ cost_cents: 44500 }),
				c({ cost_cents: 30000, status: 'passed' }),
				c({ cost_cents: 9900, status: 'booked' }),
				c({ cost_cents: 10000, currency: 'EUR', status: 'failed' }),
				c({})
			])
		).toEqual([
			{ currency: 'USD', planned: 84400, paid: 39900 },
			{ currency: 'EUR', planned: 10000, paid: 10000 }
		]);
	});
	it('finds the next booked exam and what expires soon', () => {
		const items = [
			c({ status: 'booked', exam_on: '2026-11-20', code: 'CKA' }),
			c({ status: 'booked', exam_on: '2026-10-20', code: 'CCNA' }),
			c({ status: 'booked', exam_on: '2026-10-01', code: 'old' }),
			c({ status: 'passed', expires_on: '2026-12-01', code: 'AZ-104' }),
			c({ status: 'passed', expires_on: '2028-12-01', code: 'AWS' })
		];
		expect(nextExam(items, '2026-10-14')?.code).toBe('CCNA');
		expect(expiringSoon(items, '2026-10-14').map((x) => x.code)).toEqual(['AZ-104']);
		expect(daysBetween('2026-10-14', '2026-10-20')).toBe(6);
	});
});
