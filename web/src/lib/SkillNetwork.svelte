<script lang="ts">
	import { forceLayout, nodeHref, type NetEdge, type NetNode } from '$lib/overview';

	let { nodes, edges }: { nodes: NetNode[]; edges: NetEdge[] } = $props();
	const W = 900;
	const H = 560;
	let filter = $state<'all' | 'hard' | 'soft'>('all');
	const shown = $derived(
		nodes.filter((n) => filter === 'all' || n.type !== 'skill' || n.kind === filter)
	);
	const ids = $derived(new Set(shown.map((n) => n.id)));
	const visibleEdges = $derived(edges.filter((e) => ids.has(e.from) && ids.has(e.to)));
	const placed = $derived(forceLayout(shown, visibleEdges, W, H));
	const at = $derived(new Map(placed.map((p) => [p.id, p])));
	const counts = $derived({
		hard: nodes.filter((n) => n.kind === 'hard').length,
		soft: nodes.filter((n) => n.kind === 'soft').length
	});
	const cls = (n: NetNode) => (n.type === 'skill' ? `${n.kind} ${n.status}` : n.type);
	const radius = (n: NetNode) => (n.type === 'skill' ? (n.status === 'done' ? 9 : 7) : 11);
	const label = (n: NetNode) =>
		n.type === 'skill' ? `${n.title}, ${n.kind} skill, ${n.status}` : `${n.title}, ${n.type}`;
	let hover = $state<string | null>(null);
</script>

<figure class="network">
	<figcaption>
		<span>Skills network</span>
		<span class="filters" role="group" aria-label="Show skills">
			{#each [['all', 'All'], ['hard', `Hard (${counts.hard})`], ['soft', `Soft (${counts.soft})`]] as [v, l] (v)}
				<button
					type="button"
					aria-pressed={filter === v}
					onclick={() => (filter = v as typeof filter)}>{l}</button
				>
			{/each}
		</span>
	</figcaption>
	{#if nodes.length === 0}
		<p class="muted">
			No skills yet. Import a roadmap from the <a href="/roadmaps">library</a> to see your skills grow
			here.
		</p>
	{:else}
		<svg
			viewBox="0 0 {W} {H}"
			role="group"
			aria-label="Skills network: {counts.hard} hard skills and {counts.soft} soft skills"
		>
			{#each visibleEdges as e, i (i)}
				{@const a = at.get(e.from)}
				{@const b = at.get(e.to)}
				{#if a && b}
					<line
						x1={a.x}
						y1={a.y}
						x2={b.x}
						y2={b.y}
						class:requires={e.relation === 'requires'}
						class:hot={hover === a.id || hover === b.id}
					/>
				{/if}
			{/each}
			{#each placed as n (n.id)}
				<a
					href={nodeHref(n)}
					aria-label={label(n)}
					onmouseenter={() => (hover = n.id)}
					onmouseleave={() => (hover = null)}
					onfocus={() => (hover = n.id)}
					onblur={() => (hover = null)}
				>
					<circle cx={n.x} cy={n.y} r={radius(n)} class={cls(n)} />
					{#if n.type !== 'skill' || hover === n.id || placed.length <= 15 || (placed.length <= 60 && n.status !== 'todo')}
						<text x={n.x + radius(n) + 4} y={n.y + 4}>{n.title.slice(0, 32)}</text>
					{/if}
				</a>
			{/each}
		</svg>
		<p class="legend small">
			<span><i class="hard"></i>Hard skill</span>
			<span><i class="soft"></i>Soft skill</span>
			<span
				>Larger dots are learned; faded dots are still to learn; dashed lines are prerequisites.</span
			>
		</p>
		<details>
			<summary>Show as a list</summary>
			<ul class="list">
				{#each shown as n (n.id)}
					<li><a href={nodeHref(n)}>{label(n)}</a></li>
				{/each}
			</ul>
		</details>
	{/if}
</figure>

<style>
	.network {
		margin: 0;
	}
	figcaption {
		display: flex;
		justify-content: space-between;
		flex-wrap: wrap;
		gap: var(--space-2);
		font-weight: 650;
		margin-bottom: var(--space-2);
	}
	.filters {
		display: flex;
		gap: var(--space-1);
	}
	.filters button {
		font: inherit;
		font-size: var(--text-sm);
		font-weight: 500;
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		border-radius: 999px;
		padding: var(--space-1) var(--space-3);
		cursor: pointer;
	}
	.filters button[aria-pressed='true'] {
		background: var(--accent-soft);
		border-color: var(--accent);
	}
	svg {
		width: 100%;
		height: auto;
		border-radius: var(--radius);
		background: radial-gradient(circle at 50% 45%, var(--accent-soft), transparent 70%);
	}
	line {
		stroke: var(--border);
		stroke-width: 1.2;
	}
	line.requires {
		stroke-dasharray: 4 3;
	}
	line.hot {
		stroke: var(--accent);
		stroke-width: 2;
	}
	circle {
		stroke: var(--surface);
		stroke-width: 1.5;
	}
	circle.hard,
	i.hard {
		fill: var(--chart-1);
		background: var(--chart-1);
	}
	circle.soft,
	i.soft {
		fill: var(--chart-2);
		background: var(--chart-2);
	}
	circle.todo {
		opacity: 0.45;
	}
	a:focus-visible circle {
		stroke: var(--accent);
		stroke-width: 3;
	}
	text {
		font-size: 12px;
		fill: var(--fg);
		paint-order: stroke;
		stroke: var(--bg);
		stroke-width: 3px;
	}
	.legend {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2) var(--space-4);
		color: var(--muted);
		font-size: var(--text-sm);
	}
	.legend i {
		display: inline-block;
		width: 0.7rem;
		height: 0.7rem;
		border-radius: 50%;
		margin-right: 0.3rem;
	}
	summary {
		cursor: pointer;
		color: var(--accent);
		font-size: var(--text-sm);
	}
	.list {
		columns: 2 16rem;
		font-size: var(--text-sm);
	}
</style>
