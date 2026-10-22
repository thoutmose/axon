<script lang="ts">
	import { onMount } from 'svelte';
	import { api, message, type Schemas } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import { progress } from '$lib/needs';
	import Button from '$lib/ui/Button.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// Daily needs (EP-88): reference intakes for a healthy adult, worked out
	// on the server from a profile, shown against what the day's log holds.
	let { totals }: { totals: Schemas['Nutrients'] | null } = $props();

	type Profile = Schemas['HealthProfile'];
	type Needs = Schemas['DailyNeeds'];
	let profile = $state<Profile | null>(null);
	let needs = $state<Needs | null>(null);
	let loaded = $state(false);
	let editing = $state(false);
	let busy = $state(false);
	let form = $state({
		birth_year: '',
		sex: 'unspecified',
		height_cm: '',
		weight_kg: '',
		activity: 'light',
		goal: 'maintain',
		pregnant_or_breastfeeding: false
	});

	// A bounded goal (EP-191): the needs move by a share, and a loss never
	// goes under the basal rate or 1,200 kcal, nor over maintenance; the bars and the chef follow them.
	const goals = $derived([
		['maintain', t('Keep my weight')],
		['lose', t('Lose weight gently: 15 % under, never under your basal rate or 1,200 kcal')],
		['gain', t('Gain weight slowly: 10 % over')]
	]);
	const goalNames: Record<string, string> = $derived({
		maintain: t('keeping your weight'),
		lose: t('losing weight gently'),
		gain: t('gaining weight slowly')
	});

	const activities = $derived([
		['sedentary', t('Sedentary: mostly sitting, little exercise')],
		['light', t('Light: exercise 1 to 3 days a week')],
		['moderate', t('Moderate: exercise 3 to 5 days a week')],
		['active', t('Active: hard exercise 6 or 7 days a week')],
		['very_active', t('Very active: hard daily training or a physical job')]
	]);

	function fill(p: Profile | null) {
		if (!p) return;
		form = {
			birth_year: String(p.birth_year),
			sex: p.sex,
			height_cm: String(p.height_cm),
			weight_kg: String(p.weight_kg),
			activity: p.activity,
			goal: p.goal ?? 'maintain',
			pregnant_or_breastfeeding: p.pregnant_or_breastfeeding
		};
	}

	onMount(async () => {
		try {
			const r = await api<{ profile: Profile | null; needs: Needs | null }>('/v1/health/needs');
			profile = r.profile;
			needs = r.needs;
			fill(profile);
			editing = !profile;
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			loaded = true;
		}
	});

	async function save(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			const r = await api<{ profile: Profile; needs: Needs }>('/v1/health/profile', {
				method: 'PUT',
				body: {
					birth_year: Number(form.birth_year),
					sex: form.sex,
					height_cm: Number(form.height_cm.replace(',', '.')),
					weight_kg: Number(form.weight_kg.replace(',', '.')),
					activity: form.activity,
					goal: form.goal,
					pregnant_or_breastfeeding: form.pregnant_or_breastfeeding
				}
			});
			profile = r.profile;
			needs = r.needs;
			editing = false;
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	const rows = $derived(
		needs?.available && totals
			? [
					{ label: t('Energy'), eaten: totals.kcal, need: needs.kcal, unit: 'kcal' },
					{ label: t('Protein'), eaten: totals.protein_g, need: needs.protein_g, unit: 'g' },
					{ label: t('Carbohydrates'), eaten: totals.carbs_g, need: needs.carbs_g, unit: 'g' },
					{ label: t('Fat'), eaten: totals.fat_g, need: needs.fat_g, unit: 'g' }
				]
			: []
	);
</script>

<section class="card needs" aria-labelledby="needs-title">
	<h2 id="needs-title">{t('Your daily needs')}</h2>
	<p class="warning" role="note">
		{t(
			'These are general estimates for healthy adults, worked out from published references. They are not medical or dietary advice: check them with a registered dietitian or nutritionist, especially if you are pregnant or breastfeeding, have a medical condition or take medication, have or had an eating disorder, or are under 18.'
		)}
	</p>

	{#if !loaded}
		<p class="muted">{t('Loading…')}</p>
	{:else if editing}
		<form onsubmit={save} class="profile">
			<label
				>{t('Birth year')}
				<input
					inputmode="numeric"
					required
					bind:value={form.birth_year}
					placeholder="1990"
				/></label
			>
			<label
				>{t('Sex, for the energy equation')}
				<select bind:value={form.sex}>
					<option value="female">{t('Female')}</option>
					<option value="male">{t('Male')}</option>
					<option value="unspecified">{t('Prefer not to say')}</option>
				</select></label
			>
			<label
				>{t('Height (cm)')}
				<input inputmode="decimal" required bind:value={form.height_cm} placeholder="170" /></label
			>
			<label
				>{t('Weight (kg)')}
				<input inputmode="decimal" required bind:value={form.weight_kg} placeholder="65" /></label
			>
			<label class="wide"
				>{t('Activity')}
				<select bind:value={form.activity}>
					{#each activities as [value, label] (value)}<option {value}>{label}</option>{/each}
				</select></label
			>
			<label class="wide"
				>{t('Goal')}
				<select bind:value={form.goal}>
					{#each goals as [value, label] (value)}<option {value}>{label}</option>{/each}
				</select></label
			>
			<label class="check wide"
				><input type="checkbox" bind:checked={form.pregnant_or_breastfeeding} />
				{t('I am pregnant or breastfeeding')}</label
			>
			<p class="small muted wide">
				{t(
					'Kept as health data, only with your consent, and deleted with it. Your weight is used for the numbers only; Axon never comments on it.'
				)}
			</p>
			<div class="wide actions">
				<Button type="submit" {busy}>{t('Work out my needs')}</Button>
				{#if profile}<Button variant="ghost" onclick={() => (editing = false)}>{t('Cancel')}</Button
					>{/if}
			</div>
		</form>
	{:else if needs && !needs.available}
		<p>
			{needs.reason === 'under_18'
				? t(
						'Daily needs are not worked out before 18: growth changes them. Ask a doctor or a registered dietitian.'
					)
				: t(
						'During pregnancy and breastfeeding, needs change from month to month: ask your midwife, your doctor, or a registered dietitian.'
					)}
		</p>
		<Button variant="secondary" onclick={() => (editing = true)}>{t('Edit my profile')}</Button>
	{:else if needs}
		<p class="small muted">{t('For your goal: {goal}.', { goal: goalNames[needs.goal] })}</p>
		<p class="kpis">
			<span><strong>{needs.kcal}</strong> kcal</span>
			<span><strong>{needs.protein_g}</strong> g {t('protein')}</span>
			<span><strong>{needs.carbs_g}</strong> g {t('carbohydrates')}</span>
			<span><strong>{needs.fat_g}</strong> g {t('fat')}</span>
			<span><strong>{needs.fiber_g}</strong> g {t('fibre')}</span>
			<span><strong>{needs.water_l}</strong> L {t('water')}</span>
		</p>
		{#if rows.length}
			<ul class="progress" aria-label={t('This day against your needs')}>
				{#each rows as r (r.label)}
					{@const p = progress(r.eaten, r.need)}
					<li>
						<span>{r.label}</span>
						<span
							class="bar"
							role="progressbar"
							aria-label={r.label}
							aria-valuemin="0"
							aria-valuemax={r.need}
							aria-valuenow={Math.min(r.eaten, r.need)}
							><span style:width="{p.share * 100}%"></span></span
						>
						<span class="small"
							>{t('{eaten} of {need} {unit}', {
								eaten: r.eaten,
								need: r.need,
								unit: r.unit
							})}{p.over ? ` · ${t('above the reference')}` : ''}</span
						>
					</li>
				{/each}
			</ul>
		{/if}
		<details>
			<summary>{t('How is this worked out?')}</summary>
			<ul class="small">
				{#each needs.explanation as line (line)}<li>{line}</li>{/each}
			</ul>
			<p class="small muted">
				{t(
					'With a goal, Axon moves these by 15 % at most, and never plans a loss under your basal rate or 1,200 kcal, nor when keeping your weight already takes 1,200 kcal or less: a gentle pace, not a diet. A dietitian can adapt them to you.'
				)}
			</p>
			<ul class="small sources">
				{#each needs.sources as s (s.url)}
					<li><a href={s.url} rel="noopener noreferrer" target="_blank">{s.title}</a></li>
				{/each}
			</ul>
		</details>
		<Button variant="secondary" onclick={() => (editing = true)}>{t('Edit my profile')}</Button>
	{/if}
</section>

<style>
	.needs {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin: var(--space-4) 0;
	}
	.warning {
		border-left: 4px solid var(--accent);
		background: var(--accent-soft);
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		font-size: var(--text-sm);
	}
	.profile {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 12rem), 1fr));
		gap: var(--space-3);
	}
	.profile label {
		display: grid;
		gap: var(--space-1);
	}
	.profile .wide {
		grid-column: 1 / -1;
	}
	.profile .check {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}
	.actions {
		display: flex;
		gap: var(--space-2);
	}
	.kpis {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-4);
	}
	.progress {
		list-style: none;
		padding: 0;
		display: grid;
		gap: var(--space-2);
	}
	.progress li {
		display: grid;
		grid-template-columns: 8rem 1fr auto;
		align-items: center;
		gap: var(--space-3);
	}
	.bar {
		height: 0.5rem;
		border-radius: 999px;
		background: var(--accent-soft);
		overflow: hidden;
	}
	.bar span {
		display: block;
		height: 100%;
		background: var(--accent);
	}
	.small {
		font-size: var(--text-sm);
	}
	summary {
		cursor: pointer;
		color: var(--accent);
	}
</style>
