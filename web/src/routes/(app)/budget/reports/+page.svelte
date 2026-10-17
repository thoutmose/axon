<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { api, message, type Schemas } from '$lib/api';
	import BudgetCharts from '$lib/BudgetCharts.svelte';
	import { monthLabel } from '$lib/calendar';
	import { localDate } from '$lib/dates';
	import { formatCents } from '$lib/money';
	import Status from '$lib/ui/Status.svelte';

	// Reports (EP-138): a period, a month by default, with its totals, the two
	// donuts of where the money went and where it came from, and its year.
	type Report = Schemas['BudgetReport'];
	let month = $state(localDate(new Date()).slice(0, 7));
	let months = $state(1);
	let period = $state<Report | null>(null);
	let error = $state('');
	const money = (c: number) => formatCents(c, period?.currency ?? 'EUR');

	$effect(() => {
		const [m, n] = [month, months];
		if (!/^\d{4}-\d{2}$/.test(m)) return;
		api<Report>(`/v1/budget/report?months=${n}&month=${m}`)
			.then((p) => {
				period = p;
				error = '';
			})
			.catch((err) => (error = message(err)));
	});

	const income = $derived(period?.months.reduce((n, m) => n + m.income_cents, 0) ?? 0);
	const outgo = $derived(period?.months.reduce((n, m) => n + m.expense_cents, 0) ?? 0);
</script>

<svelte:head><title>{t('Reports · Budget · Axon')}</title></svelte:head>

<div class="row head">
	<h1>{t('Reports')}</h1>
	<div class="row">
		<label class="row small"
			>{t('Month')}
			<input type="month" bind:value={month} /></label
		>
		<label class="row small"
			>{t('Period')}
			<select bind:value={months}>
				<option value={1}>{t('That month')}</option>
				<option value={3}>{t('3 months')}</option>
				<option value={6}>{t('6 months')}</option>
				<option value={12}>{t('12 months')}</option>
			</select></label
		>
	</div>
</div>

{#if error}
	<Status kind="error" message={error} />
{:else if !period}
	<Status kind="loading" message={t('Working it out…')} />
{:else}
	<section class="card" aria-labelledby="totals">
		<h2 id="totals">
			{months === 1 ? monthLabel(month) : t('{from} to {to}', { from: period.from, to: period.to })}
		</h2>
		<dl class="tiles">
			<div>
				<dt>{t('In')}</dt>
				<dd>{money(income)}</dd>
			</div>
			<div>
				<dt>{t('Out')}</dt>
				<dd>{money(outgo)}</dd>
			</div>
			<div>
				<dt>{t('Saved')}</dt>
				<dd class:bad={income - outgo < 0} class:ok={income - outgo > 0}>
					{money(income - outgo)}{income
						? ` · ${Math.round(((income - outgo) / income) * 100)} %`
						: ''}
				</dd>
			</div>
			<div>
				<dt>{t('Net worth today')}</dt>
				<dd>{money(period.net_worth_cents)}</dd>
			</div>
		</dl>
	</section>

	<BudgetCharts {month} {months} />
{/if}

<style>
	.head {
		justify-content: space-between;
		flex-wrap: wrap;
	}
	.tiles {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
		gap: var(--space-3);
		margin: 0;
	}
	.tiles dt {
		color: var(--muted);
		font-size: var(--text-sm);
	}
	.tiles dd {
		margin: 0;
		font-size: 1.5rem;
		font-weight: 650;
	}
</style>
