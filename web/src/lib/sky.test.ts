import { describe, expect, it } from 'vitest';
import { lightAt, palette, skyAt } from './sky';

describe('palette', () => {
	it('is the night under astronomical twilight and the day high up', () => {
		expect(palette(-40)).toEqual(palette(-18));
		expect(palette(-40).stars).toBe(1);
		expect(palette(60)).toEqual(palette(25));
		expect(palette(60).stars).toBe(0);
	});
	it('blends between two altitudes', () => {
		const p = palette(-14);
		expect(p.stars).toBeCloseTo(0.8, 5);
		expect(p.sky[0]).not.toBe(palette(-18).sky[0]);
		expect(p.sky[0]).toMatch(/^#[0-9a-f]{6}$/);
	});
});

describe('the theme that follows the sun', () => {
	it('is light from civil dawn to civil dusk', () => {
		expect(lightAt(-10)).toBe(false);
		expect(lightAt(-2)).toBe(true);
		expect(skyAt(new Date('2026-06-21T12:00:00Z'), 'Europe/Paris').light).toBe(true);
		expect(skyAt(new Date('2026-06-21T23:30:00Z'), 'Europe/Paris').light).toBe(false);
		// The same instant is night in Tokyo.
		expect(skyAt(new Date('2026-06-21T14:00:00Z'), 'Asia/Tokyo').light).toBe(false);
	});
});
