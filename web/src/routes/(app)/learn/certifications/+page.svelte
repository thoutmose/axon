<script lang="ts">
	import { api, message, type Schemas } from '$lib/api';
	import { costs, daysBetween, expiringSoon, nextExam } from '$lib/certs';
	import { localDate } from '$lib/dates';
	import { i18n, t } from '$lib/i18n.svelte';
	import LearnTabs from '$lib/LearnTabs.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// Certifications (EP-104): the exams planned, booked, passed, or failed,
	// what they cost, and when they expire. Exams and expiries are on the calendar.
	type Cert = Schemas['Certification'];
	let items = $state<Cert[] | null>(null);
	let error = $state('');
	const today = localDate(new Date());

	async function load() {
		error = '';
		try {
			items = (await api<{ items: Cert[] }>('/v1/certifications')).items;
		} catch (err) {
			error = message(err);
		}
	}
	load();

	const statusNames = $derived({
		planned: t('Planned'),
		booked: t('Booked'),
		passed: t('Passed'),
		failed: t('Failed')
	} as Record<string, string>);
	const money = (cents: number, currency: string) =>
		(cents / 100).toLocaleString(i18n.locale, {
			style: 'currency',
			currency,
			maximumFractionDigits: 0
		});
	const when = (date: string) => {
		const n = daysBetween(today, date);
		return n === 0 ? t('today') : n > 0 ? t('in {n} days', { n }) : t('{n} days ago', { n: -n });
	};
	const fields = (c: Cert) => {
		const { id: _id, created_at: _c, updated_at: _u, ...rest } = c;
		return rest;
	};

	let draft = $state({
		name: '',
		code: '',
		provider: '',
		url: '',
		target_on: '',
		cost: '',
		currency: 'USD',
		validity: ''
	});
	async function add(e: SubmitEvent) {
		e.preventDefault();
		try {
			await api('/v1/certifications', {
				method: 'POST',
				body: {
					name: draft.name,
					code: draft.code,
					provider: draft.provider,
					url: draft.url || null,
					status: 'planned',
					target_on: draft.target_on || null,
					cost_cents: draft.cost ? Math.round(Number(draft.cost.replace(',', '.')) * 100) : null,
					currency: draft.currency,
					validity_months: draft.validity ? Number(draft.validity) : null
				}
			});
			draft = {
				...draft,
				name: '',
				code: '',
				provider: '',
				url: '',
				target_on: '',
				cost: '',
				validity: ''
			};
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	// The update form of one certification: booking, result, and credential.
	let editing = $state<string | null>(null);
	let upd = $state({ status: 'planned', exam_on: '', score: '', credential_url: '' });
	function open(c: Cert) {
		editing = c.id;
		upd = {
			status: c.status,
			exam_on: c.exam_on ?? '',
			score: c.score,
			credential_url: c.credential_url ?? ''
		};
	}
	async function save(c: Cert) {
		try {
			await api(`/v1/certifications/${c.id}`, {
				method: 'PUT',
				body: {
					...fields(c),
					status: upd.status,
					exam_on: upd.exam_on || null,
					passed_on: upd.status === 'passed' ? (c.passed_on ?? upd.exam_on ?? null) : null,
					expires_on: upd.status === 'passed' && c.status === 'passed' ? c.expires_on : null,
					score: upd.score,
					credential_url: upd.credential_url || null
				}
			});
			editing = null;
			toast(t('Saved.'));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	async function remove(c: Cert) {
		if (!confirm(t('Remove “{name}”?', { name: c.name }))) return;
		try {
			await api(`/v1/certifications/${c.id}`, { method: 'DELETE' });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	const next = $derived(items ? nextExam(items, today) : null);
	const soon = $derived(items ? expiringSoon(items, today) : []);
	const passed = $derived((items ?? []).filter((c) => c.status === 'passed').length);
</script>

<svelte:head><title>{t('Certifications')} · Axon</title></svelte:head>

<h1>{t('Certifications')}</h1>
<LearnTabs />
<p class="muted hint">
	{t(
		'The exams you plan, book, and pass, what they cost, and when they expire. Booked exams and expiries are on the calendar.'
	)}
</p>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !items}
	<Status kind="loading" message={t('Loading…')} />
{:else}
	<section class="card summary" aria-labelledby="sum-title">
		<h2 id="sum-title" class="visually-hidden">{t('Summary')}</h2>
		<p class="kpis">
			<span><strong>{passed}</strong> {t('passed of {n}', { n: items.length })}</span>
			{#if next}
				<span
					>{t('Next exam: {name}, {when}', {
						name: next.code || next.name,
						when: when(next.exam_on ?? today)
					})}</span
				>
			{/if}
			{#each costs(items) as row (row.currency)}
				<span
					>{t('{paid} paid of {planned}', {
						paid: money(row.paid, row.currency),
						planned: money(row.planned, row.currency)
					})}</span
				>
			{/each}
		</p>
		{#each soon as c (c.id)}
			<p class="warning" role="note">
				{t('{name} expires {when}: plan the renewal.', {
					name: c.code || c.name,
					when: when(c.expires_on ?? today)
				})}
			</p>
		{/each}
	</section>

	{#if items.length}
		<ul class="certs">
			{#each items as c (c.id)}
				<li class="card">
					<div class="head">
						<h2>
							{#if c.url}<a href={c.url} rel="noopener noreferrer" target="_blank">{c.name}</a
								>{:else}{c.name}{/if}
						</h2>
						<span class="chip {c.status}">{statusNames[c.status]}</span>
					</div>
					<p class="muted small">
						{[c.code, c.provider].filter(Boolean).join(' · ')}
						{#if c.cost_cents != null}· {money(c.cost_cents, c.currency)}{/if}
						{#if c.validity_months}· {t('valid {n} months', { n: c.validity_months })}{/if}
					</p>
					<p class="small">
						{#if c.status === 'planned' && c.target_on}
							{t('Aim: by {date}', { date: c.target_on })}
						{:else if c.status === 'booked' && c.exam_on}
							{t('Exam on {date}, {when}', { date: c.exam_on, when: when(c.exam_on) })}
						{:else if c.status === 'passed'}
							{t('Passed on {date}', { date: c.passed_on ?? '' })}{#if c.score}
								· {c.score}{/if}{#if c.expires_on}
								· {t('expires {date}', { date: c.expires_on })}{/if}
							{#if c.credential_url}· <a
									href={c.credential_url}
									rel="noopener noreferrer"
									target="_blank">{t('Credential')}</a
								>{/if}
						{:else if c.status === 'failed' && c.exam_on}
							{t('Not passed on {date}: book a retake when ready.', { date: c.exam_on })}
						{/if}
					</p>
					{#if editing === c.id}
						<form
							class="grid"
							onsubmit={(e) => {
								e.preventDefault();
								save(c);
							}}
						>
							<label
								>{t('Status')}
								<select bind:value={upd.status}>
									{#each Object.entries(statusNames) as [v, l] (v)}<option value={v}>{l}</option
										>{/each}
								</select></label
							>
							<label>{t('Exam date')} <input type="date" bind:value={upd.exam_on} /></label>
							<label>{t('Score')} <input bind:value={upd.score} maxlength="40" /></label>
							<label
								>{t('Credential link')} <input type="url" bind:value={upd.credential_url} /></label
							>
							<div class="row">
								<Button type="submit">{t('Save')}</Button>
								<Button variant="ghost" onclick={() => (editing = null)}>{t('Cancel')}</Button>
							</div>
						</form>
					{:else}
						<div class="row">
							<Button
								variant="secondary"
								onclick={() => open(c)}
								aria-label={t('Update {name}', { name: c.name })}>{t('Update')}</Button
							>
							<Button
								variant="ghost"
								onclick={() => remove(c)}
								aria-label={t('Remove {name}', { name: c.name })}>{t('Remove')}</Button
							>
						</div>
					{/if}
				</li>
			{/each}
		</ul>
	{:else}
		<Status
			kind="empty"
			message={t(
				'No certifications yet. Add one below, or start the roadmap: it plans its 43 certifications.'
			)}
		/>
	{/if}

	<section class="card" aria-labelledby="add-title">
		<h2 id="add-title">{t('Add a certification')}</h2>
		<form class="grid" onsubmit={add}>
			<label class="wide"
				>{t('Name')} <input bind:value={draft.name} required maxlength="200" /></label
			>
			<label
				>{t('Code')} <input bind:value={draft.code} maxlength="40" placeholder={t('CKA')} /></label
			>
			<label>{t('Provider')} <input bind:value={draft.provider} maxlength="100" /></label>
			<label class="wide">{t('Link')} <input type="url" bind:value={draft.url} /></label>
			<label>{t('Aim to take it by')} <input type="date" bind:value={draft.target_on} /></label>
			<label>{t('Cost')} <input inputmode="decimal" bind:value={draft.cost} /></label>
			<label
				>{t('Currency')}
				<select bind:value={draft.currency}>
					{#each ['USD', 'EUR', 'GBP', 'CHF', 'CAD'] as cur (cur)}<option value={cur}>{cur}</option
						>{/each}
				</select></label
			>
			<label
				>{t('Valid for (months)')} <input inputmode="numeric" bind:value={draft.validity} /></label
			>
			<div class="wide">
				<Button type="submit" disabled={!draft.name.trim()}>{t('Add')}</Button>
			</div>
		</form>
	</section>
{/if}

<style>
	.hint,
	.small {
		font-size: var(--text-sm);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.kpis {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-4);
		margin: 0;
	}
	.warning {
		border-left: 4px solid var(--accent);
		background: var(--accent-soft);
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		font-size: var(--text-sm);
	}
	.certs {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 20rem), 1fr));
		gap: var(--space-3);
	}
	.certs .card {
		margin: 0;
	}
	.head {
		display: flex;
		justify-content: space-between;
		gap: var(--space-2);
		align-items: start;
	}
	h2 {
		font-size: var(--text-lg);
		margin: 0;
	}
	.chip {
		font-size: var(--text-sm);
		padding: 0 var(--space-2);
		border-radius: 999px;
		background: var(--accent-soft);
		white-space: nowrap;
	}
	.chip.passed {
		background: color-mix(in srgb, var(--ok) 25%, transparent);
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 11rem), 1fr));
		gap: var(--space-3);
		margin-top: var(--space-3);
	}
	.grid label {
		display: grid;
		gap: var(--space-1);
	}
	.wide {
		grid-column: 1 / -1;
	}
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		margin-top: var(--space-2);
	}
	input,
	select {
		font: inherit;
		padding: var(--space-2);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--surface);
		color: var(--fg);
	}
</style>
