import { moonPosition, placeOf, sunPosition } from './sun';

// The sky's colours for the sun's altitude (EP-96): night under -18°,
// the three twilights, the golden hour, and day. Colours between two
// altitudes are blended, so the picture changes as smoothly as the sky.

export type Palette = {
	sky: [string, string, string, string]; // top to horizon
	glow: string;
	sun: string;
	hills: [string, string, string, string]; // far to near
	stars: number; // opacity
};

const keys: { at: number; p: Palette }[] = [
	{
		at: -18,
		p: {
			sky: ['#03050f', '#0a1230', '#152150', '#24356e'],
			glow: '#7f9cff',
			sun: '#e9eeff',
			hills: ['#253467', '#1a2650', '#111a3a', '#090f25'],
			stars: 1
		}
	},
	{
		at: -10,
		p: {
			sky: ['#0b1438', '#1d2a63', '#3c3f86', '#6a5a9c'],
			glow: '#8f86ff',
			sun: '#ffe2c4',
			hills: ['#3a3f78', '#2a2e60', '#1c2048', '#11142f'],
			stars: 0.6
		}
	},
	{
		at: -4,
		p: {
			sky: ['#1d1a4a', '#5b2d74', '#c4527a', '#ff9a62'],
			glow: '#ff8a5c',
			sun: '#ffd1a1',
			hills: ['#7a4a86', '#56356f', '#3a2453', '#211638'],
			stars: 0.15
		}
	},
	{
		at: 2,
		p: {
			sky: ['#2b3a7a', '#8a6fb8', '#f4a6a6', '#ffd8a8'],
			glow: '#ffc58a',
			sun: '#fff1d6',
			hills: ['#9b8ac4', '#7466a8', '#4f4a86', '#2f2d5c'],
			stars: 0
		}
	},
	{
		at: 10,
		p: {
			sky: ['#3567c4', '#6f9be0', '#f2c9a0', '#ffe2b8'],
			glow: '#ffd58a',
			sun: '#fff6dc',
			hills: ['#97a9cf', '#6a83b3', '#455f92', '#283f6b'],
			stars: 0
		}
	},
	{
		at: 25,
		p: {
			sky: ['#3d7bd9', '#6aa6ee', '#a9d0f7', '#e6f2fd'],
			glow: '#fff3c4',
			sun: '#fffdf2',
			hills: ['#8fb7d9', '#5f92c0', '#3c6e9e', '#244d75'],
			stars: 0
		}
	}
];

const hex = (c: string) => [1, 3, 5].map((i) => parseInt(c.slice(i, i + 2), 16));
export const mix = (a: string, b: string, t: number) =>
	'#' +
	hex(a)
		.map((x, i) =>
			Math.round(x + (hex(b)[i] - x) * t)
				.toString(16)
				.padStart(2, '0')
		)
		.join('');

/** The palette for a sun altitude, in degrees. */
export function palette(altitude: number): Palette {
	if (altitude <= keys[0].at) return keys[0].p;
	const j = keys.findIndex((k) => k.at >= altitude);
	if (j < 0) return keys[keys.length - 1].p;
	const [a, b] = [keys[j - 1], keys[j]];
	const t = (altitude - a.at) / (b.at - a.at);
	const each = <T extends string[]>(x: T, y: T) => x.map((c, i) => mix(c, y[i], t)) as T;
	return {
		sky: each(a.p.sky, b.p.sky),
		glow: mix(a.p.glow, b.p.glow, t),
		sun: mix(a.p.sun, b.p.sun, t),
		hills: each(a.p.hills, b.p.hills),
		stars: a.p.stars + (b.p.stars - a.p.stars) * t
	};
}

/** Light from civil dawn to civil dusk, dark otherwise. */
export const lightAt = (altitude: number) => altitude > -4;

export type Sky = {
	sun: { altitude: number; azimuth: number; hourAngle: number };
	moon: { altitude: number; azimuth: number; hourAngle: number; phase: number };
	palette: Palette;
	light: boolean;
};

/** The sky of a time zone at a moment. */
export function skyAt(date: Date, zone: string): Sky {
	const { lat, lon } = placeOf(zone);
	const sun = sunPosition(date, lat, lon);
	return {
		sun,
		moon: moonPosition(date, lat, lon),
		palette: palette(sun.altitude),
		light: lightAt(sun.altitude)
	};
}
