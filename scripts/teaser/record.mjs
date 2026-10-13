// Records the teaser shown on the home page when the demo is on (EP-84): a
// scripted tour of the demo in Chromium, captured frame by frame, a
// soundtrack from Axon's own lo-fi focus music, and captions in English and
// French timed from the tour itself. Run by make-teaser.sh.
import { chromium } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';

const [base, work, out] = process.argv.slice(2);
const W = 1280;
const H = 720;

// Each scene: where, what the caption says, and what happens there.
const scenes = [
	{
		en: ['Axon', 'Your self-hosted life OS'],
		fr: ['Axon', 'Votre OS de vie auto-hébergé'],
		go: async (p) => {
			await p.goto(base + '/');
			await p.getByRole('button', { name: 'Try the demo' }).waitFor();
			await p.waitForTimeout(3000);
			await p.getByRole('button', { name: 'Try the demo' }).click();
			await p.waitForURL(/\/today$/);
		}
	},
	{
		en: ['Today', 'Your day, your progress, in one place'],
		fr: ['Aujourd’hui', 'Votre journée et vos progrès, au même endroit'],
		go: (p) => tour(p, '/today', (p) => p.getByRole('heading', { name: 'Your progress' }), 2)
	},
	{
		en: ['Tasks and projects', 'GTD, Kanban, and a calendar'],
		fr: ['Tâches et projets', 'GTD, Kanban et un calendrier'],
		go: (p) => tour(p, '/tasks', null, 1)
	},
	{
		en: ['Notes that connect', 'A Zettelkasten graph of your second brain'],
		fr: ['Des notes qui se relient', 'Le graphe Zettelkasten de votre second cerveau'],
		go: (p) => tour(p, '/notes/explore', (p) => p.getByRole('heading', { name: 'Knowledge graph' }), 0, 4500)
	},
	{
		en: ['Goals you can explain', 'Milestones, tasks, and habits, connected'],
		fr: ['Des objectifs expliqués', 'Jalons, tâches et habitudes, reliés'],
		go: (p) => tour(p, '/goals', (p) => p.getByRole('heading', { name: 'Weekly goal review' }), 1)
	},
	{
		en: ['Skills, at a glance', 'Hard and soft skills acquired, on a radar'],
		fr: ['Vos compétences d’un coup d’œil', 'Savoir-faire techniques et humains, en toile d’araignée'],
		go: (p) => tour(p, '/roadmaps', (p) => p.getByRole('heading', { name: 'Skills acquired' }), 0, 4500)
	},
	{
		en: ['Learn for good', 'Spaced repetition that measures your memory'],
		fr: ['Apprendre pour de bon', 'La répétition espacée qui mesure votre mémoire'],
		go: (p) => tour(p, '/learn', null, 1)
	},
	{
		en: ['Focus', 'Pomodoro with music composed for your work'],
		fr: ['Concentration', 'Pomodoro et musique composée pour votre travail'],
		go: async (p) => {
			await p.goto(base + '/focus');
			await p.getByLabel('A task, project, skill, note, or anything else').fill('About');
			await p.locator('.hits button').first().click();
			await p.waitForTimeout(1200);
			await p.getByRole('button', { name: /^Start/ }).click();
			await bring(p.locator('section.timer'), 'center');
			await p.waitForTimeout(3500);
			await p.getByRole('button', { name: 'Stop and save' }).click();
		}
	},
	{
		en: ['Books, films, and series', 'Everything you read and watch'],
		fr: ['Livres, films et séries', 'Tout ce que vous lisez et regardez'],
		go: (p) => tour(p, '/media', (p) => p.getByRole('heading', { name: /^In progress/ }), 1)
	},
	{
		en: ['Yours, on your server', 'Open source. Try the demo.'],
		fr: ['À vous, sur votre serveur', 'Open source. Essayez la démo.'],
		go: async (p) => {
			await p.goto(base + '/today');
			await endCard();
			await p.waitForTimeout(4000);
		}
	}
];

