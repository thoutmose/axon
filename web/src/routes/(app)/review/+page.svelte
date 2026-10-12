<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, apiResponse, message, type Schemas } from '$lib/api';
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

	const steps = [
		{ id: 'head', group: 'Get clear', title: 'Empty your head' },
		{ id: 'inbox', group: 'Get clear', title: 'Process your inbox' },
		{ id: 'notes', group: 'Get clear', title: 'Process your fleeting notes' },
		{ id: 'back', group: 'Get current', title: 'Look back at the week' },
		{ id: 'ahead', group: 'Get current', title: 'Look at the next two weeks' },
		{ id: 'waiting', group: 'Get current', title: 'Waiting for' },
		{ id: 'projects', group: 'Get current', title: 'Projects' },
		{ id: 'goals', group: 'Get current', title: 'Goals, habits, and learning' },
		{ id: 'someday', group: 'Get creative', title: 'Someday, maybe' },
		{ id: 'done', group: 'Wrap up', title: 'Into next week' }
	] as const;
	const step = $derived(
		Math.min(Math.max(Number(page.url.searchParams.get('step') ?? 0), 0), steps.length - 1)
	);
	const current = $derived(steps[step]);
	const go = (n: number) => goto(`?step=${n}`, { noScroll: false, keepFocus: false });

	const today = new Date();
	const iso = (d: Date) => d.toLocaleDateString('sv');
	const monday = (() => {
		const d = new Date(today);
		d.setDate(d.getDate() - ((d.getDay() + 6) % 7));
		return iso(d);
	})();
	const inDays = (n: number) => {
		const d = new Date(today);
		d.setDate(d.getDate() + n);
		return iso(d);
	};

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
					const q = new URLSearchParams({ from: iso(today), to: inDays(14) });
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
								: new Date(o.start).toLocaleTimeString(undefined, {
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
			reflection.well && `Went well: ${reflection.well}`,
			reflection.improve && `To improve: ${reflection.improve}`,
			reflection.focus && `Focus next week: ${reflection.focus}`
		]
			.filter(Boolean)
			.join('\n')
			.slice(0, 2000);
		try {
			await api(`/v1/checkins/weekly/${monday}`, { method: 'PUT', body: { note } });
			toast('Weekly review done. Enjoy the week.');
			await goto('/today');
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<svelte:head><title>Weekly review · Axon</title></svelte:head>

<h1>Weekly review</h1>
<p class="muted">
	About twenty minutes, once a week: clear your head and your inboxes, see where things stand, and
	choose what matters next week (Getting Things Done).
</p>

<nav aria-label="Review steps">
	<ol class="steps">
		{#each steps as s, i (s.id)}
			<li class:done={i < step}>
				<a href="?step={i}" aria-current={i === step ? 'step' : undefined}>{s.title}</a>
			</li>
		{/each}
	</ol>
</nav>

<section class="card" aria-labelledby="step-title">
	<p class="group">{current.group} · {step + 1} of {steps.length}</p>
	<h2 id="step-title">{current.title}</h2>

	{#if error}
		<Status kind="error" message={error} retry={load} />
	{:else if loading}
		<Status kind="loading" message="Loading…" />
	{:else if current.id === 'head'}
		<p>
			Write down everything on your mind, one line at a time: tasks, ideas, books, anything. Sort it
			out in the next step.
		</p>
		<form onsubmit={capture} class="capture">
			<label for="review-line" class="visually-hidden">One thing on your mind</label>
			<input
				id="review-line"
				bind:value={line}
				placeholder="Call the bank @phone friday"
				autocomplete="off"
			/>
			<Button type="submit" variant="secondary">Add</Button>
		</form>
		{#if captured.length}<ul class="small">
				{#each captured as c, i (i)}<li>{c}</li>{/each}
			</ul>{/if}
	{:else if current.id === 'inbox' || current.id === 'waiting' || current.id === 'someday'}
		{#if current.id === 'inbox'}
			<p>For each: what is the next action? Do it now if it takes two minutes.</p>
		{:else if current.id === 'waiting'}
			<p>Still waiting? Nudge, or take it back.</p>
		{:else}
			<p>Anything to start now? Anything no longer worth keeping?</p>
		{/if}
		{#if tasks.length}
			<ul class="rows">
				{#each tasks as t (t.id)}
					<li>
						<a href="/tasks/{t.id}" class="grow">{t.title}</a>
						{#if current.id !== 'someday'}
							<Button
								variant="ghost"
								onclick={() => move(t, { status: 'done' }, 'Done.')}
								aria-label="Done: {t.title}">Done</Button
							>
						{/if}
						<Button
							variant="secondary"
							onclick={() => move(t, { status: 'next' }, 'A next action.')}
							aria-label="Next action: {t.title}">Next</Button
						>
						{#if current.id === 'inbox'}
							<Button
								variant="ghost"
								onclick={() => move(t, { status: 'waiting' }, 'Waiting for.')}
								aria-label="Waiting for: {t.title}">Waiting</Button
							>
							<Button
								variant="ghost"
								onclick={() => move(t, { status: 'someday' }, 'Someday, maybe.')}
								aria-label="Someday: {t.title}">Someday</Button
							>
						{/if}
						<Button
							variant="ghost"
							onclick={() => move(t, { delete: true }, 'Deleted.')}
							aria-label="Delete: {t.title}">Delete</Button
						>
					</li>
				{/each}
			</ul>
		{:else}
			<p class="empty">{current.id === 'inbox' ? 'Inbox zero.' : 'Nothing here.'}</p>
		{/if}
	{:else if current.id === 'notes'}
		<p>Rewrite each in your own words as a literature or permanent note, link it, or let it go.</p>
		{#if notes.length}
			<ul class="rows">
				{#each notes as n (n.id)}<li><a href="/notes/{n.id}">{n.title}</a></li>{/each}
			</ul>
		{:else}
			<p class="empty">No fleeting notes waiting.</p>
		{/if}
	{:else if current.id === 'back' && review}
		{#if review.highlights.length}
			<ul>
				{#each review.highlights as h (h)}<li>{h}</li>{/each}
			</ul>
		{:else}
			<p class="empty">A steady week: nothing changed much.</p>
		{/if}
		<p class="small"><a href="/insights">All metrics and what is behind them</a></p>
	{:else if current.id === 'ahead'}
		{#if ahead.length}
			<ul class="rows">
				{#each ahead as a, i (i)}
					<li>
						<span class="date"
							>{new Date(`${a.date}T00:00:00`).toLocaleDateString(undefined, {
								weekday: 'short',
								day: 'numeric',
								month: 'short'
							})}</span
						>
						<span class="grow">{a.title}</span><span class="muted small">{a.what}</span>
					</li>
				{/each}
			</ul>
		{:else}
			<p class="empty">Nothing dated in the next two weeks.</p>
		{/if}
		<p class="small">
			Anything to prepare? Add it in the first step, or on the <a href="/calendar">Calendar</a>.
		</p>
	{:else if current.id === 'projects'}
		<p>Every active project needs at least one next action.</p>
		{#if projects.length}
			<ul class="rows">
				{#each projects as p (p.id)}
					<li>
						<a href="/projects/{p.id}" class="grow">{p.title}</a>
						{#if p.total - p.done === 0}<span class="stuck">No next action</span>{:else}<span
								class="muted small">{p.total - p.done} open</span
							>{/if}
					</li>
				{/each}
			</ul>
		{:else}
			<p class="empty">No active project.</p>
		{/if}
	{:else if current.id === 'goals'}
		{#if goals.length}
			<h3>Goals</h3>
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
			<h3>Learning</h3>
			<p class="small">
				{plan.cards.due} cards to review{plan.cards.reviewed >= 10
					? `, ${Math.round(plan.cards.retention * 100)}% remembered over 30 days`
					: ''}{plan.cards.leeches ? `, ${plan.cards.leeches} that keep slipping` : ''}.
				{plan.recall.due.length ? `${plan.recall.due.length} notes to recall.` : ''}
				<a href="/learn">Learn → Today</a>
			</p>
		{/if}
		<p class="small"><a href="/habits">Habits</a> · <a href="/goals">All goals</a></p>
	{:else if current.id === 'done'}
		<form onsubmit={finish} class="reflect">
			<label for="well">What went well?</label>
			<textarea id="well" rows="2" bind:value={reflection.well}></textarea>
			<label for="improve">What would you do differently?</label>
			<textarea id="improve" rows="2" bind:value={reflection.improve}></textarea>
			<label for="focus">What matters most next week?</label>
			<textarea id="focus" rows="2" bind:value={reflection.focus}></textarea>
			<Button type="submit">Finish the review</Button>
		</form>
	{/if}

	<div class="nav">
		{#if step > 0}<Button variant="ghost" onclick={() => go(step - 1)}>Back</Button>{/if}
		{#if step < steps.length - 1}<Button onclick={() => go(step + 1)}>Next step</Button>{/if}
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
