<script lang="ts">
	import {
		acquired,
		anchor,
		level,
		lines,
		namesRoadmaps,
		percent,
		point,
		polygon,
		reached,
		short,
		type Spoke
	} from '$lib/radar';
	import { t } from '$lib/i18n.svelte';

	let { title, spokes }: { title: string; spokes: Spoke[] } = $props();
	const rings = [0.25, 0.5, 0.75, 1];
	const n = $derived(spokes.length);
	// A roadmap's domains (EP-163) can be dozens: the chart grows to keep them legible.
	const big = $derived(n > 12);
	const W = $derived(big ? 840 : 440);
	const R = $derived(big ? 250 : 100);
	const C = $derived(W / 2);
	// Domains the person has assessed show their level; progress shows beside it.
	// Around a big chart, each label runs along its spoke, read left to right.
	const turn = (i: number) => {
		const deg = (360 * i) / n - 90;
		return deg > 90 && deg < 270 ? deg - 180 : deg;
	};
	const leftSide = (i: number) => {
		const deg = (360 * i) / n - 90;
		return deg > 90 && deg < 270;
	};
	const leveled = $derived(spokes.some((s) => s.level != null));
	const done = $derived(spokes.reduce((a, s) => a + s.done, 0));
	const total = $derived(spokes.reduce((a, s) => a + s.total, 0));
	const summary = $derived(
		spokes.map((s) => `${label(s)} ${percent(acquired(s))}`).join(', ') || t('no skills yet')
	);
	const ring = (v: number) =>
		polygon(
			spokes.map(() => v),
			R,
			C
		);
	const withRoadmap = $derived(namesRoadmaps(spokes));
	let active = $state<number | null>(null);
	// The API names the folded spokes "Other"; roadmap and section names are the person's own.
	const label = (s: Spoke) => (s.label === 'Other' && !s.roadmap ? t('Other') : s.label);
	// Labels above the centre grow upwards, below it downwards, and those at
	// the sides are centred on their spoke.
	const firstLine = (y: number, count: number) =>
		y < C - R * 0.5 ? -(count - 1) * 13 : y > C + R * 0.5 ? 6 : -(count - 1) * 6.5;
</script>

<figure class="radar">
	<figcaption>
		<span>{t('{title}: {done} of {total} skills acquired', { title, done, total })}</span>
		<span class="legend">
			{#if leveled}
				<span><i class="acquired"></i>{t('Assessed level')}</span>
				<span><i class="reached"></i>{t('Done in the roadmap')}</span>
			{:else}
				<span><i class="acquired"></i>{t('Acquired')}</span>
				<span><i class="reached"></i>{t('With those being learnt')}</span>
			{/if}
		</span>
	</figcaption>
	{#if n >= 3}
		<svg class:big viewBox="0 0 {W} {big ? W : W - 110}" role="img" aria-label="{title}: {summary}">
			<g transform="translate(0 {big ? 0 : -55})">
				{#each rings as v (v)}
					<polygon class="grid" points={ring(v)} />
				{/each}
				<text class="axis" x={C + 3} y={C - R * 0.5 - 2}>50%</text>
				{#each spokes as s, i (i)}
					{@const end = point(i, n, 1, R, C)}
					{@const at = point(i, n, 1, R + 14, C)}
					{@const ls = withRoadmap
						? [short(label(s)), short(s.roadmap)]
						: big
							? [short(label(s), 22)]
							: lines(label(s))}
					<line class="grid" class:on={active === i} x1={C} y1={C} x2={end.x} y2={end.y} />
					{#if big}
						<text
							class="label"
							x={at.x}
							y={at.y + 4}
							text-anchor={leftSide(i) ? 'end' : 'start'}
							transform="rotate({turn(i)} {at.x} {at.y})">{ls[0]}</text
						>
					{:else}
						<text class="label" x={at.x} y={at.y + 4} text-anchor={anchor(i, n)}>
							{#each ls as l, j (j)}
								<tspan
									class:sub={withRoadmap && j}
									x={at.x}
									dy={j ? 13 : firstLine(at.y, ls.length)}>{l}</tspan
								>
							{/each}
						</text>
					{/if}
				{/each}
				<polygon class="reached" points={polygon(spokes.map(leveled ? acquired : reached), R, C)} />
				<polygon class="acquired" points={polygon(spokes.map(leveled ? level : acquired), R, C)} />
				{#each spokes as s, i (i)}
					{@const p = point(i, n, leveled ? level(s) : acquired(s), R, C)}
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
							>{withRoadmap ? `${s.roadmap}: ` : ''}{t(
								'{label}: {done} of {total} acquired ({percent}), {learning} being learnt',
								{
									label: label(s),
									done: s.done,
									total: s.total,
									percent: percent(acquired(s)),
									learning: s.learning
								}
							)}{s.level != null ? ' · ' + t('level {level} of 5', { level: s.level }) : ''}</title
						></circle
					>
				{/each}
			</g>
		</svg>
	{:else}
		<p class="muted small">
			{t(
				'A radar needs three areas. Add skills to more roadmaps, or sections to this one, to see its shape.'
			)}
		</p>
	{/if}
	{#if n}
		<details open={n < 3}>
			<summary>{t('Show as a table')}</summary>
			<table>
				<thead>
					<tr
						>{#if withRoadmap}<th scope="col">{t('Roadmap')}</th>{/if}<th scope="col"
							>{t('Area')}</th
						>{#if leveled}<th scope="col">{t('Level')}</th>{/if}<th scope="col">{t('Acquired')}</th
						><th scope="col">{t('Being learnt')}</th><th scope="col">{t('Skills')}</th></tr
					>
				</thead>
				<tbody>
					{#each spokes as s, i (i)}
						<tr>
							{#if withRoadmap}<td>{s.roadmap}</td>{/if}
							<th scope="row">{label(s)}</th>
							{#if leveled}<td>{s.level != null ? `${s.level} / 5` : '—'}</td>{/if}
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
	svg.big {
		max-width: 52rem;
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
