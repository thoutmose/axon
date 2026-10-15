<script lang="ts">
	import { appLocale } from '\$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import { deleted } from '$lib/undo';
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

	const views = $derived([
		{ id: 'gtd', label: t('GTD board') },
		{ id: 'kanban', label: t('Kanban') },
		{ id: 'today', label: t('Today') },
		{ id: 'overdue', label: t('Overdue') },
		{ id: 'week', label: t('Week') },
		{ id: 'month', label: t('Month') },
		{ id: 'inbox', label: t('Inbox') },
		{ id: 'next', label: t('Next') },
		{ id: 'doing', label: t('In progress') },
		{ id: 'waiting', label: t('Waiting') },
		{ id: 'someday', label: t('Someday') },
		{ id: 'done', label: t('Done') }
	]);
	const empty: Record<string, string> = $derived({
		today: t('Nothing due or planned for today.'),
		overdue: t('Nothing overdue.'),
		week: t('Nothing planned this week.'),
		month: t('Nothing planned in the next 30 days.'),
		inbox: t('Inbox zero. Press n anywhere to capture something.'),
		next: t('No next actions. Clarify your inbox to find some.'),
		doing: t('Nothing in progress.'),
		waiting: t('Not waiting on anyone.'),
		someday: t('No someday/maybe ideas yet.'),
		done: t('Nothing completed yet.')
	});

	// Boards show several statuses side by side; each column is one status.
	const boards: Record<string, Omit<BoardColumn, 'tasks'>[]> = $derived({
		gtd: [
			{ status: 'inbox', label: t('Inbox'), hint: t('Captured, not yet clarified.') },
			{
				status: 'next',
				label: t('Next actions'),
				hint: t('The very next physical step, by context.'),
				byContext: true
			},
			{ status: 'waiting', label: t('Waiting for'), hint: t('Delegated or blocked on someone.') },
			{ status: 'someday', label: t('Someday / maybe'), hint: t('Not now; review weekly.') }
		],
		kanban: [
			{ status: 'next', label: t('To do'), hint: t('Ready to start.') },
			{ status: 'doing', label: t('In progress'), hint: t('Keep this short.') },
			{ status: 'waiting', label: t('Blocked'), hint: t('Waiting on someone or something.') },
			{ status: 'done', label: t('Done'), hint: t('The most recent 20.') }
		]
	});

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
	// Tasks done today stay in their list, crossed out, until tomorrow (EP-116).
	let doneToday = $state<Task[]>([]);
	const rows = $derived([...(tasks ?? []), ...doneToday]);

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
			const res = await api<{
				items: Task[];
				today: string;
				next_cursor: string | null;
				done_today?: Task[];
			}>(`/v1/tasks?${params}`);
			tasks = more && tasks ? [...tasks, ...res.items] : res.items;
			if (!more) doneToday = res.done_today ?? [];
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

	async function toggle(task: Task) {
		try {
			const { etag } = await apiResponse<Task>(`/v1/tasks/${task.id}`);
			const done = task.status !== 'done';
			await api(`/v1/tasks/${task.id}`, {
				method: 'PATCH',
				headers: { 'If-Match': etag },
				body: { status: done ? 'done' : (task.done_from ?? 'next') }
			});
			toast(
				done
					? task.recurrence
						? t('Done. The next one is scheduled.')
						: t('Done.')
					: t('Reopened.')
			);
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function move(task: Task, status: Task['status']) {
		if (task.status === status) return;
		try {
			const { etag } = await apiResponse<Task>(`/v1/tasks/${task.id}`);
			await api(`/v1/tasks/${task.id}`, {
				method: 'PATCH',
				headers: { 'If-Match': etag },
				body: { status }
			});
			const to = board?.find((c) => c.status === status)?.label ?? status;
			toast(t('Moved “{title}” to {to}.', { title: task.title, to }));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function checkIn() {
		try {
			await api(`/v1/checkins/daily/${today}`, { method: 'PUT', body: {} });
			toast(t('Checked in for today.'));
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

	// A reminder's notification opens its task (EP-68).
	$effect(() => {
		const open = page.url.searchParams.get('open');
		if (open) editing = open;
	});

	// Bulk actions (EP-64): select tasks, then change them all at once.
	let selecting = $state(false);
	let picked = $state<string[]>([]);
	let bulkContext = $state('');
	let bulkDue = $state('');
	function pick(id: string, on: boolean) {
		picked = on ? [...picked, id] : picked.filter((x) => x !== id);
	}
	async function bulk(body: Record<string, unknown>, said: string) {
		try {
			const ids = picked;
			const r = await api<{ changed: number }>('/v1/tasks/bulk', {
				method: 'POST',
				body: { ids, ...body }
			});
			const text = `${r.changed} ${r.changed === 1 ? 'task' : 'tasks'} ${said}.`;
			if (body.delete) deleted(text, ids[0], () => load());
			else toast(text);
			picked = [];
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<svelte:head><title>{t('Tasks · Axon')}</title></svelte:head>

<div class="head">
	<h1>{t('Tasks')}</h1>
	{#if view === 'today' && today}
		<Button variant={checkedIn ? 'ghost' : 'secondary'} onclick={checkIn} disabled={checkedIn}>
			{checkedIn ? t('✓ Checked in today') : t('Daily check-in')}
		</Button>
	{/if}
</div>

<AgentDock {...pageAgents.tasks} onapplied={() => load()} />
<p class="muted hint">
	{t('Press')} <kbd>n</kbd>
	{t('anywhere to capture a thought to your inbox.')}
</p>

<nav aria-label={t('Task views')} class="tabs">
	{#each views as v (v.id)}
		<a href="?view={v.id}" aria-current={view === v.id ? 'page' : undefined}>{v.label}</a>
	{/each}
</nav>

{#if contexts.length}
	<div class="contexts" role="group" aria-label={t('Filter by context')}>
		<button class:active={!context} onclick={() => filter(null)}>{t('All')}</button>
		{#each contexts as c (c.context)}
			<button class:active={context === c.context} onclick={() => filter(c.context)}
				>{c.context} <span class="muted">{c.open}</span></button
			>
		{/each}
	</div>
{/if}

{#if view !== 'done'}
	<form class="quick" onsubmit={quickAdd}>
		<label for="quick" class="visually-hidden">{t('Add a task')}</label>
		<input
			id="quick"
			placeholder={t('Add a task…')}
			bind:value={quick}
			maxlength="500"
			required
			autocomplete="off"
		/>
		<Button type="submit" variant="secondary">{t('Add')}</Button>
	</form>
{/if}

{#if error}
	<Status kind="error" message={error} retry={() => load()} />
{:else if board}
	{#if columns}
		<TaskBoard
			label={view === 'gtd' ? t('GTD board') : t('Kanban board')}
			{columns}
			{today}
			onmove={move}
			onopen={(id) => (editing = id)}
			ontoggle={toggle}
		/>
	{:else}
		<Status kind="loading" message={t('Loading tasks…')} />
	{/if}
{:else if !tasks}
	<Status kind="loading" message={t('Loading tasks…')} />
{:else if rows.length === 0}
	<Status kind="empty" message={empty[view] ?? t('Nothing here.')} />
{:else}
	<div class="select-bar">
		<Button
			variant="ghost"
			onclick={() => {
				selecting = !selecting;
				picked = [];
			}}>{selecting ? t('Done selecting') : t('Select')}</Button
		>
		{#if selecting}
			<Button variant="ghost" onclick={() => (picked = tasks!.map((t) => t.id))}
				>{t('Select all')}</Button
			>
		{/if}
	</div>
	{#if selecting && picked.length}
		<div
			class="bulk"
			role="group"
			aria-label={t('Change {length} selected tasks', { length: picked.length })}
		>
			<strong>{t('{length} selected', { length: picked.length })}</strong>
			<Button
				variant="secondary"
				onclick={() => bulk({ change: { status: 'next' } }, 'moved to Next')}>{t('Next')}</Button
			>
			<Button variant="secondary" onclick={() => bulk({ change: { status: 'waiting' } }, 'waiting')}
				>{t('Waiting')}</Button
			>
			<Button
				variant="secondary"
				onclick={() => bulk({ change: { status: 'someday' } }, 'set to someday')}
				>{t('Someday')}</Button
			>
			<Button variant="secondary" onclick={() => bulk({ change: { status: 'done' } }, 'done')}
				>{t('Done')}</Button
			>
			<form
				onsubmit={(e) => {
					e.preventDefault();
					bulk({ change: { context: bulkContext.trim() || null } }, 'given a context');
				}}
			>
				<label for="bulk-context" class="visually-hidden"
					>{t('Context for the selected tasks')}</label
				>
				<input
					id="bulk-context"
					placeholder={t('@context')}
					bind:value={bulkContext}
					maxlength="50"
				/>
				<Button type="submit" variant="ghost">{t('Set context')}</Button>
			</form>
			<form
				onsubmit={(e) => {
					e.preventDefault();
					bulk(
						{ change: { due_on: bulkDue || null } },
						bulkDue ? 'given a due date' : 'without a due date'
					);
				}}
			>
				<label for="bulk-due" class="visually-hidden">{t('Due date for the selected tasks')}</label>
				<input id="bulk-due" type="date" bind:value={bulkDue} />
				<Button type="submit" variant="ghost">{t('Set due date')}</Button>
			</form>
			<Button variant="danger" onclick={() => bulk({ delete: true }, 'deleted')}
				>{t('Delete')}</Button
			>
		</div>
	{/if}
	<ul class="tasks">
		{#each rows as row (row.id)}
			<li
				class:done={row.status === 'done' || row.status === 'cancelled'}
				class:picked={picked.includes(row.id)}
			>
				{#if selecting}
					<input
						type="checkbox"
						class="pick"
						checked={picked.includes(row.id)}
						onchange={(e) => pick(row.id, e.currentTarget.checked)}
						aria-label={t('Select {title}', { title: row.title })}
					/>
				{:else}
					<input
						type="checkbox"
						checked={row.status === 'done'}
						onchange={() => toggle(row)}
						aria-label={t('Complete {title}', { title: row.title })}
					/>
				{/if}
				<button class="title" onclick={() => (editing = row.id)}>{row.title}</button>
				<span class="meta">
					{#if row.recurrence}<span title={t('Repeats')} aria-label={t('Repeats')}>↻</span>{/if}
					{#if row.remind_at && new Date(row.remind_at).getTime() > Date.now()}<span
							title={t('Reminder')}
							>{'🔔 ' +
								new Date(row.remind_at).toLocaleString(appLocale(), {
									weekday: 'short',
									hour: '2-digit',
									minute: '2-digit'
								})}</span
						>{/if}
					{#if row.context}<span class="chip">{row.context}</span>{/if}
					{#if row.due_on}<span class:overdue={overdue(row, today)}
							>{t('Due {value}', { value: relativeDay(row.due_on, today) })}</span
						>
					{:else if row.scheduled_on}<span>{relativeDay(row.scheduled_on, today)}</span>{/if}
				</span>
			</li>
		{/each}
	</ul>
	{#if cursor}<Button variant="secondary" onclick={() => load(true)}>{t('Show more')}</Button>{/if}
{/if}

<TaskEditor bind:id={editing} onchange={() => load()} />

<style>
	.select-bar {
		display: flex;
		gap: var(--space-1);
		justify-content: flex-end;
	}
	.bulk {
		position: sticky;
		top: 3.5rem;
		z-index: 5;
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
		padding: var(--space-2) var(--space-3);
		margin-bottom: var(--space-2);
		border: 1px solid var(--accent);
		border-radius: var(--radius-lg);
		background: var(--surface-solid);
	}
	.bulk form {
		display: flex;
		gap: var(--space-1);
		align-items: center;
	}
	.bulk input {
		font: inherit;
		max-width: 10rem;
	}
	.picked {
		background: var(--accent-soft);
	}
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
