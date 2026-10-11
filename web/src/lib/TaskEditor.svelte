<script lang="ts">
	import { api, apiResponse, message } from './api';
	import { recurrencePresets, type Task } from './tasks';
	import Button from './ui/Button.svelte';
	import Dialog from './ui/Dialog.svelte';
	import TextField from './ui/TextField.svelte';
	import { toast } from './ui/toast.svelte';

	let { id = $bindable(null), onchange }: { id: string | null; onchange: () => void } = $props();

	let open = $state(false);
	let etag = '';
	let busy = $state(false);
	let form = $state({
		title: '',
		notes: '',
		status: 'inbox' as Task['status'],
		context: '',
		scheduled_on: '',
		due_on: '',
		repeat: '' // a preset rule, "custom", or "" for none
	});
	let customRule = $state('');

	$effect(() => {
		if (id) load(id);
	});
	$effect(() => {
		if (!open) id = null;
	});

	async function load(taskID: string) {
		try {
			const res = await apiResponse<Task>(`/v1/tasks/${taskID}`);
			const t = res.data;
			etag = res.etag;
			const preset = recurrencePresets.find((p) => p.rule === t.recurrence);
			form = {
				title: t.title,
				notes: t.notes,
				status: t.status,
				context: t.context ?? '',
				scheduled_on: t.scheduled_on ?? '',
				due_on: t.due_on ?? '',
				repeat: t.recurrence ? (preset ? (preset.rule ?? '') : 'custom') : ''
			};
			customRule = preset ? '' : (t.recurrence ?? '');
			open = true;
		} catch (err) {
			toast(message(err), 'error');
			id = null;
		}
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		const recurrence = form.repeat === 'custom' ? customRule : form.repeat || null;
		try {
			await api(`/v1/tasks/${id}`, {
				method: 'PATCH',
				headers: { 'If-Match': etag },
				body: {
					title: form.title,
					notes: form.notes,
					status: form.status,
					context: form.context || null,
					scheduled_on: form.scheduled_on || null,
					due_on: form.due_on || null,
					recurrence
				}
			});
			toast('Saved.');
			open = false;
			onchange();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function remove() {
		busy = true;
		try {
			await api(`/v1/tasks/${id}`, { method: 'DELETE' });
			toast('Deleted.');
			open = false;
			onchange();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
</script>

<Dialog bind:open title="Edit task">
	<form onsubmit={save}>
		<TextField id="task-title" label="Title" bind:value={form.title} maxlength={500} required />
		<div class="field">
			<label for="task-notes">Notes</label>
			<textarea id="task-notes" rows="3" bind:value={form.notes}></textarea>
		</div>
		<div class="grid">
			<div class="field">
				<label for="task-status">Status</label>
				<select id="task-status" bind:value={form.status}>
					<option value="inbox">Inbox</option>
					<option value="next">Next action</option>
					<option value="doing">In progress</option>
					<option value="waiting">Waiting for</option>
					<option value="someday">Someday / maybe</option>
					<option value="done">Done</option>
					<option value="cancelled">Cancelled</option>
				</select>
			</div>
			<TextField id="task-context" label="Context" placeholder="@home" bind:value={form.context} />
			<TextField id="task-scheduled" label="Do on" type="date" bind:value={form.scheduled_on} />
			<TextField id="task-due" label="Due" type="date" bind:value={form.due_on} />
		</div>
		<div class="field">
			<label for="task-repeat">Repeat</label>
			<select id="task-repeat" bind:value={form.repeat}>
				{#each recurrencePresets as p (p.label)}<option value={p.rule ?? ''}>{p.label}</option
					>{/each}
				<option value="custom">Custom rule…</option>
			</select>
		</div>
		{#if form.repeat === 'custom'}
			<TextField
				id="task-rule"
				label="RRULE"
				hint="For example FREQ=WEEKLY;BYDAY=MO,TH or FREQ=MONTHLY;BYMONTHDAY=-1"
				bind:value={customRule}
				required
			/>
		{/if}
		<div class="actions">
			<Button variant="danger" onclick={remove} {busy}>Delete</Button>
			<a class="open" href="/tasks/{id}">Open page and connections</a>
			<span class="spacer"></span>
			<Button variant="secondary" onclick={() => (open = false)}>Cancel</Button>
			<Button type="submit" {busy}>Save</Button>
		</div>
	</form>
</Dialog>

<style>
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
		column-gap: var(--space-4);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
	}
	label {
		font-weight: 600;
	}
	textarea,
	select {
		font: inherit;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		min-height: 2.75rem;
	}
	.actions {
		display: flex;
		gap: var(--space-3);
		flex-wrap: wrap;
	}
	.spacer {
		flex: 1;
	}
	.open {
		align-self: center;
	}
</style>
