<script lang="ts">
	import { api, message } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import { forceLayout } from '$lib/overview';
	import GraphToolbar from '$lib/GraphToolbar.svelte';
	import { centerOn, fit, placeLabels, zoomAt, type View } from '$lib/graphView';
	import { panzoom } from '$lib/panzoom';
	import Status from '$lib/ui/Status.svelte';
	import { lastDays, trendWord, type WatchGraph, type WatchTopic } from '$lib/watch';

	// The knowledge graph of the news (EP-111). It is built from the tech
	// watch only, and drawn as a constellation, so it never reads as the
	// Zettelkasten graph of your own notes; a ringed star is a concept your
	// literature and permanent notes share.
	type GNode = WatchGraph['nodes'][number];

	let days = $state(30);
	let topic = $state('');
	let fringe = $state(false);
	let min = $state(0.15);
	let topics = $state<WatchTopic[]>([]);
	let data = $state<WatchGraph | null>(null);
	let error = $state('');
	let hovered = $state<string | null>(null);
	let pinned = $state<string | null>(null);
	let community = $state<number | null>(null);

	async function load() {
		error = '';
		try {
			const q = new URLSearchParams(lastDays(days));
			if (topic) q.set('topic', topic);
			if (fringe) q.set('fringe', '1');
			if (min !== 0.15) q.set('min', String(min));
			data = await api<WatchGraph>(`/v1/watch/graph?${q}`);
			pinned = null;
		} catch (err) {
			error = message(err);
		}
	}
	$effect(() => {
		void [days, topic, fringe, min];
		load();
	});
	api<{ items: WatchTopic[] }>('/v1/watch/topics')
		.then((r) => (topics = r.items))
		.catch(() => {});

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
	const colour = (c: number) => (c > 0 ? palette[(c - 1) % palette.length] : 'var(--muted)');
	const placed = $derived(
		data
			? forceLayout(data.nodes, data.edges, W, H, 260)
			: ([] as (GNode & { x: number; y: number })[])
	);
	const at = $derived(new Map(placed.map((p) => [p.id, p])));
	const byId = $derived(new Map((data?.nodes ?? []).map((n) => [n.id, n])));
	const active = $derived(hovered ?? pinned);
	const near = $derived(
		active
			? new Set(
					(data?.edges ?? [])
						.filter((e) => e.from === active || e.to === active)
						.flatMap((e) => [e.from, e.to])
				)
			: null
	);
	const dim = (id: string) =>
		(near !== null && !near.has(id)) ||
		(community !== null && byId.get(id)?.community !== community);
	const radius = (n: GNode) => 4 + n.rank * 12;
	const tip = $derived(hovered ? at.get(hovered) : undefined);
	const card = $derived(pinned ? byId.get(pinned) : undefined);
	const rising = $derived(
		(data?.nodes ?? [])
			.filter((n) => trendWord(n.trend) === 'rising')
			.sort((a, b) => b.trend - a.trend)
	);
	const trendLabel = (n: GNode) =>
		({ rising: t('rising'), fading: t('fading'), steady: t('steady') })[trendWord(n.trend)];
	const sourcesLabel = (n: number) => (n === 1 ? t('1 source') : t('{n} sources', { n }));
	const pick = (id: string) => (pinned = pinned === id ? null : id);

	// Pan, zoom, find, and labels that never overlap (EP-122).
	let view = $state<View>({ x: 0, y: 0, k: 1 });
	let sky = $state<HTMLDivElement>();
	$effect(() => {
		view = fit(placed, W, H);
	});
	const labels = $derived(
		placeLabels(
			placed.map((n) => ({ ...n, r: radius(n) })),
			view,
			(s) => s.length * 7.4,
			new Set([...(active ? [active] : []), ...(near ?? [])])
		)
	);
	function find(id: string) {
		pinned = id;
		const n = at.get(id);
		if (n) view = centerOn({ ...view, k: Math.max(view.k, 1.8) }, n.x, n.y, W, H);
	}
	const screen = (gx: number, gy: number) => ({ x: view.x + view.k * gx, y: view.y + view.k * gy });
