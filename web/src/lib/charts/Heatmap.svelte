<script lang="ts">
	import { levels, type DayCount } from '$lib/overview';

	let { days, title = 'Activity' }: { days: DayCount[]; title?: string } = $props();
	const CELL = 12;
	const GAP = 3;
	// Columns are weeks starting on Monday; rows are weekdays.
	const offset = $derived(days.length ? (new Date(days[0].day + 'T00:00:00').getDay() + 6) % 7 : 0);
	const lv = $derived(levels(days));
	const weeks = $derived(Math.ceil((days.length + offset) / 7));
	const total = $derived(days.reduce((a, d) => a + d.value, 0));
	const active = $derived(days.filter((d) => d.value > 0).length);
	const months = $derived.by(() => {
		const out: { label: string; changes: number; activeDays: number }[] = [];
		for (const d of days) {
			const label = new Date(d.day + 'T00:00:00').toLocaleDateString(undefined, {
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
	<figcaption>{title}: {total} changes on {active} of {days.length} days</figcaption>
	<!-- svelte-ignore a11y_no_noninteractive_tabindex (a scrollable region must be reachable by keyboard) -->
	<div class="scroll" role="region" aria-label="{title}, scrollable" tabindex="0">
		<svg
			width={weeks * (CELL + GAP)}
			height={7 * (CELL + GAP)}
			role="img"
			aria-label="{title}: {total} changes on {active} of {days.length} days"
		>
			{#each days as d, i (d.day)}
				<rect
					x={Math.floor((i + offset) / 7) * (CELL + GAP)}
					y={((i + offset) % 7) * (CELL + GAP)}
					width={CELL}
					height={CELL}
					rx="2"
					class="l{lv[i]}"><title>{d.day}: {d.value}</title></rect
				>
			{/each}
		</svg>
	</div>
	<details>
		<summary>Show by month</summary>
		<table>
			<thead
				><tr
					><th scope="col">Month</th><th scope="col">Changes</th><th scope="col">Active days</th
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
	.scroll {
		overflow-x: auto;
	}
	rect.l0 {
		fill: var(--accent-soft);
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
