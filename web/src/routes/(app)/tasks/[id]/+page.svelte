<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, apiResponse, message, type Schemas } from '$lib/api';
	import Connections from '$lib/Connections.svelte';
	import { addDays, localDate } from '$lib/dates';
	import TaskEditor from '$lib/TaskEditor.svelte';
	import type { Task } from '$lib/tasks';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// A task you can act on (owner request: opened from the calendar, it only
	// showed its description): done, started, moved, focused on, deleted, with
	// the project board it sits on, its parent card, and its subtasks.
	type Ref = Schemas['CardRef'];
	type Card = { project: Ref; column: string; kind: string; parent: Ref | null; children: Ref[] };

	const id = $derived(page.params.id ?? '');
	let task = $state<Task | null>(null);
	let etag = '';
	let card = $state<Card | null>(null);
	let error = $state('');
	let editing = $state<string | null>(null);
	let busy = $state(false);
	const today = localDate(new Date());

	const statusNames: Record<string, string> = $derived({
		inbox: t('Inbox'),
		next: t('Next action'),
		doing: t('In progress'),
		waiting: t('Waiting for'),
		someday: t('Someday / maybe'),
		done: t('Done'),
		cancelled: t('Cancelled')
	});
	const kindNames: Record<string, string> = $derived({
		epic: t('Epic'),
		issue: t('Issue'),
		subtask: t('Subtask'),
		task: t('Card')
	});

	async function load() {
		error = '';
		try {
			const r = await apiResponse<Task>(`/v1/tasks/${id}`);
			task = r.data;
			etag = r.etag;
			card = (await api<{ card: Card | null }>(`/v1/tasks/${id}/card`)).card;
		} catch (err) {
			error = message(err);
		}
	}

	$effect(() => {
		void id;
		task = null;
		card = null;
		load();
	});

	async function change(body: Record<string, unknown>, done: string) {
		busy = true;
		try {
			await api(`/v1/tasks/${id}`, { method: 'PATCH', headers: { 'If-Match': etag }, body });
			toast(done);
			await load();
		} catch (err) {
			toast(message(err), 'error');
			await load();
		} finally {
			busy = false;
		}
	}
	// Moving a task moves the date it has: its due date, else the day to do it.
	const field = $derived(task?.due_on && !task?.scheduled_on ? 'due_on' : 'scheduled_on');
	function moveTo(day: string) {
		change({ [field]: day }, t('Moved to {day}.', { day }));
	}
	function nextMonday() {
		const d = new Date(today + 'T12:00:00');
		return addDays(today, (8 - d.getDay()) % 7 || 7);
	}
	async function remove() {
		if (!task || !confirm(t('Delete “{title}”? It goes to the trash.', { title: task.title })))
			return;
		try {
			await api(`/v1/tasks/${id}`, { method: 'DELETE' });
			toast(t('Moved to the trash.'));
			goto(card ? `/projects/${card.project.id}` : '/tasks');
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	async function toggleChild(c: { id: string; status?: string }) {
		try {
			const r = await apiResponse(`/v1/tasks/${c.id}`);
			await api(`/v1/tasks/${c.id}`, {
				method: 'PATCH',
				headers: { 'If-Match': r.etag },
				body: { status: c.status === 'done' ? 'next' : 'done' }
			});
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	const closed = $derived(task?.status === 'done' || task?.status === 'cancelled');
</script>

<svelte:head><title>{task?.title ?? t('Task')} · Axon</title></svelte:head>

<p class="crumbs">
	{#if card}<a href="/projects/{card.project.id}">← {card.project.title}</a>{:else}<a href="/tasks"
			>{t('← Tasks')}</a
		>{/if}
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !task}
	<Status kind="loading" message={t('Loading task…')} />
{:else}
	<div class="head">
		<h1 class:closed>{task.title}</h1>
		<span class="chip {task.status}">{statusNames[task.status]}</span>
	</div>

	<div class="actions" role="group" aria-label={t('Actions')}>
		{#if closed}
			<Button {busy} onclick={() => change({ status: 'next' }, t('Reopened.'))}
				>{t('Reopen')}</Button
			>
		{:else}
			<Button {busy} onclick={() => change({ status: 'done' }, t('Done. Well done!'))}
				>✓ {t('Mark done')}</Button
			>
			{#if task.status !== 'doing'}
				<Button
					variant="secondary"
					{busy}
					onclick={() => change({ status: 'doing' }, t('Started.'))}>▶ {t('Start')}</Button
				>
			{/if}
			<Button variant="secondary" href="/focus?target={task.id}">{t('Focus on it')}</Button>
			<Button variant="secondary" {busy} onclick={() => moveTo(addDays(today, 1))}
				>{t('Tomorrow')}</Button
			>
			<Button variant="secondary" {busy} onclick={() => moveTo(nextMonday())}
				>{t('Next week')}</Button
			>
		{/if}
		<Button variant="ghost" onclick={() => (editing = task?.id ?? null)}>{t('Edit')}</Button>
		<Button variant="ghost" onclick={remove}>{t('Delete')}</Button>
	</div>

	<dl class="facts">
		<dt>{t('Status')}</dt>
		<dd>
			<select
				value={task.status}
				disabled={busy}
				onchange={(e) => change({ status: e.currentTarget.value }, t('Saved.'))}
				aria-label={t('Status')}
			>
				{#each Object.entries(statusNames) as [v, l] (v)}<option value={v}>{l}</option>{/each}
			</select>
		</dd>
		<dt><label for="do-on">{t('Do on')}</label></dt>
		<dd>
			<input
				id="do-on"
				type="date"
				value={task.scheduled_on ?? ''}
				disabled={busy}
				onchange={(e) => change({ scheduled_on: e.currentTarget.value || null }, t('Saved.'))}
			/>
		</dd>
		<dt><label for="due-on">{t('Due')}</label></dt>
		<dd>
			<input
				id="due-on"
				type="date"
				value={task.due_on ?? ''}
				disabled={busy}
				onchange={(e) => change({ due_on: e.currentTarget.value || null }, t('Saved.'))}
			/>
		</dd>
		{#if task.context}<dt>{t('Context')}</dt>
			<dd>{task.context}</dd>{/if}
		{#if task.recurrence}<dt>{t('Repeats')}</dt>
			<dd>{task.recurrence}</dd>{/if}
		{#if card}
			<dt>{t('Project')}</dt>
			<dd>
				<a href="/projects/{card.project.id}">{card.project.title}</a>
				<span class="muted small">· {kindNames[card.kind] ?? card.kind} · {card.column}</span>
			</dd>
			{#if card.parent}
				<dt>{t('Part of')}</dt>
				<dd><a href="/tasks/{card.parent.id}">{card.parent.title}</a></dd>
			{/if}
		{/if}
	</dl>

	{#if task.notes}<p class="notes">{task.notes}</p>{/if}

	{#if card && card.children.length}
		<section aria-labelledby="subtasks">
			<h2 id="subtasks">
				{t('Subtasks')}
				<span class="muted small"
					>{card.children.filter((c) => c.status === 'done').length}/{card.children.length}</span
				>
			</h2>
			<ul class="children">
				{#each card.children as c (c.id)}
					<li>
						<input
							type="checkbox"
							checked={c.status === 'done'}
							onchange={() => toggleChild(c)}
							aria-label={t('Done: {title}', { title: c.title })}
						/>
						<a href="/tasks/{c.id}" class:closed={c.status === 'done'}>{c.title}</a>
					</li>
				{/each}
			</ul>
		</section>
	{/if}

	<Connections id={task.id} type="task" />
{/if}

<TaskEditor bind:id={editing} onchange={load} />

<style>
	.crumbs {
		margin: 0 0 var(--space-2);
	}
	.crumbs a {
		color: var(--muted);
		text-decoration: none;
	}
	.head {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		flex-wrap: wrap;
	}
	h1 {
		overflow-wrap: anywhere;
		margin: 0;
	}
	.closed {
		text-decoration: line-through;
		color: var(--muted);
	}
	.chip {
		font-size: var(--text-sm);
		padding: 0 var(--space-2);
		border-radius: 999px;
		background: var(--accent-soft);
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		margin: var(--space-4) 0;
	}
	.facts {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: var(--space-2) var(--space-4);
		align-items: center;
		margin: 0 0 var(--space-6);
	}
	dt {
		color: var(--muted);
	}
	dd {
		margin: 0;
	}
	select,
	input[type='date'] {
		font: inherit;
		padding: var(--space-1) var(--space-2);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		color: var(--fg);
	}
	.notes {
		white-space: pre-wrap;
	}
	.small {
		font-size: var(--text-sm);
	}
	.children {
		list-style: none;
		padding: 0;
		display: grid;
		gap: var(--space-1);
	}
	.children li {
		display: flex;
		gap: var(--space-2);
		align-items: center;
	}
</style>
