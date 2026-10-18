<script lang="ts">
	import { api, message } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import GraphToolbar from '$lib/GraphToolbar.svelte';
	import { centerOn, fit, placeLabels, zoomAt, type View } from '$lib/graphView';
	import { kgLayout } from '$lib/kgLayout';
	import { panzoom } from '$lib/panzoom';
	import Status from '$lib/ui/Status.svelte';
	import { lastDays, trendWord, type WatchGraph, type WatchTopic } from '$lib/watch';

	// The knowledge graph of the news (EP-111, owner request): entities of an
	// ontology of the tech world, typed by their class, and the facts the news
	// state about them as subject–predicate–object triples, each with the news
	// and the sentence it was read in. The key phrases of the news that are
	// not entities stay as concepts. Built from the tech watch only; your
	// notes have their own graph.
	type GNode = WatchGraph['nodes'][number];
	type GEdge = WatchGraph['edges'][number];

	let days = $state(30);
	let topic = $state('');
	let fringe = $state(false);
	let topics = $state<WatchTopic[]>([]);
	let data = $state<WatchGraph | null>(null);
	let error = $state('');
	let hovered = $state<string | null>(null);
	let pinned = $state<string | null>(null);
	let limit = $state(90);

	async function load() {
		error = '';
		try {
			const q = new URLSearchParams(lastDays(days));
			if (topic) q.set('topic', topic);
			if (fringe) q.set('fringe', '1');
			const g = await api<WatchGraph>(`/v1/watch/graph?${q}`);
			// An ontology entity typed "concept" (DevOps, Data mesh) is not a key
			// phrase: it gets its own group, shown with the entities.
			for (const n of g.nodes)
				if (n.kind === 'entity' && n.type === 'concept') n.type = 'idea' as GNode['type'];
			data = g;
			pinned = null;
			// Concepts make way for entities once there are some.
			hiddenTypes = new Set(data.stats.entities > 0 ? ['concept'] : []);
		} catch (err) {
			error = message(err);
		}
	}
	$effect(() => {
		void [days, topic, fringe];
		load();
	});
	api<{ items: WatchTopic[] }>('/v1/watch/topics')
		.then((r) => (topics = r.items))
		.catch(() => {});

	// Types of node, each with its colour and its own pole in the layout.
	const types = $derived([
		{ id: 'technology', label: t('Technologies'), colour: '#60a5fa' },
		{ id: 'database', label: t('Databases'), colour: '#34d399' },
		{ id: 'organization', label: t('Organizations'), colour: '#fbbf24' },
		{ id: 'cloud_service', label: t('Cloud services'), colour: '#a78bfa' },
		{ id: 'language', label: t('Languages'), colour: '#f472b6' },
		{ id: 'standard', label: t('Standards'), colour: '#2dd4bf' },
		{ id: 'vulnerability', label: t('Vulnerabilities'), colour: '#f87171' },
		{ id: 'person', label: t('People'), colour: '#fb923c' },
		{ id: 'product', label: t('Products'), colour: '#38bdf8' },
		{ id: 'event', label: t('Events'), colour: '#e879f9' },
		{ id: 'location', label: t('Places'), colour: '#a3e635' },
		{ id: 'dataset', label: t('Datasets'), colour: '#facc15' },
		{ id: 'publication', label: t('Publications'), colour: '#d6d3d1' },
		{ id: 'idea', label: t('Ideas and practices'), colour: '#94a3b8' },
		{ id: 'concept', label: t('Concepts'), colour: '#cbd5e1' },
		{ id: 'class', label: t('Classes (ontology)'), colour: '#e2e8f0' }
	]);
	const colourOf = (type: string) => types.find((x) => x.id === type)?.colour ?? '#cbd5e1';
	let hiddenTypes = $state(new Set<string>());
	function toggleType(id: string) {
		const next = new Set(hiddenTypes);
		if (next.has(id)) next.delete(id);
		else next.add(id);
		hiddenTypes = next;
	}

	// Predicates, grouped as the legend shows them.
	const predicateNames = $derived<Record<string, string>>({
		is_a: t('is a'),
		developed_by: t('developed by'),
		fork_of: t('fork of'),
		acquires: t('acquires'),
		releases: t('releases'),
		integrates_with: t('integrates with'),
		uses: t('uses'),
		alternative_to: t('alternative to'),
		migrates_to: t('migrates to'),
		partners_with: t('partners with'),
		invests_in: t('invests in'),
		deprecates: t('deprecates'),
		affects: t('affects'),
		part_of: t('part of'),
		mentioned_with: t('mentioned with'),
		related_to: t('goes with'),
		develops: t('develops'),
		announces: t('announces'),
		built_on: t('built on'),
		supports: t('supports'),
		competes_with: t('competes with'),
		outperforms: t('outperforms'),
		funds: t('funds'),
		fixes: t('fixes'),
		version_of: t('version of'),
		founded: t('founded'),
		leads: t('leads'),
		works_at: t('works at'),
		hires: t('hires'),
		located_in: t('located in'),
		regulates: t('regulates'),
		adopts: t('adopts')
	});
	const groupOf = (p: string) =>
		p === 'is_a' || p === 'developed_by' || p === 'fork_of'
			? 'ontology'
			: p === 'mentioned_with'
				? 'together'
				: p === 'related_to'
					? 'concepts'
					: 'facts';
	const groupColour: Record<string, string> = {
		facts: '#fde68a',
		ontology: '#94a3b8',
		together: '#64748b',
		concepts: '#c7d2fe'
	};
	let shownGroups = $state({ facts: true, ontology: true, together: false, concepts: true });

	// The nodes shown: the types on, the most central first, up to the limit;
	// a class only with a member shown.
	const visible = $derived.by(() => {
		if (!data) return { nodes: [] as GNode[], edges: [] as GEdge[] };
		const entities = data.nodes
			.filter((n) => n.kind !== 'class' && !hiddenTypes.has(n.type))
			.sort((a, b) => b.rank - a.rank || b.count - a.count)
			.slice(0, limit);
		const ids = new Set(entities.map((n) => n.id));
		const classes = hiddenTypes.has('class')
			? []
			: data.nodes.filter(
					(c) =>
						c.kind === 'class' &&
						data!.edges.some((e) => e.predicate === 'is_a' && e.to === c.id && ids.has(e.from))
				);
		for (const c of classes) ids.add(c.id);
		const edges = data.edges.filter(
			(e) =>
				ids.has(e.from) &&
				ids.has(e.to) &&
				shownGroups[groupOf(e.predicate) as keyof typeof shownGroups]
		);
		return { nodes: [...entities, ...classes], edges };
	});

	const layout = $derived(kgLayout(visible.nodes, visible.edges));
	// The frame stays the size of the screen; the view fits the layout's
	// larger space into it, so labels keep their size (13 px).
	const W = 960;
	const H = 600;
	const at = $derived(new Map(layout.placed.map((p) => [p.id, p])));
	const byId = $derived(new Map((data?.nodes ?? []).map((n) => [n.id, n])));
	const active = $derived(hovered ?? pinned);
	const near = $derived(
		active
			? new Set(
					visible.edges
						.filter((e) => e.from === active || e.to === active)
						.flatMap((e) => [e.from, e.to])
				)
			: null
	);
	const dim = (id: string) => near !== null && !near.has(id) && id !== active;
	const touches = (e: GEdge) => active !== null && (e.from === active || e.to === active);
	const pick = (id: string) => (pinned = pinned === id ? null : id);
	const card = $derived(pinned ? byId.get(pinned) : undefined);
	const tip = $derived(hovered && hovered !== pinned ? at.get(hovered) : undefined);

	// A fact's facts: what the selected node states and what is stated of it.
	const factsOf = (id: string) => {
		const all = (data?.edges ?? []).filter((e) => e.from === id || e.to === id);
		return {
			out: all.filter(
				(e) => e.from === id && e.predicate !== 'mentioned_with' && e.predicate !== 'related_to'
			),
			in: all.filter(
				(e) => e.to === id && e.predicate !== 'mentioned_with' && e.predicate !== 'related_to'
			),
			with: all.filter((e) => e.predicate === 'mentioned_with' || e.predicate === 'related_to')
		};
	};
	const label = (id: string) => byId.get(id)?.label ?? id;
	const typeName = (type: string) => types.find((x) => x.id === type)?.label ?? type;
	const topFacts = $derived(
		(data?.edges ?? [])
			.filter((e) => groupOf(e.predicate) === 'facts')
			.sort((a, b) => b.together - a.together)
			.slice(0, 12)
	);
	const rising = $derived(
		(data?.nodes ?? [])
			.filter((n) => n.kind !== 'class' && trendWord(n.trend) === 'rising')
			.sort((a, b) => b.trend - a.trend)
	);
	const trendLabel = (n: GNode) =>
		({ rising: t('rising'), fading: t('fading'), steady: t('steady') })[trendWord(n.trend)];
	const sourcesLabel = (n: number) => (n === 1 ? t('1 source') : t('{n} sources', { n }));

	// Pan, zoom, find, and labels that never overlap (EP-122).
	let view = $state<View>({ x: 0, y: 0, k: 1 });
	let sky = $state<HTMLDivElement>();
	$effect(() => {
		view = fit(layout.placed, W, H);
	});
	const labels = $derived(
		placeLabels(
			layout.placed,
			view,
			(s) => s.length * 7.2,
			new Set([...(active ? [active] : []), ...(near ?? [])])
		)
	);
	function find(id: string) {
		if (!at.has(id)) {
			const n = byId.get(id);
			if (n && hiddenTypes.has(n.type)) toggleType(n.type);
			limit = Math.max(limit, 250);
		}
		pinned = id;
		queueMicrotask(() => {
			const n = at.get(id);
			if (n) view = centerOn({ ...view, k: Math.max(view.k, 1.6) }, n.x, n.y, W, H);
		});
	}
	const screen = (gx: number, gy: number) => ({ x: view.x + view.k * gx, y: view.y + view.k * gy });
	const arrow = (group: string) => 'url(#arrow-' + group + ')';
	// An edge stops at the rim of its target, where its arrow points.
	const trim = (e: GEdge) => {
		const a = at.get(e.from)!;
		const b = at.get(e.to)!;
		const dx = b.x - a.x;
		const dy = b.y - a.y;
		const d = Math.hypot(dx, dy) || 1;
		return { x1: a.x, y1: a.y, x2: b.x - (dx / d) * (b.r + 3), y2: b.y - (dy / d) * (b.r + 3) };
	};
