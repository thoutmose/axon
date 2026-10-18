<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { api, message } from '$lib/api';
	import { localDate } from '$lib/dates';
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
			toast(t('Type an opening balance like 1200.50.'), 'error');
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

	// Move money from one account to another (owner request): a transfer,
	// which changes both balances and never counts as spending.
	let move = $state({ from: '', to: '', amount: '', on: localDate(new Date()), note: '' });
	const open = $derived((accounts ?? []).filter((a) => !a.archived));
	async function transfer(e: SubmitEvent) {
		e.preventDefault();
		const cents = toCents(move.amount);
		if (!cents || cents <= 0) {
			toast(t('Type an amount like 150.00.'), 'error');
			return;
		}
		try {
			await api('/v1/budget/transactions', {
				method: 'POST',
				body: {
					kind: 'transfer',
					amount_cents: cents,
					account_id: move.from,
					to_account_id: move.to,
					occurred_on: move.on,
					note: move.note
				}
			});
			toast(t('Transferred {amount}.', { amount: money(cents) }));
			move = { ...move, amount: '', note: '' };
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

<svelte:head><title>{t('Accounts · Budget · Axon')}</title></svelte:head>

<h1>{t('Accounts')}</h1>
<p class="muted small">
	{t(
		'Where your money is, and what you owe. A balance is the opening balance plus income, minus expenses, plus or minus transfers; transfers between your accounts never count as spending.'
	)}
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !accounts}
	<Status kind="loading" message={t('Loading accounts…')} />
{:else}
	<section class="card" aria-labelledby="list">
		<h2 id="list">{t('Net worth:')} <span class:bad={total < 0}>{money(total)}</span></h2>
		{#if accounts.length === 0}
			<p class="muted">{t('No accounts yet. Add your bank account, cash, and cards below.')}</p>
		{:else}
			<table>
				<thead
					><tr
						><th scope="col">{t('Account')}</th><th scope="col">{t('Kind')}</th><th
							scope="col"
							class="num">{t('Balance')}</th
						><th scope="col"><span class="visually-hidden">{t('Actions')}</span></th></tr
					></thead
				>
				<tbody>
					{#each accounts as a (a.id)}
						<tr class:muted={a.archived}>
							<td>{a.name}{a.archived ? ' (archived)' : ''}</td>
							<td>{a.type === 'liability' ? t('Owed · ') : ''}{t(roleNames[a.role])}</td>
							<td class="num" class:bad={a.balance_cents < 0}>{money(a.balance_cents)}</td>
							<td class="row">
								<button class="link" onclick={() => patch(a, { archived: !a.archived })}
									>{a.archived ? t('Restore') : t('Archive')}</button
								>
								<button
									class="link"
									onclick={() => remove(a)}
									aria-label={t('Delete {name}', { name: a.name })}>{t('Delete')}</button
								>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	</section>

	{#if open.length >= 2}
		<section class="card" aria-labelledby="move">
			<h2 id="move">{t('Transfer between accounts')}</h2>
			<form class="grid" onsubmit={transfer}>
				<div class="field">
					<label for="mv-from">{t('From account')}</label>
					<select id="mv-from" bind:value={move.from} required>
						<option value="" disabled>{t('Choose…')}</option>
						{#each open as a (a.id)}<option value={a.id}>{a.name} · {money(a.balance_cents)}</option
							>{/each}
					</select>
				</div>
				<div class="field">
					<label for="mv-to">{t('To account')}</label>
					<select id="mv-to" bind:value={move.to} required>
						<option value="" disabled>{t('Choose…')}</option>
						{#each open.filter((a) => a.id !== move.from) as a (a.id)}<option value={a.id}
								>{a.name} · {money(a.balance_cents)}</option
							>{/each}
					</select>
				</div>
				<div class="field">
					<label for="mv-amount">{t('Amount')}</label>
					<input
						id="mv-amount"
						inputmode="decimal"
						placeholder="0.00"
						bind:value={move.amount}
						required
					/>
				</div>
				<div class="field">
					<label for="mv-on">{t('Date')}</label>
					<input id="mv-on" type="date" bind:value={move.on} required />
				</div>
				<div class="field">
					<label for="mv-note">{t('Note')}</label>
					<input id="mv-note" bind:value={move.note} maxlength="500" />
				</div>
				<div>
					<Button type="submit" disabled={!move.from || !move.to || move.from === move.to}
						>{t('Transfer')}</Button
					>
				</div>
			</form>
		</section>
	{/if}

	<section class="card" aria-labelledby="new">
		<h2 id="new">{t('Add an account')}</h2>
		<form class="grid" onsubmit={create}>
			<div class="field">
				<label for="acc-name">{t('Name')}</label>
				<input id="acc-name" bind:value={form.name} maxlength="100" required />
			</div>
			<div class="field">
				<label for="acc-type">{t('It is')}</label>
				<select
					id="acc-type"
					bind:value={form.type}
					onchange={() => (form.role = form.type === 'liability' ? 'credit_card' : 'checking')}
				>
					<option value="asset">{t('Money I have')}</option>
					<option value="liability">{t('Money I owe')}</option>
				</select>
			</div>
			<div class="field">
				<label for="acc-role">{t('Type')}</label>
				<select id="acc-role" bind:value={form.role}>
					{#each Object.entries(roleNames) as [k, v] (k)}<option value={k}>{t(v)}</option>{/each}
				</select>
			</div>
			<div class="field">
				<label for="acc-opening"
					>{form.type === 'liability' ? t('Amount owed today') : t('Balance today')}</label
				>
				<input id="acc-opening" inputmode="decimal" placeholder="0.00" bind:value={form.opening} />
			</div>
			<div><Button type="submit">{t('Add account')}</Button></div>
		</form>
	</section>
{/if}
