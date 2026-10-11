<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import Overview from '$lib/Overview.svelte';
	import { api, apiResponse, message, type Schemas } from '$lib/api';
	import type { paths } from '$lib/api-types';
	import { capturing } from '$lib/capture.svelte';
	import ProgressBar from '$lib/ProgressBar.svelte';
	import { overdue, relativeDay, type Task } from '$lib/tasks';
	import Button from '$lib/ui/Button.svelte';
	import Scenery, { timeOfDay } from '$lib/ui/Scenery.svelte';
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

	async function complete(t: Task) {
		try {
			const { etag } = await apiResponse<Schemas['Task']>(`/v1/tasks/${t.id}`);
			await api(`/v1/tasks/${t.id}`, {
				method: 'PATCH',
				headers: { 'If-Match': etag },
				body: { status: 'done' }
			});
			toast(t.recurrence ? 'Done. The next one is scheduled.' : 'Done.');
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function checkIn() {
		if (!d) return;
		try {
			await api(`/v1/checkins/daily/${d.today}`, { method: 'PUT', body: {} });
			d.checked_in = true;
			toast('Checked in for today.');
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	const tod = timeOfDay();
	const dateLabel = new Date().toLocaleDateString(undefined, {
		weekday: 'long',
		day: 'numeric',
		month: 'long'
	});

	function greeting(): string {
		const h = new Date().getHours();
		return h < 12 ? 'Good morning' : h < 18 ? 'Good afternoon' : 'Good evening';
	}
</script>

<svelte:head><title>Today · Axon</title></svelte:head>

<section class="hero" aria-labelledby="today-title">
	<Scenery time={tod} />
	<div class="shade" aria-hidden="true"></div>
	<div class="hero-copy">
		<p class="date">{dateLabel}</p>
		<h1 id="today-title">{greeting()}</h1>
		{#if d}
			<Button
				variant={d.checked_in ? 'ghost' : 'secondary'}
				onclick={checkIn}
				disabled={d.checked_in}
			>
				{d.checked_in ? '✓ Checked in today' : 'Daily check-in'}
			</Button>
		{/if}
	</div>
</section>

<div class="body">
	{#if error}
		<Status kind="error" message={error} retry={load} />
	{:else if !d}
		<Status kind="loading" message="Loading your day…" />
	{:else}
		{#if firstRun}
			<section class="card start" aria-labelledby="start-title">
				<h2 id="start-title">Get started</h2>
				<ol>
					<li class:done={d.totals.tasks > 0}>
						{#if d.totals.tasks > 0}<span class="visually-hidden">Done:</span>✓{/if}
						<span
							>Capture your first task: press <kbd>n</kbd> anywhere, or
							<a href="/tasks?view=inbox">open your inbox</a>.</span
						>
					</li>
					<li class:done={d.totals.notes > 0}>
						{#if d.totals.notes > 0}<span class="visually-hidden">Done:</span>✓{/if}
						<span><a href="/notes">Write your first note</a>, or import your Markdown vault.</span>
					</li>
					<li class:done={d.totals.goals > 0}>
						{#if d.totals.goals > 0}<span class="visually-hidden">Done:</span>✓{/if}
						<span><a href="/goals">Name a goal</a> you want to move this year.</span>
					</li>
				</ol>
			</section>
		{/if}

		<ul class="kpis" aria-label="Task counts">
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

		<div class="columns">
			<section class="card" aria-labelledby="tasks-title">
				<h2 id="tasks-title">{t("Today's tasks")}</h2>
				{#if d.tasks.length === 0}
					<p class="muted">{t('Nothing due or planned for today.')}</p>
				{:else}
					<ul class="rows">
						{#each d.tasks as t (t.id)}
							<li>
								<input
									type="checkbox"
									onchange={() => complete(t)}
									aria-label="Complete {t.title}"
								/>
								<span class="grow">{t.title}</span>
								{#if t.due_on}
									<span class="small" class:late={overdue(t, d.today)}
										>Due {relativeDay(t.due_on, d.today)}</span
									>
								{/if}
							</li>
						{/each}
					</ul>
					{#if (d.kpis.find((k) => k.id === 'today')?.value ?? 0) > d.tasks.length}
						<a href="/tasks?view=today">All of today's tasks →</a>
					{/if}
				{/if}
			</section>

			{#if d.milestones.length}
				<section class="card" aria-labelledby="milestones-title">
					<h2 id="milestones-title">{t('Coming up')}</h2>
					<ul class="rows">
						{#each d.milestones as m (m.id)}
							<li>
								<a class="grow" href="/goals/{m.goal_id}">{m.title}</a>
								<span class="muted small">{m.goal_title} · {relativeDay(m.due_on, d.today)}</span>
							</li>
						{/each}
					</ul>
				</section>
			{/if}

			<section class="card" aria-labelledby="goals-title">
				<h2 id="goals-title">{t('Active goals')}</h2>
				{#if d.goals.length === 0}
					<p class="muted">No active goals. <a href="/goals">Name one</a>.</p>
				{:else}
					<ul class="goals">
						{#each d.goals as g (g.id)}
							<li>
								<a href="/goals/{g.id}">{g.title}</a>
								<ProgressBar progress={g.progress} label="Progress of {g.title}" />
							</li>
						{/each}
					</ul>
				{/if}
			</section>

			<section class="card" aria-labelledby="notes-title">
				<h2 id="notes-title">{t('Recent notes')}</h2>
				{#if d.notes.length === 0}
					<p class="muted">No notes yet. <a href="/notes">Write one</a>.</p>
				{:else}
					<ul class="rows">
						{#each d.notes as n (n.id)}
							<li><a class="grow" href="/notes/{n.id}">{n.title}</a></li>
						{/each}
					</ul>
				{/if}
			</section>
		</div>
		<Overview />
	{/if}
</div>

<style>
	/* Buttons on scenery: frosted white instead of a grey surface. */
	.hero :global(.btn.secondary),
	.hero :global(.btn.ghost) {
		background: rgb(255 255 255 / 0.18);
		color: #fff;
		border-color: rgb(255 255 255 / 0.35);
	}
	.hero :global(.btn.secondary:hover) {
		background: rgb(255 255 255 / 0.28);
	}
	.hero {
		position: relative;
		min-height: clamp(16rem, 42vh, 26rem);
		display: grid;
		align-items: end;
		overflow: hidden;
		color: #fff;
	}
	.shade {
		position: absolute;
		inset: 0;
		background: linear-gradient(180deg, rgb(0 0 0 / 0) 35%, rgb(0 0 0 / 0.5) 100%);
	}
	.hero-copy {
		position: relative;
		max-width: var(--wide-width);
		width: 100%;
		margin: 0 auto;
		padding: var(--space-8) var(--space-4) var(--space-12);
		display: flex;
		align-items: flex-end;
		justify-content: space-between;
		gap: var(--space-4);
		flex-wrap: wrap;
	}
	.hero-copy h1 {
		margin: 0;
		flex-basis: 100%;
		text-shadow: 0 2px 24px rgb(0 0 0 / 0.3);
	}
	.date {
		margin: 0;
		flex-basis: 100%;
		font-weight: 600;
		opacity: 0.92;
		text-shadow: 0 1px 10px rgb(0 0 0 / 0.35);
	}
	.body {
		position: relative;
		max-width: var(--wide-width);
		margin: var(--space-8) auto 0;
		padding: 0 var(--space-4);
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
</style>
