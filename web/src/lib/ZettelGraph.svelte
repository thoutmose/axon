<script lang="ts">
	import { t } from '$lib/i18n.svelte';
	import { api, message, type Schemas } from './api';
	import { forceLayout } from './overview';
	import GraphToolbar from './GraphToolbar.svelte';
	import { centerOn, fit, placeLabels, zoomAt, type View } from './graphView';
	import { panzoom } from './panzoom';
	import Status from './ui/Status.svelte';

	/** The Zettelkasten knowledge graph (EP-47): literature and permanent notes
	 * linked by combined NLP evidence, coloured by community, sized by PageRank. */
	type G = Schemas['ZettelGraph'];
	type ZNode = G['nodes'][number];

	let kind = $state('');
	let min = $state(0.08);
	let data = $state<G | null>(null);
	let error = $state('');
	let focused = $state<string | null>(null);
	let community = $state<number | null>(null);

	async function load() {
		error = '';
		try {
			const q = new URLSearchParams();
			if (kind) q.set('kind', kind);
			if (min > 0.08) q.set('min', String(min));
			data = await api<G>(`/v1/notes/zettelkasten-graph?${q}`);
		} catch (err) {
			error = message(err);
		}
	}
	$effect(() => {
		void kind;
		void min;
		load();
	});

	const W = 960;
	const H = 600;
	const palette = [
		'var(--chart-1)',
		'var(--chart-2)',
		'var(--chart-3)',
		'var(--ok)',
		'#c2410c',
		'#7c3aed',
		'#0f766e',
		'#b45309',
		'#be185d',
		'#4d7c0f'
	];
	const colour = (c: number) => palette[(c - 1) % palette.length];
	const placed = $derived(
		data
			? forceLayout(data.nodes, data.edges, W, H, 260)
			: ([] as (ZNode & { x: number; y: number })[])
	);
	const at = $derived(new Map(placed.map((p) => [p.id, p])));
	const byId = $derived(new Map((data?.nodes ?? []).map((n) => [n.id, n])));
	const near = $derived(
		focused
			? new Set(
					(data?.edges ?? [])
						.filter((e) => e.from === focused || e.to === focused)
						.flatMap((e) => [e.from, e.to])
				)
			: null
	);
	const dim = (id: string) =>
		(near !== null && !near.has(id)) ||
		(community !== null && byId.get(id)?.community !== community);
	const node = $derived(focused ? byId.get(focused) : undefined);
	const nodeEdges = $derived(
		focused
			? (data?.edges ?? [])
					.filter((e) => e.from === focused || e.to === focused)
					.sort((a, b) => b.score - a.score)
			: []
	);
	const other = (e: G['edges'][number]) => byId.get(e.from === focused ? e.to : e.from);
	const radius = (n: ZNode) => 5 + n.pagerank * 13;

	// Pan, zoom, find, and labels that never overlap (EP-122).
	let view = $state<View>({ x: 0, y: 0, k: 1 });
	let frame = $state<HTMLDivElement>();
	$effect(() => {
		view = fit(placed, W, H);
	});
	const labels = $derived(
		placeLabels(
			placed.map((n) => ({
				id: n.id,
				label: n.title,
				x: n.x,
				y: n.y,
				r: radius(n),
				rank: n.pagerank
			})),
			view,
			(s) => s.length * 7.4,
			new Set([...(focused ? [focused] : []), ...(near ?? [])])
		)
	);
	function find(id: string) {
		focused = id;
		const n = at.get(id);
		if (n) view = centerOn({ ...view, k: Math.max(view.k, 1.8) }, n.x, n.y, W, H);
	}
	const label = (c: number) => data?.communities.find((x) => x.id === c)?.label ?? t('Unconnected');
</script>

