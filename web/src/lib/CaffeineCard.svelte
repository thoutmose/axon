<script lang="ts">
	import { api, ApiError, message } from '$lib/api';
	import { advice, drinkNames } from '$lib/caffeine';
	import { appLocale } from '$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import { skyState } from '$lib/skyState.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// Coffee and tea (EP-117): each drink in one tap, the day against EFSA's
	// limit, and what is left at bedtime, with the last coffee that keeps
	// sleep intact.
	let { date = '' }: { date?: string } = $props();

	type Day = {
		date: string;
		total_mg: number;
		limit_mg: number;
		entries: { id: string; at: string; kind: string; mg: number }[];
		now_mg: number;
		bedtime: string;
		at_bedtime_mg: number;
		bedtime_mg: number;
		cutoff: string | null;
		half_life_hours: number;
		week: { date: string; total_mg: number }[];
		drinks: { kind: string; mg: number; ml: number }[];
	};
	let day = $state<Day | null>(null);
	let off = $state(false);
	let busy = $state(false);
	let custom = $state('');

	async function load() {
		try {
			day = await api<Day>(`/v1/health/caffeine${date ? `?date=${date}` : ''}`);
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

	async function add(kind: string, mg?: number) {
		busy = true;
		try {
			await api('/v1/health/caffeine', { method: 'POST', body: mg ? { kind, mg } : { kind } });
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
			await api(`/v1/health/caffeine/${id}`, { method: 'DELETE' });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	// In the account's time zone, where bedtime is.
	const time = (iso: string) =>
		new Date(iso).toLocaleTimeString(appLocale(), {
			hour: '2-digit',
			minute: '2-digit',
			timeZone: skyState.zone
		});
	const say = $derived(day ? advice(day, new Date()).key : null);
	const weekMax = $derived(
		Math.max(day?.limit_mg ?? 400, ...(day?.week ?? []).map((d) => d.total_mg))
	);
	const offered = $derived((day?.drinks ?? []).filter((d) => d.kind !== 'other'));
</script>

<section class="card caffeine" aria-labelledby="caffeine-title">
	<h2 id="caffeine-title">{t('Coffee and tea')}</h2>
	{#if off}
		<p class="muted">
			{t('Caffeine is health data: turn on health logging on the')}
			<a href="/health">{t('Health')}</a>
			{t('page to keep it.')}
		</p>
	{:else if !day}
		<p class="muted">{t('Loading…')}</p>
	{:else}
		<p class="total">
			{t('{mg} of {limit} mg of caffeine', { mg: day.total_mg, limit: day.limit_mg })}
		</p>
		<meter
			min="0"
			max={day.limit_mg}
			high={day.limit_mg * 0.75}
			optimum={0}
			value={Math.min(day.total_mg, day.limit_mg)}
			aria-label={t('Caffeine today against the daily limit')}
		></meter>
		<p class="advice" class:warn={say === 'over' || say === 'none' || say === 'past'} role="status">
			{#if say === 'over'}
				{t('Over the {limit} mg a day that is safe for you: no more caffeine today.', {
					limit: day.limit_mg
				})}
			{:else if say === 'before'}
				{t('A coffee is still fine until {time}, to sleep well at {bedtime}.', {
					time: time(day.cutoff!),
					bedtime: time(day.bedtime)
				})}
			{:else if say === 'past'}
				{t('Too late for a coffee: one now would still be in you at {bedtime}.', {
					bedtime: time(day.bedtime)
				})}
			{:else}
				{t('About {mg} mg will still be in you at {bedtime}: no more caffeine today.', {
					mg: day.at_bedtime_mg,
					bedtime: time(day.bedtime)
				})}
			{/if}
		</p>
		<p class="small muted">
			{t('In your body now: about {now} mg · at bedtime: about {bed} mg', {
				now: day.now_mg,
				bed: day.at_bedtime_mg
			})}
		</p>
		<div class="drinks">
			{#each offered as d (d.kind)}
				<Button
					variant="secondary"
					{busy}
					onclick={() => add(d.kind)}
					aria-label={t('Add {drink}, {mg} mg', { drink: t(drinkNames[d.kind]), mg: d.mg })}
					>{t(drinkNames[d.kind])} <span class="mg">{t('{mg} mg', { mg: d.mg })}</span></Button
				>
			{/each}
			<form
				class="custom"
				onsubmit={(e) => {
					e.preventDefault();
					const mg = Number(custom);
					if (mg >= 1 && mg <= 1000) add('other', mg);
				}}
			>
				<label
					><span class="visually-hidden">{t('Caffeine of another drink, in mg')}</span><input
						inputmode="numeric"
						bind:value={custom}
						placeholder={t('mg')}
						size="4"
					/></label
				>
				<Button type="submit" variant="ghost" disabled={!custom}>{t('Add')}</Button>
			</form>
		</div>
		{#if day.entries.length}
			<ul class="entries" aria-label={t('Caffeine this day')}>
				{#each day.entries as e (e.id)}
					<li>
						<span
							>{t('{time} · {drink} · {mg} mg', {
								time: time(e.at),
								drink: t(drinkNames[e.kind] ?? e.kind),
								mg: e.mg
							})}</span
						>
						<button
							type="button"
							class="link"
							onclick={() => remove(e.id)}
							aria-label={t('Remove {drink} at {time}', {
								drink: t(drinkNames[e.kind] ?? e.kind),
								time: time(e.at)
							})}>{t('Remove')}</button
						>
					</li>
				{/each}
			</ul>
		{/if}
		<figure>
			<figcaption class="small muted">
				{t('The last seven days, against the daily limit')}
			</figcaption>
			<ol class="week">
				{#each day.week as d (d.date)}
					<li>
						<span
							class="bar"
							class:over={d.total_mg > day.limit_mg}
							style:height="{Math.round((d.total_mg / weekMax) * 56)}px"
							aria-hidden="true"
						></span>
						<span class="small">{d.date.slice(5)}</span>
						<span class="visually-hidden"
							>{t('{date}: {mg} mg', { date: d.date, mg: d.total_mg })}</span
						>
					</li>
				{/each}
			</ol>
		</figure>
		<p class="small muted">
			{t(
				'Caffeine halves about every {hours} hours. Limits from EFSA (2015); the sleep cut-off from Gardiner et al. (2023). For information, not medical advice.',
				{ hours: day.half_life_hours }
			)}
			<a href="/account/preferences#caffeine">{t('Bedtime and metabolism')}</a>
		</p>
	{/if}
</section>

<style>
	.total {
		font-size: var(--text-xl);
		margin: 0 0 var(--space-2);
	}
	meter {
		width: 100%;
		height: 0.75rem;
	}
	.advice {
		margin: var(--space-2) 0;
	}
	.advice.warn {
		color: var(--bad);
	}
	.small {
		font-size: var(--text-sm);
	}
	.drinks {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		margin: var(--space-3) 0;
	}
	.mg {
		font-size: var(--text-sm);
		opacity: 0.8;
	}
	.custom {
		display: flex;
		gap: var(--space-1);
		align-items: center;
	}
	.entries {
		list-style: none;
		padding: 0;
		margin: 0 0 var(--space-3);
	}
	.entries li {
		display: flex;
		justify-content: space-between;
		gap: var(--space-2);
		padding: var(--space-1) 0;
		border-bottom: 1px solid var(--border);
	}
	.link {
		background: none;
		border: 0;
		color: var(--accent);
		cursor: pointer;
		font: inherit;
		padding: 0;
	}
	figure {
		margin: var(--space-3) 0;
	}
	.week {
		display: flex;
		gap: var(--space-2);
		align-items: flex-end;
		list-style: none;
		padding: 0;
		margin: var(--space-2) 0 0;
		height: 80px;
	}
	.week li {
		display: grid;
		justify-items: center;
		align-content: end;
		gap: var(--space-1);
		flex: 1;
	}
	.bar {
		width: 100%;
		max-width: 2rem;
		background: var(--accent);
		border-radius: 4px 4px 0 0;
		min-height: 2px;
	}
	.bar.over {
		background: var(--bad);
	}
</style>
