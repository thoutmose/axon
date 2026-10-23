<script lang="ts">
	import { appLocale } from '$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import { ago, type WatchStats } from '$lib/watch';

	// The watch at a glance: counters and news stored per day.
	let { stats }: { stats: WatchStats } = $props();

	const maxDay = $derived(Math.max(1, ...stats.daily.map((d) => d.count)));
	let hover = $state<{ day: string; count: number; x: number } | null>(null);
</script>

<section class="tiles" aria-label={t('At a glance')}>
	<div class="tile">
		<strong>{stats.sources.total}</strong><span>{t('sources')}</span>
		<small
			>{stats.sources.ok} OK{#if stats.sources.failing}
				· <span class="bad">{stats.sources.failing} {t('failing')}</span
				>{/if}{#if stats.sources.waiting}
				· {stats.sources.waiting} {t('waiting')}{/if}</small
		>
	</div>
	<div class="tile">
		<strong>{stats.items.unread}</strong><span>{t('unread')}</span>
		<small>{stats.items.today} {t('today')} · {stats.items.week} {t('this week')}</small>
	</div>
	<div class="tile">
		<strong>{stats.items.merged}</strong><span>{t('stories seen twice, kept once')}</span>
		<small>{stats.items.outliers} {t('outliers set aside')}</small>
	</div>
	<div class="tile">
		<strong>{stats.next_fetch_at ? ago(stats.next_fetch_at).replace('-', '') : '—'}</strong>
		<span>{t('next fetch')}</span>
		<small
			>{stats.next_fetch_at
				? new Date(stats.next_fetch_at).toLocaleString(appLocale(), {
						weekday: 'short',
						hour: '2-digit',
						minute: '2-digit'
					})
				: ''}</small
		>
	</div>
</section>

<figure class="chart">
	<figcaption>{t('News stored per day, last 30 days')}</figcaption>
	<svg viewBox="0 0 300 80" role="img" aria-label={t('News stored per day, last 30 days')}>
		<line x1="0" y1="79.5" x2="300" y2="79.5" class="axis" />
		{#each stats.daily as d, i (d.day)}
			{@const h = Math.max(d.count ? 2 : 0, (d.count / maxDay) * 72)}
			<rect x={i * 10 + 1} y={79 - h} width="8" height={h} rx="2" class="bar" />
			<rect
				x={i * 10}
				y="0"
				width="10"
				height="80"
				fill="transparent"
				role="presentation"
				onmouseenter={() => (hover = { ...d, x: i * 10 + 5 })}
				onmouseleave={() => (hover = null)}
			/>
		{/each}
	</svg>
	{#if hover}
		<div class="tip" style:left="{(hover.x / 300) * 100}%">
			{new Date(hover.day + 'T00:00:00').toLocaleDateString(appLocale())}:
			<strong>{hover.count}</strong>
		</div>
	{/if}
	<table class="sr-only">
		<caption>{t('News stored per day, last 30 days')}</caption>
		<tbody>
			{#each stats.daily as d (d.day)}<tr><th>{d.day}</th><td>{d.count}</td></tr>{/each}
		</tbody>
	</table>
</figure>

<style>
	.tiles {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(11rem, 1fr));
		gap: var(--space-3);
		margin: var(--space-4) 0;
	}
	.tile {
		display: grid;
		gap: 0.1rem;
		padding: var(--space-3);
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
	}
	.tile strong {
		font-size: var(--text-2xl);
	}
	.tile small {
		color: var(--muted);
	}
	.chart {
		position: relative;
		margin: 0 0 var(--space-6);
	}
	.chart figcaption {
		font-size: var(--text-sm);
		color: var(--muted);
		margin-bottom: var(--space-1);
	}
	.chart svg {
		width: 100%;
		height: 8rem;
	}
	.bar {
		fill: var(--accent);
	}
	.axis {
		stroke: var(--border);
		stroke-width: 1;
	}
	.tip {
		position: absolute;
		top: 1.2rem;
		transform: translateX(-50%);
		background: var(--surface-solid);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: 0.1rem var(--space-2);
		font-size: var(--text-sm);
		pointer-events: none;
		white-space: nowrap;
	}
	table {
		width: 100%;
		border-collapse: collapse;
	}
	th,
	td {
		text-align: left;
		vertical-align: top;
		padding: var(--space-2);
		border-bottom: 1px solid var(--border);
	}
	.bad {
		color: var(--bad);
	}
	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		overflow: hidden;
		clip: rect(0 0 0 0);
		white-space: nowrap;
	}
</style>
