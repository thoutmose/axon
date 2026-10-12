<script lang="ts">
	import { localDate } from '$lib/dates';
	import { page } from '$app/state';
	import { api, message, type Schemas } from '$lib/api';
	import { monthLabel, shiftMonth } from '$lib/calendar';
	import { parseTags } from '$lib/finance';
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
		categories.find((c) => c.id === id)?.name ?? 'Uncategorised';
	const expenses = $derived(summary?.lines.filter((l) => l.kind === 'expense') ?? []);
	const incomes = $derived(summary?.lines.filter((l) => l.kind === 'income') ?? []);
	const net = $derived(summary ? summary.income_cents - summary.expense_cents : 0);

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
			toast('Type an amount like 12.50.', 'error');
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
			toast('Recorded.');
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
			toast('Type a budget like 400 or 400.00.', 'error');
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
	const starters: [string, Kind, number | null][] = [
		['Salary', 'income', null],
		['Other income', 'income', null],
		['Rent or mortgage', 'expense', null],
		['Groceries', 'expense', null],
		['Utilities', 'expense', null],
		['Transport', 'expense', null],
		['Eating out', 'expense', null],
		['Health', 'expense', null],
		['Leisure', 'expense', null],
		['Savings', 'expense', null]
	];
	async function addStarters() {
		busy = true;
		try {
			for (const [name, kind, monthly_cents] of starters) {
				if (categories.some((c) => c.name === name && c.kind === kind)) continue;
				await api('/v1/budget/categories', { method: 'POST', body: { name, kind, monthly_cents } });
			}
			await load();
			toast('Added. Set a monthly budget on each expense category.');
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
			toast('Type a budget like 400 or 400.00.', 'error');
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

<svelte:head><title>Budget · Axon</title></svelte:head>

<div class="head">
	<h1>Budget</h1>
	<nav aria-label="Month" class="months">
		<a href="?month={shiftMonth(month, -1)}" aria-label="Previous month">←</a>
		<h2 aria-live="polite">{monthLabel(month)}</h2>
		<a href="?month={shiftMonth(month, 1)}" aria-label="Next month">→</a>
		{#if month !== today.slice(0, 7)}<a href="?" class="small">This month</a>{/if}
	</nav>
	{#if summary}
		<label class="currency"
			><span class="muted small">Currency</span>
			<select value={summary.currency} onchange={setCurrency}>
				{#each currencies.includes(summary.currency) ? currencies : [summary.currency, ...currencies] as c (c)}<option
						value={c}>{c}</option
					>{/each}
			</select></label
		>
	{/if}
</div>
<p class="muted hint">
	Private to you: never searched, never shown to agents, included in your export, erased with your
	account. Amounts are exact to the cent.
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !summary}
	<Status kind="loading" message="Loading the budget…" />
{:else}
	<section class="kpis" aria-label="This month">
		<div><span class="muted small">Income</span><strong>{money(summary.income_cents)}</strong></div>
		<div><span class="muted small">Spent</span><strong>{money(summary.expense_cents)}</strong></div>
		<div>
			<span class="muted small">Left over</span><strong class:bad={net < 0} class:ok={net > 0}
				>{money(net)}</strong
			>
		</div>
		<div>
			<span class="muted small">Budgeted</span><strong>{money(summary.budgeted_cents)}</strong>
			{#if summary.budgeted_cents}<span class="muted small"
					>{share(summary.expense_cents, summary.budgeted_cents)} % used</span
				>{/if}
		</div>
	</section>

	<div class="layout">
		<div class="col">
			<section class="card" aria-labelledby="spending">
				<h2 id="spending">Spending by category</h2>
				{#if categories.length === 0}
					<p class="muted">No categories yet.</p>
					<Button variant="secondary" {busy} onclick={addStarters}
						>Start with common categories</Button
					>
				{:else}
					<table>
						<thead
							><tr
								><th scope="col">Category</th><th scope="col" class="num">Spent</th><th
									scope="col"
									class="num">Budget</th
								><th scope="col" class="num">Left</th><th scope="col"
									><span class="visually-hidden">Edit</span></th
								></tr
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
												aria-label="{l.name}: {share(l.actual_cents, l.monthly_cents)} % of budget"
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
										><button class="link" onclick={() => openEdit(l)} aria-label="Edit {l.name}"
											>Edit</button
										></td
									>
								</tr>
							{/each}
							{#if summary.uncategorised_expense_cents}
								<tr
									><td>Uncategorised</td><td class="num"
										>{money(summary.uncategorised_expense_cents)}</td
									><td class="num">—</td><td class="num">—</td><td></td></tr
								>
							{/if}
						</tbody>
					</table>
					{#if incomes.length || summary.uncategorised_income_cents}
						<h3>Income</h3>
						<table>
							<tbody>
								{#each incomes as l (l.id)}
									<tr
										><td>{l.name}</td><td class="num">{money(l.actual_cents)}</td><td
											><button class="link" onclick={() => openEdit(l)} aria-label="Edit {l.name}"
												>Edit</button
											></td
										></tr
									>
								{/each}
								{#if summary.uncategorised_income_cents}<tr
										><td>Uncategorised</td><td class="num"
											>{money(summary.uncategorised_income_cents)}</td
										><td></td></tr
									>{/if}
							</tbody>
						</table>
					{/if}
				{/if}
				<form class="inline" onsubmit={addCategory}>
					<label for="cat-name" class="visually-hidden">New category</label>
					<input
						id="cat-name"
						placeholder="New category…"
						bind:value={newCategory.name}
						maxlength="100"
						required
					/>
					<label for="cat-kind" class="visually-hidden">Kind</label>
					<select id="cat-kind" bind:value={newCategory.kind}>
						<option value="expense">Expense</option>
						<option value="income">Income</option>
					</select>
					{#if newCategory.kind === 'expense'}
						<label for="cat-budget" class="visually-hidden">Monthly budget (optional)</label>
						<input
							id="cat-budget"
							placeholder="Monthly budget"
							inputmode="decimal"
							bind:value={newCategory.budget}
						/>
					{/if}
					<Button type="submit" variant="secondary">Add</Button>
				</form>
			</section>
		</div>

		<div class="col">
			<section class="card" aria-labelledby="record">
				<h2 id="record">Record money in or out</h2>
				<form class="grid" onsubmit={addTransaction}>
					<fieldset class="kind">
						<legend class="visually-hidden">Kind</legend>
						<label><input type="radio" value="expense" bind:group={form.kind} /> Expense</label>
						<label><input type="radio" value="income" bind:group={form.kind} /> Income</label>
						<label><input type="radio" value="transfer" bind:group={form.kind} /> Transfer</label>
					</fieldset>
					<div class="field">
						<label for="tx-amount">Amount</label>
						<input
							id="tx-amount"
							inputmode="decimal"
							placeholder="12.50"
							bind:value={form.amount}
							required
						/>
					</div>
					<div class="field">
						<label for="tx-date">Date</label>
						<input id="tx-date" type="date" bind:value={form.occurred_on} required />
					</div>
					{#if openAccounts.length || form.kind === 'transfer'}
						<div class="field">
							<label for="tx-account"
								>{form.kind === 'income' ? 'Into account' : 'From account'}</label
							>
							<select
								id="tx-account"
								bind:value={form.account_id}
								required={form.kind === 'transfer'}
							>
								<option value="">{form.kind === 'transfer' ? 'Choose…' : 'No account'}</option>
								{#each openAccounts as a (a.id)}<option value={a.id}>{a.name}</option>{/each}
							</select>
						</div>
					{/if}
					{#if form.kind === 'transfer'}
						<div class="field">
							<label for="tx-to-account">To account</label>
							<select id="tx-to-account" bind:value={form.to_account_id} required>
								<option value="">Choose…</option>
								{#each openAccounts.filter((a) => a.id !== form.account_id) as a (a.id)}<option
										value={a.id}>{a.name}</option
									>{/each}
							</select>
						</div>
					{:else}
						<div class="field">
							<label for="tx-category">Category</label>
							<select id="tx-category" bind:value={form.category_id}>
								<option value="">Uncategorised (or by your rules)</option>
								{#each formCategories as c (c.id)}<option value={c.id}>{c.name}</option>{/each}
							</select>
						</div>
					{/if}
					<div class="field">
						<label for="tx-tags">Tags</label>
						<input id="tx-tags" bind:value={form.tags} placeholder="#holiday #work" />
					</div>
					<div class="field">
						<label for="tx-payee">Payee</label>
						<input id="tx-payee" bind:value={form.payee} maxlength="200" />
					</div>
					<div class="field wide">
						<label for="tx-note">Note</label>
						<input id="tx-note" bind:value={form.note} maxlength="2000" />
					</div>
					<div class="wide"><Button type="submit" {busy}>Record</Button></div>
				</form>
			</section>

			<section class="card" aria-labelledby="transactions">
				<h2 id="transactions">Transactions</h2>
				{#if transactions.length === 0}
					<p class="muted">Nothing recorded in {monthLabel(month)}.</p>
				{:else}
					<table>
						<thead
							><tr
								><th scope="col">Date</th><th scope="col">Payee</th><th scope="col">Category</th><th
									scope="col"
									class="num">Amount</th
								><th scope="col"><span class="visually-hidden">Delete</span></th></tr
							></thead
						>
						<tbody>
							{#each transactions as t (t.id)}
								<tr>
									<td>{t.occurred_on.slice(5)}</td>
									<td
										>{t.payee || '—'}{#if t.note}<span class="muted small">
												· {t.note}</span
											>{/if}</td
									>
									<td
										>{t.kind === 'transfer'
											? `${accountName(t.account_id)} → ${accountName(t.to_account_id)}`
											: nameOf(t.category_id)}{#if t.tags.length}<span class="muted small">
												{t.tags.map((x) => `#${x}`).join(' ')}</span
											>{/if}</td
									>
									<td class="num" class:ok={t.kind === 'income'}
										>{t.kind === 'income' ? '+' : t.kind === 'transfer' ? '⇄' : '−'}{money(
											t.amount_cents
										)}</td
									>
									<td
										><button
											class="link"
											onclick={() => removeTransaction(t)}
											aria-label="Delete {t.payee || 'transaction'} of {t.occurred_on}"
											>Delete</button
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

<Dialog bind:open={editOpen} title="Edit category">
	<form class="grid" onsubmit={submitEdit}>
		<div class="field wide">
			<label for="edit-cat-name">Name</label>
			<input id="edit-cat-name" bind:value={edit.name} maxlength="100" required />
		</div>
		{#if editing?.kind === 'expense'}
			<div class="field wide">
				<label for="edit-cat-budget">Monthly budget (empty for none)</label>
				<input id="edit-cat-budget" inputmode="decimal" bind:value={edit.budget} />
			</div>
		{/if}
		<div class="wide row">
			<Button type="submit">Save</Button>
			<Button variant="secondary" onclick={() => saveCategory({ archived: !editing?.archived })}
				>{editing?.archived ? 'Restore' : 'Archive'}</Button
			>
			<Button variant="danger" onclick={deleteCategory}>Delete</Button>
		</div>
		<p class="muted small wide">
			Deleting a category keeps its transactions, uncategorised. Archiving hides it from new
			transactions.
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
		grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
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
	.ok {
		color: var(--ok);
	}
	.bad {
		color: var(--bad);
	}
	.layout {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(26rem, 1fr));
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
		grid-template-columns: repeat(auto-fit, minmax(11rem, 1fr));
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
