/**
 * Focus music composed in the browser (EP-83): lo-fi beats, ambient pads, or
 * calm piano, never the same twice and never anything with words, which
 * compete with reading and writing for attention. Nothing is streamed, so
 * no other site learns when or how long someone works.
 *
 * `bar` writes the notes (pure and seeded, so it is unit-tested); `playMusic`
 * plays them with Web Audio a little ahead of time.
 */

export type Style = 'lofi' | 'ambient' | 'piano';
export type Voice = 'keys' | 'bass' | 'pad' | 'piano' | 'bell' | 'kick' | 'snare' | 'hat';
/** One note: when in the bar and for how long, in beats. */
export type Note = { at: number; length: number; midi: number; velocity: number; voice: Voice };
type Chord = { root: number; notes: number[] };

const chord = (root: number, ...notes: number[]): Chord => ({ root, notes });

/** Tempo, chords (one per bar, or per two bars for ambient), and melody notes. */
export const styles: Record<Style, { bpm: number; chords: Chord[]; melody: number[] }> = {
	// ii–V–I–vi in F with ninths, the usual lo-fi loop.
	lofi: {
		bpm: 72,
		chords: [
			chord(43, 58, 62, 65, 69), // Gm9
			chord(36, 52, 58, 62, 67), // C9
			chord(41, 57, 60, 64, 67), // Fmaj9
			chord(38, 53, 57, 60, 64) // Dm9
		],
		melody: [72, 74, 77, 79, 81] // F major pentatonic
	},
	// Slow, open chords in C that never resolve hard.
	ambient: {
		bpm: 60,
		chords: [
			chord(36, 55, 59, 62, 64), // Cmaj9
			chord(45, 57, 60, 64, 67, 71), // Am9
			chord(41, 57, 60, 64, 71), // Fmaj7#11
			chord(43, 57, 62, 64, 67) // G6sus2
		],
		melody: [72, 74, 76, 79, 81, 84] // C major pentatonic
	},
	// I–V–vi–iii–IV–I–IV–V in C, broken chords as in a study.
	piano: {
		bpm: 66,
		chords: [
			chord(48, 60, 64, 67), // C
			chord(43, 59, 62, 67), // G
			chord(45, 60, 64, 69), // Am
			chord(40, 59, 64, 67), // Em
			chord(41, 60, 65, 69), // F
			chord(48, 60, 64, 67), // C
			chord(41, 60, 65, 69), // F
			chord(43, 59, 62, 67) // G
		],
		melody: []
	}
};

/** Music for what a session is about: beats for doing, calm for thinking. */
export function styleFor(type: string | undefined): Style {
	if (type === 'note' || type === 'resource' || type === 'area') return 'ambient';
	if (type === 'skill' || type === 'roadmap' || type === 'card' || type === 'deck') return 'piano';
	return 'lofi';
}

/** A small seeded random generator (mulberry32), so a bar can be replayed in tests. */
export function seeded(seed: number): () => number {
	let a = seed >>> 0;
	return () => {
		a = (a + 0x6d2b79f5) >>> 0;
		let t = a;
		t = Math.imul(t ^ (t >>> 15), t | 1);
		t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
		return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
	};
}

/** Equal temperament, A4 = 440 Hz. */
export const hz = (midi: number) => 440 * 2 ** ((midi - 69) / 12);

const pick = <T>(xs: T[], random: () => number) => xs[Math.floor(random() * xs.length)];
const soft = (v: number, random: () => number) => Math.min(1, v * (0.85 + random() * 0.3));

