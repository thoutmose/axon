<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { localDate } from '$lib/dates';
	import { api, message, type Schemas } from '$lib/api';
	import { parseTags } from '$lib/finance';
	import { toCents } from '$lib/money';
	import Button from '$lib/ui/Button.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	let {
		categories,
		accounts,
		onchange
	}: {
		categories: Schemas['BudgetCategory'][];
		accounts: Schemas['BudgetAccount'][];
		onchange: () => Promise<void>;
	} = $props();

	let busy = $state(false);
	let form = $state({
		kind: 'expense' as 'income' | 'expense' | 'transfer',
		occurred_on: localDate(new Date()),
		amount: '',
		category_id: '',
		account_id: '',
		to_account_id: '',
		tags: '',
		payee: '',
		note: ''
	});
	const openAccounts = $derived(accounts.filter((a) => !a.archived));
	const formCategories = $derived(categories.filter((c) => c.kind === form.kind && !c.archived));

	async function addTransaction(e: SubmitEvent) {
		e.preventDefault();
		const cents = toCents(form.amount);
		if (!cents) {
			toast(t('Type an amount like 12.50.'), 'error');
			return;
		}
		busy = true;
		try {
			await api('/v1/budget/transactions', {
				method: 'POST',
				body: {
					kind: form.kind,
					occurred_on: form.occurred_on,
					amount_cents: cents,
					category_id: form.kind === 'transfer' ? null : form.category_id || null,
					account_id: form.account_id || null,
					to_account_id: form.kind === 'transfer' ? form.to_account_id || null : null,
					tags: parseTags(form.tags),
					payee: form.payee,
					note: form.note
				}
			});
			form = { ...form, amount: '', payee: '', note: '', tags: '' };
			toast(t('Recorded.'));
			await onchange();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
</script>

<section class="card" aria-labelledby="record">
	<h2 id="record">{t('Record money in or out')}</h2>
	<form class="grid" onsubmit={addTransaction}>
		<fieldset class="kind">
			<legend class="visually-hidden">{t('Kind')}</legend>
			<label><input type="radio" value="expense" bind:group={form.kind} /> {t('Expense')}</label>
			<label><input type="radio" value="income" bind:group={form.kind} /> {t('Income')}</label>
			<label><input type="radio" value="transfer" bind:group={form.kind} /> {t('Transfer')}</label>
		</fieldset>
		<div class="field">
			<label for="tx-amount">{t('Amount')}</label>
			<input
				id="tx-amount"
				inputmode="decimal"
				placeholder="12.50"
				bind:value={form.amount}
				required
			/>
		</div>
		<div class="field">
			<label for="tx-date">{t('Date')}</label>
			<input id="tx-date" type="date" bind:value={form.occurred_on} required />
		</div>
		{#if openAccounts.length || form.kind === 'transfer'}
			<div class="field">
				<label for="tx-account"
					>{form.kind === 'income' ? t('Into account') : t('From account')}</label
				>
				<select id="tx-account" bind:value={form.account_id} required={form.kind === 'transfer'}>
					<option value="">{form.kind === 'transfer' ? t('Choose…') : t('No account')}</option>
					{#each openAccounts as a (a.id)}<option value={a.id}>{a.name}</option>{/each}
				</select>
			</div>
		{/if}
		{#if form.kind === 'transfer'}
			<div class="field">
				<label for="tx-to-account">{t('To account')}</label>
				<select id="tx-to-account" bind:value={form.to_account_id} required>
					<option value="">{t('Choose…')}</option>
					{#each openAccounts.filter((a) => a.id !== form.account_id) as a (a.id)}<option
							value={a.id}>{a.name}</option
						>{/each}
				</select>
			</div>
		{:else}
			<div class="field">
				<label for="tx-category">{t('Category')}</label>
				<select id="tx-category" bind:value={form.category_id}>
					<option value="">{t('By your rules')}</option>
					{#each formCategories as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
				</select>
			</div>
		{/if}
		<div class="field">
			<label for="tx-tags">{t('Tags')}</label>
			<input id="tx-tags" bind:value={form.tags} placeholder={t('#holiday #work')} />
		</div>
		<div class="field">
			<label for="tx-payee">{t('Payee')}</label>
			<input id="tx-payee" bind:value={form.payee} maxlength="200" />
		</div>
		<div class="field wide">
			<label for="tx-note">{t('Note')}</label>
			<input id="tx-note" bind:value={form.note} maxlength="2000" />
		</div>
		<div class="wide"><Button type="submit" {busy}>{t('Record')}</Button></div>
	</form>
</section>

<style>
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
	form.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 11rem), 1fr));
		gap: var(--space-3);
	}
	.field {
		display: grid;
		gap: var(--space-1);
	}
	.field label {
		font-weight: 600;
		font-size: var(--text-sm);
	}
	form.grid label {
		display: grid;
		gap: var(--space-1);
		font-weight: 600;
		font-size: var(--text-sm);
	}
	.wide {
		grid-column: 1 / -1;
	}
	.kind {
		grid-column: 1 / -1;
		border: none;
		padding: 0;
		margin: 0;
		display: flex;
		gap: var(--space-4);
	}
	.kind label {
		display: flex !important;
		align-items: center;
		gap: var(--space-2);
		font-weight: 400 !important;
	}
	input:not([type='radio']),
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
