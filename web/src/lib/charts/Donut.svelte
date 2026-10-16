<script lang="ts">
	import { arc, type Slice } from '$lib/budgetCharts';
	import { t } from '$lib/i18n.svelte';

	// A donut of shares (EP-138): each slice named beside it with its amount and
	// share, so colour is never alone; hover a slice to light its row.
	let {
		title,
		slices,
		total,
		format
	}: { title: string; slices: Slice[]; total: string; format: (cents: number) => string } =
		$props();
	const C = 100;
	const R = 92;
	const INNER = 62;
	const GAP = 0.004; // a thin gap of the surface between slices
	let active = $state<number | null>(null);
	const parts = $derived.by(() => {
		let at = 0;
		return slices.map((s) => {
			const from = at;
			at += s.share;
			const gap = slices.length > 1 ? GAP : 0;
			return { ...s, d: arc(from + gap, at - gap, R, INNER, C) };
		});
	});
	const pct = (v: number) => `${Math.round(v * 100)} %`;
</script>

<figure class="donut">
	<figcaption>{title}</figcaption>
	{#if slices.length === 0}
		<p class="muted">{t('Nothing in this period.')}</p>
	{:else}
		<div class="layout">
			<svg viewBox="0 0 200 200" role="img" aria-label="{title}: {total}">
				{#each parts as p, i (p.label)}
					<path
						d={p.d}
						fill={p.color}
						class:dim={active != null && active !== i}
						role="presentation"
						onmouseenter={() => (active = i)}
						onmouseleave={() => (active = null)}
						><title>{p.label}: {format(p.cents)} ({pct(p.share)})</title></path
					>
				{/each}
				<text x={C} y={C - 4} class="total">{total}</text>
				<text x={C} y={C + 16} class="sub"
					>{active != null ? parts[active].label : t('in all')}</text
				>
			</svg>
			<table>
				<tbody>
					{#each parts as p, i (p.label)}
						<tr
							class:on={active === i}
							onmouseenter={() => (active = i)}
							onmouseleave={() => (active = null)}
						>
							<td><i style:background={p.color}></i>{p.label}</td>
							<td class="num">{format(p.cents)}</td>
							<td class="num muted">{pct(p.share)}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}
</figure>

<style>
	.donut {
		margin: 0;
	}
	figcaption {
		font-weight: 600;
		margin-bottom: var(--space-2);
	}
	.layout {
		display: grid;
		grid-template-columns: minmax(0, 11rem) minmax(0, 1fr);
		gap: var(--space-4);
		align-items: center;
	}
	@media (max-width: 30rem) {
		.layout {
			grid-template-columns: 1fr;
		}
		svg {
			max-width: 11rem;
		}
	}
	svg {
		width: 100%;
		height: auto;
	}
	path {
		transition: opacity var(--motion-fast) var(--ease);
	}
	path.dim {
		opacity: 0.35;
	}
	.total {
		text-anchor: middle;
		font-size: 17px;
		font-weight: 700;
		fill: var(--fg);
	}
	.sub {
		text-anchor: middle;
		font-size: 11px;
		fill: var(--muted);
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: var(--text-sm);
	}
	td {
		padding: 0.2rem var(--space-1);
	}
	tr.on td {
		background: var(--accent-soft);
	}
	td i {
		display: inline-block;
		width: 0.7rem;
		height: 0.7rem;
		border-radius: 3px;
		margin-right: var(--space-2);
		vertical-align: -0.05rem;
	}
	.num {
		text-align: right;
		white-space: nowrap;
	}
</style>
