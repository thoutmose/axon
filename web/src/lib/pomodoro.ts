/**
 * The Pomodoro timer (EP-35). It is driven by timestamps, never by counting
 * ticks, so a sleeping tab or a reload cannot make it drift: the state can be
 * saved and restored, and `now` is passed in, so it is tested with a fake clock.
 */

export type PhaseKind = 'work' | 'short' | 'long';
export type Phase = { kind: PhaseKind; minutes: number };

export type FocusConfig = {
	work_minutes: number;
	short_break_minutes: number;
	long_break_minutes: number;
	rounds: number; // work blocks before the long break
	sessions: number; // cycles in a row
};

/** Default: 25 work + 5 break, twice, then 25 work + 15 long break; one session. */
export const defaults: FocusConfig = {
	work_minutes: 25,
	short_break_minutes: 5,
	long_break_minutes: 15,
	rounds: 3,
	sessions: 1
};

export function phases(c: FocusConfig): Phase[] {
	const out: Phase[] = [];
	for (let s = 0; s < c.sessions; s++) {
		for (let r = 0; r < c.rounds; r++) {
			out.push({ kind: 'work', minutes: c.work_minutes });
			out.push(
				r < c.rounds - 1
					? { kind: 'short', minutes: c.short_break_minutes }
					: { kind: 'long', minutes: c.long_break_minutes }
			);
		}
	}
	return out;
}

export type Timer = {
	phases: Phase[];
	index: number; // current phase; phases.length when finished
	startedAt: number; // ms when the current phase (re)started counting
	elapsed: number; // ms of the current phase counted before the last pause
	paused: boolean;
	worked: number; // ms of work in phases already finished
};

export const newTimer = (p: Phase[], now: number): Timer => ({
	phases: p,
	index: 0,
	startedAt: now,
	elapsed: 0,
	paused: false,
	worked: 0
});

const lengthMs = (p: Phase) => p.minutes * 60_000;
const current = (t: Timer, now: number) => t.elapsed + (t.paused ? 0 : now - t.startedAt);

export const finished = (t: Timer) => t.index >= t.phases.length;

/** Milliseconds left in the current phase. */
export function remaining(t: Timer, now: number): number {
	if (finished(t)) return 0;
	return Math.max(0, lengthMs(t.phases[t.index]) - current(t, now));
}

/**
 * Moves past every phase that has ended by `now` (after a long sleep, several
 * may have) and returns the phases that ended, so the page can chime.
 */
export function advance(t: Timer, now: number): Phase[] {
	const ended: Phase[] = [];
	while (!finished(t) && !t.paused) {
		const p = t.phases[t.index];
		const over = current(t, now) - lengthMs(p);
		if (over < 0) break;
		if (p.kind === 'work') t.worked += lengthMs(p);
		ended.push(p);
		t.index++;
		t.startedAt = now - over; // the next phase began when this one ended
		t.elapsed = 0;
	}
	return ended;
}

export function pause(t: Timer, now: number) {
	if (t.paused || finished(t)) return;
	t.elapsed = current(t, now);
	t.paused = true;
}

export function resume(t: Timer, now: number) {
	if (!t.paused) return;
	t.paused = false;
	t.startedAt = now;
}

/** Ends the current phase now (a skipped work block counts what was done). */
export function skip(t: Timer, now: number) {
	if (finished(t)) return;
	const p = t.phases[t.index];
	if (p.kind === 'work') t.worked += Math.min(lengthMs(p), current(t, now));
	t.index++;
	t.startedAt = now;
	t.elapsed = 0;
	t.paused = false;
}

/** Whole minutes of work so far, breaks left out. */
export function workedMinutes(t: Timer, now: number): number {
	let ms = t.worked;
	if (!finished(t) && t.phases[t.index].kind === 'work')
		ms += Math.min(lengthMs(t.phases[t.index]), current(t, now));
	return Math.floor(ms / 60_000);
}

/** Work blocks completed. */
export function rounds(t: Timer): number {
	return t.phases.slice(0, t.index).filter((p) => p.kind === 'work').length;
}

export function totalWorkMinutes(p: Phase[]): number {
	return p.filter((x) => x.kind === 'work').reduce((a, x) => a + x.minutes, 0);
}

/** "04:59" */
export function clock(ms: number): string {
	const s = Math.ceil(ms / 1000);
	return `${String(Math.floor(s / 60)).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`;
}
