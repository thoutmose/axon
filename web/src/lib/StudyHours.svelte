<script lang="ts">
	import { api } from '$lib/api';
	import { formatDate, localDate } from '$lib/dates';
	import { i18n, t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';

	// Study hours against the plan (EP-105): focus time on the roadmap this
	// week, day by day beside the plan's block, and over twelve weeks.
	let { roadmapId }: { roadmapId: string } = $props();

	type M = { id: string; title: string; minutes: number };
	type Hours = {
		target_hours: number | null;
		weeks: { week: string; minutes: number }[];
		days: { date: string; block: string; minutes: number }[];
		skills: M[];
		projects: M[];
	};
	let h = $state<Hours | null>(null);
	$effect(() => {
		void roadmapId;
		api<Hours>(`/v1/roadmaps/${roadmapId}/hours`)
			.then((r) => (h = r))
			.catch(() => (h = null));
	});

	const today = localDate(new Date());
	const hours = (m: number) => (m / 60).toLocaleString(i18n.locale, { maximumFractionDigits: 1 });
	const week = $derived(h?.weeks[h.weeks.length - 1]?.minutes ?? 0);
	const shown = $derived(!!h && (h.target_hours != null || h.weeks.some((w) => w.minutes > 0)));
	const max = $derived(
		Math.max((h?.target_hours ?? 0) * 60, ...(h?.weeks ?? []).map((w) => w.minutes), 1)
	);
	const block = $derived(h?.days.find((d) => d.date === today)?.block ?? '');
</script>

{#if h && shown}
	<section class="card hours" aria-labelledby="hours-title">
		<h2 id="hours-title">{t('Study hours')}</h2>
		<p class="lead">
			{#if h.target_hours}
				{t('{done} h of {target} h this week', { done: hours(week), target: h.target_hours })}
			{:else}
				{t('{done} h this week', { done: hours(week) })}
			{/if}
		</p>
		{#if h.target_hours}
			<div
				class="bar"
				role="progressbar"
				aria-label={t('This week against the plan')}
				aria-valuemin="0"
				aria-valuemax={h.target_hours * 60}
				aria-valuenow={Math.min(week, h.target_hours * 60)}
			>
				<span style:width="{Math.min(100, (week / (h.target_hours * 60)) * 100)}%"></span>
			</div>
		{/if}
		{#if h.days.some((d) => d.block)}
			<ul class="days">
				{#each h.days as d (d.date)}
					<li class:today={d.date === today}>
						<span class="day">{formatDate(d.date, { weekday: 'short' })}</span>
						<span class="block">{d.block}</span>
						<span class="done">{d.minutes ? `${hours(d.minutes)} h` : '—'}</span>
					</li>
				{/each}
			</ul>
		{/if}
		<div class="row">
			<Button href="/focus?target={roadmapId}">{t('Focus on today’s block')}</Button>
			{#if block}<span class="small muted">{block}</span>{/if}
		</div>
		<figure>
			<figcaption class="small muted">{t('The last 12 weeks')}</figcaption>
			<ol class="weeks">
				{#each h.weeks as w (w.week)}
					<li>
						<span
							class="col"
							style:height="{Math.round((w.minutes / max) * 64)}px"
							aria-hidden="true"
						></span>
						<span class="visually-hidden"
							>{t('Week of {week}: {h} h', { week: w.week, h: hours(w.minutes) })}</span
						>
					</li>
				{/each}
				{#if h.target_hours}<span
						class="target"
						style:bottom="{Math.round(((h.target_hours * 60) / max) * 64)}px"
						aria-hidden="true"
					></span>{/if}
			</ol>
		</figure>
		{#if h.skills.length || h.projects.length}
			<p class="small muted">
				{[...h.skills, ...h.projects]
					.slice(0, 6)
					.map((m) => `${m.title} ${hours(m.minutes)} h`)
					.join(' · ')}
			</p>
		{/if}
		<p class="small muted">
			{t(
				'Focus sessions count when they are for this roadmap, one of its skills, its goal and milestones, or the projects, habits, and tasks of its plan.'
			)}
		</p>
	</section>
{/if}

<style>
	.hours {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.lead {
		font-size: var(--text-xl);
		font-weight: 650;
		margin: 0 0 var(--space-2);
	}
	.bar {
		height: 0.6rem;
		border-radius: 999px;
		background: var(--accent-soft);
		overflow: hidden;
		max-width: 32rem;
	}
	.bar span {
		display: block;
		height: 100%;
		background: var(--ok);
	}
	.days {
		list-style: none;
		padding: 0;
		margin: var(--space-3) 0;
		display: grid;
		gap: 2px;
		font-size: var(--text-sm);
	}
	.days li {
		display: grid;
		grid-template-columns: 3.5rem 1fr 3.5rem;
		gap: var(--space-2);
		padding: 2px var(--space-1);
		border-radius: var(--radius);
	}
	.days li.today {
		background: var(--accent-soft);
	}
	.done {
		text-align: right;
		font-variant-numeric: tabular-nums;
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
		align-items: center;
	}
	.weeks {
		position: relative;
		list-style: none;
		padding: 0;
		margin: var(--space-2) 0;
		display: grid;
		grid-template-columns: repeat(12, 1fr);
		gap: var(--space-1);
		align-items: end;
		height: 4.5rem;
		max-width: 32rem;
	}
	.weeks li {
		display: flex;
		justify-content: center;
		align-items: end;
		height: 100%;
	}
	.col {
		width: 70%;
		min-height: 2px;
		background: var(--accent);
		border-radius: 3px 3px 0 0;
	}
	.target {
		position: absolute;
		left: 0;
		right: 0;
		border-top: 2px dashed var(--muted);
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
