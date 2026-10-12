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

test.beforeAll(async ({ browser }) => {
	page = await (await browser.newContext()).newPage(); // axe needs an explicit context
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

test('home page is accessible and offers registration', async () => {
	// Artwork and fonts ship with Axon: nothing is fetched from another origin.
	const foreign: string[] = [];
	page.on('request', (r) => {
		if (!r.url().startsWith('http://localhost:3000') && !r.url().startsWith('data:'))
			foreign.push(r.url());
	});
	await page.goto('/');
	await page.waitForLoadState('networkidle');
	expect(foreign).toEqual([]);
	await expect(page.getByRole('heading', { level: 1 })).toBeVisible();
	await expect(page.getByRole('status').filter({ hasText: 'Server running' })).toBeVisible();
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
	const today = new Date().toLocaleDateString('sv');
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

test('import an open curriculum with attribution, track skills, and use the graph or the list', async () => {
	await page.goto('/roadmaps');
	await expect(page.getByText('MIT License')).toBeVisible(); // attribution before import
	await page.getByRole('button', { name: 'Import', exact: true }).click();
	await expect(
		page.getByRole('heading', { level: 1, name: 'OSSU Computer Science' })
	).toBeVisible();
	await expect(page.getByText('Open Source Society University (OSSU)')).toBeVisible();
	await page.getByLabel('Status of Systematic program design').selectOption('done');
	await expect(page.getByText('1 of 42 skills done')).toBeVisible();
	await noSeriousA11yViolations(page);

	await page.getByRole('link', { name: 'Skill graph' }).click();
	const graph = page.getByRole('group', { name: /Skill graph: 42 skills, 1 done/ });
	await expect(graph).toBeVisible();
	await graph.getByRole('link', { name: 'Programming language concepts, todo' }).click();
	const detail = page.getByRole('complementary', { name: 'Programming language concepts' });
	await expect(detail).toContainText('Needs first: Systematic program design');
	await expect(detail.getByRole('link', { name: /Programming Languages/ })).toBeVisible(); // the course teaching it
	await detail.getByRole('button', { name: 'Learning' }).click();
	await expect(
		graph.getByRole('link', { name: 'Programming language concepts, learning' })
	).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('import a library roadmap of the best resources, then rename and edit it', async () => {
	await page.goto('/roadmaps');
	await page.getByLabel('Filter domains').fill('podman');
	await expect(page.getByRole('button', { name: /^Import / })).toHaveCount(1);
	await page.getByRole('button', { name: 'Import Docker, Podman, and Kubernetes' }).click();
	await expect(
		page.getByRole('heading', { level: 1, name: 'Docker, Podman, and Kubernetes' })
	).toBeVisible();
	await expect(page.getByText(/belongs to its authors/)).toBeVisible();

	await page.getByRole('button', { name: 'Edit', exact: true }).click();
	const edit = page.getByRole('dialog', { name: 'Edit roadmap' });
	await edit.getByLabel('Title').fill('My container path');
	await edit.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByRole('heading', { level: 1, name: 'My container path' })).toBeVisible();

	await page.getByRole('link', { name: 'Containers', exact: true }).click();
	const detail = page.getByRole('complementary', { name: 'Containers' });
	await expect(detail.getByRole('link', { name: /Docker: Get started/ })).toBeVisible();
	await detail.getByRole('button', { name: 'Edit' }).click();
	const dialog = page.getByRole('dialog', { name: 'Edit skill' });
	await dialog.getByLabel('Skill', { exact: true }).fill('Docker basics');
	await dialog.getByRole('button', { name: 'Save' }).click();
	await expect(page.getByRole('link', { name: 'Docker basics' })).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('take the skills assessment, then start a roadmap at your level', async () => {
	await page.goto('/roadmaps');
	await page.getByRole('link', { name: 'Take the assessment' }).click();
	await page.getByLabel('Your role or situation').fill('Support engineer');
	await page.getByLabel('What do you want to achieve?').fill('Become a backend engineer');
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
	await page.getByRole('button', { name: 'Start Python at my level' }).click();
	await expect(page.getByRole('heading', { level: 1, name: 'Python' })).toBeVisible();
	await expect(page.getByText(/Placed by your skills assessment/)).toBeVisible();
	await expect(page.getByText(/^\d+ of \d+ skills done/)).not.toHaveText(/^0 of/);
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
	await dialog
		.getByRole('radio', { name: /Literature note Templates\/Literature note\.md/ })
		.check();
	await dialog.getByRole('button', { name: 'Create' }).click();
	await expect(page).toHaveURL(/\/notes\/[0-9a-f-]+\?edit=1/);
	await expect(page.getByRole('textbox', { name: /Markdown/ })).toHaveValue(
		/title: Raft consensus/
	);
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

	await page.getByRole('link', { name: 'Training' }).click();
	await page.getByLabel('Title').fill('Full body');
	await page.getByLabel('Minutes').fill('40');
	await page.getByLabel('Exercise 1').fill('Squat');
	await page.getByRole('button', { name: 'Save workout' }).click();
	await expect(page.getByText('Full body')).toBeVisible();
	await expect(page.getByRole('region', { name: 'Last seven days' })).toContainText('40');
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
	await expect(page.getByRole('list', { name: 'Sections of Habit' })).toContainText(
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
		headers: { Origin: 'http://localhost:3000' },
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
	await page.getByRole('button', { name: 'Start 100 min of focus' }).click();
	const timer = page.getByRole('region', { name: 'Focus' });
	await expect(timer).toContainText('Buy timber');
	await expect(timer.getByRole('img', { name: /left in Focus/ })).toBeVisible();
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
	await expect(page.getByRole('link', { name: 'Python' })).toBeVisible();
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
