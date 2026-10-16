<script lang="ts">
	import { deleted } from '$lib/undo';
	import AgentDock from '$lib/AgentDock.svelte';
	import { pageAgents } from '$lib/pageAgents';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, message, type Schemas } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import { left, stars } from '$lib/media';
	import MediaCover from '$lib/MediaCover.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Item = Schemas['MediaItem'];
	type Candidate = Schemas['MediaCandidate'];
	type Kind = Item['kind'];
	type Rec = { item: Item; score: number; reasons: string[] };

	const kinds: { value: Kind | ''; label: string }[] = $derived([
		{ value: '', label: t('All') },
		{ value: 'film', label: t('Films') },
		{ value: 'series', label: t('Series') },
		{ value: 'book', label: t('Books') }
	]);
	const kindName: Record<Kind, string> = $derived({
		film: t('Film'),
		series: t('Series'),
		book: t('Book')
	});
	const columns: { status: Item['status']; label: string }[] = $derived([
		{ status: 'doing', label: t('In progress') },
		{ status: 'want', label: t('Want to') },
		{ status: 'done', label: t('Finished') },
		{ status: 'dropped', label: t('Dropped') }
	]);
	const unit: Record<Kind, string> = { film: 'minutes', series: 'episodes', book: 'pages' };

	const kind = $derived((page.url.searchParams.get('kind') ?? '') as Kind | '');
	let items = $state<Item[] | null>(null);
	let recs = $state<Rec[]>([]);
	let error = $state('');
	let catalogs = $state({ books: true, films_and_series: false, consented: false });

	async function load() {
		error = '';
		const q = kind ? `?kind=${kind}` : '';
		try {
			const [list, r] = await Promise.all([
				api<{ items: Item[] }>(`/v1/media${q}`),
				api<{ items: Rec[] }>(`/v1/media/recommendations${q}`)
			]);
			items = list.items;
			recs = r.items.slice(0, 5);
		} catch (err) {
			error = message(err);
		}
	}

	onMount(() => {
		api<typeof catalogs>('/v1/media/catalogs')
			.then((c) => (catalogs = c))
			.catch(() => {});
	});
	$effect(() => {
		void kind;
		load().then(() => {
			const open = page.url.searchParams.get('open');
			const it = items?.find((i) => i.id === open);
			if (it) edit(it);
		});
	});

	// --- Adding ---
	let addKind = $state<Kind>('book');
	let query = $state('');
	let results = $state<Candidate[] | null>(null);
	let searching = $state(false);
	let adding = $state('');
	const available = $derived(addKind === 'book' ? catalogs.books : catalogs.films_and_series);
	const canSearch = $derived(available && catalogs.consented);

	// Lookups send the title to Open Library or TMDB, so they wait for consent.
	async function allowLookups() {
		try {
			await api('/v1/me/consents/external_metadata', { method: 'PUT', body: { granted: true } });
			catalogs = { ...catalogs, consented: true };
			toast(t('Lookups are on. You can turn them off in Privacy.'));
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function search(e: SubmitEvent) {
		e.preventDefault();
		if (!query.trim()) return;
		if (!canSearch) return addByHand();
		searching = true;
		try {
			const r = await api<{ items: Candidate[] }>(
				`/v1/media/search?kind=${addKind}&q=${encodeURIComponent(query.trim())}`
			);
			results = r.items;
		} catch (err) {
			toast(message(err), 'error');
			results = null;
		} finally {
			searching = false;
		}
	}

	async function addFrom(c: Candidate) {
		adding = c.source_id;
		try {
			await api('/v1/media', {
				method: 'POST',
				body: { kind: c.kind, catalog: { source: c.source, source_id: c.source_id } }
			});
			toast(t('Added to your list.'));
			results = null;
			query = '';
			await load();
		} catch (err) {
			toast(message(err), 'error');
		} finally {
			adding = '';
		}
	}

	async function addByHand() {
		if (!query.trim()) return;
		try {
			await api('/v1/media', { method: 'POST', body: { kind: addKind, title: query.trim() } });
			toast(t('Added to your list.'));
			results = null;
			query = '';
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	// --- Editing ---
	let editing = $state<Item | null>(null);
	let editOpen = $state(false);
	let form = $state({
		status: 'want' as Item['status'],
		rating: 0,
		progress: 0,
		total: 0,
		started_on: '',
		finished_on: '',
		review: '',
		year: '',
		creators: '',
		genres: ''
	});

	function edit(it: Item) {
		editing = it;
		form = {
			status: it.status,
			rating: it.rating ?? 0,
			progress: it.progress,
			total: it.total ?? 0,
			started_on: it.started_on ?? '',
			finished_on: it.finished_on ?? '',
			review: it.review,
			year: it.year ? String(it.year) : '',
			creators: it.creators,
			genres: it.genres.join(', ')
		};
		editOpen = true;
	}

	async function save(e: SubmitEvent) {
		e.preventDefault();
		if (!editing) return;
		try {
			await api(`/v1/media/${editing.id}`, {
				method: 'PATCH',
				body: {
					...form,
					year: Number(form.year) || 0,
					genres: form.genres
						.split(',')
						.map((g) => g.trim())
						.filter(Boolean),
					rating: Number(form.rating),
					progress: Number(form.progress),
					total: Number(form.total)
				}
			});
			editOpen = false;
			toast(t('Saved.'));
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	async function remove() {
		if (!editing || !confirm(t('Delete this item?'))) return;
		try {
			const gone = editing.id;
			await api(`/v1/media/${gone}`, { method: 'DELETE' });
			deleted(t('Removed.'), gone, () => load());
			editOpen = false;
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	// Items added by hand, imported, or from the demo have no cover until looked up.
	const missing = $derived(
		(items ?? []).filter((i) => !i.cover_id && (i.kind === 'book' || catalogs.films_and_series))
			.length
	);
	let finding = $state(false);
	// Covers are looked for by themselves once per visit (EP-134); quietly,
	// unless some were found.
	let looked = false;
	$effect(() => {
		if (!looked && items && catalogs.consented && missing > 0) {
			looked = true;
			findCovers(true);
		}
	});
	async function findCovers(quiet = false) {
		finding = true;
		try {
			const r = await api<{ checked: number; found: number; more: boolean }>(
				'/v1/media/covers/find',
				{ method: 'POST' }
			);
			const said = r.found
				? t('{found} covers found ({checked} looked up).', { found: r.found, checked: r.checked })
				: t('No cover found for the {checked} looked up.', { checked: r.checked });
			if (!quiet || r.found) toast(r.more ? `${said} ${t('Press again for the rest.')}` : said);
			if (r.found) await load();
		} catch (err) {
			if (!quiet) toast(message(err), 'error');
		} finally {
			finding = false;
		}
	}

	async function setStatus(it: Item, status: Item['status']) {
		try {
			await api(`/v1/media/${it.id}`, { method: 'PATCH', body: { status } });
			await load();
		} catch (err) {
			toast(message(err), 'error');
		}
	}

	$effect(() => {
		if (!editOpen && page.url.searchParams.has('open')) {
			const u = new URL(page.url);
			u.searchParams.delete('open');
			goto(u.pathname + u.search, { replaceState: true, keepFocus: true, noScroll: true });
		}
	});
</script>

<svelte:head><title>{t('Media')} · Axon</title></svelte:head>

<h1>{t('Films, series, and books')}</h1>
<AgentDock {...pageAgents.media} onapplied={() => load()} />

<nav class="tabs" aria-label={t('Show')}>
	{#each kinds as k (k.value)}
		<a
			href={k.value ? `?kind=${k.value}` : '?'}
			aria-current={kind === k.value ? 'page' : undefined}>{t(k.label)}</a
		>
	{/each}
</nav>

<section class="card" aria-labelledby="add">
	<h2 id="add">{t('Add something')}</h2>
	<form class="row" onsubmit={search}>
		<label class="visually-hidden" for="media-kind">{t('Kind')}</label>
		<select id="media-kind" bind:value={addKind} onchange={() => (results = null)}>
			<option value="book">{t('Book')}</option>
			<option value="film">{t('Film')}</option>
			<option value="series">{t('Series')}</option>
		</select>
		<label class="visually-hidden" for="media-q">{t('Title')}</label>
		<input id="media-q" bind:value={query} maxlength="200" placeholder={t('Title')} required />
		{#if canSearch}
			<Button type="submit" busy={searching}>{t('Search')}</Button>
			<Button variant="secondary" onclick={addByHand}>{t('Add by hand')}</Button>
		{:else}
			<Button type="submit">{t('Add')}</Button>
		{/if}
	</form>
	{#if !available}
		<p class="small muted">
			{t('Films and series are added by hand: this server has no TMDB key.')}
		</p>
	{:else if !catalogs.consented}
		<p class="small">
			{t(
				'To search, Axon sends the title you type to Open Library (books) or TMDB (films and series). Nothing else about you is sent.'
			)}
			<Button variant="secondary" onclick={allowLookups}>{t('Allow lookups')}</Button>
		</p>
	{/if}
	{#if results}
		{#if results.length === 0}
			<p class="muted">{t('Nothing found. You can add it by hand.')}</p>
		{:else}
			<ul class="results" aria-label={t('Search results')}>
				{#each results as c (c.source_id)}
					<li>
						<span>
							<strong>{c.title}</strong>
							{#if c.year}<span class="muted">({c.year})</span>{/if}
							{#if c.creators}<span class="small"> · {c.creators}</span>{/if}
						</span>
						<Button
							variant="secondary"
							busy={adding === c.source_id}
							onclick={() => addFrom(c)}
							aria-label="{t('Add')} {c.title}">{t('Add')}</Button
						>
					</li>
				{/each}
			</ul>
		{/if}
	{/if}
</section>

{#if error}
	<Status kind="error" message={error} retry={load} />
{:else if !items}
	<Status kind="loading" message={t('Loading…')} />
{:else}
	{#if recs.length}
		<section class="card" aria-labelledby="next">
			<h2 id="next">{t('Up next')}</h2>
			<ol class="recs">
				{#each recs as r (r.item.id)}
					<li>
						<button class="link" onclick={() => edit(r.item)}>{r.item.title}</button>
						<span class="muted small">{t(kindName[r.item.kind])}</span>
						<ul class="small">
							{#each r.reasons as reason (reason)}<li>{reason}</li>{/each}
						</ul>
					</li>
				{/each}
			</ol>
		</section>
	{/if}

	{#if catalogs.consented && missing > 0}
		<p class="covers-hint small">
			{missing === 1
				? t('One item has no cover yet.')
				: t('{n} items have no cover yet.', { n: missing })}
			<Button variant="secondary" busy={finding} onclick={() => findCovers()}
				>{t('Find missing covers')}</Button
			>
		</p>
	{/if}
	{#if items.length === 0}
		<Status
			kind="empty"
			message={t('Nothing here yet. Add a film, a series, or a book you want to enjoy.')}
		/>
	{/if}
	{#each columns as col (col.status)}
		{@const list = items.filter((i) => i.status === col.status)}
		{#if list.length}
			<section aria-labelledby="col-{col.status}">
				<h2 id="col-{col.status}">{t(col.label)} <span class="muted">({list.length})</span></h2>
				<ul class="shelf">
					{#each list as it (it.id)}
						{@const rest = left(it)}
						<li class="item">
							<a class="poster" href="/media/{it.id}" aria-label={it.title}>
								<MediaCover item={it} kind={kindName[it.kind]} />
							</a>
							<div class="meta">
								<a class="title" href="/media/{it.id}">{it.title}</a>
								<span class="small muted line"
									>{t(kindName[it.kind])}{it.year ? ` · ${it.year}` : ''}{it.creators
										? ` · ${it.creators}`
										: ''}</span
								>
								{#if it.rating}
									<span class="stars" title="{it.rating}/10"
										><span aria-hidden="true">{stars(it.rating)}</span><span class="visually-hidden"
											>{t('Rated')} {it.rating}/10</span
										></span
									>
								{/if}
								{#if it.genres.length}
									<span class="genres">
										{#each it.genres.slice(0, 3) as g (g)}<span class="chip">{g}</span>{/each}
									</span>
								{/if}
								{#if it.overview}<p class="overview small">{it.overview}</p>{/if}
								{#if it.total && it.status === 'doing'}
									<progress
										max={it.total}
										value={it.progress}
										aria-label="{it.title}: {it.progress} {t('of')} {it.total} {t(unit[it.kind])}"
									></progress>
									<span class="small muted"
										>{it.progress} / {it.total}
										{t(unit[it.kind])} · {t('{n} left', {
											n: rest ?? 0
										})}</span
									>
								{/if}
								<span class="actions">
									{#if it.status === 'want'}
										<Button variant="secondary" onclick={() => setStatus(it, 'doing')}
											>{t('Start')}</Button
										>
									{:else if it.status === 'doing'}
										<Button variant="secondary" onclick={() => setStatus(it, 'done')}
											>{t('Finish')}</Button
										>
									{/if}
									<Button
										variant="ghost"
										onclick={() => edit(it)}
										aria-label="{t('Edit')} {it.title}">{t('Edit')}</Button
									>
								</span>
							</div>
						</li>
					{/each}
				</ul>
			</section>
		{/if}
	{/each}
{/if}

<Dialog bind:open={editOpen} title={editing?.title ?? ''}>
	{#if editing}
		{#if editing.overview}<p class="small">{editing.overview}</p>{/if}
		<form class="grid" onsubmit={save}>
			<label
				>{t('Status')}
				<select bind:value={form.status}>
					<option value="want">{t('Want to')}</option>
					<option value="doing">{t('In progress')}</option>
					<option value="done">{t('Finished')}</option>
					<option value="dropped">{t('Dropped')}</option>
				</select></label
			>
			<label
				>{t('Rating')}
				<select bind:value={form.rating}>
					<option value={0}>{t('Not rated')}</option>
					{#each [10, 9, 8, 7, 6, 5, 4, 3, 2, 1] as n (n)}<option value={n}>{n}/10</option>{/each}
				</select></label
			>
			<label
				>{t('Progress')} ({t(unit[editing.kind])})
				<input type="number" min="0" bind:value={form.progress} /></label
			>
			<label
				>{t('Total')} ({t(unit[editing.kind])})
				<input type="number" min="0" bind:value={form.total} /></label
			>
			<label class="wide"
				>{t('By')}
				<input
					bind:value={form.creators}
					maxlength="500"
					placeholder={t('Author, director, or creator')}
				/></label
			>
			<label>{t('Year')} <input inputmode="numeric" maxlength="4" bind:value={form.year} /></label>
			<label
				>{t('Genres')} <input bind:value={form.genres} placeholder={t('Comma-separated')} /></label
			>
			<label>{t('Started')} <input type="date" bind:value={form.started_on} /></label>
			<label>{t('Finished on')} <input type="date" bind:value={form.finished_on} /></label>
			<label class="wide"
				>{t('Review')}
				<textarea rows="4" maxlength="10000" bind:value={form.review}></textarea></label
			>
			<div class="row wide">
				<Button type="submit">{t('Save')}</Button>
				<Button variant="danger" onclick={remove}>{t('Delete')}</Button>
				<Button variant="ghost" onclick={() => (editOpen = false)}>{t('Cancel')}</Button>
			</div>
		</form>
	{/if}
</Dialog>

<footer class="attribution small muted">
	<p>
		{t('Book data and covers from')} <a href="https://openlibrary.org/">{t('Open Library')}</a>.
	</p>
	{#if catalogs.films_and_series}
		<p class="tmdb">
			<img src="/attribution/tmdb.svg" alt={t('TMDB')} width="90" height="12" />
			{t(
				'This product uses TMDB and the TMDB APIs but is not endorsed, certified, or otherwise approved by TMDB.'
			)}
		</p>
	{/if}
</footer>

<style>
	.small {
		font-size: var(--text-sm);
	}
	.tabs {
		display: flex;
		gap: var(--space-1);
		margin-bottom: var(--space-4);
		overflow-x: auto;
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
	.card {
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
		padding: var(--space-4);
		margin-bottom: var(--space-6);
	}
	.row {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		align-items: center;
	}
	.row input {
		flex: 1 1 14rem;
	}
	.results,
	.recs {
		display: grid;
		gap: var(--space-2);
		padding: 0;
		margin: var(--space-3) 0 0;
	}
	.results {
		list-style: none;
	}
	.results li {
		display: flex;
		justify-content: space-between;
		align-items: center;
		gap: var(--space-3);
	}
	.recs {
		padding-left: var(--space-6);
	}
	.recs ul {
		margin: var(--space-1) 0 0;
	}
	.shelf {
		list-style: none;
		padding: 0;
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 26rem), 1fr));
		gap: var(--space-4);
		margin-bottom: var(--space-6);
	}
	/* A card worth opening (EP-135): the cover beside what matters. */
	.item {
		display: grid;
		grid-template-columns: 8.5rem minmax(0, 1fr);
		gap: var(--space-4);
		align-items: start;
		padding: var(--space-3);
		background: var(--surface);
		border: 1px solid var(--border);
		border-radius: var(--radius-lg);
	}
	.poster {
		display: block;
		border-radius: var(--radius);
		box-shadow: 0 2px 8px rgb(0 0 0 / 0.25);
		transition:
			transform var(--motion-fast) var(--ease),
			box-shadow var(--motion-fast) var(--ease);
	}
	.poster:hover,
	.poster:focus-visible {
		transform: translateY(-2px);
		box-shadow: 0 6px 16px rgb(0 0 0 / 0.35);
	}
	.title {
		font-weight: 650;
		font-size: var(--text-lg);
		color: var(--fg);
		text-decoration: none;
		line-height: 1.25;
	}
	.title:hover {
		text-decoration: underline;
	}
	.stars {
		color: #d99a00;
		letter-spacing: 0.1em;
	}
	.genres {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-1);
	}
	.chip {
		font-size: 0.75rem;
		border-radius: 999px;
		padding: 0 var(--space-2);
		background: var(--accent-soft);
	}
	.overview {
		margin: 0;
		color: var(--muted);
		display: -webkit-box;
		-webkit-line-clamp: 3;
		line-clamp: 3;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		margin-top: var(--space-1);
	}
	.meta {
		display: grid;
		gap: var(--space-1);
		min-width: 0;
	}
	.meta .title {
		display: -webkit-box;
		-webkit-line-clamp: 2;
		line-clamp: 2;
		-webkit-box-orient: vertical;
		overflow: hidden;
	}
	.meta .line {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.meta progress {
		width: 100%;
	}
	.covers-hint {
		display: flex;
		gap: var(--space-3);
		align-items: center;
		flex-wrap: wrap;
	}
	.grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(min(100%, 12rem), 1fr));
		gap: var(--space-3);
	}
	.grid label {
		display: grid;
		gap: var(--space-1);
	}
	.wide {
		grid-column: 1 / -1;
	}
	button.link {
		background: none;
		border: 0;
		padding: 0;
		color: var(--accent);
		text-decoration: underline;
		cursor: pointer;
		justify-self: start;
		font: inherit;
	}
	.attribution {
		margin-top: var(--space-8);
		border-top: 1px solid var(--border);
		padding-top: var(--space-3);
	}
	.tmdb {
		display: flex;
		gap: var(--space-2);
		align-items: center;
		flex-wrap: wrap;
	}
	.tmdb img {
		height: 12px;
		width: auto;
	}
</style>
