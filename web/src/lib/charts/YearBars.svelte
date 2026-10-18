<script lang="ts">
	import { t } from '$lib/i18n.svelte';

	// A year of income against spending (EP-138): two bars a month on one
	// axis, the month's balance on hover or focus, and a table elsewhere for
	// every figure.
	type Month = { label: string; income: number; expense: number };
	let { months, format }: { months: Month[]; format: (cents: number) => string } = $props();
	const top = $derived(Math.max(1, ...months.flatMap((m) => [m.income, m.expense])));
	const pct = (v: number) => `${(v / top) * 100}%`;
	let active = $state<number | null>(null);
	// At rest, the latest month with money in or out: this month, not December.
	const latest = $derived(months.findLastIndex((m) => m.income > 0 || m.expense > 0));
	const shown = $derived(active ?? (latest >= 0 ? latest : months.length - 1));
</script>

<figure>
	<p class="legend small">
		<span><i class="in"></i>{t('Income')}</span>
		<span><i class="out"></i>{t('Spending')}</span>
	</p>
	<div class="plot" role="img" aria-label={t('Income and spending by month')}>
		{#each months as m, i (m.label)}
			<div
				class="month"
				class:dim={active != null && active !== i}
				role="presentation"
				onmouseenter={() => (active = i)}
				onmouseleave={() => (active = null)}
				title="{m.label}: {format(m.income)} / {format(m.expense)}"
			>
				<div class="bars">
					<span class="bar in" style:height={pct(m.income)}></span>
					<span class="bar out" style:height={pct(m.expense)}></span>
				</div>
				<span class="label">{m.label}</span>
			</div>
		{/each}
	</div>
	{#if months.length}
		{@const m = months[shown]}
		<p class="small tip" aria-live="polite">
			<strong>{m.label}</strong> · {t('in')}
			{format(m.income)} · {t('out')}
			{format(m.expense)} · {t('balance')}
			<span class:bad={m.income < m.expense}>{format(m.income - m.expense)}</span>
		</p>
	{/if}
</figure>

<style>
	figure {
		margin: 0;
	}
	.plot {
		display: grid;
		grid-auto-flow: column;
		grid-auto-columns: minmax(0, 1fr);
		gap: var(--space-1);
		border-bottom: 1px solid var(--border);
	}
	.month {
		display: grid;
		grid-template-rows: 10rem auto;
		transition: opacity var(--motion-fast) var(--ease);
	}
	.month.dim {
		opacity: 0.4;
	}
	.bars {
		display: flex;
		align-items: flex-end;
		justify-content: center;
		gap: 2px;
	}
	.bar {
		width: min(1rem, 40%);
		border-radius: 4px 4px 0 0;
		min-height: 1px;
	}
	.bar.in {
		background: var(--series-1);
	}
	.bar.out {
		background: var(--series-2);
	}
	.label {
		text-align: center;
		font-size: 0.75rem;
		color: var(--muted);
		padding: var(--space-1) 0;
	}
	.legend {
		display: flex;
		gap: var(--space-3);
		margin: 0 0 var(--space-2);
	}
	.legend i {
		display: inline-block;
		width: 0.7rem;
		height: 0.7rem;
		border-radius: 3px;
		margin-right: var(--space-1);
		vertical-align: -0.05rem;
	}
	.legend i.in {
		background: var(--series-1);
	}
	.legend i.out {
		background: var(--series-2);
	}
	.small {
		font-size: var(--text-sm);
	}
	.tip {
		margin: var(--space-2) 0 0;
	}
	.bad {
		color: var(--bad);
	}
</style>