</script>

<svelte:head><title>{t('Knowledge graph')} · Axon</title></svelte:head>

<p><a href="/watch">← {t('Tech watch')}</a></p>
<header class="head">
	<div>
		<h1>{t('Knowledge graph')}</h1>
		<p class="muted">
			{t(
				'The entities of your news (technologies, organizations, databases, standards…), each an instance of a class, and the facts the news state about them, with the news they were read in. Built from the tech watch only; your notes have their own graph.'
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
			>{t('Show the {n} most central', { n: limit })}
			<input type="range" min="20" max="250" step="10" bind:value={limit} /></label
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
			'Not enough news in this period to see entities. A concept needs at least two news items.'
		)}
	/>
{:else}
	<p class="small muted" aria-label={t('Graph summary')}>
		{t(
			'{items} news · {entities} entities in {classes} classes · {facts} facts read in the news · {concepts} concepts · {communities} communities',
			{
				items: data.stats.items,
				entities: data.stats.entities,
				classes: data.stats.classes,
				facts: data.stats.facts,
				concepts: data.stats.concepts,
				communities: data.stats.communities
			}
		)}{#if data.stats.outliers && !fringe}
			· {t('{n} outliers set aside', { n: data.stats.outliers })}{/if}
	</p>
	<div class="legend" role="group" aria-label={t('Types')}>
		{#each types as ty (ty.id)}
			{#if data.nodes.some((n) => n.type === ty.id)}
				<button
					type="button"
					class="chip"
					aria-pressed={!hiddenTypes.has(ty.id)}
					onclick={() => toggleType(ty.id)}
					><i style:background={ty.colour} class:square={ty.id === 'class'}></i>{ty.label}
					<span class="muted">{data.nodes.filter((n) => n.type === ty.id).length}</span></button
				>
			{/if}
		{/each}
	</div>
	<div class="legend" role="group" aria-label={t('Links')}>
		{#each [['facts', t('Facts from the news')], ['ontology', t('Ontology: is a, developed by, fork of')], ['together', t('Mentioned together')], ['concepts', t('Concepts that go together')]] as [g, name] (g)}
			<label class="chip"
				><input type="checkbox" bind:checked={shownGroups[g as keyof typeof shownGroups]} /><i
					class="line"
					style:background={groupColour[g]}
				></i>{name}</label
			>
		{/each}
	</div>
	{#if rising.length}
		<p class="rising">
			<strong>{t('Rising this week')}</strong>
			{#each rising.slice(0, 6) as n (n.id)}
				<button class="chip" onclick={() => find(n.id)}>↑ {n.label}</button>
			{/each}
		</p>
	{/if}
	<GraphToolbar
		nodes={data.nodes}
		onfind={find}
		onzoom={(f) => (view = zoomAt(view, f, W / 2, H / 2))}
		onfit={() => (view = fit(layout.placed, W, H))}
		fullscreen={sky}
	/>
	<div class="layout">
		<div class="sky" bind:this={sky}>
			<svg
				use:panzoom={{ view: () => view, set: (v) => (view = v), width: W, height: H }}
				viewBox="0 0 {W} {H}"
				role="group"
				aria-label={t('Knowledge graph: {n} nodes. The tables below list them and their facts.', {
					n: layout.placed.length
				})}
			>
				<defs>
					{#each Object.entries(groupColour) as [g, c] (g)}
						<marker
							id="arrow-{g}"
							viewBox="0 0 10 10"
							refX="9"
							refY="5"
							markerWidth="7"
							markerHeight="7"
							orient="auto-start-reverse"><path d="M0 0L10 5L0 10z" fill={c} /></marker
						>
					{/each}
				</defs>
				<g transform="translate({view.x} {view.y}) scale({view.k})">
					{#each visible.edges as e (e.from + e.predicate + e.to)}
						{#if at.has(e.from) && at.has(e.to)}
							{@const l = trim(e)}
							{@const g = groupOf(e.predicate)}
							<line
								{...l}
								stroke={groupColour[g]}
								stroke-width={g === 'facts' ? 1.2 + Math.min(e.together, 6) * 0.35 : 0.8}
								stroke-dasharray={e.predicate === 'is_a' ? '3 3' : undefined}
								marker-end={e.directed ? arrow(g) : undefined}
								vector-effect="non-scaling-stroke"
								class="edge {g}"
								class:dim={active !== null && !touches(e)}
								class:lit={touches(e)}
							/>
							{#if touches(e) && g !== 'together'}
								<text
									class="predicate"
									x={(l.x1 + l.x2) / 2}
									y={(l.y1 + l.y2) / 2}
									style:font-size="{11 / view.k}px"
									style:stroke-width="{3 / view.k}px">{predicateNames[e.predicate]}</text
								>
							{/if}
						{/if}
					{/each}
					{#each layout.placed as n (n.id)}
						<g
							class="node {n.kind}"
							class:dim={dim(n.id)}
							class:focused={active === n.id}
							transform="translate({n.x},{n.y})"
							role="button"
							tabindex="0"
							aria-label="{n.label}, {typeName(n.type)}{n.class ? ', ' + n.class : ''}{n.count
								? ', ' + t('{n} news', { n: n.count })
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
							<circle class="hit" r={Math.max(12, n.r + 6)} />
							{#if n.kind === 'class'}
								<rect x={-n.r} y={-n.r} width={n.r * 2} height={n.r * 2} rx="2" />
							{:else}
								<circle class="halo" r={n.r * 2} fill={colourOf(n.type)} />
								<circle r={n.r} fill={colourOf(n.type)} />
								{#if n.in_notes.length}<circle class="ring" r={n.r + 4} />{/if}
							{/if}
							{#if labels.has(n.id) || (n.kind === 'class' && view.k > 0.9)}
								<text
									class:cls={n.kind === 'class'}
									y={-n.r - 6 / view.k}
									style:font-size="{(n.kind === 'class' ? 11 : 13) / view.k}px"
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
						>{typeName(tip.type)}{tip.class ? ' · ' + tip.class : ''}{tip.count
							? ' · ' + t('{n} news', { n: tip.count })
							: ''}</span
					>
					{#if tip.summary}<p>{tip.summary}</p>{/if}
				</div>
			{/if}
		</div>

		<aside aria-live="polite">
			{#if card}
				{@const f = factsOf(card.id)}
				<h2>{card.label}</h2>
				<p class="small muted">
					<i class="dot" style:background={colourOf(card.type)}></i>{typeName(
						card.type
					)}{#if card.class}
						· {t('is a')} <strong>{card.class}</strong>{/if}
					{#if card.count}· {t('{n} news', { n: card.count })} · {sourcesLabel(card.sources)} · {trendLabel(
							card
						)}{/if}
				</p>
				{#if card.summary}<p>{card.summary}</p>{/if}
				{#if f.out.length || f.in.length}
					<h3>{t('Facts')}</h3>
					<ul class="facts">
						{#each [...f.out, ...f.in] as e (e.from + e.predicate + e.to)}
							<li>
								<button class="link" onclick={() => find(e.from)}>{label(e.from)}</button>
								<span class="pred">{predicateNames[e.predicate]}</span>
								<button class="link" onclick={() => find(e.to)}>{label(e.to)}</button>
								{#if e.together}<span class="muted small">· {t('{n} news', { n: e.together })}</span
									>{/if}
								{#each e.evidence as ev (ev.id)}
									<blockquote class="small">
										“{ev.sentence}” — <a href="/watch?item={ev.id}">{ev.title}</a>
									</blockquote>
								{/each}
							</li>
						{/each}
					</ul>
				{/if}
				{#if f.with.length}
					<h3>{t('Mentioned with')}</h3>
					<p class="small">
						{#each f.with as e, i (e.from + e.to)}{#if i},
							{/if}<button class="link" onclick={() => find(e.from === card.id ? e.to : e.from)}
								>{label(e.from === card.id ? e.to : e.from)}</button
							>{/each}
					</p>
				{/if}
				{#if card.items.length}
					<h3>{t('News')}</h3>
					<ul>
						{#each card.items as it (it.id)}<li>
								<a href="/watch?item={it.id}">{it.title}</a>
							</li>{/each}
					</ul>
				{/if}
				{#if card.in_notes.length}
					<h3>{t('In your notes')}</h3>
					<ul>
						{#each card.in_notes as nt (nt.id)}<li>
								<a href="/notes/{nt.id}">{nt.title}</a>
							</li>{/each}
					</ul>
				{/if}
				{#if card.kind !== 'class'}
					<p>
						<a href="/watch?q={encodeURIComponent(card.label)}&status=all">{t('All its news')} →</a>
					</p>
				{/if}
			{:else}
				<h2>{t('Facts read in the news')}</h2>
				<ul class="facts">
					{#each topFacts as e (e.from + e.predicate + e.to)}
						<li>
							<button class="link" onclick={() => find(e.from)}>{label(e.from)}</button>
							<span class="pred">{predicateNames[e.predicate]}</span>
							<button class="link" onclick={() => find(e.to)}>{label(e.to)}</button>
							<span class="muted small">· {t('{n} news', { n: e.together })}</span>
						</li>
					{:else}
						<li class="muted">{t('No fact read yet: entities are only mentioned together.')}</li>
					{/each}
				</ul>
				<p class="small muted">
					{t(
						'Select a node for its facts and the sentences they were read in. Arrows point from subject to object; dashed lines join an entity to its class. Larger nodes are more central; a ring means your notes share it.'
					)}
				</p>
			{/if}
		</aside>
	</div>
	<details>
		<summary>{t('Show the facts as a table')}</summary>
		<table>
			<thead>
				<tr>
					<th scope="col">{t('Subject')}</th>
					<th scope="col">{t('Predicate')}</th>
					<th scope="col">{t('Object')}</th>
					<th scope="col">{t('News')}</th>
					<th scope="col">{t('Read in')}</th>
				</tr>
			</thead>
			<tbody>
				{#each data.edges.filter((e) => groupOf(e.predicate) !== 'together' && e.predicate !== 'related_to') as e (e.from + e.predicate + e.to)}
					<tr>
						<td>{label(e.from)}</td>
						<td>{predicateNames[e.predicate]}</td>
						<td>{label(e.to)}</td>
						<td>{e.together || '—'}</td>
						<td
							>{#each e.evidence as ev (ev.id)}<a href="/watch?item={ev.id}">{ev.title}</a><br
								/>{:else}{t('The ontology')}{/each}</td
						>
					</tr>
				{/each}
			</tbody>
		</table>
	</details>
	<details>
		<summary>{t('Show the entities and concepts as a table')}</summary>
		<table>
			<thead>
				<tr>
					<th scope="col">{t('Name')}</th>
					<th scope="col">{t('Type')}</th>
					<th scope="col">{t('Class')}</th>
					<th scope="col">{t('News')}</th>
					<th scope="col">{t('Trend')}</th>
				</tr>
			</thead>
			<tbody>
				{#each data.nodes.filter((n) => n.kind !== 'class') as n (n.id)}
					<tr>
						<td>{n.label}</td>
						<td>{typeName(n.type)}</td>
						<td>{n.class || '—'}</td>
						<td>{n.count}</td>
						<td>{trendLabel(n)}</td>
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
	.legend,
	.rising {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
		margin: var(--space-2) 0;
	}
	.chip {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		border: 1px solid var(--border);
		border-radius: 999px;
		background: var(--surface);
		color: var(--fg);
		padding: 0 var(--space-2);
		font: inherit;
		font-size: var(--text-sm);
		cursor: pointer;
	}
	.chip[aria-pressed='false'] {
		opacity: 0.5;
		text-decoration: line-through;
	}
	.chip i,
	.dot {
		display: inline-block;
		width: 0.7rem;
		height: 0.7rem;
		border-radius: 50%;
		margin-right: var(--space-1);
		vertical-align: middle;
	}
	.chip i.square {
		border-radius: 2px;
		border: 1px solid var(--muted);
	}
	.chip i.line {
		height: 3px;
		width: 1rem;
		border-radius: 2px;
	}
	.layout {
		display: grid;
		grid-template-columns: minmax(0, 3fr) minmax(min(100%, 18rem), 1fr);
		gap: var(--space-4);
		margin: var(--space-3) 0;
	}
	@media (max-width: 60rem) {
		.layout {
			grid-template-columns: 1fr;
		}
	}
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
			radial-gradient(ellipse at 30% 20%, #1e3a5f 0%, transparent 60%),
			radial-gradient(ellipse at 80% 90%, #3b1d4f 0%, transparent 55%), #0b1020;
		--fg-sky: #e8ecf6;
	}
	svg {
		display: block;
		width: 100%;
		height: auto;
		max-height: 80vh;
	}
	.edge {
		opacity: 0.45;
	}
	.edge.together {
		opacity: 0.25;
	}
	.edge.lit {
		opacity: 1;
	}
	.dim {
		opacity: 0.08;
	}
	.node {
		cursor: pointer;
		outline: none;
	}
	.halo {
		opacity: 0.16;
	}
	.hit {
		fill: transparent;
	}
	.node.class rect {
		fill: #0b1020;
		stroke: #e2e8f0;
		stroke-width: 1.2;
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
		fill: var(--fg-sky);
		text-anchor: middle;
		paint-order: stroke;
		stroke: #0b1020;
		pointer-events: none;
	}
	text.cls {
		font-style: italic;
		fill: #cbd5e1;
	}
	text.predicate {
		fill: #fde68a;
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
	.facts {
		list-style: none;
		padding: 0 !important;
		display: grid;
		gap: var(--space-2);
	}
	.pred {
		font-style: italic;
		color: var(--muted);
		margin: 0 var(--space-1);
	}
	blockquote {
		margin: var(--space-1) 0 0 var(--space-2);
		padding-left: var(--space-2);
		border-left: 2px solid var(--border);
		color: var(--muted);
	}
	button.link {
		background: none;
		border: 0;
		padding: 0;
		cursor: pointer;
		font: inherit;
		color: var(--accent, var(--fg));
		text-align: left;
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