/** The notes of bar `index` (4 beats). */
export function bar(style: Style, index: number, random: () => number): Note[] {
	const s = styles[style];
	const out: Note[] = [];
	const add = (at: number, length: number, midi: number, velocity: number, voice: Voice) =>
		out.push({ at, length, midi, velocity: soft(velocity, random), voice });

	if (style === 'lofi') {
		const c = s.chords[index % s.chords.length];
		add(0, 1.5, c.root, 0.7, 'bass');
		add(2.5, 1, random() < 0.5 ? c.root : c.root + 7, 0.55, 'bass');
		// A lazily strummed chord, and sometimes a softer one on the "and" of three.
		c.notes.forEach((m, k) => add(k * 0.03, 1.75, m, 0.42, 'keys'));
		if (random() < 0.6) c.notes.forEach((m, k) => add(2.5 + k * 0.03, 1, m, 0.3, 'keys'));
		for (let e = 0; e < 8; e++)
			if (random() < 0.16) add(e / 2, 0.75, pick(s.melody, random), 0.35, 'keys');
		add(0, 0.5, 0, 0.9, 'kick');
		add(2.5, 0.5, 0, 0.8, 'kick');
		if (random() < 0.3) add(1.75, 0.5, 0, 0.5, 'kick');
		add(1, 0.5, 0, 0.6, 'snare');
		add(3, 0.5, 0, 0.6, 'snare');
		// Swung eighths.
		for (let e = 0; e < 8; e++)
			add(e / 2 + (e % 2 ? 0.08 : 0), 0.25, 0, e % 2 ? 0.18 : 0.26, 'hat');
	} else if (style === 'ambient') {
		const c = s.chords[Math.floor(index / 2) % s.chords.length];
		if (index % 2 === 0) {
			add(0, 8, c.root, 0.35, 'bass');
			c.notes.forEach((m) => add(0, 8, m, 0.3, 'pad'));
		}
		for (let b = 0; b < 4; b++)
			if (random() < 0.35) add(b + random() * 0.5, 3, pick(s.melody, random), 0.3, 'bell');
	} else {
		const c = s.chords[index % s.chords.length];
		add(0, 4, c.root, 0.4, 'piano');
		add(2, 2, c.root + 12, 0.3, 'piano');
		// Alberti-style broken chord: low, high, middle, high.
		[0, 2, 1, 2, 0, 2, 1, 2].forEach((k, e) => add(e / 2, 0.9, c.notes[k], 0.26, 'piano'));
		if (random() < 0.6) add(0, 2, pick(c.notes, random) + 12, 0.38, 'piano');
		if (random() < 0.4) add(2, 2, pick(c.notes, random) + 12, 0.32, 'piano');
	}
	return out;
}

/** Plays a style until the returned function is called. */
export function playMusic(ctx: AudioContext, style: Style, volume: number): () => void {
	const band = music(ctx, style, volume, seeded(Date.now()));
	// A second and a half ahead: background tabs run timers only once a second.
	const schedule = () => band.until(ctx.currentTime + 1.5);
	schedule();
	const timer = setInterval(schedule, 250);
	return () => {
		clearInterval(timer);
		band.fade();
		setTimeout(band.disconnect, 600);
	};
}

/**
 * Wires a style's instruments into `ctx`; `until(t)` schedules every bar that
 * starts before time t. It takes any audio context, so a test can render
 * music offline and measure it.
 */
