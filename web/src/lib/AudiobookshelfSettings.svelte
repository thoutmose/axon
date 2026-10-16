<script lang="ts">
	import { api, message } from '$lib/api';
	import { appLocale } from '$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// Podcasts from the person's Audiobookshelf (EP-124): each show follows
	// in the tech watch, each episode with its notes and, with the
	// operator's Whisper server, its words.
	type Status = {
		connected: boolean;
		url: string;
		synced_at: string | null;
		error: string;
		shows: number;
	};
	let st = $state<Status | null>(null);
	let url = $state('');
	let token = $state('');
	let busy = $state(false);
	const uid = $props.id();

	async function load() {
		try {
			st = await api<Status>('/v1/audiobookshelf');
			url = st.url;
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	load();

	async function connect(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		try {
			st = await api<Status>('/v1/audiobookshelf', { method: 'PUT', body: { url, token } });
			token = '';
			toast(t('Connected: {n} shows in your tech watch.', { n: st.shows }));
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
	async function sync() {
		busy = true;
		try {
			const r = await api<{ shows: number; new_shows: number; new_episodes: number }>(
				'/v1/audiobookshelf/sync',
				{ method: 'POST' }
			);
			toast(t('{n} new episodes.', { n: r.new_episodes }));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
	async function disconnect() {
		if (
			!confirm(
				t('Stop following your Audiobookshelf podcasts? Their episodes stay in the tech watch.')
			)
		)
			return;
		try {
			await api('/v1/audiobookshelf', { method: 'DELETE' });
			token = '';
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<p class="muted small">
	{t(
		'Follow the podcasts your Audiobookshelf server downloads in the tech watch: each show is a source, each episode news with its notes, and, when Axon is set up with a Whisper server, its words. The token is sealed and never shown again.'
	)}
</p>
{#if st?.connected}
	<p>
		{t('Connected to {url} · {n} shows', { url: st.url, n: st.shows })}{#if st.synced_at}
			· {t('synced {date}', { date: new Date(st.synced_at).toLocaleString(appLocale()) })}{/if}
	</p>
	{#if st.error}<p class="error" role="alert">{st.error}</p>{/if}
{/if}
<form class="grid" onsubmit={connect}>
	<div class="field">
		<label for="{uid}-url">{t('Address')}</label>
		<input
			id="{uid}-url"
			type="url"
			bind:value={url}
			placeholder="https://abs.example.org"
			required
			autocomplete="off"
		/>
	</div>
	<div class="field">
		<label for="{uid}-token"
			>{st?.connected
				? t('New API token (empty keeps the saved one)')
				: t('API key (Audiobookshelf → Settings → API Keys)')}</label
		>
		<input
			id="{uid}-token"
			type="password"
			bind:value={token}
			autocomplete="off"
			required={!st?.connected}
		/>
	</div>
	<div class="row">
		<Button type="submit" {busy}>{st?.connected ? t('Save') : t('Connect')}</Button>
		{#if st?.connected}
			<Button variant="secondary" onclick={sync} {busy}>{t('Sync now')}</Button>
			<Button variant="ghost" onclick={disconnect}>{t('Disconnect')}</Button>
		{/if}
	</div>
</form>

<style>
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
		gap: var(--space-3);
	}
	.field {
		display: grid;
		gap: var(--space-1);
	}
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		grid-column: 1 / -1;
	}
	.small {
		font-size: var(--text-sm);
	}
	.error {
		color: var(--bad);
	}
</style>
