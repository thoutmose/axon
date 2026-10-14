<script lang="ts">
	import { api, message, type Schemas } from './api';
	import { t } from './i18n.svelte';
	import Button from './ui/Button.svelte';
	import { toast } from './ui/toast.svelte';

	// Where the Markdown vault lives (EP-69): in Axon only by default, or also
	// as files on a server, paired once with a code and kept in step there.
	type Source = Schemas['VaultSource'];
	type Setup = { code: string; expires_at: string; install: string; docker: string };

	let source = $state<Source | null>(null);
	let loaded = $state(false);
	let where = $state<'axon' | 'server'>('axon');
	let host = $state('');
	let dir = $state('~/SecondBrain');
	let scaffold = $state(true);
	let setup = $state<Setup | null>(null);
	let busy = $state(false);

	async function load() {
		try {
			const res = await api<{ source: Source | null }>('/v1/vault/source');
			source = res.source;
			if (source) {
				where = 'server';
				host = source.host;
				dir = source.dir;
				scaffold = source.scaffold;
			}
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			loaded = true;
		}
	}
	load();

	async function connect() {
		busy = true;
		try {
			const res = await api<{ source: Source; setup: Setup }>('/v1/vault/source', {
				method: 'PUT',
				body: { host, dir, scaffold }
			});
			source = res.source;
			setup = res.setup;
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function disconnect() {
		if (!source) {
			where = 'axon';
			return;
		}
		if (
			!confirm(
				t(
					'Stop syncing with {host}? Its token is revoked; your notes stay in Axon and the files stay on the server.',
					{ host: source.host }
				)
			)
		)
			return;
		try {
			await api('/v1/vault/source', { method: 'DELETE' });
			source = null;
			setup = null;
			where = 'axon';
			toast(t('Your notes now live in Axon only.'));
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function copy(text: string) {
		try {
			await navigator.clipboard.writeText(text);
			toast(t('Copied.'));
		} catch {
			toast(t('Select the command and copy it.'), 'error');
		}
	}

	const when = (iso: string | null | undefined) => (iso ? new Date(iso).toLocaleString() : '—');
	const statusText: Record<string, string> = {
		waiting: 'Waiting for the server to pair',
		connected: 'In step',
		silent: 'No news from the server for 15 minutes',
		failing: 'The last sync failed'
	};
</script>

<fieldset class="where">
	<legend>{t('Where your notes live')}</legend>
	<label class="check"
		><input type="radio" bind:group={where} value="axon" disabled={!loaded} />
		<span
			><strong>{t('In Axon only')}</strong>
			<span class="muted small"
				>{t(
					'The default. Notes are Markdown in the database: nothing to set up, included in backups and exports.'
				)}</span
			></span
		></label
	>
	<label class="check"
		><input type="radio" bind:group={where} value="server" disabled={!loaded} />
		<span
			><strong>{t('Also as files on a server')}</strong>
			<span class="muted small"
				>{t(
					'A folder of Markdown files (Obsidian, a Git repository…) on a server of yours, kept in step both ways within seconds.'
				)}</span
			></span
		></label
	>
</fieldset>

{#if where === 'server'}
	{#if source && !setup}
		<p class="state {source.status}" role="status">
			<strong>{t(statusText[source.status] ?? source.status)}</strong> ·
			{source.paired_host || source.host}:{source.dir}
		</p>
		<p class="muted small">
			{t('Last report')}: {when(source.last_seen_at)}
			{#if source.status !== 'waiting'}
				· {source.last_report.notes} {t('notes')}, {source.last_report.files} {t('files')}
			{/if}
			{#if source.token_expires_at}· {t('token until')} {when(source.token_expires_at)}{/if}
		</p>
		{#if source.last_report.error}<p class="bad small">{source.last_report.error}</p>{/if}
		{#if source.last_report.conflicts?.length}
			<p class="small">
				{t('Kept both versions of')}: {source.last_report.conflicts?.join(', ')}
				<span class="muted">({t('merge them, then delete the *.axon-conflict.* copy')})</span>
			</p>
		{/if}
	{/if}
	<div class="grid">
		<div class="field">
			<label for="vault-host">{t('Server')}</label>
			<input
				id="vault-host"
				bind:value={host}
				maxlength="100"
				placeholder="srv-secondbrain"
				autocomplete="off"
			/>
		</div>
		<div class="field">
			<label for="vault-dir">{t('Folder on the server')}</label>
			<input id="vault-dir" bind:value={dir} maxlength="500" placeholder="~/SecondBrain" />
		</div>
	</div>
	<label class="check"
		><input type="checkbox" bind:checked={scaffold} />
		{t(
			'If the folder is empty, start it with the folder structure (Inbox, PARA, Zettelkasten, Daily) and the templates'
		)}</label
	>
	<div class="row">
		<Button variant="secondary" {busy} disabled={!host || !dir} onclick={connect}
			>{source ? t('Pair again') : t('Set up this server')}</Button
		>
		<Button variant="ghost" onclick={disconnect}>{source ? t('Stop syncing') : t('Cancel')}</Button>
	</div>

	{#if setup}
		<div class="setup" aria-live="polite">
			<p>
				{t('On {host}, run this once. The code {code} works once, until {time}.', {
					host,
					code: setup.code,
					time: new Date(setup.expires_at).toLocaleTimeString()
				})}
			</p>
			<pre><code>{setup.install}</code></pre>
			<Button variant="ghost" onclick={() => copy(setup!.install)}>{t('Copy')}</Button>
			<p class="muted small">
				{t(
					'It downloads axon to ~/.local/bin, pairs, syncs once, and installs a user service. Nothing runs as root. With Docker instead:'
				)}
			</p>
			<pre><code>{setup.docker}</code></pre>
			<Button variant="ghost" onclick={() => copy(setup!.docker)}>{t('Copy')}</Button>
			<p class="muted small">
				{t(
					'Axon must be reachable from the server (on your tailnet, for example). This page shows when it is in step.'
				)}
				<Button variant="ghost" onclick={() => ((setup = null), load())}>{t('Check now')}</Button>
			</p>
		</div>
	{/if}
{/if}

<style>
	.where {
		border: 0;
		padding: 0;
		margin: 0 0 var(--space-3);
	}
	legend {
		font-weight: 600;
		margin-bottom: var(--space-2);
	}
	.where .check span {
		display: grid;
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 14rem), 1fr));
		gap: var(--space-3) var(--space-4);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-3);
	}
	.field label {
		font-weight: 600;
	}
	.check {
		display: flex;
		gap: var(--space-2);
		align-items: baseline;
		margin-bottom: var(--space-2);
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}
	.state::before {
		content: '●';
		margin-right: var(--space-1);
	}
	.state.connected::before {
		color: var(--ok);
	}
	.state.silent::before,
	.state.waiting::before {
		color: var(--warn);
	}
	.state.failing::before {
		color: var(--bad);
	}
	.setup {
		margin-top: var(--space-3);
		padding: var(--space-3);
		border: 1px solid var(--border);
		border-radius: var(--radius);
	}
	pre {
		overflow-x: auto;
		white-space: pre-wrap;
		word-break: break-all;
		background: var(--surface-solid);
		padding: var(--space-2);
		border-radius: var(--radius);
		font-size: var(--text-sm);
	}
	.small {
		font-size: var(--text-sm);
	}
	.bad {
		color: var(--bad);
	}
</style>
