// Takes the trailer's screenshots (EP-125) from a running Axon with the
// demo on, in the light theme: node shots.mjs <base URL> <out dir>.
import { chromium } from '@playwright/test';
import { mkdirSync } from 'node:fs';

const [base, out] = process.argv.slice(2);
mkdirSync(out, { recursive: true });

// What each shot shows, and what to bring to the top before taking it.
const shots = [
	{ name: 'today', path: '/today' },
	{ name: 'tasks', path: '/tasks?view=kanban' },
	{ name: 'notes', path: '/notes', heading: 'Knowledge graph' },
	{ name: 'roadmap', path: '/roadmaps', heading: 'Skills acquired' },
	{ name: 'learn', path: '/learn' },
	{ name: 'focus', path: '/focus' },
	{ name: 'watch', path: '/watch' },
	{ name: 'graph', path: '/watch/graph', heading: 'News graph', offset: 160 },
	{ name: 'collection', path: '/progress', heading: 'Your collection' },
	{ name: 'health', path: '/today', heading: 'Coffee and tea' },
	{ name: 'decisions', path: '/decisions', heading: 'Your calibration' },
	{ name: 'habits', path: '/habits', heading: 'Protocols for health, after Andrew Huberman' },
	{ name: 'digest', path: '/digest' },
	{ name: 'goals', path: '/goals' }
];

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
await page.request.post(base + '/api/v1/auth/demo', { headers: { Origin: new URL(base).origin } });
for (const s of shots) {
	await page.goto(base + s.path);
	await page.waitForLoadState('networkidle');
	// The demo's banner is for visitors, not for the film.
	await page.addStyleTag({ content: '.demo-banner, .space-credit { display: none !important; }' });
	if (s.heading) {
		const h = page.getByRole('heading', { name: s.heading }).first();
		await h.waitFor();
		await h.evaluate((el, offset) => scrollTo(0, el.getBoundingClientRect().top + scrollY - offset), s.offset ?? 90);
	}
	await page.waitForTimeout(1200);
	await page.screenshot({ path: `${out}/${s.name}.jpg`, type: 'jpeg', quality: 86 });
	console.log(s.name);
}
await browser.close();
