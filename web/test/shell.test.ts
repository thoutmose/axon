import { expect, test, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

// One story on a fresh database: create the owner account, use every shell
// page, sign out, and sign back in with the same (virtual) passkey.

test.describe.configure({ mode: 'serial' });

let page: Page;

async function noSeriousA11yViolations(p: Page) {
	const result = await new AxeBuilder({ page: p }).analyze();
	const serious = result.violations.filter(
		(v) => v.impact === 'serious' || v.impact === 'critical'
	);
	expect(serious.map((v) => `${v.id}: ${v.help} (${v.nodes.length})`)).toEqual([]);
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
		'/projects'
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
	await expect(page.getByText('No notes yet.').first()).toBeVisible();
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

test('design a habit, check in with the two-minute version, and link it to a goal', async () => {
	await page.goto('/habits');
	await expect(page.getByText('No habits yet.')).toBeVisible();
	await page.getByLabel('Habit', { exact: true }).fill('Swim one length');
	await page.getByLabel('Two-minute version').fill('Pack my swim bag');
	await page.getByLabel('After…').fill('After breakfast');
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
	await page.getByRole('button', { name: 'Did the two-minute version: Pack my swim bag' }).click();
	await expect(page.getByText('Two-minute version today')).toBeVisible();
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
	await expect(page.getByText('Nothing. Agents can only remember what you approve.')).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('sign out, then sign back in with the passkey', async () => {
	await page.goto('/account');
	await page.getByRole('button', { name: 'Sign out', exact: true }).click();
	await page.goto('/account');
	await expect(page).toHaveURL(/\/signin\?next=%2Faccount/);
	await noSeriousA11yViolations(page);
	await page.getByRole('button', { name: 'Sign in with a passkey' }).click();
	await expect(page).toHaveURL(/\/account$/);
	await expect(page.getByLabel('Your name')).toHaveValue('E2E Owner');
});