</script>

<svelte:head><title>{t('News graph')} · Axon</title></svelte:head>

<p><a href="/watch">← {t('Tech watch')}</a></p>
<header class="head">
	<div>
		<h1>{t('News graph')}</h1>
		<p class="muted">
			{t(
				'The concepts of your news and how often they appear together. Built from the tech watch only; your notes have their own graph.'
			)}
			<a href="/notes/analysis">{t('Your notes’ graph')}</a>
		</p>
	</div>
	<div class="controls">
		<label
			>{t('Period')}
			<select bind:value={days}>
				<option value={7}>{t('Last 7 days')}</option>
				<option value={30}>{t('Last 30 days')}</option>
				<option value={90}>{t('Last 90 days')}</option>
				<option value={365}>{t('Last year')}</option>
			</select></label
		>
		<label
			>{t('Topic')}
			<select bind:value={topic}>
				<option value="">{t('All topics')}</option>
				{#each topics as tp (tp.id)}<option value={tp.id}>{tp.name}</option>{/each}
			</select></label
		>
		<label
			>{t('Link strength at least {n}', { n: min.toFixed(2) })}
			<input type="range" min="0.05" max="0.6" step="0.05" bind:value={min} /></label
		>
		<label class="check"
			><input type="checkbox" bind:checked={fringe} /> {t('Show the fringe')}</label
		>
	</div>
</header>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !data}
	<Status kind="loading" message={t('Reading your news…')} />
{:else if data.nodes.length === 0}
	<Status
		kind="empty"
		message={t(
			'Not enough news in this period to see concepts. A concept needs at least two news items.'
		)}
	/>
{:else}
	<p class="small muted" aria-label={t('Graph summary')}>
		{t(
			'{items} news · {concepts} concepts · {edges} links · {communities} communities · {bridges} also in your notes',
			{
				items: data.stats.items,
				concepts: data.stats.concepts,
				edges: data.stats.edges,
				communities: data.stats.communities,
				bridges: data.stats.bridges
			}
		)}{#if data.stats.outliers && !fringe}
			· {t('{n} outliers set aside', { n: data.stats.outliers })}{/if}
	</p>
	{#if rising.length}
		<p class="rising">
			<strong>{t('Rising this week')}</strong>
			{#each rising.slice(0, 6) as n (n.id)}
				<button class="chip" onclick={() => pick(n.id)}>↑ {n.label}</button>
			{/each}
		</p>
	{/if}
	<GraphToolbar
		nodes={data.nodes}
		onfind={find}
		onzoom={(f) => (view = zoomAt(view, f, W / 2, H / 2))}
		onfit={() => (view = fit(placed, W, H))}
		fullscreen={sky}
	/>
	<div class="layout">
		<div class="sky" bind:this={sky}>
			<svg
				use:panzoom={{ view: () => view, set: (v) => (view = v), width: W, height: H }}
				viewBox="0 0 {W} {H}"
				role="group"
				aria-label={t('News graph: {n} concepts. The table below lists them.', {
					n: data.nodes.length
				})}
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
								stroke-width={0.5 + e.npmi * 2}
								vector-effect="non-scaling-stroke"
								class:dim={dim(e.from) || dim(e.to)}
							/>
						{/if}
					{/each}
					{#each placed as n (n.id)}
						<g
							class="node"
							class:dim={dim(n.id)}
							class:focused={active === n.id}
							transform="translate({n.x},{n.y})"
							role="button"
							tabindex="0"
							aria-label="{n.label}, {t('{n} news', { n: n.count })}, {trendLabel(n)}{n.in_notes
								.length
								? ', ' + t('also in your notes')
								: ''}"
							onmouseenter={() => (hovered = n.id)}
							onmouseleave={() => (hovered = null)}
							onfocus={() => (hovered = n.id)}
							onblur={() => (hovered = null)}
							onclick={() => pick(n.id)}
							onkeydown={(ev) => {
								if (ev.key === 'Enter' || ev.key === ' ') {
									ev.preventDefault();
									pick(n.id);
								}
							}}
						>
							<circle class="halo" r={radius(n) * 2.2} fill={colour(n.community)} />
							<circle class="hit" r={Math.max(12, radius(n) + 6)} />
							<circle r={radius(n)} fill={colour(n.community)} />
							{#if n.in_notes.length}<circle class="ring" r={radius(n) + 4} />{/if}
							{#if labels.has(n.id)}
								<text
									y={-radius(n) - 6 / view.k}
									style:font-size="{13 / view.k}px"
									style:stroke-width="{3 / view.k}px">{n.label}</text
								>
							{/if}
						</g>
					{/each}
				</g>
			</svg>
			{#if tip}
				<div
					class="tip"
					role="tooltip"
					style:left="{(screen(tip.x, tip.y).x / W) * 100}%"
					style:top="{(screen(tip.x, tip.y).y / H) * 100}%"
					class:below={screen(tip.x, tip.y).y < H / 3}
				>
					<strong>{tip.label}</strong>
					<span class="small muted"
						>{t('{n} news', { n: tip.count })} · {sourcesLabel(tip.sources)} · {trendLabel(
							tip
						)}</span
					>
					{#if tip.summary}<p>{tip.summary}</p>{/if}
					{#if tip.related.length}<p class="small muted">
							{t('With')}: {tip.related.join(', ')}
						</p>{/if}
					{#if tip.in_notes.length}<p class="small">
							◎ {t('In your notes')}: {tip.in_notes.map((x) => x.title).join(', ')}
						</p>{/if}
				</div>
			{/if}
		</div>

		<aside aria-live="polite">
			{#if card}
				<h2>{card.label}</h2>
				<p class="small muted">
					{t('{n} news', { n: card.count })} · {sourcesLabel(card.sources)} · {trendLabel(card)}
					· {t('first seen {date}', { date: new Date(card.first_seen).toLocaleDateString() })}
				</p>
				{#if card.summary}<p>{card.summary}</p>{/if}
				{#if card.topics.length}<p class="small">{t('Topics')}: {card.topics.join(', ')}</p>{/if}
				<h3>{t('News')}</h3>
				<ul>
					{#each card.items as it (it.id)}<li>
							<a href="/watch?item={it.id}">{it.title}</a>
						</li>{/each}
				</ul>
				{#if card.related.length}
					<h3>{t('Goes with')}</h3>
					<p class="small">{card.related.join(' · ')}</p>
				{/if}
				<h3>{t('In your notes')}</h3>
				<ul>
					{#each card.in_notes as nt (nt.id)}<li><a href="/notes/{nt.id}">{nt.title}</a></li>
					{:else}<li class="muted small">
							{t('No literature or permanent note shares it yet.')}
						</li>{/each}
				</ul>
				<p>
					<a href="/watch?q={encodeURIComponent(card.label)}&status=all">{t('All its news')} →</a>
				</p>
			{:else}
				<h2>{t('Communities')}</h2>
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
							<span class="muted small">{t('{n} concepts', { n: c.size })}</span>
						</li>
					{:else}
						<li class="muted">{t('No two concepts go together often enough yet.')}</li>
					{/each}
				</ul>
				<p class="small muted">
					{t(
						'Hover a star for its summary; select it to keep it here. Larger stars are more central; a ring means your notes share the concept.'
					)}
				</p>
			{/if}
		</aside>
	</div>
	<details>
		<summary>{t('Show the concepts as a table')}</summary>
		<table>
			<thead>
				<tr>
					<th scope="col">{t('Concept')}</th>
					<th scope="col">{t('News')}</th>
					<th scope="col">{t('Trend')}</th>
					<th scope="col">{t('Community')}</th>
					<th scope="col">{t('Summary')}</th>
				</tr>
			</thead>
			<tbody>
				{#each data.nodes as n (n.id)}
					<tr>
						<td>{n.label}</td>
						<td>{n.count}</td>
						<td>{trendLabel(n)}</td>
						<td>{data.communities.find((c) => c.id === n.community)?.label ?? '—'}</td>
						<td>{n.summary}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</details>
{/if}

<style>
	.small {
		font-size: var(--text-sm);
	}
	.head {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: end;
		gap: var(--space-3);
	}
	.head h1 {
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
	.controls .check {
		display: flex;
		align-items: center;
		gap: var(--space-1);
	}
	.rising {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
	}
	.chip {
		border: 1px solid var(--border);
		border-radius: 999px;
		background: var(--surface);
		color: var(--fg);
		padding: 0 var(--space-2);
		font: inherit;
		font-size: var(--text-sm);
		cursor: pointer;
	}
	.layout {
		display: grid;
		grid-template-columns: minmax(0, 3fr) minmax(min(100%, 16rem), 1fr);
		gap: var(--space-4);
		margin: var(--space-3) 0;
	}
	@media (max-width: 60rem) {
		.layout {
			grid-template-columns: 1fr;
		}
	}
	/* A night sky, distinct from the notes' graph: a deep field and a few
	   faint stars, in both themes. */
	.sky svg {
		cursor: grab;
		touch-action: none;
	}
	.sky svg:active {
		cursor: grabbing;
	}
	.sky:fullscreen {
		display: grid;
		place-items: center;
	}
	.sky:fullscreen svg {
		width: 100vw;
		height: 100vh;
	}
	.sky {
		position: relative;
		border-radius: var(--radius-lg);
		border: 1px solid var(--border);
		overflow: hidden;
		background:
			radial-gradient(1px 1px at 12% 18%, #fff8 50%, transparent 51%),
			radial-gradient(1px 1px at 72% 12%, #fff6 50%, transparent 51%),
			radial-gradient(1.5px 1.5px at 38% 82%, #fff5 50%, transparent 51%),
			radial-gradient(1px 1px at 88% 66%, #fff7 50%, transparent 51%),
			radial-gradient(ellipse at 30% 20%, #1e3a5f 0%, transparent 60%),
			radial-gradient(ellipse at 80% 90%, #3b1d4f 0%, transparent 55%), #0b1020;
		--fg-sky: #e8ecf6;
	}
	svg {
		display: block;
		width: 100%;
		height: auto;
	}
	line {
		stroke: #c7d2fe;
		opacity: 0.35;
	}
	.dim {
		opacity: 0.12;
	}
	.node {
		cursor: pointer;
		outline: none;
	}
	.halo {
		opacity: 0.18;
	}
	.hit {
		fill: transparent;
	}
	.ring {
		fill: none;
		stroke: var(--fg-sky);
		stroke-width: 1.5;
		stroke-dasharray: 3 2;
	}
	.node:focus-visible circle:not(.halo):not(.hit),
	.node.focused circle:not(.halo):not(.hit):not(.ring) {
		stroke: var(--fg-sky);
		stroke-width: 2;
	}
	text {
		font-size: 12px;
		fill: var(--fg-sky);
		text-anchor: middle;
		paint-order: stroke;
		stroke: #0b1020;
		stroke-width: 3px;
		pointer-events: none;
	}
	.tip {
		position: absolute;
		transform: translate(-50%, calc(-100% - 1rem));
		width: min(22rem, 80%);
		background: var(--surface-solid, var(--surface));
		color: var(--fg);
		border: 1px solid var(--border);
		border-radius: var(--radius);
		box-shadow: var(--shadow);
		padding: var(--space-2) var(--space-3);
		pointer-events: none;
		z-index: 2;
	}
	.tip.below {
		transform: translate(-50%, 1rem);
	}
	.tip strong {
		display: block;
	}
	.tip p {
		margin: var(--space-1) 0 0;
		font-size: var(--text-sm);
	}
	aside h2 {
		margin-top: 0;
	}
	aside h3 {
		font-size: var(--text-base);
		margin-bottom: var(--space-1);
	}
	aside ul {
		padding-left: var(--space-4);
	}
	.communities {
		list-style: none;
		padding: 0 !important;
		display: grid;
		gap: var(--space-2);
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
