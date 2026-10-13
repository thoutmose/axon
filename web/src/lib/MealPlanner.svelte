<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { mealRequest, restrictionNames, type MealPlanForm } from '$lib/mealPlan';
	import Button from '$lib/ui/Button.svelte';

	// Meal plans with the Chef & Nutritionist (EP-90): the form says what the
	// agent needs to know, and the page's agent dock runs it, where the plan
	// waits for approval before it becomes a note.
	let { onplan }: { onplan: (request: string) => void } = $props();

	let form = $state<MealPlanForm>({
		days: 3,
		people: 1,
		meals: ['lunch', 'dinner'],
		budget: '',
		currency: 'EUR',
		restrictions: [],
		minutes: 30,
		other: ''
	});
	const mealChoices = $derived([
		['breakfast', t('Breakfast')],
		['lunch', t('Lunch')],
		['dinner', t('Dinner')],
		['snack', t('Snack')]
	]);

	function submit(e: SubmitEvent) {
		e.preventDefault();
		if (!form.meals.length) return;
		onplan(mealRequest(form));
	}
</script>

<details class="planner">
	<summary>{t('Plan my meals with the chef')}</summary>
	<form onsubmit={submit}>
		<p class="small muted">
			{t(
				'The Chef & Nutritionist agent plans meals close to your daily needs, within your budget, restrictions, and time, with recipes and a shopping list. It uses your AI provider; the plan becomes a note only when you approve it.'
			)}
		</p>
		<div class="grid">
			<label
				>{t('Days')} <input type="number" min="1" max="7" required bind:value={form.days} /></label
			>
			<label
				>{t('People')}
				<input type="number" min="1" max="8" required bind:value={form.people} /></label
			>
			<label
				>{t('Minutes of cooking per meal, at most')}
				<select bind:value={form.minutes}>
					{#each [15, 30, 45, 60, 90] as m (m)}<option value={m}>{m}</option>{/each}
				</select></label
			>
			<label
				>{t('Budget for the whole plan (optional)')}
				<span class="money"
					><input inputmode="decimal" bind:value={form.budget} placeholder="40" />
					<select bind:value={form.currency} aria-label={t('Currency')}>
						{#each ['EUR', 'USD', 'GBP', 'CHF', 'CAD'] as c (c)}<option value={c}>{c}</option
							>{/each}
					</select></span
				></label
			>
		</div>
		<fieldset>
			<legend>{t('Meals')}</legend>
			{#each mealChoices as [value, label] (value)}
				<label class="check"
					><input type="checkbox" {value} bind:group={form.meals} /> {label}</label
				>
			{/each}
		</fieldset>
		<fieldset>
			<legend>{t('Dietary restrictions and allergies')}</legend>
			{#each Object.entries(restrictionNames) as [value, label] (value)}
				<label class="check"
					><input type="checkbox" {value} bind:group={form.restrictions} /> {t(label)}</label
				>
			{/each}
		</fieldset>
		<label class="wide"
			>{t('Other constraints (optional)')}
			<textarea
				rows="2"
				maxlength="500"
				bind:value={form.other}
				placeholder={t('No oven, no coriander, leftovers for lunch…')}
			></textarea></label
		>
		<p class="small muted">
			{t(
				'For an allergy, always check labels: the agent cannot guarantee a meal is free of traces. For a medical condition, pregnancy, or an eating disorder, plan with a registered dietitian.'
			)}
		</p>
		<Button type="submit" disabled={!form.meals.length}>{t('Ask the chef for a plan')}</Button>
	</form>
</details>

<style>
	.planner {
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-3) var(--space-4);
		margin: var(--space-4) 0;
		background: var(--surface);
	}
	summary {
		cursor: pointer;
		font-weight: 650;
	}
	form {
		display: grid;
		gap: var(--space-3);
		margin-top: var(--space-3);
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 12rem), 1fr));
		gap: var(--space-3);
	}
	label {
		display: grid;
		gap: var(--space-1);
	}
	.money {
		display: flex;
		gap: var(--space-2);
		min-width: 0;
	}
	/* Width 0 and flex 1: Firefox otherwise keeps the input's natural width. */
	.money input {
		flex: 1;
		min-width: 0;
		width: 0;
	}
	.money select {
		width: auto;
		flex: none;
	}
	fieldset {
		border: 1px solid var(--border);
		border-radius: var(--radius);
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2) var(--space-4);
	}
	.check {
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}
	textarea {
		font: inherit;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
