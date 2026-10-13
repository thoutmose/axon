<script lang="ts">
	import { appLocale } from '\$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import { heatGrid, type DayCount } from '$lib/overview';
	import { localDate } from '$lib/dates';

	let { days, title = t('Activity') }: { days: DayCount[]; title?: string } = $props();
	const CELL = 12;
	const GAP = 3;
	const LEFT = 0;
	const TOP = 16; // month labels
	// Columns are weeks starting on Monday, rows weekdays, the current week last (EP-99).
	const grid = $derived(heatGrid(days, appLocale() ?? 'en'));
	const weeks = $derived(grid.weeks);
	const today = localDate(new Date());
	const weekdayLabels = $derived(
		[0, 2, 4].map((row) => ({
			row,
			label: new Date(Date.UTC(2024, 0, 1 + row)).toLocaleDateString(appLocale(), {
				weekday: 'short',
				timeZone: 'UTC'
			})
		}))
	);
	// Open on the current month: the scroll starts at the end.
	let scroller = $state<HTMLElement>();
	$effect(() => {
		void days;
		if (scroller) scroller.scrollLeft = scroller.scrollWidth;
	});
	const total = $derived(days.reduce((a, d) => a + d.value, 0));
	const active = $derived(days.filter((d) => d.value > 0).length);
	const months = $derived.by(() => {
		const out: { label: string; changes: number; activeDays: number }[] = [];
		for (const d of days) {
			const label = new Date(d.day + 'T00:00:00').toLocaleDateString(appLocale(), {
				month: 'long',
				year: 'numeric'
			});
			let m = out[out.length - 1];
			if (!m || m.label !== label) out.push((m = { label, changes: 0, activeDays: 0 }));
			m.changes += d.value;
			if (d.value) m.activeDays++;
		}
		return out;
	});
</script>

<figure class="heat">
	<figcaption>
		{t('{title}: {total} changes on {active} of {length} days', {
			title,
			total,
			active,
			length: days.length
		})}
	</figcaption>
	<div class="grid">
		<!-- Weekday labels stay in view while the weeks scroll. -->
		<svg class="days" width="30" height={TOP + 7 * (CELL + GAP)} aria-hidden="true">
			{#each weekdayLabels as w (w.row)}
				<text class="label" x="0" y={TOP + w.row * (CELL + GAP) + 10}>{w.label}</text>
			{/each}
		</svg>
		<!-- svelte-ignore a11y_no_noninteractive_tabindex (a scrollable region must be reachable by keyboard) -->
		<div
			class="scroll"
			role="region"
			aria-label={t('{title}, scrollable', { title })}
			tabindex="0"
			bind:this={scroller}
		>
			<svg
				width={LEFT + weeks * (CELL + GAP)}
				height={TOP + 7 * (CELL + GAP)}
				role="img"
				aria-label={t('{title}: {total} changes on {active} of {length} days', {
					title,
					total,
					active,
					length: days.length
				})}
			>
				{#each grid.months as m (m.col)}
					<text class="label" x={LEFT + m.col * (CELL + GAP)} y="11">{m.label}</text>
				{/each}
				{#each grid.cells as c (c.day)}
					<rect
						x={LEFT + c.col * (CELL + GAP)}
						y={TOP + c.row * (CELL + GAP)}
						width={CELL}
						height={CELL}
						rx="2"
						class="l{c.level}"
						class:today={c.day === today}><title>{c.day}: {c.value}</title></rect
					>
				{/each}
			</svg>
		</div>
	</div>
	<p class="legend">
		<span
			><svg width="12" height="12" aria-hidden="true"
				><rect width="12" height="12" rx="2" class="l0" /></svg
			>{t('No activity')}</span
		>
		<span
			>{t('Less')}
			{#each [1, 2, 3, 4] as l (l)}<svg width="12" height="12" aria-hidden="true"
					><rect width="12" height="12" rx="2" class="l{l}" /></svg
				>{/each}
			{t('More')}</span
		>
		<span
			><svg width="12" height="12" aria-hidden="true"
				><rect x="1" y="1" width="10" height="10" rx="2" class="l0 today" /></svg
			>{t('Today')}</span
		>
	</p>
	<details>
		<summary>{t('Show by month')}</summary>
		<table>
			<thead
				><tr
					><th scope="col">{t('Month')}</th><th scope="col">{t('Changes')}</th><th scope="col"
						>{t('Active days')}</th
					></tr
				></thead
			>
			<tbody>
				{#each months as m (m.label)}
					<tr><th scope="row">{m.label}</th><td>{m.changes}</td><td>{m.activeDays}</td></tr>
				{/each}
			</tbody>
		</table>
	</details>
</figure>

<style>
	.heat {
		margin: 0;
	}
	figcaption {
		font-weight: 650;
		margin-bottom: var(--space-2);
	}
	.grid {
		display: flex;
	}
	.days {
		flex: none;
	}
	.scroll {
		overflow-x: auto;
		min-width: 0;
	}
	rect.l0 {
		fill: color-mix(in srgb, var(--muted) 18%, transparent);
	}
	rect.today {
		stroke: var(--fg);
		stroke-width: 1.5;
	}
	.label {
		font-size: 10px;
		fill: var(--muted);
	}
	.legend {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-4);
		font-size: var(--text-sm);
		color: var(--muted);
		margin: var(--space-2) 0;
	}
	.legend span {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
	}
	rect.l1 {
		fill: color-mix(in srgb, var(--ok) 35%, transparent);
	}
	rect.l2 {
		fill: color-mix(in srgb, var(--ok) 55%, transparent);
	}
	rect.l3 {
		fill: color-mix(in srgb, var(--ok) 78%, transparent);
	}
	rect.l4 {
		fill: var(--ok);
	}
	summary {
		cursor: pointer;
		color: var(--accent);
		font-size: var(--text-sm);
	}
	table {
		border-collapse: collapse;
		font-size: var(--text-sm);
	}
	th,
	td {
		text-align: left;
		padding: var(--space-1) var(--space-3);
		border-bottom: 1px solid var(--border);
	}
</style>
