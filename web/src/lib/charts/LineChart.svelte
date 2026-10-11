<script lang="ts">
	import type { DayCount } from '$lib/overview';

	let {
		title,
		series
	}: { title: string; series: { id: string; label: string; points: DayCount[] }[] } = $props();
	const W = 640;
	const H = 200;
	const PAD = 28;
	const colors = ['var(--chart-1)', 'var(--chart-2)', 'var(--chart-3)'];
	const days = $derived(series[0]?.points ?? []);
	const max = $derived(Math.max(1, ...series.flatMap((s) => s.points.map((p) => p.value))));
	const x = (i: number) => PAD + (i * (W - 2 * PAD)) / Math.max(1, days.length - 1);
	const y = (v: number) => H - PAD - (v * (H - 2 * PAD)) / max;
	const path = (pts: DayCount[]) =>
		pts.map((p, i) => `${i ? 'L' : 'M'}${x(i).toFixed(1)},${y(p.value).toFixed(1)}`).join(' ');
	const totals = $derived(series.map((s) => s.points.reduce((a, p) => a + p.value, 0)));
	const short = (d: string) =>
		new Date(d + 'T00:00:00').toLocaleDateString(undefined, { day: 'numeric', month: 'short' });
</script>

<figure class="chart">
	<figcaption>
		{title}
		<span class="legend">
			{#each series as s, i (s.id)}
				<span><i style:background={colors[i]}></i>{s.label} ({totals[i]})</span>
			{/each}
		</span>
	</figcaption>
	<svg
		viewBox="0 0 {W} {H}"
		role="img"
		aria-label="{title}: {series.map((s, i) => `${s.label} ${totals[i]}`).join(', ')}"
	>
		{#each [0, 0.5, 1] as t (t)}
			<line class="grid" x1={PAD} x2={W - PAD} y1={y(max * t)} y2={y(max * t)} />
			<text class="axis" x={PAD - 6} y={y(max * t) + 4} text-anchor="end"
				>{Math.round(max * t)}</text
			>
		{/each}
		{#if days.length}
			<text class="axis" x={PAD} y={H - 8}>{short(days[0].day)}</text>
			<text class="axis" x={W - PAD} y={H - 8} text-anchor="end"
				>{short(days[days.length - 1].day)}</text
			>
		{/if}
		{#each series as s, i (s.id)}
			<path d={path(s.points)} fill="none" stroke={colors[i]} stroke-width="2.5" />
		{/each}
	</svg>
	<details>
		<summary>Show as a table</summary>
		<div class="table-wrap">
			<table>
				<thead>
					<tr
						><th scope="col">Day</th>{#each series as s (s.id)}<th scope="col">{s.label}</th
							>{/each}</tr
					>
				</thead>
				<tbody>
					{#each days as d, i (d.day)}
						<tr>
							<th scope="row">{d.day}</th>
							{#each series as s (s.id)}<td>{s.points[i]?.value ?? 0}</td>{/each}
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</details>
</figure>

<style>
	.chart {
		margin: 0;
	}
	figcaption {
		font-weight: 650;
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2) var(--space-4);
		justify-content: space-between;
		margin-bottom: var(--space-2);
	}
	.legend {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
		font-weight: 400;
		font-size: var(--text-sm);
		color: var(--muted);
	}
	.legend i {
		display: inline-block;
		width: 0.75rem;
		height: 0.75rem;
		border-radius: 3px;
		margin-right: 0.3rem;
		vertical-align: -1px;
	}
	svg {
		width: 100%;
		height: auto;
	}
	.grid {
		stroke: var(--border);
	}
	.axis {
		font-size: 11px;
		fill: var(--muted);
	}
	summary {
		cursor: pointer;
		color: var(--accent);
		font-size: var(--text-sm);
	}
	.table-wrap {
		max-height: 16rem;
		overflow: auto;
	}
	table {
		border-collapse: collapse;
		font-size: var(--text-sm);
		width: 100%;
	}
	th,
	td {
		text-align: left;
		padding: var(--space-1) var(--space-2);
		border-bottom: 1px solid var(--border);
	}
</style>
