/**
 * Focus sounds generated in the browser (EP-35): white, pink, and brown
 * noise, rain, and music (EP-83). Nothing is downloaded, so no third-party
 * audio is loaded unless the person opens their own music links.
 */

import { playMusic, styleFor } from './music';

type Noise = 'white' | 'pink' | 'brown' | 'rain';
export type Sound = 'none' | Noise | 'music' | 'lofi' | 'ambient' | 'piano';

/** The choices, in the order the Focus and Preferences pages offer them. */
export const soundOptions: { group: string; items: [Sound, string][] }[] = [
	{
		group: 'Music',
		items: [
			['music', 'Matched to what you focus on'],
			['lofi', 'Lo-fi beats'],
			['ambient', 'Ambient'],
			['piano', 'Calm piano']
		]
	},
	{
		group: 'Noise',
		items: [
			['white', 'White noise'],
			['pink', 'Pink noise'],
			['brown', 'Brown noise'],
			['rain', 'Rain']
		]
	}
];

/** Fills a buffer with noise; pure, so it is unit-tested. */
export function noise(kind: Noise, length: number, random = Math.random): Float32Array {
	const out = new Float32Array(length);
	let b0 = 0,
		b1 = 0,
		b2 = 0,
		last = 0;
	for (let i = 0; i < length; i++) {
		const w = random() * 2 - 1;
		switch (kind) {
			case 'white':
				out[i] = w * 0.5;
				break;
			case 'pink': // Paul Kellet's economy filter
				b0 = 0.99765 * b0 + w * 0.099046;
				b1 = 0.963 * b1 + w * 0.2965164;
				b2 = 0.57 * b2 + w * 1.0526913;
				out[i] = (b0 + b1 + b2 + w * 0.1848) * 0.11;
				break;
			case 'brown':
			case 'rain':
				last = (last + 0.02 * w) / 1.02;
				out[i] = last * 3.5;
				// Rain: soft drops over the brown rumble.
				if (kind === 'rain' && random() < 0.0006) out[i] += (random() * 2 - 1) * 0.6;
				break;
		}
		out[i] = Math.max(-1, Math.min(1, out[i]));
	}
	return out;
}

let shared: AudioContext | null = null;

/**
 * The page's one audio context. Browsers (Safari above all) only let sound
 * start from a click, so call this in the click handler itself, before any
 * await; later sounds reuse it.
 */
export function audio(): AudioContext | null {
	if (typeof AudioContext === 'undefined') return null;
	if (!shared || shared.state === 'closed') shared = new AudioContext();
	if (shared.state === 'suspended') shared.resume().catch(() => {});
	return shared;
}

/**
 * Starts a sound; returns a function that stops it. "music" picks the style
 * for the type of what the session is about.
 */
export function play(kind: Sound, volume: number, about?: string): () => void {
	if (kind === 'none') return () => {};
	const ctx = audio();
	if (!ctx) return () => {};
	if (kind === 'music') return playMusic(ctx, styleFor(about), volume);
	if (kind === 'lofi' || kind === 'ambient' || kind === 'piano')
		return playMusic(ctx, kind, volume);
	const seconds = 4;
	const buffer = ctx.createBuffer(1, ctx.sampleRate * seconds, ctx.sampleRate);
	buffer.copyToChannel(noise(kind, buffer.length) as Float32Array<ArrayBuffer>, 0);
	const src = ctx.createBufferSource();
	src.buffer = buffer;
	src.loop = true;
	const gain = ctx.createGain();
	gain.gain.value = Math.max(0, Math.min(1, volume / 100)) * 0.6;
	src.connect(gain).connect(ctx.destination);
	src.start();
	return () => {
		src.stop();
		gain.disconnect();
	};
}

/** A short, gentle chime at the end of a phase. */
export function chime(volume: number) {
	const ctx = audio();
	if (!ctx) return;
	const osc = ctx.createOscillator();
	const gain = ctx.createGain();
	osc.frequency.value = 660;
	gain.gain.setValueAtTime(Math.max(0.05, volume / 200), ctx.currentTime);
	gain.gain.exponentialRampToValueAtTime(0.0001, ctx.currentTime + 1.2);
	osc.connect(gain).connect(ctx.destination);
	osc.start();
	osc.stop(ctx.currentTime + 1.2);
	osc.onended = () => gain.disconnect();
}
