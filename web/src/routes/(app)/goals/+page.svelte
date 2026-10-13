<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import AgentDock from '$lib/AgentDock.svelte';
	import { pageAgents } from '$lib/pageAgents';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, message } from '$lib/api';
	import { addDays, daysBetween, localDate, mondayOf } from '$lib/dates';
	import type { Goal } from '$lib/goals';
	import { between, goalTemplates, type GoalTemplate } from '$lib/entityTemplates';
	import ProgressBar from '$lib/ProgressBar.svelte';
	import TemplateChips from '$lib/TemplateChips.svelte';
	import { relativeDay } from '$lib/tasks';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	const tabs = $derived([
		{ id: 'active', label: t('Active') },
		{ id: 'achieved', label: t('Achieved') },
		{ id: 'archived', label: t('Archived') }
	]);
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

	let busy = $state(false);

	// A goal from a template: outcome, target date, measure, and milestones,
	// all editable on the goal page.
	async function fromTemplate(t: GoalTemplate) {
		busy = true;
		try {
			const start = today || localDate(new Date());
			const target = addDays(start, t.days);
			const g = await api<Goal>('/v1/goals', {
				method: 'POST',
				body: {
					title: t.title,
					outcome: t.outcome,
					target_date: target,
					measure: t.measure ?? null
				}
			});
			for (const [title, share] of t.milestones) {
				await api(`/v1/goals/${g.id}/milestones`, {
					method: 'POST',
					body: { title, due_on: between(start, target, share) }
				});
			}
			await goto(`/goals/${g.id}`);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
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
			toast(t('Weekly review saved.'));
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function due(g: Goal): string {
		if (!g.target_date) return '';
		const days = daysBetween(today, g.target_date);
		return days < 0
			? `Target date passed ${relativeDay(g.target_date, today)}`
			: `Target ${relativeDay(g.target_date, today)}`;
	}
</script>

<svelte:head><title>{t('Goals · Axon')}</title></svelte:head>

<h1>{t('Goals')}</h1>
<AgentDock {...pageAgents.goals} onapplied={() => load()} />
<p class="muted hint">
	{t(
		'Name an outcome, then connect the milestones, tasks, and notes that move it. Every figure shows how it was worked out.'
	)}
</p>

<nav aria-label={t('Goal status')} class="tabs">
	{#each tabs as tab (tab.id)}
		<a href="?status={tab.id}" aria-current={status === tab.id ? 'page' : undefined}>{tab.label}</a>
	{/each}
</nav>

{#if status === 'active'}
	<form class="quick" onsubmit={create}>
		<label for="new-goal" class="visually-hidden">{t('New goal')}</label>
		<input
			id="new-goal"
			placeholder={t('New goal, e.g. Run a 10 km race…')}
			bind:value={title}
			maxlength="500"
			required
			autocomplete="off"
		/>
		<Button type="submit" variant="secondary">{t('Create')}</Button>
	</form>
	<TemplateChips
		items={goalTemplates.map((tpl) => ({ title: t(tpl.title), hint: t(tpl.outcome) }))}
		onpick={(i) => fromTemplate(goalTemplates[i])}
		{busy}
	/>
{/if}

{#if status === 'active' && goals?.length && today && !reviewed}
	<section class="review" aria-labelledby="review-title">
		<h2 id="review-title">{t('Weekly goal review')}</h2>
		<p class="muted">
			{t(
				'A few minutes to look at each goal: what moved this week, and what is the next step? Nothing changes unless you change it.'
			)}
		</p>
		<form onsubmit={review}>
			<label for="reflection">{t("What moved forward, and what's next?")}</label>
			<textarea id="reflection" rows="3" maxlength="2000" bind:value={reflection}></textarea>
			<div class="row">
				<Button type="submit" variant="secondary">{t('Save review')}</Button>
				<Button variant="ghost" onclick={() => (reviewed = true)}>{t('Not this week')}</Button>
			</div>
		</form>
	</section>
{/if}

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !goals}
	<Status kind="loading" message={t('Loading goals…')} />
{:else if goals.length === 0}
	<Status
		kind="empty"
		message={status === 'active'
			? t('No active goals. Name one outcome you want this year, above.')
			: status === 'archived'
				? t('No archived goals.')
				: t('No achieved goals.')}
	/>
{:else}
	<ul class="goals">
		{#each goals as g (g.id)}
			<li>
				<div class="title-row">
					<a href="/goals/{g.id}">{g.title}</a>
					{#if today && g.target_date}<span
							class="muted small"
							class:late={daysBetween(today, g.target_date) < 0 && g.status === 'active'}
							>{due(g)}</span
						>{/if}
				</div>
				<ProgressBar progress={g.progress} label={t('Progress of {title}', { title: g.title })} />
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
