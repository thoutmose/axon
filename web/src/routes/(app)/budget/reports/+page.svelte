<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { api, message, type Schemas } from '$lib/api';
	import { colourOrder, slices } from '$lib/budgetCharts';
	import { monthLabel } from '$lib/calendar';
	import Donut from '$lib/charts/Donut.svelte';
	import YearBars from '$lib/charts/YearBars.svelte';
	import { formatDate, localDate } from '$lib/dates';
	import { formatCents } from '$lib/money';
	import Status from '$lib/ui/Status.svelte';

	// Reports (EP-138): a period, a month by default, as two donuts of where the
	// money went and where it came from; below, the year to that month.
	type Report = Schemas['BudgetReport'];
	let month = $state(localDate(new Date()).slice(0, 7));
	let months = $state(1);
	let period = $state<Report | null>(null);
	let year = $state<Report | null>(null);
	let error = $state('');
	const money = (c: number) => formatCents(c, period?.currency ?? 'EUR');

	$effect(() => {
		const [m, n] = [month, months];
		if (!/^\d{4}-\d{2}$/.test(m)) return;
		Promise.all([
			api<Report>(`/v1/budget/report?months=${n}&month=${m}`),
			api<Report>(`/v1/budget/report?months=12&month=${m}`)
		])
			.then(([p, y]) => {
				period = p;
				year = y;
				error = '';
			})
			.catch((err) => (error = message(err)));
	});

	const income = $derived(period?.months.reduce((n, m) => n + m.income_cents, 0) ?? 0);
	const outgo = $derived(period?.months.reduce((n, m) => n + m.expense_cents, 0) ?? 0);
	const spending = $derived(
		period && year
			? slices(period.categories, 'expense', colourOrder(year.categories, 'expense'), t('Other'))
			: []
	);
	const earning = $derived(
		period && year
			? slices(period.categories, 'income', colourOrder(year.categories, 'income'), t('Other'))
			: []
	);
	const short = (m: string) => formatDate(`${m}-01`, { month: 'short' });
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
{:else if !period || !year}
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

	<div class="pies">
		<section class="card" aria-label={t('Where the money went')}>
			<Donut
				title={t('Where the money went')}
				slices={spending}
				total={money(outgo)}
				format={money}
			/>
		</section>
		<section class="card" aria-label={t('Where it came from')}>
			<Donut
				title={t('Where it came from')}
				slices={earning}
				total={money(income)}
				format={money}
			/>
		</section>
	</div>

	<section class="card" aria-labelledby="by-month">
		<h2 id="by-month">{t('The year to {month}', { month: monthLabel(month) })}</h2>
		<YearBars
			months={year.months.map((m) => ({
				label: short(m.month),
				income: m.income_cents,
				expense: m.expense_cents
			}))}
			format={money}
		/>
		<details>
			<summary>{t('Show the figures as a table')}</summary>
			<table>
				<thead
					><tr
						><th scope="col">{t('Month')}</th><th scope="col" class="num">{t('In')}</th><th
							scope="col"
							class="num">{t('Out')}</th
						><th scope="col" class="num">{t('Balance')}</th></tr
					></thead
				>
				<tbody>
					{#each year.months as m (m.month)}
						<tr>
							<td>{monthLabel(m.month)}</td>
							<td class="num">{money(m.income_cents)}</td>
							<td class="num">{money(m.expense_cents)}</td>
							<td class="num" class:bad={m.income_cents < m.expense_cents}
								>{money(m.income_cents - m.expense_cents)}</td
							>
						</tr>
					{/each}
				</tbody>
			</table>
		</details>
	</section>
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
	.pies {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 24rem), 1fr));
		gap: var(--space-4);
	}
	details {
		margin-top: var(--space-3);
	}
	table {
		width: 100%;
		border-collapse: collapse;
	}
	.num {
		text-align: right;
	}
</style>
