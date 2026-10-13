// Where the sun and the moon are (EP-96), for the sky that follows them.
// Low-precision formulas from the Astronomical Almanac (about 0.01° for
// the sun, a few degrees for the moon, which is plenty for a picture).
// The place comes from the time zone, never from the device's location.

const rad = Math.PI / 180;
const deg = 180 / Math.PI;

/** Days since J2000.0 (2000-01-01 12:00 UTC). */
const days = (date: Date) => date.getTime() / 86_400_000 - 10_957.5;

const wrap = (x: number) => ((x % 360) + 360) % 360;

/** Altitude above the horizon and azimuth from north, in degrees. */
export type Position = { altitude: number; azimuth: number; hourAngle: number };

function horizontal(d: number, ra: number, dec: number, lat: number, lon: number): Position {
	const lst = wrap(280.46061837 + 360.98564736629 * d + lon);
	const h = wrap(lst - ra + 180) - 180; // -180 (morning) to 180 (evening)
	const [hr, dr, lr] = [h * rad, dec * rad, lat * rad];
	const alt = Math.asin(Math.sin(lr) * Math.sin(dr) + Math.cos(lr) * Math.cos(dr) * Math.cos(hr));
	const az = Math.atan2(Math.sin(hr), Math.cos(hr) * Math.sin(lr) - Math.tan(dr) * Math.cos(lr));
	return { altitude: alt * deg, azimuth: wrap(az * deg + 180), hourAngle: h };
}

/** The sun's ecliptic longitude and the obliquity, in degrees. */
function sunLongitude(d: number): { lon: number; obliquity: number } {
	const g = wrap(357.529 + 0.98560028 * d) * rad;
	const q = 280.459 + 0.98564736 * d;
	return {
		lon: wrap(q + 1.915 * Math.sin(g) + 0.02 * Math.sin(2 * g)),
		obliquity: 23.439 - 0.00000036 * d
	};
}

/** Right ascension and declination of a point on the ecliptic. */
function equatorial(lon: number, obliquity: number): { ra: number; dec: number } {
	const [l, e] = [lon * rad, obliquity * rad];
	return {
		ra: wrap(Math.atan2(Math.cos(e) * Math.sin(l), Math.cos(l)) * deg),
		dec: Math.asin(Math.sin(e) * Math.sin(l)) * deg
	};
}

export function sunPosition(date: Date, lat: number, lon: number): Position {
	const d = days(date);
	const s = sunLongitude(d);
	const { ra, dec } = equatorial(s.lon, s.obliquity);
	return horizontal(d, ra, dec, lat, lon);
}

const synodic = 29.530588853;
const newMoon = Date.UTC(2000, 0, 6, 18, 14); // a known new moon

/** The moon: where it is, and its phase (0 new, 0.5 full). */
export function moonPosition(date: Date, lat: number, lon: number): Position & { phase: number } {
	const phase = ((((date.getTime() - newMoon) / 86_400_000 / synodic) % 1) + 1) % 1;
	const d = days(date);
	const s = sunLongitude(d);
	// The moon runs ahead of the sun by its phase, near the ecliptic.
	const { ra, dec } = equatorial(wrap(s.lon + phase * 360), s.obliquity);
	return { ...horizontal(d, ra, dec, lat, lon), phase };
}

