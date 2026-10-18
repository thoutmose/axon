// Renders the trailer (EP-125): every frame of trailer.html at 60 fps in 4K
// (1920 × 1080 CSS pixels at a device scale of 2), its captions in English
// and French, and its soundtrack. Frames are shared among a few pages, each
// drawing every n-th one. Run by make-trailer.sh:
// node render.mjs <work dir> [--preview t1,t2,…].
import { chromium } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import { availableParallelism } from 'node:os';

const [work, flag, list] = process.argv.slice(2);
const FPS = 60;
const WORKERS = Math.max(1, Math.min(6, availableParallelism() - 2));
const browser = await chromium.launch();
const open = async () => {
	const page = await browser.newPage({ viewport: { width: 1920, height: 1080 }, deviceScaleFactor: 2 });
	await page.goto(`file://${work}/trailer.html`);
	await page.evaluate(async () => {
		await document.fonts.ready;
		await Promise.all([...document.images].map((i) => i.decode().catch(() => {})));
	});
	const stage = page.locator('#stage');
	return async (t, path) => {
		await page.evaluate((t) => window.render(t), t);
		await stage.screenshot({ path, type: 'jpeg', quality: 95 });
	};
};

if (flag === '--preview') {
	const frame = await open();
	for (const t of list.split(',').map(Number)) await frame(t, `${work}/preview-${t}.jpg`);
	await browser.close();
	process.exit(0);
}

const first = await browser.newPage();
await first.goto(`file://${work}/trailer.html`);
const { DURATION, captions } = await first.evaluate(() => window.film);
mkdirSync(`${work}/frames`, { recursive: true });
const frames = Math.ceil(DURATION * FPS);
let done = 0;
await Promise.all(
	Array.from({ length: WORKERS }, async (_, w) => {
		const frame = await open();
		for (let i = w; i < frames; i += WORKERS) {
			await frame(i / FPS, `${work}/frames/${String(i).padStart(5, '0')}.jpg`);
			if (++done % 600 === 0) console.log(`${done}/${frames}`);
		}
	})
);

const stamp = (s) => {
	const ms = Math.round(s * 1000);
	const h = String(Math.floor(ms / 3_600_000)).padStart(2, '0');
	const m = String(Math.floor(ms / 60_000) % 60).padStart(2, '0');
	const sec = String(Math.floor(ms / 1000) % 60).padStart(2, '0');
	return `${h}:${m}:${sec}.${String(ms % 1000).padStart(3, '0')}`;
};
for (const lang of ['en', 'fr']) {
	let vtt = 'WEBVTT\n';
	captions.forEach((c, i) => (vtt += `\n${i + 1}\n${stamp(c.from)} --> ${stamp(c.to)}\n${c[lang]}\n`));
	writeFileSync(`${work}/axon.${lang}.vtt`, vtt);
}
const wav = await first.evaluate((s) => window.composeTrack(s), DURATION + 1);
writeFileSync(`${work}/music.wav`, Buffer.from(wav, 'base64'));
writeFileSync(`${work}/duration`, DURATION.toFixed(2));
console.log(`${frames} frames, ${DURATION.toFixed(1)} s, ${WORKERS} pages`);
await browser.close();
