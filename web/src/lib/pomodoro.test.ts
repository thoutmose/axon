import { describe, expect, it } from 'vitest';
import {
	advance,
	clock,
	defaults,
	finished,
	newTimer,
	pause,
	phases,
	remaining,
	resume,
	rounds,
	skip,
	totalWorkMinutes,
	workedMinutes
} from './pomodoro';

const min = 60_000;

describe('phases', () => {
	it('defaults to 25/5/25/5/25/15, one session', () => {
		expect(phases(defaults).map((p) => `${p.kind}${p.minutes}`)).toEqual([
			'work25',
			'short5',
			'work25',
			'short5',
			'work25',
			'long15'
		]);
		expect(totalWorkMinutes(phases(defaults))).toBe(75);
	});
	it('repeats for more sessions and other lengths', () => {
		const p = phases({ ...defaults, sessions: 2, work_minutes: 50, short_break_minutes: 10 });
		expect(p.length).toBe(12);
		expect(p[0]).toEqual({ kind: 'work', minutes: 50 });
		expect(p[1]).toEqual({ kind: 'short', minutes: 10 });
		expect(p[5].kind).toBe('long');
		expect(p[11].kind).toBe('long');
	});
});

describe('timer', () => {
	it('counts down and moves through phases from timestamps', () => {
		const t = newTimer(phases(defaults), 0);
		expect(remaining(t, 10 * min)).toBe(15 * min);
		expect(advance(t, 10 * min)).toEqual([]);
		expect(advance(t, 26 * min).map((p) => p.kind)).toEqual(['work']);
		expect(t.index).toBe(1);
		expect(remaining(t, 26 * min)).toBe(4 * min); // the break began at minute 25
	});

	it('catches up after the tab slept through several phases', () => {
		const t = newTimer(phases(defaults), 0);
		expect(advance(t, 61 * min).map((p) => p.kind)).toEqual(['work', 'short', 'work', 'short']);
		expect(workedMinutes(t, 61 * min)).toBe(51);
		expect(rounds(t)).toBe(2);
	});

	it('pauses and resumes without counting the pause', () => {
		const t = newTimer(phases(defaults), 0);
		pause(t, 10 * min);
		expect(remaining(t, 40 * min)).toBe(15 * min);
		expect(advance(t, 40 * min)).toEqual([]);
		resume(t, 40 * min);
		expect(remaining(t, 45 * min)).toBe(10 * min);
	});

	it('skips, finishes, and counts only work', () => {
		const t = newTimer(phases(defaults), 0);
		skip(t, 10 * min); // 10 minutes of work done
		skip(t, 11 * min); // skip the break
		expect(workedMinutes(t, 11 * min)).toBe(10);
		advance(t, 200 * min);
		expect(finished(t)).toBe(true);
		expect(workedMinutes(t, 200 * min)).toBe(60);
		expect(rounds(t)).toBe(3);
		expect(remaining(t, 200 * min)).toBe(0);
	});

	it('survives a save and restore as JSON', () => {
		const t = newTimer(phases(defaults), 0);
		advance(t, 30 * min);
		const back = JSON.parse(JSON.stringify(t));
		expect(remaining(back, 31 * min)).toBe(remaining(t, 31 * min));
	});

	it('formats the clock', () => {
		expect(clock(25 * min)).toBe('25:00');
		expect(clock(59_001)).toBe('01:00');
		expect(clock(1)).toBe('00:01');
	});
});
