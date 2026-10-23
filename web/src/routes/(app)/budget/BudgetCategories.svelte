<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { api, message, type Schemas } from '$lib/api';
	import { centsText, toCents } from '$lib/money';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Category = Schemas['BudgetCategory'];
	type Kind = 'income' | 'expense';

	let {
		summary,
		categories,
		money,
		onchange
	}: {
		summary: Schemas['BudgetMonth'];
		categories: Category[];
		money: (c: number) => string;
		onchange: () => Promise<void>;
	} = $props();

	let busy = $state(false);
	const expenses = $derived(summary.lines.filter((l) => l.kind === 'expense'));
	const incomes = $derived(summary.lines.filter((l) => l.kind === 'income'));
	const share = (actual: number, budget: number | null) =>
		budget ? Math.min(100, Math.round((actual / budget) * 100)) : 0;

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
			await onchange();
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
			await onchange();
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
			await onchange();
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
			await onchange();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<section class="card" aria-labelledby="spending">
	<h2 id="spending">{t('Spending by category')}</h2>
	{#if categories.length === 0}
		<p class="muted">{t('No categories yet.')}</p>
		<Button variant="secondary" {busy} onclick={addStarters}
			>{t('Start with common categories')}</Button
		>
	{:else}
		<table>
			<thead>
				<tr>
					<th scope="col">{t('Category')}</th>
					<th scope="col" class="num">{t('Spent')}</th>
					<th scope="col" class="num">{t('Budget')}</th>
					<th scope="col" class="num">{t('Left')}</th>
					<th scope="col"><span class="visually-hidden">{t('Edit')}</span></th>
				</tr>
			</thead>
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
						<td class="num" class:bad={l.monthly_cents !== null && l.actual_cents > l.monthly_cents}
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
					<tr>
						<td>{t('Uncategorised')}</td>
						<td class="num">{money(summary.uncategorised_expense_cents)}</td>
						<td class="num">—</td>
						<td class="num">—</td>
						<td></td>
					</tr>
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
	.small {
		font-size: var(--text-sm);
	}
	.bad {
		color: var(--bad);
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
