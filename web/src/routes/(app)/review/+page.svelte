<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, apiResponse, message, type Schemas } from '$lib/api';
	import { addDays, formatDate, localDate, mondayOf, appLocale } from '$lib/dates';
	import { describe, parse } from '$lib/quickAdd';
	import { addParsed } from '$lib/quickAddApi';
	import type { Task } from '$lib/tasks';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// The weekly review (EP-59), after Getting Things Done: get clear (empty
	// your head, process the inboxes), get current (last week, the weeks
	// ahead, waiting for, projects, goals, habits, learning), get creative
	// (someday/maybe), then write down what you take into next week.

	type Note = Schemas['Note'];
	type Project = Schemas['Project'];
	type Goal = Schemas['Goal'];
	type Review = {
		week: string;
		highlights: string[];
		metrics: { id: string; title: string; value: number; previous: number; unit: string }[];
	};
	type Plan = Schemas['LearningPlan'];
	type Own = Schemas['CalendarEvent'];
	type Occurrence = { title: string; date: string; all_day: boolean; start: string };

	const steps = $derived([
		{ id: 'head', group: t('Get clear'), title: t('Empty your head') },
		{ id: 'inbox', group: t('Get clear'), title: t('Process your inbox') },
		{ id: 'notes', group: t('Get clear'), title: t('Process your fleeting notes') },
		{ id: 'back', group: t('Get current'), title: t('Look back at the week') },
		{ id: 'ahead', group: t('Get current'), title: t('Look at the next two weeks') },
		{ id: 'waiting', group: t('Get current'), title: t('Waiting for') },
		{ id: 'projects', group: t('Get current'), title: t('Projects') },
		{ id: 'goals', group: t('Get current'), title: t('Goals, habits, and learning') },
		{ id: 'someday', group: t('Get creative'), title: t('Someday, maybe') },
		{ id: 'done', group: t('Wrap up'), title: t('Into next week') }
	] as const);
	const step = $derived(
		Math.min(Math.max(Number(page.url.searchParams.get('step') ?? 0), 0), steps.length - 1)
	);
	const current = $derived(steps[step]);
	const go = (n: number) => goto(`?step=${n}`, { noScroll: false, keepFocus: false });

	const todayISO = localDate(new Date());
	const monday = mondayOf(todayISO);
	const inDays = (n: number) => addDays(todayISO, n);

	let error = $state('');
	let loading = $state(false);
	let tasks = $state<Task[]>([]);
	let notes = $state<Note[]>([]);
	let projects = $state<Project[]>([]);
	let goals = $state<Goal[]>([]);
	let review = $state<Review | null>(null);
	let plan = $state<Plan | null>(null);
	let ahead = $state<{ date: string; title: string; what: string }[]>([]);
	let line = $state('');
	let captured = $state<string[]>([]);
	let reflection = $state({ well: '', improve: '', focus: '' });

	async function load() {
		error = '';
		loading = true;
		try {
			switch (current.id) {
				case 'inbox':
					tasks = (await api<{ items: Task[] }>('/v1/tasks?view=inbox&limit=100')).items;
					break;
				case 'waiting':
					tasks = (await api<{ items: Task[] }>('/v1/tasks?view=waiting&limit=100')).items;
					break;
				case 'someday':
					tasks = (await api<{ items: Task[] }>('/v1/tasks?view=someday&limit=100')).items;
					break;
				case 'notes':
					notes = (await api<{ items: Note[] }>('/v1/notes?kind=fleeting&mirrored=false&limit=100'))
						.items;
					break;
				case 'back':
					review = await api<Review>('/v1/insights/review');
					break;
				case 'ahead': {
					const q = new URLSearchParams({ from: todayISO, to: inDays(14) });
					const [mine, theirs] = await Promise.all([
						api<{ items: Own[] }>(`/v1/calendar?${q}`),
						api<{ items: Occurrence[] }>(`/v1/calendar/events?${q}`)
					]);
					ahead = [
						...mine.items
							.filter((e) => !e.done)
							.map((e) => ({
								date: e.date,
								title: e.title,
								what: e.kind === 'due' ? 'due' : e.type
							})),
						...theirs.items.map((o) => ({
							date: o.date,
							title: o.title,
							what: o.all_day
								? 'event'
								: new Date(o.start).toLocaleTimeString(appLocale(), {
										hour: '2-digit',
										minute: '2-digit'
									})
						}))
					].sort((a, b) => a.date.localeCompare(b.date));
					break;
				}
				case 'projects':
					projects = (await api<{ items: Project[] }>('/v1/projects?status=active')).items;
					break;
				case 'goals':
					[goals, plan] = await Promise.all([
						api<{ items: Goal[] }>('/v1/goals?status=active').then((r) => r.items),
						api<Plan>('/v1/learning/plan')
					]);
					break;
			}
		} catch (err) {
			error = message(err);
		} finally {
			loading = false;
		}
	}
	$effect(() => {
		void step;
		load();
	});

	async function capture(e: SubmitEvent) {
		e.preventDefault();
		if (!line.trim()) return;
		try {
			const p = parse(line);
			await addParsed(p);
			captured = [...captured, `${describe(p)[0]}: ${p.title}`];
			line = '';
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	/** Moves a task on (needs its version for the update). */
	async function move(t: Task, change: Record<string, unknown>, said: string) {
		try {
			if (change.delete) {
				await api(`/v1/tasks/${t.id}`, { method: 'DELETE' });
			} else {
				const { etag } = await apiResponse(`/v1/tasks/${t.id}`);
				await api(`/v1/tasks/${t.id}`, {
					method: 'PATCH',
					headers: { 'If-Match': etag },
					body: change
				});
			}
			tasks = tasks.filter((x) => x.id !== t.id);
			toast(said);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function finish(e: SubmitEvent) {
		e.preventDefault();
		const note = [
			reflection.well && t('Went well: {well}', { well: reflection.well }),
			reflection.improve && t('To improve: {improve}', { improve: reflection.improve }),
			reflection.focus && t('Focus next week: {focus}', { focus: reflection.focus })
		]
			.filter(Boolean)
			.join('\n')
			.slice(0, 2000);
		try {
			await api(`/v1/checkins/weekly/${monday}`, { method: 'PUT', body: { note } });
			toast(t('Weekly review done. Enjoy the week.'));
			await goto('/today');
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<svelte:head><title>{t('Weekly review · Axon')}</title></svelte:head>

<h1>{t('Weekly review')}</h1>
<p class="muted">
	{t(
		'About twenty minutes, once a week: clear your head and your inboxes, see where things stand, and choose what matters next week (Getting Things Done).'
	)}
</p>

<nav aria-label={t('Review steps')}>
	<ol class="steps">
		{#each steps as s, i (s.id)}
			<li class:done={i < step}>
				<a href="?step={i}" aria-current={i === step ? 'step' : undefined}>{s.title}</a>
			</li>
		{/each}
	</ol>
</nav>

<section class="card" aria-labelledby="step-title">
	<p class="group">
		{t('{group} · {value} of {length}', {
			group: current.group,
			value: step + 1,
			length: steps.length
		})}
	</p>
	<h2 id="step-title">{current.title}</h2>

	{#if error}
		<Status kind="error" message={error} retry={load} />
	{:else if loading}
		<Status kind="loading" message={t('Loading…')} />
	{:else if current.id === 'head'}
		<p>
			{t(
				'Write down everything on your mind, one line at a time: tasks, ideas, books, anything. Sort it out in the next step.'
			)}
		</p>
		<form onsubmit={capture} class="capture">
			<label for="review-line" class="visually-hidden">{t('One thing on your mind')}</label>
			<input
				id="review-line"
				bind:value={line}
				placeholder={t('Call the bank @phone friday')}
				autocomplete="off"
			/>
			<Button type="submit" variant="secondary">{t('Add')}</Button>
		</form>
		{#if captured.length}<ul class="small">
				{#each captured as c, i (i)}<li>{c}</li>{/each}
			</ul>{/if}
	{:else if current.id === 'inbox' || current.id === 'waiting' || current.id === 'someday'}
		{#if current.id === 'inbox'}
			<p>{t('For each: what is the next action? Do it now if it takes two minutes.')}</p>
		{:else if current.id === 'waiting'}
			<p>{t('Still waiting? Nudge, or take it back.')}</p>
		{:else}
			<p>{t('Anything to start now? Anything no longer worth keeping?')}</p>
		{/if}
		{#if tasks.length}
			<ul class="rows">
				{#each tasks as item (item.id)}
					<li>
						<a href="/tasks/{item.id}" class="grow">{item.title}</a>
						{#if current.id !== 'someday'}
							<Button
								variant="ghost"
								onclick={() => move(item, { status: 'done' }, 'Done.')}
								aria-label={t('Done: {title}', { title: item.title })}>{t('Done')}</Button
							>
						{/if}
						<Button
							variant="secondary"
							onclick={() => move(item, { status: 'next' }, 'A next action.')}
							aria-label={t('Next action: {title}', { title: item.title })}>{t('Next')}</Button
						>
						{#if current.id === 'inbox'}
							<Button
								variant="ghost"
								onclick={() => move(item, { status: 'waiting' }, 'Waiting for.')}
								aria-label={t('Waiting for: {title}', { title: item.title })}>{t('Waiting')}</Button
							>
							<Button
								variant="ghost"
								onclick={() => move(item, { status: 'someday' }, 'Someday, maybe.')}
								aria-label={t('Someday: {title}', { title: item.title })}>{t('Someday')}</Button
							>
						{/if}
						<Button
							variant="ghost"
							onclick={() => move(item, { delete: true }, 'Deleted.')}
							aria-label={t('Delete: {title}', { title: item.title })}>{t('Delete')}</Button
						>
					</li>
				{/each}
			</ul>
		{:else}
			<p class="empty">{current.id === 'inbox' ? t('Inbox zero.') : t('Nothing here.')}</p>
		{/if}
	{:else if current.id === 'notes'}
		<p>
			{t(
				'Rewrite each in your own words as a literature or permanent note, link it, or let it go.'
			)}
		</p>
		{#if notes.length}
			<ul class="rows">
				{#each notes as n (n.id)}<li><a href="/notes/{n.id}">{n.title}</a></li>{/each}
			</ul>
		{:else}
			<p class="empty">{t('No fleeting notes waiting.')}</p>
		{/if}
	{:else if current.id === 'back' && review}
		{#if review.highlights.length}
			<ul>
				{#each review.highlights as h (h)}<li>{h}</li>{/each}
			</ul>
		{:else}
			<p class="empty">{t('A steady week: nothing changed much.')}</p>
		{/if}
		<p class="small"><a href="/insights">{t('All metrics and what is behind them')}</a></p>
	{:else if current.id === 'ahead'}
		{#if ahead.length}
			<ul class="rows">
				{#each ahead as a, i (i)}
					<li>
						<span class="date"
							>{formatDate(a.date, { weekday: 'short', day: 'numeric', month: 'short' })}</span
						>
						<span class="grow">{a.title}</span><span class="muted small">{a.what}</span>
					</li>
				{/each}
			</ul>
		{:else}
			<p class="empty">{t('Nothing dated in the next two weeks.')}</p>
		{/if}
		<p class="small">
			{t('Anything to prepare? Add it in the first step, or on the')}
			<a href="/calendar">{t('Calendar')}</a>.
		</p>
	{:else if current.id === 'projects'}
		<p>{t('Every active project needs at least one next action.')}</p>
		{#if projects.length}
			<ul class="rows">
				{#each projects as p (p.id)}
					<li>
						<a href="/projects/{p.id}" class="grow">{p.title}</a>
						{#if p.total - p.done === 0}<span class="stuck">{t('No next action')}</span>{:else}<span
								class="muted small">{t('{value} open', { value: p.total - p.done })}</span
							>{/if}
					</li>
				{/each}
			</ul>
		{:else}
			<p class="empty">{t('No active project.')}</p>
		{/if}
	{:else if current.id === 'goals'}
		{#if goals.length}
			<h3>{t('Goals')}</h3>
			<ul class="rows">
				{#each goals as g (g.id)}
					<li>
						<a href="/goals/{g.id}" class="grow">{g.title}</a>
						{#if g.progress?.percent !== null && g.progress?.percent !== undefined}<span
								class="muted small">{g.progress.percent}%</span
							>{/if}
					</li>
				{/each}
			</ul>
		{/if}
		{#if plan}
			<h3>{t('Learning')}</h3>
			<p class="small">
				{t('{due} cards to review.', { due: plan.cards.due }) +
					(plan.cards.reviewed >= 10
						? ' ' +
							t('{pct}% remembered over 30 days.', { pct: Math.round(plan.cards.retention * 100) })
						: '') +
					(plan.cards.leeches ? ' ' + t('{n} keep slipping.', { n: plan.cards.leeches }) : '') +
					(plan.recall.due.length
						? ' ' + t('{n} notes to recall.', { n: plan.recall.due.length })
						: '')}
				<a href="/learn">{t('Learn → Today')}</a>
			</p>
		{/if}
		<p class="small"><a href="/habits">{t('Habits')}</a> · <a href="/goals">{t('All goals')}</a></p>
	{:else if current.id === 'done'}
		<form onsubmit={finish} class="reflect">
			<label for="well">{t('What went well?')}</label>
			<textarea id="well" rows="2" bind:value={reflection.well}></textarea>
			<label for="improve">{t('What would you do differently?')}</label>
			<textarea id="improve" rows="2" bind:value={reflection.improve}></textarea>
			<label for="focus">{t('What matters most next week?')}</label>
			<textarea id="focus" rows="2" bind:value={reflection.focus}></textarea>
			<Button type="submit">{t('Finish the review')}</Button>
		</form>
	{/if}

	<div class="nav">
		{#if step > 0}<Button variant="ghost" onclick={() => go(step - 1)}>{t('Back')}</Button>{/if}
		{#if step < steps.length - 1}<Button onclick={() => go(step + 1)}>{t('Next step')}</Button>{/if}
	</div>
</section>

<style>
	.steps {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		list-style: none;
		padding: 0;
		margin: var(--space-4) 0;
		counter-reset: step;
	}
	.steps a {
		display: block;
		padding: var(--space-1) var(--space-3);
		border-radius: 999px;
		font-size: var(--text-sm);
		color: var(--muted);
		text-decoration: none;
	}
	.steps .done a {
		color: var(--fg);
	}
	.steps a[aria-current='step'] {
		background: var(--accent-soft);
		color: var(--fg);
		font-weight: 600;
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-6);
	}
	.group {
		margin: 0;
		font-size: var(--text-sm);
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: var(--muted);
	}
	.rows {
		list-style: none;
		padding: 0;
		display: grid;
		gap: var(--space-2);
	}
	.rows li {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		flex-wrap: wrap;
	}
	.grow {
		flex: 1;
		min-width: 10rem;
	}
	.date {
		min-width: 7rem;
		font-variant-numeric: tabular-nums;
	}
	.stuck {
		font-size: var(--text-sm);
		color: var(--bad);
	}
	.empty {
		color: var(--muted);
	}
	.small {
		font-size: var(--text-sm);
	}
	.capture {
		display: flex;
		gap: var(--space-2);
	}
	.capture input {
		flex: 1;
		font: inherit;
	}
	.reflect {
		display: grid;
		gap: var(--space-2);
		justify-items: start;
	}
	.reflect textarea {
		width: 100%;
		font: inherit;
	}
	.nav {
		display: flex;
		justify-content: space-between;
		margin-top: var(--space-6);
	}
</style>
