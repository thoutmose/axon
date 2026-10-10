<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, message } from '$lib/api';
	import { daysUntil, mondayOf, type Goal } from '$lib/goals';
	import ProgressBar from '$lib/ProgressBar.svelte';
	import { relativeDay } from '$lib/tasks';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	const tabs = [
		{ id: 'active', label: 'Active' },
		{ id: 'achieved', label: 'Achieved' },
		{ id: 'archived', label: 'Archived' }
	];
	const status = $derived(page.url.searchParams.get('status') ?? 'active');
	let goals = $state<Goal[] | null>(null);
	let error = $state('');
	let title = $state('');
	let today = $state('');
	let reviewed = $state(true);
	let reflection = $state('');

	async function load() {
		error = '';
		try {
			goals = (await api<{ items: Goal[] }>(`/v1/goals?status=${status}`)).items;
			const c = await api<{ items: { kind: string; day: string }[]; today: string }>(
				'/v1/checkins'
			);
			today = c.today;
			reviewed = c.items.some((x) => x.kind === 'weekly' && x.day === mondayOf(today));
		} catch (err) {
			error = message(err);
		}
	}

	$effect(() => {
		void status;
		goals = null;
		load();
	});

	async function create(e: SubmitEvent) {
		e.preventDefault();
		try {
			const g = await api<Goal>('/v1/goals', { method: 'POST', body: { title } });
			title = '';
			await goto(`/goals/${g.id}`);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function review(e: SubmitEvent) {
		e.preventDefault();
		try {
			await api(`/v1/checkins/weekly/${mondayOf(today)}`, {
				method: 'PUT',
				body: { note: reflection }
			});
			reviewed = true;
			toast('Weekly review saved.');
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function due(g: Goal): string {
		if (!g.target_date) return '';
		const days = daysUntil(g.target_date, today);
		return days < 0
			? `Target date passed ${relativeDay(g.target_date, today)}`
			: `Target ${relativeDay(g.target_date, today)}`;
	}
</script>

<svelte:head><title>Goals · Axon</title></svelte:head>

<h1>Goals</h1>
<p class="muted hint">
	Name an outcome, then connect the milestones, tasks, and notes that move it. Every figure shows
	how it was worked out.
</p>

<nav aria-label="Goal status" class="tabs">
	{#each tabs as t (t.id)}
		<a href="?status={t.id}" aria-current={status === t.id ? 'page' : undefined}>{t.label}</a>
	{/each}
</nav>

{#if status === 'active'}
	<form class="quick" onsubmit={create}>
		<label for="new-goal" class="visually-hidden">New goal</label>
		<input
			id="new-goal"
			placeholder="New goal, e.g. Run a 10 km race…"
			bind:value={title}
			maxlength="500"
			required
			autocomplete="off"
		/>
		<Button type="submit" variant="secondary">Create</Button>
	</form>
{/if}

{#if status === 'active' && goals?.length && today && !reviewed}
	<section class="review" aria-labelledby="review-title">
		<h2 id="review-title">Weekly goal review</h2>
		<p class="muted">
			A few minutes to look at each goal: what moved this week, and what is the next step? Nothing
			changes unless you change it.
		</p>
		<form onsubmit={review}>
			<label for="reflection">What moved forward, and what's next?</label>
			<textarea id="reflection" rows="3" maxlength="2000" bind:value={reflection}></textarea>
			<div class="row">
				<Button type="submit" variant="secondary">Save review</Button>
				<Button variant="ghost" onclick={() => (reviewed = true)}>Not this week</Button>
			</div>
		</form>
	</section>
{/if}

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !goals}
	<Status kind="loading" message="Loading goals…" />
{:else if goals.length === 0}
	<Status
		kind="empty"
		message={status === 'active'
			? 'No active goals. Name one outcome you want this year, above.'
			: `No ${status} goals.`}
	/>
{:else}
	<ul class="goals">
		{#each goals as g (g.id)}
			<li>
				<div class="title-row">
					<a href="/goals/{g.id}">{g.title}</a>
					{#if today && g.target_date}<span
							class="muted small"
							class:late={daysUntil(g.target_date, today) < 0 && g.status === 'active'}
							>{due(g)}</span
						>{/if}
				</div>
				<ProgressBar progress={g.progress} label="Progress of {g.title}" />
			</li>
		{/each}
	</ul>
{/if}

<style>
	.hint {
		font-size: var(--text-sm);
	}
	.small {
		font-size: var(--text-sm);
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
	.quick {
		display: flex;
		gap: var(--space-2);
		margin-bottom: var(--space-6);
	}
	.quick input,
	textarea {
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
	.review {
		background: var(--surface);
		border: 1px solid var(--accent);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-6);
	}
	.review h2 {
		font-size: var(--text-lg);
		margin-top: 0;
	}
	.review form {
		display: grid;
		gap: var(--space-2);
	}
	.review label {
		font-weight: 600;
	}
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}
	.goals {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		gap: var(--space-3);
	}
	.goals li {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
	}
	.title-row {
		display: flex;
		justify-content: space-between;
		align-items: baseline;
		gap: var(--space-3);
		flex-wrap: wrap;
		margin-bottom: var(--space-2);
	}
	.title-row a {
		color: var(--fg);
		font-weight: 600;
		text-decoration: none;
		overflow-wrap: anywhere;
	}
	.title-row a:hover {
		color: var(--accent);
	}
	.late {
		color: var(--warn);
	}
</style>