// The main city of common time zones; others are placed by their offset.
const cities: Record<string, [number, number]> = {
	'Europe/Paris': [48.86, 2.35],
	'Europe/London': [51.51, -0.13],
	'Europe/Dublin': [53.35, -6.26],
	'Europe/Lisbon': [38.72, -9.14],
	'Europe/Madrid': [40.42, -3.7],
	'Europe/Brussels': [50.85, 4.35],
	'Europe/Amsterdam': [52.37, 4.9],
	'Europe/Luxembourg': [49.61, 6.13],
	'Europe/Zurich': [47.38, 8.54],
	'Europe/Berlin': [52.52, 13.4],
	'Europe/Rome': [41.9, 12.5],
	'Europe/Vienna': [48.21, 16.37],
	'Europe/Prague': [50.08, 14.44],
	'Europe/Warsaw': [52.23, 21.01],
	'Europe/Stockholm': [59.33, 18.07],
	'Europe/Oslo': [59.91, 10.75],
	'Europe/Copenhagen': [55.68, 12.57],
	'Europe/Helsinki': [60.17, 24.94],
	'Europe/Athens': [37.98, 23.73],
	'Europe/Istanbul': [41.01, 28.98],
	'Europe/Kyiv': [50.45, 30.52],
	'Europe/Moscow': [55.76, 37.62],
	'Africa/Casablanca': [33.57, -7.59],
	'Africa/Algiers': [36.75, 3.06],
	'Africa/Tunis': [36.81, 10.18],
	'Africa/Cairo': [30.04, 31.24],
	'Africa/Lagos': [6.52, 3.38],
	'Africa/Dakar': [14.72, -17.47],
	'Africa/Abidjan': [5.36, -4.01],
	'Africa/Nairobi': [-1.29, 36.82],
	'Africa/Johannesburg': [-26.2, 28.05],
	'Indian/Reunion': [-20.88, 55.45],
	'Asia/Dubai': [25.2, 55.27],
	'Asia/Tehran': [35.69, 51.39],
	'Asia/Karachi': [24.86, 67.0],
	'Asia/Kolkata': [22.57, 88.36],
	'Asia/Dhaka': [23.81, 90.41],
	'Asia/Bangkok': [13.76, 100.5],
	'Asia/Jakarta': [-6.21, 106.85],
	'Asia/Singapore': [1.35, 103.82],
	'Asia/Ho_Chi_Minh': [10.82, 106.63],
	'Asia/Hong_Kong': [22.32, 114.17],
	'Asia/Shanghai': [31.23, 121.47],
	'Asia/Taipei': [25.03, 121.57],
	'Asia/Manila': [14.6, 120.98],
	'Asia/Seoul': [37.57, 126.98],
	'Asia/Tokyo': [35.68, 139.69],
	'Australia/Perth': [-31.95, 115.86],
	'Australia/Adelaide': [-34.93, 138.6],
	'Australia/Brisbane': [-27.47, 153.03],
	'Australia/Sydney': [-33.87, 151.21],
	'Australia/Melbourne': [-37.81, 144.96],
	'Pacific/Auckland': [-36.85, 174.76],
	'Pacific/Noumea': [-22.27, 166.46],
	'Pacific/Tahiti': [-17.53, -149.57],
	'Pacific/Honolulu': [21.31, -157.86],
	'America/Anchorage': [61.22, -149.9],
	'America/Los_Angeles': [34.05, -118.24],
	'America/Vancouver': [49.28, -123.12],
	'America/Denver': [39.74, -104.99],
	'America/Phoenix': [33.45, -112.07],
	'America/Chicago': [41.88, -87.63],
	'America/Mexico_City': [19.43, -99.13],
	'America/New_York': [40.71, -74.01],
	'America/Toronto': [43.65, -79.38],
	'America/Montreal': [45.5, -73.57],
	'America/Halifax': [44.65, -63.57],
	'America/Guadeloupe': [16.24, -61.53],
	'America/Martinique': [14.6, -61.07],
	'America/Cayenne': [4.92, -52.31],
	'America/Bogota': [4.71, -74.07],
	'America/Lima': [-12.05, -77.04],
	'America/Caracas': [10.48, -66.9],
	'America/Santiago': [-33.45, -70.67],
	'America/Argentina/Buenos_Aires': [-34.6, -58.38],
	'America/Sao_Paulo': [-23.55, -46.63],
	UTC: [51.48, 0]
};

/** Where a time zone's sky is: its main city, or a place from its offset. */
export function placeOf(zone: string): { lat: number; lon: number } {
	const c = cities[zone];
	if (c) return { lat: c[0], lon: c[1] };
	let minutes = 0;
	try {
		const name = new Intl.DateTimeFormat('en-US', { timeZone: zone, timeZoneName: 'longOffset' })
			.formatToParts(new Date())
			.find((p) => p.type === 'timeZoneName')?.value;
		const m = name?.match(/GMT([+-])(\d{2}):?(\d{2})?/);
		if (m) minutes = (m[1] === '-' ? -1 : 1) * (Number(m[2]) * 60 + Number(m[3] ?? 0));
	} catch {
		// An unknown zone: Greenwich.
	}
	const south =
		/^(Australia|Antarctica)\/|^America\/(Argentina|Sao_Paulo|Santiago|Montevideo|Asuncion|La_Paz)|^Africa\/(Johannesburg|Maputo|Harare|Windhoek|Gaborone)|^Pacific\/(Auckland|Fiji|Chatham)/;
	return { lat: south.test(zone) ? -30 : 45, lon: minutes / 4 };
}
