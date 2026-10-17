// Takes the trailer's screenshots (EP-125) from a running Axon with the
// demo on, in the light theme: node shots.mjs <base URL> <out dir>. It
// writes one note in the demo, which resets every night.
import { chromium } from '@playwright/test';
import { mkdirSync } from 'node:fs';

const [base, out] = process.argv.slice(2);
mkdirSync(out, { recursive: true });

// What each shot shows, and what to bring to the top before taking it.
// path may be a function of the demo's data; open clicks before the shot;
// region names a labelled section to bring up instead of a heading; close
// clicks after it, so a panel that remembers being open leaves the next shots.
const shots = [
	{ name: 'today', path: '/today', heading: 'Your roadmap today', offset: 120 },
	{ name: 'plan', path: (d) => `/roadmaps/${d.roadmap}?view=plan`, heading: 'The plan', offset: 110 },
	{ name: 'skills', path: '/roadmaps', heading: 'Skills acquired', offset: 70 },
	{ name: 'tasks', path: '/tasks?view=kanban' },
	{ name: 'notes', path: '/notes', heading: 'Knowledge graph' },
	{ name: 'sketch', path: (d) => `/notes/${d.note}` },
	{ name: 'learn', path: '/learn' },
	{ name: 'jarvis', path: '/today', open: 'Open Jarvis', close: 'Close Jarvis' },
	{ name: 'graph', path: '/watch/graph', heading: 'News graph', offset: 160 },
	{ name: 'media', path: '/media', heading: 'In progress' },
	{ name: 'health', path: '/today', heading: 'Coffee and tea' },
	{ name: 'budget', path: '/budget', region: 'Where the money went' },
	{ name: 'collection', path: '/progress', heading: 'Your collection' }
];

// A note with a drawing and maths, as someone would write it (EP-150, EP-143).
const sketch = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1200 800" width="1200" height="800"><title>Raft</title><rect width="1200" height="800" fill="#ffffff"/>
<g fill="none" stroke-linecap="round" stroke-linejoin="round">
<ellipse cx="300" cy="260" rx="120" ry="70" stroke="#0969da" stroke-width="4"/><ellipse cx="900" cy="260" rx="120" ry="70" stroke="#1f2328" stroke-width="4"/><ellipse cx="600" cy="600" rx="120" ry="70" stroke="#1f2328" stroke-width="4"/>
<path d="M420 260H760M752 248L780 260L752 272" stroke="#d1242f" stroke-width="4"/><path d="M360 320L520 540M512 524L530 552L540 520" stroke="#d1242f" stroke-width="4"/>
<path d="M120 700Q300 640 480 700T840 700" stroke="#1a7f37" stroke-width="12" stroke-opacity="0.35"/></g>
<g font-family="system-ui, sans-serif" font-size="40"><text x="232" y="274" fill="#0969da">Leader</text><text x="820" y="274" fill="#1f2328">Follower</text><text x="520" y="614" fill="#1f2328">Follower</text><text x="470" y="230" fill="#d1242f" font-size="30">append entries</text></g></svg>`;

const browser = await chromium.launch();
const context = await browser.newContext({
	viewport: { width: 1440, height: 900 },
	deviceScaleFactor: 2,
	colorScheme: 'light',
	reducedMotion: 'reduce',
	bypassCSP: true // to hide the demo banner in this capture only
});
await context.addInitScript(() => localStorage.setItem('axon-theme', 'light'));
const page = await context.newPage();
const origin = { Origin: new URL(base).origin };
await page.request.post(base + '/api/v1/auth/demo', { headers: origin });
// The demo's own data: its roadmap, covers looked up, and the note above.
const data = {};
const roadmaps = await (await page.request.get(base + '/api/v1/roadmaps')).json();
data.roadmap = roadmaps.items.find((r) => r.plan_id)?.id;
await page.request.put(base + '/api/v1/me/consents/external_metadata', { headers: origin, data: { granted: true } });
for (let i = 0; i < 3; i++) await page.request.post(base + '/api/v1/media/covers/find', { headers: origin });
const file = await (
	await page.request.post(base + '/api/v1/notes/files?name=raft.svg', { headers: { ...origin, 'Content-Type': 'image/svg+xml' }, data: sketch })
).json();
const note = await (
	await page.request.post(base + '/api/v1/notes', {
		headers: origin,
		data: {
			path: `Zettel/permanent/raft-in-a-picture-${Date.now()}.md`,
			content: `# Raft, in a picture\n\nA leader appends entries; a majority commits them: $q = \\lfloor n/2 \\rfloor + 1$.\n\n![raft](${file.path})\n\n$$\nP(\\text{split vote}) \\approx \\frac{1}{2^{k}}\n$$\n`
		}
	})
).json();
data.note = note.id;
for (const s of shots) {
	await page.goto(base + (typeof s.path === 'function' ? s.path(data) : s.path));
	await page.waitForLoadState('networkidle');
	// The demo's banner is for visitors, not for the film.
	await page.addStyleTag({ content: '.demo-banner, .space-credit, #jarvis-panel .notice { display: none !important; } #jarvis-panel .chips button { opacity: 1 !important; }' });
	if (s.open) {
		await page.getByRole('button', { name: s.open }).click();
		await page.waitForTimeout(800);
	}
	if (s.heading || s.region) {
		const h = page.getByRole(s.region ? 'region' : 'heading', { name: s.region ?? s.heading }).first();
		await h.waitFor();
		await h.evaluate((el, offset) => scrollTo(0, el.getBoundingClientRect().top + scrollY - offset), s.offset ?? 90);
	}
	await page.waitForTimeout(1200);
	await page.screenshot({ path: `${out}/${s.name}.jpg`, type: 'jpeg', quality: 95 });
	if (s.close) await page.getByRole('button', { name: s.close }).click();
	console.log(s.name);
}
await browser.close();
