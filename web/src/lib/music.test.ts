import { describe, expect, it } from 'vitest';
import { bar, hz, seeded, styleFor, styles, type Style } from './music';

const F_MAJOR = [5, 7, 9, 10, 0, 2, 4];
const C_MAJOR = [0, 2, 4, 5, 7, 9, 11];
const drums = ['kick', 'snare', 'hat'];

describe('focus music', () => {
	it('matches the music to what the session is about', () => {
		expect(styleFor('task')).toBe('lofi');
		expect(styleFor('project')).toBe('lofi');
		expect(styleFor('note')).toBe('ambient');
		expect(styleFor('resource')).toBe('ambient');
		expect(styleFor('skill')).toBe('piano');
		expect(styleFor('card')).toBe('piano');
		expect(styleFor(undefined)).toBe('lofi');
	});

	it('keeps every note inside its bar', () => {
		for (const style of Object.keys(styles) as Style[]) {
			for (let i = 0; i < 16; i++) {
				for (const n of bar(style, i, seeded(i))) {
					expect(n.at).toBeGreaterThanOrEqual(0);
					expect(n.at).toBeLessThan(4);
					expect(n.length).toBeGreaterThan(0);
					expect(n.velocity).toBeGreaterThan(0);
					expect(n.velocity).toBeLessThanOrEqual(1);
				}
			}
		}
	});

	it('stays in key, so nothing jars', () => {
		const key: Record<Style, number[]> = { lofi: F_MAJOR, ambient: C_MAJOR, piano: C_MAJOR };
		for (const style of Object.keys(styles) as Style[]) {
			for (let i = 0; i < 16; i++) {
				for (const n of bar(style, i, seeded(i)).filter((n) => !drums.includes(n.voice))) {
					expect(key[style], `${style} ${n.midi}`).toContain(n.midi % 12);
				}
			}
		}
	});

	it('gives lo-fi a beat and leaves ambient and piano without drums', () => {
		const voices = (style: Style) => new Set(bar(style, 0, seeded(1)).map((n) => n.voice));
		expect(voices('lofi')).toContain('kick');
		expect(voices('lofi')).toContain('snare');
		for (const style of ['ambient', 'piano'] as const)
			for (const d of drums) expect(voices(style)).not.toContain(d);
	});

	it('follows the chord progression bar by bar, and varies', () => {
		const roots = (i: number) => bar('lofi', i, seeded(9)).find((n) => n.voice === 'bass')?.midi;
		expect(roots(0)).not.toBe(roots(1));
		expect(roots(0)).toBe(roots(styles.lofi.chords.length));
		const a = JSON.stringify(bar('lofi', 0, seeded(1)));
		expect(JSON.stringify(bar('lofi', 0, seeded(1)))).toBe(a);
		expect(JSON.stringify(bar('lofi', 0, seeded(2)))).not.toBe(a);
	});

	it('keeps a calm tempo', () => {
		for (const s of Object.values(styles)) {
			expect(s.bpm).toBeGreaterThanOrEqual(50);
			expect(s.bpm).toBeLessThanOrEqual(80);
		}
	});

	it('tunes A4 to 440 Hz', () => {
		expect(hz(69)).toBe(440);
		expect(hz(57)).toBe(220);
	});
});
