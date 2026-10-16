<script lang="ts">
	import { onMount } from 'svelte';
	import { api, message, type Schemas } from '$lib/api';
	import { addDays } from '$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import {
		average,
		axes,
		level,
		score,
		total,
		weakest,
		type Axis,
		type DayLog,
		type Scores
	} from '$lib/lifeWeb';
	import { anchor, point, polygon } from '$lib/radar';

	// The life web (EP-131): today across food, water, sleep, activity,
	// caffeine, and mood, drawn over last week's average, with a level to reach
	// and the best day of the week to beat.
	type Week = { date: string; total_ml?: number; total_mg?: number };
	let today = $state<Scores | null>(null);
	let week = $state<Scores | null>(null);
	let best = $state<number | null>(null);
	let error = $state('');

	const name: Record<Axis, string> = $derived({
		food: t('Food'),
		water: t('Water'),
		sleep: t('Sleep'),
		activity: t('Activity'),
		caffeine: t('Caffeine'),
		mood: t('Mood')
	});
	const tip: Record<Axis, { text: string; href: string }> = $derived({
		food: { text: t('Log your meals'), href: '/health' },
		water: { text: t('Drink a glass of water'), href: '/health' },
		sleep: { text: t('Write how you slept'), href: '/journal' },
		activity: { text: t('Log a workout or a walk'), href: '/health?tab=training' },
		caffeine: { text: t('Log your coffee and tea'), href: '/coffee' },
		mood: { text: t('Write how you feel'), href: '/journal' }
	});

	onMount(async () => {
		try {
			const [food, water, caffeine, needs, journal] = await Promise.all([
				api<{ date: string; week: (Schemas['Nutrients'] & { date: string })[] }>('/v1/health/food'),
				api<Schemas['WaterDay']>('/v1/health/water'),
				api<{ limit_mg: number; week: Week[] }>('/v1/health/caffeine'),
				api<{ needs: Schemas['DailyNeeds'] | null }>('/v1/health/needs'),
				api<{ items: Schemas['JournalEntry'][]; today: string }>('/v1/journal')
			]);
			const end = journal.today;
			const days = Array.from({ length: 8 }, (_, i) => addDays(end, i - 7));
			const workouts = await api<{ items: Schemas['Workout'][] }>(
				`/v1/health/workouts?from=${days[0]}&to=${end}`
			);
			const n = needs.needs?.available ? needs.needs : null;
			const targets = {
				kcal: n?.kcal ?? 2000,
				protein_g: n?.protein_g ?? 50,
				water_ml: water.target_ml ?? 1600,
				caffeine_mg: caffeine.limit_mg,
				tracksActivity: workouts.items.length > 0,
				tracksCaffeine: caffeine.week.some((d) => (d.total_mg ?? 0) > 0)
			};
			const scores = days.map((d) => {
				const f = food.week.find((x) => x.date === d);
				const j = journal.items.find((x) => x.day === d);
				const log: DayLog = {
					kcal: f?.kcal ?? 0,
					protein_g: f?.protein_g ?? 0,
					water_ml: water.week.find((x) => x.date === d)?.total_ml ?? 0,
					sleep_hours: j?.sleep_hours ?? null,
					active_minutes: workouts.items
						.filter((w) => w.done_on === d)
						.reduce((a, w) => a + (w.minutes ?? 0), 0),
					caffeine_mg: caffeine.week.find((x) => x.date === d)?.total_mg ?? 0,
					mood: j?.mood ?? null
				};
				return score(log, targets);
			});
			today = scores[7];
			week = average(scores.slice(0, 7));
			const totals = scores
				.slice(0, 7)
				.map(total)
				.filter((x): x is number => x != null);
			best = totals.length ? Math.max(...totals) : null;
		} catch (err) {
			error = message(err);
		}
	});

	const W = 360;
	const R = 110;
	const C = W / 2;
	const n = axes.length;
	const ring = (v: number) => polygon(Array(n).fill(v), R, C);
	const values = (s: Scores) => axes.map((a) => (s[a] ?? 0) / 100);
	const points = $derived(today ? total(today) : null);
	const lvl = $derived(level(points));
	const next = $derived(today ? weakest(today) : null);
	const missing = $derived(today ? axes.filter((a) => today![a] == null) : []);
	const levelName: Record<string, string> = $derived({
		Platinum: t('Platinum'),
		Gold: t('Gold'),
		Silver: t('Silver'),
		Bronze: t('Bronze')
	});
</script>