let page;

/** Scrolls smoothly until an element is at the top (below the menu bar) or in the middle. */
async function bring(locator, where = 'top') {
	await locator.first().waitFor();
	await locator.first().evaluate((el, where) => {
		const box = el.getBoundingClientRect();
		const top = where === 'top' ? box.top - 70 : box.top - (innerHeight - box.height) / 2;
		scrollTo({ top: scrollY + top, behavior: 'smooth' });
	}, where);
	await page.waitForTimeout(900);
}

/** Opens a page, brings its best part into view, and scrolls on slowly. */
async function tour(p, path, target, scrolls, wait = 3000) {
	await p.goto(base + path);
	await p.waitForLoadState('networkidle');
	await caption();
	await p.waitForTimeout(600);
	if (target) await bring(target(p));
	await p.waitForTimeout(wait / 2);
	for (let i = 0; i < scrolls * 12; i++) {
		await p.mouse.wheel(0, 30);
		await p.waitForTimeout(40);
	}
	await p.waitForTimeout(wait / 2);
}

let current = null;
/** Shows the current scene's caption, again after each navigation. */
async function caption() {
	if (!current) return;
	await page.evaluate(([title, sub]) => {
		document.getElementById('teaser-caption')?.remove();
		const box = document.createElement('div');
		box.id = 'teaser-caption';
		box.style.cssText =
			'position:fixed;left:32px;bottom:32px;z-index:99999;padding:14px 22px;border-radius:14px;' +
			'background:rgb(15 23 42 / .82);color:#fff;font:600 26px/1.25 system-ui,sans-serif;' +
			'box-shadow:0 10px 30px rgb(0 0 0 / .25);max-width:70%';
		const s = document.createElement('div');
		s.style.cssText = 'font-weight:400;font-size:18px;opacity:.85;margin-top:4px';
		s.textContent = sub;
		box.append(title, s);
		document.body.append(box);
	}, current);
}

/** The closing card, over the page. */
async function endCard() {
	current = null; // no caption over the card
	await page.evaluate(() => {
		document.getElementById('teaser-caption')?.remove();
		const card = document.createElement('div');
		card.style.cssText =
			'position:fixed;inset:0;z-index:99999;display:grid;place-content:center;text-align:center;gap:14px;' +
			'background:linear-gradient(160deg,rgb(30 27 75 / .94),rgb(76 29 149 / .9));color:#fff;' +
			'font:400 24px/1.4 system-ui,sans-serif;opacity:0;transition:opacity .8s';
		const h = document.createElement('div');
		h.style.cssText = 'font-weight:700;font-size:72px;letter-spacing:-.02em';
		h.textContent = 'Axon';
		const p = document.createElement('div');
		p.textContent = 'Your life, connected. Open source, on your own server.';
		const b = document.createElement('div');
		b.style.cssText =
			'justify-self:center;margin-top:12px;padding:10px 26px;border-radius:999px;background:#fff;color:#312e81;font-weight:600';
		b.textContent = 'Try the demo';
		card.append(h, p, b);
		document.body.append(card);
		requestAnimationFrame(() => requestAnimationFrame(() => (card.style.opacity = '1')));
	});
}

const browser = await chromium.launch();
const context = await browser.newContext({
	viewport: { width: W, height: H },
	bypassCSP: true, // the captions are drawn into the page
	reducedMotion: 'no-preference'
});
page = await context.newPage();
page.on('load', () => caption().catch(() => {}));

// Frames arrive when the page changes; each lasts until the next.
mkdirSync(`${work}/frames`, { recursive: true });
const frames = [];
const cdp = await context.newCDPSession(page);
cdp.on('Page.screencastFrame', async (f) => {
	const file = `f${String(frames.length).padStart(5, '0')}.jpg`;
	writeFileSync(`${work}/frames/${file}`, Buffer.from(f.data, 'base64'));
	frames.push({ file, t: f.metadata.timestamp });
	await cdp.send('Page.screencastFrameAck', { sessionId: f.sessionId }).catch(() => {});
});
await page.goto(base + '/');
await cdp.send('Page.startScreencast', { format: 'jpeg', quality: 92, maxWidth: W, maxHeight: H });
await page.waitForTimeout(300);

