<script lang="ts">
	import { page } from '$app/state';
	import { api, message, type Schemas } from '$lib/api';
	import { forceLayout } from '$lib/overview';
	import Status from '$lib/ui/Status.svelte';

	type A = Schemas['NoteCorpusAnalysis'];
	const kind = $derived(page.url.searchParams.get('kind') ?? '');
	let data = $state<A | null>(null);
	let error = $state('');

	async function load() {
		error = '';
		try {
			data = await api<A>(`/v1/notes/analysis${kind ? `?kind=${kind}` : ''}`);
		} catch (err) {
			error = message(err);
		}
	}
	$effect(() => {
		void kind;
		load();
	});

	const W = 900;
	const H = 520;
	const placed = $derived(
		data
			? forceLayout(data.concepts, data.edges, W, H)
			: ([] as (A['concepts'][number] & { x: number; y: number })[])
	);
	const at = $derived(new Map(placed.map((p) => [p.id, p])));
	const maxNotes = $derived(Math.max(1, ...(data?.concepts ?? []).map((c) => c.notes)));
	const textOf = (id: string) => data?.concepts.find((c) => c.id === id)?.text ?? id;
</script>

<svelte:head><title>Analysis · Notes · Axon</title></svelte:head>

<h1>Analysis of your literature and permanent notes</h1>
<p class="muted">
	All computed on this server, no AI: English and French stemming, key phrases (TextRank), topics
	from similar wording, and concepts linked when they appear together more than chance (NPMI) or are
	synonyms (Open English WordNet and Axon's French list).
</p>

<nav class="tabs" aria-label="Which notes">
	<a href="?" aria-current={kind === '' ? 'page' : undefined}>Both</a>
	<a href="?kind=literature" aria-current={kind === 'literature' ? 'page' : undefined}>Literature</a
	>
	<a href="?kind=permanent" aria-current={kind === 'permanent' ? 'page' : undefined}>Permanent</a>
</nav>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !data}
	<Status kind="loading" message="Reading your notes…" />
{:else if data.notes === 0}
	<Status
		kind="empty"
		message="No literature or permanent notes yet. Notes with type: literature or type: permanent in their front matter are analysed here."
	/>
{:else}
	<section class="kpis" aria-label="Summary">
		<p><strong>{data.notes}</strong> notes</p>
		<p><strong>{data.words.toLocaleString()}</strong> words</p>
		<p>
			<strong>{data.languages.en ?? 0}</strong> English · <strong>{data.languages.fr ?? 0}</strong> French
		</p>
		<p><strong>{data.ease}</strong>/100 average readability</p>
		<p>
			<strong>{data.topics.length}</strong> topics · <strong>{data.concepts.length}</strong> concepts
		</p>
	</section>
	{#if data.truncated}<p class="muted small">Only the most recent notes were analysed.</p>{/if}

	<section aria-labelledby="topics-title">
		<h2 id="topics-title">Topics</h2>
		{#if data.topics.length === 0}
			<p class="muted">No two notes are close enough to form a topic yet.</p>
		{:else}
			<ul class="topics">
				{#each data.topics as tp (tp.id)}
					<li class="card" id="topic-{tp.id}">
						<h3>{tp.label}</h3>
						<p class="small muted">{tp.terms.join(', ')}</p>
						<ul class="small">
							{#each tp.notes as n (n.id)}<li><a href="/notes/{n.id}">{n.title}</a></li>{/each}
						</ul>
					</li>
				{/each}
			</ul>
		{/if}
	</section>

	<section aria-labelledby="concepts-title">
		<h2 id="concepts-title">Concept graph</h2>
		{#if data.concepts.length === 0}
			<p class="muted">Concepts appear once a key phrase is in at least two notes.</p>
		{:else}
			<svg
				viewBox="0 0 {W} {H}"
				role="img"
				aria-label="Concept graph: {data.concepts.length} concepts, {data.edges
					.length} connections. The table below lists them."
			>
				{#each data.edges as e (e.from + e.to + e.relation)}
					{@const a = at.get(e.from)}
					{@const b = at.get(e.to)}
					{#if a && b}
						<line
							x1={a.x}
							y1={a.y}
							x2={b.x}
							y2={b.y}
							class={e.relation}
							stroke-width={1 + e.npmi * 3}
						/>
					{/if}
				{/each}
				{#each placed as c (c.id)}
					<g>
						<circle cx={c.x} cy={c.y} r={5 + (c.notes / maxNotes) * 9} />
						<text x={c.x} y={c.y - 12 - (c.notes / maxNotes) * 9}>{c.text}</text>
					</g>
				{/each}
			</svg>
			<p class="small muted">
				Larger dots are in more notes; solid lines appear together more than chance, thicker when
				stronger; dashed lines are synonyms.
			</p>
			<details>
				<summary>Show as a table</summary>
				<table>
					<thead
						><tr
							><th scope="col">Concept</th><th scope="col">Concept</th><th scope="col">Why</th></tr
						></thead
					>
					<tbody>
						{#each data.edges as e (e.from + e.to + e.relation)}
							<tr><td>{textOf(e.from)}</td><td>{textOf(e.to)}</td><td>{e.reason}</td></tr>
						{/each}
					</tbody>
				</table>
			</details>
		{/if}
	</section>
{/if}

<p class="small muted attribution">
	English synonyms: <a href="https://github.com/globalwordnet/english-wordnet"
		>Open English WordNet</a
	>
	(CC BY 4.0).
</p>

<style>
	.small {
		font-size: var(--text-sm);
	}
	.tabs {
		display: flex;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
	}
	.tabs a {
		padding: var(--space-2) var(--space-3);
		border-radius: var(--radius);
		color: var(--muted);
		text-decoration: none;
	}
	.tabs a[aria-current='page'] {
		background: var(--accent-soft);
		color: var(--fg);
		font-weight: 600;
	}
	.kpis {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-6);
	}
	.topics {
		list-style: none;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 16rem), 1fr));
		gap: var(--space-3);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
	}
	.card h3 {
		margin: 0 0 var(--space-1);
	}
	svg {
		width: 100%;
		height: auto;
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
	}
	line {
		stroke: var(--muted);
		opacity: 0.6;
	}
	line.synonym {
		stroke-dasharray: 4 4;
	}
	circle {
		fill: var(--accent);
	}
	text {
		font-size: 12px;
		fill: var(--fg);
		text-anchor: middle;
		paint-order: stroke;
		stroke: var(--surface);
		stroke-width: 3px;
	}
	table {
		width: 100%;
		border-collapse: collapse;
	}
	th,
	td {
		text-align: left;
		padding: var(--space-1) var(--space-2);
		border-bottom: 1px solid var(--border);
	}
	.attribution {
		margin-top: var(--space-8);
	}
</style>
