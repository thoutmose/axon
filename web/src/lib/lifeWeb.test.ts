import { describe, expect, it } from 'vitest';
import { average, level, score, total, weakest, type DayLog, type Targets } from './lifeWeb';

const targets: Targets = {
	kcal: 2000,
	protein_g: 50,
	water_ml: 1600,
	caffeine_mg: 400,
	tracksActivity: true,
	tracksCaffeine: true
};
const day: DayLog = {
	kcal: 2000,
	protein_g: 60,
	water_ml: 1600,
	sleep_hours: 8,
	active_minutes: 30,
	caffeine_mg: 100,
	mood: 5
};

describe('the life web', () => {
	it('gives full marks to a day on target', () => {
		expect(score(day, targets)).toEqual({
			food: 100,
			water: 100,
			sleep: 100,
			activity: 100,
			caffeine: 100,
			mood: 100
		});
	});
	it('scores each axis against its target', () => {
		const s = score(
			{
				...day,
				kcal: 2400,
				protein_g: 25,
				water_ml: 800,
				sleep_hours: 5,
				active_minutes: 15,
				caffeine_mg: 400,
				mood: 3
			},
			targets
		);
		expect(s).toEqual({ food: 55, water: 50, sleep: 50, activity: 50, caffeine: 60, mood: 50 });
		expect(score({ ...day, caffeine_mg: 700 }, targets).caffeine).toBe(0);
		expect(score({ ...day, sleep_hours: 10 }, targets).sleep).toBe(75);
	});
	it('says when nothing was logged instead of scoring zero', () => {
		const s = score(
			{ ...day, kcal: 0, water_ml: 0, sleep_hours: null, mood: null },
			{ ...targets, tracksActivity: false, tracksCaffeine: false }
		);
		expect(Object.values(s).every((v) => v == null)).toBe(true);
		expect(total(s)).toBeNull();
	});
	it('averages the axes it has, and names the weakest', () => {
		const s = { food: 80, water: null, sleep: 60, activity: 100, caffeine: null, mood: 40 };
		expect(total(s)).toBe(70);
		expect(weakest(s)).toBe('mood');
		expect(average([s, { ...s, water: 50, mood: 60 }])).toMatchObject({ water: 50, mood: 50 });
	});
	it('gives a level and what the next one needs', () => {
		expect(level(92)).toEqual({ reached: 'Platinum', next: null });
		expect(level(70)).toEqual({ reached: 'Silver', next: { name: 'Gold', needs: 10 } });
		expect(level(30)).toEqual({ reached: null, next: { name: 'Bronze', needs: 20 } });
	});
});
