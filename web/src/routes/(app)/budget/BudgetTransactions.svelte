<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { api, message, type Schemas } from '$lib/api';
	import { monthLabel } from '$lib/calendar';
	import { toast } from '$lib/ui/toast.svelte';

	let {
		transactions,
		categories,
		accounts,
		month,
		money,
		onchange
	}: {
		transactions: Schemas['BudgetTransaction'][];
		categories: Schemas['BudgetCategory'][];
		accounts: Schemas['BudgetAccount'][];
		month: string;
		money: (c: number) => string;
		onchange: () => Promise<void>;
	} = $props();

	const accountName = (id: string | null) => accounts.find((a) => a.id === id)?.name ?? '';
	const nameOf = (id: string | null) =>
		categories.find((c) => c.id === id)?.name ?? t('Uncategorised');

	async function removeTransaction(t: Schemas['BudgetTransaction']) {
		try {
			await api(`/v1/budget/transactions/${t.id}`, { method: 'DELETE' });
			await onchange();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<section class="card" aria-labelledby="transactions">
	<h2 id="transactions">{t('Transactions')}</h2>
	{#if transactions.length === 0}
		<p class="muted">{t('Nothing recorded in {value}.', { value: monthLabel(month) })}</p>
	{:else}
		<table>
			<thead
				><tr
					><th scope="col">{t('Date')}</th><th scope="col">{t('Payee')}</th><th scope="col"
						>{t('Category')}</th
					><th scope="col" class="num">{t('Amount')}</th><th scope="col"
						><span class="visually-hidden">{t('Delete')}</span></th
					></tr
				></thead
			>
			<tbody>
				{#each transactions as txn (txn.id)}
					<tr>
						<td>{txn.occurred_on.slice(5)}</td>
						<td
							>{txn.payee || '—'}{#if txn.note}<span class="muted small">
									· {txn.note}</span
								>{/if}</td
						>
						<td
							>{txn.kind === 'transfer'
								? `${accountName(txn.account_id)} → ${accountName(txn.to_account_id)}`
								: nameOf(txn.category_id)}{#if txn.tags.length}{' '}<span class="muted small">
									{txn.tags.map((x) => `#${x}`).join(' ')}</span
								>{/if}</td
						>
						<td class="num" class:ok={txn.kind === 'income'}
							>{txn.kind === 'income' ? '+' : txn.kind === 'transfer' ? '⇄' : '−'}{money(
								txn.amount_cents
							)}</td
						>
						<td
							><button
								class="link"
								onclick={() => removeTransaction(txn)}
								aria-label={txn.payee
									? t('Delete {payee} of {date}', { payee: txn.payee, date: txn.occurred_on })
									: t('Delete the transaction of {date}', { date: txn.occurred_on })}
								>{t('Delete')}</button
							></td
						>
					</tr>
				{/each}
			</tbody>
		</table>
	{/if}
</section>

<style>
	.small {
		font-size: var(--text-sm);
	}
	.ok {
		color: var(--ok);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6) var(--space-6);
		overflow-x: auto;
	}
	.card h2 {
		font-size: var(--text-lg);
	}
	table {
		width: 100%;
		border-collapse: collapse;
		margin-bottom: var(--space-4);
	}
	th {
		text-align: left;
		font-size: var(--text-sm);
		color: var(--muted);
		font-weight: 600;
	}
	th,
	td {
		padding: var(--space-2);
		border-bottom: 1px solid var(--border);
		vertical-align: middle;
	}
	.num {
		text-align: right;
		font-variant-numeric: tabular-nums;
		white-space: nowrap;
	}
	.link {
		font: inherit;
		background: none;
		border: none;
		padding: 0;
		color: var(--accent);
		cursor: pointer;
	}
</style>
