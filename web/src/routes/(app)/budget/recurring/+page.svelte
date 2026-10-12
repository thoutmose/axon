<script lang="ts">
	import { localDate } from '$lib/dates';
	import { api, message } from '$lib/api';
	import {
		frequencies,
		loadBasics,
		type Account,
		type Category,
		type Recurring
	} from '$lib/finance';
	import { formatCents, toCents } from '$lib/money';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	let items = $state<Recurring[] | null>(null);
	let accounts = $state<Account[]>([]);
	let categories = $state<Category[]>([]);
	let currency = $state('EUR');
	let error = $state('');
	const today = localDate(new Date());
	let form = $state({
		name: '',
		kind: 'expense',
		amount: '',
		rule: 'FREQ=MONTHLY',
		start_on: today,
		account_id: '',
		to_account_id: '',
		category_id: '',
		payee: ''
	});
	const money = (c: number) => formatCents(c, currency);
	const every = (rule: string) => frequencies.find((f) => f.rule === rule)?.label ?? rule;
	const name = (list: { id: string; name: string }[], id: string | null) =>
		list.find((x) => x.id === id)?.name ?? '';

	async function load() {
		try {
			const [r, basics] = await Promise.all([
				api<{ items: Recurring[] }>('/v1/budget/recurring'),
				loadBasics()
			]);
			items = r.items;
			({ accounts, categories, currency } = basics);
		} catch (err) {
			error = message(err);
		}
	}
	load();

	async function create(e: SubmitEvent) {
		e.preventDefault();
		const cents = toCents(form.amount);
		if (!cents) {
			toast('Type an amount like 850.00.', 'error');
			return;
		}
		try {
			await api('/v1/budget/recurring', {
				method: 'POST',
				body: {
					name: form.name,
					kind: form.kind,
					amount_cents: cents,
					rule: form.rule,
					start_on: form.start_on,
					payee: form.payee,
					account_id: form.account_id || null,
					to_account_id: form.kind === 'transfer' ? form.to_account_id || null : null,
					category_id: form.kind === 'transfer' ? null : form.category_id || null
				}
			});
			const run = await api<{ created: number }>('/v1/budget/recurring/run', { method: 'POST' });
			toast(
				run.created
					? `Added, and ${run.created} past occurrence${run.created === 1 ? '' : 's'} recorded.`
					: 'Added.'
			);
			form = { ...form, name: '', amount: '', payee: '' };
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function toggle(r: Recurring) {
		try {
			await api(`/v1/budget/recurring/${r.id}`, { method: 'PATCH', body: { active: !r.active } });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function remove(r: Recurring) {
		try {
			await api(`/v1/budget/recurring/${r.id}`, { method: 'DELETE' });
			toast('Stopped. Transactions already recorded stay.');
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<svelte:head><title>Recurring · Budget · Axon</title></svelte:head>

<h1>Recurring transactions</h1>
<p class="muted small">
	Rent, salary, subscriptions: each occurrence is recorded on its day when you open the budget,
	exactly once, even if you were away for months.
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !items}
	<Status kind="loading" message="Loading…" />
{:else}
	<section class="card" aria-labelledby="list">
		<h2 id="list">Series</h2>
		{#if items.length === 0}
			<p class="muted">None yet.</p>
		{:else}
			<table>
				<thead
					><tr
						><th scope="col">Name</th><th scope="col">Repeats</th><th scope="col">Next</th><th
							scope="col"
							class="num">Amount</th
						><th scope="col"><span class="visually-hidden">Actions</span></th></tr
					></thead
				>
				<tbody>
					{#each items as r (r.id)}
						<tr class:muted={!r.active}>
							<td
								>{r.name}<span class="muted small">
									· {r.kind === 'transfer'
										? `${name(accounts, r.account_id)} → ${name(accounts, r.to_account_id)}`
										: name(categories, r.category_id) || 'Uncategorised'}</span
								></td
							>
							<td>{every(r.rule)}</td>
							<td>{r.next_on ?? 'Ended'}{r.active ? '' : ' (paused)'}</td>
							<td class="num" class:ok={r.kind === 'income'}
								>{r.kind === 'income' ? '+' : r.kind === 'transfer' ? '⇄' : '−'}{money(
									r.amount_cents
								)}</td
							>
							<td class="row"
								><button class="link" onclick={() => toggle(r)}
									>{r.active ? 'Pause' : 'Resume'}</button
								><button class="link" onclick={() => remove(r)} aria-label="Delete {r.name}"
									>Delete</button
								></td
							>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</section>

	<section class="card" aria-labelledby="new">
		<h2 id="new">Add a recurring transaction</h2>
		<form class="grid" onsubmit={create}>
			<div class="field">
				<label for="rec-name">Name</label>
				<input id="rec-name" bind:value={form.name} maxlength="100" required placeholder="Rent" />
			</div>
			<div class="field">
				<label for="rec-kind">Kind</label>
				<select id="rec-kind" bind:value={form.kind}>
					<option value="expense">Expense</option>
					<option value="income">Income</option>
					<option value="transfer">Transfer</option>
				</select>
			</div>
			<div class="field">
				<label for="rec-amount">Amount</label>
				<input id="rec-amount" inputmode="decimal" bind:value={form.amount} required />
			</div>
			<div class="field">
				<label for="rec-rule">Repeats</label>
				<select id="rec-rule" bind:value={form.rule}>
					{#each frequencies as f (f.rule)}<option value={f.rule}>{f.label}</option>{/each}
				</select>
			</div>
			<div class="field">
				<label for="rec-start">First on</label>
				<input id="rec-start" type="date" bind:value={form.start_on} required />
			</div>
			<div class="field">
				<label for="rec-account">{form.kind === 'income' ? 'Into account' : 'From account'}</label>
				<select id="rec-account" bind:value={form.account_id} required={form.kind === 'transfer'}>
					<option value="">{form.kind === 'transfer' ? 'Choose…' : 'No account'}</option>
					{#each accounts.filter((a) => !a.archived) as a (a.id)}<option value={a.id}
							>{a.name}</option
						>{/each}
				</select>
			</div>
			{#if form.kind === 'transfer'}
				<div class="field">
					<label for="rec-to">To account</label>
					<select id="rec-to" bind:value={form.to_account_id} required>
						<option value="">Choose…</option>
						{#each accounts.filter((a) => !a.archived && a.id !== form.account_id) as a (a.id)}<option
								value={a.id}>{a.name}</option
							>{/each}
					</select>
				</div>
			{:else}
				<div class="field">
					<label for="rec-category">Category</label>
					<select id="rec-category" bind:value={form.category_id}>
						<option value="">Uncategorised</option>
						{#each categories.filter((c) => c.kind === form.kind && !c.archived) as c (c.id)}<option
								value={c.id}>{c.name}</option
							>{/each}
					</select>
				</div>
			{/if}
			<div class="field">
				<label for="rec-payee">Payee</label>
				<input id="rec-payee" bind:value={form.payee} maxlength="200" />
			</div>
			<div><Button type="submit">Add</Button></div>
		</form>
	</section>
{/if}