<section class="card web" id="life-web" aria-labelledby="life-web-title">
	<h2 id="life-web-title">{t('Life web')}</h2>
	{#if error}
		<p class="muted">{error}</p>
	{:else if today && week}
		<div class="layout">
			<svg
				viewBox="0 0 {W} {W}"
				role="img"
				aria-label={t('Today: {scores}', {
					scores: axes
						.map((a) => `${name[a]} ${today![a] == null ? t('no data') : today![a]}`)
						.join(', ')
				})}
			>
				{#each [0.25, 0.5, 0.75, 1] as v (v)}
					<polygon class="grid" points={ring(v)} />
				{/each}
				{#each axes as a, i (a)}
					{@const end = point(i, n, 1, R, C)}
					{@const at = point(i, n, 1, R + 16, C)}
					<line class="grid" x1={C} y1={C} x2={end.x} y2={end.y} />
					<text class="label" x={at.x} y={at.y + 4} text-anchor={anchor(i, n)}
						>{name[a]}
						<tspan class="value">{today[a] ?? '–'}</tspan></text
					>
				{/each}
				<polygon class="week" points={polygon(values(week), R, C)} />
				<polygon class="today" points={polygon(values(today), R, C)} />
				{#each axes as a, i (a)}
					{#if today[a] != null}
						{@const p = point(i, n, today[a]! / 100, R, C)}
						<circle class="dot" cx={p.x} cy={p.y} r="4" />
					{/if}
				{/each}
			</svg>

			<div class="side">
				<p class="score">
					<strong>{points ?? '–'}</strong><span class="muted">/ 100</span>
					{#if lvl.reached}<span class="medal {lvl.reached.toLowerCase()}"
							>{levelName[lvl.reached]}</span
						>{/if}
				</p>
				{#if lvl.next}
					<p>
						{t('{points} points to {level}', {
							points: lvl.next.needs,
							level: levelName[lvl.next.name]
						})}
					</p>
					<div
						class="bar"
						role="progressbar"
						aria-label={t('Towards {level}', { level: levelName[lvl.next.name] })}
						aria-valuemin="0"
						aria-valuemax="100"
						aria-valuenow={points ?? 0}
					>
						<i style:width="{points ?? 0}%"></i>
					</div>
				{:else}
					<p>{t('The highest level. Keep it up.')}</p>
				{/if}
				{#if best != null}
					<p>
						{(points ?? 0) > best
							? t('Better than every day this week (best {best}).', { best })
							: t('Best day this week: {best}. Beat it.', { best })}
					</p>
				{/if}
				{#if next}
					<p>
						{t('Raise next: {axis}.', { axis: name[next] })}
						<a href={tip[next].href}>{tip[next].text}</a>
					</p>
				{/if}
				{#if missing.length}
					<p class="muted small">
						{t('Nothing logged today for: {axes}.', {
							axes: missing.map((a) => name[a]).join(', ')
						})}
					</p>
				{/if}
				<p class="legend small muted">
					<span><i class="today"></i>{t('Today')}</span>
					<span><i class="week"></i>{t('Last week, on average')}</span>
				</p>
			</div>
		</div>
	{:else}
		<p class="muted">{t('Loading…')}</p>
	{/if}
</section>

<style>
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}
	h2 {
		margin-top: 0;
	}
	.layout {
		display: grid;
		grid-template-columns: minmax(0, 22rem) 1fr;
		gap: var(--space-4);
		align-items: center;
	}
	@media (max-width: 40rem) {
		.layout {
			grid-template-columns: 1fr;
		}
	}
	svg {
		width: 100%;
		height: auto;
		overflow: visible;
	}
	.grid {
		fill: none;
		stroke: var(--border);
	}
	.week {
		fill: none;
		stroke: var(--muted);
		stroke-dasharray: 4 3;
	}
	.today {
		fill: color-mix(in srgb, var(--accent) 22%, transparent);
		stroke: var(--accent);
		stroke-width: 2;
		stroke-linejoin: round;
	}
	.dot {
		fill: var(--accent);
	}
	.label {
		font-size: 12px;
		fill: var(--fg);
	}
	.value {
		font-weight: 700;
		fill: var(--accent);
	}
	.score {
		display: flex;
		align-items: baseline;
		gap: var(--space-2);
		margin: 0 0 var(--space-2);
	}
	.score strong {
		font-size: 3rem;
		line-height: 1;
	}
	.medal {
		align-self: center;
		font-size: var(--text-sm);
		font-weight: 600;
		border-radius: 999px;
		padding: 0.1em 0.7em;
		color: #1a1a1a;
	}
	.bronze {
		background: #e3b48c;
	}
	.silver {
		background: #d6dbe1;
	}
	.gold {
		background: #f5cf5b;
	}
	.platinum {
		background: linear-gradient(135deg, #e7f0f7, #b9cfe0);
	}
	.bar {
		height: 0.5rem;
		border-radius: 999px;
		background: var(--border);
		overflow: hidden;
		max-width: 20rem;
	}
	.bar i {
		display: block;
		height: 100%;
		background: var(--accent);
	}
	.small {
		font-size: var(--text-sm);
	}
	.legend {
		display: flex;
		gap: var(--space-3);
	}
	.legend i {
		display: inline-block;
		width: 1.2em;
		height: 0;
		margin-right: 0.4em;
		vertical-align: middle;
		border-top: 2px solid var(--accent);
	}
	.legend i.week {
		border-top: 2px dashed var(--muted);
	}
</style>
