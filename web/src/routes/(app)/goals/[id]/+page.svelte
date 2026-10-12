<script lang="ts">
	import { deleted } from '$lib/undo';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, apiResponse, message, type Schemas } from '$lib/api';
	import type { Goal, Milestone } from '$lib/goals';
	import Connections from '$lib/Connections.svelte';
	import { addDays, localDate } from '$lib/dates';
	import { between, milestonePlans } from '$lib/entityTemplates';
	import ProgressBar from '$lib/ProgressBar.svelte';
	import TemplateChips from '$lib/TemplateChips.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Explanation = Schemas['Progress'] & {
		items: { id: string; type: string; title: string; done: boolean; via: string | null }[];
	};

	const id = $derived(page.params.id ?? '');
	let goal = $state<Goal | null>(null);
	let etag = '';
	let error = $state('');
	let busy = $state(false);
	let explanation = $state<Explanation | null>(null);
	let confirmDelete = $state(false);
	let form = $state({
		title: '',
		outcome: '',
		status: 'active' as Goal['status'],
		target_date: '',
		measured: false,
		unit: '',
		start: 0,
		target: 0,
		current: 0
	});
	let milestone = $state({ title: '', due_on: '' });

	async function load() {
		error = '';
		try {
			const res = await apiResponse<Goal>(`/v1/goals/${id}`);
			goal = res.data;
			etag = res.etag;
			const m = goal.measure;
			form = {
				title: goal.title,
				outcome: goal.outcome,
				status: goal.status,
				target_date: goal.target_date ?? '',
				measured: !!m,
				unit: m?.unit ?? '',
				start: m?.start ?? 0,
				target: m?.target ?? 0,
				current: m?.current ?? 0
			};
			await refresh();
		} catch (err) {
			error = message(err);
		}
	}

	// Progress changes when milestones or connections do; the goal's own
	// fields (and its ETag) do not.
	async function refresh() {
		explanation = await api<Explanation>(`/v1/goals/${id}/progress`);
		if (goal) {
			const fresh = await api<Goal>(`/v1/goals/${id}`);
			goal.milestones = fresh.milestones;
			goal.progress = fresh.progress;
		}
	}

	$effect(() => {
		void id;
		goal = null;
		load();
	});

	async function save(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			const res = await apiResponse<Goal>(`/v1/goals/${id}`, {
				method: 'PATCH',
				headers: { 'If-Match': etag },
				body: {
					title: form.title,
					outcome: form.outcome,
					status: form.status,
					target_date: form.target_date || null,
					measure: form.measured
						? {
								unit: form.unit,
								start: Number(form.start),
								target: Number(form.target),
								current: Number(form.current)
							}
						: null
				}
			});
			goal = res.data;
			etag = res.etag;
			toast('Saved.');
			await refresh();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function addMilestone(e: SubmitEvent) {
		e.preventDefault();
		const { title, due_on } = milestone;
		milestone = { title: '', due_on: '' }; // ready for the next one straight away
		try {
			await api(`/v1/goals/${id}/milestones`, {
				method: 'POST',
				body: { title, ...(due_on ? { due_on } : {}) }
			});
			await refresh();
		} catch (err) {
			milestone = { title, due_on };
			toast(message(err), 'error');
		}
	}

	// Milestones from a plan, spread from today to the target date (90 days without one).
	async function addPlan(steps: [string, number][]) {
		if (!goal) return;
		const today = localDate(new Date());
		const target =
			goal.target_date && goal.target_date > today ? goal.target_date : addDays(today, 90);
		busy = true;
		try {
			for (const [title, share] of steps) {
				await api(`/v1/goals/${id}/milestones`, {
					method: 'POST',
					body: { title, due_on: between(today, target, share) }
				});
			}
			await refresh();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function toggle(m: Milestone) {
		try {
			const { etag: version } = await apiResponse(`/v1/milestones/${m.id}`);
			await api(`/v1/milestones/${m.id}`, {
				method: 'PATCH',
				headers: { 'If-Match': version },
				body: { done: !m.done_at }
			});
			await refresh();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function removeMilestone(m: Milestone) {
		try {
			await api(`/v1/milestones/${m.id}`, { method: 'DELETE' });
			deleted(`Removed “${m.title}”.`, m.id, () => load());
			await refresh();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function remove() {
		busy = true;
		try {
			await api(`/v1/goals/${id}`, { method: 'DELETE' });
			const gone = id;
			deleted('Goal deleted.', gone, () => goto(`/goals/${gone}`));
			await goto('/goals');
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
</script>

<svelte:head><title>{goal?.title ?? 'Goal'} · Axon</title></svelte:head>

<p class="crumbs"><a href="/goals">← Goals</a></p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !goal}
	<Status kind="loading" message="Loading goal…" />
{:else}
	<h1>{goal.title}</h1>
	{#if goal.status !== 'active'}<p class="muted">This goal is {goal.status}.</p>{/if}

	<section class="card" aria-labelledby="progress-title">
		<h2 id="progress-title">Progress</h2>
		<ProgressBar progress={goal.progress} label="Progress" />
		{#if explanation && explanation.method !== 'none'}
			<details>
				<summary>How is this worked out?</summary>
				{#if explanation.method === 'measure'}
					<p>
						From your measure: how far the current value has moved from the start towards the target
						({explanation.summary}), capped between 0 and 100%.
					</p>
				{:else}
					<p>
						{explanation.done} of {explanation.total} items are done, rounded down. Items are this goal's
						milestones and the tasks linked to it or to one of its milestones. Cancelled tasks and linked
						notes do not count.
					</p>
					<ul class="items">
						{#each explanation.items as it (it.id)}
							<li>
								<span aria-hidden="true">{it.done ? '☑' : '☐'}</span>
								<span class="visually-hidden">{it.done ? 'Done:' : 'Not done:'}</span>
								{it.title}
								<span class="muted small"
									>{it.type}{it.via ? ` · for milestone “${it.via}”` : ''}</span
								>
							</li>
						{/each}
					</ul>
				{/if}
			</details>
		{/if}
	</section>

	<section class="card" aria-labelledby="milestones-title">
		<h2 id="milestones-title">Milestones</h2>
		{#if goal.milestones.length === 0}
			<p class="muted">
				Break the goal into a few checkpoints you can recognise when you reach them.
			</p>
		{:else}
			<ul class="rows">
				{#each goal.milestones as m (m.id)}
					<li>
						<input
							type="checkbox"
							checked={!!m.done_at}
							onchange={() => toggle(m)}
							aria-label="Done: {m.title}"
						/>
						<span class="grow" class:done={!!m.done_at}>{m.title}</span>
						{#if m.due_on}<span class="muted small">by {m.due_on}</span>{/if}
						<Button variant="ghost" onclick={() => removeMilestone(m)} aria-label="Remove {m.title}"
							>Remove</Button
						>
					</li>
				{/each}
			</ul>
		{/if}
		<form class="inline" onsubmit={addMilestone}>
			<label for="milestone-title" class="visually-hidden">New milestone</label>
			<input
				id="milestone-title"
				placeholder="New milestone…"
				bind:value={milestone.title}
				maxlength="500"
				required
				autocomplete="off"
			/>
			<label for="milestone-due" class="visually-hidden">Milestone date (optional)</label>
			<input id="milestone-due" type="date" bind:value={milestone.due_on} />
			<Button type="submit" variant="secondary">Add</Button>
		</form>
		<TemplateChips
			label="Milestone templates"
			items={milestonePlans.map((p) => ({
				title: p.title,
				hint: p.steps.map((s) => s[0]).join(' → ')
			}))}
			onpick={(i) => addPlan(milestonePlans[i].steps)}
			{busy}
		/>
	</section>

	<Connections id={goal.id} type="goal" onchange={refresh} />

	<section class="card" aria-labelledby="details-title">
		<h2 id="details-title">Details</h2>
		<form onsubmit={save}>
			<TextField id="goal-title" label="Goal" bind:value={form.title} maxlength={500} required />
			<div class="field">
				<label for="goal-outcome">Outcome</label>
				<p class="muted small" id="goal-outcome-hint">What will be true when you get there?</p>
				<textarea
					id="goal-outcome"
					rows="3"
					maxlength="20000"
					bind:value={form.outcome}
					aria-describedby="goal-outcome-hint"
				></textarea>
			</div>
			<div class="grid">
				<TextField
					id="goal-target-date"
					label="Target date (optional)"
					type="date"
					bind:value={form.target_date}
				/>
				<div class="field">
					<label for="goal-status">Status</label>
					<select id="goal-status" bind:value={form.status}>
						<option value="active">Active</option>
						<option value="achieved">Achieved</option>
						<option value="archived">Archived (keeps history and links)</option>
					</select>
				</div>
			</div>
			<label class="check">
				<input type="checkbox" bind:checked={form.measured} />
				Track a number (for example km, books, or kg)
			</label>
			{#if form.measured}
				<div class="grid">
					<TextField id="m-unit" label="Unit" bind:value={form.unit} maxlength={40} required />
					<TextField id="m-start" label="Start" type="number" step="any" bind:value={form.start} />
					<TextField
						id="m-target"
						label="Target"
						type="number"
						step="any"
						bind:value={form.target}
					/>
					<TextField
						id="m-current"
						label="Current"
						type="number"
						step="any"
						bind:value={form.current}
					/>
				</div>
			{/if}
			<div class="row">
				<Button type="submit" {busy}>Save</Button>
				<span class="grow"></span>
				<Button variant="danger" onclick={() => (confirmDelete = true)}>Delete goal</Button>
			</div>
		</form>
	</section>
{/if}

<Dialog bind:open={confirmDelete} title="Delete this goal?">
	<p>
		“{goal?.title}” and its milestones are removed. Linked tasks and notes stay. To keep the
		history, archive the goal instead.
	</p>
	<div class="row end">
		<Button variant="secondary" onclick={() => (confirmDelete = false)}>Cancel</Button>
		<Button variant="danger" onclick={remove} {busy}>Delete</Button>
	</div>
</Dialog>

<style>
	.crumbs {
		margin: 0 0 var(--space-2);
	}
	.crumbs a {
		color: var(--muted);
		text-decoration: none;
	}
	h1 {
		overflow-wrap: anywhere;
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
	.small {
		font-size: var(--text-sm);
	}
	details {
		margin-top: var(--space-3);
	}
	summary {
		cursor: pointer;
		color: var(--accent);
	}
	.items,
	.rows {
		list-style: none;
		padding: 0;
		margin: var(--space-2) 0 var(--space-4);
	}
	.items li {
		padding: var(--space-1) 0;
	}
	.rows li {
		display: flex;
		align-items: center;
		gap: var(--space-3);
		padding: var(--space-2) 0;
		border-bottom: 1px solid var(--border);
		flex-wrap: wrap;
	}
	.rows input[type='checkbox'],
	.check input {
		width: 1.25rem;
		height: 1.25rem;
		accent-color: var(--accent);
	}
	.grow {
		flex: 1;
		min-width: 8rem;
		overflow-wrap: anywhere;
	}
	.done {
		color: var(--muted);
		text-decoration: line-through;
	}
	.inline {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}
	.inline input,
	textarea,
	.field select {
		font: inherit;
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	.inline input:first-of-type {
		flex: 1;
		min-width: 10rem;
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
	}
	.field label {
		font-weight: 600;
	}
	.field p {
		margin: 0;
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
		column-gap: var(--space-4);
	}
	.check {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}
	.row {
		display: flex;
		gap: var(--space-3);
		flex-wrap: wrap;
		align-items: center;
	}
	.end {
		justify-content: flex-end;
	}
</style>
