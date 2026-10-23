<script lang="ts">
	import BudgetCharts from '$lib/BudgetCharts.svelte';
	import { t } from '$lib/i18n.svelte';
	import { localDate } from '$lib/dates';
	import { page } from '$app/state';
	import { api, message, type Schemas } from '$lib/api';
	import { monthLabel, shiftMonth } from '$lib/calendar';
	import { formatCents } from '$lib/money';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';
	import BudgetCategories from './BudgetCategories.svelte';
	import BudgetOverview from './BudgetOverview.svelte';
	import BudgetRecord from './BudgetRecord.svelte';
	import BudgetTransactions from './BudgetTransactions.svelte';

	type Month = Schemas['BudgetMonth'];
	type Category = Schemas['BudgetCategory'];
	type Transaction = Schemas['BudgetTransaction'];
	type Account = Schemas['BudgetAccount'];

	const today = localDate(new Date());
	const month = $derived(page.url.searchParams.get('month') ?? today.slice(0, 7));
	let summary = $state<Month | null>(null);
	let categories = $state<Category[]>([]);
	let transactions = $state<Transaction[]>([]);
	let accounts = $state<Account[]>([]);
	let error = $state('');

	const money = (c: number) => formatCents(c, summary?.currency ?? 'EUR');

	async function load() {
		error = '';
		try {
			const [m, c, t, a] = await Promise.all([
				api<Month>(`/v1/budget?month=${month}`),
				api<{ items: Category[] }>('/v1/budget/categories'),
				api<{ items: Transaction[] }>(`/v1/budget/transactions?month=${month}`),
				api<{ items: Account[] }>('/v1/budget/accounts')
			]);
			summary = m;
			categories = c.items;
			transactions = t.items;
			accounts = a.items;
		} catch (err) {
			error = message(err);
		}
	}
	$effect(() => {
		void month;
		load();
	});

	async function setCurrency(e: Event) {
		try {
			await api('/v1/budget/currency', {
				method: 'PUT',
				body: { currency: (e.currentTarget as HTMLSelectElement).value }
			});
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	const currencies = [
		'EUR',
		'USD',
		'GBP',
		'CHF',
		'CAD',
		'AUD',
		'JPY',
		'SEK',
		'NOK',
		'DKK',
		'PLN',
		'MAD',
		'XOF'
	];
</script>

<svelte:head><title>{t('Budget · Axon')}</title></svelte:head>

<div class="head">
	<h1>{t('Budget')}</h1>
	<nav aria-label={t('Month')} class="months">
		<a href="?month={shiftMonth(month, -1)}" aria-label={t('Previous month')}>←</a>
		<h2 aria-live="polite">{monthLabel(month)}</h2>
		<a href="?month={shiftMonth(month, 1)}" aria-label={t('Next month')}>→</a>
		{#if month !== today.slice(0, 7)}<a href="?" class="small">{t('This month')}</a>{/if}
	</nav>
	{#if summary}
		<label class="currency"
			><span class="muted small">{t('Currency')}</span>
			<select value={summary.currency} onchange={setCurrency}>
				{#each currencies.includes(summary.currency) ? currencies : [summary.currency, ...currencies] as c (c)}<option
						value={c}>{c}</option
					>{/each}
			</select></label
		>
	{/if}
</div>
<p class="muted hint">
	{t(
		'Private to you: never searched, never shown to agents, included in your export, erased with your account. Amounts are exact to the cent.'
	)}
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !summary}
	<Status kind="loading" message={t('Loading the budget…')} />
{:else}
	<BudgetOverview {summary} {month} {money} />

	<BudgetCharts {month} />

	<div class="layout">
		<div class="col">
			<BudgetCategories {summary} {categories} {money} onchange={load} />
		</div>

		<div class="col">
			<BudgetRecord {categories} {accounts} onchange={load} />
			<BudgetTransactions {transactions} {categories} {accounts} {month} {money} onchange={load} />
		</div>
	</div>
{/if}

<style>
	.head {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: var(--space-4);
		flex-wrap: wrap;
	}
	.months {
		display: flex;
		align-items: center;
		gap: var(--space-3);
	}
	.months h2 {
		margin: 0;
		font-size: var(--text-lg);
		min-width: 10rem;
		text-align: center;
	}
	.months a {
		text-decoration: none;
	}
	.currency {
		display: flex;
		align-items: center;
		gap: var(--space-2);
	}
	.hint,
	.small {
		font-size: var(--text-sm);
	}
	.layout {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 26rem), 1fr));
		gap: var(--space-4);
		align-items: start;
	}
	@media (max-width: 30rem) {
		.layout {
			grid-template-columns: 1fr;
		}
	}
	.col {
		display: grid;
		gap: var(--space-4);
		min-width: 0;
	}
	select {
		font: inherit;
		min-height: 2.5rem;
		padding: var(--space-1) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		min-width: 0;
	}
</style>
