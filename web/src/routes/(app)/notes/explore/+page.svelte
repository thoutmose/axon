<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, message, type Schemas } from '$lib/api';
	import { monthLabel } from '$lib/calendar';
	import { forceLayout } from '$lib/overview';
	import Status from '$lib/ui/Status.svelte';

	type Exploration = Schemas['NoteExploration'];
	type Graph = Schemas['KnowledgeGraph'];
	type EdgeKind = Graph['edges'][number]['kind'];

	const keys = ['para', 'kind', 'tag', 'q', 'from', 'to', 'orphans'] as const;
	const query = $derived(
		new URLSearchParams(
			keys.flatMap((k) => {
				const v = page.url.searchParams.get(k);
				return v ? [[k, v]] : [];
			})
		).toString()
	);
	let ex = $state<Exploration | null>(null);
	let graph = $state<Graph | null>(null);
	let error = $state('');
	let text = $state(page.url.searchParams.get('q') ?? '');

	$effect(() => {
		const q = query;
		ex = null;
		graph = null;
		error = '';
		Promise.all([api<Exploration>(`/v1/notes/explore?${q}`), api<Graph>(`/v1/notes/graph?${q}`)])
			.then(([e, g]) => {
				ex = e;
				graph = g;
			})
			.catch((err) => (error = message(err)));
	});

	function set(key: string, value: string | null) {
		const u = new URL(page.url);
		if (value) u.searchParams.set(key, value);
		else u.searchParams.delete(key);
		goto(u, { keepFocus: true, noScroll: true, replaceState: true });
	}

	const paraNames: Record<string, string> = {
		inbox: 'Inbox',
		projects: 'Projects',
		areas: 'Areas',
		resources: 'Resources',
		archive: 'Archive',
		other: 'Zettelkasten & more'
	};
	const edgeNames: Record<EdgeKind, string> = {
		link: 'Links',
		entity: 'Shared names',
		keyword: 'Shared keywords',
		synonym: 'Synonyms',
		similar: 'Similar wording'
	};
	let shown = $state<Record<EdgeKind, boolean>>({
		link: true,
		entity: true,
		keyword: true,
		synonym: true,
		similar: true
	});

	const W = 1000;
	const H = 620;
	const edges = $derived((graph?.edges ?? []).filter((e) => shown[e.kind]));
	const placed = $derived(graph ? forceLayout(graph.nodes, edges, W, H) : []);
	const at = $derived(new Map(placed.map((p) => [p.id, p])));
	const degree = $derived.by(() => {
		const d = new Map<string, number>();
		for (const e of edges) {
			d.set(e.from, (d.get(e.from) ?? 0) + 1);
			d.set(e.to, (d.get(e.to) ?? 0) + 1);
		}
		return d;
	});
	let focus = $state<string | null>(null);
	const focusEdges = $derived(focus ? edges.filter((e) => e.from === focus || e.to === focus) : []);
	const title = (id: string) => graph?.nodes.find((n) => n.id === id)?.title ?? '';
	const peak = (list: { count: number }[]) => Math.max(1, ...list.map((c) => c.count));
</script>

<svelte:head><title>Explore notes · Axon</title></svelte:head>

<h1>Explore your knowledge</h1>
<p class="muted small">
	Filter by PARA, Zettelkasten type, tag, text, and date; see what you write about; and follow the
	knowledge graph. Connections come from your links and from local language analysis (keywords,
	names, synonyms, similar wording). Each one says why, and nothing leaves your Axon.
</p>

