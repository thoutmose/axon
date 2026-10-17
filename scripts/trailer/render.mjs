// Renders the trailer (EP-125): every frame of trailer.html at 30 fps, its
// captions in English and French, and its soundtrack. Run by
// make-trailer.sh: node render.mjs <work dir> [--preview t1,t2,…].
import { chromium } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';

const [work, flag, list] = process.argv.slice(2);
const FPS = 30;
const browser = await chromium.launch();
const page = await browser.newPage({ viewport: { width: 1920, height: 1080 } });
await page.goto(`file://${work}/trailer.html`);
await page.evaluate(async () => {
	await document.fonts.ready;
	await Promise.all([...document.images].map((i) => i.decode().catch(() => {})));
});
const stage = page.locator('#stage');
const frame = async (t, path) => {
	await page.evaluate((t) => window.render(t), t);
	await stage.screenshot({ path, type: 'jpeg', quality: 97 });
};

if (flag === '--preview') {
	for (const t of list.split(',').map(Number)) await frame(t, `${work}/preview-${t}.jpg`);
	await browser.close();
	process.exit(0);
}

const { DURATION, captions } = await page.evaluate(() => window.film);
mkdirSync(`${work}/frames`, { recursive: true });
const frames = Math.ceil(DURATION * FPS);
for (let i = 0; i < frames; i++) {
	await frame(i / FPS, `${work}/frames/${String(i).padStart(5, '0')}.jpg`);
	if (i % 300 === 0) console.log(`${i}/${frames}`);
}

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
const wav = await page.evaluate((s) => window.composeTrack(s), DURATION + 1);
writeFileSync(`${work}/music.wav`, Buffer.from(wav, 'base64'));
writeFileSync(`${work}/duration`, DURATION.toFixed(2));
console.log(`${frames} frames, ${DURATION.toFixed(1)} s`);
await browser.close();
