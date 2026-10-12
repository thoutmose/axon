<script lang="ts">
	import { api, message, type Schemas } from './api';
	import { t } from './i18n.svelte';
	import Button from './ui/Button.svelte';
	import { toast } from './ui/toast.svelte';

	// Search by meaning (EP-62): the person's own embedding service, behind
	// its own consent; off by default.
	type Status = Schemas['EmbeddingStatus'];
	let status = $state<Status | null>(null);
	let provider = $state<'off' | 'local' | 'openai' | 'voyage'>('off');
	let baseURL = $state('http://ollama:11434');
	let model = $state('');
	let key = $state('');
	const defaults: Record<string, string> = {
		local: 'nomic-embed-text',
		openai: 'text-embedding-3-small',
		voyage: 'voyage-3.5-lite'
	};

	async function load() {
		try {
			status = await api<Status>('/v1/me/embeddings');
			if (status.settings) {
				provider = status.settings.provider;
				model = status.settings.model;
				if (status.settings.base_url) baseURL = status.settings.base_url;
			}
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	load();

	async function save() {
		try {
			if (provider === 'off') {
				await api('/v1/me/embeddings', { method: 'DELETE' });
				toast(t('Search by meaning is off; stored embeddings were deleted.'));
			} else {
				await api('/v1/me/embeddings', {
					method: 'PUT',
					body: {
						provider,
						model: model || defaults[provider],
						...(provider === 'local' ? { base_url: baseURL } : {}),
						...(key ? { api_key: key } : {})
					}
				});
				key = '';
				toast(t('Saved. Notes are embedded within a few minutes once you consent.'));
			}
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function consent(on: boolean) {
		try {
			await api('/v1/me/consents/ai_embeddings', { method: 'PUT', body: { granted: on } });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<div class="field">
	<label for="emb-provider">{t('Embedding service')}</label>
	<select id="emb-provider" bind:value={provider}>
		<option value="off">{t('Off')}</option>
		<option value="local">{t('A local model server (Ollama, LM Studio)')}</option>
		<option value="openai">OpenAI</option>
		<option value="voyage">Voyage AI</option>
	</select>
</div>
{#if provider === 'local'}
	<div class="field">
		<label for="emb-base">{t('Server address')}</label>
		<input id="emb-base" bind:value={baseURL} maxlength="300" placeholder="http://ollama:11434" />
	</div>
{/if}
{#if provider !== 'off'}
	<div class="field">
		<label for="emb-model">{t('Model')}</label>
		<input id="emb-model" bind:value={model} maxlength="100" placeholder={defaults[provider]} />
	</div>
	<div class="field">
		<label for="emb-key"
			>{t('API key')}{#if status?.settings?.has_key}<span class="muted small">
					({t('saved; leave empty to keep it')})</span
				>{/if}</label
		>
		<input id="emb-key" type="password" bind:value={key} maxlength="500" autocomplete="off" />
	</div>
{/if}
<div class="row">
	<Button variant="secondary" onclick={save}>{t('Save the embedding service')}</Button>
</div>
{#if status?.settings}
	<label class="check"
		><input
			type="checkbox"
			checked={status.consent}
			onchange={(e) => consent(e.currentTarget.checked)}
		/>
		{t('Send the text of my notes to this service to search them by meaning')}</label
	>
	<p class="muted small">
		{status.indexed} / {status.notes}
		{t('notes embedded')}{#if status.settings.error}
			· <span class="bad">{status.settings.error}</span>{/if}
	</p>
{/if}
<p class="muted small">
	{t(
		'Off, Axon finds related notes with its own local word statistics. On, notes are compared by meaning: in the command center (Ctrl+K) and under “Close in meaning” on each note. A local model server keeps your notes on your network; OpenAI and Voyage AI receive their text.'
	)}
</p>

<style>
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-3);
	}
	.row {
		margin-bottom: var(--space-3);
	}
	.small {
		font-size: var(--text-sm);
	}
	.bad {
		color: var(--bad);
	}
</style>
