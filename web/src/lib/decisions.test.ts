import { describe, expect, it } from 'vitest';
import { fr } from './i18n.fr';
import { addMonths, confidence, confidences, plot, verdict, verdicts } from './decisions';

describe('decision calibration (EP-114)', () => {
	it('reads a Brier score against always saying 50%', () => {
		expect(verdict(0.05)).toBe('sharp');
		expect(verdict(0.18)).toBe('better');
		expect(verdict(0.25)).toBe('coin');
		expect(verdict(0.4)).toBe('worse');
	});
	it('says over- or under-confident only with enough reviews and a real gap', () => {
		expect(confidence({ count: 4, predicted: 0.9, happened: 0.5 })).toBe(null);
		expect(confidence({ count: 10, predicted: 0.85, happened: 0.6 })).toBe('over');
		expect(confidence({ count: 10, predicted: 0.6, happened: 0.8 })).toBe('under');
		expect(confidence({ count: 10, predicted: 0.75, happened: 0.7 })).toBe('calibrated');
	});
	it('places each bucket by chance given (x) and share that happened (y)', () => {
		const [p] = plot([{ from: 0.8, count: 4, predicted: 0.8, happened: 0.5 }], 200, 20);
		expect(p).toMatchObject({ x: 20 + 0.8 * 160, y: 20 + 0.5 * 160 });
		expect(p.r).toBeGreaterThanOrEqual(5);
	});
	it('has French for the verdicts, which are translated by name', () => {
		const texts = [...Object.values(verdicts), ...Object.values(confidences)];
		expect(texts.filter((s) => !(s in fr))).toEqual([]);
	});
	it('adds months to a day, clamping to the month’s end', () => {
		expect(addMonths('2026-10-15', 3)).toBe('2027-01-15');
		expect(addMonths('2026-01-31', 1)).toBe('2026-02-28');
	});
});
