<script lang="ts">
	import { onMount } from 'svelte';
	import { api, ApiError, message, type Schemas } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import { emptyDay, hasContent, scales, type JournalFields } from '$lib/journal';
	import Scale from '$lib/Scale.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	// "How are you today?" on Today (EP-91): mood and energy in two taps,
	// merged into the day's journal entry so nothing else written is lost.
	let day = $state('');
	let form = $state<JournalFields | null>(null);
	let off = $state(false);
	let saved = $state(false);
	const s = $derived(scales());

	onMount(async () => {
		try {
			const r = await api<{ items: Schemas['JournalEntry'][]; today: string }>('/v1/journal');
			day = r.today;
			const e = r.items.find((x) => x.day === r.today);
			form = e ? { ...e, gratitude: e.gratitude } : emptyDay();
		} catch (err) {
			if (err instanceof ApiError && err.status === 403) off = true;
			else toast(message(err), 'error');
		}
	});

	async function save() {
		if (!form || !hasContent(form)) return;
		try {
			await api(`/v1/journal/${day}`, {
				method: 'PUT',
				body: { ...form, gratitude: form.gratitude.filter((g) => g.trim()) }
			});
			saved = true;
		} catch (err) {
			toast(message(err), 'error');
		}
	}
</script>

<section aria-labelledby="journal-today-title">
	<h2 id="journal-today-title">{t('How are you today?')}</h2>
	{#if off}
		<p class="muted">
			{t('The journal keeps mood and sleep, which are health data.')}
			<a href="/journal">{t('Turn it on in the journal')}</a>.
		</p>
	{:else if form}
		<Scale {...s.mood} bind:value={form.mood} onchange={save} />
		<Scale {...s.energy} bind:value={form.energy} onchange={save} />
		<p class="small">
			{#if saved}<span class="muted" role="status">{t('Saved in your journal.')}</span>{/if}
			<a href="/journal">{t('Open the journal →')}</a>
		</p>
	{/if}
</section>

<style>
	/* Today's own card look; its styles do not reach into components. */
	section {
		display: grid;
		gap: var(--space-3);
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6) var(--space-6);
		margin-bottom: var(--space-6);
	}
	.small {
		font-size: var(--text-sm);
		display: flex;
		gap: var(--space-3);
		margin: 0;
	}
</style>
