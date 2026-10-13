<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { layout, NODE_H, NODE_W, type Edge, type Skill } from './skillGraph';

	let {
		skills,
		edges,
		selected = null,
		onselect
	}: {
		skills: Skill[];
		edges: Edge[];
		selected?: string | null;
		onselect: (id: string) => void;
	} = $props();
	let zoom = $state(1);

	const g = $derived(layout(skills, edges));
	const pos = $derived(new Map(g.nodes.map((n) => [n.skill.id, n])));
	const done = $derived(skills.filter((s) => s.status === 'done').length);
	const short = (t: string) => (t.length > 26 ? t.slice(0, 25) + '…' : t);
</script>

<div class="toolbar">
	<button
		type="button"
		onclick={() => (zoom = Math.max(0.3, zoom - 0.2))}
		aria-label={t('Zoom out')}>−</button
	>
	<span class="muted small">{Math.round(zoom * 100)}%</span>
	<button type="button" onclick={() => (zoom = Math.min(2, zoom + 0.2))} aria-label={t('Zoom in')}
		>＋</button
	>
	<span class="legend small">
		<span class="dot todo"></span>
		{t('to learn')} <span class="dot learning"></span>
		{t('learning')}
		<span class="dot done"></span>
		{t('done')}
		<span class="dot skipped"></span>
		{t('skipped')}
	</span>
</div>
<div class="viewport glass">
	<svg
		width={(g.width + 40) * zoom}
		height={(g.height + 40) * zoom}
		viewBox="-20 -20 {g.width + 40} {g.height + 40}"
		role="group"
		aria-label={t(
			'Skill graph: {length} skills, {done} done. Arrows point from a skill to what it needs first.',
			{ length: skills.length, done }
		)}
	>
		<defs>
			<marker
				id="arrow"
				viewBox="0 0 10 10"
				refX="9"
				refY="5"
				markerWidth="6"
				markerHeight="6"
				orient="auto-start-reverse"
			>
				<path d="M 0 0 L 10 5 L 0 10 z" class="arrowhead" />
			</marker>
		</defs>
		{#each edges as e (e.from + e.to)}
			{@const a = pos.get(e.to)}
			{@const b = pos.get(e.from)}
			{#if a && b}
				<path
					class="edge"
					d="M {a.x + NODE_W} {a.y + NODE_H / 2} C {a.x + NODE_W + 35} {a.y + NODE_H / 2}, {b.x -
						35} {b.y + NODE_H / 2}, {b.x} {b.y + NODE_H / 2}"
					marker-end="url(#arrow)"
				/>
			{/if}
		{/each}
		{#each g.nodes as n (n.skill.id)}
			<a
				href="?view=graph&skill={n.skill.id}"
				onclick={(e) => {
					e.preventDefault();
					onselect(n.skill.id);
				}}
				aria-label="{n.skill.title}, {n.skill.status}"
			>
				<rect
					x={n.x}
					y={n.y}
					width={NODE_W}
					height={NODE_H}
					rx="12"
					class="node {n.skill.status}"
					class:selected={selected === n.skill.id}
				/>
				<text x={n.x + 12} y={n.y + 22} class="title">{short(n.skill.title)}</text>
				<text x={n.x + 12} y={n.y + 40} class="section">{short(n.skill.section)}</text>
			</a>
		{/each}
	</svg>
</div>

<style>
	.toolbar {
		display: flex;
		gap: var(--space-2);
		align-items: center;
		margin-bottom: var(--space-2);
		flex-wrap: wrap;
	}
	.toolbar button {
		font: inherit;
		min-width: 2.25rem;
		min-height: 2.25rem;
		border-radius: 999px;
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		cursor: pointer;
	}
	.small {
		font-size: var(--text-sm);
	}
	.legend {
		margin-left: auto;
		display: flex;
		align-items: center;
		gap: var(--space-1);
		color: var(--muted);
	}
	.dot {
		width: 0.75rem;
		height: 0.75rem;
		border-radius: 4px;
		display: inline-block;
		border: 1px solid var(--border);
		margin-left: var(--space-2);
	}
	.viewport {
		overflow: auto;
		max-height: 70vh;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
	}
	.edge {
		fill: none;
		stroke: var(--muted);
		stroke-opacity: 0.45;
		stroke-width: 1.5;
	}
	.arrowhead {
		fill: var(--muted);
		opacity: 0.6;
	}
	.node,
	.dot.todo {
		fill: var(--surface-solid);
		background: var(--surface-solid);
		stroke: var(--border);
	}
	.node.learning,
	.dot.learning {
		fill: #fff4d6;
		background: #fff4d6;
		stroke: #e6a700;
	}
	.node.done,
	.dot.done {
		fill: #dff5e5;
		background: #dff5e5;
		stroke: #2f9e5a;
	}
	.node.skipped,
	.dot.skipped {
		fill: var(--bg);
		background: var(--bg);
		stroke-dasharray: 4 3;
	}
	.node.selected {
		stroke: var(--accent);
		stroke-width: 3;
	}
	a:focus-visible rect {
		stroke: var(--accent);
		stroke-width: 3;
	}
	.title {
		font-size: 13px;
		font-weight: 600;
		fill: #1d1d1f;
	}
	.section {
		font-size: 11px;
		fill: #55555c;
	}
</style>
