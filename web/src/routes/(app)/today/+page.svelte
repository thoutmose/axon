<script lang="ts">
	import Welcome from '$lib/Welcome.svelte';
	import WhatsNew from '$lib/WhatsNew.svelte';
	import Scenery from '$lib/ui/Scenery.svelte';
	import WaterCard from '$lib/WaterCard.svelte';
	import CaffeineCard from '$lib/CaffeineCard.svelte';
	import JournalToday from '$lib/JournalToday.svelte';
	import { localDate, appLocale } from '$lib/dates';
	import AgentDock from '$lib/AgentDock.svelte';
	import { pageAgents } from '$lib/pageAgents';
	import { t } from '$lib/i18n.svelte';
	import Overview from '$lib/Overview.svelte';
	import RoadmapToday from '$lib/RoadmapToday.svelte';
	import { api, apiResponse, message, type Schemas } from '$lib/api';
	import type { paths } from '$lib/api-types';
	import { capturing } from '$lib/capture.svelte';
	import ProgressBar from '$lib/ProgressBar.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import { overdue, relativeDay, type Task } from '$lib/tasks';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Dashboard = paths['/v1/dashboard']['get']['responses'][200]['content']['application/json'];

	// Every number links to exactly the items it counts.
	const kpiLinks: Record<string, string> = {
		overdue: '/tasks?view=overdue',
		today: '/tasks?view=today',
		inbox: '/tasks?view=inbox'
	};

	let d = $state<Dashboard | null>(null);
	let error = $state('');

	// Customisable Today (EP-63): the cards shown, in order; counts stay at
	// the top and progress at the bottom.
	const labels: Record<string, string> = $derived({
		kpis: t('Counts (overdue, today, inbox)'),
		roadmap: t('Your roadmap today'),
		tasks: t("Today's tasks"),
		milestones: t('Coming up (milestones)'),
		goals: t('Active goals'),
		notes: t('Recent notes'),
		learning: t('Learning today'),
		habits: t('Habits today'),
		calendar: t('Calendar today'),
		household: t('Household'),
		journal: t('How are you today?'),
		water: t('Water today'),
		caffeine: t('Coffee and tea today'),
		overview: t('Your progress')
	});
	const all = $derived(Object.keys(labels));
	let sections = $state<string[]>([
		'kpis',
		'roadmap',
		'tasks',
		'milestones',
		'goals',
		'notes',
		'overview'
	]);
	const cards = $derived(
		sections.filter((x) => x !== 'kpis' && x !== 'overview' && x !== 'roadmap')
	);
	// Sections new in a release show until they are turned off (EP-145).
	const fresh = ['roadmap'];
	let customising = $state(false);
	let draft = $state<{ id: string; on: boolean }[]>([]);
	type Plan = Schemas['LearningPlan'];
	type Habit = Schemas['Habit'];
	type Household = Schemas['Household'];
	type Occurrence = { title: string; start: string; all_day: boolean; location: string };
	let plan = $state<Plan | null>(null);
	let habits = $state<Habit[]>([]);
	let events = $state<Occurrence[]>([]);
	let household = $state<Household | null>(null);
	let me = $state('');

	// A new account is first asked what it wants Axon for (EP-94).
	let chosen = $state(true);
	api<{ today?: { sections: string[] | null; hidden?: string[] }; menu?: { chosen: boolean } }>(
		'/v1/me/settings'
	)
		.then((s) => {
			// Without a list (an older document), the defaults stay.
			if (Array.isArray(s.today?.sections)) {
				const hidden = s.today.hidden ?? [];
				const list = s.today.sections;
				const added = fresh.filter((x) => !list.includes(x) && !hidden.includes(x));
				sections = list[0] === 'kpis' ? ['kpis', ...added, ...list.slice(1)] : [...added, ...list];
			}
			chosen = s.menu?.chosen ?? true;
		})
		.catch(() => {});
	api<{ id: string }>('/v1/me')
		.then((m) => (me = m.id))
		.catch(() => {});

	// Each optional card loads its own data, only when shown.
	$effect(() => {
		void capturing.count;
		const today = localDate(new Date());
		if (sections.includes('learning'))
			api<Plan>('/v1/learning/plan')
				.then((p) => (plan = p))
				.catch(() => {});
		if (sections.includes('habits'))
			api<{ items: Habit[] }>('/v1/habits?status=active')
				.then((r) => (habits = r.items.filter((h) => h.state.scheduled_today)))
				.catch(() => {});
		if (sections.includes('calendar'))
			api<{ items: Occurrence[] }>(`/v1/calendar/events?from=${today}&to=${today}`)
				.then((r) => (events = r.items))
				.catch(() => {});
		if (sections.includes('household'))
			api<Household>('/v1/household')
				.then((h) => (household = h))
				.catch(() => (household = null));
	});
	const mine = $derived(
		(household?.lists ?? []).flatMap((l) =>
			l.items
				.filter((i) => !i.done_at && (i.assignee === me || i.assignee === null))
				.map((i) => ({ ...i, list: l.title }))
		)
	);

	function openCustomise() {
		draft = [
			...sections.map((id) => ({ id, on: true })),
			...all.filter((id) => !sections.includes(id)).map((id) => ({ id, on: false }))
		];
		customising = true;
	}
	function move(i: number, by: number) {
		const j = i + by;
		if (j < 0 || j >= draft.length) return;
		const next = [...draft];
		[next[i], next[j]] = [next[j], next[i]];
		draft = next;
	}
	async function saveCustom() {
		try {
			const doc = await api<Record<string, unknown>>('/v1/me/settings');
			delete doc.ai;
			const chosen = draft.filter((x) => x.on).map((x) => x.id);
			const hidden = draft.filter((x) => !x.on).map((x) => x.id);
			await api('/v1/me/settings', {
				method: 'PUT',
				body: { ...doc, today: { sections: chosen, hidden } }
			});
			sections = chosen;
			customising = false;
			toast(t('Today is set.'));
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function load() {
		error = '';
		try {
			d = await api<Dashboard>('/v1/dashboard');
		} catch (err) {
			error = message(err);
		}
	}

	$effect(() => {
		void capturing.count;
		load();
	});

	const firstRun = $derived(
		!!d && (d.totals.tasks === 0 || d.totals.notes === 0 || d.totals.goals === 0)
	);

	async function complete(task: Task) {
		try {
			const { etag } = await apiResponse<Schemas['Task']>(`/v1/tasks/${task.id}`);
			await api(`/v1/tasks/${task.id}`, {
				method: 'PATCH',
				headers: { 'If-Match': etag },
				body: { status: 'done' }
			});
			toast(task.recurrence ? t('Done. The next one is scheduled.') : t('Done.'));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	// A task done today stays, crossed out, until tomorrow (EP-116); unchecking
	// it puts it back in the list it came from.
	async function reopen(task: Task) {
		try {
			const { etag } = await apiResponse<Schemas['Task']>(`/v1/tasks/${task.id}`);
			await api(`/v1/tasks/${task.id}`, {
				method: 'PATCH',
				headers: { 'If-Match': etag },
				body: { status: task.done_from ?? 'next' }
			});
			toast(t('Reopened.'));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function habitDone(h: Habit, done: boolean) {
		try {
			await api(`/v1/habits/${h.id}/checkins/${h.state.today}`, {
				method: done ? 'PUT' : 'DELETE',
				body: done ? { status: 'done' } : undefined
			});
			const r = await api<{ items: Habit[] }>('/v1/habits?status=active');
			habits = r.items.filter((x) => x.state.scheduled_today);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function checkIn() {
		if (!d) return;
		try {
			await api(`/v1/checkins/daily/${d.today}`, { method: 'PUT', body: {} });
			d.checked_in = true;
			toast(t('Checked in for today.'));
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	// Derived: the saved language is applied after this page is first made.
	const dateLabel = $derived(
		new Date().toLocaleDateString(appLocale(), { weekday: 'long', day: 'numeric', month: 'long' })
	);

	function greeting(): string {
		const h = new Date().getHours();
		return h < 12 ? t('Good morning') : h < 18 ? t('Good afternoon') : t('Good evening');
	}
</script>

<svelte:head><title>{t('Today · Axon')}</title></svelte:head>

<!-- A compact header: the dashboard starts right under the navigation. -->
<section class="hero" aria-labelledby="today-title">
	<!-- The sky above the person's day, as it is now (EP-96). -->
	<div class="sky-strip" aria-hidden="true"><Scenery /></div>
	<div class="hero-copy">
		<h1 id="today-title">{greeting()} <span class="date">· {dateLabel}</span></h1>
		{#if d}
			<Button
				variant={d.checked_in ? 'ghost' : 'secondary'}
				onclick={checkIn}
				disabled={d.checked_in}
			>
				{d.checked_in ? t('✓ Checked in today') : t('Daily check-in')}
			</Button>
		{/if}
	</div>
</section>

<div class="body">
	<AgentDock {...pageAgents.today} onapplied={() => load()} />
	{#if error}
		<Status kind="error" message={error} retry={load} />
	{:else if !d}
		<Status kind="loading" message={t('Loading your day…')} />
	{:else}
		{#if !firstRun}<WhatsNew />{/if}
		{#if firstRun && !chosen}
			<Welcome
				ondone={(s) => {
					chosen = true;
					if (s) sections = s;
				}}
			/>
		{/if}
		{#if firstRun}
			<section class="card start" aria-labelledby="start-title">
				<h2 id="start-title">{t('Get started')}</h2>
				<ol>
					<li class:done={d.totals.tasks > 0}>
						{#if d.totals.tasks > 0}<span class="visually-hidden">{t('Done:')}</span>✓{/if}
						<span
							>{t('Capture your first task: press')} <kbd>n</kbd>
							{t('anywhere, or')}
							<a href="/tasks?view=inbox">{t('open your inbox')}</a>.</span
						>
					</li>
					<li class:done={d.totals.notes > 0}>
						{#if d.totals.notes > 0}<span class="visually-hidden">{t('Done:')}</span>✓{/if}
						<span
							><a href="/notes">{t('Write your first note')}</a>{t(
								', or import your Markdown vault.'
							)}</span
						>
					</li>
					<li class:done={d.totals.goals > 0}>
						{#if d.totals.goals > 0}<span class="visually-hidden">{t('Done:')}</span>✓{/if}
						<span><a href="/goals">{t('Name a goal')}</a> {t('you want to move this year.')}</span>
					</li>
				</ol>
			</section>
		{/if}

		<div class="customise">
			<Button variant="ghost" onclick={openCustomise}>{t('Customise Today')}</Button>
		</div>
		{#if sections.includes('kpis')}
			<ul class="kpis" aria-label={t('Task counts')}>
				{#each d.kpis as k (k.id)}
					<li>
						<a href={kpiLinks[k.id]} class="glass" class:alert={k.id === 'overdue' && k.value > 0}>
							<span class="value">{k.value}</span>
							<span class="label">{k.label}</span>
						</a>
						<p class="definition">{k.definition}</p>
					</li>
				{/each}
			</ul>
		{/if}

		{#if sections.includes('roadmap')}<RoadmapToday today={d.today} />{/if}

		<div class="columns">
			{#each cards as id (id)}
				{#if id === 'tasks'}
					<section class="card" aria-labelledby="tasks-title">
						<h2 id="tasks-title">{t("Today's tasks")}</h2>
						{#if d.tasks.length === 0 && d.done_today.length === 0}
							<p class="muted">{t('Nothing due or planned for today.')}</p>
						{:else}
							<ul class="rows">
								{#each d.done_today as doneTask (doneTask.id)}
									<li class="done">
										<input
											type="checkbox"
											checked
											onchange={() => reopen(doneTask)}
											aria-label={t('Reopen {title}', { title: doneTask.title })}
										/>
										<span class="grow">{doneTask.title}</span>
									</li>
								{/each}
								{#each d.tasks as dayTask (dayTask.id)}
									<li>
										<input
											type="checkbox"
											onchange={() => complete(dayTask)}
											aria-label={t('Complete {title}', { title: dayTask.title })}
										/>
										<span class="grow">{dayTask.title}</span>
										{#if dayTask.due_on}
											<span class="small" class:late={overdue(dayTask, d.today)}
												>{t('Due {value}', { value: relativeDay(dayTask.due_on, d.today) })}</span
											>
										{/if}
									</li>
								{/each}
							</ul>
							{#if (d.kpis.find((k) => k.id === 'today')?.value ?? 0) > d.tasks.length}
								<a href="/tasks?view=today">{t("All of today's tasks →")}</a>
							{/if}
						{/if}
					</section>
				{:else if id === 'milestones'}
					{#if d.milestones.length}
						<section class="card" aria-labelledby="milestones-title">
							<h2 id="milestones-title">{t('Coming up')}</h2>
							<ul class="rows">
								{#each d.milestones as m (m.id)}
									<li>
										<a class="grow" href="/goals/{m.goal_id}">{m.title}</a>
										<span class="muted small"
											>{m.goal_title} · {relativeDay(m.due_on, d.today)}</span
										>
									</li>
								{/each}
							</ul>
						</section>
					{/if}
				{:else if id === 'goals'}
					<section class="card" aria-labelledby="goals-title">
						<h2 id="goals-title">{t('Active goals')}</h2>
						{#if d.goals.length === 0}
							<p class="muted">{t('No active goals.')} <a href="/goals">{t('Name one')}</a>.</p>
						{:else}
							<ul class="goals">
								{#each d.goals as g (g.id)}
									<li>
										<a href="/goals/{g.id}">{g.title}</a>
										<ProgressBar
											progress={g.progress}
											label={t('Progress of {title}', { title: g.title })}
										/>
									</li>
								{/each}
							</ul>
						{/if}
					</section>
				{:else if id === 'notes'}
					<section class="card" aria-labelledby="notes-title">
						<h2 id="notes-title">{t('Recent notes')}</h2>
						{#if d.notes.length === 0}
							<p class="muted">{t('No notes yet.')} <a href="/notes">{t('Write one')}</a>.</p>
						{:else}
							<ul class="rows">
								{#each d.notes as n (n.id)}
									<li><a class="grow" href="/notes/{n.id}">{n.title}</a></li>
								{/each}
							</ul>
						{/if}
					</section>
				{:else if id === 'learning'}
					<section class="card" aria-labelledby="learning-title">
						<h2 id="learning-title">{t('Learning today')}</h2>
						{#if plan}
							<p>
								<strong>{plan.cards.due}</strong>
								{t(plan.cards.due ? 'cards · about {minutes} min' : 'cards', {
									minutes: plan.cards.minutes
								})}{#if plan.recall.due.length}
									· <strong>{plan.recall.due.length}</strong>
									{t('notes to recall')}{/if}{#if plan.test.due}
									{t('· the weekly test')}{/if}
							</p>
							<a href="/learn">{t('Learn → Today')}</a>
						{:else}
							<p class="muted">{t('Loading…')}</p>
						{/if}
					</section>
				{:else if id === 'habits'}
					<section class="card" aria-labelledby="habits-title">
						<h2 id="habits-title">{t('Habits today')}</h2>
						{#if habits.length === 0}
							<p class="muted">{t('No habit scheduled today.')}</p>
						{:else}
							<ul class="rows">
								{#each habits as h (h.id)}
									<li class:done={!!h.state.checked_in}>
										<input
											type="checkbox"
											checked={!!h.state.checked_in}
											onchange={(e) => habitDone(h, e.currentTarget.checked)}
											aria-label={h.state.checked_in
												? t("Undo today's check-in for {title}", { title: h.title })
												: t('Done: {title}', { title: h.title })}
										/>
										<a class="grow" href="/habits">{h.title}</a>
										<span class="small muted"
											>{h.state.checked_in
												? '✓ ' + h.state.checked_in
												: h.two_minute || t('to do')}</span
										>
									</li>
								{/each}
							</ul>
						{/if}
					</section>
				{:else if id === 'calendar'}
					<section class="card" aria-labelledby="calendar-title">
						<h2 id="calendar-title">{t('Calendar today')}</h2>
						{#if events.length === 0}
							<p class="muted">
								{t('Nothing in the calendars you follow today.')}
								<a href="/calendar">{t('Calendar')}</a>
							</p>
						{:else}
							<ul class="rows">
								{#each events as ev, i (i)}
									<li>
										<span class="small time"
											>{ev.all_day
												? t('All day')
												: new Date(ev.start).toLocaleTimeString(appLocale(), {
														hour: '2-digit',
														minute: '2-digit'
													})}</span
										>
										<span class="grow">{ev.title}</span>
										{#if ev.location}<span class="small muted">{ev.location}</span>{/if}
									</li>
								{/each}
							</ul>
						{/if}
					</section>
				{:else if id === 'journal'}
					<JournalToday />
				{:else if id === 'water'}
					<WaterCard />
				{:else if id === 'caffeine'}
					<CaffeineCard />
				{:else if id === 'household'}
					<section class="card" aria-labelledby="household-title">
						<h2 id="household-title">{t('Household')}</h2>
						{#if !household}
							<p class="muted">
								{t('No household.')} <a href="/household">{t('Start or join one')}</a>.
							</p>
						{:else if mine.length === 0}
							<p class="muted">{t('Nothing open for you.')}</p>
						{:else}
							<ul class="rows">
								{#each mine.slice(0, 6) as i (i.id)}
									<li>
										<span class="grow">{i.title}</span><span class="small muted">{i.list}</span>
									</li>
								{/each}
							</ul>
							<a href="/household">{t('All lists →')}</a>
						{/if}
					</section>
				{/if}
			{/each}
		</div>
		{#if sections.includes('overview')}
			<Overview />
		{/if}
	{/if}
</div>

<Dialog bind:open={customising} title={t('Customise Today')}>
	<p class="muted small">
		{t('Choose the cards and their order. Counts stay at the top, your progress at the bottom.')}
	</p>
	<ul class="choose">
		{#each draft as x, i (x.id)}
			<li>
				<label><input type="checkbox" bind:checked={x.on} /> {labels[x.id]}</label>
				<span class="moves">
					<Button
						variant="ghost"
						onclick={() => move(i, -1)}
						disabled={i === 0}
						aria-label={t('Move {value} up', { value: labels[x.id] })}>↑</Button
					>
					<Button
						variant="ghost"
						onclick={() => move(i, 1)}
						disabled={i === draft.length - 1}
						aria-label={t('Move {value} down', { value: labels[x.id] })}>↓</Button
					>
				</span>
			</li>
		{/each}
	</ul>
	<div class="dialog-actions">
		<Button onclick={saveCustom}>{t('Save')}</Button>
		<Button variant="secondary" onclick={() => (customising = false)}>{t('Cancel')}</Button>
	</div>
</Dialog>

<style>
	.rows li.done .grow {
		color: var(--muted);
		text-decoration: line-through;
	}
	.customise {
		display: flex;
		justify-content: flex-end;
		margin-bottom: var(--space-2);
	}
	.choose {
		list-style: none;
		padding: 0;
		display: grid;
		gap: var(--space-1);
	}
	.choose li {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-2);
	}
	.moves {
		display: flex;
	}
	.dialog-actions {
		display: flex;
		gap: var(--space-2);
		margin-top: var(--space-3);
	}
	.time {
		min-width: 4rem;
		font-variant-numeric: tabular-nums;
	}
	.hero-copy {
		max-width: var(--wide-width);
		margin: 0 auto;
		padding: var(--space-6) var(--gutter) 0;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-4);
		flex-wrap: wrap;
	}
	.hero-copy h1 {
		margin: 0;
		font-size: var(--text-2xl);
	}
	.date {
		font-weight: 400;
		color: var(--muted);
		font-size: var(--text-lg);
	}
	.body {
		position: relative;
		max-width: var(--wide-width);
		margin: var(--space-4) auto 0;
		padding: 0 var(--gutter);
	}
	.columns {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 22rem), 1fr));
		gap: var(--space-6);
		align-items: start;
	}
	.columns .card {
		margin-bottom: 0;
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6) var(--space-6);
		margin-bottom: var(--space-6);
	}
	@media (max-width: 30rem) {
		.card {
			padding: var(--space-4);
		}
	}
	.card h2 {
		font-size: var(--text-lg);
	}
	.start {
		border-color: var(--accent);
	}
	.start ol {
		padding-left: var(--space-6);
		display: grid;
		gap: var(--space-2);
	}
	.start li.done {
		color: var(--muted);
	}
	kbd {
		font-family: var(--font-mono);
		border: 1px solid var(--border);
		border-bottom-width: 2px;
		border-radius: var(--radius-sm);
		padding: 0 0.35em;
	}
	.kpis {
		list-style: none;
		padding: 0;
		margin: var(--space-6) 0;
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
		gap: var(--space-3);
	}
	.kpis a {
		display: grid;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-3) var(--space-4);
		color: var(--fg);
		text-decoration: none;
	}
	.kpis a:hover {
		border-color: var(--accent);
	}
	.kpis a.alert .value {
		color: var(--bad);
	}
	.value {
		font-size: var(--text-2xl);
		font-weight: 700;
		line-height: 1.1;
	}
	.label {
		font-weight: 600;
	}
	.definition {
		margin: var(--space-1) var(--space-1) 0;
		font-size: var(--text-sm);
		color: var(--muted);
	}
	.rows,
	.goals {
		list-style: none;
		padding: 0;
		margin: 0 0 var(--space-3);
	}
	.rows li {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-2) 0;
		border-bottom: 1px solid var(--border);
		flex-wrap: wrap;
	}
	.rows li:last-child {
		border-bottom: none;
	}
	.rows input[type='checkbox'] {
		width: 1.25rem;
		height: 1.25rem;
		accent-color: var(--accent);
	}
	.goals li {
		padding: var(--space-2) 0;
	}
	.goals a {
		font-weight: 600;
	}
	.grow {
		flex: 1;
		min-width: 8rem;
		overflow-wrap: anywhere;
	}
	.small {
		font-size: var(--text-sm);
		color: var(--muted);
	}
	.late {
		color: var(--bad);
		font-weight: 600;
	}
	.sky-strip {
		position: relative;
		max-width: var(--wide-width);
		height: 5.5rem;
		margin: var(--space-4) auto 0;
		border-radius: var(--radius-lg);
		overflow: hidden;
		width: calc(100% - 2 * var(--gutter));
	}
</style>