const starts = [];
for (const s of scenes) {
	current = s.en;
	starts.push(Date.now() / 1000);
	await caption();
	await s.go(page);
}
const end = Date.now() / 1000;
await cdp.send('Page.stopScreencast');

const t0 = frames[0].t;
const duration = end - t0;
let list = 'ffconcat version 1.0\n';
frames.forEach((f, i) => {
	const next = i + 1 < frames.length ? frames[i + 1].t : end;
	list += `file 'frames/${f.file}'\nduration ${Math.max(0.001, next - f.t).toFixed(3)}\n`;
});
list += `file 'frames/${frames[frames.length - 1].file}'\n`;
writeFileSync(`${work}/frames.txt`, list);

// Captions from the scenes' real start times.
const stamp = (s) => {
	const ms = Math.max(0, Math.round(s * 1000));
	const h = String(Math.floor(ms / 3_600_000)).padStart(2, '0');
	const m = String(Math.floor(ms / 60_000) % 60).padStart(2, '0');
	const sec = String(Math.floor(ms / 1000) % 60).padStart(2, '0');
	return `${h}:${m}:${sec}.${String(ms % 1000).padStart(3, '0')}`;
};
for (const lang of ['en', 'fr']) {
	let vtt = 'WEBVTT\n';
	scenes.forEach((s, i) => {
		const from = starts[i] - t0;
		const to = (i + 1 < starts.length ? starts[i + 1] : end) - t0;
		vtt += `\n${i + 1}\n${stamp(from)} --> ${stamp(to)}\n${s[lang][0]}: ${s[lang][1]}\n`;
	});
	writeFileSync(`${out}/axon.${lang}.vtt`, vtt);
}

// The soundtrack: Axon's lo-fi focus music, rendered offline.
const music = await browser.newPage();
await music.addScriptTag({ path: `${work}/music.js` });
const wav = await music.evaluate(async (seconds) => {
	const rate = 44100;
	const ctx = new OfflineAudioContext(2, Math.ceil(seconds * rate), rate);
	AxonMusic.music(ctx, 'lofi', 100, AxonMusic.seeded(7)).until(seconds);
	const buf = await ctx.startRendering();
	const n = buf.length;
	const view = new DataView(new ArrayBuffer(44 + n * 4));
	const text = (o, s) => [...s].forEach((c, i) => view.setUint8(o + i, c.charCodeAt(0)));
	text(0, 'RIFF');
	view.setUint32(4, 36 + n * 4, true);
	text(8, 'WAVEfmt ');
	view.setUint32(16, 16, true);
	view.setUint16(20, 1, true);
	view.setUint16(22, 2, true);
	view.setUint32(24, rate, true);
	view.setUint32(28, rate * 4, true);
	view.setUint16(32, 4, true);
	view.setUint16(34, 16, true);
	text(36, 'data');
	view.setUint32(40, n * 4, true);
	const l = buf.getChannelData(0);
	const r = buf.getChannelData(1);
	for (let i = 0; i < n; i++) {
		view.setInt16(44 + i * 4, Math.max(-1, Math.min(1, l[i])) * 32767, true);
		view.setInt16(46 + i * 4, Math.max(-1, Math.min(1, r[i])) * 32767, true);
	}
	let bin = '';
	const bytes = new Uint8Array(view.buffer);
	for (let i = 0; i < bytes.length; i += 0x8000) bin += String.fromCharCode(...bytes.subarray(i, i + 0x8000));
	return btoa(bin);
}, duration + 1);
writeFileSync(`${work}/music.wav`, Buffer.from(wav, 'base64'));
writeFileSync(`${work}/duration`, duration.toFixed(2));
console.log(`${frames.length} frames, ${duration.toFixed(1)} s`);
await browser.close();
