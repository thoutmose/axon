<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { page } from '$app/state';
	import { api, message, type Schemas } from '$lib/api';
	import { highlight } from '$lib/notes';
	import Status from '$lib/ui/Status.svelte';

	type Hit = Schemas['SearchHit'];
	const q = $derived(page.url.searchParams.get('q') ?? '');
	let hits = $state<Hit[] | null>(null);
	let error = $state('');

	$effect(() => {
		const query = q;
		hits = null;
		error = '';
		if (!query) {
			hits = [];
			return;
		}
		api<{ items: Hit[] }>(`/v1/search?type=note&limit=50&q=${encodeURIComponent(query)}`)
			.then((r) => (hits = r.items))
			.catch((err) => (error = message(err)));
	});
</script>

<svelte:head><title>{q ? t('Search “{q}” · Axon', { q }) : t('Search · Axon')}</title></svelte:head>

<h1>{t('Search')}</h1>
{#if !q}
	<p class="muted">
		{t(
			'Type words in “Search notes” to find a note. Use quotes for a phrase and -word to exclude.'
		)}
	</p>
{:else}
	<p class="muted">
		{t('Notes matching “{q}”. Use quotes for a phrase and -word to exclude.', { q })}
	</p>
{/if}

{#if !q}
	<!-- Nothing asked yet: no results to show. -->
{:else if error}
	<Status kind="error" message={error} />
{:else if !hits}
	<Status kind="loading" message={t('Searching…')} />
{:else if hits.length === 0}
	<Status kind="empty" message={t('No note matches. Try fewer or different words.')} />
{:else}
	<ol class="hits">
		{#each hits as h (h.id)}
			<li class="card">
				<a href="/notes/{h.id}">{h.title}</a>
				<!-- highlight() escapes the snippet before marking matches. -->
				<p>{@html highlight(h.snippet)}</p>
			</li>
		{/each}
	</ol>
{/if}

<style>
	.hits {
		list-style: none;
		padding: 0;
		margin: 0;
		display: grid;
		gap: var(--space-3);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
	}
	.card a {
		font-weight: 600;
		text-decoration: none;
	}
	.card p {
		margin: var(--space-2) 0 0;
		color: var(--muted);
		font-size: var(--text-sm);
	}
	.card :global(mark) {
		background: var(--accent-soft);
		color: var(--fg);
		border-radius: 3px;
		padding: 0 2px;
	}
</style>
