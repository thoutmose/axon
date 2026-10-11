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
		'/roadmaps'
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
	await page.getByLabel('Status of Systematic Program Design').selectOption('done');
	await expect(page.getByText('1 of 63 skills done')).toBeVisible();
	await noSeriousA11yViolations(page);

	await page.getByRole('link', { name: 'Skill graph' }).click();
	const graph = page.getByRole('group', { name: /Skill graph: 63 skills, 1 done/ });
	await expect(graph).toBeVisible();
	await graph.getByRole('link', { name: 'Class-based Program Design, todo' }).click();
	const detail = page.getByRole('complementary', { name: 'Class-based Program Design' });
	await expect(detail).toContainText('Needs first: Systematic Program Design');
	await detail.getByRole('button', { name: 'Learning' }).click();
	await expect(
		graph.getByRole('link', { name: 'Class-based Program Design, learning' })
	).toBeVisible();
	await noSeriousA11yViolations(page);
});

test('import a library roadmap of the best resources, then rename and edit it', async () => {
	await page.goto('/roadmaps');
	await page.getByLabel('Filter domains').fill('kubernetes');
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

	await page.getByRole('link', { name: 'Docker: Get started' }).click();
	const detail = page.getByRole('complementary', { name: 'Docker: Get started' });
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
	await page.keyboard.press('?');
	const help = page.getByRole('dialog', { name: 'Keyboard shortcuts' });
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
	for (const k of [':', 'w', 'Enter']) await page.keyboard.press(k);
	await expect(editor).toHaveValue('# Vim test\n\ntyped in vim');
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

test('try the shared demo without a passkey, with sample data in every area', async () => {
	await page.goto('/account');
	await page.getByRole('button', { name: 'Sign out', exact: true }).click();
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
		name: /Skills network: \d+ hard skills, \d+ soft skills/
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
	await page.getByRole('button', { name: 'Sign out', exact: true }).click();
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
