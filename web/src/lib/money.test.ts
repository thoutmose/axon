import { describe, expect, it } from 'vitest';
import { centsText, toCents } from './money';

describe('toCents', () => {
	it('reads the usual ways people type amounts', () => {
		expect(toCents('12')).toBe(1200);
		expect(toCents('12.5')).toBe(1250);
		expect(toCents('12,05')).toBe(1205);
		expect(toCents('1 234,56')).toBe(123456);
		expect(toCents('1,234.56')).toBe(123456);
		expect(toCents('1.234.567,89')).toBe(123456789);
		expect(toCents('0.1')).toBe(10);
	});
	it('never goes through floating point', () => {
		expect(toCents('0.29')).toBe(29); // 0.29 * 100 is 28.999… in floats
		expect(toCents('1.005')).toBe(100500); // three digits: a thousands separator
	});
	it('rejects what is not an amount', () => {
		for (const s of ['', 'abc', '-5', '1e3', '12.', '99999999999']) expect(toCents(s)).toBeNull();
	});
	it('round-trips through the editable text', () => {
		for (const c of [0, 1, 10, 99, 100, 123456]) expect(toCents(centsText(c))).toBe(c);
	});
});
