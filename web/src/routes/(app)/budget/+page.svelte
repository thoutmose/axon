<script lang="ts">
	import BudgetCharts from '$lib/BudgetCharts.svelte';
	import { t } from '$lib/i18n.svelte';
	import { localDate } from '$lib/dates';
	import { page } from '$app/state';
	import { api, message, type Schemas } from '$lib/api';
	import { monthLabel, shiftMonth } from '$lib/calendar';
	import { parseTags, roleNames } from '$lib/finance';
	import { centsText, formatCents, toCents } from '$lib/money';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Month = Schemas['BudgetMonth'];
	type Category = Schemas['BudgetCategory'];
	type Transaction = Schemas['BudgetTransaction'];
	type Kind = 'income' | 'expense';
	type Account = Schemas['BudgetAccount'];

	const today = localDate(new Date());
	const month = $derived(page.url.searchParams.get('month') ?? today.slice(0, 7));
	let summary = $state<Month | null>(null);
	let categories = $state<Category[]>([]);
	let transactions = $state<Transaction[]>([]);
	let accounts = $state<Account[]>([]);
	const accountName = (id: string | null) => accounts.find((a) => a.id === id)?.name ?? '';
	let error = $state('');
	let busy = $state(false);

	const money = (c: number) => formatCents(c, summary?.currency ?? 'EUR');
	const nameOf = (id: string | null) =>
		categories.find((c) => c.id === id)?.name ?? t('Uncategorised');
	const expenses = $derived(summary?.lines.filter((l) => l.kind === 'expense') ?? []);
	const incomes = $derived(summary?.lines.filter((l) => l.kind === 'income') ?? []);
	const net = $derived(summary ? summary.income_cents - summary.expense_cents : 0);
	const current = $derived(month === today.slice(0, 7));
	// What the bills and income still to come will do before the month ends (EP-175).
	const coming = $derived(
		summary ? summary.upcoming_income_cents - summary.upcoming_expense_cents : 0
	);
	const billsDue = $derived(summary?.upcoming.filter((u) => u.kind === 'expense').length ?? 0);
	const incomesDue = $derived((summary?.upcoming.length ?? 0) - billsDue);
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

	// --- Transactions ---
	let form = $state({
		kind: 'expense' as Kind | 'transfer',
		occurred_on: today,
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
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function removeTransaction(t: Transaction) {
		try {
			await api(`/v1/budget/transactions/${t.id}`, { method: 'DELETE' });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	// --- Categories ---
	let newCategory = $state({ name: '', kind: 'expense' as Kind, budget: '' });
	let editing = $state<Category | null>(null);
	let editOpen = $state(false);
	let edit = $state({ name: '', budget: '' });

	async function addCategory(e: SubmitEvent) {
		e.preventDefault();
		const cents = newCategory.budget ? toCents(newCategory.budget) : null;
		if (newCategory.budget && cents === null) {
			toast(t('Type a budget like 400 or 400.00.'), 'error');
			return;
		}
		try {
			await api('/v1/budget/categories', {
				method: 'POST',
				body: { name: newCategory.name, kind: newCategory.kind, monthly_cents: cents }
			});
			newCategory = { name: '', kind: newCategory.kind, budget: '' };
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	// Common categories to start from; existing names are skipped.
	const starters: [string, Kind, number | null][] = $derived([
		[t('Salary'), 'income', null],
		[t('Other income'), 'income', null],
		[t('Rent or mortgage'), 'expense', null],
		[t('Groceries'), 'expense', null],
		[t('Utilities'), 'expense', null],
		[t('Transport'), 'expense', null],
		[t('Eating out'), 'expense', null],
		[t('Health'), 'expense', null],
		[t('Leisure'), 'expense', null],
		[t('Savings'), 'expense', null]
	]);
	async function addStarters() {
		busy = true;
		try {
			for (const [name, kind, monthly_cents] of starters) {
				if (categories.some((c) => c.name === name && c.kind === kind)) continue;
				await api('/v1/budget/categories', { method: 'POST', body: { name, kind, monthly_cents } });
			}
			await load();
			toast(t('Added. Set a monthly budget on each expense category.'));
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	function openEdit(c: Category) {
		editing = c;
		edit = { name: c.name, budget: c.monthly_cents === null ? '' : centsText(c.monthly_cents) };
		editOpen = true;
	}

	async function saveCategory(body: Record<string, unknown>) {
		if (!editing) return;
		try {
			await api(`/v1/budget/categories/${editing.id}`, { method: 'PATCH', body });
			editOpen = false;
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	function submitEdit(e: SubmitEvent) {
		e.preventDefault();
		const cents = edit.budget ? toCents(edit.budget) : null;
		if (edit.budget && cents === null) {
			toast(t('Type a budget like 400 or 400.00.'), 'error');
			return;
		}
		saveCategory({ name: edit.name, monthly_cents: cents });
	}

	async function deleteCategory() {
		if (!editing) return;
		try {
			await api(`/v1/budget/categories/${editing.id}`, { method: 'DELETE' });
			editOpen = false;
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

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

	const share = (actual: number, budget: number | null) =>
		budget ? Math.min(100, Math.round((actual / budget) * 100)) : 0;
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
	<section class="kpis" aria-label={t('This month')}>
		<div>
			<span class="muted small">{t('Income')}</span><strong>{money(summary.income_cents)}</strong>
		</div>
		<div>
			<span class="muted small">{t('Spent')}</span><strong>{money(summary.expense_cents)}</strong>
		</div>
		<div>
			<span class="muted small">{t('Left over')}</span><strong
				class:bad={net < 0}
				class:ok={net > 0}>{money(net)}</strong
			>
		</div>
		<div>
			<span class="muted small">{t('Budgeted')}</span><strong
				>{money(summary.budgeted_cents)}</strong
			>
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

	<BudgetCharts {month} />

	<div class="layout">
		<div class="col">
			<section class="card" aria-labelledby="spending">
				<h2 id="spending">{t('Spending by category')}</h2>
				{#if categories.length === 0}
					<p class="muted">{t('No categories yet.')}</p>
					<Button variant="secondary" {busy} onclick={addStarters}
						>{t('Start with common categories')}</Button
					>
				{:else}
					<table>
						<thead
							><tr
								><th scope="col">{t('Category')}</th><th scope="col" class="num">{t('Spent')}</th
								><th scope="col" class="num">{t('Budget')}</th><th scope="col" class="num"
									>{t('Left')}</th
								><th scope="col"><span class="visually-hidden">{t('Edit')}</span></th></tr
							></thead
						>
						<tbody>
							{#each expenses as l (l.id)}
								<tr class:archived={l.archived}>
									<td>
										{l.name}
										{#if l.monthly_cents}
											<span
												class="bar"
												class:over={l.actual_cents > l.monthly_cents}
												role="progressbar"
												aria-label={t('{name}: {value} % of budget', {
													name: l.name,
													value: share(l.actual_cents, l.monthly_cents)
												})}
												aria-valuemin="0"
												aria-valuemax="100"
												aria-valuenow={share(l.actual_cents, l.monthly_cents)}
												><span style:width="{share(l.actual_cents, l.monthly_cents)}%"></span></span
											>
										{/if}
									</td>
									<td class="num">{money(l.actual_cents)}</td>
									<td class="num">{l.monthly_cents === null ? '—' : money(l.monthly_cents)}</td>
									<td
										class="num"
										class:bad={l.monthly_cents !== null && l.actual_cents > l.monthly_cents}
										>{l.monthly_cents === null ? '—' : money(l.monthly_cents - l.actual_cents)}</td
									>
									<td
										><button
											class="link"
											onclick={() => openEdit(l)}
											aria-label={t('Edit {name}', { name: l.name })}>{t('Edit')}</button
										></td
									>
								</tr>
							{/each}
							{#if summary.uncategorised_expense_cents}
								<tr
									><td>{t('Uncategorised')}</td><td class="num"
										>{money(summary.uncategorised_expense_cents)}</td
									><td class="num">—</td><td class="num">—</td><td></td></tr
								>
							{/if}
						</tbody>
					</table>
					{#if incomes.length || summary.uncategorised_income_cents}
						<h3>{t('Income')}</h3>
						<table>
							<tbody>
								{#each incomes as l (l.id)}
									<tr
										><td>{l.name}</td><td class="num">{money(l.actual_cents)}</td><td
											><button
												class="link"
												onclick={() => openEdit(l)}
												aria-label={t('Edit {name}', { name: l.name })}>{t('Edit')}</button
											></td
										></tr
									>
								{/each}
								{#if summary.uncategorised_income_cents}<tr
										><td>{t('Uncategorised')}</td><td class="num"
											>{money(summary.uncategorised_income_cents)}</td
										><td></td></tr
									>{/if}
							</tbody>
						</table>
					{/if}
				{/if}
				<form class="inline" onsubmit={addCategory}>
					<label for="cat-name" class="visually-hidden">{t('New category')}</label>
					<input
						id="cat-name"
						placeholder={t('New category…')}
						bind:value={newCategory.name}
						maxlength="100"
						required
					/>
					<label for="cat-kind" class="visually-hidden">{t('Kind')}</label>
					<select id="cat-kind" bind:value={newCategory.kind}>
						<option value="expense">{t('Expense')}</option>
						<option value="income">{t('Income')}</option>
					</select>
					{#if newCategory.kind === 'expense'}
						<label for="cat-budget" class="visually-hidden">{t('Monthly budget (optional)')}</label>
						<input
							id="cat-budget"
							placeholder={t('Monthly budget')}
							inputmode="decimal"
							bind:value={newCategory.budget}
						/>
					{/if}
					<Button type="submit" variant="secondary">{t('Add')}</Button>
				</form>
			</section>
		</div>

		<div class="col">
			<section class="card" aria-labelledby="record">
				<h2 id="record">{t('Record money in or out')}</h2>
				<form class="grid" onsubmit={addTransaction}>
					<fieldset class="kind">
						<legend class="visually-hidden">{t('Kind')}</legend>
						<label
							><input type="radio" value="expense" bind:group={form.kind} /> {t('Expense')}</label
						>
						<label><input type="radio" value="income" bind:group={form.kind} /> {t('Income')}</label
						>
						<label
							><input type="radio" value="transfer" bind:group={form.kind} /> {t('Transfer')}</label
						>
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
							<select
								id="tx-account"
								bind:value={form.account_id}
								required={form.kind === 'transfer'}
							>
								<option value="">{form.kind === 'transfer' ? t('Choose…') : t('No account')}</option
								>
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
		</div>
	</div>
{/if}

<Dialog bind:open={editOpen} title={t('Edit category')}>
	<form class="grid" onsubmit={submitEdit}>
		<div class="field wide">
			<label for="edit-cat-name">{t('Name')}</label>
			<input id="edit-cat-name" bind:value={edit.name} maxlength="100" required />
		</div>
		{#if editing?.kind === 'expense'}
			<div class="field wide">
				<label for="edit-cat-budget">{t('Monthly budget (empty for none)')}</label>
				<input id="edit-cat-budget" inputmode="decimal" bind:value={edit.budget} />
			</div>
		{/if}
		<div class="wide row">
			<Button type="submit">{t('Save')}</Button>
			<Button variant="secondary" onclick={() => saveCategory({ archived: !editing?.archived })}
				>{editing?.archived ? t('Restore') : t('Archive')}</Button
			>
			<Button variant="danger" onclick={deleteCategory}>{t('Delete')}</Button>
		</div>
		<p class="muted small wide">
			{t(
				'Deleting a category keeps its transactions, uncategorised. Archiving hides it from new transactions.'
			)}
		</p>
	</form>
</Dialog>

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
	.archived {
		opacity: 0.6;
	}
	.bar {
		display: block;
		height: 0.35rem;
		margin-top: var(--space-1);
		background: var(--accent-soft);
		border-radius: 999px;
		overflow: hidden;
	}
	.bar span {
		display: block;
		height: 100%;
		background: var(--accent);
	}
	.bar.over span {
		background: var(--bad);
	}
	.link {
		font: inherit;
		background: none;
		border: none;
		padding: 0;
		color: var(--accent);
		cursor: pointer;
	}
	form.inline {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
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
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}
</style>
