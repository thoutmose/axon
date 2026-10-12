<script lang="ts">
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

<svelte:head><title>Search “{q}” · Axon</title></svelte:head>

<h1>Search</h1>
<p class="muted">Notes matching “{q}”. Use quotes for a phrase and -word to exclude.</p>

{#if error}
	<Status kind="error" message={error} />
{:else if !hits}
	<Status kind="loading" message="Searching…" />
{:else if hits.length === 0}
	<Status kind="empty" message="No note matches. Try fewer or different words." />
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
