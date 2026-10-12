<script lang="ts">
	import { page } from '$app/state';
	import { api, message } from '$lib/api';
	import Connections from '$lib/Connections.svelte';
	import TaskEditor from '$lib/TaskEditor.svelte';
	import type { Task } from '$lib/tasks';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';

	const id = $derived(page.params.id ?? '');
	let task = $state<Task | null>(null);
	let error = $state('');
	let editing = $state<string | null>(null);

	const statusNames: Record<string, string> = {
		inbox: 'Inbox',
		next: 'Next action',
		doing: 'In progress',
		waiting: 'Waiting for',
		someday: 'Someday / maybe',
		done: 'Done',
		cancelled: 'Cancelled'
	};

	async function load() {
		error = '';
		try {
			task = await api<Task>(`/v1/tasks/${id}`);
		} catch (err) {
			error = message(err);
		}
	}

	$effect(() => {
		void id;
		task = null;
		load();
	});
</script>

<svelte:head><title>{task?.title ?? 'Task'} · Axon</title></svelte:head>

<p class="crumbs"><a href="/tasks">← Tasks</a></p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !task}
	<Status kind="loading" message="Loading task…" />
{:else}
	<div class="head">
		<h1>{task.title}</h1>
		<Button variant="secondary" onclick={() => (editing = task?.id ?? null)}>Edit</Button>
	</div>
	<dl class="facts">
		<dt>Status</dt>
		<dd>{statusNames[task.status]}</dd>
		{#if task.context}<dt>Context</dt>
			<dd>{task.context}</dd>{/if}
		{#if task.scheduled_on}<dt>Do on</dt>
			<dd>{task.scheduled_on}</dd>{/if}
		{#if task.due_on}<dt>Due</dt>
			<dd>{task.due_on}</dd>{/if}
		{#if task.recurrence}<dt>Repeats</dt>
			<dd>{task.recurrence}</dd>{/if}
	</dl>
	{#if task.notes}<p class="notes">{task.notes}</p>{/if}

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
		justify-content: space-between;
		align-items: center;
		gap: var(--space-4);
		flex-wrap: wrap;
	}
	h1 {
		overflow-wrap: anywhere;
	}
	.facts {
		display: grid;
		grid-template-columns: max-content 1fr;
		gap: var(--space-1) var(--space-4);
		margin: 0 0 var(--space-6);
	}
	dt {
		color: var(--muted);
	}
	dd {
		margin: 0;
	}
	.notes {
		white-space: pre-wrap;
	}
</style>
