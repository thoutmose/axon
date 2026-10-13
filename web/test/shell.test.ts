import { expect, test, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

// One story on a fresh database: create the owner account, use every shell
// page, sign out, and sign back in with the same (virtual) passkey.

test.describe.configure({ mode: 'serial' });

let page: Page;

async function noSeriousA11yViolations(p: Page) {
	// Check final colours, not a button caught halfway through a transition.
	await p.emulateMedia({ reducedMotion: 'reduce' });
	await p.waitForTimeout(250); // a transition started before this keeps running; let it end
	const result = await new AxeBuilder({ page: p }).analyze();
	await p.emulateMedia({ reducedMotion: 'no-preference' });
	const serious = result.violations.filter(
		(v) => v.impact === 'serious' || v.impact === 'critical'
	);
	expect(
		serious.map(
			(v) =>
				`${v.id}: ${v.help} (${v.nodes.map((n) => `${n.target.join(' ')}: ${n.failureSummary}`).join(', ')})`
		)
	).toEqual([]);
}

// The browser runs in a time zone whose date is not UTC's right now (UTC+14
// late in the UTC day, UTC-11 early in it), so showing a date shifted by the
// browser's time zone is caught whenever the suite runs.
// The stack under test: its own on E2E_BASE_URL, so a running local stack is left alone.
declare const process: { env: Record<string, string | undefined> }; // Node's, without its types in the web check
const origin = process.env.E2E_BASE_URL ?? 'http://localhost:3000';

const browserZone = new Date().getUTCHours() >= 10 ? 'Pacific/Kiritimati' : 'Pacific/Pago_Pago';

/** Today in the browser's time zone (the test runner's is UTC), as YYYY-MM-DD. */
const browserToday = () => page.evaluate(() => new Date().toLocaleDateString('sv'));

test.beforeAll(async ({ browser }) => {
	// axe needs an explicit context
	page = await (await browser.newContext({ timezoneId: browserZone })).newPage();
	// A platform authenticator that always verifies the user, as Touch ID would.
	const cdp = await page.context().newCDPSession(page);
	await cdp.send('WebAuthn.enable');
	await cdp.send('WebAuthn.addVirtualAuthenticator', {
		options: {
			protocol: 'ctap2',
			transport: 'internal',
			hasResidentKey: true,
			hasUserVerification: true,
			isUserVerified: true,
			automaticPresenceSimulation: true
		}
	});
});

test('responses keep other sites out: no framing, opening, or embedding', async () => {
	for (const url of ['/', '/api/v1/openapi.yaml']) {
		const h = (await page.request.get(url)).headers();
		if (url === '/') expect(h['content-security-policy']).toContain("frame-ancestors 'none'");
		expect(h['cross-origin-opener-policy'], url).toBe('same-origin');
		expect(h['cross-origin-resource-policy'], url).toBe('same-origin');
	}
});

test('pages, scripts, and the API are sent compressed', async () => {
	const accept = { 'Accept-Encoding': 'gzip, zstd' };
	await page.goto('/');
	const script = await page.evaluate(
		() => (document.querySelector('link[rel="modulepreload"]') as HTMLLinkElement).href
	);
	for (const url of ['/', script, '/api/v1/openapi.yaml']) {
		const res = await page.request.get(url, { headers: accept });
		expect(res.ok(), url).toBe(true);
		expect(res.headers()['content-encoding'], url).toMatch(/^(gzip|zstd)$/);
	}
});

test('home page is accessible and offers registration', async () => {
	// Artwork and fonts ship with Axon: nothing is fetched from another origin.
	const foreign: string[] = [];
	page.on('request', (r) => {
		if (!r.url().startsWith(origin) && !r.url().startsWith('data:')) foreign.push(r.url());
	});
	await page.goto('/');
	await page.waitForLoadState('networkidle');
	expect(foreign).toEqual([]);
	await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
	await expect(page.getByRole('status').filter({ hasText: 'Server running' })).toBeVisible();
	// With the demo on, a teaser from Axon's own server, with captions (EP-84).
	const teaser = page.getByRole('region', { name: 'Axon in under a minute' });
	await expect(teaser.locator('video source')).toHaveCount(2);
	await expect(teaser.locator('video track')).toHaveCount(2);
	for (const src of ['/teaser/axon.webm', '/teaser/axon.mp4', '/teaser/axon.en.vtt'])
		expect((await page.request.get(src, { headers: { Range: 'bytes=0-99' } })).status(), src).toBe(
			206
		);
	await noSeriousA11yViolations(page);
	await page.getByRole('link', { name: 'Create your account' }).click();
});

test('register with a passkey and see recovery codes once', async () => {
	await noSeriousA11yViolations(page);
	await page.getByLabel('Your name').fill('E2E Owner');
	await page.getByRole('button', { name: 'Create account with a passkey' }).click();
	await expect(page.getByRole('heading', { name: 'Save your recovery codes' })).toBeVisible();
	await expect(
		page.getByRole('list', { name: 'Recovery codes' }).getByRole('listitem')
	).toHaveCount(10);
	await page.getByRole('link', { name: 'I saved them, continue' }).click();
	await expect(page).toHaveURL(/\/today$/);
	// A new account is guided to its first task, note, and goal.
	await expect(page.getByRole('heading', { name: 'Get started' })).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('a new account says what Axon is for, and the menu shows just that', async () => {
	const welcome = page.getByRole('region', { name: 'What do you want Axon for?' });
	await welcome.getByLabel('Tasks and projects').check();
	await welcome.getByLabel('Goals and habits').check();
	await noSeriousA11yViolations(page);
	await welcome.getByRole('button', { name: 'Set up my Axon' }).click();
	await expect(welcome).toBeHidden();
	await expect(page.getByRole('heading', { name: 'Calendar today' })).toBeVisible();
	await expect(page.locator('header nav a[href="/budget"]')).toHaveCount(0);
	await expect(page.locator('header nav a[href="/goals"]')).not.toHaveCount(0);
	// Preferences bring a page back, and saving them keeps Today as it was set.
	await page.goto('/account/preferences');
	const pages = page.getByRole('group', { name: 'Pages in the menu' });
	// all() does not wait: let the page load its settings before changing them.
	await page.waitForLoadState('networkidle');
	for (const box of await pages.getByRole('checkbox').all()) await box.check();
	await expect(pages.getByRole('checkbox', { checked: false })).toHaveCount(0);
	await page.getByRole('button', { name: 'Save preferences' }).click();
	await expect(page.getByText('Preferences saved.')).toBeVisible();
	await expect(page.locator('header nav a[href="/budget"]')).not.toHaveCount(0);
	await page.goto('/today');
	await expect(page.getByRole('heading', { name: 'Calendar today' })).toBeVisible();
	// Back to the default Today for the tests that follow.
	await page.evaluate(async () => {
		const doc = await (await fetch('/api/v1/me/settings')).json();
		delete doc.ai;
		doc.today = { sections: ['kpis', 'tasks', 'milestones', 'goals', 'notes', 'overview'] };
		await fetch('/api/v1/me/settings', {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(doc)
		});
	});
});

test('API answers are sent compressed', async () => {
	// The roadmap library: the same 12 kB for everyone (under 512 bytes is not worth compressing).
	const res = await page.request.get('/api/v1/roadmaps/sources', {
		headers: { 'Accept-Encoding': 'gzip, zstd' }
	});
	expect(res.ok()).toBe(true);
	expect(res.headers()['content-type']).toContain('application/json');
	expect(res.headers()['content-encoding']).toMatch(/^(gzip|zstd)$/);
});

test('the page title announced to screen readers is not shown on the page', async () => {
	await page.goto('/today');
	await page.getByRole('link', { name: 'Account', exact: true }).click();
	await expect(page).toHaveURL(/\/account/);
	const announcer = page.locator('#svelte-announcer');
	await expect(announcer).toHaveText(/Axon/); // still read out
	// SvelteKit hides it with an inline style, which the CSP (style-src 'self')
	// blocks in Firefox: without that style it must stay hidden.
	await announcer.evaluate((el) => el.removeAttribute('style'));
	const box = await announcer.boundingBox();
	expect(box === null || (box.width <= 1 && box.height <= 1)).toBe(true);
});

test('every account page is reachable by keyboard and accessible', async () => {
	await page.goto('/account');
	await expect(page.getByLabel('Your name')).toHaveValue('E2E Owner');
	for (const name of [
		'Sign-in & security',
		'Access tokens',
		'Privacy & data',
		'Activity',
		'Overview'
	]) {
		await page.getByRole('link', { name, exact: true }).focus();
		await page.keyboard.press('Enter');
		await expect(page.getByRole('link', { name, exact: true })).toHaveAttribute(
			'aria-current',
			'page'
		);
		await expect(page.getByRole('status').filter({ hasText: 'Loading' })).toHaveCount(0);
		await noSeriousA11yViolations(page);
	}
	await page.getByRole('link', { name: 'Activity', exact: true }).click();
	await expect(page.getByText('Account created')).toBeVisible();
});

test('layout fits a 360 px screen without horizontal scrolling', async () => {
	await page.setViewportSize({ width: 360, height: 740 });
	for (const path of [
		'/',
		'/account',
		'/account/security',
		'/account/tokens',
		'/account/privacy',
		'/tasks',
		'/notes',
		'/goals',
		'/today',
		'/calendar',
		'/agents',
		'/habits',
		'/projects',
		'/roadmaps',
		'/learn',
		'/learn/palaces',
		'/areas',
		'/resources',
		'/archive',
		'/budget',
		'/notes/explore',
		'/media',
		'/health',
		'/budget/pots',
		'/notes/analysis',
		'/insights',
		'/notes/templates'
	]) {
		await page.goto(path);
		await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
		const overflow = await page.evaluate(
			() => document.documentElement.scrollWidth - window.innerWidth
		);
		expect(overflow, path).toBeLessThanOrEqual(0);
	}
	await page.setViewportSize({ width: 1280, height: 800 });
});

test('reduced motion turns animations off', async () => {
	await page.emulateMedia({ reducedMotion: 'reduce' });
	await page.goto('/');
	const motion = await page.evaluate(() =>
		getComputedStyle(document.documentElement).getPropertyValue('--motion').trim()
	);
	expect(motion).toBe('0ms');
	await page.emulateMedia({ reducedMotion: 'no-preference' });
});

test('offline state is announced', async () => {
	await page.goto('/account');
	await page.context().setOffline(true);
	await expect(page.getByRole('alert').filter({ hasText: 'You are offline' })).toBeVisible();
	await page.context().setOffline(false);
	await expect(page.getByRole('alert').filter({ hasText: 'You are offline' })).toHaveCount(0);
});

test('capture with the n shortcut, plan, and complete tasks', async () => {
	await page.goto('/account');
	await expect(page.getByRole('heading', { level: 1 })).toBeVisible(); // shortcut is live
	await page.keyboard.press('n');
	await page.getByLabel("What's on your mind?").fill('Buy stamps');
	await page.keyboard.press('Enter');
	await expect(page.getByText('Captured to your inbox.')).toBeVisible();

	await page.goto('/tasks?view=inbox');
	await expect(page.getByRole('button', { name: 'Buy stamps' })).toBeVisible();
	await noSeriousA11yViolations(page);

	await page.goto('/tasks?view=today');
	await page.getByLabel('Add a task').fill('Stretch for ten minutes');
	await page.keyboard.press('Enter');
	const row = page.getByRole('listitem').filter({ hasText: 'Stretch for ten minutes' });
	await expect(row).toContainText('Today');
	await row.getByRole('checkbox').check();
	await expect(page.getByText('Done.')).toBeVisible();
	await page.goto('/tasks?view=done');
	await expect(page.getByRole('button', { name: 'Stretch for ten minutes' })).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('write notes, link them with wikilinks, and follow backlinks', async () => {
	await page.goto('/notes');
	await expect(page.getByRole('heading', { level: 1, name: 'Notes' })).toBeVisible(); // tasks are already mirrored as notes
	await page.getByLabel('New note title').fill('Spaced repetition');
	await page.keyboard.press('Enter');
	await expect(page).toHaveURL(/\/notes\/[0-9a-f-]+\?edit=1/);
	await page
		.getByLabel('Markdown')
		.fill('# Spaced repetition\n\nReview just before forgetting. See [[Forgetting curve]].');
	await expect(page.getByText('All changes saved')).toBeVisible(); // autosaved
	await noSeriousA11yViolations(page);

	await page.getByRole('button', { name: 'Create note' }).click();
	await expect(page.getByRole('link', { name: 'Forgetting curve' }).first()).toBeVisible();
	await page.getByRole('article').getByRole('link', { name: 'Forgetting curve' }).click();
	await expect(page.getByRole('heading', { level: 1, name: 'Forgetting curve' })).toBeVisible();
	const connections = page.getByRole('region', { name: 'Connections' });
	await expect(connections).toContainText('Mentioned in');
	await expect(connections.getByRole('link', { name: 'Spaced repetition' })).toBeVisible();
	await noSeriousA11yViolations(page);

	await page.goto('/notes?para=inbox');
	const sidebar = page.getByRole('navigation', { name: 'Folders' });
	await expect(sidebar.getByRole('link', { name: 'Spaced repetition' })).toBeVisible();
	await expect(sidebar.getByRole('link', { name: 'Forgetting curve' })).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('notes workspace: command center, daily note with [[ suggestions, templates, and search', async () => {
	await page.goto('/today');
	await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
	await page.keyboard.press('Control+k');
	await page.getByLabel('Search or type a command').fill('Forgetting');
	await expect(page.getByRole('option', { name: /^Forgetting curve/ })).toBeVisible();
	await page.keyboard.press('Enter');
	await expect(page.getByRole('heading', { level: 1, name: 'Forgetting curve' })).toBeVisible();

	// Today's daily note, linked to another note through the [[ suggestions.
	await page.getByRole('button', { name: "Today's note" }).click();
	await expect(page).toHaveURL(/edit=1/);
	const editor = page.getByLabel('Markdown');
	await editor.press('Control+End');
	await editor.pressSequentially('Studied [[Spaced');
	await expect(page.getByRole('option', { name: /Spaced repetition/ })).toBeVisible();
	await page.keyboard.press('Enter');
	await expect(editor).toHaveValue(/Studied \[\[spaced-repetition\]\]/);
	await expect(page.getByText('All changes saved')).toBeVisible();
	await noSeriousA11yViolations(page);
	await page.getByRole('button', { name: 'Read' }).click();
	await expect(
		page.getByRole('article').getByRole('link', { name: 'Spaced repetition' })
	).toBeVisible();

	// A permanent Zettelkasten note from its template.
	await page.getByRole('button', { name: '＋ New note' }).click();
	await page
		.getByRole('dialog', { name: 'New note' })
		.getByRole('textbox', { name: 'Title' })
		.fill('Atomic notes');
	await page.getByRole('radio', { name: /Permanent note/ }).check();
	await page.getByRole('button', { name: 'Create', exact: true }).click();
	await expect(page.getByText(/Zettel\/permanent\/\d{12}-atomic-notes\.md/)).toBeVisible();
	await expect(page.getByRole('heading', { level: 1, name: 'Atomic notes' })).toBeVisible();

	// Full-text search with highlighted excerpts.
	await page.getByLabel('Search notes').fill('forgetting');
	await page.keyboard.press('Enter');
	await expect(page.getByRole('heading', { level: 1, name: 'Search' })).toBeVisible();
	await expect(page.locator('mark').first()).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('set a goal, break it into milestones, link work, and see how progress is worked out', async () => {
	await page.goto('/goals');
	await expect(page.getByText('No active goals.')).toBeVisible();
	await page.getByLabel('New goal').fill('Learn to swim 1 km');
	await page.keyboard.press('Enter');
	await expect(page.getByRole('heading', { level: 1, name: 'Learn to swim 1 km' })).toBeVisible();
	await expect(page.getByText('No progress measure yet')).toBeVisible();

	await page.getByLabel('New milestone').fill('Swim 100 m');
	await page.getByRole('button', { name: 'Add', exact: true }).click();
	await expect(page.getByLabel('Done: Swim 100 m')).toBeVisible();
	await page.getByLabel('New milestone').fill('Swim 500 m');
	await page.getByRole('button', { name: 'Add', exact: true }).click();
	await page.getByLabel('Done: Swim 100 m').check();
	await expect(page.getByRole('progressbar', { name: 'Progress' })).toHaveAttribute(
		'aria-valuenow',
		'50'
	);

	await page.getByLabel('Connect to').fill('Buy');
	await page.getByRole('button', { name: /Buy stamps/ }).click();
	await page.getByRole('button', { name: 'Connect', exact: true }).click();
	await expect(page.getByRole('region', { name: 'Connections' })).toContainText('Supported by');
	await expect(page.getByRole('region', { name: 'Connections' })).toContainText('Buy stamps');
	await expect(page.getByText('1 of 3 milestones and linked tasks done')).toBeVisible();
	await page.getByText('How is this worked out?').click();
	await expect(page.getByRole('region', { name: 'Progress' }).getByRole('listitem')).toHaveCount(3);
	await noSeriousA11yViolations(page);

	await page.goto('/goals');
	await expect(page.getByRole('heading', { name: 'Weekly goal review' })).toBeVisible();
	await page
		.getByLabel("What moved forward, and what's next?")
		.fill('First lengths without stopping.');
	await page.getByRole('button', { name: 'Save review' }).click();
	await expect(page.getByText('Weekly review saved.')).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Weekly goal review' })).toHaveCount(0);
	await expect(page.getByRole('link', { name: 'Learn to swim 1 km' })).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('today shows defined numbers that open the items behind them', async () => {
	await page.goto('/');
	await expect(page).toHaveURL(/\/today$/);
	await expect(page.getByText('Get started')).toHaveCount(0); // a task, note, and goal exist
	const inbox = page.getByRole('link', { name: /Inbox/ });
	await expect(inbox).toContainText('1');
	await expect(page.getByText('Captured tasks you have not clarified yet.')).toBeVisible();
	await expect(
		page
			.getByRole('region', { name: 'Active goals' })
			.getByRole('link', { name: 'Learn to swim 1 km' })
	).toBeVisible();
	await noSeriousA11yViolations(page);
	await inbox.click();
	await expect(page).toHaveURL(/view=inbox/);
	await expect(page.getByRole('button', { name: 'Buy stamps' })).toBeVisible();
});

test('subscribe to a calendar feed, read it, and revoke it', async () => {
	await page.goto('/calendar');
	await expect(page.getByRole('heading', { level: 1, name: 'Calendar' })).toBeVisible();
	await page.getByLabel('Feed name').fill('Phone');
	await page.getByRole('button', { name: 'Create feed' }).click();
	const url = (await page.locator('.new-feed code').textContent()) ?? '';
	expect(url).toMatch(/\/api\/v1\/calendar-feeds\/.+\.ics$/);
	const feed = await page.request.get(url, { headers: { cookie: '' } });
	expect(feed.status()).toBe(200);
	expect(await feed.text()).toContain('BEGIN:VCALENDAR');
	await noSeriousA11yViolations(page);

	await page.getByRole('button', { name: 'Revoke Phone' }).click();
	await expect(page.getByText('Its URL no longer works.')).toBeVisible();
	expect((await page.request.get(url)).status()).toBe(404);
});

test('follow another calendar by its ICS address and see its events beside your own', async () => {
	const origin = new URL(page.url()).origin;
	const today = await browserToday();
	await page.request.post('/api/v1/tasks', {
		headers: { Origin: origin },
		data: { title: 'Dentist appointment', status: 'next', due_on: today }
	});
	await page.goto('/calendar');
	// The calendar to follow is Axon's own feed, reached the way the API reaches Caddy.
	await page.getByLabel('Feed name').fill('To follow');
	await page.getByRole('button', { name: 'Create feed' }).click();
	const feedURL = (await page.locator('.new-feed code').textContent()) ?? '';
	await page.getByRole('button', { name: 'Done' }).click();
	await page.getByLabel('Name', { exact: true }).fill('Shared');
	await page
		.getByLabel('Calendar address (iCal / ICS)')
		.fill(feedURL.replace(/^https?:\/\/[^/]+/, 'http://caddy:3000'));
	await page.getByRole('button', { name: 'Follow', exact: true }).click();
	await expect(page.getByText(/Following “Shared”: \d+ events\./)).toBeVisible();
	await expect(page.locator('li.ev.external', { hasText: 'Dentist appointment' })).toBeVisible();
	await expect(page.getByText('caddy ·')).toBeVisible();
	await noSeriousA11yViolations(page);
	await page.getByRole('checkbox', { name: 'Calendars you follow' }).uncheck();
	await expect(page.locator('li.ev.external')).toHaveCount(0);
	await page.getByRole('button', { name: 'Stop following Shared' }).click();
	await expect(page.getByText('No longer following “Shared”.')).toBeVisible();
	await page.getByRole('button', { name: 'Revoke To follow' }).click();
});

test('design a habit, check in with the two-minute version, and link it to a goal', async () => {
	await page.goto('/habits');
	await expect(page.getByText('No habits yet.')).toBeVisible();
	await page.getByLabel('Habit', { exact: true }).fill('Swim one length');
	await page.getByLabel('Two-minute version').fill('Pack my swim bag');
	await page.getByLabel('After…').fill('After breakfast');
	await page.getByLabel('Who do you want to become?').fill('I am a swimmer');
	await page.getByLabel('I will do it at').fill('07:30');
	await page.getByLabel('In or at').fill('at the pool');
	await expect(page.getByText('After breakfast, I will swim one length.')).toBeVisible();
	await page.getByRole('button', { name: 'Create habit' }).click();
	await expect(page.getByRole('heading', { level: 1, name: 'Swim one length' })).toBeVisible();
	await page.getByLabel('Connect to').fill('swim 1');
	await page.getByRole('button', { name: /Learn to swim 1 km/ }).click();
	await page.getByRole('button', { name: 'Connect', exact: true }).click();
	await expect(page.getByRole('region', { name: 'Connections' })).toContainText('Supports');
	await expect(page.getByRole('region', { name: 'Connections' })).toContainText(
		'Learn to swim 1 km'
	);
	await noSeriousA11yViolations(page);

	await page.goto('/habits');
	await expect(page.getByText('I will swim one length at 07:30 at the pool.')).toBeVisible();
	await page.getByRole('button', { name: 'Did the two-minute version: Pack my swim bag' }).click();
	await expect(page.getByText('Two-minute version today')).toBeVisible();
	await expect(page.getByText('1 vote for “I am a swimmer”')).toBeVisible();

	// The habits scorecard.
	await page.getByLabel('Add a behaviour').fill('Check my phone');
	await page.getByRole('button', { name: 'Add', exact: true }).click();
	await page.getByRole('button', { name: 'bad: Check my phone' }).click();
	await expect(page.getByRole('button', { name: 'bad: Check my phone' })).toHaveAttribute(
		'aria-pressed',
		'true'
	);
	await noSeriousA11yViolations(page);
	await page.getByRole('button', { name: "Undo today's check-in for Swim one length" }).click();
	await expect(page.getByRole('button', { name: 'Done: Swim one length' })).toBeVisible();
});

test('connect a task to a note by keyboard alone, and see it from both ends', async () => {
	await page.goto('/tasks?view=inbox');
	await page.getByRole('button', { name: 'Buy stamps' }).click();
	await page.getByRole('link', { name: 'Open page and connections' }).click();
	await expect(page.getByRole('heading', { level: 1, name: 'Buy stamps' })).toBeVisible();
	await page.getByLabel('Connect to').focus();
	await page.keyboard.type('Forgetting');
	await expect(page.getByRole('button', { name: /Forgetting curve/ })).toBeVisible();
	await page.keyboard.press('Tab');
	await page.keyboard.press('Enter');
	await page.getByLabel('How they connect').focus();
	await page.keyboard.press('Tab');
	await page.keyboard.press('Enter');
	const here = page.getByRole('region', { name: 'Connections' });
	await expect(here.getByRole('link', { name: 'Forgetting curve' })).toBeVisible();
	await expect(here).toContainText('Supports'); // the goal link made earlier
	await noSeriousA11yViolations(page);
	await here.getByRole('link', { name: 'Forgetting curve' }).click();
	await expect(
		page.getByRole('region', { name: 'Connections' }).getByRole('link', { name: 'Buy stamps' })
	).toBeVisible();
});

test('run a project on a Kanban board by keyboard, with a WIP limit that asks first', async () => {
	await page.goto('/projects');
	await page.getByLabel('New project').fill('Garden shed');
	await page.keyboard.press('Enter');
	await expect(page.getByRole('heading', { level: 1, name: 'Garden shed' })).toBeVisible();
	await page.getByLabel('WIP limit for Doing').fill('1');
	await page.getByRole('button', { name: 'Save', exact: true }).click();
	await expect(page.getByTitle('WIP limit')).toHaveText('0 of 1');

	await page.getByLabel('Add a card to To do').fill('Buy timber');
	await page.keyboard.press('Enter');
	await expect(page.getByRole('link', { name: 'Buy timber' })).toBeVisible();
	await page.getByLabel('Add a card to To do').fill('Pour the base');
	await page.keyboard.press('Enter');
	await expect(page.getByRole('link', { name: 'Pour the base' })).toBeVisible();
	await noSeriousA11yViolations(page);

	// Keyboard: Shift+→ moves a card into Doing; Shift+↑ reorders.
	await page.getByRole('link', { name: 'Buy timber' }).focus();
	await page.keyboard.press('Shift+ArrowRight');
	await expect(
		page.getByRole('status').filter({ hasText: 'Buy timber is in Doing' })
	).toBeAttached();
	await expect(page.getByTitle('WIP limit')).toHaveText('1 of 1');
	await expect(page.getByRole('link', { name: 'Buy timber' })).toBeFocused();

	// Past the limit, Axon asks; accepting moves the card anyway.
	page.once('dialog', (d) => d.accept());
	await page.getByLabel('Move Pour the base to').selectOption('doing');
	await expect(page.getByTitle('WIP limit')).toHaveText('2 of 1');

	// Done on the board completes the task.
	await page.getByRole('link', { name: 'Buy timber' }).focus();
	await page.keyboard.press('Shift+ArrowRight');
	await expect(page.getByRole('region', { name: 'Done' })).toContainText('Buy timber');

	// A project doc is a note in the project's folder.
	page.once('dialog', (d) => d.accept('Shed plan'));
	await page.getByRole('button', { name: 'New doc' }).click();
	await expect(page.getByText('1-Projects/garden-shed/shed-plan.md')).toBeVisible();
	await page.goBack();
	await expect(page.getByRole('region', { name: 'Docs' })).toContainText('Shed plan');
});

test('take the skills assessment, then start the roadmap at your level', async () => {
	await page.goto('/roadmaps');
	// The library is one roadmap, with its plan (EP-95).
	const library = page.getByRole('region', { name: 'Top-5% in 26 engineering domains' });
	await expect(library).toContainText('14 phases');
	await expect(library.getByRole('button', { name: 'Start this roadmap' })).toBeVisible();
	await page.getByRole('link', { name: 'Take the assessment' }).click();
	await page.getByLabel('Your role or situation').fill('Data engineer');
	await page.getByLabel('What do you want to achieve?').fill('Top 5% in 26 engineering domains');
	await page.getByLabel('Practice and projects').check();
	await page.getByRole('button', { name: 'Next' }).click();
	await page.getByLabel('Python', { exact: true }).check();
	await page.getByRole('button', { name: 'Next' }).click();
	const python = page.getByRole('region', { name: 'Python' });
	await python.getByLabel('How well do you know it?').selectOption('5');
	await python.getByLabel('[0, 2, 4]').check();
	await python.getByLabel('tuple').check();
	await noSeriousA11yViolations(page);
	await page.getByRole('button', { name: 'See my results' }).click();
	await expect(
		page.getByRole('cell', { name: /You rated it 5\/5 and answered 2 of 3/ })
	).toBeVisible();
	await noSeriousA11yViolations(page);
	await page.getByRole('button', { name: 'Start the roadmap at my level' }).click();
	await expect(
		page.getByRole('heading', { level: 1, name: 'Top-5% in 26 engineering domains' })
	).toBeVisible();
	await expect(page.getByText(/Placed by your skills assessment: “Python”/)).toBeVisible();
	await expect(page.getByLabel('Status of Python')).toHaveValue('learning');
});

test('the roadmap is a whole plan: skills with resources, a goal, milestones, projects, and habits', async () => {
	await page.goto('/roadmaps');
	await page.getByRole('link', { name: 'Open the roadmap' }).click();
	await page.getByRole('link', { name: 'Skill graph' }).click();
	const graph = page.getByRole('group', { name: /Skill graph: 32 skills/ });
	await graph.getByRole('link', { name: 'Operating systems, todo' }).click();
	const detail = page.getByRole('complementary', { name: 'Operating systems' });
	await expect(detail).toContainText('Needs first: C and C++');
	await expect(
		detail.getByRole('link', { name: /Operating Systems: Three Easy Pieces/ })
	).toBeVisible();
	await detail.getByRole('button', { name: 'Learning' }).click();
	await expect(graph.getByRole('link', { name: 'Operating systems, learning' })).toBeVisible();
	await noSeriousA11yViolations(page);

	await page.goto('/goals');
	await expect(page.getByText('Top-5% in 26 engineering domains').first()).toBeVisible();
	await page.goto('/projects');
	await expect(
		page.getByRole('link', { name: 'pipekit: mini orchestrator in Python' })
	).toBeVisible();
	await expect(page.getByRole('link', { name: '42 cursus (self-taught)' })).toBeVisible();
	await page.goto('/habits');
	await expect(page.getByText('Daily study block').first()).toBeVisible();

	// The weekly rhythm is on the calendar, each block opening the roadmap (EP-97).
	const month = new Date().toISOString().slice(0, 7);
	await page.goto(`/calendar?types=study&month=${month}`);
	const blocks = page.locator('.grid .ev.study');
	if ((await blocks.count()) === 0) {
		const [y, m] = month.split('-').map(Number);
		await page.goto(
			`/calendar?types=study&month=${new Date(Date.UTC(y, m, 1)).toISOString().slice(0, 7)}`
		);
	}
	await expect(blocks.first()).toBeVisible();
	await expect(page.getByRole('link', { name: /^Maths: .* · 2–3 h$/ }).first()).toHaveAttribute(
		'href',
		/^\/roadmaps\//
	);
	await noSeriousA11yViolations(page);
});

test('tasks open on a GTD board, and a Kanban board moves work along', async () => {
	await page.goto('/tasks');
	const gtd = page.getByRole('group', { name: 'GTD board' });
	await expect(gtd.getByRole('heading', { name: /Next actions/ })).toBeVisible();
	await page.getByLabel('Add a task').fill('Plan the garden');
	await page.keyboard.press('Enter');
	const card = gtd.getByRole('listitem').filter({ hasText: 'Plan the garden' });
	await expect(card).toBeVisible(); // captured to the inbox column
	await card.getByLabel('Move Plan the garden to').selectOption('next');
	await expect(page.getByText('Moved “Plan the garden” to Next actions.')).toBeVisible();
	await noSeriousA11yViolations(page);

	await page.getByRole('link', { name: 'Kanban' }).click();
	const kanban = page.getByRole('group', { name: 'Kanban board' });
	await kanban.getByRole('button', { name: 'Plan the garden' }).focus();
	await page.keyboard.press('Shift+ArrowRight');
	await expect(
		kanban
			.getByRole('region', { name: /In progress/ })
			.getByRole('button', { name: 'Plan the garden' })
	).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('the calendar opens on a month grid, with the agenda one click away', async () => {
	await page.goto('/calendar');
	await expect(page.getByRole('table')).toBeVisible();
	await expect(page.getByRole('columnheader')).toHaveCount(7);
	await noSeriousA11yViolations(page);
	await page.getByRole('link', { name: 'Agenda' }).click();
	await expect(page).toHaveURL(/layout=agenda/);
	await expect(page.getByRole('table')).toHaveCount(0);
});

test('areas and resources: start from a template, archive, and restore', async () => {
	await page.goto('/areas');
	await page.getByRole('button', { name: '＋ Health' }).click();
	await expect(page.getByRole('heading', { level: 1, name: 'Health' })).toBeVisible();
	await expect(page.getByLabel('Standard to keep')).toHaveValue(/sleep 7–8 h/);
	await noSeriousA11yViolations(page);
	await page.getByRole('button', { name: 'Archive' }).click();
	await expect(page.getByText('Archived. Find it in the Archive.')).toBeVisible();
	await page.goto('/archive');
	await page.locator('#main').getByRole('link', { name: 'Health' }).click();
	await page.getByRole('button', { name: 'Restore' }).click();
	await expect(page.getByText('Restored.')).toBeVisible();

	await page.goto('/resources');
	await page.getByLabel('New resource').fill('Rust');
	await page.keyboard.press('Enter');
	await expect(page.getByRole('heading', { level: 1, name: 'Rust' })).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('goals, habits, and projects start from templates', async () => {
	await page.goto('/goals');
	await page.getByRole('button', { name: '＋ Read 12 books this year' }).click();
	await expect(
		page.getByRole('heading', { level: 1, name: 'Read 12 books this year' })
	).toBeVisible();
	await expect(page.getByText('12 books', { exact: true })).toBeVisible(); // a dated milestone
	await page.getByRole('button', { name: '＋ Plan, build, review' }).click();
	await expect(page.getByText('Plan written')).toBeVisible();

	await page.goto('/habits');
	await page.getByRole('button', { name: '＋ Meditate' }).click();
	await expect(page.getByRole('textbox', { name: 'Two-minute version' })).toHaveValue(
		'One mindful breath'
	);

	await page.goto('/projects');
	await page.getByRole('button', { name: '＋ Writing project' }).click();
	await expect(page.getByRole('heading', { level: 1, name: 'Writing project' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'First draft' })).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('note templates live in the vault and fill their placeholders', async () => {
	await page.goto('/notes');
	await page.getByRole('button', { name: /New note/ }).click();
	const dialog = page.getByRole('dialog', { name: 'New note' });
	await dialog.getByRole('button', { name: 'copy these built-in ones there' }).click();
	await expect(page.getByText(/The templates are in Templates\//)).toBeVisible();
	await dialog.getByLabel('Title', { exact: true }).fill('Raft consensus');
	// The vault's own templates (EP-100), copied as they are: Templates/literature.md.
	await dialog.getByRole('radio', { name: /^literature .*Templates\/literature\.md/ }).check();
	await dialog.getByRole('button', { name: 'Create' }).click();
	await expect(page).toHaveURL(/\/notes\/[0-9a-f-]+\?edit=1/);
	await expect(page.getByRole('textbox', { name: /Markdown/ })).toHaveValue(
		/title: Raft consensus\ncreated: \d{4}-\d{2}-\d{2} \d{2}:\d{2}\n/
	);
	await expect(page.getByText(/Zettel\/literature\/\d{12}-raft-consensus\.md/)).toBeVisible();
});

test('budget: categories, a monthly budget, and exact totals', async () => {
	await page.goto('/budget');
	await page.getByRole('button', { name: 'Start with common categories' }).click();
	await page.getByRole('button', { name: 'Edit Groceries' }).click();
	const edit = page.getByRole('dialog', { name: 'Edit category' });
	await edit.getByLabel(/Monthly budget/).fill('400');
	await edit.getByRole('button', { name: 'Save' }).click();
	await expect(edit).toBeHidden();
	await expect(page.getByRole('progressbar', { name: /Groceries: 0 % of budget/ })).toBeVisible();
	await page.getByLabel('Amount', { exact: true }).fill('0,29');
	await page.getByLabel('Category', { exact: true }).selectOption({ label: 'Groceries' });
	await page.getByLabel('Payee', { exact: true }).fill('Bakery');
	await page.getByRole('button', { name: 'Record' }).click();
	await expect(page.getByRole('cell', { name: 'Bakery', exact: true })).toBeVisible();
	await page.getByLabel('Amount', { exact: true }).fill('12.71');
	await page.getByLabel('Category', { exact: true }).selectOption({ label: 'Groceries' });
	await page.getByRole('button', { name: 'Record' }).click();
	await expect(page.getByLabel('Amount', { exact: true })).toHaveValue('');
	const spent = page.getByRole('region', { name: 'This month' });
	await expect(spent).toContainText(/13[.,]00/); // 0.29 + 12.71, with no float drift
	await expect(page.getByRole('progressbar', { name: /Groceries: 3 % of budget/ })).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('finance: accounts, a transfer that is not spending, and net worth in reports', async () => {
	await page.goto('/budget/accounts');
	for (const [name, opening] of [
		['Main account', '1000'],
		['Savings', '0']
	]) {
		await page.getByLabel('Name', { exact: true }).fill(name);
		await page.getByLabel('Balance today').fill(opening);
		await page.getByRole('button', { name: 'Add account' }).click();
		await expect(page.getByRole('cell', { name, exact: true })).toBeVisible();
	}
	await page
		.getByRole('navigation', { name: 'Finance' })
		.getByRole('link', { name: 'Overview' })
		.click();
	await page.getByRole('radio', { name: 'Transfer' }).check();
	await page.getByLabel('Amount', { exact: true }).fill('250');
	await page.getByLabel('From account').selectOption({ label: 'Main account' });
	await page.getByLabel('To account').selectOption({ label: 'Savings' });
	await page.getByRole('button', { name: 'Record' }).click();
	await expect(page.getByLabel('Amount', { exact: true })).toHaveValue('');
	await expect(page.getByText('⇄')).toBeVisible();
	await page
		.getByRole('navigation', { name: 'Finance' })
		.getByRole('link', { name: 'Accounts' })
		.click();
	await expect(page.getByRole('heading', { name: 'Accounts', level: 1 })).toBeVisible();
	const savings = page.getByRole('row', { name: /^Savings/ });
	await expect(savings).toContainText(/250[.,]00/);
	await expect(page.getByRole('heading', { name: /Net worth/ })).toContainText(
		/1[\s,. ]?000[.,]00/
	);
	await page
		.getByRole('navigation', { name: 'Finance' })
		.getByRole('link', { name: 'Reports' })
		.click();
	await expect(page.getByText('Net worth today')).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('themed spaces keep AA contrast in light and dark, and can be turned off', async () => {
	test.slow(); // 24 pages, each checked by axe
	for (const scheme of ['light', 'dark'] as const) {
		await page.emulateMedia({ colorScheme: scheme });
		for (const [path, space] of [
			['/notes', 'notebook'],
			['/roadmaps', 'library'],
			['/projects', 'blueprint'],
			['/habits', 'garden'],
			['/goals', 'summit'],
			['/calendar', 'almanac'],
			['/budget', 'ledger'],
			['/areas', 'atlas'],
			['/tasks', 'desk'],
			['/cards', 'library'],
			['/focus', 'desk'],
			['/progress', 'summit']
		]) {
			await page.goto(path);
			await expect(page.locator('html')).toHaveAttribute('data-space', space);
			await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
			await noSeriousA11yViolations(page);
		}
	}
	await page.emulateMedia({ colorScheme: 'light' });
	await page.goto('/account/preferences');
	await page.getByLabel('Themed spaces').selectOption('off');
	await page.getByRole('button', { name: 'Save preferences' }).click();
	await page.goto('/notes');
	await expect(page.locator('html')).toHaveAttribute('data-spaces', 'off');
	await expect(page.locator('.space-art')).toBeHidden();
	await page.goto('/account/preferences');
	await page.getByLabel('Themed spaces').selectOption('on');
	await page.getByRole('button', { name: 'Save preferences' }).click();
});

test('explore notes with filters, charts, and a knowledge graph that says why', async () => {
	await page.goto('/notes/explore');
	await expect(
		page.getByRole('heading', { level: 1, name: 'Explore your knowledge' })
	).toBeVisible();
	const graph = page.getByRole('group', { name: /Knowledge graph: \d+ notes/ });
	await expect(graph).toBeVisible();
	await noSeriousA11yViolations(page);
	await page.getByLabel('PARA', { exact: true }).selectOption('other');
	await expect(page).toHaveURL(/para=other/);
	await expect(page.getByRole('region', { name: 'Summary' })).toBeVisible();
});

test('track films by hand, rate them, and see recommendations that say why', async () => {
	await page.goto('/media');
	await expect(
		page.getByRole('heading', { level: 1, name: 'Films, series, and books' })
	).toBeVisible();
	await page.getByLabel('Kind', { exact: true }).selectOption('film');
	for (const title of ['Arrival', 'Dune: Part One']) {
		await page.getByLabel('Title', { exact: true }).fill(title);
		await page.getByRole('button', { name: 'Add', exact: true }).click();
		await page.getByRole('button', { name: `Edit ${title}` }).click();
		const dialog = page.getByRole('dialog', { name: title });
		await dialog.getByLabel('By', { exact: true }).fill('Denis Villeneuve');
		if (title === 'Arrival') {
			await dialog.getByLabel('Status').selectOption('done');
			await dialog.getByLabel('Rating').selectOption('9');
		}
		await dialog.getByRole('button', { name: 'Save' }).click();
		await expect(dialog).toBeHidden();
	}
	await expect(page.getByRole('region', { name: 'Up next' })).toContainText(
		'By Denis Villeneuve, like Arrival, which you rated 9/10.'
	);
	await expect(page.getByText('Rated 9/10', { exact: true })).toBeVisible();
	// Without a cover, a generated one carries the title; the rating sits on the poster.
	const arrival = page.locator('.item', { hasText: 'Arrival' }).first();
	await expect(arrival.locator('.generated')).toContainText('Arrival');
	await expect(arrival.locator('.badge.rating')).toContainText('9');
	// Finding covers sends titles away, so it waits for lookups to be allowed.
	await expect(page.getByRole('button', { name: 'Find missing covers' })).toHaveCount(0);
	await noSeriousA11yViolations(page);
});

test('health logging waits for consent, then logs food and a workout', async () => {
	await page.goto('/health');
	await expect(page.getByRole('heading', { name: 'Health logging is off' })).toBeVisible();
	await noSeriousA11yViolations(page);
	await page.getByRole('button', { name: 'Turn on health logging' }).click();
	await page.getByLabel('What you ate').fill('Lentil soup');
	await page.getByLabel('kcal').fill('320,5');
	await page.getByRole('button', { name: 'Log it' }).click();
	await expect(page.getByRole('region', { name: 'This day' })).toContainText('320.5');
	await expect(page.getByText('Lentil soup · 320.5 kcal')).toBeVisible();
	await noSeriousA11yViolations(page);

	// Daily needs (EP-88): the warning first, then numbers worked out from a profile.
	const needs = page.getByRole('region', { name: 'Your daily needs' });
	await expect(needs.getByRole('note')).toContainText('registered dietitian');
	await needs.getByLabel('Birth year').fill(String(new Date().getFullYear() - 30));
	await needs.getByLabel('Sex, for the energy equation').selectOption('male');
	await needs.getByLabel('Height (cm)').fill('180');
	await needs.getByLabel('Weight (kg)').fill('80');
	await needs.getByLabel('Activity').selectOption('moderate');
	await needs.getByRole('button', { name: 'Work out my needs' }).click();
	await expect(needs).toContainText('2760');
	await expect(needs.getByRole('progressbar', { name: 'Energy' })).toBeVisible();
	await noSeriousA11yViolations(page);

	// A photo of a meal (EP-89): nothing is sent before the person allows it.
	const sent: string[] = [];
	page.on('request', (r) => r.url().includes('/food/estimate') && sent.push(r.url()));
	const onePixel = (
		globalThis as unknown as { Buffer: { from(s: string, encoding: string): never } }
	).Buffer.from(
		'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==',
		'base64'
	);
	await page
		.locator('label', { hasText: 'Photo of a meal' })
		.locator('input[type=file]')
		.setInputFiles({ name: 'meal.png', mimeType: 'image/png', buffer: onePixel });
	const ask = page.getByRole('group', { name: 'Before the photo is sent' });
	await expect(ask).toContainText('The photo is not kept');
	expect(sent).toEqual([]);
	await noSeriousA11yViolations(page);
	await ask.getByRole('button', { name: 'Cancel' }).click();

	// Meal plans (EP-90): the form hands its request to the Chef & Nutritionist.
	await page.getByText('Plan my meals with the chef').click();
	await page.getByLabel('People').fill('2');
	await page.getByLabel('vegetarian').check();
	await page.getByRole('button', { name: 'Ask the chef for a plan' }).click();
	await expect(page.getByText('Chef & Nutritionist').first()).toBeVisible();
	await noSeriousA11yViolations(page);

	await page.getByRole('link', { name: 'Training' }).click();
	await page.getByLabel('Title').fill('Full body');
	await page.getByLabel('Minutes').fill('40');
	await page.getByLabel('Exercise 1').fill('Squat');
	await page.getByRole('button', { name: 'Save workout' }).click();
	await expect(page.getByText('Full body')).toBeVisible();
	await expect(page.getByRole('region', { name: 'Last seven days' })).toContainText('40');
	await noSeriousA11yViolations(page);
});

test('the journal: how the day felt, the night, and what you are grateful for', async () => {
	await page.goto('/journal');
	const entry = page.getByRole('form', { name: 'Today' });
	// The faces are what people click; the radios underneath carry the names.
	const mood = entry.getByRole('group', { name: 'Mood' });
	await mood.locator('label', { hasText: 'Good' }).click();
	await expect(mood.getByRole('radio', { name: 'Good' })).toBeChecked();
	const energy = entry.getByRole('group', { name: 'Energy' });
	await energy.locator('label', { hasText: 'Tired' }).click();
	await expect(energy.getByRole('radio', { name: 'Tired' })).toBeChecked();
	await entry.getByLabel('Hours of sleep last night').fill('7.5');
	await entry.getByLabel('Grateful for, 1 of 3').fill('A call with Sam');
	await entry.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByText('Saved.')).toBeVisible();
	const month = page.getByRole('region', { name: 'The last 30 days' });
	await expect(month).toContainText('7.5 h of sleep');
	await expect(month).toContainText('A call with Sam');
	// One day is too few to say what helps: the page says how many more it needs.
	await page.reload();
	await expect(page.getByRole('region', { name: 'What helps you' })).toContainText(
		/Write in your journal on \d+ more days/
	);
	await noSeriousA11yViolations(page);
});

test('savings pots: put money aside for a later purchase', async () => {
	await page.goto('/budget/pots');
	await page.getByLabel('What it is for').fill('New bike');
	await page.getByLabel('Target').fill('1200');
	await page.getByRole('button', { name: 'Start the pot' }).click();
	await page.getByLabel('Amount for New bike').fill('150,50');
	await page.getByRole('button', { name: 'Put money in New bike' }).click();
	await expect(page.getByRole('progressbar', { name: 'New bike: 13 % saved' })).toBeVisible();
	await page.getByLabel('Amount for New bike').fill('500');
	await page.getByRole('button', { name: 'Take money out of New bike' }).click();
	await expect(page.getByText('There is not that much in the pot.')).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('analyse literature and permanent notes: topics, concepts, and a note panel', async () => {
	await page.goto('/notes');
	const first: string = await page.evaluate(async () => {
		let id = '';
		for (const [path, kind, body] of [
			[
				'Zettel/permanent/interleaving.md',
				'permanent',
				'Interleaving practice mixes problem types. Interleaving practice beats blocked practice.'
			],
			[
				'Zettel/literature/peak.md',
				'literature',
				'Deliberate practice and interleaving practice build skill faster than blocked practice.'
			]
		]) {
			const res = await fetch('/api/v1/notes', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					path,
					content: `---\ntype: ${kind}\n---\n# ${kind} note\n\n${body}\n`
				})
			});
			if (!res.ok) throw new Error(`create note: ${res.status}`);
			id ||= (await res.json()).id;
		}
		return id;
	});
	await page.goto('/notes/analysis');
	await expect(
		page.getByRole('heading', { level: 1, name: /Analysis of your literature/ })
	).toBeVisible();
	await expect(page.getByRole('region', { name: 'Summary' })).toContainText('notes');
	await expect(page.getByRole('heading', { name: 'Concept graph' })).toBeVisible();
	await noSeriousA11yViolations(page);
	await page.goto(`/notes/${first}`);
	await page.getByText('Analysis', { exact: true }).click();
	await expect(page.getByRole('list', { name: 'Key phrases' })).toContainText(
		'interleaving practice'
	);
	await noSeriousA11yViolations(page);
});

test('insights: weekly metrics that explain themselves, and the weekly review', async () => {
	await page.goto('/insights');
	const done = page
		.getByRole('listitem')
		.filter({ has: page.getByRole('heading', { name: 'Tasks completed' }) });
	await expect(done).toBeVisible();
	await done.getByText('How it is counted').click();
	await expect(done).toContainText('Tasks whose completion time falls in the week.');
	await expect(page.getByRole('heading', { name: /Weekly review · \d{4}-W\d{2}/ })).toBeVisible();
	await expect(page.getByRole('table')).toContainText('Tasks completed');
	await noSeriousA11yViolations(page);
	await page.getByRole('link', { name: /^← \d{4}-W\d{2}$/ }).click();
	await expect(page).toHaveURL(/week=\d{4}-W\d{2}/);
	await expect(page.getByRole('link', { name: /^\d{4}-W\d{2} →$/ })).toBeVisible();
});

test('the Notes page shows a knowledge graph of literature and permanent notes that says why', async () => {
	await page.goto('/notes');
	const graph = page.getByRole('region', { name: 'Knowledge graph' });
	await expect(graph.getByLabel('Graph summary')).toContainText(
		/\d+ notes · \d+ connections · \d+ communities/
	);
	await noSeriousA11yViolations(page);
	const note = graph
		.getByRole('button', { name: /interleaving|peak|literature note|permanent note/i })
		.first();
	await note.focus();
	await page.keyboard.press('Enter');
	await expect(graph.getByRole('heading', { name: 'Why it connects' })).toBeVisible();
	await graph.getByText('Show the connections as a table').click();
	await expect(graph.getByRole('table')).toBeVisible();
});

test('templates show their sections, and a task note in the vault becomes a task', async () => {
	await page.goto('/notes/templates');
	await expect(page.getByRole('heading', { level: 1, name: 'Templates' })).toBeVisible();
	await expect(page.getByRole('list', { name: 'Sections of Literature note' })).toContainText(
		'Key ideas'
	);
	await expect(page.getByRole('list', { name: 'Sections of Five Ws test' })).toContainText(
		'My answers'
	);
	await expect(page.getByRole('list', { name: 'Sections of Habit', exact: true })).toContainText(
		'Make it easy (two-minute version)'
	);
	await noSeriousA11yViolations(page);
	await page.getByRole('link', { name: 'Use the Task template' }).click();
	const dialog = page.getByRole('dialog', { name: 'New note' });
	await expect(dialog.getByRole('radio', { name: /^Task/ })).toBeChecked();
	await expect(dialog).toContainText('Sections: Notes');
	await dialog.getByLabel('Title', { exact: true }).fill('Water the plants from the vault');
	await dialog.getByRole('button', { name: 'Create' }).click();
	await expect(page).toHaveURL(/\/notes\/[0-9a-f-]+\?edit=1/);
	// The vault mirror turns the note into a task within a few seconds.
	await expect
		.poll(
			async () => {
				await page.goto('/tasks?view=inbox');
				return page.getByText('Water the plants from the vault').count();
			},
			{ timeout: 20000 }
		)
		.toBeGreaterThan(0);
});

test('paste a picture into a note: it is uploaded, embedded, and shown', async () => {
	const res = await page.request.post('/api/v1/notes', {
		headers: { Origin: new URL(page.url()).origin },
		data: { path: 'Zettel/Whiteboard.md', content: '# Whiteboard\n\n' }
	});
	const note = await res.json();
	await page.goto(`/notes/${note.id}?edit=1`);
	const editor = page.getByLabel('Markdown');
	await editor.click();
	await editor.press('End');
	await editor.evaluate(async (area: HTMLTextAreaElement) => {
		const c = document.createElement('canvas');
		c.width = 80;
		c.height = 40;
		c.getContext('2d')!.fillRect(0, 0, 40, 40);
		const blob: Blob = await new Promise((r) => c.toBlob((b) => r(b!), 'image/png'));
		const data = new DataTransfer();
		data.items.add(new File([blob], 'image.png', { type: 'image/png' }));
		area.dispatchEvent(
			new ClipboardEvent('paste', { clipboardData: data, bubbles: true, cancelable: true })
		);
	});
	await expect(editor).toHaveValue(/!\[image\]\(Attachments\/image-[0-9a-f]{8}\.png\)/);
	const img = page.locator('article.markdown img.embed');
	await expect(img).toBeVisible();
	await expect.poll(() => img.evaluate((i: HTMLImageElement) => i.naturalWidth)).toBe(80);
	await expect(page.getByText('All changes saved')).toBeVisible({ timeout: 10000 });
	await page.getByRole('button', { name: 'Read' }).click();
	await expect(page.locator('article.reading img.embed')).toHaveAttribute('alt', 'image');
	await noSeriousA11yViolations(page);
});

test('the menu groups pages: Do, Think, Grow, Life, Reflect', async () => {
	await page.goto('/today');
	const nav = page.getByRole('navigation', { name: 'Main' });
	await expect(nav.getByRole('link', { name: 'Today' })).toBeVisible();
	await expect(nav.getByRole('link', { name: 'Tasks' })).toBeHidden();
	const doGroup = nav.getByRole('button', { name: 'Do' });
	await doGroup.click();
	await expect(doGroup).toHaveAttribute('aria-expanded', 'true');
	await noSeriousA11yViolations(page);
	await page.keyboard.press('Escape');
	await expect(doGroup).toHaveAttribute('aria-expanded', 'false');
	await nav.getByRole('button', { name: 'Life' }).click();
	await nav.getByRole('link', { name: 'Budget' }).click();
	await expect(page).toHaveURL(/\/budget/);
	await expect(nav.getByRole('button', { name: 'Life (current section)' })).toBeVisible();
	await expect(nav.getByRole('link', { name: 'Budget' })).toBeHidden();
	// A phone gets one Menu button with every group.
	await page.setViewportSize({ width: 390, height: 844 });
	await nav.getByRole('button', { name: 'Menu' }).click();
	await expect(nav.getByRole('heading', { name: 'Grow' })).toBeVisible();
	await noSeriousA11yViolations(page);
	await nav.getByRole('link', { name: 'Habits' }).click();
	await expect(page).toHaveURL(/\/habits/);
	await page.setViewportSize({ width: 1280, height: 720 });
});

test('installable, and a capture made offline is kept on the device and sent later', async () => {
	await page.goto('/today');
	const manifest = await (await page.request.get('/manifest.webmanifest')).json();
	expect(manifest.start_url).toBe('/today');
	expect(manifest.icons.some((i: { sizes: string }) => i.sizes === '512x512')).toBe(true);
	await page.evaluate(() => navigator.serviceWorker.ready.then(() => true));
	await page.reload(); // now under the service worker
	await expect(page.locator('#today-title')).toBeVisible(); // loaded before the connection drops
	// The app is kept for offline use; the API reference (1.8 MB) is not.
	const cached = await page.evaluate(async () => {
		const out: string[] = [];
		for (const name of await caches.keys())
			for (const r of await (await caches.open(name)).keys()) out.push(new URL(r.url).pathname);
		return out;
	});
	expect(cached).toContain('/offline');
	expect(cached.filter((p) => p.startsWith('/api-docs/'))).toEqual([]);

	await page.context().setOffline(true);
	await page.keyboard.press('n');
	await page.getByLabel("What's on your mind?").fill('Buy batteries while offline');
	await page.getByRole('button', { name: 'Capture task' }).click();
	await expect(page.getByText('No connection: kept on this device')).toBeVisible();
	await page.goto('/tasks');
	await expect(page.getByRole('heading', { level: 1, name: 'You are offline' })).toBeVisible();
	await expect(page.getByText('Task: Buy batteries while offline')).toBeVisible();
	await noSeriousA11yViolations(page);

	await page.context().setOffline(false);
	await page.goto('/today');
	await expect(page.getByText('1 capture kept offline was sent.')).toBeVisible();
	await page.goto('/tasks?view=inbox');
	await expect(page.getByText('Buy batteries while offline')).toBeVisible();
});

test('reminders: times saved, and a test says when no device can receive it', async () => {
	await page.goto('/account/preferences');
	const section = page.getByRole('region', { name: 'Reminders' });
	await section.getByLabel('Send me a morning digest and an evening nudge').check();
	await section.getByLabel('Morning digest at').fill('07:30');
	await page.getByRole('button', { name: 'Save preferences' }).click();
	await expect(page.getByText('Preferences saved.')).toBeVisible();
	await page.reload();
	await expect(
		page.getByRole('region', { name: 'Reminders' }).getByLabel('Morning digest at')
	).toHaveValue('07:30');
	await page
		.getByRole('region', { name: 'Reminders' })
		.getByRole('button', { name: 'Send a test' })
		.click();
	await expect(page.getByText('No device or email received it')).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('quick add: one line becomes a task with a context and a day, a book, a habit, or a card', async () => {
	await page.goto('/today');
	await expect(page.locator('#today-title')).toBeVisible(); // shortcuts work once the page has loaded
	const add = async (line: string, button: string, toast: string) => {
		await expect(page.getByRole('dialog', { name: 'Capture' })).toBeHidden();
		await page.locator('#today-title').click();
		await page.keyboard.press('n');
		const dialog = page.getByRole('dialog', { name: 'Capture' });
		await dialog.getByLabel("What's on your mind?").fill(line);
		await dialog.getByRole('button', { name: button }).click();
		await expect(page.getByText(toast)).toBeVisible();
	};
	await page.keyboard.press('n');
	const dialog = page.getByRole('dialog', { name: 'Capture' });
	await dialog.getByLabel("What's on your mind?").fill('Call the plumber @phone tomorrow');
	await expect(dialog.getByText('@phone', { exact: true })).toBeVisible();
	await expect(dialog.getByText(/^due \d{4}-\d{2}-\d{2}$/)).toBeVisible();
	await noSeriousA11yViolations(page);
	await dialog.getByRole('button', { name: 'Capture task' }).click();
	await expect(page.getByText('Captured to your inbox.')).toBeVisible();
	await add('book: The Art of Learning', 'Add book', 'Added to books to read.');
	await add('habit: Stretch for two minutes', 'Add habit', 'Habit added.');
	await add(
		'card: What is interleaving? :: Mixing topics in practice #Quick',
		'Add card',
		'Card added to Quick.'
	);
	await page.goto('/media');
	await expect(page.getByText('The Art of Learning').first()).toBeVisible();
	await page.goto('/habits');
	await expect(page.getByText('Stretch for two minutes').first()).toBeVisible();
});

test('the weekly review walks from an empty head to next week', async () => {
	await page.goto('/today');
	const nav = page.getByRole('navigation', { name: 'Main' });
	await nav.getByRole('button', { name: 'Reflect' }).click();
	await nav.getByRole('link', { name: 'Weekly review' }).click();
	await expect(page.getByRole('heading', { level: 2, name: 'Empty your head' })).toBeVisible();
	await page.getByLabel('One thing on your mind').fill('Return the library books @errand');
	await page.getByRole('button', { name: 'Add', exact: true }).click();
	await expect(page.getByText('Task: Return the library books')).toBeVisible();
	await noSeriousA11yViolations(page);
	await page.getByRole('button', { name: 'Next step' }).click();
	await expect(page.getByRole('heading', { level: 2, name: 'Process your inbox' })).toBeVisible();
	await page.getByRole('button', { name: 'Next action: Return the library books' }).click();
	await expect(page.getByText('A next action.')).toBeVisible();
	await expect(page.getByRole('link', { name: 'Return the library books' })).toHaveCount(0);
	for (const title of [
		'Process your fleeting notes',
		'Look back at the week',
		'Look at the next two weeks',
		'Waiting for',
		'Projects',
		'Goals, habits, and learning',
		'Someday, maybe'
	]) {
		await page.getByRole('button', { name: 'Next step' }).click();
		await expect(page.getByRole('heading', { level: 2, name: title })).toBeVisible();
	}
	await page.getByRole('button', { name: 'Next step' }).click();
	await page.getByLabel('What matters most next week?').fill('Finish the garden shed.');
	await page.getByRole('button', { name: 'Finish the review' }).click();
	await expect(page.getByText('Weekly review done. Enjoy the week.')).toBeVisible();
	await expect(page).toHaveURL(/\/today$/);
});

test('import from another app: preview, import, and nothing twice', async () => {
	await page.goto('/account/import');
	const card = page
		.getByRole('listitem')
		.filter({ has: page.getByRole('heading', { name: 'Goodreads' }) });
	const csv =
		'Book Id,Title,Author,Author l-f,Additional Authors,ISBN,ISBN13,My Rating,Average Rating,Publisher,Binding,Number of Pages,Year Published,Original Publication Year,Date Read,Date Added,Bookshelves,Bookshelves with positions,Exclusive Shelf\n' +
		'9,Ultralearning,Scott Young,,,,,4,4.1,,,,2019,2019,2025/05/01,2025/01/01,,,read\n';
	await card.getByLabel('Goodreads export').setInputFiles({
		name: 'goodreads_library_export.csv',
		mimeType: 'text/csv',
		buffer: (globalThis as unknown as { Buffer: { from(s: string): never } }).Buffer.from(csv)
	});
	await card.getByRole('button', { name: 'Preview' }).click();
	await expect(card.getByText('Ultralearning')).toBeVisible();
	await noSeriousA11yViolations(page);
	await card.getByRole('button', { name: 'Import' }).click();
	await expect(card.getByText(/1\s+added/)).toBeVisible();
	await card.getByRole('button', { name: 'Preview' }).click();
	await card.getByRole('button', { name: 'Import' }).click();
	await expect(card.getByText(/0\s+added, 1 already in Axon/)).toBeVisible();
});

test('a household: start it, invite with a code, share a list, tick an item, and leave', async () => {
	await page.goto('/household');
	await page.getByLabel('Name', { exact: true }).fill('Home');
	await page.getByRole('button', { name: 'Start', exact: true }).click();
	await expect(page.getByRole('heading', { level: 2, name: 'Home' })).toBeVisible();
	await page.getByRole('button', { name: 'Invite someone' }).click();
	await expect(page.locator('.invite code')).toHaveText(
		/^[A-Z2-9]{4}-[A-Z2-9]{4}-[A-Z2-9]{4}-[A-Z2-9]{4}$/
	);
	await page.getByLabel('List', { exact: true }).fill('Groceries');
	await page.getByRole('button', { name: 'Create' }).click();
	await page.getByLabel('Add to Groceries').fill('Oat milk');
	await page.getByRole('button', { name: 'Add', exact: true }).click();
	await expect(page.getByText('1 to do')).toBeVisible();
	await noSeriousA11yViolations(page);
	await page.getByRole('checkbox', { name: 'Oat milk' }).check();
	await expect(page.getByText('0 to do')).toBeVisible();
	await page.getByRole('button', { name: 'Leave the household' }).click();
	await page
		.getByRole('dialog', { name: 'Leave the household?' })
		.getByRole('button', { name: 'Leave' })
		.click();
	await expect(page.getByRole('heading', { name: 'Start a household' })).toBeVisible();
});

test('search by meaning is off until a service and the consent are given', async () => {
	await page.goto('/account/preferences');
	const section = page.getByRole('region', { name: 'Search by meaning' });
	await expect(section.getByLabel('Embedding service')).toHaveValue('off');
	await section.getByLabel('Embedding service').selectOption('local');
	await section.getByLabel('Server address').fill('http://127.0.0.1:9');
	await section.getByRole('button', { name: 'Save the embedding service' }).click();
	await expect(
		page.getByText('Saved. Notes are embedded within a few minutes once you consent.')
	).toBeVisible();
	const consent = section.getByRole('checkbox', {
		name: 'Send the text of my notes to this service to search them by meaning'
	});
	await expect(consent).not.toBeChecked();
	await consent.check();
	await expect(consent).toBeChecked();
	await noSeriousA11yViolations(page);
	// The command center still answers when the service does not.
	await page.keyboard.press('Control+k');
	await page.getByLabel('Search or type a command').fill('learning');
	await expect(
		page.getByRole('listbox', { name: 'Results' }).getByRole('option').first()
	).toBeVisible();
	await page.keyboard.press('Escape');
	await section.getByLabel('Embedding service').selectOption('off');
	await section.getByRole('button', { name: 'Save the embedding service' }).click();
	await expect(
		page.getByText('Search by meaning is off; stored embeddings were deleted.')
	).toBeVisible();
});

test('Today can be customised: cards chosen and ordered', async () => {
	await page.goto('/today');
	await page.getByRole('button', { name: 'Customise Today' }).click();
	const dialog = page.getByRole('dialog', { name: 'Customise Today' });
	await dialog.getByRole('checkbox', { name: 'Learning today' }).check();
	await dialog.getByRole('checkbox', { name: 'Habits today' }).check();
	await dialog.getByRole('checkbox', { name: 'Recent notes' }).uncheck();
	// Learning today first among the cards: up past goals, milestones, tasks (and counts, which stay on top).
	for (let i = 0; i < 6; i++) {
		const up = dialog.getByRole('button', { name: 'Move Learning today up' });
		if (await up.isDisabled()) break;
		await up.click();
	}
	await noSeriousA11yViolations(page);
	await dialog.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByText('Today is set.')).toBeVisible();
	await page.reload();
	await expect(page.getByRole('heading', { name: 'Learning today' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Habits today' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Recent notes' })).toHaveCount(0);
	const learning = await page.getByRole('heading', { name: 'Learning today' }).boundingBox();
	const tasks = await page.getByRole('heading', { name: "Today's tasks" }).boundingBox();
	expect(learning && tasks && (learning.y < tasks.y || learning.x < tasks.x)).toBeTruthy();
	// Back to the defaults for the stories that follow.
	const origin = new URL(page.url()).origin;
	const doc = await (await page.request.get('/api/v1/me/settings')).json();
	delete doc.ai;
	doc.today = { sections: ['kpis', 'tasks', 'milestones', 'goals', 'notes', 'overview'] };
	expect(
		(await page.request.put('/api/v1/me/settings', { headers: { Origin: origin }, data: doc })).ok()
	).toBe(true);
});

test('bulk actions change several tasks at once', async () => {
	const origin = new URL(page.url()).origin;
	for (const title of ['Bulk one', 'Bulk two', 'Bulk three']) {
		await page.request.post('/api/v1/tasks', { headers: { Origin: origin }, data: { title } });
	}
	await page.goto('/tasks?view=inbox');
	await page.getByRole('button', { name: 'Select', exact: true }).click();
	for (const title of ['Bulk one', 'Bulk two', 'Bulk three']) {
		await page.getByRole('checkbox', { name: `Select ${title}` }).check();
	}
	const bar = page.getByRole('group', { name: 'Change 3 selected tasks' });
	await expect(bar).toBeVisible();
	await noSeriousA11yViolations(page);
	await bar.getByRole('button', { name: 'Next' }).click();
	await expect(page.getByText('3 tasks moved to Next.')).toBeVisible();
	await page.goto('/tasks?view=next');
	await page.getByRole('button', { name: 'Select', exact: true }).click();
	for (const title of ['Bulk one', 'Bulk two', 'Bulk three']) {
		await page.getByRole('checkbox', { name: `Select ${title}` }).check();
	}
	await page
		.getByRole('group', { name: 'Change 3 selected tasks' })
		.getByRole('button', { name: 'Delete' })
		.click();
	await expect(page.getByText('3 tasks deleted.')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Bulk one' })).toHaveCount(0);
});

test('drag a task onto a day of the calendar, or pick its date', async () => {
	const origin = new URL(page.url()).origin;
	for (const title of ['Plan the picnic', 'Book the train']) {
		await page.request.post('/api/v1/tasks', {
			headers: { Origin: origin },
			data: { title, status: 'next' }
		});
	}
	await page.goto('/calendar');
	const tray = page.getByRole('region', { name: 'Next actions without a date' });
	await expect(tray.getByText('Plan the picnic', { exact: true })).toBeVisible();
	// The browser's own drag events, carrying one DataTransfer from the task to the day.
	const data = await page.evaluateHandle(() => new DataTransfer());
	await tray
		.locator('li', { hasText: 'Plan the picnic' })
		.dispatchEvent('dragstart', { dataTransfer: data });
	await page.locator('td.today').dispatchEvent('dragover', { dataTransfer: data });
	await page.locator('td.today').dispatchEvent('drop', { dataTransfer: data });
	await expect(page.getByText(/^Moved to /)).toBeVisible();
	await expect(page.locator('td.today').getByText('Plan the picnic')).toBeVisible();
	await tray.getByLabel('Schedule Book the train on').fill(await browserToday());
	await expect(page.locator('td.today').getByText('Book the train')).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('a deleted note can be undone, and restored from the trash', async () => {
	const origin = new URL(page.url()).origin;
	const note = await (
		await page.request.post('/api/v1/notes', {
			headers: { Origin: origin },
			data: { path: 'Zettel/To keep.md', content: '# To keep\n\nWorth keeping.' }
		})
	).json();
	await page.goto(`/notes/${note.id}`);
	await page.getByRole('button', { name: 'Delete' }).click();
	await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click();
	await expect(page).toHaveURL(/\/notes$/);
	await page.getByRole('button', { name: 'Undo' }).click();
	await expect(page.getByText('Put back.')).toBeVisible();
	await expect(page).toHaveURL(new RegExp(`/notes/${note.id}$`));
	await expect(page.getByText('Worth keeping.')).toBeVisible();

	await page.getByRole('button', { name: 'Delete' }).click();
	await page.getByRole('dialog').getByRole('button', { name: 'Delete' }).click();
	await page.goto('/account/trash');
	await expect(page.getByText('To keep', { exact: true })).toBeVisible();
	await noSeriousA11yViolations(page);
	await page.getByRole('button', { name: 'Restore To keep' }).click();
	await expect(page).toHaveURL(new RegExp(`/notes/${note.id}$`));
	await page.goto('/account/trash');
	await expect(page.getByRole('heading', { level: 1, name: 'Account' })).toBeVisible();
	await expect(page.getByText('To keep', { exact: true })).toHaveCount(0);
});

test("share to Axon: a shared link becomes a note, or the page's text a literature note", async () => {
	await page.goto(
		'/share?title=Spacing&text=' + encodeURIComponent('Worth reading https://example.org/spacing')
	);
	await expect(page.getByLabel('Link')).toHaveValue('https://example.org/spacing');
	await expect(page.getByLabel('Text')).toHaveValue('Worth reading');
	await noSeriousA11yViolations(page);
	await page.getByRole('button', { name: 'Save as a note' }).click();
	await expect(page).toHaveURL(/\/notes\/[0-9a-f-]+$/);
	await expect(page.getByRole('link', { name: 'https://example.org/spacing' })).toHaveCount(0); // plain text in the note
	await expect(page.getByText('https://example.org/spacing')).toBeVisible();
	// The page's own text: Axon's sign-in page, reached as the API reaches Caddy.
	await page.goto('/share?url=' + encodeURIComponent('http://caddy:3000/signin'));
	await page.getByRole('button', { name: "Save the page's text as a literature note" }).click();
	await expect(page).toHaveURL(/\/notes\/[0-9a-f-]+\?edit=1$/);
	await expect(page.getByLabel('Markdown')).toHaveValue(/## The page/);
	await expect(page.getByLabel('Markdown')).toHaveValue(/source: "http:\/\/caddy:3000\/signin"/);
});

test('a task can carry a reminder at a time, set by quick add or in the task', async () => {
	await page.goto('/today');
	await expect(page.locator('#today-title')).toBeVisible();
	await page.keyboard.press('n');
	const dialog = page.getByRole('dialog', { name: 'Capture' });
	await dialog.getByLabel("What's on your mind?").fill('Call the bank tomorrow at 15:00');
	await expect(dialog.getByText(/^remind .* 15:00$/)).toBeVisible();
	await dialog.getByRole('button', { name: 'Capture task' }).click();
	await expect(page.getByText('Captured to your inbox.')).toBeVisible();
	await page.goto('/tasks?view=inbox');
	await expect(page.getByText(/🔔/).first()).toBeVisible();
	await page.getByRole('button', { name: 'Call the bank' }).click();
	const remind = page.getByLabel('Remind me at');
	await expect(remind).toHaveValue(/T15:00$/);
	await noSeriousA11yViolations(page);
	await remind.fill('');
	await page.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByText('Saved.')).toBeVisible();
});

test('each page offers its own agents, and says when none can run yet', async () => {
	await page.goto('/tasks');
	await page.getByRole('button', { name: '✨ Ask an agent' }).click();
	const dock = page.getByRole('region', { name: 'Agents for this page' });
	await expect(dock.getByRole('radio', { name: 'Task Manager' })).toBeChecked();
	await expect(dock.getByText('No AI provider is set up yet.')).toBeVisible();
	await expect(dock.getByRole('link', { name: 'Set up agents' })).toHaveAttribute(
		'href',
		'/agents'
	);
	await dock.getByText('Goals', { exact: true }).click();
	await expect(dock.getByRole('radio', { name: 'Goals' })).toBeChecked();
	await noSeriousA11yViolations(page);
	await page.goto('/habits');
	await page.getByRole('button', { name: '✨ Ask the agent' }).click();
	await expect(page.getByRole('region', { name: 'Agents for this page' })).toContainText('Habits');
});

test('without an AI provider agents say so, and a local model can be chosen', async () => {
	await page.goto('/agents');
	await expect(page.getByRole('heading', { level: 1, name: 'Agents' })).toBeVisible();
	await expect(page.getByText('No AI provider is set up yet.')).toBeVisible();
	await expect(page.getByRole('button', { name: 'Ask Task Manager' })).toBeDisabled();
	// Choose a local model; nothing answers at this address, and the test says so.
	await page.getByLabel('Provider', { exact: true }).selectOption('local');
	await page.getByLabel('Model', { exact: true }).fill('llama3.1:8b');
	await page.getByLabel('Server address').fill('http://127.0.0.1:9/v1');
	await page.getByRole('button', { name: 'Save provider' }).click();
	await expect(page.getByText('Your agents use Local model')).toBeVisible();
	await page.getByRole('button', { name: 'Test connection' }).click();
	await expect(page.getByText('The provider did not answer')).toBeVisible();

	// Repositories for the delivery agents: read-only unless comments are allowed.
	await page.getByLabel('Repository', { exact: true }).fill('ada/garden');
	await page.getByRole('button', { name: 'Connect repository' }).click();
	const repos = page.getByRole('region', { name: 'Repositories' });
	await expect(repos).toContainText('ada/garden');
	await expect(repos).toContainText('public, no token');
	await expect(
		repos.getByRole('checkbox', { name: 'Allow approved comments' }).first()
	).not.toBeChecked();
	await noSeriousA11yViolations(page);
	await repos.getByRole('button', { name: 'Disconnect ada/garden' }).click();
	await expect(repos).not.toContainText('ada/garden');
	await expect(page.getByText('Nothing. Agents can only remember what you approve.')).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('the theme follows the sun, over a landscape of your choice', async () => {
	await page.goto('/account/preferences');
	await page.waitForLoadState('networkidle');
	await page.getByLabel('Theme', { exact: true }).selectOption('sun');
	await page.getByLabel('Landscape under the sky').selectOption('taj-mahal');
	// The preview shows the chosen landscape, later in the day too.
	const preview = page.locator('.sky-preview svg');
	await expect(preview).toHaveAttribute('data-landscape', 'taj-mahal');
	await page.getByLabel(/Preview the sky in/).fill('12');
	await page.getByRole('button', { name: 'Save preferences' }).click();
	await expect(page.getByText('Preferences saved.')).toBeVisible();
	// Light from dawn to dusk in the account's time zone, dark otherwise.
	await expect(page.locator('html')).toHaveAttribute('data-theme', /^(light|dark)$/);
	const light = await page.locator('.sky-preview svg').getAttribute('data-light');
	await page.goto('/today');
	await expect(page.locator('.sky-strip svg')).toHaveAttribute('data-landscape', 'taj-mahal');
	const now = await page.locator('.sky-strip svg').getAttribute('data-light');
	await expect(page.locator('html')).toHaveAttribute(
		'data-theme',
		now === 'true' ? 'light' : 'dark'
	);
	expect(light).toMatch(/^(true|false)$/);
	await noSeriousA11yViolations(page);

	await page.goto('/account/preferences');
	await page.waitForLoadState('networkidle');
	await page.getByLabel('Theme', { exact: true }).selectOption('system');
	await page.getByLabel('Landscape under the sky').selectOption('mountains');
	await page.getByRole('button', { name: 'Save preferences' }).click();
	await expect(page.getByText('Preferences saved.')).toBeVisible();
});

test('preferences switch the language, theme, and text size, and export as YAML', async () => {
	await page.goto('/account/preferences');
	await page.getByLabel('Interface language').selectOption('fr');
	await page.getByLabel('Theme', { exact: true }).selectOption('dark');
	await page.getByLabel('Text size').selectOption('large');
	await page.getByRole('button', { name: 'Save preferences' }).click();
	await expect(page.getByRole('link', { name: 'Aujourd’hui' })).toBeVisible();
	await expect(page.getByRole('heading', { level: 1, name: 'Compte' })).toBeVisible();
	await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
	await expect(page.locator('html')).toHaveAttribute('data-size', 'large');
	await expect(page.locator('html')).toHaveAttribute('lang', 'fr');
	await noSeriousA11yViolations(page);

	const yaml = await page.request.get('/api/v1/me/settings.yaml');
	expect(await yaml.text()).toContain('language: fr');

	// The choice follows the person to another page load.
	await page.goto('/today');
	await expect(page.getByRole('heading', { name: 'Vos progrès' })).toBeVisible();

	await page.goto('/account/preferences');
	await page.getByLabel('Langue de l’interface').selectOption('en');
	await page.getByLabel('Thème', { exact: true }).selectOption('system');
	await page.getByLabel('Taille du texte').selectOption('normal');
	await page.getByRole('button', { name: 'Enregistrer les préférences' }).click();
	await expect(page.getByRole('link', { name: 'Today' })).toBeVisible();
});

test('keyboard shortcuts navigate, and Vim mode edits and saves a note', async () => {
	await page.goto('/today');
	const help = page.getByRole('dialog', { name: 'Keyboard shortcuts' });
	// The app listens for keys once it has mounted; retry until it has.
	await expect(async () => {
		await page.keyboard.press('?');
		await expect(help).toBeVisible({ timeout: 1000 });
	}).toPass();
	await expect(help).toContainText('g t');
	await noSeriousA11yViolations(page);
	await page.keyboard.press('Escape');
	await expect(help).toBeHidden();
	await page.keyboard.press('g');
	await page.keyboard.press('t');
	await expect(page).toHaveURL(/\/tasks/);

	// Vim mode, turned on in Preferences.
	await page.goto('/account/preferences');
	await page.getByLabel('Vim keybindings in the note editor').check();
	await page.getByRole('button', { name: 'Save preferences' }).click();
	await expect(page.getByText('Preferences saved.')).toBeVisible();
	const res = await page.request.post('/api/v1/notes', {
		headers: { Origin: origin },
		data: { path: '0-Inbox/vim-test.md', content: '# Vim test\n\nfirst line' }
	});
	const note = await res.json();
	await page.goto(`/notes/${note.id}?edit=1`);
	const editor = page.getByLabel('Markdown');
	await editor.click();
	await expect(page.getByText('-- NORMAL --')).toBeVisible();
	for (const k of ['G', 'o']) await page.keyboard.press(k);
	await expect(page.getByText('-- INSERT --')).toBeVisible();
	await page.keyboard.type('typed in vim');
	await page.keyboard.press('Escape');
	await page.keyboard.press('k');
	await page.keyboard.press('d');
	await page.keyboard.press('d');
	const saved = page.waitForResponse(
		(r) =>
			r.request().method() === 'PUT' &&
			(r.request().postData() ?? '').includes('typed in vim') &&
			!(r.request().postData() ?? '').includes('first line')
	);
	for (const k of [':', 'w', 'Enter']) await page.keyboard.press(k);
	await expect(editor).toHaveValue('# Vim test\n\ntyped in vim');
	await saved;
	await page.reload();
	await expect(page.getByLabel('Markdown')).toHaveValue('# Vim test\n\ntyped in vim');

	await page.goto('/account/preferences');
	await page.getByLabel('Vim keybindings in the note editor').uncheck();
	await page.getByRole('button', { name: 'Save preferences' }).click();
	await expect(page.getByText('Preferences saved.')).toBeVisible();
});

test('upload a profile picture, see it in the header, and go back to the default', async () => {
	await page.goto('/account');
	const pic = page.getByRole('img', { name: 'E2E Owner' });
	await expect(pic).toHaveJSProperty('complete', true);
	// A 32×32 PNG made in the page, handed to the file input as if picked.
	await page.getByLabel('Upload a picture').evaluate(async (input: HTMLInputElement) => {
		const c = document.createElement('canvas');
		c.width = c.height = 32;
		const g = c.getContext('2d')!;
		g.fillStyle = '#3366cc';
		g.fillRect(0, 0, 32, 32);
		const blob: Blob = await new Promise((r) => c.toBlob((b) => r(b!), 'image/png'));
		const files = new DataTransfer();
		files.items.add(new File([blob], 'me.png', { type: 'image/png' }));
		input.files = files.files;
		input.dispatchEvent(new Event('change', { bubbles: true }));
	});
	await expect(page.getByText('Picture updated.')).toBeVisible();
	await expect(page.locator('a.account img')).toHaveAttribute('src', /\?v=/);
	await noSeriousA11yViolations(page);
	await page.getByRole('button', { name: 'Use the default picture' }).click();
	await expect(page.getByText('Back to your default picture.')).toBeVisible();
});

test('a Pomodoro session for a task: start, pause, skip to the break, and save', async () => {
	await page.goto('/focus');
	await expect(
		page
			.locator('p.hint')
			.filter({ hasText: '25 min of focus and 5 min breaks, a 15 min break after 4 rounds' })
	).toBeVisible();
	await page.getByLabel('A task, project, skill, note, or anything else').fill('Buy timber');
	await page.getByRole('button', { name: /Buy timber/ }).click();
	// Music matched to the work: beats for a task (EP-83).
	await page.getByLabel('Focus sound').selectOption('music');
	await expect(page.getByText('Lo-fi beats for “Buy timber”.')).toBeVisible();
	await page.getByRole('button', { name: 'Start 100 min of focus' }).click();
	const timer = page.getByRole('region', { name: 'Focus' });
	await expect(timer).toContainText('Buy timber');
	await expect(timer.getByRole('img', { name: /left in Focus/ })).toBeVisible();
	await expect(timer.getByLabel('Sound', { exact: true })).toHaveValue('music');
	await noSeriousA11yViolations(page);
	await page.getByRole('button', { name: 'Pause' }).click();
	await expect(page.getByRole('button', { name: 'Resume' })).toBeVisible();
	await page.getByRole('button', { name: 'Skip to next' }).click();
	await expect(page.getByRole('heading', { name: 'Short break' })).toBeVisible();
	// A reload resumes the running session.
	await page.reload();
	await expect(page.getByRole('heading', { name: 'Short break' })).toBeVisible();
	await page.getByRole('button', { name: 'Stop and save' }).click();
	await expect(page.getByText('Session saved.')).toBeVisible();
	await expect(page.getByRole('cell', { name: 'Buy timber' })).toBeVisible();
});

test('flashcards: add an illustrated card, review it with the keyboard, take a test, export to Anki', async () => {
	await page.goto('/cards');
	await expect(page.getByRole('navigation', { name: 'Learn' })).toBeVisible();
	await page.getByLabel('Question').fill('What does FSRS schedule?');
	await page.getByLabel('Answer').fill('The next review, just before you would forget.');
	await page.getByLabel('Deck').fill('Memory');
	await page.getByLabel('Picture (optional)').evaluate(async (input: HTMLInputElement) => {
		const c = document.createElement('canvas');
		c.width = c.height = 64;
		c.getContext('2d')!.fillRect(0, 0, 32, 32);
		const blob: Blob = await new Promise((r) => c.toBlob((b) => r(b!), 'image/png'));
		const files = new DataTransfer();
		files.items.add(new File([blob], 'diagram.png', { type: 'image/png' }));
		input.files = files.files;
		input.dispatchEvent(new Event('change', { bubbles: true }));
	});
	await page.getByRole('button', { name: 'Add card' }).click();
	await expect(page.getByText('Card added.')).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Memory' })).toBeVisible();
	await expect(page.getByRole('link', { name: 'Export Memory to Anki' })).toHaveAttribute(
		'href',
		'/api/v1/flashcards/export.apkg?deck=Memory'
	);
	await noSeriousA11yViolations(page);

	await page.getByRole('link', { name: 'Review Memory' }).click();
	await expect(page.getByRole('heading', { name: 'What does FSRS schedule?' })).toBeVisible();
	await page.keyboard.press('Space');
	await expect(page.getByRole('img', { name: /Illustration for/ })).toBeVisible();
	await expect(page.getByRole('button', { name: /Good 10 min/ })).toBeVisible();
	await noSeriousA11yViolations(page);
	await page.keyboard.press('4'); // Easy
	await expect(page.getByRole('heading', { name: 'All done for now' })).toBeVisible();

	// Connect an Anki account; with no sync server reachable, the failure is shown, not hidden.
	await page.goto('/cards');
	const anki = page.getByRole('region', { name: 'Anki' });
	await anki.getByLabel('Sync server').fill('http://127.0.0.1:9/');
	await anki.getByLabel('Anki user name').fill('e2e');
	await anki.getByLabel('Anki password').fill('not-a-real-password');
	await anki.getByRole('button', { name: 'Save' }).click();
	await expect(anki).toContainText('Connected to http://127.0.0.1:9/ as e2e');
	await anki.getByRole('button', { name: 'Sync to Anki now' }).click();
	await expect(anki.getByRole('status')).toContainText('Last sync');
	await noSeriousA11yViolations(page);
	await anki.getByRole('button', { name: 'Disconnect' }).click();
	await expect(anki.getByLabel('Anki user name')).toBeVisible();

	await page.goto('/cards/test?deck=Memory');
	await page.getByLabel('Your answer').fill('The next review');
	await page.getByRole('button', { name: 'Check' }).click();
	await page.getByRole('button', { name: 'I got it right' }).click();
	await expect(page.getByRole('heading', { name: 'Score: 1 of 1' })).toBeVisible();
});

test('learn: cards from a note, free recall, a memory palace walk, and the day plan', async () => {
	test.setTimeout(60_000); // a long story: several pages and two accessibility scans
	const origin = new URL(page.url()).origin;
	const res = await page.request.post('/api/v1/notes', {
		headers: { Origin: origin },
		data: {
			path: 'Zettel/Testing effect.md',
			content:
				'---\ntype: permanent\n---\n# Testing effect\n\nRetrieval practice strengthens long-term memory more than rereading.\nSpaced retrieval slows forgetting and makes learning durable.\n\nWhat is the testing effect? :: Remembering better what you were tested on than what you reread.\n'
		}
	});
	const note = await res.json();
	await page.goto(`/notes/${note.id}`);
	await page.getByRole('button', { name: 'Make cards' }).click();
	const drafts = page.getByRole('dialog', { name: 'Make cards from this note' });
	await expect(drafts).toContainText('What is the testing effect?');
	await drafts.getByRole('button', { name: 'Add 1 cards' }).click();
	await expect(page.getByText('1 card added to Testing effect.')).toBeVisible();

	// Free recall: the note stays hidden; the comparison names what came back.
	await page.getByRole('link', { name: 'Recall' }).click();
	await expect(
		page.getByRole('heading', { level: 1, name: 'Recall: Testing effect' })
	).toBeVisible();
	await page
		.getByLabel('What you remember')
		.fill('Retrieval practice strengthens memory more than rereading.');
	await page.getByRole('button', { name: 'Compare with the note' }).click();
	await expect(page.getByRole('heading', { name: /% of the key ideas/ })).toBeVisible();
	await expect(page.getByText('Next recall in 1 day')).toBeVisible();
	// The history shows the day the server set, not the day before or after
	// (the browser runs in a time zone whose date differs from UTC's).
	const dueOn = await page.evaluate(
		async (id) =>
			(await (await fetch(`/api/v1/notes/${id}/recall`)).json()).items[0].due_on as string,
		note.id
	);
	const shown = await page.evaluate(
		(d) => new Date(`${d}T00:00:00Z`).toLocaleDateString(undefined, { timeZone: 'UTC' }),
		dueOn
	);
	await expect(page.getByText(`next ${shown}`)).toBeVisible();
	await noSeriousA11yViolations(page);

	// A memory palace, a card placed in it, and a walk through it.
	await page.goto('/learn/palaces');
	await page.getByLabel('Place', { exact: true }).fill('My flat');
	await page
		.getByLabel('Spots, in walking order, one per line')
		.fill('Front door\nCoat hook\nKitchen sink');
	await page.getByRole('button', { name: 'Build' }).click();
	await expect(page.getByRole('heading', { level: 1, name: 'My flat' })).toBeVisible();
	const palaceURL = page.url();
	await page.goto('/cards?deck=Testing%20effect');
	await page.getByRole('button', { name: 'Edit What is the testing effect?' }).click();
	const edit = page.getByRole('dialog', { name: 'Edit card' });
	await edit.getByText('Make it stick (optional)').click();
	await edit
		.getByLabel('A vivid memory image')
		.fill('An examiner bursting out of the front door with a giant red pen.');
	await edit.getByLabel('Memory palace').selectOption({ label: 'My flat' });
	await edit.getByLabel('Place', { exact: true }).selectOption({ label: '1. Front door' });
	await edit.getByRole('button', { name: 'Save' }).click();
	await page.goto(palaceURL);
	await page.getByRole('button', { name: 'Walk the palace' }).click();
	await expect(page.getByRole('heading', { name: 'Front door' })).toBeVisible();
	await page.getByRole('button', { name: 'Show' }).click();
	await expect(page.getByText('Picture it: An examiner bursting')).toBeVisible();
	await noSeriousA11yViolations(page);

	// The day's plan, and deliberate practice.
	const nav = page.getByRole('navigation', { name: 'Main' });
	await nav.getByRole('button', { name: 'Grow' }).click();
	await nav.getByRole('link', { name: 'Learn', exact: true }).click();
	await expect(page).toHaveURL(/\/learn$/);
	await expect(page.getByRole('heading', { name: 'Flashcards' })).toBeVisible();
	await expect(page.getByRole('heading', { name: 'Recall your notes' })).toBeVisible();
	await page.getByText('How Axon helps you remember').click();
	await expect(page.getByText('Interleaving.')).toBeVisible();
	await noSeriousA11yViolations(page);
	await page.goto('/cards/review?practice=1');
	await expect(page.getByRole('heading', { level: 1, name: 'Practise weak cards' })).toBeVisible();
});

test('progress rewards are opt-in, explain every point, and reflect every area', async () => {
	await page.goto('/progress');
	await expect(page.getByRole('heading', { name: 'Progress rewards are off' })).toBeVisible();
	await page.getByRole('button', { name: 'Turn on progress rewards' }).click();
	await expect(page.getByRole('heading', { name: /^Level \d+$/ })).toBeVisible();
	await expect(page.getByRole('heading', { name: /-day run$/ })).toBeVisible();
	const ledger = page.getByRole('region', { name: /How you earned your points/ });
	await expect(ledger.getByRole('cell', { name: /Complete a task/ }).first()).toBeVisible();
	await expect(ledger.getByRole('cell', { name: /Review a flashcard/ }).first()).toBeVisible();
	await expect(page.getByRole('region', { name: 'Badges' })).toContainText('First step');
	await noSeriousA11yViolations(page);
	await page.goto('/today');
	await expect(page.getByRole('link', { name: /^Level \d+ · \d+-day run/ })).toBeVisible();
});

test('sign out, then sign back in with the passkey', async () => {
	await page.goto('/account');
	await Promise.all([
		page.waitForResponse((r) => r.url().endsWith('/v1/auth/logout')),
		page.getByRole('button', { name: 'Sign out', exact: true }).click()
	]);
	await page.goto('/account');
	await expect(page).toHaveURL(/\/signin\?next=%2Faccount/);
	await noSeriousA11yViolations(page);
	await page.getByRole('button', { name: 'Sign in with a passkey' }).click();
	await expect(page).toHaveURL(/\/account$/);
	await expect(page.getByLabel('Your name')).toHaveValue('E2E Owner');
});

test('try the shared demo without a passkey, with sample data in every area', async () => {
	await page.goto('/account');
	await Promise.all([
		page.waitForResponse((r) => r.url().endsWith('/v1/auth/logout')),
		page.getByRole('button', { name: 'Sign out', exact: true }).click()
	]);
	await page.goto('/');
	await page.getByRole('button', { name: 'Try the demo' }).click();
	await expect(page).toHaveURL(/\/today$/);
	await expect(page.getByRole('note')).toContainText('You are in the shared demo');

	// The command dashboard: KPIs, charts, tables, and the skills network.
	const overview = page.getByRole('region', { name: 'Your progress' });
	await expect(overview.getByRole('link', { name: /Habit consistency/ })).toBeVisible();
	await expect(overview.getByRole('img', { name: /Daily activity/ })).toBeVisible();
	await expect(overview.getByRole('img', { name: /The last 12 months/ })).toBeVisible();
	// The year ends on the current month, with a legend for its colours (EP-99).
	await expect(overview.getByText('No activity')).toBeVisible();
	const year = overview.getByRole('region', { name: /The last 12 months, scrollable/ });
	expect(await year.evaluate((el) => el.scrollLeft + el.clientWidth >= el.scrollWidth - 2)).toBe(
		true
	);
	const network = overview.getByRole('group', {
		name: /Skills network: \d+ hard skills and \d+ soft skills/
	});
	await expect(network).toBeVisible();
	await overview.getByRole('button', { name: /^Soft/ }).click();
	await expect(network.getByRole('link', { name: /soft skill/ }).first()).toBeVisible();
	await expect(overview.getByRole('cell', { name: /Call the dentist/ })).toBeVisible();
	await overview.getByLabel('Period').selectOption('90');
	await expect(overview.getByText(/vs previous period|same as before/).first()).toBeVisible();
	await noSeriousA11yViolations(page);

	await page.goto('/projects');
	await expect(page.getByRole('link', { name: 'Personal website' })).toBeVisible();
	await page.goto('/roadmaps');
	await expect(
		page.getByRole('link', { name: 'Top-5% in 26 engineering domains', exact: true })
	).toBeVisible();
	// The skills radar: one roadmap, so a spoke per section.
	await expect(page.getByRole('img', { name: /^Hard skills: .*Foundations \d+%/ })).toBeVisible();
	await expect(page.getByRole('img', { name: /^Soft skills: .*Communication \d+%/ })).toBeVisible();
	await noSeriousA11yViolations(page);
	await page.goto('/agents');
	await expect(page.getByText('Agents do not run in the shared demo')).toBeVisible();
	await expect(page.getByRole('region', { name: /Waiting for your approval/ })).toBeVisible();
	await noSeriousA11yViolations(page);

	await page.goto('/account');
	await Promise.all([
		page.waitForResponse((r) => r.url().endsWith('/v1/auth/logout')),
		page.getByRole('button', { name: 'Sign out', exact: true }).click()
	]);
});

test('the API reference renders the live OpenAPI document without a CDN', async () => {
	const external: string[] = [];
	page.on('request', (r) => {
		if (!r.url().startsWith('http://localhost')) external.push(r.url());
	});
	await page.goto('/api-docs/index.html#/tasks/listTasks'); // a deep link opens the operation
	await expect(page.getByRole('heading', { name: /Axon API/ })).toBeVisible();
	await expect(page.locator('#operations-tasks-listTasks .opblock-body')).toContainText(
		'Parameters'
	);
	expect(external).toEqual([]);
});
