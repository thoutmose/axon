<script lang="ts">
	import { host, originalLabel } from '$lib/watchLinks';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { api, message } from '$lib/api';
	import AgentDock from '$lib/AgentDock.svelte';
	import Connections from '$lib/Connections.svelte';
	import { pageAgents } from '$lib/pageAgents';
	import WatchExport from '$lib/WatchExport.svelte';
	import WatchFollow from '$lib/WatchFollow.svelte';
	import Icon from '$lib/ui/Icon.svelte';
	import type { IconName } from '$lib/icons';
	import { t } from '$lib/i18n.svelte';
	import { render } from '$lib/notes';
	import Button from '$lib/ui/Button.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';
	import {
		ago,
		filterQuery,
		type WatchFilter,
		type WatchItem,
		type WatchStats,
		type WatchTopic
	} from '$lib/watch';

	// The tech watch reader (EP-110): news from the sources you follow, each
	// story once, labelled by Axon; open one to read it as a feed note.
	let filter = $state<WatchFilter>({
		status: page.url.searchParams.get('source') ? '' : 'unread',
		source: page.url.searchParams.get('source') ?? ''
	});
	let items = $state<WatchItem[]>([]);
	let topics = $state<WatchTopic[]>([]);
	let stats = $state<WatchStats | null>(null);
	let error = $state('');
	let loading = $state(true);
	let more = $state(false);
	let open = $state<{ item: WatchItem; note: string } | null>(null);
	let labelsText = $state('');
	let exporting = $state(false);

	async function load(append = false) {
		loading = true;
		error = '';
		try {
			const before = append && items.length ? items[items.length - 1].published_at : '';
			const qs = filterQuery({ ...filter });
			const res = await api<{ items: WatchItem[] }>(
				`/v1/watch/items${qs}${qs ? '&' : '?'}limit=40${before ? `&before=${encodeURIComponent(before)}` : ''}`
			);
			items = append ? [...items, ...res.items] : res.items;
			more = res.items.length === 40;
		} catch (err) {
			error = message(err);
		} finally {
			loading = false;
		}
	}
	async function loadSide() {
		try {
			[{ items: topics }, stats] = await Promise.all([
				api<{ items: WatchTopic[] }>('/v1/watch/topics'),
				api<WatchStats>('/v1/watch/stats')
			]);
		} catch {
			// The list still works without the side panel.
		}
	}
	// The kind follows the address, so the menu opens the videos or the
	// podcasts (EP-127).
	const kindParam = (k: string | null): WatchFilter['kind'] =>
		k === 'article' || k === 'video' || k === 'podcast' ? k : '';
	const urlKind = $derived(kindParam(page.url.searchParams.get('kind')));
	$effect(() => {
		filter.kind = urlKind;
	});
	function pickKind(kind: WatchFilter['kind']) {
		const q = new URLSearchParams(page.url.searchParams);
		q.delete('item');
		if (kind) q.set('kind', kind);
		else q.delete('kind');
		open = null;
		goto(`?${q}`, { keepFocus: true, noScroll: true });
	}
	$effect(() => {
		(void filter.status, filter.topic, filter.label, filter.q, filter.outliers, filter.kind);
		load();
	});
	loadSide();

	async function show(id: string) {
		try {
			open = await api<{ item: WatchItem; note: string }>(`/v1/watch/items/${id}`);
			labelsText = open.item.labels.join(', ');
			if (open.item.status === 'unread') await setStatus(open.item, 'read', true);
			const q = new URLSearchParams(page.url.searchParams);
			q.set('item', id);
			goto(`?${q}`, { replaceState: true, keepFocus: true, noScroll: true });
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	$effect(() => {
		const id = page.url.searchParams.get('item');
		if (id && open?.item.id !== id) show(id);
	});

	async function setStatus(it: WatchItem, status: WatchItem['status'], quiet = false) {
		try {
			await api(`/v1/watch/items/${it.id}`, { method: 'PATCH', body: { status } });
			items = items.map((x) => (x.id === it.id ? { ...x, status } : x));
			if (open?.item.id === it.id) open.item.status = status;
			if (!quiet) toast(t('Marked {status}.', { status: t(status) }));
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	async function saveLabels() {
		if (!open) return;
		const labels = labelsText
			.split(',')
			.map((l) => l.trim().toLowerCase())
			.filter(Boolean);
		try {
			await api(`/v1/watch/items/${open.item.id}`, { method: 'PATCH', body: { labels } });
			open.item.labels = labels;
			items = items.map((x) => (x.id === open!.item.id ? { ...x, labels } : x));
			toast(t('Labels saved.'));
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	async function keep() {
		if (!open) return;
		try {
			const res = await api<{ note_id: string }>(`/v1/watch/items/${open.item.id}/keep`, {
				method: 'POST'
			});
			open.item.note_id = res.note_id;
			open.item.status = 'saved';
			toast(t('Kept as a feed note in your notes.'));
		} catch (err) {
			toast(message(err), 'error');
		}
	}
	// A video's words (EP-123): read now rather than at the next fetch.
	let transcribing = $state(false);
	async function transcribe() {
		if (!open) return;
		transcribing = true;
		try {
			const it = await api<typeof open.item>(`/v1/watch/items/${open.item.id}/transcript`, {
				method: 'POST'
			});
			open.item = { ...open.item, ...it };
			toast(t('Transcript added.'));
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			transcribing = false;
		}
	}
	const transcriptNames: Record<string, string> = {
		captions: 'from the author’s captions',
		auto_captions: 'from YouTube’s automatic captions',
		whisper: 'transcribed by Whisper'
	};
	// The original and its source, a click away (EP-126).
	async function copyLink() {
		if (!open) return;
		await navigator.clipboard.writeText(open.item.url);
		toast(t('Link copied.'));
	}
	async function readAll() {
		try {
			const { status: _s, ...rest } = filter;
			const r = await api<{ marked: number }>(`/v1/watch/items/read${filterQuery(rest)}`, {
				method: 'POST'
			});
			toast(t('{n} marked as read.', { n: r.marked }));
			load();
			loadSide();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	// j / k move through the list, as in most readers.
	function onKey(e: KeyboardEvent) {
		const tag = (e.target as HTMLElement).tagName;
		if (['INPUT', 'TEXTAREA', 'SELECT'].includes(tag) || e.ctrlKey || e.metaKey || e.altKey) return;
		if (e.key !== 'j' && e.key !== 'k') return;
		const i = items.findIndex((x) => x.id === open?.item.id);
		const next = items[e.key === 'j' ? i + 1 : Math.max(0, i - 1)];
		if (next) {
			e.preventDefault();
			show(next.id);
		}
	}
	const topicName = (id: string) => topics.find((x) => x.id === id)?.name ?? '';
	const tabs: [WatchFilter['status'], string][] = [
		['unread', 'Unread'],
		['', 'All'],
		['saved', 'Saved'],
		['archived', 'Archived']
	];
	const kinds: { kind: WatchFilter['kind']; label: string; icon: IconName }[] = [
		{ kind: '', label: 'Everything', icon: 'web' },
		{ kind: 'article', label: 'Articles', icon: 'note' },
		{ kind: 'video', label: 'YouTube', icon: 'play' },
		{ kind: 'podcast', label: 'Podcasts', icon: 'mic' }
	];
	const kindCount = (k: WatchFilter['kind']) => {
		if (!stats) return null;
		const n = stats.items;
		return {
			'': n.articles + n.videos + n.podcasts,
			article: n.articles,
			video: n.videos,
			podcast: n.podcasts
		}[k ?? ''];
	};
</script>

<svelte:window onkeydown={onKey} />
<svelte:head
	><title
		>{filter.kind === 'video'
			? t('YouTube')
			: filter.kind === 'podcast'
				? t('Podcasts')
				: t('Tech watch')} · Axon</title
	></svelte:head
>

<header class="head">
	<div>
		<h1>
			{filter.kind === 'video'
				? t('YouTube')
				: filter.kind === 'podcast'
					? t('Podcasts')
					: t('Tech watch')}
		</h1>
		{#if stats}
			<p class="muted small">
				{t('{unread} unread · {today} new today · {sources} sources', {
					unread: stats.items.unread,
					today: stats.items.today,
					sources: stats.sources.total
				})}{#if stats.sources.failing}
					· <a href="/watch/sources" class="bad">{t('{n} failing', { n: stats.sources.failing })}</a
					>{/if}
			</p>
		{/if}
	</div>
	<nav class="row" aria-label={t('Tech watch')}>
		<Button variant="secondary" href="/watch/sources">{t('Sources and health')}</Button>
		<Button variant="ghost" href="/watch/graph">{t('Knowledge graph')}</Button>
		<Button variant="ghost" href="/watch/podcast">{t('Podcast')}</Button>
	</nav>
</header>

<div class="kinds" role="tablist" aria-label={t('Kind')}>
	{#each kinds as k (k.kind)}
		<button
			type="button"
			role="tab"
			class={k.kind || 'all'}
			aria-selected={filter.kind === k.kind}
			onclick={() => pickKind(k.kind)}
			><Icon name={k.icon} />
			{t(k.label)}
			{#if kindCount(k.kind) != null}<span class="n">{kindCount(k.kind)}</span>{/if}</button
		>
	{/each}
</div>

{#if filter.kind === 'video' || filter.kind === 'podcast'}
	{#key filter.kind}
		<WatchFollow
			kind={filter.kind}
			onfollowed={() => {
				load();
				loadSide();
			}}
		/>
	{/key}
{/if}

<div class="filters">
	<div class="tabs" role="tablist" aria-label={t('Status')}>
		{#each tabs as [s, label] (label)}
			<button
				type="button"
				role="tab"
				aria-selected={filter.status === s}
				onclick={() => (filter.status = s)}>{t(label)}</button
			>
		{/each}
	</div>
	<input
		type="search"
		aria-label={t('Search the news')}
		placeholder={t('Search the news')}
		bind:value={filter.q}
	/>
	<select aria-label={t('Topic')} bind:value={filter.topic}>
		<option value="">{t('All topics')}</option>
		{#each topics as tp (tp.id)}<option value={tp.id}>{tp.name}</option>{/each}
	</select>
	{#if filter.label}
		<button type="button" class="chip on" onclick={() => (filter.label = '')}
			>{filter.label} ✕</button
		>
	{/if}
	<label class="check"
		><input type="checkbox" bind:checked={filter.outliers} />{t('Outliers')}</label
	>
	<Button variant="ghost" onclick={readAll}>{t('Mark all read')}</Button>
	<Button variant="secondary" onclick={() => (exporting = true)}>{t('Export or share')}</Button>
</div>
<WatchExport bind:open={exporting} {filter} />
<AgentDock
	{...pageAgents.watch}
	onapplied={() => {
		load();
		loadSide();
	}}
/>

{#if stats?.labels.length && !filter.label}
	<div class="labels" aria-label={t('Labels')}>
		{#each stats.labels.slice(0, 12) as l (l.label)}
			<button type="button" class="chip" onclick={() => (filter.label = l.label)}
				>{l.label} <span class="muted">{l.count}</span></button
			>
		{/each}
	</div>
{/if}

<div class="layout" class:reading={!!open}>
	<section class="list" aria-label={t('News')}>
		{#if error}
			<Status kind="error" message={error} retry={() => load()} />
		{:else if !items.length && !loading}
			<div class="empty">
				<p>{t('Nothing here yet.')}</p>
				<p class="muted small">
					{t(
						'Follow a few sources: a blog, a Medium author, a Substack, a YouTube channel, a GitHub repository. Axon fetches them every night at midnight, keeps each story once, and labels it for you.'
					)}
				</p>
				<Button href="/watch/sources">{t('Add sources')}</Button>
			</div>
		{:else}
			<ul>
				{#each items as it (it.id)}
					<li class:unread={it.status === 'unread'} class:current={open?.item.id === it.id}>
						<button type="button" class="item" onclick={() => show(it.id)}>
							<span class="title">{it.title}</span>
							<span class="meta muted small">
								{#if it.kind === 'video'}<span class="kind video"
										><Icon name="play" size={12} /> {t('YouTube')}</span
									>
								{:else if it.kind === 'podcast'}<span class="kind podcast"
										><Icon name="mic" size={12} /> {t('Podcast')}</span
									>
								{/if}{it.source_title} · {ago(it.published_at)} · {it.reading_minutes} min
								{#if it.also_in.length}
									· <span title={it.also_in.join(', ')}
										>{t('also in {n}', { n: it.also_in.length })}</span
									>{/if}
								{#if it.status === 'saved'}
									· ★{/if}
							</span>
							{#if it.summary}<span class="summary small">{it.summary}</span>{/if}
						</button>
						<p class="links small">
							<a href={it.url} target="_blank" rel="noopener noreferrer">↗ {host(it.url)}</a
							>{#if it.source_url}{' · '}<a
									href={it.source_url}
									target="_blank"
									rel="noopener noreferrer">{t('Source: {name}', { name: it.source_title })}</a
								>{/if}
						</p>
						<div class="chips">
							{#if it.outlier}<span class="chip warn">{t(it.outlier)}</span>{/if}
							{#each it.labels.slice(0, 4) as l (l)}
								<button type="button" class="chip" onclick={() => (filter.label = l)}>{l}</button>
							{/each}
						</div>
					</li>
				{/each}
			</ul>
			{#if more}
				<Button variant="ghost" onclick={() => load(true)} busy={loading}>{t('More')}</Button>
			{/if}
		{/if}
	</section>

	{#if open}
		<article class="reader" aria-label={open.item.title}>
			<div class="row actions">
				<Button variant="ghost" onclick={() => (open = null)} aria-label={t('Close')}>←</Button>
				<a class="original" href={open.item.url} target="_blank" rel="noopener noreferrer"
					>{t(originalLabel(open.item).key, { site: originalLabel(open.item).site })} ↗</a
				>
				{#if open.item.status !== 'saved'}
					<Button variant="ghost" onclick={() => setStatus(open!.item, 'saved')}
						>★ {t('Save')}</Button
					>
				{/if}
				{#if open.item.status !== 'archived'}
					<Button variant="ghost" onclick={() => setStatus(open!.item, 'archived')}
						>{t('Archive')}</Button
					>
				{/if}
				<Button variant="ghost" onclick={() => setStatus(open!.item, 'unread')}
					>{t('Mark unread')}</Button
				>
				{#if open.item.note_id}
					<Button variant="ghost" href="/notes/{open.item.note_id}">📝 {t('Open the note')}</Button>
				{:else}
					<Button variant="ghost" onclick={keep}>📝 {t('Keep as a note')}</Button>
				{/if}
			</div>
			<p class="eyebrow muted small">
				{#if open.item.source_url}<a
						href={open.item.source_url}
						target="_blank"
						rel="noopener noreferrer">{open.item.source_title} ↗</a
					>{:else}{open.item.source_title}{/if}{#if open.item.author}
					· {open.item.author}{/if} · {new Date(open.item.published_at).toLocaleString()} · {open
					.item.reading_minutes} min
			</p>
			<h2>{open.item.title}</h2>
			<p class="link-line small">
				<a class="url" href={open.item.url} target="_blank" rel="noopener noreferrer"
					>{open.item.url}</a
				>
				<button type="button" class="linkbtn" onclick={copyLink}>{t('Copy the link')}</button>
				{#if open.item.kind === 'podcast' && open.item.audio && open.item.audio !== open.item.url}
					· <a href={open.item.audio} target="_blank" rel="noopener noreferrer"
						>{t('The audio file')}</a
					>{/if}
			</p>
			{#if open.item.also_in.length}
				<p class="muted small">{t('Also in')}: {open.item.also_in.join(', ')}</p>
			{/if}
			{#if open.item.topic_ids.length}
				<p class="small">
					{#each open.item.topic_ids as id (id)}<span class="chip topic">{topicName(id)}</span
						>{/each}
				</p>
			{/if}
			{#if open.item.keywords.length}
				<p class="small muted">{t('Key points')}: {open.item.keywords.join(' · ')}</p>
			{/if}
			{#if open.item.kind === 'video'}
				<p class="small">
					{#if open.item.transcript_source}
						{t('Transcript {source}', {
							source: t(transcriptNames[open.item.transcript_source])
						})}{#if open.item.transcript_lang}
							· {open.item.transcript_lang}{/if}
					{:else}
						<Button variant="ghost" busy={transcribing} onclick={transcribe}
							>{t('Read the transcript')}</Button
						>
						<span class="muted"
							>{t('Its words become five to ten key sentences and keywords.')}</span
						>
					{/if}
				</p>
			{/if}
			{#if open.item.transcript_source && open.item.summary}
				<!-- A video or an episode in a few words, not its subtitles (EP-148). -->
				<section class="gist" aria-labelledby="gist-title">
					<h3 id="gist-title">{t('In a few words')}</h3>
					<ul>
						{#each open.item.summary.split('\n') as line, i (i)}<li>{line}</li>{/each}
					</ul>
					{#if open.item.keywords.length}
						<p class="chips" aria-label={t('Keywords')}>
							{#each open.item.keywords as k (k)}<span class="chip">{k}</span>{/each}
						</p>
					{/if}
				</section>
			{/if}
			<div class="content">
				<!-- eslint-disable-next-line svelte/no-at-html-tags -- render() escapes everything it does not produce -->
				{@html render(open.item.content || open.item.summary, () => undefined)}
			</div>
			<div class="field">
				<label for="item-labels">{t('Labels')}</label>
				<div class="row">
					<input id="item-labels" bind:value={labelsText} />
					<Button variant="ghost" onclick={saveLabels}>{t('Save labels')}</Button>
				</div>
			</div>
			<Connections id={open.item.id} type="feed_item" />
		</article>
	{/if}
</div>

<style>
	.links {
		margin: 0 0 var(--space-1);
		padding: 0 var(--space-2);
		color: var(--muted);
	}
	.link-line {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: baseline;
		margin-top: 0;
	}
	.link-line .url {
		overflow-wrap: anywhere;
	}
	.linkbtn {
		background: none;
		border: 0;
		color: var(--accent);
		cursor: pointer;
		font: inherit;
		padding: 0;
	}
	.head {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: flex-start;
		gap: var(--space-3);
	}
	h1 {
		margin: 0;
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
	}
	.filters {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
		margin: var(--space-4) 0 var(--space-2);
	}
	.filters input[type='search'] {
		flex: 1 1 12rem;
		min-width: 0;
	}
	.kinds {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		margin-top: var(--space-4);
	}
	.kinds button {
		font: inherit;
		font-weight: 600;
		display: inline-flex;
		align-items: center;
		gap: var(--space-2);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		border-radius: var(--radius-lg);
		padding: var(--space-2) var(--space-4);
		cursor: pointer;
	}
	.kinds button[aria-selected='true'] {
		background: var(--accent);
		color: var(--accent-fg);
		border-color: var(--accent);
	}
	.kinds .n {
		font-weight: 400;
		font-size: var(--text-sm);
		opacity: 0.8;
	}
	.kind {
		display: inline-block;
		font-weight: 600;
		border-radius: 999px;
		padding: 0 var(--space-2);
		margin-right: var(--space-1);
		background: var(--accent-soft);
		color: var(--fg);
	}
	.tabs {
		display: flex;
		gap: var(--space-1);
	}
	.tabs button,
	.chip {
		font: inherit;
		font-size: var(--text-sm);
		border: 1px solid var(--border);
		background: var(--surface);
		color: var(--fg);
		border-radius: 999px;
		padding: 0.15rem var(--space-3);
		cursor: pointer;
	}
	.tabs button[aria-selected='true'],
	.chip.on {
		background: var(--accent);
		color: var(--accent-fg);
		border-color: var(--accent);
	}
	.chip.warn {
		border-color: var(--warn);
		cursor: default;
	}
	.chip.topic {
		background: var(--accent-soft);
		cursor: default;
		margin-right: var(--space-1);
	}
	.labels {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		margin-bottom: var(--space-3);
	}
	.check {
		display: flex;
		gap: var(--space-1);
		align-items: center;
	}
	.layout {
		display: grid;
		gap: var(--space-4);
	}
	.layout.reading {
		grid-template-columns: minmax(0, 2fr) minmax(0, 3fr);
	}
	@media (max-width: 52rem) {
		.layout.reading {
			grid-template-columns: 1fr;
		}
		.layout.reading .list {
			display: none;
		}
	}
	.list ul {
		list-style: none;
		padding: 0;
		margin: 0;
	}
	.list li {
		border-bottom: 1px solid var(--border);
		padding: var(--space-2) 0;
	}
	.list li.current {
		background: var(--accent-soft);
		border-radius: var(--radius);
	}
	.item {
		all: unset;
		display: grid;
		gap: 0.15rem;
		cursor: pointer;
		width: 100%;
		padding: 0 var(--space-2);
		box-sizing: border-box;
	}
	.item:focus-visible {
		outline: 2px solid var(--focus);
	}
	.title {
		font-weight: 500;
	}
	.unread .title {
		font-weight: 700;
	}
	.gist {
		border-left: 3px solid var(--accent);
		padding: var(--space-2) var(--space-4);
		margin: var(--space-3) 0;
		background: var(--accent-soft);
		border-radius: var(--radius);
	}
	.gist h3 {
		margin: 0 0 var(--space-2);
		font-size: var(--text-base);
	}
	.gist ul {
		margin: 0 0 var(--space-2);
		padding-left: var(--space-4);
		display: grid;
		gap: var(--space-1);
	}
	.summary {
		color: var(--muted);
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
		padding: var(--space-1) var(--space-2) 0;
	}
	.reader {
		background: var(--surface-solid);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		align-self: start;
		position: sticky;
		top: var(--space-4);
		max-height: calc(100vh - 6rem);
		overflow: auto;
	}
	.reader h2 {
		margin: var(--space-1) 0 var(--space-2);
	}
	.content {
		line-height: 1.6;
		overflow-wrap: anywhere;
	}
	.content :global(img) {
		max-width: 100%;
	}
	.field {
		display: grid;
		gap: var(--space-1);
		margin: var(--space-4) 0;
	}
	.field input {
		flex: 1;
		min-width: 0;
	}
	.empty {
		padding: var(--space-6) 0;
	}
	.small {
		font-size: var(--text-sm);
	}
	.bad {
		color: var(--bad);
	}
	.actions {
		margin-bottom: var(--space-2);
	}
	.original {
		font-weight: 600;
		padding: var(--space-1) var(--space-3);
		border: 1px solid var(--border);
		border-radius: 999px;
		text-decoration: none;
	}
</style>
