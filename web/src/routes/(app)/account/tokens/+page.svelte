<script lang="ts">
	import { appLocale } from '\$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import { onMount } from 'svelte';
	import { api, message, type Schemas } from '$lib/api';
	import Button from '$lib/ui/Button.svelte';
	import List from '$lib/ui/List.svelte';
	import Section from '$lib/ui/Section.svelte';
	import Status from '$lib/ui/Status.svelte';
	import TextField from '$lib/ui/TextField.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	let tokens = $state<Schemas['Token'][] | null>(null);
	let error = $state('');
	let busy = $state(false);
	let name = $state('');
	let scope = $state<'read' | 'write'>('read');
	let days = $state(90);
	let secret = $state('');

	async function load() {
		error = '';
		try {
			tokens = (await api<{ items: Schemas['Token'][] }>('/v1/me/tokens')).items;
		} catch (err) {
			error = message(err);
		}
	}

	async function create(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			const res = await api<{ token: string }>('/v1/me/tokens', {
				method: 'POST',
				body: { name, scope, expires_in_days: days }
			});
			secret = res.token;
			name = '';
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function revoke(token: Schemas['Token']) {
		busy = true;
		try {
			await api(`/v1/me/tokens/${token.id}`, { method: 'DELETE' });
			toast(t('Revoked “{name}”.', { name: token.name }));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	onMount(load);
</script>

<svelte:head><title>{t('Access tokens · Axon')}</title></svelte:head>

<Section
	id="tokens"
	title={t('Access tokens')}
	description={t(
		'For scripts and sync tools. A token can read (or change) your data, but never manage your account, export it, or delete it.'
	)}
>
	<p class="muted">
		{t('Every endpoint is documented in the')}
		<a href="/api-docs/index.html" rel="external">{t('API reference')}</a>{t(
			', generated from the live OpenAPI document.'
		)}
	</p>
	{#if error}
		<Status kind="error" message={error} retry={load} />
	{:else if !tokens}
		<Status kind="loading" message={t('Loading…')} />
	{:else}
		<List items={tokens} key={(t) => t.id} empty={t('No tokens yet.')}>
			{#snippet row(token)}
				<span
					>{token.name}
					<span class="muted"
						>{t(
							token.scope === 'read'
								? '· read only · expires {date}'
								: '· read and write · expires {date}',
							{ date: new Date(token.expires_at).toLocaleDateString(appLocale()) }
						)}</span
					></span
				>
				<Button variant="secondary" onclick={() => revoke(token)} {busy}>{t('Revoke')}</Button>
			{/snippet}
		</List>
	{/if}
	{#if secret}
		<p role="status">{t('Copy this token now; it will not be shown again:')}</p>
		<p><code>{secret}</code></p>
	{/if}
	<form onsubmit={create}>
		<TextField
			id="token-name"
			label={t('Name')}
			placeholder={t('e.g. Notes sync')}
			bind:value={name}
			maxlength={100}
			required
		/>
		<div class="field">
			<label for="token-scope">{t('Access')}</label>
			<select id="token-scope" bind:value={scope}>
				<option value="read">{t('Read only')}</option>
				<option value="write">{t('Read and write')}</option>
			</select>
		</div>
		<TextField
			id="token-days"
			label={t('Expires after (days)')}
			type="number"
			min={1}
			max={365}
			bind:value={days}
			required
		/>
		<Button type="submit" {busy}>{t('Create token')}</Button>
	</form>
</Section>

<style>
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
		max-width: 28rem;
	}
	label {
		font-weight: 600;
	}
	select {
		font: inherit;
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
</style>
