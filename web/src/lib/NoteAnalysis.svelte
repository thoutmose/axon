<script lang="ts">
	import { api, message, type Schemas } from './api';
	import Status from './ui/Status.svelte';

	/** Key phrases, readability, topic, and related notes (EP-46), loaded when opened. */
	let { id }: { id: string } = $props();
	let data = $state<Schemas['NoteAnalysis'] | null>(null);
	let error = $state('');
	let loaded = '';

	async function load() {
		if (loaded === id) return;
		loaded = id;
		data = null;
		error = '';
		try {
			data = await api<Schemas['NoteAnalysis']>(`/v1/notes/${id}/analysis`);
		} catch (err) {
			error = message(err);
		}
	}
	const ease = (n: number) =>
		n >= 70 ? 'easy to read' : n >= 50 ? 'plain' : n >= 30 ? 'fairly difficult' : 'difficult';
</script>

<details
	class="card analysis"
	ontoggle={(e) => (e.currentTarget as HTMLDetailsElement).open && load()}
>
	<summary><h2>Analysis</h2></summary>
	{#if error}
		<Status kind="error" message={error} />
	{:else if !data}
		<Status kind="loading" message="Reading the note…" />
	{:else}
		<p class="small">
			{data.readability.words} words · {data.readability.reading_minutes} min read · {data
				.readability.language === 'fr'
				? 'French'
				: 'English'} · readability {data.readability.ease}/100 ({ease(data.readability.ease)})
		</p>
		{#if data.phrases.length}
			<h3>Key phrases</h3>
			<ul class="chips" aria-label="Key phrases">
				{#each data.phrases as p (p.text)}
					<li style:opacity={0.55 + p.score * 0.45}>{p.text}</li>
				{/each}
			</ul>
		{/if}
		{#if data.topic}
			<p class="small">
				Topic: <a href="/notes/analysis#topic-{data.topic.id}">{data.topic.label}</a>, with {data
					.topic.notes.length - 1} other notes
			</p>
		{/if}
		{#if data.related.length}
			<h3>Related ideas</h3>
			<ul class="related">
				{#each data.related as r (r.id)}
					<li>
						<a href="/notes/{r.id}">{r.title}</a>
						<span class="muted small">{r.reasons.join(' · ')}</span>
					</li>
				{/each}
			</ul>
		{/if}
	{/if}
</details>

<style>
	summary {
		cursor: pointer;
	}
	summary h2 {
		display: inline;
		font-size: var(--text-lg);
	}
	.small {
		font-size: var(--text-sm);
	}
	.chips {
		list-style: none;
		padding: 0;
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
	}
	.chips li {
		background: var(--accent-soft);
		border-radius: 999px;
		padding: var(--space-1) var(--space-3);
		font-size: var(--text-sm);
	}
	.related {
		padding-left: var(--space-5);
		display: grid;
		gap: var(--space-2);
	}
	.related span {
		display: block;
	}
</style>