export function music(ctx: BaseAudioContext, style: Style, volume: number, random: () => number) {
	const beat = 60 / styles[style].bpm;

	const master = ctx.createGain();
	master.gain.value = Math.max(0, Math.min(1, volume / 100)) * 0.5;
	const limiter = ctx.createDynamicsCompressor();
	// Lo-fi is darker; the rest keeps its air.
	const tone = ctx.createBiquadFilter();
	tone.type = 'lowpass';
	tone.frequency.value = style === 'lofi' ? 3200 : 9000;
	tone.connect(limiter).connect(master).connect(ctx.destination);

	const reverb = ctx.createConvolver();
	reverb.buffer = impulse(ctx, style === 'ambient' ? 4 : 2.2);
	const wet = ctx.createGain();
	wet.gain.value = style === 'ambient' ? 0.5 : 0.22;
	reverb.connect(wet).connect(tone);

	const white = ctx.createBuffer(1, ctx.sampleRate, ctx.sampleRate);
	const w = white.getChannelData(0);
	for (let i = 0; i < w.length; i++) w[i] = random() * 2 - 1;

	// The record's hiss under lo-fi.
	if (style === 'lofi') {
		const hiss = ctx.createBufferSource();
		hiss.buffer = white;
		hiss.loop = true;
		const f = ctx.createBiquadFilter();
		f.type = 'bandpass';
		f.frequency.value = 2500;
		const g = ctx.createGain();
		g.gain.value = 0.012;
		hiss.connect(f).connect(g).connect(tone);
		hiss.start(ctx.currentTime);
	}

	function voice(n: Note, t: number) {
		const len = n.length * beat;
		const out = ctx.createGain();
		out.connect(tone);
		if (n.voice !== 'kick' && n.voice !== 'hat') out.connect(reverb);
		const g = out.gain;
		const v = n.velocity;
		const osc = (type: OscillatorType, freq: number, level = 1) => {
			const o = ctx.createOscillator();
			o.type = type;
			o.frequency.value = freq;
			const og = ctx.createGain();
			og.gain.value = level;
			o.connect(og).connect(out);
			o.start(t);
			return o;
		};
		const noise = (type: BiquadFilterType, freq: number) => {
			const src = ctx.createBufferSource();
			src.buffer = white;
			const f = ctx.createBiquadFilter();
			f.type = type;
			f.frequency.value = freq;
			src.connect(f).connect(out);
			src.start(t);
			return src;
		};
		let end = t + len;
		switch (n.voice) {
			case 'kick': {
				const o = osc('sine', 110);
				o.frequency.setValueAtTime(110, t);
				o.frequency.exponentialRampToValueAtTime(45, t + 0.12);
				g.setValueAtTime(v, t);
				g.exponentialRampToValueAtTime(0.001, (end = t + 0.35));
				o.stop(end);
				return;
			}
			case 'snare':
			case 'hat': {
				const src = noise(
					n.voice === 'hat' ? 'highpass' : 'bandpass',
					n.voice === 'hat' ? 7000 : 1800
				);
				g.setValueAtTime(v * (n.voice === 'hat' ? 0.5 : 0.7), t);
				g.exponentialRampToValueAtTime(0.001, (end = t + (n.voice === 'hat' ? 0.05 : 0.18)));
				src.stop(end);
				return;
			}
		}
		const f = hz(n.midi);
		let oscs: OscillatorNode[];
		if (n.voice === 'pad') {
			// Two detuned saws, softened, swelling in and out.
			const lp = ctx.createBiquadFilter();
			lp.type = 'lowpass';
			lp.frequency.value = 900;
			out.disconnect();
			out.connect(lp);
			lp.connect(tone);
			lp.connect(reverb);
			oscs = [osc('sawtooth', f * 0.996, 0.5), osc('sawtooth', f * 1.004, 0.5)];
			g.setValueAtTime(0, t);
			g.linearRampToValueAtTime(v * 0.25, t + Math.min(2, len / 2));
			g.setValueAtTime(v * 0.25, end - Math.min(2, len / 2));
			g.linearRampToValueAtTime(0, end);
		} else if (n.voice === 'bass') {
			oscs = [osc('sine', f)];
			g.setValueAtTime(0, t);
			g.linearRampToValueAtTime(v * 0.6, t + 0.02);
			g.setValueAtTime(v * 0.6, end - 0.1);
			g.linearRampToValueAtTime(0, end);
		} else {
			// Keys, piano, and bells: struck, then fading.
			const decay = { keys: 1.2, piano: 2.4, bell: 3.5 }[n.voice];
			oscs =
				n.voice === 'bell'
					? [osc('sine', f, 0.8), osc('sine', f * 2.76, 0.15)]
					: n.voice === 'piano'
						? [osc('triangle', f, 0.7), osc('sine', f * 2, 0.15)]
						: [osc('sine', f, 0.8), osc('sine', f * 2, 0.12)];
			end = t + Math.max(len, 0.3) + 0.4;
			g.setValueAtTime(0, t);
			// Piano notes are short and soft, so they are struck harder to sit level with the rest.
			g.linearRampToValueAtTime(v * (n.voice === 'piano' ? 1.2 : 0.35), t + 0.008);
			g.setTargetAtTime(0, t + 0.01, decay / 4);
			g.setTargetAtTime(0, t + Math.max(len, 0.3), 0.12);
		}
		for (const o of oscs) o.stop(end + 0.05);
	}

	let index = 0;
	let next = ctx.currentTime + 0.1;
	return {
		until(t: number) {
			for (; next < t; next += 4 * beat, index++)
				for (const n of bar(style, index, random)) voice(n, next + n.at * beat);
		},
		fade() {
			master.gain.setTargetAtTime(0, ctx.currentTime, 0.15);
		},
		/** Silences what is already scheduled; the context stays open for the next sound. */
		disconnect() {
			master.disconnect();
		}
	};
}

/** A room: decaying noise, a little different on each side. */
function impulse(ctx: BaseAudioContext, seconds: number): AudioBuffer {
	const length = Math.floor(ctx.sampleRate * seconds);
	const buf = ctx.createBuffer(2, length, ctx.sampleRate);
	for (let ch = 0; ch < 2; ch++) {
		const d = buf.getChannelData(ch);
		for (let i = 0; i < length; i++) d[i] = (Math.random() * 2 - 1) * (1 - i / length) ** 3;
	}
	return buf;
}
