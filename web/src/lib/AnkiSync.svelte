<script lang="ts">
	import { appLocale } from '\$lib/dates';
	import { t } from '$lib/i18n.svelte';
	import { api, message } from './api';
	import type { paths } from './api-types';
	import Button from './ui/Button.svelte';
	import TextField from './ui/TextField.svelte';
	import { toast } from './ui/toast.svelte';

	type State = paths['/v1/me/anki']['get']['responses'][200]['content']['application/json'];
	let s = $state<State | null>(null);
	let form = $state({ endpoint: '', username: '', password: '', auto_sync: true });
	let syncing = $state(false);
	let editing = $state(false);

	async function load() {
		try {
			s = await api<State>('/v1/me/anki');
			const a = s.account;
			form = {
				endpoint: a?.endpoint ?? s.default_endpoint,
				username: a?.username ?? '',
				password: '',
				auto_sync: a?.auto_sync ?? true
			};
		} catch {
			s = null;
		}
	}
	load();

	async function save(e: SubmitEvent) {
		e.preventDefault();
		try {
			const { password, ...rest } = form;
			await api('/v1/me/anki', { method: 'PUT', body: password ? { ...rest, password } : rest });
			toast(t('Anki account saved.'));
			editing = false;
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function sync() {
		syncing = true;
		try {
			const r = await api<{ added: number; updated: number }>('/v1/me/anki/sync', {
				method: 'POST'
			});
			toast(
				t('Sent to Anki: {added} added, {updated} updated.', { added: r.added, updated: r.updated })
			);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			syncing = false;
			await load();
		}
	}

	async function disconnect() {
		try {
			await api('/v1/me/anki', { method: 'DELETE' });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

{#if s?.available}
	<section class="anki card glass" aria-labelledby="anki-title">
		<h2 id="anki-title">Anki</h2>
		{#if s.account && !editing}
			<p>
				{t('Connected to')} <strong>{s.account.endpoint}</strong>
				{t('as')} <strong>{s.account.username}</strong>.
				{s.account.auto_sync
					? t('Cards you approve from the Teacher are sent automatically.')
					: t('Send cards with the button.')}
			</p>
			{#if s.account.last_sync_at}
				<p class="small" class:error={s.account.last_status === 'error'} role="status">
					{t('Last sync {value}: {last_message}', {
						value: new Date(s.account.last_sync_at).toLocaleString(appLocale()),
						last_message: s.account.last_message
					})}
				</p>
			{/if}
			<div class="row">
				<Button onclick={sync} busy={syncing}>{t('Sync to Anki now')}</Button>
				<Button variant="secondary" onclick={() => (editing = true)}>{t('Change account')}</Button>
				<Button variant="ghost" onclick={disconnect}>{t('Disconnect')}</Button>
			</div>
		{:else}
			<p class="muted small">
				{t(
					'Send your cards to your own anki-sync-server (for example the one on srv-services), then sync Anki on your computer or phone. Notes are updated in place; cards that exist only in Anki are never deleted.'
				)}
			</p>
			<form onsubmit={save}>
				<TextField
					id="anki-endpoint"
					label={t('Sync server')}
					type="url"
					placeholder="https://anki.example/"
					bind:value={form.endpoint}
					required
				/>
				<TextField
					id="anki-username"
					label={t('Anki user name')}
					bind:value={form.username}
					maxlength={200}
					required
				/>
				<TextField
					id="anki-password"
					label={t('Anki password')}
					type="password"
					hint={s.account
						? t('Leave empty to keep the saved password.')
						: t('Stored encrypted; never shown again.')}
					bind:value={form.password}
					maxlength={500}
				/>
				<label class="check"
					><input type="checkbox" bind:checked={form.auto_sync} />
					{t("Send cards automatically after I approve the Teacher's proposals")}</label
				>
				<div class="row">
					<Button type="submit" variant="secondary">{t('Save')}</Button>
					{#if editing}<Button variant="ghost" onclick={() => (editing = false)}
							>{t('Cancel')}</Button
						>{/if}
				</div>
			</form>
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
	.small {
		font-size: var(--text-sm);
	}
	.error {
		color: var(--bad);
	}
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}
	.check {
		display: flex;
		gap: var(--space-2);
		align-items: baseline;
		margin-bottom: var(--space-3);
	}
</style>
