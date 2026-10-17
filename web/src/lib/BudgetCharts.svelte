<script lang="ts">
	import { api, message, type Schemas } from '$lib/api';
	import { colourOrder, slices } from '$lib/budgetCharts';
	import { monthLabel } from '$lib/calendar';
	import Donut from '$lib/charts/Donut.svelte';
	import YearBars from '$lib/charts/YearBars.svelte';
	import { formatDate } from '$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import { formatCents } from '$lib/money';
	import Status from '$lib/ui/Status.svelte';

	// The month's money as two pies, where it went and where it came from, and
	// its year, January to December, as income and expense bars (EP-138,
	// EP-147). Shown on the Budget page itself and on Reports.
	let { month, months = 1 }: { month: string; months?: number } = $props();

	type Report = Schemas['BudgetReport'];
	let period = $state<Report | null>(null);
	let year = $state<Report | null>(null);
	let error = $state('');
	const money = (c: number) => formatCents(c, period?.currency ?? 'EUR');

	$effect(() => {
		const [m, n] = [month, months];
		if (!/^\d{4}-\d{2}$/.test(m)) return;
		Promise.all([
			api<Report>(`/v1/budget/report?months=${n}&month=${m}`),
			api<Report>(`/v1/budget/report?months=12&month=${m.slice(0, 4)}-12`)
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
	const label = $derived(
		months === 1
			? monthLabel(month)
			: period
				? t('{from} to {to}', { from: period.from, to: period.to })
				: ''
	);
</script>

{#if error}
	<Status kind="error" message={error} />
{:else if !period || !year}
	<Status kind="loading" message={t('Working it out…')} />
{:else}
	<div class="pies">
		<section class="card" aria-label={t('Where the money went')}>
			<p class="muted small">{label}</p>
			<Donut
				title={t('Where the money went')}
				slices={spending}
				total={money(outgo)}
				format={money}
			/>
		</section>
		<section class="card" aria-label={t('Where it came from')}>
			<p class="muted small">{label}</p>
			<Donut
				title={t('Where it came from')}
				slices={earning}
				total={money(income)}
				format={money}
			/>
		</section>
	</div>

	<section class="card" aria-labelledby="by-month">
		<h2 id="by-month">{t('Income and expenses in {year}', { year: month.slice(0, 4) })}</h2>
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
	.pies {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 24rem), 1fr));
		gap: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.card {
		margin-bottom: var(--space-4);
	}
	.pies .card {
		margin-bottom: 0;
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
	.small {
		font-size: var(--text-sm);
		margin: 0;
	}
</style>
