<script lang="ts">
	import { api, message } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import { setHidden } from '$lib/menu.svelte';
	import { planFor, USES } from '$lib/uses';
	import Button from '$lib/ui/Button.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// The first question a new account is asked (EP-94): what Axon is for.
	// The answer sets the menu and Today, so the person starts with the
	// pages they use; Preferences changes it later.
	let { ondone }: { ondone: (sections: string[] | null) => void } = $props();

	let uses = $state<string[]>([]);
	let busy = $state(false);

	async function save(everything: boolean) {
		busy = true;
		try {
			const doc = await api<Record<string, unknown>>('/v1/me/settings');
			delete doc.ai;
			const plan = everything ? null : planFor(uses);
			const hidden = plan?.hidden ?? [];
			await api('/v1/me/settings', {
				method: 'PUT',
				body: {
					...doc,
					menu: { hidden, chosen: true },
					...(plan ? { today: { sections: plan.sections } } : {})
				}
			});
			setHidden(hidden);
			ondone(plan?.sections ?? null);
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			busy = false;
		}
	}
</script>

<section class="card welcome" aria-labelledby="welcome-title">
	<h2 id="welcome-title">{t('What do you want Axon for?')}</h2>
	<p class="muted">
		{t(
			'Pick what you use: the menu and Today will show just that. You can change it any time in Preferences.'
		)}
	</p>
	<fieldset>
		<legend class="visually-hidden">{t('What do you want Axon for?')}</legend>
		{#each USES as u (u.id)}
			<label class="use"
				><input type="checkbox" value={u.id} bind:group={uses} /> {t(u.label)}</label
			>
		{/each}
	</fieldset>
	<div class="row">
		<Button onclick={() => save(false)} disabled={!uses.length} {busy}>{t('Set up my Axon')}</Button
		>
		<Button variant="ghost" onclick={() => save(true)} {busy}>{t('Show me everything')}</Button>
	</div>
	<p class="small muted">
		{t('Coming from another app?')}
		<a href="/account/import">{t('Import from Todoist, TickTick, Notion, Goodreads, and more.')}</a>
	</p>
</section>

<style>
	.welcome {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-4);
	}
	fieldset {
		border: 0;
		padding: 0;
		margin: var(--space-3) 0;
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 14rem), 1fr));
		gap: var(--space-2);
	}
	.use {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		padding: var(--space-2) var(--space-3);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		cursor: pointer;
	}
	.use:has(input:checked) {
		border-color: var(--accent);
		background: var(--accent-soft);
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}
	.small {
		font-size: var(--text-sm);
	}
</style>
