<script lang="ts">
	import { api, message } from '$lib/api';
	import Heatmap from '$lib/charts/Heatmap.svelte';
	import LineChart from '$lib/charts/LineChart.svelte';
	import { trend, type Overview } from '$lib/overview';
	import SkillNetwork from '$lib/SkillNetwork.svelte';
	import Status from '$lib/ui/Status.svelte';

	let days = $state(30);
	let data = $state<Overview | null>(null);
	let error = $state('');

	async function load() {
		error = '';
		try {
			data = await api<Overview>(`/v1/dashboard/overview?days=${days}`);
		} catch (err) {
			error = message(err);
		}
	}
	$effect(() => {
		void days;
		load();
	});
	const fmt = (unit: string, v: number) => (unit === 'percent' ? `${v}%` : `${v}`);
	const when = (d: string) =>
		new Date(d.length === 10 ? d + 'T00:00:00' : d).toLocaleDateString(undefined, {
			weekday: 'short',
			day: 'numeric',
			month: 'short'
		});
</script>

<section class="overview" aria-labelledby="overview-title">
	<div class="head">
		<h2 id="overview-title">Your progress</h2>
		<label>
			<span class="visually-hidden">Period</span>
			<select bind:value={days}>
				<option value={7}>Last 7 days</option>
				<option value={30}>Last 30 days</option>
				<option value={90}>Last 90 days</option>
				<option value={365}>Last 12 months</option>
			</select>
		</label>
	</div>

	{#if error}
		<Status kind="error" message={error} retry={load} />
	{:else if !data}
		<Status kind="loading" message="Loading your progress…" />
	{:else}
		<ul class="kpis" aria-label="Key figures">
			{#each data.kpis as k (k.id)}
				{@const t = trend(k.value, k.previous, k.unit)}
				<li class="glass">
					<a href={k.href}>
						<span class="label">{k.label}</span>
						<strong>{fmt(k.unit, k.value)}</strong>
						{#if t}<span class="trend {t.dir}"
								>{t.dir === 'up' ? '▲' : t.dir === 'down' ? '▼' : '■'} {t.text}</span
							>{/if}
					</a>
					<details>
						<summary>How is this counted?</summary>
						<p>{k.definition}</p>
					</details>
				</li>
			{/each}
		</ul>

		<div class="grid">
			<div class="panel glass wide">
				<LineChart title="Daily activity" series={data.series} />
			</div>
			<div class="panel glass wide">
				<Heatmap days={data.activity} title="The last 12 months" />
			</div>
			<div class="panel glass full">
				<SkillNetwork nodes={data.skills.nodes} edges={data.skills.edges} />
			</div>
			<div class="panel glass">
				<h3>Due in the next 7 days</h3>
				{#if data.tables.due_soon.length}
					<table>
						<thead><tr><th scope="col">Task</th><th scope="col">Due</th></tr></thead>
						<tbody>
							{#each data.tables.due_soon as t (t.id)}
								<tr><td><a href="/tasks/{t.id}">{t.title}</a></td><td>{when(t.due_on)}</td></tr>
							{/each}
						</tbody>
					</table>
				{:else}
					<p class="muted">Nothing due this week.</p>
				{/if}
			</div>
			<div class="panel glass">
				<h3>Learning now</h3>
				{#if data.tables.learning.length}
					<table>
						<thead><tr><th scope="col">Skill</th><th scope="col">Roadmap</th></tr></thead>
						<tbody>
							{#each data.tables.learning as s (s.id)}
								<tr>
									<td><a href="/roadmaps/{s.roadmap_id}?skill={s.id}">{s.title}</a></td>
									<td>{s.roadmap_title}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{:else}
					<p class="muted">No skill in progress. <a href="/roadmaps">Pick one</a>.</p>
				{/if}
			</div>
			<div class="panel glass">
				<h3>Projects without activity for 14 days</h3>
				{#if data.tables.stalled_projects.length}
					<table>
						<thead><tr><th scope="col">Project</th><th scope="col">Last activity</th></tr></thead>
						<tbody>
							{#each data.tables.stalled_projects as p (p.id)}
								<tr
									><td><a href="/projects/{p.id}">{p.title}</a></td><td>{when(p.last_activity)}</td
									></tr
								>
							{/each}
						</tbody>
					</table>
				{:else}
					<p class="muted">Every active project moved in the last two weeks.</p>
				{/if}
			</div>
		</div>
	{/if}
</section>

<style>
	.overview {
		margin-top: var(--space-8);
	}
	.head {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-3);
		flex-wrap: wrap;
	}
	.head h2 {
		font-size: var(--text-xl);
		margin: 0;
	}
	select {
		font: inherit;
		border: 1px solid var(--border);
		border-radius: var(--radius-sm);
		background: var(--surface);
		color: var(--fg);
		padding: var(--space-2) var(--space-3);
	}
	.kpis {
		list-style: none;
		padding: 0;
		margin: var(--space-4) 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 11rem), 1fr));
		gap: var(--space-3);
	}
	.kpis li {
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		background: var(--surface);
		padding: var(--space-3) var(--space-4);
	}
	.kpis a {
		display: grid;
		gap: var(--space-1);
		color: var(--fg);
		text-decoration: none;
	}
	.kpis strong {
		font-size: 2rem;
		line-height: 1.1;
		letter-spacing: -0.02em;
	}
	.label {
		font-size: var(--text-sm);
		color: var(--muted);
	}
	.trend {
		font-size: var(--text-sm);
	}
	.trend.up {
		color: var(--ok);
	}
	.trend.down {
		color: var(--bad);
	}
	.trend.flat {
		color: var(--muted);
	}
	details {
		font-size: var(--text-sm);
		color: var(--muted);
	}
	summary {
		cursor: pointer;
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 20rem), 1fr));
		gap: var(--space-4);
	}
	.panel {
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		background: var(--surface);
		padding: var(--space-4);
		min-width: 0;
	}
	.panel.full {
		grid-column: 1 / -1;
	}
	@media (min-width: 60rem) {
		.panel.wide {
			grid-column: span 2;
		}
	}
	h3 {
		font-size: var(--text-base);
		margin: 0 0 var(--space-2);
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: var(--text-sm);
	}
	th,
	td {
		text-align: left;
		padding: var(--space-1) var(--space-2);
		border-bottom: 1px solid var(--border);
	}
</style>
