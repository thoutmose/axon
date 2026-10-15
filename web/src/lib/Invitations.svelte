<script lang="ts">
	import { api, ApiError, message } from '$lib/api';
	import { appLocale } from '$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// Invitations (EP-120): the owner invites one person at a time with a
	// link used once, that can bring them into the household.
	type Invite = {
		id: string;
		note: string;
		household: boolean;
		expires_at: string;
		created_at: string;
		used_at: string | null;
		used_by_name: string | null;
		url?: string;
	};
	let items = $state<Invite[] | null>(null);
	let owner = $state(true);
	let note = $state('');
	let household = $state(false);
	let days = $state(7);
	let link = $state('');
	let busy = $state(false);

	async function load() {
		try {
			items = (await api<{ items: Invite[] }>('/v1/invites')).items;
		} catch (err) {
			if (err instanceof ApiError && err.status === 403) owner = false;
			else toast(message(err), 'error');
		}
	}
	load();

	async function create(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			const inv = await api<Invite>('/v1/invites', {
				method: 'POST',
				body: { note, household, days }
			});
			link = inv.url ?? '';
			note = '';
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
	async function copy() {
		await navigator.clipboard.writeText(link);
		toast(t('Link copied.'));
	}
	async function share() {
		try {
			await navigator.share({ title: t('Join me on Axon'), url: link });
		} catch {
			// cancelled
		}
	}
	async function revoke(i: Invite) {
		try {
			await api(`/v1/invites/${i.id}`, { method: 'DELETE' });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	const date = (iso: string) => new Date(iso).toLocaleDateString(appLocale());
	const statusOf = (i: Invite) =>
		i.used_at
			? t('Used by {name} on {date}', { name: i.used_by_name ?? '—', date: date(i.used_at) })
			: new Date(i.expires_at) < new Date()
				? t('Expired')
				: t('Waiting, until {date}', { date: date(i.expires_at) });
	const canShare = typeof navigator !== 'undefined' && 'share' in navigator;
</script>

{#if owner}
	<section class="card" aria-labelledby="invites-title">
		<h2 id="invites-title">{t('Invitations')}</h2>
		<p class="muted small">
			{t(
				'Invite one person to this Axon with a link used once, even while registration is closed. Send it privately: anyone with it can create the account.'
			)}
		</p>
		<form class="row" onsubmit={create}>
			<label class="grow"
				>{t('For whom (a note for you)')}
				<input bind:value={note} maxlength="200" placeholder={t('For Sam')} /></label
			>
			<label
				>{t('Valid for')}
				<select bind:value={days}>
					<option value={1}>{t('1 day')}</option>
					<option value={7}>{t('7 days')}</option>
					<option value={30}>{t('30 days')}</option>
				</select></label
			>
			<label class="check"
				><input type="checkbox" bind:checked={household} />
				{t('Also into my household')}</label
			>
			<Button type="submit" {busy}>{t('Create a link')}</Button>
		</form>
		{#if link}
			<div class="link" role="status">
				<label
					>{t('Invitation link')}
					<input readonly value={link} onfocus={(e) => e.currentTarget.select()} /></label
				>
				<Button variant="secondary" onclick={copy}>{t('Copy')}</Button>
				{#if canShare}<Button variant="secondary" onclick={share}>{t('Share…')}</Button>{/if}
			</div>
			<p class="muted small">{t('The link is shown only now.')}</p>
		{/if}
		{#if items?.length}
			<ul class="list">
				{#each items as i (i.id)}
					<li>
						<span class="grow"
							>{i.note || t('Invitation')}
							<span class="muted small"
								>· {statusOf(i)}{i.household ? ' · ' + t('with the household') : ''}</span
							></span
						>
						{#if !i.used_at}
							<button
								type="button"
								class="linkbtn"
								onclick={() => revoke(i)}
								aria-label={t('Revoke {note}', { note: i.note || t('Invitation') })}
								>{t('Revoke')}</button
							>
						{/if}
					</li>
				{/each}
			</ul>
		{/if}
	</section>
{/if}

<style>
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}
	.row,
	.link {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
		align-items: end;
	}
	.row label,
	.link label {
		display: grid;
		gap: var(--space-1);
	}
	.grow {
		flex: 1;
		min-width: 12rem;
	}
	.check {
		display: flex !important;
		gap: var(--space-2);
		align-items: center;
	}
	.link {
		margin-top: var(--space-3);
	}
	.link input {
		min-width: min(28rem, 70vw);
	}
	.small {
		font-size: var(--text-sm);
	}
	.list {
		list-style: none;
		padding: 0;
		margin: var(--space-3) 0 0;
	}
	.list li {
		display: flex;
		gap: var(--space-2);
		padding: var(--space-2) 0;
		border-bottom: 1px solid var(--border);
	}
	.linkbtn {
		background: none;
		border: 0;
		color: var(--accent);
		cursor: pointer;
		font: inherit;
		padding: 0;
	}
</style>