<form class="filters" role="search" onsubmit={(e) => (e.preventDefault(), set('q', text))}>
	<div class="f">
		<label for="ex-para">PARA</label>
		<select
			id="ex-para"
			value={page.url.searchParams.get('para') ?? ''}
			onchange={(e) => set('para', e.currentTarget.value)}
		>
			<option value="">All</option>
			{#each Object.entries(paraNames) as [k, v] (k)}<option value={k}>{v}</option>{/each}
		</select>
	</div>
	<div class="f">
		<label for="ex-kind">Type</label>
		<select
			id="ex-kind"
			value={page.url.searchParams.get('kind') ?? ''}
			onchange={(e) => set('kind', e.currentTarget.value)}
		>
			<option value="">All</option>
			{#each ['fleeting', 'literature', 'permanent', 'project', 'area', 'resource', 'meeting', 'daily', 'note'] as k (k)}<option
					value={k}>{k}</option
				>{/each}
		</select>
	</div>
	<div class="f">
		<label for="ex-tag">Tag</label>
		<input
			id="ex-tag"
			value={page.url.searchParams.get('tag') ?? ''}
			onchange={(e) => set('tag', e.currentTarget.value.replace(/^#/, ''))}
			placeholder="memory"
		/>
	</div>
	<div class="f">
		<label for="ex-from">From</label>
		<input
			id="ex-from"
			type="date"
			value={page.url.searchParams.get('from') ?? ''}
			onchange={(e) => set('from', e.currentTarget.value)}
		/>
	</div>
	<div class="f">
		<label for="ex-to">To</label>
		<input
			id="ex-to"
			type="date"
			value={page.url.searchParams.get('to') ?? ''}
			onchange={(e) => set('to', e.currentTarget.value)}
		/>
	</div>
	<label class="check"
		><input
			type="checkbox"
			checked={page.url.searchParams.get('orphans') === 'true'}
			onchange={(e) => set('orphans', e.currentTarget.checked ? 'true' : null)}
		/> Only unconnected notes</label
	>
	<div class="f grow">
		<label for="ex-text">Text</label>
		<input
			id="ex-text"
			bind:value={text}
			placeholder="Words in the title or body"
			maxlength="200"
		/>
	</div>
</form>

{#if error}
	<Status kind="error" message={error} />
{:else if !ex || !graph}
	<Status kind="loading" message="Reading your notes…" />
{:else if ex.stats.total === 0}
	<Status kind="empty" message="No note matches these filters." />
{:else}
	<section class="kpis" aria-label="Summary">
		<div><strong>{ex.stats.total}</strong><span class="muted small">notes</span></div>
		<div>
			<strong>{ex.stats.words.toLocaleString()}</strong><span class="muted small">words</span>
		</div>
		<div>
			<strong>{ex.stats.orphans}</strong><span class="muted small">unconnected</span>
			{#if ex.stats.orphans}<button class="link small" onclick={() => set('orphans', 'true')}
					>Show them</button
				>{/if}
		</div>
		<div><strong>{graph.edges.length}</strong><span class="muted small">connections</span></div>
	</section>
	{#if ex.truncated}<p class="muted small">
			Showing the newest {ex.notes.length} notes; narrow the filters to see older ones.
		</p>{/if}

	<section class="card" aria-labelledby="graph-title">
		<div class="row">
			<h2 id="graph-title">Knowledge graph</h2>
			<div class="toggles" role="group" aria-label="Kinds of connection">
				{#each Object.entries(edgeNames) as [k, v] (k)}
					<label class="edge-{k}"
						><input type="checkbox" bind:checked={shown[k as EdgeKind]} />
						<span class="swatch" aria-hidden="true"></span>{v} ({graph.edges.filter(
							(e) => e.kind === k
						).length})</label
					>
				{/each}
			</div>
		</div>
		<svg
			viewBox="0 0 {W} {H}"
			role="group"
			aria-label="Knowledge graph: {graph.nodes.length} notes, {edges.length} connections"
		>
			{#each edges as e, i (i)}
				{@const a = at.get(e.from)}
				{@const b = at.get(e.to)}
				{#if a && b}
					<line
						x1={a.x}
						y1={a.y}
						x2={b.x}
						y2={b.y}
						class="edge-{e.kind}"
						class:dim={focus && e.from !== focus && e.to !== focus}
						stroke-width={1 + e.weight * 1.5}><title>{e.why}</title></line
					>
				{/if}
			{/each}
			{#each placed as n (n.id)}
				<a
					href="/notes/{n.id}"
					aria-label="{n.title}, {paraNames[n.para]}, {degree.get(n.id) ?? 0} connections"
					onfocus={() => (focus = n.id)}
					onmouseenter={() => (focus = n.id)}
					onmouseleave={() => (focus = null)}
				>
					<circle
						cx={n.x}
						cy={n.y}
						r={5 + Math.min(10, (degree.get(n.id) ?? 0) * 1.2)}
						class="para-{n.para}"
						class:dim={focus &&
							focus !== n.id &&
							!focusEdges.some((e) => e.from === n.id || e.to === n.id)}
					/>
					{#if (degree.get(n.id) ?? 0) >= 3 || focus === n.id || placed.length <= 40}
						<text x={n.x + 12} y={n.y + 4}>{n.title.slice(0, 32)}</text>
					{/if}
				</a>
			{/each}
		</svg>
		<p class="legend small">
			{#each Object.entries(paraNames) as [k, v] (k)}<span class="para-{k}"
					><span class="dot" aria-hidden="true"></span>{v}</span
				>{/each}
		</p>
		{#if focus}
			<div class="why" role="status">
				<strong>{title(focus)}</strong>
				<ul>
					{#each focusEdges as e, i (i)}
						<li>
							<span class="tag edge-{e.kind}">{edgeNames[e.kind]}</span>
							{title(e.from === focus ? e.to : e.from)} — {e.why}
						</li>
					{:else}
						<li class="muted">No connection with these filters.</li>
					{/each}
				</ul>
			</div>
		{/if}
		<details>
			<summary>Show the connections as a table</summary>
			<table>
				<thead
					><tr><th scope="col">Note</th><th scope="col">Note</th><th scope="col">Why</th></tr
					></thead
				>
				<tbody>
					{#each edges.slice(0, 300) as e, i (i)}
						<tr
							><td><a href="/notes/{e.from}">{title(e.from)}</a></td><td
								><a href="/notes/{e.to}">{title(e.to)}</a></td
							><td>{e.why}</td></tr
						>
					{/each}
				</tbody>
			</table>
		</details>
	</section>

	<div class="charts">
		<section class="card" aria-labelledby="time-title">
			<h2 id="time-title">Notes over time</h2>
			<ul class="bars">
				{#each ex.stats.by_month.slice(-18) as m (m.label)}
					<li>
						<span class="label">{monthLabel(m.label)}</span>
						<span class="bar" style:width="{(m.count / peak(ex.stats.by_month)) * 100}%"></span>
						<span class="n">{m.count}</span>
					</li>
				{/each}
			</ul>
		</section>
		<section class="card" aria-labelledby="para-title">
			<h2 id="para-title">By PARA and type</h2>
			<ul class="bars">
				{#each ex.stats.by_para as c (c.label)}
					<li>
						<button class="label link" onclick={() => set('para', c.label)}
							>{paraNames[c.label] ?? c.label}</button
						>
						<span class="bar" style:width="{(c.count / peak(ex.stats.by_para)) * 100}%"></span>
						<span class="n">{c.count}</span>
					</li>
				{/each}
			</ul>
			<ul class="bars">
				{#each ex.stats.by_kind as c (c.label)}
					<li>
						<button class="label link" onclick={() => set('kind', c.label)}>{c.label}</button>
						<span class="bar alt" style:width="{(c.count / peak(ex.stats.by_kind)) * 100}%"></span>
						<span class="n">{c.count}</span>
					</li>
				{/each}
			</ul>
		</section>
		<section class="card" aria-labelledby="words-title">
			<h2 id="words-title">What you write about</h2>
			<p class="cloud">
				{#each ex.stats.keywords as k (k.label)}
					<button
						class="word"
						style:font-size="{0.85 + (k.count / peak(ex.stats.keywords)) * 0.9}rem"
						onclick={() => ((text = k.label), set('q', k.label))}>{k.label}</button
					>
				{/each}
			</p>
			{#if ex.stats.tags.length}
				<h3 class="small">Tags</h3>
				<p class="cloud">
					{#each ex.stats.tags as tg (tg.label)}
						<button class="tag" onclick={() => set('tag', tg.label)}>#{tg.label} {tg.count}</button>
					{/each}
				</p>
			{/if}
		</section>
	</div>
{/if}

<style>
	.small {
		font-size: var(--text-sm);
	}
	.filters {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
		align-items: end;
		margin: var(--space-4) 0;
	}
	.f {
		display: grid;
		gap: var(--space-1);
	}
	.f label {
		font-size: var(--text-sm);
		font-weight: 600;
	}
	.filters .check {
		display: flex;
		align-items: center;
		gap: var(--space-2);
		font-weight: 400;
		min-height: 2.5rem;
	}
	.filters .grow {
		flex: 1;
		min-width: 12rem;
	}
	.filters input:not([type='checkbox']),
	.filters select {
		font: inherit;
		min-height: 2.5rem;
		padding: var(--space-1) var(--space-3);
		border-radius: var(--radius);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
	}
	.kpis {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
		gap: var(--space-3);
		margin-bottom: var(--space-4);
	}
	.kpis div {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-3) var(--space-4);
		display: grid;
	}
	.kpis strong {
		font-size: var(--text-2xl);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4) var(--space-6);
		margin-bottom: var(--space-4);
		overflow-x: auto;
	}
	.row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-3);
		flex-wrap: wrap;
	}
	.toggles {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
		font-size: var(--text-sm);
	}
	.toggles label {
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}
	.swatch {
		width: 1rem;
		height: 3px;
		background: var(--edge);
		display: inline-block;
	}
	svg {
		width: 100%;
		height: auto;
		max-height: 70vh;
	}
	line {
		stroke: var(--edge);
		opacity: 0.55;
	}
	.edge-link {
		--edge: var(--accent);
	}
	.edge-entity {
		--edge: var(--chart-3);
	}
	.edge-keyword {
		--edge: var(--chart-2);
	}
	.edge-synonym {
		--edge: var(--ok);
	}
	.edge-similar {
		--edge: var(--muted);
	}
	line.edge-similar {
		stroke-dasharray: 4 4;
	}
	.dim {
		opacity: 0.12;
	}
	circle {
		fill: var(--dot);
		stroke: var(--surface-solid);
		stroke-width: 1.5;
	}
	.para-inbox {
		--dot: var(--muted);
	}
	.para-projects {
		--dot: var(--chart-3);
	}
	.para-areas {
		--dot: var(--chart-2);
	}
	.para-resources {
		--dot: var(--accent);
	}
	.para-archive {
		--dot: var(--border);
	}
	.para-other {
		--dot: var(--ok);
	}
	a:focus-visible circle {
		stroke: var(--fg);
		stroke-width: 3;
	}
	text {
		font-size: 12px;
		fill: var(--fg);
		paint-order: stroke;
		stroke: var(--surface-solid);
		stroke-width: 3px;
	}
	.legend {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
	}
	.legend .dot {
		display: inline-block;
		width: 0.7rem;
		height: 0.7rem;
		border-radius: 50%;
		background: var(--dot);
		margin-right: var(--space-1);
	}
	.why ul {
		margin: var(--space-2) 0;
		padding-left: var(--space-4);
		font-size: var(--text-sm);
	}
	.tag {
		font: inherit;
		font-size: var(--text-sm);
		background: var(--accent-soft);
		color: var(--fg);
		border: none;
		border-radius: 999px;
		padding: 0 var(--space-2);
		cursor: pointer;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		font-size: var(--text-sm);
	}
	th,
	td {
		text-align: left;
		padding: var(--space-1) var(--space-2);
		border-bottom: 1px solid var(--border);
	}
	.charts {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(20rem, 1fr));
		gap: var(--space-4);
	}
	.bars {
		list-style: none;
		padding: 0;
		margin: 0 0 var(--space-4);
		display: grid;
		gap: var(--space-1);
	}
	.bars li {
		display: grid;
		grid-template-columns: 9rem 1fr 2.5rem;
		align-items: center;
		gap: var(--space-2);
		font-size: var(--text-sm);
	}
	.bar {
		height: 0.6rem;
		border-radius: 999px;
		background: var(--accent);
	}
	.bar.alt {
		background: var(--chart-2);
	}
	.n {
		text-align: right;
		font-variant-numeric: tabular-nums;
	}
	.link {
		font: inherit;
		background: none;
		border: none;
		padding: 0;
		color: var(--accent);
		cursor: pointer;
		text-align: left;
	}
	.cloud {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2) var(--space-3);
		align-items: baseline;
	}
	.word {
		font-family: inherit;
		background: none;
		border: none;
		padding: 0;
		color: var(--fg);
		cursor: pointer;
	}
	.word:hover {
		color: var(--accent);
	}
</style>
