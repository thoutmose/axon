<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { api, message, type Schemas } from '$lib/api';
	import { monthLabel } from '$lib/calendar';
	import { formatCents } from '$lib/money';
	import Status from '$lib/ui/Status.svelte';

	type Report = Schemas['BudgetReport'];
	let months = $state(12);
	let report = $state<Report | null>(null);
	let error = $state('');
	const money = (c: number) => formatCents(c, report?.currency ?? 'EUR');
	const peak = $derived(
		Math.max(1, ...(report?.months ?? []).flatMap((m) => [m.income_cents, m.expense_cents]))
	);
	const expenses = $derived(report?.categories.filter((c) => c.kind === 'expense') ?? []);
	const spent = $derived(expenses.reduce((n, c) => n + c.cents, 0));
	const income = $derived(report?.months.reduce((n, m) => n + m.income_cents, 0) ?? 0);
	const outgo = $derived(report?.months.reduce((n, m) => n + m.expense_cents, 0) ?? 0);

	$effect(() => {
		const n = months;
		api<Report>(`/v1/budget/report?months=${n}`)
			.then((r) => (report = r))
			.catch((err) => (error = message(err)));
	});
</script>

<svelte:head><title>{t('Reports · Budget · Axon')}</title></svelte:head>

<div class="row">
	<h1>{t('Reports')}</h1>
	<label class="row small"
		>{t('Period')}
		<select bind:value={months}>
			<option value={3}>{t('3 months')}</option>
			<option value={6}>{t('6 months')}</option>
			<option value={12}>{t('12 months')}</option>
			<option value={24}>{t('24 months')}</option>
		</select></label
	>
</div>

{#if error}
	<Status kind="error" message={error} />
{:else if !report}
	<Status kind="loading" message={t('Working it out…')} />
{:else}
	<section class="card" aria-labelledby="totals">
		<h2 id="totals">
			{t('{from} to {to}: in {value}, out {value2}, saved', {
				from: report.from,
				to: report.to,
				value: money(income),
				value2: money(outgo)
			})}
			<span class:bad={income - outgo < 0} class:ok={income - outgo > 0}
				>{money(income - outgo)}</span
			>{income ? ` (${Math.round(((income - outgo) / income) * 100)} %)` : ''}
		</h2>
		<p class="small">{t('Net worth today:')} <strong>{money(report.net_worth_cents)}</strong></p>
	</section>

	<section class="card" aria-labelledby="by-month">
		<h2 id="by-month">{t('Income and spending by month')}</h2>
		<table>
			<thead
				><tr
					><th scope="col">{t('Month')}</th><th scope="col">{t('Income and spending')}</th><th
						scope="col"
						class="num">{t('In')}</th
					><th scope="col" class="num">{t('Out')}</th></tr
				></thead
			>
			<tbody>
				{#each report.months as m (m.month)}
					<tr>
						<td>{monthLabel(m.month)}</td>
						<td class="bars" aria-hidden="true">
							<span class="bar in" style:width="{(m.income_cents / peak) * 100}%"></span>
							<span class="bar out" style:width="{(m.expense_cents / peak) * 100}%"></span>
						</td>
						<td class="num ok">{money(m.income_cents)}</td>
						<td class="num">{money(m.expense_cents)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</section>

	<section class="card" aria-labelledby="by-category">
		<h2 id="by-category">{t('Where the money went')}</h2>
		{#if expenses.length === 0}
			<p class="muted">{t('No spending in this period.')}</p>
		{:else}
			<table>
				<tbody>
					{#each expenses as c (c.category_id ?? 'none')}
						<tr>
							<td>{c.name}</td>
							<td class="bars" aria-hidden="true"
								><span class="bar out" style:width="{(c.cents / spent) * 100}%"></span></td
							>
							<td class="num">{money(c.cents)}</td>
							<td class="num muted">{Math.round((c.cents / spent) * 100)} %</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</section>
{/if}

<style>
	.bars {
		width: 50%;
	}
	.bar {
		display: block;
		height: 0.5rem;
		border-radius: 999px;
		margin: 2px 0;
	}
	.bar.in {
		background: var(--ok);
	}
	.bar.out {
		background: var(--chart-2);
	}
</style>
