<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { api, message, type Schemas } from '$lib/api';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Section from '$lib/ui/Section.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	let consents = $state<Schemas['Consent'][] | null>(null);
	let error = $state('');
	let busy = $state(false);
	let confirming = $state(false);
	let confirm = $state('');

	async function load() {
		error = '';
		try {
			consents = (await api<{ items: Schemas['Consent'][] }>('/v1/me/consents')).items;
		} catch (err) {
			error = message(err);
		}
	}

	async function setConsent(c: Schemas['Consent'], granted: boolean) {
		// Withdrawing this consent deletes the health logs it allowed.
		if (
			c.id === 'health_data' &&
			!granted &&
			!window.confirm(t('Turning this off deletes all your food and training logs now. Continue?'))
		)
			return;
		busy = true;
		try {
			await api(`/v1/me/consents/${c.id}`, { method: 'PUT', body: { granted } });
			toast(granted ? t('Allowed.') : t('Turned off.'));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function erase(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			await api('/v1/me', { method: 'DELETE', body: { confirm } });
			await goto('/');
		} catch (err) {
			toast(message(err), 'error');
			busy = false;
		}
	}

	onMount(load);
</script>

<svelte:head><title>{t('Privacy & data · Axon')}</title></svelte:head>

<Section
	id="choices"
	title={t('Privacy choices')}
	description={t(
		'Everything is off until you turn it on. You can change your mind at any time; each change is recorded in your activity.'
	)}
>
	{#if error}
		<Status kind="error" message={error} retry={load} />
	{:else if !consents}
		<Status kind="loading" message={t('Loading…')} />
	{:else}
		{#each consents as c (c.id)}
			<label class="choice">
				<input
					type="checkbox"
					checked={c.granted}
					disabled={busy}
					onchange={(e) => setConsent(c, e.currentTarget.checked)}
				/>
				<span>{c.description}</span>
			</label>
		{/each}
	{/if}
</Section>

<Section
	id="export"
	title={t('Download your data')}
	description={t(
		'Everything Axon holds about you: a JSON file for machines and a Markdown summary for people.'
	)}
>
	<Button href="/api/v1/me/export" variant="secondary">{t('Download everything')}</Button>
</Section>

<Section
	id="delete"
	title={t('Delete your account')}
	description={t(
		'Your account and all its data are deleted immediately. Encrypted backups expire within 30 days.'
	)}
>
	<Button variant="danger" onclick={() => (confirming = true)}>{t('Delete account…')}</Button>
</Section>

<Dialog bind:open={confirming} title={t('Delete your account?')}>
	<p>{t('This cannot be undone. Download your data first if you want to keep it.')}</p>
	<form onsubmit={erase}>
		<TextField
			id="confirm"
			label={t('Type DELETE to confirm')}
			bind:value={confirm}
			autocomplete="off"
			required
		/>
		<div class="actions">
			<Button variant="secondary" onclick={() => (confirming = false)}>{t('Cancel')}</Button>
			<Button type="submit" variant="danger" {busy} disabled={confirm !== 'DELETE'}
				>{t('Delete account')}</Button
			>
		</div>
	</form>
</Dialog>

<style>
	.choice {
		display: flex;
		gap: var(--space-3);
		align-items: flex-start;
		padding: var(--space-3) 0;
		border-bottom: 1px solid var(--border);
	}
	.choice:last-child {
		border-bottom: none;
	}
	.choice input {
		width: 1.25rem;
		height: 1.25rem;
		margin-top: 0.15rem;
		accent-color: var(--accent);
	}
	.actions {
		display: flex;
		gap: var(--space-3);
		justify-content: flex-end;
		flex-wrap: wrap;
	}
</style>
