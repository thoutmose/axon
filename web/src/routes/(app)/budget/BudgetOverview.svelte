<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { localDate } from '$lib/dates';
	import { monthLabel } from '$lib/calendar';
	import { roleNames } from '$lib/finance';
	import type { Schemas } from '$lib/api';

	let {
		summary,
		month,
		money
	}: { summary: Schemas['BudgetMonth']; month: string; money: (c: number) => string } = $props();

	const today = localDate(new Date());
	const net = $derived(summary.income_cents - summary.expense_cents);
	const current = $derived(month === today.slice(0, 7));
	// What the bills and income still to come will do before the month ends (EP-175).
	const coming = $derived(summary.upcoming_income_cents - summary.upcoming_expense_cents);
	const billsDue = $derived(summary.upcoming.filter((u) => u.kind === 'expense').length);
	const incomesDue = $derived(summary.upcoming.length - billsDue);
	const dueText = $derived(
		[
			billsDue === 1 ? t('1 bill') : billsDue ? t('{n} bills', { n: billsDue }) : '',
			incomesDue === 1 ? t('1 income') : incomesDue ? t('{n} incomes', { n: incomesDue }) : ''
		]
			.filter(Boolean)
			.join(', ')
	);
	// How much of the month has gone, to read the share of the budget used against it.
	const monthGone = $derived.by(() => {
		if (!current) return month < today.slice(0, 7) ? 100 : 0;
		const [y, m] = month.split('-').map(Number);
		return Math.round((Number(today.slice(8)) / new Date(y, m, 0).getDate()) * 100);
	});
	const share = (actual: number, budget: number | null) =>
		budget ? Math.min(100, Math.round((actual / budget) * 100)) : 0;
</script>

<section class="kpis" aria-label={t('This month')}>
	<div>
		<span class="muted small">{t('Income')}</span><strong>{money(summary.income_cents)}</strong>
	</div>
	<div>
		<span class="muted small">{t('Spent')}</span><strong>{money(summary.expense_cents)}</strong>
	</div>
	<div>
		<span class="muted small">{t('Left over')}</span><strong class:bad={net < 0} class:ok={net > 0}
			>{money(net)}</strong
		>
	</div>
	<div>
		<span class="muted small">{t('Budgeted')}</span><strong>{money(summary.budgeted_cents)}</strong>
		{#if summary.budgeted_cents}<span class="muted small"
				>{t('{value} % used', {
					value: share(summary.expense_cents, summary.budgeted_cents)
				})} · {t('{value} % of the month gone', { value: monthGone })}</span
			>{/if}
	</div>
	{#if current && summary.upcoming.length}
		<div>
			<span class="muted small">{t('Still to come')}</span><strong
				class:bad={coming < 0}
				class:ok={coming > 0}>{money(coming)}</strong
			>
			<span class="muted small">{t('{what} before the month ends', { what: dueText })}</span>
		</div>
	{/if}
</section>

<section class="accounts" aria-label={t('Your accounts')}>
	<h2 class="small muted">
		{current
			? t('Your accounts today')
			: t('Your accounts at the end of {month}', { month: monthLabel(month) })}
	</h2>
	{#if summary.accounts.length === 0}
		<p class="muted small">
			<a href="/budget/accounts">{t('Add your accounts to see each balance here.')}</a>
		</p>
	{:else}
		<div class="tiles">
			<a href="/budget/accounts" class="tile">
				<span class="muted small">{t('Net worth')}</span>
				<strong class:bad={summary.net_worth_cents < 0}>{money(summary.net_worth_cents)}</strong>
			</a>
			{#each summary.accounts as a (a.id)}
				<a href="/budget/accounts" class="tile">
					<span class="name">{a.name}</span>
					<span class="muted small"
						>{a.type === 'liability' ? t('Owed · ') : ''}{t(roleNames[a.role])}</span
					>
					<strong class:bad={a.balance_cents < 0}>{money(a.balance_cents)}</strong>
				</a>
			{/each}
		</div>
	{/if}
</section>

<style>
	.small {
		font-size: var(--text-sm);
	}
	.kpis {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 12rem), 1fr));
		gap: var(--space-3);
		margin: var(--space-4) 0;
	}
	.kpis div {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		display: grid;
		gap: var(--space-1);
	}
	.kpis strong {
		font-size: var(--text-2xl);
	}
	.accounts {
		margin: var(--space-4) 0;
	}
	.accounts h2 {
		margin: 0 0 var(--space-2);
		font-weight: 500;
	}
	.tiles {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 11rem), 1fr));
		gap: var(--space-3);
	}
	.tile {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-3) var(--space-4);
		display: grid;
		gap: var(--space-1);
		color: inherit;
		text-decoration: none;
	}
	.tile:hover,
	.tile:focus-visible {
		border-color: var(--accent);
	}
	.tile .name {
		font-weight: 500;
	}
	.tile strong {
		font-size: var(--text-xl);
	}
	.ok {
		color: var(--ok);
	}
	.bad {
		color: var(--bad);
	}
</style>
