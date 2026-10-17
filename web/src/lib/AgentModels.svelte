<script lang="ts">
	import { api, message } from '$lib/api';
	import type { paths } from '$lib/api-types';
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// The model each agent uses (EP-151): one for all of them, and any agent
	// its own. Blank means the default shown in grey.
	type Models =
		paths['/v1/me/agent-models']['get']['responses'][200]['content']['application/json'];
	let models = $state<Models | null>(null);
	let all = $state('');
	let own = $state<Record<string, string>>({});
	let error = $state('');
	let saving = $state(false);

	async function load() {
		try {
			models = await api<Models>('/v1/me/agent-models');
			all = models.all;
			own = Object.fromEntries(models.agents.map((a) => [a.agent, a.model]));
		} catch (err) {
			error = message(err);
		}
	}
	load();

	// Suggestions only: any model the provider serves can be typed.
	const known = [
		'claude-opus-5-5',
		'claude-sonnet-5-5',
		'claude-haiku-5-5',
		'claude-fable-5-1',
		'gpt-5',
		'gpt-5-mini',
		'mistral-large-latest',
		'gemini-2.5-pro',
		'deepseek-chat',
		'llama3.1:8b',
		'qwen2.5:14b'
	];

	async function save() {
		saving = true;
		try {
			models = await api<Models>('/v1/me/agent-models', {
				method: 'PUT',
				body: { models: { '*': all.trim(), ...own } }
			});
			toast(t('Models saved.'));
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			saving = false;
		}
	}
</script>

<section class="models" aria-labelledby="models-title">
	<h3 id="models-title">{t('Models')}</h3>
	<p class="muted small">
		{t(
			'One model for all agents, or one for an agent: a deep one for the Tutor, a quick one for summaries. The provider stays the one above; costs follow the model.'
		)}
	</p>
	<datalist id="known-models">
		{#each known as m (m)}<option value={m}></option>{/each}
	</datalist>
	{#if error}
		<Status kind="error" message={error} retry={load} />
	{:else if models}
		<div class="field">
			<label for="model-all">{t('All agents')}</label>
			<input
				id="model-all"
				list="known-models"
				maxlength="200"
				bind:value={all}
				placeholder={t('Each agent’s own default')}
			/>
		</div>
		<details>
			<summary>{t('A model for one agent')}</summary>
			<div class="grid">
				{#each models.agents as a (a.agent)}
					<label for="model-{a.agent}">{a.title}</label>
					<input
						id="model-{a.agent}"
						list="known-models"
						maxlength="200"
						bind:value={own[a.agent]}
						placeholder={all || a.default}
					/>
				{/each}
			</div>
		</details>
		<Button variant="secondary" busy={saving} onclick={save}>{t('Save the models')}</Button>
	{/if}
</section>

<style>
	.models {
		margin-top: var(--space-4);
	}
	h3 {
		margin: 0 0 var(--space-1);
		font-size: var(--text-base);
	}
	.grid {
		display: grid;
		grid-template-columns: minmax(8rem, 14rem) 1fr;
		gap: var(--space-2) var(--space-3);
		align-items: center;
		margin: var(--space-3) 0;
	}
	details {
		margin: var(--space-2) 0 var(--space-3);
	}
	summary {
		cursor: pointer;
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
