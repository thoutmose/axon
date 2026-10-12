<script lang="ts">
	import { deleted } from '$lib/undo';
	import AgentDock from '$lib/AgentDock.svelte';
	import { pageAgents } from '$lib/pageAgents';
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { api, message, type Schemas } from '$lib/api';
	import { t } from '$lib/i18n.svelte';
	import Button from '$lib/ui/Button.svelte';
	import Dialog from '$lib/ui/Dialog.svelte';
	import Status from '$lib/ui/Status.svelte';
	import { toast } from '$lib/ui/toast.svelte';

	type Item = Schemas['MediaItem'];
	type Candidate = Schemas['MediaCandidate'];
	type Kind = Item['kind'];
	type Rec = { item: Item; score: number; reasons: string[] };

	const kinds: { value: Kind | ''; label: string }[] = [
		{ value: '', label: 'All' },
		{ value: 'film', label: 'Films' },
		{ value: 'series', label: 'Series' },
		{ value: 'book', label: 'Books' }
	];
	const kindName: Record<Kind, string> = { film: 'Film', series: 'Series', book: 'Book' };
	const columns: { status: Item['status']; label: string }[] = [
		{ status: 'doing', label: 'In progress' },
		{ status: 'want', label: 'Want to' },
		{ status: 'done', label: 'Finished' },
		{ status: 'dropped', label: 'Dropped' }
	];
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
			deleted('Removed.', gone, () => load());
			editOpen = false;
			await load();
		} catch (err) {
			toast(message(err), 'error');
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
						<li class="item">
							<button class="cover" onclick={() => edit(it)} aria-label="{t('Edit')} {it.title}">
								{#if it.cover_id}
									<img src="/api/v1/media/covers/{it.cover_id}" alt="" loading="lazy" />
								{:else}
									<span class="initial" aria-hidden="true">{it.title.slice(0, 1)}</span>
								{/if}
							</button>
							<div class="meta">
								<strong>{it.title}</strong>
								<span class="small muted"
									>{t(kindName[it.kind])}{it.year ? ` · ${it.year}` : ''}{it.creators
										? ` · ${it.creators}`
										: ''}</span
								>
								{#if it.rating}<span class="small">{t('Rated')} {it.rating}/10</span>{/if}
								{#if it.total && it.status === 'doing'}
									<progress
										max={it.total}
										value={it.progress}
										aria-label="{it.title}: {it.progress} {t('of')} {it.total} {t(unit[it.kind])}"
									></progress>
								{/if}
								{#if it.status === 'want'}
									<button class="link small" onclick={() => setStatus(it, 'doing')}
										>{t('Start')}</button
									>
								{:else if it.status === 'doing'}
									<button class="link small" onclick={() => setStatus(it, 'done')}
										>{t('Finish')}</button
									>
								{/if}
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
		{t('Book data and covers from')} <a href="https://openlibrary.org/">Open Library</a>.
	</p>
	{#if catalogs.films_and_series}
		<p class="tmdb">
			<img src="/attribution/tmdb.svg" alt="TMDB" width="90" height="12" />
			This product uses TMDB and the TMDB APIs but is not endorsed, certified, or otherwise approved by
			TMDB.
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
		grid-template-columns: repeat(auto-fill, minmax(min(100%, 15rem), 1fr));
		gap: var(--space-4);
		margin-bottom: var(--space-6);
	}
	.item {
		display: grid;
		grid-template-columns: 4.5rem 1fr;
		gap: var(--space-3);
		align-items: start;
	}
	.cover {
		width: 4.5rem;
		aspect-ratio: 2 / 3;
		padding: 0;
		border: 1px solid var(--border);
		border-radius: var(--radius);
		background: var(--accent-soft);
		overflow: hidden;
		cursor: pointer;
		display: grid;
		place-items: center;
	}
	.cover img {
		width: 100%;
		height: 100%;
		object-fit: cover;
	}
	.initial {
		font-size: var(--text-2xl);
		font-weight: 700;
		color: var(--fg);
	}
	.meta {
		display: grid;
		gap: var(--space-1);
	}
	.meta progress {
		width: 100%;
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
