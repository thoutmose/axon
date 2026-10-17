<script lang="ts">
	import { api } from '$lib/api';
	import type { paths } from '$lib/api-types';
	import { formatDate } from '$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import { relativeDay } from '$lib/tasks';
	import Button from '$lib/ui/Button.svelte';

	// Your roadmap today (EP-145): where the plan is, what today's block
	// asks, and what comes next, so a self-learner always knows the next step.
	let { today }: { today: string } = $props();

	type Today =
		paths['/v1/roadmaps/today']['get']['responses'][200]['content']['application/json']['plan'];
	let plan = $state<Today | undefined>(undefined);
	api<{ plan: Today }>('/v1/roadmaps/today')
		.then((r) => (plan = r.plan))
		.catch(() => (plan = null));

	const long = (d: string) => formatDate(d, { weekday: 'long', day: 'numeric', month: 'long' });
	const phase = $derived(plan?.phase ?? null);
	const before = $derived(!!phase && phase.state === 'future');
	const daysTo = $derived(
		phase ? Math.round((Date.parse(phase.start) - Date.parse(today)) / 86_400_000) : 0
	);
</script>

{#if plan === null}
	<section class="card roadmap" aria-labelledby="roadmap-today-title">
		<h2 id="roadmap-today-title">{t('Your roadmap')}</h2>
		<p>
			{t(
				'Learning on your own goes further with a plan: take the skills assessment, then start the Top-5% roadmap. Each day then says what to study.'
			)}
		</p>
		<div class="row">
			<Button href="/roadmaps/assessment" variant="secondary">{t('Take the assessment')}</Button>
			<Button href="/roadmaps">{t('See the roadmap')}</Button>
		</div>
	</section>
{:else if plan && phase}
	<section class="card roadmap" aria-labelledby="roadmap-today-title">
		<div class="head">
			<h2 id="roadmap-today-title">{t('Your roadmap today')}</h2>
			<a class="small" href="/roadmaps/{plan.roadmap_id}?view=plan">{plan.title}</a>
		</div>
		<p class="lead">
			{t('Phase {n} of {total}: {title}', {
				n: phase.n,
				total: plan.phases,
				title: phase.title.replace(/^Phase \d+: /, '')
			})}
		</p>
		<p class="muted small">
			{#if before}
				{t(daysTo === 1 ? 'Starts on {date}, tomorrow.' : 'Starts on {date}, in {n} days.', {
					date: long(phase.start),
					n: daysTo
				})}
			{:else}
				{t(
					'Week {week} of {weeks} · {days} days to its milestone · {reached} of {total} milestones reached',
					{
						week: plan.week,
						weeks: plan.weeks,
						days: plan.days_left,
						reached: plan.reached,
						total: plan.phases
					}
				)}
			{/if}
		</p>
		<div
			class="bar"
			role="progressbar"
			aria-label={t('Weeks of this phase')}
			aria-valuemin="0"
			aria-valuemax={plan.weeks}
			aria-valuenow={plan.week}
		>
			<span style:width="{(plan.week / Math.max(plan.weeks, 1)) * 100}%"></span>
		</div>

		<dl class="facts">
			<div>
				<dt>{t('Today’s block')}</dt>
				<dd>
					{#if plan.block}{plan.block}{:else if before}{t(
							'Nothing yet: get ready with the first week’s tasks.'
						)}{:else}{t('No block today.')}{/if}
				</dd>
			</div>
			{#if phase.project}
				<div>
					<dt>{t('Project of the phase')}</dt>
					<dd>
						{#if phase.project.id}<a href="/projects/{phase.project.id}">{phase.project.title}</a
							>{#if phase.project.total}
								{' '}<span class="muted small">
									· {t('{done} of {total} cards done', {
										done: phase.project.done,
										total: phase.project.total
									})}</span
								>{/if}{:else}{phase.project.title}{/if}
					</dd>
				</div>
			{/if}
			{#if plan.next_certification}
				<div>
					<dt>{t('Next certification')}</dt>
					<dd>
						<a href="/learn/certifications">{plan.next_certification.code}</a>
						<span class="small">{plan.next_certification.name}</span>
						{#if plan.next_certification.target_on}<span class="muted small">
								· {relativeDay(plan.next_certification.target_on, today)}</span
							>{/if}
					</dd>
				</div>
			{/if}
		</dl>
		<div class="row">
			<Button href="/focus?target={plan.roadmap_id}">{t('Focus on today’s block')}</Button>
			<Button href="/learn" variant="secondary">{t('Review cards')}</Button>
			<Button href="/learn/practice" variant="secondary">{t('Practice')}</Button>
			<Button href="/roadmaps/{plan.roadmap_id}?view=plan" variant="ghost"
				>{t('Open the plan')}</Button
			>
		</div>
	</section>
{/if}

<style>
	.roadmap {
		background: var(--surface);
		border: 1px solid var(--border);
		border-left: 4px solid var(--accent);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6) var(--space-6);
		margin-bottom: var(--space-6);
	}
	@media (max-width: 30rem) {
		.roadmap {
			padding: var(--space-4);
		}
	}
	.head {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		gap: var(--space-2);
		align-items: baseline;
	}
	h2 {
		margin: 0;
	}
	.lead {
		font-size: var(--text-lg);
		font-weight: 650;
		margin: var(--space-2) 0 var(--space-1);
	}
	.bar {
		height: 0.4rem;
		border-radius: 999px;
		background: var(--accent-soft);
		overflow: hidden;
		margin: var(--space-2) 0 var(--space-3);
	}
	.bar span {
		display: block;
		height: 100%;
		background: var(--accent);
	}
	.facts {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
		gap: var(--space-3);
		margin: 0 0 var(--space-3);
	}
	dt {
		font-size: var(--text-sm);
		color: var(--muted);
	}
	dd {
		margin: 0;
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
