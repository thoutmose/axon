<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { onMount } from 'svelte';
	import { templates } from '$lib/noteTemplates';
	import { keep, loadOutbox, outbox } from '$lib/outbox.svelte';
	import Button from '$lib/ui/Button.svelte';
	import TextField from '$lib/ui/TextField.svelte';

	let title = $state('');
	let saved = $state(false);
	onMount(loadOutbox);

	function save(kind: 'task' | 'note') {
		if (!title.trim()) return;
		if (kind === 'note') {
			const t = templates.find((x) => x.id === 'fleeting')!;
			const now = new Date();
			keep({ kind: 'note', title, path: t.path(title, now), content: t.content(title, now) });
		} else keep({ kind: 'task', title });
		title = '';
		saved = true;
	}
</script>

<svelte:head><title>{t('Offline · Axon')}</title></svelte:head>

<h1>{t('You are offline')}</h1>
<p class="muted">
	{t(
		'Axon needs a connection for your data, but a thought should never be lost. Capture it here: it stays on this device and goes to your inbox as soon as Axon is reachable again.'
	)}
</p>

<form
	onsubmit={(e) => {
		e.preventDefault();
		save('task');
	}}
>
	<TextField
		id="offline-title"
		label={t("What's on your mind?")}
		bind:value={title}
		maxlength={500}
		required
	/>
	<div class="row">
		<Button type="submit">{t('Keep as a task')}</Button>
		<Button variant="secondary" onclick={() => save('note')}>{t('Keep as a note')}</Button>
	</div>
</form>
{#if saved}<p role="status">{t('Kept on this device.')}</p>{/if}

{#if outbox.items.length}
	<h2>{t('Waiting to be sent')}</h2>
	<ul>
		{#each outbox.items as it (it.at)}
			<li>{it.kind === 'task' ? t('Task') : t('Note')}: {it.title}</li>
		{/each}
	</ul>
{/if}
<p><a href="/today" data-sveltekit-reload>{t('Try again')}</a></p>

<style>
	.row {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
		margin-top: var(--space-3);
	}
</style>
