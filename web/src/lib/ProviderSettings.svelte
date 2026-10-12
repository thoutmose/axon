<script lang="ts">
	import { api, message, type Schemas } from './api';
	import Button from './ui/Button.svelte';
	import TextField from './ui/TextField.svelte';
	import { toast } from './ui/toast.svelte';

	let { onchange }: { onchange: () => void } = $props();

	type State = {
		settings: Schemas['ProviderSettings'] | null;
		server_default: boolean;
		can_store_keys: boolean;
		providers: string[];
	};
	const names: Record<string, string> = {
		anthropic: 'Anthropic (Claude)',
		openai: 'OpenAI',
		mistral: 'Mistral',
		gemini: 'Google Gemini',
		deepseek: 'DeepSeek',
		local: 'Local model (Ollama, LM Studio, llama.cpp)'
	};

	let info = $state<State | null>(null);
	let form = $state({
		provider: 'anthropic',
		model: 'claude-opus-5-5',
		base_url: '',
		api_key: '',
		input_price: '',
		output_price: ''
	});
	let busy = $state(false);
	let result = $state<{ ok: boolean; detail: string } | null>(null);

	async function load() {
		info = await api<State>('/v1/me/agent-provider');
		const s = info.settings;
		if (s) {
			form = {
				provider: s.provider,
				model: s.model,
				base_url: s.base_url ?? '',
				api_key: '',
				input_price: s.input_price?.toString() ?? '',
				output_price: s.output_price?.toString() ?? ''
			};
		}
	}
	load().catch((err) => toast(message(err), 'error'));

	async function save(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		result = null;
		try {
			const price = (v: string) => (v.trim() === '' ? undefined : Number(v));
			await api('/v1/me/agent-provider', {
				method: 'PUT',
				body: {
					provider: form.provider,
					model: form.model,
					...(form.provider === 'local' ? { base_url: form.base_url } : {}),
					...(form.api_key ? { api_key: form.api_key } : {}),
					input_price: price(
						form.provider === 'local' && form.input_price === '' ? '0' : form.input_price
					),
					output_price: price(
						form.provider === 'local' && form.output_price === '' ? '0' : form.output_price
					)
				}
			});
			toast('Provider saved.');
			await load();
			onchange();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}

	async function test() {
		result = await api<{ ok: boolean; detail: string }>('/v1/me/agent-provider/test', {
			method: 'POST'
		});
	}

	async function removeKey() {
		await api('/v1/me/agent-provider', {
			method: 'PUT',
			body: {
				provider: form.provider,
				model: form.model,
				...(form.provider === 'local' ? { base_url: form.base_url } : {}),
				api_key: ''
			}
		});
		await load();
		onchange();
	}

	async function useDefault() {
		await api('/v1/me/agent-provider', { method: 'DELETE' });
		info = null;
		await load();
		onchange();
	}
</script>

<section class="card" aria-labelledby="provider-title">
	<h2 id="provider-title">AI provider</h2>
	{#if info}
		<p class="muted">
			{#if info.settings}
				Your agents use {names[info.settings.provider] ?? info.settings.provider} ({info.settings
					.model}).
			{:else if info.server_default}
				Your agents use this server's Anthropic account. Choose your own provider below if you
				prefer.
			{:else}
				Choose a provider to use agents. A local model keeps everything on your own hardware.
			{/if}
		</p>
		<form onsubmit={save}>
			<div class="field">
				<label for="p-provider">Provider</label>
				<select id="p-provider" bind:value={form.provider}>
					{#each info.providers as p (p)}<option value={p}>{names[p] ?? p}</option>{/each}
				</select>
			</div>
			<TextField
				id="p-model"
				label="Model"
				hint={form.provider === 'anthropic'
					? 'For example claude-opus-5-5.'
					: form.provider === 'local'
						? 'The model name on your server, for example llama3.1:8b.'
						: "The model name from your provider's documentation."}
				bind:value={form.model}
				maxlength={100}
				required
			/>
			{#if form.provider === 'local'}
				<TextField
					id="p-url"
					label="Server address"
					hint="Its OpenAI-compatible endpoint, for example http://192.168.1.20:11434/v1 for Ollama."
					bind:value={form.base_url}
					required
				/>
			{/if}
			<TextField
				id="p-key"
				label={form.provider === 'local' ? 'API key (optional)' : 'API key'}
				type="password"
				autocomplete="off"
				hint={info.settings?.has_key
					? 'A key is stored, encrypted. Leave empty to keep it.'
					: info.can_store_keys
						? 'Stored encrypted on this server and never shown again.'
						: 'This server cannot store keys until AXON_ENCRYPTION_KEY is set.'}
				bind:value={form.api_key}
				disabled={!info.can_store_keys}
			/>
			<details>
				<summary>Prices for your budget</summary>
				<p class="muted small">
					US dollars per million tokens. Leave empty for Axon's cautious default; a local model
					costs nothing.
				</p>
				<div class="grid">
					<TextField
						id="p-in"
						label="Input"
						type="number"
						step="any"
						min="0"
						bind:value={form.input_price}
					/>
					<TextField
						id="p-out"
						label="Output"
						type="number"
						step="any"
						min="0"
						bind:value={form.output_price}
					/>
				</div>
			</details>
			<div class="row">
				<Button type="submit" {busy}>Save provider</Button>
				{#if info.settings}
					<Button variant="secondary" onclick={test}>Test connection</Button>
					{#if info.settings.has_key}<Button variant="ghost" onclick={removeKey}>Remove key</Button
						>{/if}
					{#if info.server_default}<Button variant="ghost" onclick={useDefault}
							>Use server default</Button
						>{/if}
				{/if}
			</div>
			{#if result}
				<p class="result" class:bad={!result.ok} role="status">
					{result.ok ? '✓ ' : ''}{result.detail}
				</p>
			{/if}
		</form>
		<p class="muted small">
			Consumer chat subscriptions (Claude Pro or Max, ChatGPT Plus) cannot be used by other apps;
			providers only allow API access. On the server, an administrator can sign Axon in to Anthropic
			once with <code>ant auth login</code> so no key is stored at all.
		</p>
	{/if}
</section>

<style>
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6) var(--space-6);
		margin-bottom: var(--space-6);
	}
	@media (max-width: 30rem) {
		.card {
			padding: var(--space-4);
		}
	}
	h2 {
		font-size: var(--text-lg);
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
		max-width: 28rem;
	}
	.field label {
		font-weight: 600;
	}
	select {
		width: 100%;
		max-width: 100%;
		font: inherit;
		min-height: 2.75rem;
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(9rem, 1fr));
		column-gap: var(--space-4);
		max-width: 28rem;
	}
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		margin-top: var(--space-4);
	}
	details {
		margin-bottom: var(--space-2);
	}
	summary {
		cursor: pointer;
		color: var(--accent);
	}
	.small {
		font-size: var(--text-sm);
		overflow-wrap: anywhere;
	}
	.result {
		color: var(--ok);
	}
	.result.bad {
		color: var(--bad);
	}
	code {
		font-family: var(--font-mono);
	}
</style>
