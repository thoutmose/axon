import { describe, expect, it } from 'vitest';
import { moonPosition, placeOf, sunPosition } from './sun';

const PARIS = { lat: 48.8566, lon: 2.3522 };

describe('sunPosition', () => {
	it('is high at solar noon in Paris on the June solstice', () => {
		// Solar noon in Paris on 2026-06-21 is about 11:52 UTC, at 64.6°.
		const { altitude, azimuth } = sunPosition(
			new Date('2026-06-21T11:52:00Z'),
			PARIS.lat,
			PARIS.lon
		);
		expect(altitude).toBeGreaterThan(64);
		expect(altitude).toBeLessThan(65.3);
		expect(azimuth).toBeGreaterThan(170);
		expect(azimuth).toBeLessThan(190);
	});

	it('is below the horizon at midnight, and rises in the east', () => {
		expect(
			sunPosition(new Date('2026-06-21T00:00:00Z'), PARIS.lat, PARIS.lon).altitude
		).toBeLessThan(-10);
		// Sunrise in Paris on 2026-06-21 is about 03:47 UTC.
		const rise = sunPosition(new Date('2026-06-21T03:47:00Z'), PARIS.lat, PARIS.lon);
		expect(Math.abs(rise.altitude + 0.8)).toBeLessThan(1.2);
		expect(rise.azimuth).toBeGreaterThan(45);
		expect(rise.azimuth).toBeLessThan(60);
	});

	it('is low at noon in a southern winter, and overhead at the equator on the equinox', () => {
		// Sydney, 2026-06-21, solar noon about 01:57 UTC: 32.6°.
		const sydney = sunPosition(new Date('2026-06-21T01:57:00Z'), -33.87, 151.21);
		expect(sydney.altitude).toBeGreaterThan(31.5);
		expect(sydney.altitude).toBeLessThan(33.5);
		// 0°, 0° on 2026-03-20 around 12:07 UTC.
		expect(sunPosition(new Date('2026-03-20T12:07:00Z'), 0, 0).altitude).toBeGreaterThan(88.5);
	});
});

describe('moonPosition', () => {
	it('is full on 26 October 2026 and new on 10 October 2026', () => {
		expect(moonPosition(new Date('2026-10-26T04:00:00Z'), PARIS.lat, PARIS.lon).phase).toBeCloseTo(
			0.5,
			1
		);
		const nw = moonPosition(new Date('2026-10-10T16:00:00Z'), PARIS.lat, PARIS.lon).phase;
		expect(Math.min(nw, 1 - nw)).toBeLessThan(0.03);
	});

	it('rises as the sun sets when full', () => {
		// A full moon is opposite the sun: up at midnight, down at noon.
		expect(
			moonPosition(new Date('2026-10-26T00:00:00Z'), PARIS.lat, PARIS.lon).altitude
		).toBeGreaterThan(20);
		expect(
			moonPosition(new Date('2026-10-26T12:00:00Z'), PARIS.lat, PARIS.lon).altitude
		).toBeLessThan(0);
	});
});

describe('placeOf', () => {
	it('places a time zone at its city, and others by their offset', () => {
		expect(placeOf('Europe/Paris')).toEqual({ lat: 48.86, lon: 2.35 });
		const p = placeOf('Etc/GMT-3'); // UTC+3
		expect(p.lon).toBeCloseTo(45, 0);
		expect(placeOf('Australia/Brisbane').lat).toBeLessThan(0);
	});
});