<section class="card zettel" aria-labelledby="zettel-title">
	<div class="head">
		<h2 id="zettel-title">{t('Knowledge graph')}</h2>
		<div class="controls">
			<label
				>{t('Notes')}
				<select bind:value={kind}>
					<option value="">{t('Literature and permanent')}</option>
					<option value="literature">{t('Literature')}</option>
					<option value="permanent">{t('Permanent')}</option>
				</select></label
			>
			<label
				>{t('Strength at least {value}', { value: min.toFixed(2) })}
				<input type="range" min="0.08" max="0.6" step="0.02" bind:value={min} /></label
			>
		</div>
	</div>
	<p class="small muted">
		{t(
			'Your literature and permanent notes, connected by what they say: shared key phrases weighted by rarity, synonyms (Open English WordNet), shared names, similar wording, and your own links. Colours are communities found by modularity (Louvain); larger notes are more central (PageRank); ◆ marks bridges between communities. Select a note to see why it connects.'
		)}
	</p>

	{#if error}
		<Status kind="error" message={error} retry={load} />
	{:else if !data}
		<Status kind="loading" message={t('Reading your notes…')} />
	{:else if data.stats.notes === 0}
		<Status
			kind="empty"
			message={t(
				'No literature or permanent notes yet. Give a note type: literature or type: permanent in its front matter, or start from those templates.'
			)}
		/>
	{:else}
		<p class="stats small" aria-label={t('Graph summary')}>
			{t(
				'{notes} notes · {edges} connections · {communities} communities · modularity {modularity} · {isolated} unconnected',
				{
					notes: data.stats.notes,
					edges: data.stats.edges,
					communities: data.stats.communities,
					modularity: data.stats.modularity,
					isolated: data.stats.isolated
				}
			)}
		</p>
		<GraphToolbar
			nodes={placed.map((n) => ({ id: n.id, label: n.title }))}
			onfind={find}
			onzoom={(f) => (view = zoomAt(view, f, W / 2, H / 2))}
			onfit={() => (view = fit(placed, W, H))}
			fullscreen={frame}
		/>
		<div class="layout">
			<div class="frame" bind:this={frame}>
				<svg
					use:panzoom={{ view: () => view, set: (v) => (view = v), width: W, height: H }}
					viewBox="0 0 {W} {H}"
					role="group"
					aria-label={t(
						'Knowledge graph: {notes} notes in {value} communities. The table below lists the connections.',
						{ notes: data.stats.notes, value: data.stats.communities }
					)}
				>
					<g transform="translate({view.x} {view.y}) scale({view.k})">
						{#each data.edges as e (e.from + e.to)}
							{@const a = at.get(e.from)}
							{@const b = at.get(e.to)}
							{#if a && b}
								<line
									x1={a.x}
									y1={a.y}
									x2={b.x}
									y2={b.y}
									stroke-width={0.5 + Math.min(e.score, 1) * 3.5}
									vector-effect="non-scaling-stroke"
									class:link={e.kinds.includes('link')}
									class:dim={dim(e.from) || dim(e.to)}
								/>
							{/if}
						{/each}
						{#each placed as n (n.id)}
							<g
								class="node"
								class:dim={dim(n.id)}
								class:focused={focused === n.id}
								transform="translate({n.x},{n.y})"
								role="button"
								tabindex="0"
								aria-label={t(
									n.bridge
										? '{title}, {kind}, community {community}, bridge, {degree} connections'
										: '{title}, {kind}, community {community}, {degree} connections',
									{ title: n.title, kind: n.kind, community: label(n.community), degree: n.degree }
								)}
								onclick={() => (focused = focused === n.id ? null : n.id)}
								onkeydown={(ev) => {
									if (ev.key === 'Enter' || ev.key === ' ') {
										ev.preventDefault();
										focused = focused === n.id ? null : n.id;
									}
								}}
							>
								{#if n.bridge}
									<rect
										x={-radius(n)}
										y={-radius(n)}
										width={radius(n) * 2}
										height={radius(n) * 2}
										transform="rotate(45)"
										fill={colour(n.community)}
									/>
								{:else}
									<circle r={radius(n)} fill={n.degree ? colour(n.community) : 'var(--muted)'} />
								{/if}
								{#if labels.has(n.id)}
									<text
										y={-radius(n) - 4 / view.k}
										style:font-size="{13 / view.k}px"
										style:stroke-width="{3 / view.k}px">{n.title}</text
									>
								{/if}
							</g>
						{/each}
					</g>
				</svg>
			</div>

			<aside aria-live="polite">
				{#if node}
					<h3><a href="/notes/{node.id}">{node.title}</a></h3>
					<p class="small muted">
						{t(
							node.bridge
								? '{kind} · {community} · centrality {pagerank} · bridge (participation {participation})'
								: '{kind} · {community} · centrality {pagerank}',
							{
								kind: node.kind,
								community: label(node.community),
								pagerank: node.pagerank,
								participation: node.participation
							}
						)}
					</p>
					{#if node.phrases.length}<p class="small">
							{t('Key phrases: {value}', { value: node.phrases.join(', ') })}
						</p>{/if}
					<h4>{t('Why it connects')}</h4>
					<ul class="why">
						{#each nodeEdges as e (e.from + e.to)}
							{@const o = other(e)}
							<li>
								<a href="/notes/{o?.id}">{o?.title}</a>
								<span class="muted small">{t('strength {score}', { score: e.score })}</span>
								<ul class="small">
									{#each e.reasons as r (r)}<li>{r}</li>{/each}
								</ul>
							</li>
						{:else}
							<li class="muted">{t('No connection above this strength.')}</li>
						{/each}
					</ul>
				{:else}
					<h3>{t('Communities')}</h3>
					<ul class="communities">
						{#each data.communities as c (c.id)}
							<li>
								<button
									class="link"
									aria-pressed={community === c.id}
									onclick={() => (community = community === c.id ? null : c.id)}
								>
									<i style:background={colour(c.id)} aria-hidden="true"></i>{c.label}
								</button>
								<span class="muted small"
									>{t('{size} notes · {value}', {
										size: c.size,
										value: c.core.map((n) => n.title).join(', ')
									})}</span
								>
							</li>
						{:else}
							<li class="muted">{t('No two notes are close enough to form a community yet.')}</li>
						{/each}
					</ul>
				{/if}
			</aside>
		</div>
		<details>
			<summary>{t('Show the connections as a table')}</summary>
			<table>
				<thead
					><tr
						><th scope="col">{t('Note')}</th><th scope="col">{t('Note')}</th><th scope="col"
							>{t('Strength')}</th
						><th scope="col">{t('Why')}</th></tr
					></thead
				>
				<tbody>
					{#each data.edges as e (e.from + e.to)}
						<tr>
							<td><a href="/notes/{e.from}">{byId.get(e.from)?.title}</a></td>
							<td><a href="/notes/{e.to}">{byId.get(e.to)?.title}</a></td>
							<td>{e.score}</td>
							<td>{e.reasons.join(' · ')}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</details>
	{/if}
</section>

<style>
	.small {
		font-size: var(--text-sm);
	}
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin: var(--space-4) 0;
	}
	.head {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-3);
	}
	.head h2 {
		margin: 0;
	}
	.controls {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-3);
		font-size: var(--text-sm);
	}
	.controls label {
		display: grid;
		gap: var(--space-1);
	}
	.layout {
		display: grid;
		grid-template-columns: minmax(0, 3fr) minmax(min(100%, 16rem), 1fr);
		gap: var(--space-4);
	}
	@media (max-width: 60rem) {
		.layout {
			grid-template-columns: 1fr;
		}
	}
	svg {
		cursor: grab;
		touch-action: none;
	}
	svg:active {
		cursor: grabbing;
	}
	.frame:fullscreen {
		display: grid;
		place-items: center;
		background: var(--surface-solid);
	}
	.frame:fullscreen svg {
		width: 100vw;
		height: 100vh;
	}
	svg {
		width: 100%;
		height: auto;
		background: var(--surface-solid, var(--surface));
		border: 1px solid var(--border);
		border-radius: var(--radius);
	}
	line {
		stroke: var(--muted);
		opacity: 0.55;
	}
	line.link {
		stroke: var(--fg);
	}
	.dim {
		opacity: 0.12;
	}
	.node {
		cursor: pointer;
	}
	.node:focus-visible circle,
	.node:focus-visible rect,
	.node.focused circle,
	.node.focused rect {
		stroke: var(--fg);
		stroke-width: 3;
	}
	text {
		font-size: 12px;
		fill: var(--fg);
		text-anchor: middle;
		paint-order: stroke;
		stroke: var(--surface);
		stroke-width: 3px;
		pointer-events: none;
	}
	aside h3 {
		margin-top: 0;
	}
	.why,
	.communities {
		list-style: none;
		padding: 0;
		display: grid;
		gap: var(--space-2);
	}
	.why ul {
		margin: var(--space-1) 0 0;
		padding-left: var(--space-4);
	}
	.communities span {
		display: block;
	}
	button.link {
		background: none;
		border: 0;
		padding: 0;
		cursor: pointer;
		font: inherit;
		color: var(--fg);
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		text-align: left;
	}
	button.link[aria-pressed='true'] {
		font-weight: 600;
	}
	.communities i {
		width: 0.75rem;
		height: 0.75rem;
		border-radius: 50%;
		flex: none;
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
		vertical-align: top;
	}
</style>
