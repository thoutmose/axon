<script lang="ts">
	import {
		acquired,
		anchor,
		lines,
		namesRoadmaps,
		percent,
		point,
		polygon,
		reached,
		short,
		type Spoke
	} from '$lib/radar';

	let { title, spokes }: { title: string; spokes: Spoke[] } = $props();
	const W = 440;
	const R = 100;
	const C = W / 2;
	const rings = [0.25, 0.5, 0.75, 1];
	const n = $derived(spokes.length);
	const done = $derived(spokes.reduce((a, s) => a + s.done, 0));
	const total = $derived(spokes.reduce((a, s) => a + s.total, 0));
	const summary = $derived(
		spokes.map((s) => `${s.label} ${percent(acquired(s))}`).join(', ') || 'no skills yet'
	);
	const ring = (v: number) =>
		polygon(
			spokes.map(() => v),
			R,
			C
		);
	const withRoadmap = $derived(namesRoadmaps(spokes));
	let active = $state<number | null>(null);
	// Labels above the centre grow upwards, below it downwards, and those at
	// the sides are centred on their spoke.
	const firstLine = (y: number, count: number) =>
		y < C - R * 0.5 ? -(count - 1) * 13 : y > C + R * 0.5 ? 6 : -(count - 1) * 6.5;
</script>

<figure class="radar">
	<figcaption>
		<span>{title}: {done} of {total} skills acquired</span>
		<span class="legend">
			<span><i class="acquired"></i>Acquired</span>
			<span><i class="reached"></i>With those being learnt</span>
		</span>
	</figcaption>
	{#if n >= 3}
		<svg viewBox="0 0 {W} {W - 110}" role="img" aria-label="{title}: {summary}">
			<g transform="translate(0 -55)">
				{#each rings as v (v)}
					<polygon class="grid" points={ring(v)} />
				{/each}
				<text class="axis" x={C + 3} y={C - R * 0.5 - 2}>50%</text>
				{#each spokes as s, i (i)}
					{@const end = point(i, n, 1, R, C)}
					{@const at = point(i, n, 1, R + 14, C)}
					{@const ls = withRoadmap ? [short(s.label), short(s.roadmap)] : lines(s.label)}
					<line class="grid" class:on={active === i} x1={C} y1={C} x2={end.x} y2={end.y} />
					<text class="label" x={at.x} y={at.y + 4} text-anchor={anchor(i, n)}>
						{#each ls as l, j (j)}
							<tspan class:sub={withRoadmap && j} x={at.x} dy={j ? 13 : firstLine(at.y, ls.length)}
								>{l}</tspan
							>
						{/each}
					</text>
				{/each}
				<polygon class="reached" points={polygon(spokes.map(reached), R, C)} />
				<polygon class="acquired" points={polygon(spokes.map(acquired), R, C)} />
				{#each spokes as s, i (i)}
					{@const p = point(i, n, acquired(s), R, C)}
					<circle class="dot" cx={p.x} cy={p.y} r="4" />
					<!-- A larger, invisible target shows the figures on hover. -->
					<circle
						class="hit"
						cx={p.x}
						cy={p.y}
						r="14"
						role="presentation"
						onpointerenter={() => (active = i)}
						onpointerleave={() => (active = null)}
						><title
							>{withRoadmap ? `${s.roadmap}: ` : ''}{s.label}: {s.done} of {s.total} acquired ({percent(
								acquired(s)
							)}), {s.learning} being learnt</title
						></circle
					>
				{/each}
			</g>
		</svg>
	{:else}
		<p class="muted small">
			A radar needs three areas. Add skills to more roadmaps, or sections to this one, to see its
			shape.
		</p>
	{/if}
	{#if n}
		<details open={n < 3}>
			<summary>Show as a table</summary>
			<table>
				<thead>
					<tr
						>{#if withRoadmap}<th scope="col">Roadmap</th>{/if}<th scope="col">Area</th><th
							scope="col">Acquired</th
						><th scope="col">Being learnt</th><th scope="col">Skills</th></tr
					>
				</thead>
				<tbody>
					{#each spokes as s, i (i)}
						<tr>
							{#if withRoadmap}<td>{s.roadmap}</td>{/if}
							<th scope="row">{s.label}</th>
							<td>{s.done} ({percent(acquired(s))})</td>
							<td>{s.learning}</td>
							<td>{s.total}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</details>
	{/if}
</figure>

<style>
	.radar {
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
		width: 0.85rem;
		height: 0.6rem;
		border-radius: 3px;
		margin-right: 0.3rem;
		vertical-align: 0;
	}
	.legend .acquired {
		background: var(--chart-1);
	}
	.legend .reached {
		border: 2px dashed var(--chart-2);
	}
	svg {
		width: 100%;
		max-width: 30rem;
		height: auto;
		display: block;
		margin: 0 auto;
	}
	.grid {
		fill: none;
		stroke: var(--border);
	}
	line.on {
		stroke: var(--muted);
	}
	.axis {
		font-size: 10px;
		fill: var(--muted);
	}
	.label {
		font-size: 12px;
		fill: var(--fg);
	}
	.sub {
		fill: var(--muted);
		font-size: 11px;
	}
	polygon.acquired {
		fill: var(--chart-1);
		fill-opacity: 0.22;
		stroke: var(--chart-1);
		stroke-width: 2;
		stroke-linejoin: round;
	}
	polygon.reached {
		fill: none;
		stroke: var(--chart-2);
		stroke-width: 2;
		stroke-dasharray: 5 4;
		stroke-linejoin: round;
	}
	.dot {
		fill: var(--chart-1);
		stroke: var(--surface);
		stroke-width: 2;
	}
	.hit {
		fill: transparent;
	}
	.small {
		font-size: var(--text-sm);
	}
	summary {
		cursor: pointer;
		color: var(--accent);
		font-size: var(--text-sm);
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
