<script lang="ts">
	import AgentDock from '$lib/AgentDock.svelte';
	import { pageAgents } from '$lib/pageAgents';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, apiResponse, message } from '$lib/api';
	import { capturing } from '$lib/capture.svelte';
	import TaskBoard, { type BoardColumn } from '$lib/TaskBoard.svelte';
	import TaskEditor from '$lib/TaskEditor.svelte';
	import { capture, overdue, relativeDay, type Task } from '$lib/tasks';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	const views = [
		{ id: 'gtd', label: 'GTD board' },
		{ id: 'kanban', label: 'Kanban' },
		{ id: 'today', label: 'Today' },
		{ id: 'overdue', label: 'Overdue' },
		{ id: 'week', label: 'Week' },
		{ id: 'month', label: 'Month' },
		{ id: 'inbox', label: 'Inbox' },
		{ id: 'next', label: 'Next' },
		{ id: 'doing', label: 'In progress' },
		{ id: 'waiting', label: 'Waiting' },
		{ id: 'someday', label: 'Someday' },
		{ id: 'done', label: 'Done' }
	];
	const empty: Record<string, string> = {
		today: 'Nothing due or planned for today.',
		overdue: 'Nothing overdue.',
		week: 'Nothing planned this week.',
		month: 'Nothing planned in the next 30 days.',
		inbox: 'Inbox zero. Press n anywhere to capture something.',
		next: 'No next actions. Clarify your inbox to find some.',
		doing: 'Nothing in progress.',
		waiting: 'Not waiting on anyone.',
		someday: 'No someday/maybe ideas yet.',
		done: 'Nothing completed yet.'
	};

	// Boards show several statuses side by side; each column is one status.
	const boards: Record<string, Omit<BoardColumn, 'tasks'>[]> = {
		gtd: [
			{ status: 'inbox', label: 'Inbox', hint: 'Captured, not yet clarified.' },
			{
				status: 'next',
				label: 'Next actions',
				hint: 'The very next physical step, by context.',
				byContext: true
			},
			{ status: 'waiting', label: 'Waiting for', hint: 'Delegated or blocked on someone.' },
			{ status: 'someday', label: 'Someday / maybe', hint: 'Not now; review weekly.' }
		],
		kanban: [
			{ status: 'next', label: 'To do', hint: 'Ready to start.' },
			{ status: 'doing', label: 'In progress', hint: 'Keep this short.' },
			{ status: 'waiting', label: 'Blocked', hint: 'Waiting on someone or something.' },
			{ status: 'done', label: 'Done', hint: 'The most recent 20.' }
		]
	};

	const view = $derived(page.url.searchParams.get('view') ?? 'gtd');
	const board = $derived(boards[view]);
	let columns = $state<BoardColumn[] | null>(null);
	const context = $derived(page.url.searchParams.get('context'));
	let tasks = $state<Task[] | null>(null);
	let today = $state('');
	let cursor = $state<string | null>(null);
	let contexts = $state<{ context: string; open: number }[]>([]);
	let error = $state('');
	let quick = $state('');
	let editing = $state<string | null>(null);
	let checkedIn = $state(false);

	async function load(more = false) {
		error = '';
		const params = new URLSearchParams({ view, limit: '50' });
		if (context) params.set('context', context);
		if (more && cursor) params.set('after', cursor);
		try {
			if (board) {
				const cols = board;
				const lists = await Promise.all(
					cols.map((c) => {
						const p = new URLSearchParams({
							view: c.status,
							limit: c.status === 'done' ? '20' : '100'
						});
						if (context) p.set('context', context);
						return api<{ items: Task[]; today: string }>(`/v1/tasks?${p}`);
					})
				);
				columns = cols.map((c, i) => ({ ...c, tasks: lists[i].items }));
				today = lists[0]?.today ?? today;
				tasks = lists.flatMap((l) => l.items);
				cursor = null;
				contexts = (await api<{ items: { context: string; open: number }[] }>('/v1/contexts'))
					.items;
				return;
			}
			columns = null;
			const res = await api<{ items: Task[]; today: string; next_cursor: string | null }>(
				`/v1/tasks?${params}`
			);
			tasks = more && tasks ? [...tasks, ...res.items] : res.items;
			cursor = res.next_cursor;
			today = res.today;
			contexts = (await api<{ items: { context: string; open: number }[] }>('/v1/contexts')).items;
			if (view === 'today') {
				const c = await api<{ items: { kind: string; day: string }[] }>(
					`/v1/checkins?from=${today}&to=${today}`
				);
				checkedIn = c.items.some((x) => x.kind === 'daily');
			}
		} catch (err) {
			error = message(err);
		}
	}

	$effect(() => {
		void view;
		void context;
		void capturing.count;
		tasks = null;
		columns = null;
		load();
	});

	async function quickAdd(e: SubmitEvent) {
		e.preventDefault();
		try {
			const status =
				view === 'inbox' || view === 'done' || view === 'gtd'
					? 'inbox'
					: ['kanban', 'week', 'month', 'today', 'overdue'].includes(view)
						? 'next'
						: view;
			await capture(quick, {
				status: status as Task['status'],
				...(view === 'today' ? { scheduled_on: today } : {}),
				...(context ? { context } : {})
			});
			quick = '';
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function toggle(t: Task) {
		try {
			const { etag } = await apiResponse<Task>(`/v1/tasks/${t.id}`);
			const done = t.status !== 'done';
			await api(`/v1/tasks/${t.id}`, {
				method: 'PATCH',
				headers: { 'If-Match': etag },
				body: { status: done ? 'done' : 'next' }
			});
			toast(done ? (t.recurrence ? 'Done. The next one is scheduled.' : 'Done.') : 'Reopened.');
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function move(t: Task, status: Task['status']) {
		if (t.status === status) return;
		try {
			const { etag } = await apiResponse<Task>(`/v1/tasks/${t.id}`);
			await api(`/v1/tasks/${t.id}`, {
				method: 'PATCH',
				headers: { 'If-Match': etag },
				body: { status }
			});
			const to = board?.find((c) => c.status === status)?.label ?? status;
			toast(`Moved “${t.title}” to ${to}.`);
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function checkIn() {
		try {
			await api(`/v1/checkins/daily/${today}`, { method: 'PUT', body: {} });
			toast('Checked in for today.');
			checkedIn = true;
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function filter(c: string | null) {
		const u = new URL(page.url);
		if (c) u.searchParams.set('context', c);
		else u.searchParams.delete('context');
		goto(u, { keepFocus: true, noScroll: true });
	}
</script>

<svelte:head><title>Tasks · Axon</title></svelte:head>

<div class="head">
	<h1>Tasks</h1>
	{#if view === 'today' && today}
		<Button variant={checkedIn ? 'ghost' : 'secondary'} onclick={checkIn} disabled={checkedIn}>
			{checkedIn ? '✓ Checked in today' : 'Daily check-in'}
		</Button>
	{/if}
</div>

<AgentDock {...pageAgents.tasks} onapplied={() => load()} />
<p class="muted hint">Press <kbd>n</kbd> anywhere to capture a thought to your inbox.</p>

<nav aria-label="Task views" class="tabs">
	{#each views as v (v.id)}
		<a href="?view={v.id}" aria-current={view === v.id ? 'page' : undefined}>{v.label}</a>
	{/each}
</nav>

{#if contexts.length}
	<div class="contexts" role="group" aria-label="Filter by context">
		<button class:active={!context} onclick={() => filter(null)}>All</button>
		{#each contexts as c (c.context)}
			<button class:active={context === c.context} onclick={() => filter(c.context)}
				>{c.context} <span class="muted">{c.open}</span></button
			>
		{/each}
	</div>
{/if}

{#if view !== 'done'}
	<form class="quick" onsubmit={quickAdd}>
		<label for="quick" class="visually-hidden">Add a task</label>
		<input
			id="quick"
			placeholder="Add a task…"
			bind:value={quick}
			maxlength="500"
			required
			autocomplete="off"
		/>
		<Button type="submit" variant="secondary">Add</Button>
	</form>
{/if}

{#if error}
	<Status kind="error" message={error} retry={() => load()} />
{:else if board}
	{#if columns}
		<TaskBoard
			label={view === 'gtd' ? 'GTD board' : 'Kanban board'}
			{columns}
			{today}
			onmove={move}
			onopen={(id) => (editing = id)}
			ontoggle={toggle}
		/>
	{:else}
		<Status kind="loading" message="Loading tasks…" />
	{/if}
{:else if !tasks}
	<Status kind="loading" message="Loading tasks…" />
{:else if tasks.length === 0}
	<Status kind="empty" message={empty[view] ?? 'Nothing here.'} />
{:else}
	<ul class="tasks">
		{#each tasks as t (t.id)}
			<li class:done={t.status === 'done' || t.status === 'cancelled'}>
				<input
					type="checkbox"
					checked={t.status === 'done'}
					onchange={() => toggle(t)}
					aria-label={`Complete ${t.title}`}
				/>
				<button class="title" onclick={() => (editing = t.id)}>{t.title}</button>
				<span class="meta">
					{#if t.recurrence}<span title="Repeats" aria-label="Repeats">↻</span>{/if}
					{#if t.context}<span class="chip">{t.context}</span>{/if}
					{#if t.due_on}<span class:overdue={overdue(t, today)}
							>Due {relativeDay(t.due_on, today)}</span
						>
					{:else if t.scheduled_on}<span>{relativeDay(t.scheduled_on, today)}</span>{/if}
				</span>
			</li>
		{/each}
	</ul>
	{#if cursor}<Button variant="secondary" onclick={() => load(true)}>Show more</Button>{/if}
{/if}

<TaskEditor bind:id={editing} onchange={() => load()} />

<style>
	.head {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-4);
		flex-wrap: wrap;
	}
	.hint {
		font-size: var(--text-sm);
	}
	kbd {
		font-family: var(--font-mono);
		border: 1px solid var(--border);
		border-bottom-width: 2px;
		border-radius: var(--radius-sm);
		padding: 0 0.35em;
	}
	.tabs {
		display: flex;
		gap: var(--space-1);
		overflow-x: auto;
		margin: var(--space-4) 0;
	}
	.tabs a {
		white-space: nowrap;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		color: var(--muted);
		text-decoration: none;
	}
	.tabs a[aria-current='page'] {
		background: var(--accent-soft);
		color: var(--fg);
		font-weight: 600;
	}
	.contexts {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}
	.contexts button {
		font: inherit;
		font-size: var(--text-sm);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		border-radius: 999px;
		padding: var(--space-1) var(--space-3);
		cursor: pointer;
		min-height: 2rem;
	}
	.contexts button.active {
		border-color: var(--accent);
		background: var(--accent-soft);
	}
	.quick {
		display: flex;
		gap: var(--space-2);
		margin-bottom: var(--space-6);
	}
	.quick input {
		flex: 1;
		min-width: 0;
		font: inherit;
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	.tasks {
		list-style: none;
		padding: 0;
		margin: 0 0 var(--space-6);
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
	}
	.tasks li {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-3) var(--space-4);
		border-bottom: 1px solid var(--border);
		flex-wrap: wrap;
	}
	.tasks li:last-child {
		border-bottom: none;
	}
	.tasks input[type='checkbox'] {
		width: 1.25rem;
		height: 1.25rem;
		accent-color: var(--accent);
	}
	.title {
		flex: 1;
		min-width: 8rem;
		text-align: left;
		font: inherit;
		color: var(--fg);
		background: none;
		border: none;
		padding: var(--space-1) 0;
		cursor: pointer;
	}
	.done .title {
		color: var(--muted);
		text-decoration: line-through;
	}
	.meta {
		display: flex;
		gap: var(--space-2);
		align-items: center;
		color: var(--muted);
		font-size: var(--text-sm);
	}
	.chip {
		background: var(--accent-soft);
		color: var(--fg);
		border-radius: 999px;
		padding: 0 var(--space-2);
	}
	.overdue {
		color: var(--bad);
		font-weight: 600;
	}
</style>
