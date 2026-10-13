<script lang="ts">
	import { api, ApiError, message, type Schemas } from '$lib/api';
	import { appLocale } from '$lib/dates';
	import { i18n, t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import { toast } from '$lib/ui/toast.svelte';
	import { GLASS, glasses, litres } from '$lib/water';

	// Water drunk (EP-101): a glass or a bottle in one tap, the day as
	// glasses against the drinks' share of the daily need, and the week.
	let { date = '' }: { date?: string } = $props();

	type Day = Schemas['WaterDay'];
	let day = $state<Day | null>(null);
	let off = $state(false);
	let busy = $state(false);
	let custom = $state('');

	async function load() {
		try {
			day = await api<Day>(`/v1/health/water${date ? `?date=${date}` : ''}`);
			off = false;
		} catch (err) {
			if (err instanceof ApiError && err.status === 403) off = true;
			else toast(message(err), 'error');
		}
	}
	$effect(() => {
		void date;
		load();
	});

	async function add(ml: number) {
		if (!day || !(ml >= 1 && ml <= 3000)) return;
		busy = true;
		try {
			await api('/v1/health/water', { method: 'POST', body: { ml, drunk_on: day.date } });
			custom = '';
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
	async function remove(id: string) {
		try {
			await api(`/v1/health/water/${id}`, { method: 'DELETE' });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	const locale = $derived(i18n.locale);
	const g = $derived(day ? glasses(day.total_ml, day.target_ml) : { filled: 0, total: 8 });
	const weekMax = $derived(
		Math.max(day?.target_ml ?? 0, ...(day?.week ?? []).map((d) => d.total_ml), 1)
	);
	const time = (iso: string) =>
		new Date(iso).toLocaleTimeString(appLocale(), { hour: '2-digit', minute: '2-digit' });
</script>

<section class="card water" aria-labelledby="water-title">
	<h2 id="water-title">{t('Water')}</h2>
	{#if off}
		<p class="muted">
			{t('Water is health data: turn on health logging on the')}
			<a href="/health">{t('Health')}</a>
			{t('page to keep it.')}
		</p>
	{:else if !day}
		<p class="muted">{t('Loading…')}</p>
	{:else}
		<p class="total">
			{#if day.target_ml}
				{t('{drunk} of {target} L', {
					drunk: litres(day.total_ml, locale),
					target: litres(day.target_ml, locale)
				})}
			{:else}
				{t('{drunk} L drunk', { drunk: litres(day.total_ml, locale) })}
			{/if}
		</p>
		<div
			class="glasses"
			role="img"
			aria-label={t('{filled} of {total} glasses', { filled: g.filled, total: g.total })}
		>
			{#each Array.from({ length: g.total }, (_, i) => i) as i (i)}
				<svg
					viewBox="0 0 24 32"
					width="22"
					height="30"
					aria-hidden="true"
					class:full={i < g.filled}
				>
					<path class="water-level" d="M6.4 12 H17.6 L16 29 H8 Z" />
					<path class="glass" d="M4 3 H20 L17.6 29 H6.4 Z" />
				</svg>
			{/each}
		</div>
		<div class="row">
			<Button onclick={() => add(GLASS)} {busy}>{t('+ Glass, 250 ml')}</Button>
			<Button variant="secondary" onclick={() => add(500)} {busy}>{t('+ Bottle, 500 ml')}</Button>
			<form
				class="custom"
				onsubmit={(e) => {
					e.preventDefault();
					add(Number(custom));
				}}
			>
				<label
					><span class="visually-hidden">{t('Millilitres')}</span><input
						inputmode="numeric"
						bind:value={custom}
						placeholder={t('ml')}
						size="5"
					/></label
				>
				<Button type="submit" variant="ghost" disabled={!custom}>{t('Add')}</Button>
			</form>
		</div>
		{#if day.entries.length}
			<ul class="entries" aria-label={t('Drinks this day')}>
				{#each day.entries as e (e.id)}
					<li>
						<span>{t('{time} · {ml} ml', { time: time(e.created_at), ml: e.ml })}</span>
						<button
							type="button"
							class="link"
							onclick={() => remove(e.id)}
							aria-label={t('Remove {ml} ml at {time}', { ml: e.ml, time: time(e.created_at) })}
							>{t('Remove')}</button
						>
					</li>
				{/each}
			</ul>
		{/if}
		<figure>
			<figcaption class="small muted">{t('The last seven days')}</figcaption>
			<ol class="week">
				{#each day.week as d (d.date)}
					<li>
						<span
							class="bar"
							style:height="{Math.round((d.total_ml / weekMax) * 56)}px"
							aria-hidden="true"
						></span>
						<span class="small">{d.date.slice(5)}</span>
						<span class="visually-hidden"
							>{t('{date}: {drunk} L', { date: d.date, drunk: litres(d.total_ml, locale) })}</span
						>
					</li>
				{/each}
			</ol>
		</figure>
		<p class="small muted">
			{day.target_ml
				? t(
						'Drinks give about 80 % of the water you need, food the rest (EFSA). Drink more when it is hot or when you train. A reference, not medical advice.'
					)
				: t(
						'Add your profile under Daily needs to see a target: drinks give about 80 % of the water you need, food the rest (EFSA).'
					)}
		</p>
	{/if}
</section>

<style>
	.water {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin: var(--space-4) 0;
	}
	.total {
		font-size: var(--text-xl);
		font-weight: 650;
		margin: 0 0 var(--space-2);
	}
	.glasses {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		margin-bottom: var(--space-3);
	}
	.glass {
		fill: none;
		stroke: var(--muted);
		stroke-width: 1.5;
		stroke-linejoin: round;
	}
	.water-level {
		fill: transparent;
	}
	.full .water-level {
		fill: #3b8fd9;
	}
	.full .glass {
		stroke: #2a6fb0;
	}
	.row,
	.custom {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
	}
	.custom input {
		width: 5rem;
		font: inherit;
		padding: var(--space-2);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		color: var(--fg);
	}
	.entries {
		list-style: none;
		padding: 0;
		margin: var(--space-3) 0;
		display: grid;
		gap: var(--space-1);
		font-size: var(--text-sm);
	}
	.entries li {
		display: flex;
		justify-content: space-between;
		max-width: 20rem;
	}
	.link {
		background: none;
		border: 0;
		color: var(--accent);
		cursor: pointer;
		font: inherit;
		padding: 0;
	}
	.week {
		list-style: none;
		padding: 0;
		margin: var(--space-2) 0;
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: var(--space-2);
		height: 5rem;
		align-items: end;
		max-width: 24rem;
	}
	.week li {
		display: grid;
		justify-items: center;
		align-content: end;
		height: 100%;
		gap: 2px;
	}
	.bar {
		width: 70%;
		min-height: 2px;
		background: #3b8fd9;
		border-radius: 3px 3px 0 0;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
