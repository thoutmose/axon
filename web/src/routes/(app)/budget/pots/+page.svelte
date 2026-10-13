<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { api, message, type Schemas } from '$lib/api';
	import { formatCents, toCents } from '$lib/money';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Pot = Schemas['SavingsPot'];

	let pots = $state<Pot[] | null>(null);
	let currency = $state('EUR');
	let error = $state('');
	let form = $state({ name: '', target: '', date: '' });
	let amounts = $state<Record<string, string>>({});
	const money = (c: number) => formatCents(c, currency);
	const share = (p: Pot) => Math.min(100, Math.round((p.saved_cents / p.target_cents) * 100));

	async function load() {
		try {
			const [p, m] = await Promise.all([
				api<{ items: Pot[] }>('/v1/budget/pots'),
				api<{ currency: string }>('/v1/budget')
			]);
			pots = p.items;
			currency = m.currency;
		} catch (err) {
			error = message(err);
		}
	}
	load();

	async function create(e: SubmitEvent) {
		e.preventDefault();
		const cents = toCents(form.target);
		if (!cents) {
			toast(t('Type a target like 1200 or 1200.50.'), 'error');
			return;
		}
		try {
			await api('/v1/budget/pots', {
				method: 'POST',
				body: { name: form.name, target_cents: cents, target_date: form.date }
			});
			form = { name: '', target: '', date: '' };
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function move(p: Pot, sign: 1 | -1) {
		const cents = toCents(amounts[p.id] ?? '');
		if (!cents) {
			toast(t('Type an amount like 50 or 49.90.'), 'error');
			return;
		}
		try {
			await api(`/v1/budget/pots/${p.id}/moves`, {
				method: 'POST',
				body: { amount_cents: sign * cents }
			});
			amounts[p.id] = '';
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function patch(p: Pot, body: Record<string, unknown>) {
		try {
			await api(`/v1/budget/pots/${p.id}`, { method: 'PATCH', body });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function remove(p: Pot) {
		if (!confirm(t('Delete “{name}” and its history?', { name: p.name }))) return;
		try {
			await api(`/v1/budget/pots/${p.id}`, { method: 'DELETE' });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<svelte:head><title>{t('Savings pots · Budget · Axon')}</title></svelte:head>

<h1>{t('Savings pots')}</h1>
<p class="muted">
	{t('A pot (a')} <i lang="fr">{t('tirelire')}</i>{t(
		') is money you put aside for something to buy later. Give it a target and, if you like, a date: Axon says how much to put in each month.'
	)}
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !pots}
	<Status kind="loading" message={t('Loading your pots…')} />
{:else}
	{#if pots.length === 0}
		<p class="muted">{t('No pots yet. Start one below.')}</p>
	{/if}
	<ul class="pots">
		{#each pots as p (p.id)}
			<li class="card" class:muted={p.archived} aria-labelledby="pot-{p.id}">
				<h2 id="pot-{p.id}">{p.name}{p.archived ? ' (archived)' : ''}</h2>
				<p>
					<strong>{money(p.saved_cents)}</strong>
					{p.target_date
						? t('of {target} by {date}', { target: money(p.target_cents), date: p.target_date })
						: t('of {target}', { target: money(p.target_cents) })}
				</p>
				<span
					class="bar"
					role="progressbar"
					aria-label={t('{name}: {value} % saved', { name: p.name, value: share(p) })}
					aria-valuemin="0"
					aria-valuemax="100"
					aria-valuenow={share(p)}><span style:width="{share(p)}%"></span></span
				>
				{#if p.saved_cents >= p.target_cents}
					<p class="ok">{t('Target reached.')}</p>
				{:else if p.monthly_cents}
					<p class="small">
						{t('Put in {value} a month to get there on time.', { value: money(p.monthly_cents) })}
					</p>
				{/if}
				{#if !p.archived}
					<div class="row">
						<label class="visually-hidden" for="amount-{p.id}"
							>{t('Amount for {name}', { name: p.name })}</label
						>
						<input
							id="amount-{p.id}"
							inputmode="decimal"
							placeholder="0.00"
							bind:value={amounts[p.id]}
						/>
						<Button
							onclick={() => move(p, 1)}
							aria-label={t('Put money in {name}', { name: p.name })}>{t('Put in')}</Button
						>
						<Button
							variant="secondary"
							onclick={() => move(p, -1)}
							aria-label={t('Take money out of {name}', { name: p.name })}>{t('Take out')}</Button
						>
					</div>
				{/if}
				{#if p.moves.length}
					<details>
						<summary>{t('History ({length})', { length: p.moves.length })}</summary>
						<ul class="small">
							{#each p.moves as m (m.id)}
								<li>
									{`${m.occurred_on} · ${m.amount_cents > 0 ? '+' : '−'}${money(Math.abs(m.amount_cents))}${m.note ? ` · ${m.note}` : ''}`}
								</li>
							{/each}
						</ul>
					</details>
				{/if}
				<div class="row small">
					<button class="link" onclick={() => patch(p, { archived: !p.archived })}
						>{p.archived ? t('Restore') : t('Archive')}</button
					>
					<button
						class="link"
						onclick={() => remove(p)}
						aria-label={t('Delete {name}', { name: p.name })}>{t('Delete')}</button
					>
				</div>
			</li>
		{/each}
	</ul>

	<section class="card" aria-labelledby="new-pot">
		<h2 id="new-pot">{t('Start a pot')}</h2>
		<form class="grid" onsubmit={create}>
			<div class="field">
				<label for="pot-name">{t('What it is for')}</label>
				<input id="pot-name" bind:value={form.name} maxlength="100" required />
			</div>
			<div class="field">
				<label for="pot-target">{t('Target')}</label>
				<input
					id="pot-target"
					inputmode="decimal"
					placeholder="0.00"
					bind:value={form.target}
					required
				/>
			</div>
			<div class="field">
				<label for="pot-date">{t('By (optional)')}</label>
				<input id="pot-date" type="date" bind:value={form.date} />
			</div>
			<div><Button type="submit">{t('Start the pot')}</Button></div>
		</form>
	</section>
{/if}

<style>
	.pots {
		list-style: none;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 20rem), 1fr));
		gap: var(--space-4);
	}
	.pots h2 {
		margin-top: 0;
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
		margin-top: var(--space-2);
	}
	.row input {
		flex: 1 1 6rem;
	}
	.bar {
		display: block;
		height: 0.5rem;
		background: var(--accent-soft);
		border-radius: 999px;
		overflow: hidden;
	}
	.bar span {
		display: block;
		height: 100%;
		background: var(--accent);
	}
	.ok {
		color: var(--ok);
		font-weight: 600;
	}
</style>
