<script lang="ts">
	import AgentDock from '$lib/AgentDock.svelte';
	import { pageAgents } from '$lib/pageAgents';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { api, message, type Schemas } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Entry = Schemas['FoodEntry'];
	type Food = Schemas['Food'];
	type Workout = Schemas['Workout'];
	type WorkoutSet = Schemas['WorkoutSet'];
	type Day = {
		date: string;
		entries: Entry[];
		totals: Schemas['Nutrients'];
		week: (Schemas['Nutrients'] & { date: string })[];
	};

	const meals = ['breakfast', 'lunch', 'dinner', 'snack'] as const;
	const mealName: Record<string, string> = {
		breakfast: 'Breakfast',
		lunch: 'Lunch',
		dinner: 'Dinner',
		snack: 'Snack'
	};
	const kinds = ['strength', 'cardio', 'mobility', 'sport', 'other'] as const;
	const kindName: Record<string, string> = {
		strength: 'Strength',
		cardio: 'Cardio',
		mobility: 'Mobility',
		sport: 'Sport',
		other: 'Other'
	};

	const tab = $derived(page.url.searchParams.get('tab') === 'training' ? 'training' : 'food');
	let status = $state<{ consented: boolean; lookups_consented: boolean; usda: boolean } | null>(
		null
	);
	let error = $state('');
	const today = new Date().toLocaleDateString('en-CA');

	async function loadStatus() {
		try {
			status = await api('/v1/health/status');
		} catch (err) {
			error = message(err);
		}
	}
	onMount(loadStatus);

	async function consent(purpose: 'health_data' | 'food_lookups') {
		try {
			await api(`/v1/me/consents/${purpose}`, { method: 'PUT', body: { granted: true } });
			await loadStatus();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function deleteAll() {
		if (!confirm(t('Delete every food entry and workout? This cannot be undone.'))) return;
		try {
			await api('/v1/health/data', { method: 'DELETE' });
			toast(t('Your health data is deleted.'));
			await Promise.all([loadDay(), loadWorkouts()]);
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	// --- Food ---
	let date = $state(today);
	let day = $state<Day | null>(null);
	let entry = $state({ meal: 'lunch', name: '', kcal: '', protein_g: '', carbs_g: '', fat_g: '' });
	let query = $state('');
	let foods = $state<Food[] | null>(null);
	let picked = $state<Food | null>(null);
	let grams = $state('');
	let searching = $state(false);

	async function loadDay() {
		try {
			day = await api<Day>(`/v1/health/food?date=${date}`);
		} catch (err) {
			error = message(err);
		}
	}

	const num = (v: string) => (v.trim() === '' ? undefined : Number(v.replace(',', '.')));

	async function addTyped(e: SubmitEvent) {
		e.preventDefault();
		try {
			await api('/v1/health/food', {
				method: 'POST',
				body: {
					eaten_on: date,
					meal: entry.meal,
					name: entry.name,
					kcal: num(entry.kcal),
					protein_g: num(entry.protein_g),
					carbs_g: num(entry.carbs_g),
					fat_g: num(entry.fat_g)
				}
			});
			entry = { ...entry, name: '', kcal: '', protein_g: '', carbs_g: '', fat_g: '' };
			await loadDay();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function search(e: SubmitEvent) {
		e.preventDefault();
		searching = true;
		try {
			const r = await api<{ items: Food[] }>(
				`/v1/health/foods/search?q=${encodeURIComponent(query.trim())}`
			);
			foods = r.items;
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			searching = false;
		}
	}

	function pick(f: Food) {
		picked = f;
		grams = f.serving_g ? String(f.serving_g) : '100';
	}

	async function addPicked(e: SubmitEvent) {
		e.preventDefault();
		if (!picked) return;
		try {
			await api('/v1/health/food', {
				method: 'POST',
				body: { eaten_on: date, meal: entry.meal, food_id: picked.id, grams: num(grams) }
			});
			picked = null;
			foods = null;
			query = '';
			await loadDay();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function removeEntry(id: string) {
		try {
			await api(`/v1/health/food/${id}`, { method: 'DELETE' });
			await loadDay();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	const weekMax = $derived(Math.max(1, ...(day?.week ?? []).map((d) => d.kcal)));

	// --- Training ---
	let workouts = $state<{
		items: Workout[];
		week: { sessions: number; minutes: number; days: number };
	} | null>(null);
	const emptySet = (): WorkoutSet => ({ exercise: '', reps: null, weight_kg: null });
	let wk = $state({
		done_on: today,
		kind: 'strength',
		title: '',
		minutes: '',
		effort: '',
		notes: '',
		sets: [emptySet()]
	});

	async function loadWorkouts() {
		try {
			workouts = await api('/v1/health/workouts');
		} catch (err) {
			error = message(err);
		}
	}

	async function addWorkout(e: SubmitEvent) {
		e.preventDefault();
		try {
			await api('/v1/health/workouts', {
				method: 'POST',
				body: {
					done_on: wk.done_on,
					kind: wk.kind,
					title: wk.title,
					minutes: num(wk.minutes) ?? null,
					effort: num(wk.effort) ?? null,
					notes: wk.notes,
					sets: wk.sets
						.filter((s) => s.exercise.trim())
						.map((s) => ({
							exercise: s.exercise,
							reps: s.reps ? Number(s.reps) : null,
							weight_kg: s.weight_kg ? Number(s.weight_kg) : null
						}))
				}
			});
			wk = { ...wk, title: '', minutes: '', effort: '', notes: '', sets: [emptySet()] };
			toast(t('Workout saved.'));
			await loadWorkouts();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function removeWorkout(id: string) {
		try {
			await api(`/v1/health/workouts/${id}`, { method: 'DELETE' });
			await loadWorkouts();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	$effect(() => {
		if (!status?.consented) return;
		if (tab === 'food') {
			void date;
			loadDay();
		} else loadWorkouts();
	});

	const setText = (s: WorkoutSet) =>
		[
			s.exercise,
			s.reps ? `${s.reps} ×` : '',
			s.weight_kg ? `${s.weight_kg} kg` : '',
			s.distance_km ? `${s.distance_km} km` : '',
			s.seconds ? `${Math.round(s.seconds / 60)} min` : ''
		]
			.filter(Boolean)
			.join(' ');
</script>

<svelte:head><title>{t('Health')} · Axon</title></svelte:head>

<h1>{t('Food and training')}</h1>
<AgentDock {...pageAgents.health} />

{#if error}
	<Status kind="error" message={error} retry={loadStatus} />
{:else if !status}
	<Status kind="loading" message={t('Loading…')} />
{:else if !status.consented}
	<section class="card" aria-labelledby="consent">
		<h2 id="consent">{t('Health logging is off')}</h2>
		<p>
			{t(
				'Food and training logs are health data, a special category under the GDPR. Axon stores them only with your explicit consent, keeps them out of search and AI agents, and deletes them all if you turn this off again.'
			)}
		</p>
		<Button onclick={() => consent('health_data')}>{t('Turn on health logging')}</Button>
	</section>
{:else}
	<nav class="tabs" aria-label={t('Health sections')}>
		<a href="?tab=food" aria-current={tab === 'food' ? 'page' : undefined}>{t('Food')}</a>
		<a href="?tab=training" aria-current={tab === 'training' ? 'page' : undefined}
			>{t('Training')}</a
		>
	</nav>

	{#if tab === 'food'}
		<label class="row">{t('Day')} <input type="date" bind:value={date} max="9999-12-31" /></label>
		{#if day}
			<section class="card" aria-labelledby="totals">
				<h2 id="totals">{t('This day')}</h2>
				<p class="kpis">
					<span><strong>{day.totals.kcal}</strong> kcal</span>
					<span><strong>{day.totals.protein_g}</strong> g {t('protein')}</span>
					<span><strong>{day.totals.carbs_g}</strong> g {t('carbohydrates')}</span>
					<span><strong>{day.totals.fat_g}</strong> g {t('fat')}</span>
				</p>
				<figure>
					<figcaption class="small muted">{t('Energy over the last seven days')}</figcaption>
					<ol class="bars">
						{#each day.week as d (d.date)}
							<li>
								<span class="bar" style:height="{(d.kcal / weekMax) * 100}%" aria-hidden="true"
								></span>
								<span class="small">{d.date.slice(5)}</span>
								<span class="visually-hidden">{d.date}: {d.kcal} kcal</span>
							</li>
						{/each}
					</ol>
				</figure>
			</section>

			{#each meals as m (m)}
				{@const list = day.entries.filter((e) => e.meal === m)}
				{#if list.length}
					<section aria-labelledby="meal-{m}">
						<h2 id="meal-{m}">{t(mealName[m])}</h2>
						<ul class="entries">
							{#each list as e (e.id)}
								<li>
									<span
										>{e.name}{e.grams ? ` · ${e.grams} g` : ''}{e.kcal != null
											? ` · ${e.kcal} kcal`
											: ''}</span
									>
									<button
										class="link"
										onclick={() => removeEntry(e.id)}
										aria-label="{t('Delete')} {e.name}">{t('Delete')}</button
									>
								</li>
							{/each}
						</ul>
					</section>
				{/if}
			{/each}
		{/if}

		<section class="card" aria-labelledby="log">
			<h2 id="log">{t('Log food')}</h2>
			<label class="row"
				>{t('Meal')}
				<select bind:value={entry.meal}>
					{#each meals as m (m)}<option value={m}>{t(mealName[m])}</option>{/each}
				</select></label
			>
			{#if status.lookups_consented}
				<form class="row" onsubmit={search}>
					<label class="visually-hidden" for="food-q">{t('Find a food')}</label>
					<input
						id="food-q"
						bind:value={query}
						placeholder={t('Find a food')}
						maxlength="100"
						required
					/>
					<Button type="submit" variant="secondary" busy={searching}>{t('Search')}</Button>
				</form>
				{#if foods}
					{#if foods.length === 0}<p class="muted">
							{t('Nothing found. Type the numbers in below.')}
						</p>{/if}
					<ul class="entries" aria-label={t('Foods found')}>
						{#each foods as f (f.id)}
							<li>
								<span>{f.name}{f.brand ? ` (${f.brand})` : ''} · {f.kcal} kcal / 100 g</span>
								<Button
									variant="secondary"
									onclick={() => pick(f)}
									aria-label="{t('Choose')} {f.name}">{t('Choose')}</Button
								>
							</li>
						{/each}
					</ul>
				{/if}
				{#if picked}
					<form class="row" onsubmit={addPicked}>
						<label
							>{t('Grams of')}
							{picked.name}
							<input inputmode="decimal" bind:value={grams} required /></label
						>
						<Button type="submit">{t('Add')}</Button>
					</form>
				{/if}
			{:else}
				<p class="small">
					{t(
						'To search foods, Axon sends the name or barcode you type to Open Food Facts or USDA FoodData Central. Nothing else is sent.'
					)}
					<Button variant="secondary" onclick={() => consent('food_lookups')}
						>{t('Allow food lookups')}</Button
					>
				</p>
			{/if}
			<form class="grid" onsubmit={addTyped}>
				<label class="wide"
					>{t('What you ate')} <input bind:value={entry.name} maxlength="300" required /></label
				>
				<label>kcal <input inputmode="decimal" bind:value={entry.kcal} /></label>
				<label>{t('Protein')} (g) <input inputmode="decimal" bind:value={entry.protein_g} /></label>
				<label
					>{t('Carbohydrates')} (g) <input inputmode="decimal" bind:value={entry.carbs_g} /></label
				>
				<label>{t('Fat')} (g) <input inputmode="decimal" bind:value={entry.fat_g} /></label>
				<div class="wide"><Button type="submit">{t('Log it')}</Button></div>
			</form>
		</section>
	{:else}
		{#if workouts}
			<section class="card" aria-labelledby="week">
				<h2 id="week">{t('Last seven days')}</h2>
				<p class="kpis">
					<span><strong>{workouts.week.sessions}</strong> {t('sessions')}</span>
					<span><strong>{workouts.week.minutes}</strong> {t('minutes')}</span>
					<span><strong>{workouts.week.days}</strong> {t('active days')}</span>
				</p>
			</section>
			{#if workouts.items.length === 0}
				<Status kind="empty" message={t('No workouts in the last 30 days. Any movement counts.')} />
			{:else}
				<ul class="workouts">
					{#each workouts.items as w (w.id)}
						<li class="card">
							<strong>{w.title}</strong>
							<span class="small muted"
								>{w.done_on} · {t(kindName[w.kind])}{w.minutes
									? ` · ${w.minutes} min`
									: ''}{w.effort ? ` · ${t('effort')} ${w.effort}/10` : ''}</span
							>
							{#if w.sets.length}
								<ul class="small">
									{#each w.sets as s, i (i)}<li>{setText(s)}</li>{/each}
								</ul>
							{/if}
							<button
								class="link"
								onclick={() => removeWorkout(w.id)}
								aria-label="{t('Delete')} {w.title}">{t('Delete')}</button
							>
						</li>
					{/each}
				</ul>
			{/if}
		{/if}

		<section class="card" aria-labelledby="new-workout">
			<h2 id="new-workout">{t('Log a workout')}</h2>
			<form class="grid" onsubmit={addWorkout}>
				<label class="wide"
					>{t('Title')} <input bind:value={wk.title} maxlength="200" required /></label
				>
				<label>{t('Date')} <input type="date" bind:value={wk.done_on} required /></label>
				<label
					>{t('Kind')}
					<select bind:value={wk.kind}>
						{#each kinds as k (k)}<option value={k}>{t(kindName[k])}</option>{/each}
					</select></label
				>
				<label>{t('Minutes')} <input inputmode="numeric" bind:value={wk.minutes} /></label>
				<label
					>{t('Effort (1–10)')}
					<input type="number" min="1" max="10" bind:value={wk.effort} /></label
				>
				<fieldset class="wide">
					<legend>{t('Sets')}</legend>
					{#each wk.sets as s, i (i)}
						<div class="row">
							<label
								>{t('Exercise')}
								{i + 1}
								<input bind:value={s.exercise} maxlength="120" /></label
							>
							<label>{t('Reps')} <input type="number" min="1" bind:value={s.reps} /></label>
							<label>kg <input inputmode="decimal" bind:value={s.weight_kg} /></label>
						</div>
					{/each}
					<Button variant="ghost" onclick={() => (wk.sets = [...wk.sets, emptySet()])}
						>{t('Add a set')}</Button
					>
				</fieldset>
				<label class="wide">{t('Notes')} <textarea rows="2" bind:value={wk.notes}></textarea></label
				>
				<div class="wide"><Button type="submit">{t('Save workout')}</Button></div>
			</form>
		</section>
	{/if}

	<footer class="attribution small muted">
		<p>
			{t('Food data from')} <a href="https://world.openfoodfacts.org/">Open Food Facts</a>
			(ODbL){#if status.usda}
				{' '}{t('and')} <a href="https://fdc.nal.usda.gov/">USDA FoodData Central</a>{/if}.
			{t('Axon gives no medical or dietary advice.')}
		</p>
		<Button variant="danger" onclick={deleteAll}>{t('Delete all my health data')}</Button>
	</footer>
{/if}

<style>
	.small {
		font-size: var(--text-sm);
	}
	.tabs {
		display: flex;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
	}
	.tabs a {
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		color: var(--muted);
		text-decoration: none;
	}
	.tabs a[aria-current='page'] {
		background: var(--accent-soft);
		color: var(--fg);
		font-weight: 600;
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: end;
		margin-bottom: var(--space-3);
	}
	.kpis {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-6);
	}
	.bars {
		list-style: none;
		padding: 0;
		margin: var(--space-2) 0 0;
		display: grid;
		grid-template-columns: repeat(7, 1fr);
		gap: var(--space-2);
		align-items: end;
		height: 8rem;
	}
	.bars li {
		display: grid;
		grid-template-rows: 1fr auto;
		height: 100%;
		align-items: end;
		justify-items: center;
	}
	.bar {
		width: 70%;
		min-height: 2px;
		background: var(--accent);
		border-radius: var(--radius) var(--radius) 0 0;
	}
	.entries,
	.workouts {
		list-style: none;
		padding: 0;
		display: grid;
		gap: var(--space-2);
	}
	.entries li {
		display: flex;
		justify-content: space-between;
		gap: var(--space-3);
		align-items: center;
	}
	.workouts li {
		display: grid;
		gap: var(--space-1);
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 10rem), 1fr));
		gap: var(--space-3);
	}
	.grid label {
		display: grid;
		gap: var(--space-1);
	}
	.wide {
		grid-column: 1 / -1;
	}
	fieldset {
		border: 1px solid var(--border);
		border-radius: var(--radius);
		padding: var(--space-3);
	}
	button.link {
		background: none;
		border: 0;
		padding: 0;
		color: var(--accent);
		text-decoration: underline;
		cursor: pointer;
		font: inherit;
		justify-self: start;
	}
	.attribution {
		margin-top: var(--space-8);
		border-top: 1px solid var(--border);
		padding-top: var(--space-3);
		display: grid;
		gap: var(--space-3);
		justify-items: start;
	}
</style>
