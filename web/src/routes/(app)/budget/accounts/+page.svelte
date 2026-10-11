<script lang="ts">
	import { api, message } from '$lib/api';
	import { roleNames, type Account } from '$lib/finance';
	import { formatCents, toCents } from '$lib/money';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	let accounts = $state<Account[] | null>(null);
	let currency = $state('EUR');
	let error = $state('');
	let form = $state({ name: '', type: 'asset', role: 'checking', opening: '', owed: false });
	const money = (c: number) => formatCents(c, currency);
	const total = $derived((accounts ?? []).reduce((n, a) => n + a.balance_cents, 0));

	async function load() {
		try {
			const [a, m] = await Promise.all([
				api<{ items: Account[] }>('/v1/budget/accounts'),
				api<{ currency: string }>('/v1/budget')
			]);
			accounts = a.items;
			currency = m.currency;
		} catch (err) {
			error = message(err);
		}
	}
	load();

	async function create(e: SubmitEvent) {
		e.preventDefault();
		const cents = form.opening ? toCents(form.opening) : 0;
		if (cents === null) {
			toast('Type an opening balance like 1200.50.', 'error');
			return;
		}
		try {
			await api('/v1/budget/accounts', {
				method: 'POST',
				body: {
					name: form.name,
					type: form.type,
					role: form.role,
					opening_cents: form.type === 'liability' ? -cents : cents
				}
			});
			form = { name: '', type: 'asset', role: 'checking', opening: '', owed: false };
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function patch(a: Account, body: Record<string, unknown>) {
		try {
			await api(`/v1/budget/accounts/${a.id}`, { method: 'PATCH', body });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function remove(a: Account) {
		try {
			await api(`/v1/budget/accounts/${a.id}`, { method: 'DELETE' });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<svelte:head><title>Accounts · Budget · Axon</title></svelte:head>

<h1>Accounts</h1>
<p class="muted small">
	Where your money is, and what you owe. A balance is the opening balance plus income, minus
	expenses, plus or minus transfers; transfers between your accounts never count as spending.
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !accounts}
	<Status kind="loading" message="Loading accounts…" />
{:else}
	<section class="card" aria-labelledby="list">
		<h2 id="list">Net worth: <span class:bad={total < 0}>{money(total)}</span></h2>
		{#if accounts.length === 0}
			<p class="muted">No accounts yet. Add your bank account, cash, and cards below.</p>
		{:else}
			<table>
				<thead
					><tr
						><th scope="col">Account</th><th scope="col">Kind</th><th scope="col" class="num"
							>Balance</th
						><th scope="col"><span class="visually-hidden">Actions</span></th></tr
					></thead
				>
				<tbody>
					{#each accounts as a (a.id)}
						<tr class:muted={a.archived}>
							<td>{a.name}{a.archived ? ' (archived)' : ''}</td>
							<td>{a.type === 'liability' ? 'Owed · ' : ''}{roleNames[a.role]}</td>
							<td class="num" class:bad={a.balance_cents < 0}>{money(a.balance_cents)}</td>
							<td class="row">
								<button class="link" onclick={() => patch(a, { archived: !a.archived })}
									>{a.archived ? 'Restore' : 'Archive'}</button
								>
								<button class="link" onclick={() => remove(a)} aria-label="Delete {a.name}"
									>Delete</button
								>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</section>

	<section class="card" aria-labelledby="new">
		<h2 id="new">Add an account</h2>
		<form class="grid" onsubmit={create}>
			<div class="field">
				<label for="acc-name">Name</label>
				<input id="acc-name" bind:value={form.name} maxlength="100" required />
			</div>
			<div class="field">
				<label for="acc-type">It is</label>
				<select
					id="acc-type"
					bind:value={form.type}
					onchange={() => (form.role = form.type === 'liability' ? 'credit_card' : 'checking')}
				>
					<option value="asset">Money I have</option>
					<option value="liability">Money I owe</option>
				</select>
			</div>
			<div class="field">
				<label for="acc-role">Type</label>
				<select id="acc-role" bind:value={form.role}>
					{#each Object.entries(roleNames) as [k, v] (k)}<option value={k}>{v}</option>{/each}
				</select>
			</div>
			<div class="field">
				<label for="acc-opening"
					>{form.type === 'liability' ? 'Amount owed today' : 'Balance today'}</label
				>
				<input id="acc-opening" inputmode="decimal" placeholder="0.00" bind:value={form.opening} />
			</div>
			<div><Button type="submit">Add account</Button></div>
		</form>
	</section>
{/if}
