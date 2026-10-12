<script lang="ts">
	import { page } from '$app/state';
	import { api, message, type Schemas } from '$lib/api';
	import { href } from '$lib/links';
	import Status from '$lib/ui/Status.svelte';

	type Series = Schemas['InsightSeries'];
	type Review = Schemas['WeeklyReview'];

	const weeks = $derived(Number(page.url.searchParams.get('weeks') ?? '12'));
	const week = $derived(page.url.searchParams.get('week') ?? '');
	let metrics = $state<Series[] | null>(null);
	let review = $state<Review | null>(null);
	let error = $state('');

	async function load() {
		error = '';
		try {
			const [m, r] = await Promise.all([
				api<{ metrics: Series[] }>(`/v1/insights?weeks=${weeks}`),
				api<Review>(`/v1/insights/review${week ? `?week=${week}` : ''}`)
			]);
			metrics = m.metrics;
			review = r;
		} catch (err) {
			error = message(err);
		}
	}
	$effect(() => {
		void weeks;
		void week;
		load();
	});

	const unit = (u: string) => (u === 'percent' ? ' %' : u === 'minutes' ? ' min' : '');
	const arrow: Record<string, string> = { up: '↗', down: '↘', flat: '→', new: '✦' };
	const link = (i: Schemas['InsightItem']) => href(i);
	const query = (p: Record<string, string>) => {
		const u = new URL(page.url);
		for (const [k, v] of Object.entries(p)) u.searchParams.set(k, v);
		return u.search;
	};
	const spark = (s: Series) => {
		const max = Math.max(1, ...s.points.map((p) => p.value));
		return s.points.map((p, i) => ({ x: i * (100 / s.points.length), h: (p.value / max) * 28, p }));
	};
</script>

<svelte:head><title>Insights · Axon</title></svelte:head>

<h1>Insights</h1>
<p class="muted">
	Weekly numbers from what you actually did, computed on this server. Each one says how it is
	counted and opens the items behind it.
</p>

<nav class="tabs" aria-label="Period">
	{#each [8, 12, 26, 52] as n (n)}
		<a href={query({ weeks: String(n) })} aria-current={weeks === n ? 'page' : undefined}
			>{n} weeks</a
		>
	{/each}
	<a class="csv" href="/api/v1/insights/export.csv?weeks=52" download>Download CSV (52 weeks)</a>
</nav>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !metrics || !review}
	<Status kind="loading" message="Counting…" />
{:else}
	<ul class="metrics">
		{#each metrics as s (s.id)}
			<li class="card">
				<h2>{s.title}</h2>
				<p class="total"><strong>{s.total}</strong>{unit(s.unit)}</p>
				<svg viewBox="0 0 100 30" preserveAspectRatio="none" aria-hidden="true">
					{#each spark(s) as b (b.p.week)}
						<rect
							x={b.x + 0.5}
							y={30 - b.h}
							width={100 / s.points.length - 1}
							height={Math.max(b.h, 0.5)}
						/>
					{/each}
				</svg>
				<p class="small trend {s.trend.direction}">
					<span aria-hidden="true">{arrow[s.trend.direction]}</span>
					{s.trend.sentence}
				</p>
				<details class="small">
					<summary>How it is counted</summary>
					<p>{s.formula}</p>
					<p class="muted">Source: <code>{s.source}</code></p>
					<p class="visually-hidden">
						Week by week: {s.points.map((p) => `${p.week} ${p.value}`).join(', ')}
					</p>
				</details>
			</li>
		{/each}
	</ul>

	<section class="card" aria-labelledby="review-title">
		<div class="head">
			<h2 id="review-title">Weekly review · {review.week}</h2>
			<p class="small">
				{review.from} to {review.to} ·
				<a href={query({ week: review.previous })}>← {review.previous}</a>
				{#if review.next}· <a href={query({ week: review.next })}>{review.next} →</a>{/if}
			</p>
		</div>
		<table>
			<caption class="visually-hidden">This week against the week before</caption>
			<thead
				><tr
					><th scope="col">Metric</th><th scope="col">This week</th><th scope="col">Week before</th
					></tr
				></thead
			>
			<tbody>
				{#each review.metrics as m (m.id)}
					<tr
						><th scope="row" title={m.formula}>{m.title}</th><td>{m.value}{unit(m.unit)}</td><td
							>{m.previous}{unit(m.unit)}</td
						></tr
					>
				{/each}
			</tbody>
		</table>
		{#if review.highlights.length}
			<h3>What changed</h3>
			<ul>
				{#each review.highlights as h (h)}<li>{h}</li>{/each}
			</ul>
		{/if}
		{#each review.sections as sec (sec.metric)}
			<details>
				<summary>{sec.title} ({sec.items.length + sec.more})</summary>
				<ul class="small">
					{#each sec.items as i, n (i.id + n)}
						<li>
							{#if link(i)}<a href={link(i)}>{i.title}</a>{:else}{i.title}{/if}
							<span class="muted">· {new Date(i.when).toLocaleDateString()}</span>
						</li>
					{/each}
					{#if sec.more}<li class="muted">and {sec.more} more</li>{/if}
				</ul>
			</details>
		{/each}
		<h3>To carry over</h3>
		{#if review.carry_over.length}
			<ul>
				{#each review.carry_over as i (i.id)}
					<li>
						<a href="/tasks/{i.id}">{i.title}</a>
						<span class="muted small">· due {new Date(i.when).toLocaleDateString()}</span>
					</li>
				{/each}
			</ul>
		{:else}
			<p class="muted">No open task was due by the end of this week.</p>
		{/if}
	</section>
{/if}

<style>
	.small {
		font-size: var(--text-sm);
	}
	.tabs {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
		align-items: center;
	}
	.tabs a {
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
	.tabs .csv {
		margin-left: auto;
		text-decoration: underline;
	}
	.metrics {
		list-style: none;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 15rem), 1fr));
		gap: var(--space-3);
		margin-bottom: var(--space-6);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
	}
	.card h2 {
		font-size: var(--text-base);
		margin: 0;
	}
	.total {
		font-size: var(--text-2xl);
		margin: var(--space-1) 0;
	}
	svg {
		width: 100%;
		height: 2rem;
	}
	rect {
		fill: var(--accent);
		opacity: 0.8;
	}
	.trend.up {
		color: var(--ok);
	}
	.head {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: baseline;
		gap: var(--space-2);
	}
	table {
		width: 100%;
		border-collapse: collapse;
		margin: var(--space-3) 0;
	}
	th,
	td {
		text-align: left;
		padding: var(--space-1) var(--space-2);
		border-bottom: 1px solid var(--border);
	}
</style>
