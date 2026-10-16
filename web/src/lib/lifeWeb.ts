/**
 * The life web (EP-131): a day across six axes, each scored from 0 to 100, or
 * null when nothing says how it went. Pure, so the scores can be tested.
 */

export const axes = ['food', 'water', 'sleep', 'activity', 'caffeine', 'mood'] as const;
export type Axis = (typeof axes)[number];
export type Scores = Record<Axis, number | null>;

/** What one day holds; null where nothing was logged. */
export type DayLog = {
	kcal: number;
	protein_g: number;
	water_ml: number;
	sleep_hours: number | null;
	active_minutes: number;
	caffeine_mg: number;
	mood: number | null;
};

/**
 * The targets: the person's own needs, else the EU reference intakes for an
 * adult (2,000 kcal, 50 g of protein: Regulation 1169/2011, Annex XIII) and
 * EFSA's drinks for a woman (80 % of 2.0 L).
 */
export type Targets = {
	kcal: number;
	protein_g: number;
	water_ml: number;
	caffeine_mg: number;
	/** Whether workouts and caffeine are logged at all: if not, their axes say so. */
	tracksActivity: boolean;
	tracksCaffeine: boolean;
};

const clamp = (v: number) => Math.round(Math.max(0, Math.min(100, v)));

export function score(d: DayLog, t: Targets): Scores {
	const energy = 100 - Math.abs(d.kcal / t.kcal - 1) * 200; // within 10 %: 80 and more
	const protein = Math.min(1, d.protein_g / t.protein_g) * 100;
	const h = d.sleep_hours;
	const half = t.caffeine_mg / 2;
	return {
		food: d.kcal > 0 ? clamp((clamp(energy) + protein) / 2) : null,
		water: d.water_ml > 0 ? clamp((d.water_ml / t.water_ml) * 100) : null,
		// 7 to 9 hours for adults (American Academy of Sleep Medicine, 2015).
		sleep: h == null ? null : clamp(100 - Math.max(7 - h, h - 9, 0) * 25),
		// 150 minutes a week (WHO, 2020), about 30 on five days.
		activity: t.tracksActivity ? clamp((d.active_minutes / 30) * 100) : null,
		// Full marks to half the limit, 60 at the limit, nothing at one and a half.
		caffeine: t.tracksCaffeine
			? clamp(
					d.caffeine_mg <= half
						? 100
						: d.caffeine_mg <= t.caffeine_mg
							? 100 - ((d.caffeine_mg - half) / half) * 40
							: 60 - ((d.caffeine_mg - t.caffeine_mg) / half) * 60
				)
			: null,
		mood: d.mood == null ? null : clamp(((d.mood - 1) / 4) * 100)
	};
}

/** The day's score: the mean of the axes that have one. */
export function total(s: Scores): number | null {
	const v = axes.map((a) => s[a]).filter((x): x is number => x != null);
	return v.length ? Math.round(v.reduce((a, b) => a + b, 0) / v.length) : null;
}

export const levels = [
	{ name: 'Platinum', min: 90 },
	{ name: 'Gold', min: 80 },
	{ name: 'Silver', min: 65 },
	{ name: 'Bronze', min: 50 }
] as const;

/** The level a score reaches, and the next one with the points it needs. */
export function level(points: number | null) {
	const i = levels.findIndex((l) => (points ?? 0) >= l.min);
	const reached = i < 0 ? null : levels[i].name;
	const next = i === 0 ? null : levels[i < 0 ? levels.length - 1 : i - 1];
	return { reached, next: next && { name: next.name, needs: next.min - (points ?? 0) } };
}

/** Each axis's mean over several days, null where none had it. */
export function average(days: Scores[]): Scores {
	const out = {} as Scores;
	for (const a of axes) {
		const v = days.map((d) => d[a]).filter((x): x is number => x != null);
		out[a] = v.length ? Math.round(v.reduce((x, y) => x + y, 0) / v.length) : null;
	}
	return out;
}

/** The axis with the lowest score: the one to raise next. */
export function weakest(s: Scores): Axis | null {
	let low: Axis | null = null;
	for (const a of axes) if (s[a] != null && (low == null || s[a]! < s[low]!)) low = a;
	return low;
}
